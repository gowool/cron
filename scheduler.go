package cron

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron-redis-lock/v2"
	"github.com/go-co-op/gocron/v2"
	"github.com/go-redsync/redsync/v4"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type scheduler struct {
	gocron.Scheduler
	jobSyncer    Syncer
	syncDuration time.Duration
	cancel       context.CancelFunc
}

type disablePingUniversalClient struct {
	redis.UniversalClient
}

func (disablePingUniversalClient) Ping(ctx context.Context) *redis.StatusCmd {
	return redis.NewStatusCmd(ctx)
}

func NewScheduler(client redis.UniversalClient, jobSyncer Syncer, config Config, logger *slog.Logger) (gocron.Scheduler, error) {
	locker, _ := redislock.NewRedisLocker(
		disablePingUniversalClient{UniversalClient: client},
		redsync.WithTries(*config.Locker.Tries),
		redsync.WithDriftFactor(*config.Locker.DriftFactor),
		redsync.WithTimeoutFactor(*config.Locker.TimeoutFactor),
		redsync.WithExpiry(config.Locker.Expiry),
		redsync.WithRetryDelay(config.Locker.RetryDelay),
		redsync.WithValue(config.Locker.Value),
		redsync.WithFailFast(config.Locker.FailFast),
		redsync.WithShufflePools(config.Locker.ShufflePools),
	)

	s, err := gocron.NewScheduler(
		gocron.WithLocation(time.UTC),
		gocron.WithDistributedLocker(locker),
		gocron.WithStopTimeout(config.StopTimeout),
		gocron.WithLimitConcurrentJobs(config.Limit, gocron.LimitModeReschedule),
		gocron.WithLogger(logger),
		gocron.WithGlobalJobOptions(
			gocron.WithSingletonMode(gocron.LimitModeReschedule),
			gocron.WithEventListeners(
				gocron.BeforeJobRuns(func(jobID uuid.UUID, jobName string) {
					logger.WithGroup("job").Info("job start running", "id", jobID, "name", jobName)
				}),
				gocron.AfterJobRuns(func(jobID uuid.UUID, jobName string) {
					logger.WithGroup("job").Info("job stop running", "id", jobID, "name", jobName)
				}),
				gocron.AfterJobRunsWithError(func(jobID uuid.UUID, jobName string, err error) {
					logger.WithGroup("job").Error("job stop running with error", "id", jobID, "name", jobName, "error", err)
				}),
			),
		),
	)
	if err != nil {
		return nil, err
	}

	return &scheduler{
		Scheduler:    s,
		jobSyncer:    jobSyncer,
		syncDuration: config.Sync,
	}, nil
}

func (s *scheduler) Start() {
	var ctx context.Context
	ctx, s.cancel = context.WithCancel(context.Background())

	s.Scheduler.Start()

	go s.run(ctx)
}

func (s *scheduler) StopJobs() error {
	defer func() {
		if s.cancel != nil {
			s.cancel()
			s.cancel = nil
		}
	}()

	return s.Scheduler.StopJobs()
}

func (s *scheduler) Shutdown() error {
	defer func() {
		if s.cancel != nil {
			s.cancel()
			s.cancel = nil
		}
	}()

	return s.Scheduler.Shutdown()
}

func (s *scheduler) run(ctx context.Context) {
	t := time.NewTicker(s.syncDuration)

	defer t.Stop()

	s.jobSyncer.Sync(ctx, s)

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.jobSyncer.Sync(ctx, s)
		}
	}
}
