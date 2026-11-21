package machines

import (
	"context"
	"time"
)

// Service is the domain boundary used by handlers and enrollment orchestration.
// It exposes Machine intent and lineage only; Registry Nodes and Presence remain
// owned by their respective domains and are joined through projection readers.
type Service interface {
	Create(context.Context, CreateInput) (Machine, error)
	Resolve(context.Context, IdentityQuery) (Machine, error)
	Get(context.Context, string) (Machine, error)
	List(context.Context) ([]Machine, error)
	Archive(context.Context, string, int64) (Machine, error)
	Remove(context.Context, string, int64) (Machine, error)
	LinkNode(context.Context, string, string, string) (Machine, error)
	Merge(context.Context, MergeInput) (Machine, error)
	ListMigrationReviews(context.Context) ([]MigrationReview, error)
	AcknowledgeMigrationReview(context.Context, string) (MigrationReview, error)
	CreateCleanupTombstone(context.Context, CleanupTombstone) (CleanupTombstone, error)
	ListCleanupTombstones(context.Context, string) ([]CleanupTombstone, error)
	UpdateCleanupTombstone(context.Context, string, CleanupStatus, string) (CleanupTombstone, error)
	UpsertTrust(context.Context, TrustRecord) (TrustRecord, error)
	GetTrust(context.Context, string) (TrustRecord, error)
	ReviewHostKey(context.Context, string, string) (TrustRecord, error)
	SavePolicySnapshot(context.Context, PolicySnapshot) (PolicySnapshot, error)
	ApplyPolicy(context.Context, PolicyChangeInput) (Machine, PolicySnapshot, error)
	MarkProfileApplied(context.Context, string, string, string, time.Time) (Machine, error)
}

func NewService(repo Repository) Service { return repo }

// PolicyReader exposes the immutable policy evidence needed by machine
// projections. It is separate from Service so lightweight handler fakes and
// legacy implementations can continue to provide built-in fallback behavior.
type PolicyReader interface {
	LatestPolicySnapshot(context.Context, string) (PolicySnapshot, error)
}
