package authorization

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("authorization not found")

type Usage struct {
	BudgetTotalMinor  int64
	BudgetPeriodMinor int64
	MandateTotalMinor int64
}

type Repository interface {
	Create(context.Context, Record) (Record, error)
	Get(context.Context, string) (Record, error)
	GetByIdempotencyKey(context.Context, string) (Record, error)
	Usage(context.Context, string, string, time.Time, time.Time) (Usage, error)
	Release(context.Context, string) (Record, error)
	Approve(context.Context, string) (Record, error)
	Settle(context.Context, string) (Record, error)
}
