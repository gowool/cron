package cron

import (
	"github.com/go-co-op/gocron-redis-lock/v2"
	"github.com/go-co-op/gocron/v2"
	"github.com/go-redsync/redsync/v4"
	"github.com/redis/go-redis/v9"
)

func NewRedisLocker(config LockerConfig, client redis.UniversalClient) gocron.Locker {
	locker, _ := redislock.NewRedisLocker(
		disablePingUniversalClient{UniversalClient: client},
		redsync.WithTries(config.Tries),
		redsync.WithDriftFactor(config.DriftFactor),
		redsync.WithTimeoutFactor(config.TimeoutFactor),
		redsync.WithExpiry(config.Expiry),
		redsync.WithRetryDelay(config.RetryDelay),
		redsync.WithValue(config.Value),
		redsync.WithFailFast(config.FailFast),
		redsync.WithShufflePools(config.ShufflePools),
	)
	return locker
}
