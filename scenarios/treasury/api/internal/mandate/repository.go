package mandate

import (
	"context"
	"errors"

	"treasury/internal/mandate/flow"
)

var ErrNotFound = errors.New("mandate not found")

type Repository interface {
	Create(context.Context, Mandate) (Mandate, error)
	Get(context.Context, string) (Mandate, error)
	GetByIdempotencyKey(context.Context, string) (Mandate, error)
	List(context.Context) ([]Mandate, error)
	UpdateStatus(context.Context, string, flow.MandateStatus, flow.MandateStatus) error
}
