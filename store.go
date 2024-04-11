package cron

import (
	"context"
	"time"
)

type JobType string

func (t JobType) String() string {
	return string(t)
}

type Job struct {
	ID      string    `json:"id,omitempty"`
	Type    JobType   `json:"type,omitempty"`
	Crontab string    `json:"crontab,omitempty"`
	Tags    []string  `json:"tags,omitempty"`
	Payload []byte    `json:"payload,omitempty"`
	Updated time.Time `json:"updated,omitempty"`
}

type Store interface {
	GetJobs(ctx context.Context, offset, size int) ([]Job, error)
}
