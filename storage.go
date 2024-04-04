package cron

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type JobType string

func (t JobType) String() string {
	return string(t)
}

type Job struct {
	ID      uuid.UUID       `json:"id,omitempty"`
	Type    JobType         `json:"type,omitempty"`
	Crontab string          `json:"crontab,omitempty"`
	Tags    []string        `json:"tags,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Enabled bool            `json:"enabled,omitempty"`
	Created time.Time       `json:"created,omitempty"`
	Updated time.Time       `json:"updated,omitempty"`
}

type Storage interface {
	FindEnabled(ctx context.Context, offset, size int) ([]Job, error)
}
