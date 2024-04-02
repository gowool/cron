package cron

import (
	"context"
	"time"

	"github.com/go-co-op/gocron-redis-lock/v2"
	"github.com/go-co-op/gocron/v2"
	"github.com/go-redsync/redsync/v4"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
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

func NewScheduler(client redis.UniversalClient, jobSyncer Syncer, config Config, logger *zap.Logger) (gocron.Scheduler, error) {
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
		gocron.WithLogger(Logger{l: logger}),
		gocron.WithGlobalJobOptions(
			gocron.WithSingletonMode(gocron.LimitModeReschedule),
			gocron.WithEventListeners(
				gocron.BeforeJobRuns(func(id uuid.UUID, name string) {
					logger.Named("job").Info("job start running", zap.Any("job_id", id), zap.String("job_name", name))
				}),
				gocron.AfterJobRuns(func(id uuid.UUID, name string) {
					logger.Named("job").Info("job stop running", zap.Any("job_id", id), zap.String("job_name", name))
				}),
				gocron.AfterJobRunsWithError(func(id uuid.UUID, name string, err error) {
					logger.Named("job").Error("job stop running with error", zap.Any("job_id", id), zap.String("job_name", name), zap.Error(err))
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
