// Package transfer is the heart of Device Sync Hub: it owns the server-relayed
// movement of files and text snippets between the owner's trusted devices.
//
// Layering mirrors the canonical Vrooli pattern (see internal/devices):
//
//	HTTP → handler → Service (validation, retention policy, quota, ACL)
//	                     ↓
//	                 Repository (items metadata)  +  BlobStore (file bytes)
//
// Trust is enforced one layer up: the handler resolves the caller's hub device
// token to a TRUSTED Device (devices.Authenticator) and passes the owner +
// origin-device ids in. This package never re-validates the token; it scopes
// every read and write to that owner and applies the per-item delivery ACL.
//
// Two item kinds are first-class. A TEXT item carries its body inline (the
// text_content column); a FILE item stores opaque bytes in api-core/blobstore
// under BlobKey and only metadata here. Retention (Live/Held/Pinned) is the one
// knob on the relay: the service stamps ExpiresAt at create time and the purge
// sweep removes expired rows (and their blobs).
package transfer

import (
	"fmt"
	"time"
)

// Kind is an item's payload type. The string values are the persisted form
// (items.kind) and map 1:1 to the proto ItemKind enum at the handler edge.
type Kind string

const (
	// KindText — a text snippet whose body rides inline in the item.
	KindText Kind = "text"
	// KindFile — opaque file bytes stored in the blob store.
	KindFile Kind = "file"
)

// Retention is an item's lifetime policy. String values persist in items.retention.
type Retention string

const (
	// RetentionLive — deliver to connected trusted devices, then purge. The
	// service stamps a short ExpiresAt so an undelivered Live item still drains.
	RetentionLive Retention = "live"
	// RetentionHeld — keep for a bounded window (HeldTTL), then auto-purge.
	RetentionHeld Retention = "held"
	// RetentionPinned — keep until the owner deletes it (no ExpiresAt).
	RetentionPinned Retention = "pinned"
)

// normalizeRetention maps a possibly-unspecified retention to the global
// default; an unknown string also falls back to the default rather than erroring
// so a forward-compatible client can never wedge a create.
func normalizeRetention(r Retention, def Retention) Retention {
	switch r {
	case RetentionLive, RetentionHeld, RetentionPinned:
		return r
	default:
		return def
	}
}

// Item is the internal domain shape for one transferred payload. Distinct from
// the proto wire type — the handler translates at the boundary so this layer
// never imports proto.
type Item struct {
	ID             string
	OwnerID        string
	OriginDeviceID string
	Kind           Kind
	Name           string
	MIME           string
	SizeBytes      int64
	// Text is the inline body for KindText items; empty for files.
	Text string
	// BlobKey locates the file bytes in the blob store; empty for text.
	BlobKey string
	// ThumbKey locates a generated image thumbnail; empty when none.
	ThumbKey  string
	Retention Retention
	// TargetDeviceID is empty for a broadcast item; set to direct it to one device.
	TargetDeviceID string
	// Delivered marks a Live item that has been pulled at least once, so the
	// next purge sweep removes it even before its short ExpiresAt.
	Delivered bool
	// ExpiresAt is when the item auto-purges; zero for Pinned items.
	ExpiresAt time.Time
	CreatedAt time.Time
}

// HasThumbnail reports whether a generated thumbnail exists for this item.
func (i Item) HasThumbnail() bool { return i.ThumbKey != "" }

// CreateText is the validated input for storing a text snippet.
type CreateText struct {
	OwnerID        string
	OriginDeviceID string
	Text           string
	Name           string
	Retention      Retention
	TargetDeviceID string
}

// CreateFile is the validated input for registering an already-stored file
// blob. The handler streams the bytes to the blob store first, then calls the
// service with the resulting BlobKey (+ optional ThumbKey) so a failed metadata
// write can compensate by deleting the orphaned blob.
type CreateFile struct {
	OwnerID        string
	OriginDeviceID string
	Name           string
	MIME           string
	SizeBytes      int64
	BlobKey        string
	ThumbKey       string
	Retention      Retention
	TargetDeviceID string
}

// ---- Typed sentinels (translated at the handler edge) -----------------------

// ErrItemNotFound is returned when no item matches an id within the caller's
// visibility (owner scope + delivery ACL).
type ErrItemNotFound struct{ ID string }

func (e ErrItemNotFound) Error() string { return fmt.Sprintf("item %q not found", e.ID) }

// ErrInvalidItem is returned when create-time validation fails.
type ErrInvalidItem struct {
	Field  string
	Reason string
}

func (e ErrInvalidItem) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Reason) }

// ErrInvalidTarget is returned when a directed item names a target device that
// is not a trusted device of the same owner.
type ErrInvalidTarget struct{ DeviceID string }

func (e ErrInvalidTarget) Error() string {
	return fmt.Sprintf("target device %q is not a trusted device", e.DeviceID)
}

// ErrQuotaExceeded is returned when accepting an item would push the owner (or
// the origin device) past its storage quota. Scope is "owner" or "device".
type ErrQuotaExceeded struct {
	Scope     string
	LimitByte int64
	UsedByte  int64
	WantByte  int64
}

func (e ErrQuotaExceeded) Error() string {
	return fmt.Sprintf("%s storage quota exceeded: %d used + %d requested > %d limit",
		e.Scope, e.UsedByte, e.WantByte, e.LimitByte)
}
