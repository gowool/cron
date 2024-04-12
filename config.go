package cron

import (
	"runtime"
	"time"
)

type LockerConfig struct {
	// Tries can be used to set the number of times lock acquire is attempted.
	Tries int `cfg:"tries" json:"tries,omitempty" yaml:"tries,omitempty" bson:"tries,omitempty"`

	// DriftFactor can be used to set the clock drift factor.
	DriftFactor float64 `cfg:"drift_factor,omitempty" json:"drift_factor,omitempty" yaml:"drift_factor,omitempty" bson:"drift_factor,omitempty"`

	// TimeoutFactor can be used to set the timeout factor.
	TimeoutFactor float64 `cfg:"timeout_factor,omitempty" json:"timeout_factor,omitempty" yaml:"timeout_factor,omitempty" bson:"timeout_factor,omitempty"`

	// Expiry can be used to set the expiry of a mutex to the given value.
	Expiry time.Duration `cfg:"expiry,omitempty" json:"expiry,omitempty" yaml:"expiry,omitempty" bson:"expiry,omitempty"`

	// RetryDelay can be used to set the amount of time to wait between retries.
	RetryDelay time.Duration `cfg:"retry_delay,omitempty" json:"retry_delay,omitempty" yaml:"retry_delay,omitempty" bson:"retry_delay,omitempty"`

	// Value can be used to assign the random value without having to call lock.
	// This allows the ownership of a lock to be "transferred" and allows the lock to be unlocked from elsewhere.
	Value string `cfg:"value,omitempty" json:"value,omitempty" yaml:"value,omitempty" bson:"value,omitempty"`

	// FailFast can be used to quickly acquire and release the lock.
	// When some Redis servers are blocking, we do not need to wait for responses from all the Redis servers response.
	// As long as the quorum is met, we can assume the lock is acquired. The effect of this parameter is to achieve low
	// latency, avoid Redis blocking causing Lock/Unlock to not return for a long time.
	FailFast bool `cfg:"fail_fast,omitempty" json:"fail_fast,omitempty" yaml:"fail_fast,omitempty" bson:"fail_fast,omitempty"`

	// ShufflePools can be used to shuffle Redis pools to reduce centralized access in concurrent scenarios.
	ShufflePools bool `cfg:"shuffle_pools,omitempty" json:"shuffle_pools,omitempty" yaml:"shuffle_pools,omitempty" bson:"shuffle_pools,omitempty"`
}

func (c *LockerConfig) InitDefaults() {
	if c.Tries == 0 {
		c.Tries = 32
	}

	if c.DriftFactor == 0.0 {
		c.DriftFactor = 0.01
	}

	if c.TimeoutFactor == 0.0 {
		c.TimeoutFactor = 0.05
	}

	if c.Expiry == 0 {
		c.Expiry = 8 * time.Second
	}

	if c.RetryDelay == 0 {
		c.Expiry = 250 * time.Millisecond
	}
}

type Config struct {
	// Limit sets the limit to be used by the Scheduler for limiting
	// the number of jobs that may be running at a given time.
	Limit uint `cfg:"limit,omitempty" json:"limit,omitempty" yaml:"limit,omitempty" bson:"limit,omitempty"`

	// Sync all the jobs
	Sync time.Duration `cfg:"sync,omitempty" json:"sync,omitempty" yaml:"sync,omitempty" bson:"sync,omitempty"`

	Locker      *LockerConfig `cfg:"locker,omitempty" json:"locker,omitempty" yaml:"locker,omitempty" bson:"locker,omitempty"`
	StopTimeout time.Duration `cfg:"stop_timeout,omitempty" json:"stop_timeout,omitempty" yaml:"stop_timeout,omitempty" bson:"stop_timeout,omitempty"`
}

func (c *Config) InitDefaults() {
	if c.Limit == 0 {
		c.Limit = uint(runtime.NumCPU())
	}

	if c.StopTimeout == 0 {
		c.StopTimeout = 15 * time.Second
	}

	if c.Sync == 0 || c.Sync < time.Minute {
		c.Sync = 30 * time.Minute
	}

	if c.Locker == nil {
		c.Locker = &LockerConfig{}
	}

	c.Locker.InitDefaults()
}
