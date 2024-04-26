package cron

import (
	"context"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
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

func NewSchedulerWithRedisLocker(config Config, jobSyncer Syncer, client redis.UniversalClient, logger *zap.Logger, opts ...gocron.SchedulerOption) (gocron.Scheduler, error) {
	o := append(opts, gocron.WithDistributedLocker(NewRedisLocker(*config.Locker, client)))
	return NewScheduler(config, jobSyncer, logger, o...)
}

func NewScheduler(config Config, jobSyncer Syncer, logger *zap.Logger, opts ...gocron.SchedulerOption) (gocron.Scheduler, error) {
	o := []gocron.SchedulerOption{
		gocron.WithLocation(time.UTC),
		gocron.WithStopTimeout(config.StopTimeout),
		gocron.WithLimitConcurrentJobs(config.Limit, gocron.LimitModeReschedule),
		gocron.WithLogger(Logger{l: logger}),
		gocron.WithGlobalJobOptions(
			gocron.WithSingletonMode(gocron.LimitModeReschedule),
			gocron.WithEventListeners(
				gocron.BeforeJobRuns(func(_ uuid.UUID, name string) {
					logger.Named("job").Info("job start running", zap.String("name", name))
				}),
				gocron.AfterJobRuns(func(_ uuid.UUID, name string) {
					logger.Named("job").Info("job stop running", zap.String("name", name))
				}),
				gocron.AfterJobRunsWithError(func(_ uuid.UUID, name string, err error) {
					logger.Named("job").Error("job stop running with error", zap.String("name", name), zap.Error(err))
				}),
			),
		),
	}

	s, err := gocron.NewScheduler(append(o, opts...)...)
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
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = sync.OnceFunc(cancel)

	s.Scheduler.Start()

	go s.run(ctx)
}

func (s *scheduler) StopJobs() error {
	defer s.cancel()

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

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.jobSyncer.Sync(ctx, s)
		}
	}
}
