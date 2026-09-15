package assets

import (
	"context"
	"errors"
	"log"
	"path/filepath"
	"strings"
)

// maxAssetBytes caps an uploaded asset at 32 MiB — the same ceiling the old
// REST multipart handler enforced. Brand assets are logos/icons; this is
// generous headroom while still rejecting accidental large uploads.
const maxAssetBytes = 32 << 20

// allowedMimeTypes is the whitelist of acceptable asset mime types. Mirrors the
// old REST handler: brand assets are images only.
var allowedMimeTypes = map[string]bool{
	"image/png":                true,
	"image/jpeg":               true,
	"image/svg+xml":            true,
	"image/webp":               true,
	"image/gif":                true,
	"image/x-icon":             true,
	"image/vnd.microsoft.icon": true,
}

// extMimeTypes infers a mime type from a filename extension when the caller
// does not supply one. Only the whitelisted image types are inferable.
var extMimeTypes = map[string]string{
	".png":  "image/png",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".svg":  "image/svg+xml",
	".webp": "image/webp",
	".gif":  "image/gif",
	".ico":  "image/x-icon",
}

// Service is the application-layer surface the assets handlers depend on. Owns
// validation (brand existence, mime whitelist, filename safety, size cap),
// byte persistence via the BlobStore seam, and idempotent delete semantics. The
// handler is intentionally thin around it: decode → call service → translate
// errors.
type Service interface {
	// List returns assets, optionally filtered by brandID, newest-uploaded
	// first.
	List(ctx context.Context, brandID string) ([]Asset, error)

	// Upload validates the input, confirms the brand exists, writes the bytes,
	// and upserts the catalog row keyed by (brand_id, filename). Re-uploading the
	// same filename replaces the bytes and keeps the asset id. Returns
	// ErrInvalidAsset on validation failure.
	Upload(ctx context.Context, in UploadInput) (Asset, error)

	// Get is a thin pass-through to Repository.Get.
	Get(ctx context.Context, id string) (Asset, error)

	// Download returns the stored bytes for an asset plus its filename and mime
	// type. Returns ErrAssetNotFound when the catalog row is absent.
	Download(ctx context.Context, id string) (Content, error)

	// Delete removes the asset row and best-effort removes the on-disk file.
	// Idempotent: deleting a missing asset returns nil.
	Delete(ctx context.Context, id string) error
}

type service struct {
	repo   Repository
	blobs  BlobStore
	brands BrandResolver
	logger *log.Logger
}

// NewService constructs the production Service. brands confirms the referenced
// brand exists at upload time; blobs persists the file bytes. A nil logger
// defaults to log.Default().
func NewService(repo Repository, blobs BlobStore, brands BrandResolver, logger *log.Logger) Service {
	if logger == nil {
		logger = log.Default()
	}
	return &service{repo: repo, blobs: blobs, brands: brands, logger: logger}
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) List(ctx context.Context, brandID string) ([]Asset, error) {
	return s.repo.ListByBrand(ctx, strings.TrimSpace(brandID))
}

func (s *service) Upload(ctx context.Context, in UploadInput) (Asset, error) {
	brandID := strings.TrimSpace(in.BrandID)
	if brandID == "" {
		return Asset{}, ErrInvalidAsset{Field: "brand_id", Reason: "required"}
	}

	filename, err := sanitizeFilename(in.Filename)
	if err != nil {
		return Asset{}, err
	}

	mimeType, err := resolveMime(in.MimeType, filename)
	if err != nil {
		return Asset{}, err
	}

	if len(in.Content) == 0 {
		return Asset{}, ErrInvalidAsset{Field: "content", Reason: "empty upload"}
	}
	if len(in.Content) > maxAssetBytes {
		return Asset{}, ErrInvalidAsset{Field: "content", Reason: "exceeds 32 MiB limit"}
	}

	exists, err := s.brands.BrandExists(ctx, brandID)
	if err != nil {
		return Asset{}, err
	}
	if !exists {
		return Asset{}, ErrInvalidAsset{Field: "brand_id", Reason: "brand not found"}
	}

	path, err := s.blobs.Put(brandID, filename, in.Content)
	if err != nil {
		return Asset{}, err
	}

	return s.repo.Upsert(ctx, Asset{
		BrandID:  brandID,
		Filename: filename,
		MimeType: mimeType,
		FilePath: path,
		Size:     int64(len(in.Content)),
	})
}

func (s *service) Get(ctx context.Context, id string) (Asset, error) {
	return s.repo.Get(ctx, id)
}

func (s *service) Download(ctx context.Context, id string) (Content, error) {
	a, err := s.repo.Get(ctx, id)
	if err != nil {
		return Content{}, err
	}
	data, err := s.blobs.Get(a.FilePath)
	if err != nil {
		return Content{}, err
	}
	return Content{Filename: a.Filename, MimeType: a.MimeType, Bytes: data}, nil
}

func (s *service) Delete(ctx context.Context, id string) error {
	a, err := s.repo.Get(ctx, id)
	if err != nil {
		// Idempotent: deleting an asset that does not exist is a success.
		var notFound ErrAssetNotFound
		if errors.As(err, &notFound) {
			return nil
		}
		return err
	}

	// Remove the bytes best-effort: a stale file is preferable to a dangling
	// catalog row, so a blob-removal failure is logged, not fatal.
	if err := s.blobs.Remove(a.FilePath); err != nil {
		s.logger.Printf("assets: remove blob for %q (%s): %v", a.ID, a.FilePath, err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		var notFound ErrAssetNotFound
		if errors.As(err, &notFound) {
			return nil
		}
		return err
	}
	return nil
}

// sanitizeFilename reduces in to a safe basename, rejecting empty, dot, and
// traversal inputs.
func sanitizeFilename(in string) (string, error) {
	trimmed := strings.TrimSpace(in)
	if trimmed == "" {
		return "", ErrInvalidAsset{Field: "filename", Reason: "required"}
	}
	if strings.ContainsAny(trimmed, "/\\") {
		return "", ErrInvalidAsset{Field: "filename", Reason: "must be a bare filename, not a path"}
	}
	base := filepath.Base(trimmed)
	if base == "." || base == ".." {
		return "", ErrInvalidAsset{Field: "filename", Reason: "invalid filename"}
	}
	return base, nil
}

// resolveMime returns the supplied mime type when set, else infers it from the
// filename extension. The result must be in the image whitelist.
func resolveMime(supplied, filename string) (string, error) {
	mimeType := strings.TrimSpace(strings.ToLower(supplied))
	if mimeType == "" {
		ext := strings.ToLower(filepath.Ext(filename))
		inferred, ok := extMimeTypes[ext]
		if !ok {
			return "", ErrInvalidAsset{Field: "mime_type", Reason: "cannot infer type from filename; supply mime_type"}
		}
		mimeType = inferred
	}
	if !allowedMimeTypes[mimeType] {
		return "", ErrInvalidAsset{Field: "mime_type", Reason: "unsupported type; allowed: png, jpeg, svg, webp, gif, ico"}
	}
	return mimeType, nil
}
