package transfer

import (
	"context"
	"time"
)

// ListFilter narrows a visibility list query. The zero value lists everything
// the device may see.
type ListFilter struct {
	// Query is a case-insensitive substring matched against name and inline text.
	Query string
	// Kind restricts to one payload type; empty returns both kinds.
	Kind Kind
}

// Repository is the persistence seam the transfer service depends on. Production
// wires the sqlite-backed implementation (sqlite.go); service unit tests wire
// mocks.FakeRepository. Blob bytes are NOT this layer's concern — only metadata.
//
// Visibility is enforced in SQL: an item is visible to device D of owner O when
// it belongs to O and is either broadcast, directed to D, or originated by D.
type Repository interface {
	// Create persists item, assigning ID/CreatedAt when zero, and returns it.
	Create(ctx context.Context, item Item) (Item, error)

	// GetVisible returns the item if owner-scoped AND visible to deviceID per
	// the delivery ACL, else ErrItemNotFound{id}.
	GetVisible(ctx context.Context, ownerID, deviceID, id string) (Item, error)

	// GetByOwner returns the owner-scoped item regardless of delivery ACL
	// (used by delete, which any trusted device of the owner may perform), or
	// ErrItemNotFound{id}.
	GetByOwner(ctx context.Context, ownerID, id string) (Item, error)

	// ListVisible returns the items visible to deviceID of ownerID, newest-first,
	// filtered by f.
	ListVisible(ctx context.Context, ownerID, deviceID string, f ListFilter) ([]Item, error)

	// Delete removes the owner-scoped item and returns the deleted row (its
	// BlobKey/ThumbKey let the caller clean up blobs). ErrItemNotFound when no
	// row matches.
	Delete(ctx context.Context, ownerID, id string) (Item, error)

	// MarkDelivered flags an owner-scoped item delivered (drives Live purge).
	// A no-op match is not an error — delivery marking is best-effort.
	MarkDelivered(ctx context.Context, ownerID, id string) error

	// UsageByOwner sums the stored bytes across all of the owner's items.
	UsageByOwner(ctx context.Context, ownerID string) (int64, error)

	// UsageByDevice sums the stored bytes the given origin device contributed.
	UsageByDevice(ctx context.Context, ownerID, deviceID string) (int64, error)

	// DueForPurge returns every item that should be removed as of now: any item
	// past its non-empty expires_at, plus delivered Live items. Cross-owner —
	// the purge sweep is global.
	DueForPurge(ctx context.Context, now time.Time) ([]Item, error)

	// PurgeByID removes a single item row by id (blob cleanup is the caller's).
	PurgeByID(ctx context.Context, id string) error
}
