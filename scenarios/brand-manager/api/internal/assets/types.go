// Package assets is the domain-scoped home for brand asset files (logo,
// favicon, icon, …).
//
// Layering mirrors the canonical Vrooli pattern (see internal/brands for the
// reference example domain), with one extra seam because assets carry bytes:
//
//	Connect handler → Service (validates, resolves brand via the BrandResolver
//	                  seam, writes/reads bytes via the BlobStore seam) →
//	                  Repository (persist the catalog row)
//	                     ↑                          ↑
//	                     FakeService (handler tests) FakeRepository / FakeBlobStore
//	                                                 Real sqlite (repository tests)
//
// types.go owns the domain entities and the typed sentinels handlers translate
// at the transport edge. The proto wire types live one floor up
// (packages/proto/...) and never import this package; the handler is the only
// translation point (api-steer §7).
package assets

import (
	"fmt"
	"time"
)

// Asset is the internal domain shape for a brand asset file's catalog entry.
// FilePath is the server-internal on-disk location; the handler never copies it
// onto the wire shape (callers fetch bytes via DownloadAsset).
type Asset struct {
	ID        string
	BrandID   string
	Filename  string
	MimeType  string
	FilePath  string
	Size      int64
	CreatedAt time.Time
}

// Content is the byte payload DownloadAsset returns alongside the metadata
// needed to save or render it.
type Content struct {
	Filename string
	MimeType string
	Bytes    []byte
}

// UploadInput is the explicit input DTO Service.Upload accepts. The service
// resolves the mime type, stamps CreatedAt, and assigns the on-disk path, so
// callers cannot pass them.
type UploadInput struct {
	BrandID  string
	Filename string
	MimeType string
	Content  []byte
}

// ErrAssetNotFound is the typed sentinel returned when no row matches. Handlers
// translate via errors.As into a Connect NotFound response; the service
// swallows it for idempotent Delete.
type ErrAssetNotFound struct {
	ID string
}

func (e ErrAssetNotFound) Error() string {
	return fmt.Sprintf("asset %q not found", e.ID)
}

// ErrInvalidAsset is the typed sentinel returned when validation fails or the
// referenced brand does not exist. Field names the offending field; Reason is a
// human-safe explanation. Handlers translate via errors.As into a Connect
// InvalidArgument response.
type ErrInvalidAsset struct {
	Field  string
	Reason string
}

func (e ErrInvalidAsset) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Reason)
}
