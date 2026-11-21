package monetization

import "context"

type paidContextKey uint8

const (
	snapshotContextKey paidContextKey = iota + 1
	identityContextKey
)

// WithSnapshot attaches a server-verified lease projection to a request.
func WithSnapshot(ctx context.Context, snapshot *Snapshot) context.Context {
	return context.WithValue(ctx, snapshotContextKey, snapshot)
}

// SnapshotFromContext returns the server-verified lease projection, if any.
func SnapshotFromContext(ctx context.Context) *Snapshot {
	snapshot, _ := ctx.Value(snapshotContextKey).(*Snapshot)
	return snapshot
}

// WithUserIdentity attaches an identity lookup hint to a request context.
func WithUserIdentity(ctx context.Context, identity string) context.Context {
	return context.WithValue(ctx, identityContextKey, identity)
}

// UserIdentityFromContext returns the identity hint attached by trusted server code.
func UserIdentityFromContext(ctx context.Context) string {
	identity, _ := ctx.Value(identityContextKey).(string)
	return identity
}
