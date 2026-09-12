// Package mocks contains test doubles owned by the grants domain.
package mocks

import (
	"context"
	"time"

	"token-economy/internal/grants"
)

type FakeRepository struct {
	CreateFunc func(context.Context, grants.Grant, grants.Credit) (grants.Grant, error)
	GetFunc    func(context.Context, string) (grants.Grant, error)
	ListFunc   func(context.Context, string, string, bool) ([]grants.Grant, error)
	RevokeFunc func(context.Context, string, string, string, time.Time) (grants.Grant, error)
}

func (r *FakeRepository) Create(ctx context.Context, grant grants.Grant, event grants.Credit) (grants.Grant, error) {
	return r.CreateFunc(ctx, grant, event)
}

func (r *FakeRepository) Get(ctx context.Context, id string) (grants.Grant, error) {
	return r.GetFunc(ctx, id)
}

func (r *FakeRepository) List(ctx context.Context, holderID, tokenTypeID string, includeInactive bool) ([]grants.Grant, error) {
	if r.ListFunc == nil {
		return nil, nil
	}
	return r.ListFunc(ctx, holderID, tokenTypeID, includeInactive)
}

func (r *FakeRepository) Revoke(ctx context.Context, id, reason, idempotencyKey string, at time.Time) (grants.Grant, error) {
	if r.RevokeFunc == nil {
		return grants.Grant{}, nil
	}
	return r.RevokeFunc(ctx, id, reason, idempotencyKey, at)
}
