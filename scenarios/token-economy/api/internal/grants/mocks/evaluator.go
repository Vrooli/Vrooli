package mocks

import (
	"context"

	"token-economy/internal/grants"
)

type FakeEvaluator struct {
	EvaluateFunc func(context.Context, grants.Grant, grants.EvaluationRequest) (grants.Decision, error)
}

func (e *FakeEvaluator) Evaluate(ctx context.Context, grant grants.Grant, request grants.EvaluationRequest) (grants.Decision, error) {
	return e.EvaluateFunc(ctx, grant, request)
}
