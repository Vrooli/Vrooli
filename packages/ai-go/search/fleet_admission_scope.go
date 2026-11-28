package aisearch

import (
	"context"
	"errors"
)

// FleetAdmissionScope authorizes physical steps under one already acquired
// provider lease. Implementations may issue sub-operation receipts that all
// refer to the same underlying capacity lease; they must never acquire or
// release a second lease here. CompleteQdrantFleetOperation reports every
// physical effect so an uncertain parent operation can retain its lease.
type FleetAdmissionScope interface {
	AdmitQdrantFleetOperation(context.Context, FleetOperation, string, string, int64) (FleetAdmission, error)
	CompleteQdrantFleetOperation(context.Context, FleetAdmission, bool) error
}

type fleetAdmissionScopeContextKey struct{}

// WithFleetAdmissionScope binds a provider's complete-operation lease to one
// Qdrant fleet lifecycle call. The Qdrant adapter asks this scope to validate
// each target and byte bound while all physical effects remain under the same
// outer lease.
func WithFleetAdmissionScope(ctx context.Context, scope FleetAdmissionScope) (context.Context, error) {
	if ctx == nil || scope == nil {
		return nil, errors.New("qdrant fleet admission scope and context are required")
	}
	return context.WithValue(ctx, fleetAdmissionScopeContextKey{}, scope), nil
}

// FleetAdmissionScopeFromContext returns the provider lease scope bound to a
// lifecycle call. A missing scope is not admission.
func FleetAdmissionScopeFromContext(ctx context.Context) (FleetAdmissionScope, bool) {
	if ctx == nil {
		return nil, false
	}
	scope, ok := ctx.Value(fleetAdmissionScopeContextKey{}).(FleetAdmissionScope)
	return scope, ok && scope != nil
}
