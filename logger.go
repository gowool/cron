package cron

import (
	"fmt"

	"go.uber.org/zap"
)

type Logger struct {
	l *zap.Logger
}

func (l Logger) Debug(msg string, args ...any) {
	l.l.Debug(msg, fields(args)...)
}

func (l Logger) Error(msg string, args ...any) {
	l.l.Error(msg, fields(args)...)
}

func (l Logger) Info(msg string, args ...any) {
	l.l.Info(msg, fields(args)...)
}

func (l Logger) Warn(msg string, args ...any) {
	l.l.Warn(msg, fields(args)...)
}

func fields(args []any) []zap.Field {
	if len(args)%2 != 0 {
		return []zap.Field{zap.Any("data", args)}
	}

	m := len(args) / 2
	fields := make([]zap.Field, m)
	for i, j := 0, 0; i < m; i++ {
		fields[i] = zap.Any(fmt.Sprintf("%v", args[j]), args[j+1])
		j += 2
	}

	return fields
}
