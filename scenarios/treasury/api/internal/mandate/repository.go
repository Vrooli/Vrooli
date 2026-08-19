package mandate

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("mandate not found")

type Repository interface {
	Create(context.Context, Mandate) (Mandate, error)
	Get(context.Context, string) (Mandate, error)
	GetByIdempotencyKey(context.Context, string) (Mandate, error)
}
