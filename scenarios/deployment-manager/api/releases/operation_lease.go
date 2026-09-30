package releases

import (
	"context"
	"fmt"
	"os"
	"time"
)

type operationLeaseContextKey struct{}

type operationLeaseContext struct {
	OperationID string
	Owner       string
	Fence       uint64
}

// WithOperationLease carries verified worker authority through the owner
// pipeline. Repositories use it to guard release and operation mutations.
func WithOperationLease(ctx context.Context, operationID, owner string, fence uint64) context.Context {
	return context.WithValue(ctx, operationLeaseContextKey{}, operationLeaseContext{OperationID: operationID, Owner: owner, Fence: fence})
}

func operationLeaseFromContext(ctx context.Context) (operationLeaseContext, bool) {
	value, ok := ctx.Value(operationLeaseContextKey{}).(operationLeaseContext)
	return value, ok && value.OperationID != "" && value.Owner != "" && value.Fence > 0
}

func newOperationWorkerID() string {
	return fmt.Sprintf("dm-worker-%d-%d", os.Getpid(), time.Now().UnixNano())
}

// OperationLease is the current fencing authority for one operation.
type OperationLease struct {
	OperationID string
	Owner       string
	Fence       uint64
	ExpiresAt   time.Time
}

// FencedOperationRepository adds worker lease and append-only transition
// evidence to the release repository without forcing test repositories to
// implement production scheduling mechanics.
type FencedOperationRepository interface {
	AcquireOperationLease(context.Context, string, string, time.Duration) (*OperationLease, bool, error)
	ReleaseOperationLease(context.Context, *OperationLease) error
	UpdateOperationFenced(context.Context, *OperationLease, string, string, string) error
	AppendOperationEvent(context.Context, *OperationLease, string, string, string) error
}
