package metering

import (
	"context"
	"fmt"
)

type Gateway interface {
	Reserve(context.Context, string, int64) (string, error)
	Finalize(context.Context, string, int64) error
	Release(context.Context, string) error
}
type Result struct {
	ChargeID  string
	CostCents int64
	BYOK      bool
}

func Run(ctx context.Context, g Gateway, providerKey, bundle string, estimate int64, work func() (int64, error)) (Result, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if providerKey != "" {
		cost, err := work()
		return Result{CostCents: cost, BYOK: true}, err
	}
	if g == nil {
		return Result{}, fmt.Errorf("metering gateway unavailable")
	}
	id, err := g.Reserve(ctx, bundle, estimate)
	if err != nil {
		return Result{}, err
	}
	cost, workErr := work()
	if workErr != nil {
		if releaseErr := g.Release(ctx, id); releaseErr != nil {
			return Result{}, fmt.Errorf("metered work failed: %w; release reservation: %v", workErr, releaseErr)
		}
		return Result{}, workErr
	}
	if finErr := g.Finalize(ctx, id, cost); finErr != nil {
		return Result{}, finErr
	}
	return Result{ChargeID: id, CostCents: cost}, nil
}
