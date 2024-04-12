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
	// Name of the job aka ID, should be unique
	Name string `json:"name,omitempty"`
	// Type of the job, it's a string representation of JobType
	Type    JobType   `json:"type,omitempty"`
	Crontab string    `json:"crontab,omitempty"`
	Tags    []string  `json:"tags,omitempty"`
	Payload []byte    `json:"payload,omitempty"`
	Updated time.Time `json:"updated,omitempty"`
}

type Provider interface {
	GetJobs(ctx context.Context, offset, size int) ([]Job, error)
}
