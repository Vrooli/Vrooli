package budget

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("budget not found")

type Repository interface {
	Create(context.Context, Budget) (Budget, error)
	Get(context.Context, string) (Budget, error)
}
