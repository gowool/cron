package cron

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const instrumName = "github.com/gowool/cron"

func MetricsMiddleware() (Middleware, error) {
	meter := otel.GetMeterProvider().Meter(instrumName)

	taskTime, err := meter.Float64Histogram(
		"cron.task.run_time",
		metric.WithDescription("The time it took to execute the task."),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return nil, err
	}

	return func(next MiddlewareFunc) MiddlewareFunc {
		return func(ctx context.Context, job *Job) error {
			start := time.Now()

			err := next(ctx, job)

			dur := time.Since(start)

			attrs := make([]attribute.KeyValue, 0, 3)
			attrs = append(attrs, attribute.String("job_name", job.ID.String()))
			attrs = append(attrs, attribute.String("job_type", job.Type.String()))
			attrs = append(attrs, statusAttr(err))

			taskTime.Record(ctx, milliseconds(dur), metric.WithAttributes(attrs...))

			return err
		}
	}, nil
}

func TracingMiddleware() Middleware {
	tracer := otel.GetTracerProvider().Tracer(instrumName)

	return func(next MiddlewareFunc) MiddlewareFunc {
		return func(ctx context.Context, job *Job) error {
			ctx, span := tracer.Start(ctx, fmt.Sprintf("execute.Job(%s)", job.Type.String()),
				trace.WithSpanKind(trace.SpanKindInternal),
				trace.WithAttributes(
					attribute.String("job_name", job.ID.String()),
					attribute.String("job_type", job.Type.String()),
				),
			)
			defer span.End()

			err := next(ctx, job)

			recordError(span, err)

			return err
		}
	}
}

func milliseconds(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}

func statusAttr(err error) attribute.KeyValue {
	if err != nil {
		return attribute.String("status", "error")
	}
	return attribute.String("status", "ok")
}

func recordError(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}
