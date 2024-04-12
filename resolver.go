package cron

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-co-op/gocron/v2"
)

var _ Resolver = (*TaskResolver)(nil)

var ErrTaskNotFound = errors.New("task not found")

type Resolver interface {
	Resolve(ctx context.Context, job Job) (Task, error)
}

type (
	Task           = gocron.Task
	TaskFunc       func(ctx context.Context, job Job) error
	MiddlewareFunc func(next TaskFunc) TaskFunc
)

type TaskResolver struct {
	tasks map[JobType]TaskFunc
}

func NewTaskResolver(tasks map[JobType]TaskFunc, middleware ...MiddlewareFunc) *TaskResolver {
	r := &TaskResolver{tasks: map[JobType]TaskFunc{}}
	for jt, task := range tasks {
		for i := len(middleware) - 1; i >= 0; i-- {
			task = middleware[i](task)
		}
		r.tasks[jt] = task
	}
	return r
}

func (r *TaskResolver) Resolve(ctx context.Context, job Job) (Task, error) {
	if task, ok := r.tasks[job.Type]; ok {
		return gocron.NewTask(task, ctx, job), nil
	}

	return nil, fmt.Errorf("%w: `%s`", ErrTaskNotFound, job.Type)
}
