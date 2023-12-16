package cron

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/go-co-op/gocron/v2"
	"github.com/samber/lo"
)

var _ Resolver = (*TaskResolver)(nil)

var ErrTaskNotFound = errors.New("task not found")

type Resolver interface {
	Middleware(middlewares ...Middleware)
	Resolve(ctx context.Context, job *Job) (gocron.Task, error)
}

type Task interface {
	Type() JobType
	Execute(ctx context.Context, payload []byte) error
}

type (
	MiddlewareFunc func(ctx context.Context, job *Job) error
	Middleware     func(next MiddlewareFunc) MiddlewareFunc
)

type TaskResolver struct {
	tasks map[JobType]Task
	mdws  []Middleware
	mu    sync.Mutex
}

func NewResolver(tasks ...Task) *TaskResolver {
	return &TaskResolver{
		tasks: lo.SliceToMap(tasks, func(item Task) (JobType, Task) {
			return item.Type(), item
		}),
	}
}

func (r *TaskResolver) Middleware(middlewares ...Middleware) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.mdws = append(r.mdws, middlewares...)
}

func (r *TaskResolver) Resolve(ctx context.Context, job *Job) (gocron.Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if task, ok := r.tasks[job.Type]; ok {
		var next MiddlewareFunc = func(ctx context.Context, job *Job) error {
			return task.Execute(ctx, job.Payload)
		}

		for i := len(r.mdws) - 1; i >= 0; i-- {
			mdw := r.mdws[i]
			next = mdw(next)
		}

		return gocron.NewTask(next, ctx, job), nil
	}

	return nil, fmt.Errorf("%w: `%s`", ErrTaskNotFound, job.Type)
}
