package cron

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Syncer interface {
	Sync(ctx context.Context, s gocron.Scheduler)
}

type JobSyncer struct {
	storage  Storage
	resolver Resolver
	logger   *zap.Logger
}

func NewSyncer(storage Storage, resolver Resolver, logger *zap.Logger) *JobSyncer {
	return &JobSyncer{
		storage:  storage,
		resolver: resolver,
		logger:   logger.Named("sync"),
	}
}

func (syncer JobSyncer) Sync(ctx context.Context, s gocron.Scheduler) {
	p := &processor{
		s:        s,
		syncer:   syncer,
		cronJobs: make(map[string]gocron.Job),
	}

	for _, item := range s.Jobs() {
		p.cronJobs[item.Name()] = item
	}

	for i, size := 0, 20; ; i += size {
		data, err := syncer.storage.FindEnabled(ctx, i, size)
		if err != nil {
			if isCanceled(ctx) {
				return
			}

			syncer.logger.Error("error db find jobs", zap.Error(err))

			return
		}

		for _, item := range data {
			if err = p.process(ctx, item); err != nil {
				if isCanceled(ctx) {
					return
				}

				syncer.logger.Error("error due to sync a job", zap.Any("job_id", item.ID), zap.Any("type", item.Type), zap.Error(err))
			}
		}

		if len(data) < size {
			break
		}
	}

	for _, job := range p.cronJobs {
		if err := s.RemoveJob(job.ID()); err != nil {
			syncer.logger.Error("error remove job", zap.String("job_id", job.Name()), zap.Error(err))
		}
	}
}

type processor struct {
	s        gocron.Scheduler
	syncer   JobSyncer
	cronJobs map[string]gocron.Job
}

func (p *processor) process(ctx context.Context, job Job) error {
	task, err := p.resolve(ctx, job)
	if err != nil {
		return err
	}

	if p.isUpdate(job.ID) {
		return p.update(task, job)
	}

	return p.add(task, job)
}

func (p *processor) isUpdate(name uuid.UUID) bool {
	_, ok := p.cronJobs[name.String()]
	return ok
}

func (p *processor) update(task gocron.Task, job Job) error {
	name := job.ID.String()
	cronJob := p.cronJobs[name]
	delete(p.cronJobs, name)

	if lastRun, err := cronJob.LastRun(); err != nil || !lastRun.Before(job.Updated) {
		return nil
	}

	if _, err := p.s.Update(cronJob.ID(), definition(job.Crontab), task, options(job)...); err != nil {
		return fmt.Errorf("error due to update cron job: %w", err)
	}

	return nil
}

func (p *processor) add(task gocron.Task, job Job) error {
	if _, err := p.s.NewJob(definition(job.Crontab), task, options(job)...); err != nil {
		return fmt.Errorf("error due to add cron job: %w", err)
	}

	return nil
}

func (p *processor) resolve(ctx context.Context, job Job) (gocron.Task, error) {
	task, err := p.syncer.resolver.Resolve(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("error due to resove job task: %w", err)
	}
	return task, nil
}

func definition(crontab string) gocron.JobDefinition {
	return gocron.CronJob(crontab, strings.Count(crontab, " ") == 5)
}

func options(job Job) []gocron.JobOption {
	return []gocron.JobOption{
		gocron.WithName(job.ID.String()),
		gocron.WithTags(tags(job)...),
	}
}

func tags(job Job) []string {
	t := make([]string, 0, 3+len(job.Tags))
	t = append(t, "job", job.ID.String(), job.Type.String())
	t = append(t, job.Tags...)

	return t
}

func isCanceled(ctx context.Context) bool {
	if errors.Is(ctx.Err(), context.Canceled) {
		return true
	}
	return false
}
