package machines

import "context"

type Repository interface {
	Create(context.Context, CreateInput) (Machine, error)
	Get(context.Context, string) (Machine, error)
	List(context.Context) ([]Machine, error)
	Archive(context.Context, string, int64) (Machine, error)
	Remove(context.Context, string, int64) (Machine, error)
	LinkNode(context.Context, string, string, string) (Machine, error)
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
}
