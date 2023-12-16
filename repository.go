package cron

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/gowool/cr"
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

type Repository interface {
	Find(ctx context.Context, criteria *cr.Criteria) ([]*Job, int, error)
	FindEnabled(ctx context.Context, offset, size int) ([]*Job, error)
	FindByID(ctx context.Context, id uuid.UUID) (*Job, error)
	DeleteByID(ctx context.Context, id uuid.UUID) error
	Save(ctx context.Context, job *Job) error
}
