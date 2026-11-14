package physical

import (
	"context"
	"errors"
	"fmt"
	"io"

	"device-studio/internal/library"
)

// ArtifactRef pins both immutable bytes and the exact review decision that
// authorizes their use. It is a soft reference, not a qualification claim.
type ArtifactRef struct {
	BundleID       string               `json:"bundle_id"`
	BundleRevision uint32               `json:"bundle_revision"`
	ManifestSHA256 string               `json:"manifest_sha256"`
	ReviewSHA256   string               `json:"review_sha256"`
	ArtifactID     string               `json:"artifact_id"`
	Role           library.ArtifactRole `json:"role"`
	SizeBytes      int64                `json:"size_bytes"`
	SHA256         string               `json:"sha256"`
}

// ReviewedLibrary is the existing immutable library service boundary.
type ReviewedLibrary interface {
	GetRevision(context.Context, string, uint32) (library.BundleRevision, error)
	OpenArtifact(context.Context, string, uint32, string, library.Use) (library.ResolvedArtifact, error)
}

type LibraryArtifactResolver struct{ service ReviewedLibrary }

func NewLibraryArtifactResolver(service ReviewedLibrary) (LibraryArtifactResolver, error) {
	if service == nil {
		return LibraryArtifactResolver{}, errors.New("reviewed library service is required")
	}
	return LibraryArtifactResolver{service: service}, nil
}

// Resolve verifies the immutable bundle/review/artifact binding and use right
// through the existing library owner, then returns its verified stream.
func (r LibraryArtifactResolver) Resolve(ctx context.Context, ref ArtifactRef, use library.Use) (library.ResolvedArtifact, error) {
	if r.service == nil || ref.BundleID == "" || ref.BundleRevision == 0 || ref.ArtifactID == "" || ref.Role == "" || ref.SizeBytes <= 0 ||
		!validSHA256(ref.ManifestSHA256) || !validSHA256(ref.ReviewSHA256) || !validSHA256(ref.SHA256) {
		return library.ResolvedArtifact{}, errors.New("physical artifact reference is incomplete")
	}
	revision, err := r.service.GetRevision(ctx, ref.BundleID, ref.BundleRevision)
	if err != nil {
		return library.ResolvedArtifact{}, fmt.Errorf("resolve reviewed bundle revision: %w", err)
	}
	if revision.Manifest.BundleID != ref.BundleID || revision.Manifest.Revision != ref.BundleRevision || revision.ManifestSHA256 != ref.ManifestSHA256 || revision.ReviewSHA256 != ref.ReviewSHA256 || revision.Review.Decision != "approved" {
		return library.ResolvedArtifact{}, errors.New("physical artifact reference does not match its approved reviewed revision and use")
	}
	if !hasUse(revision.Review.AllowedUses, use) {
		return library.ResolvedArtifact{}, library.ErrUseDenied
	}
	var expected *library.Artifact
	for i := range revision.Manifest.Artifacts {
		if revision.Manifest.Artifacts[i].ID == ref.ArtifactID {
			expected = &revision.Manifest.Artifacts[i]
			break
		}
	}
	if expected == nil || expected.Role != ref.Role || expected.SizeBytes != ref.SizeBytes || expected.SHA256 != ref.SHA256 {
		return library.ResolvedArtifact{}, errors.New("physical artifact reference does not match the reviewed artifact facts and rights")
	}
	if !hasUse(expected.Rights, use) {
		return library.ResolvedArtifact{}, library.ErrUseDenied
	}
	resolved, err := r.service.OpenArtifact(ctx, ref.BundleID, ref.BundleRevision, ref.ArtifactID, use)
	if err != nil {
		return library.ResolvedArtifact{}, fmt.Errorf("open reviewed physical artifact: %w", err)
	}
	if resolved.Reader == nil || resolved.BundleID != ref.BundleID || resolved.Revision != ref.BundleRevision || resolved.Artifact.ID != ref.ArtifactID || resolved.Artifact.Role != ref.Role || resolved.Artifact.SizeBytes != ref.SizeBytes || resolved.Artifact.SHA256 != ref.SHA256 {
		if resolved.Reader != nil {
			_ = resolved.Reader.Close()
		}
		return library.ResolvedArtifact{}, errors.New("library opened bytes that do not match the pinned physical artifact")
	}
	return resolved, nil
}

func OpenReader(ctx context.Context, resolver LibraryArtifactResolver, ref ArtifactRef, use library.Use) (io.ReadCloser, error) {
	resolved, err := resolver.Resolve(ctx, ref, use)
	if err != nil {
		return nil, err
	}
	return resolved.Reader, nil
}

func hasUse(uses []library.Use, wanted library.Use) bool {
	for _, use := range uses {
		if use == wanted {
			return true
		}
	}
	return false
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, c := range value {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
