// Package mocks contains test doubles owned by the earning domain.
package mocks

import (
	"context"

	"token-economy/internal/earning"
)

type FakeRepository struct {
	GetByDedupFunc func(context.Context, string, string) (earning.Submission, error)
	StoreFunc      func(context.Context, earning.Submission) (earning.Submission, bool, error)
}

func (r *FakeRepository) GetByDedup(ctx context.Context, adapterIdentity, dedupKey string) (earning.Submission, error) {
	return r.GetByDedupFunc(ctx, adapterIdentity, dedupKey)
}

func (r *FakeRepository) Store(ctx context.Context, submission earning.Submission) (earning.Submission, bool, error) {
	return r.StoreFunc(ctx, submission)
}
