package chat

import (
	"context"
	"strings"
)

type ownerKey struct{}

// WithOwner binds a validated server-side identity to repository operations.
// It is not a request field. An unbound context accesses legacy rows only.
func WithOwner(ctx context.Context, owner string) (context.Context, error) {
	if owner == "" || len(owner) > 256 || strings.TrimSpace(owner) != owner {
		return nil, ErrInvalidInput
	}
	return context.WithValue(ctx, ownerKey{}, owner), nil
}

func ownerFromContext(ctx context.Context) string {
	owner, _ := ctx.Value(ownerKey{}).(string)
	return owner
}

const (
	chatOwnerClause  = "COALESCE((SELECT owner FROM chat_owners WHERE chat_id=chats.id),'')=?"
	groupOwnerClause = "COALESCE((SELECT owner FROM chat_group_owners WHERE group_id=chat_groups.id),'')=?"
)

// RequestOwner exposes only the validated subject scope to collaborating domains.
func RequestOwner(ctx context.Context) string { return ownerFromContext(ctx) }
