package library

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	ManifestVersion        uint32 = 1
	MaxBundleArtifacts            = 64
	MaxBundleBytes         int64  = 256 << 20
	MaxGeometryBytes       int64  = 128 << 20
	MaxTextureBytes        int64  = 24 << 20
	MaxStoredManifestBytes        = 1 << 20
	MaxStoredReviewBytes          = 64 << 10
)

type ArtifactRole string

const (
	RoleGeometryGLB  ArtifactRole = "geometry_glb"
	RoleTextureImage ArtifactRole = "texture_image"
)

type Use string

const (
	UseRender           Use = "render"
	UseEdit             Use = "edit"
	UseRedistribute     Use = "redistribute"
	UseExternalAIUpload Use = "external_ai_upload"
)

type ProvenanceKind string

const (
	ProvenanceAuthored ProvenanceKind = "authored"
	ProvenanceAcquired ProvenanceKind = "acquired"
	ProvenanceAdapted  ProvenanceKind = "adapted"
)

type ArtifactProvenance struct {
	Kind                  ProvenanceKind `json:"kind"`
	SourceLabel           string         `json:"source_label"`
	SourceSHA256          string         `json:"source_sha256,omitempty"`
	DerivedFromArtifactID string         `json:"derived_from_artifact_id,omitempty"`
	Adaptation            string         `json:"adaptation,omitempty"`
}

type Artifact struct {
	ID          string             `json:"id"`
	Role        ArtifactRole       `json:"role"`
	MIME        string             `json:"mime"`
	SizeBytes   int64              `json:"size_bytes"`
	SHA256      string             `json:"sha256"`
	LicenseID   string             `json:"license_id"`
	Attribution string             `json:"attribution"`
	Provenance  ArtifactProvenance `json:"provenance"`
	Rights      []Use              `json:"rights"`
}

type BundleManifest struct {
	Version   uint32     `json:"version"`
	BundleID  string     `json:"bundle_id"`
	Revision  uint32     `json:"revision"`
	Artifacts []Artifact `json:"artifacts"`
}

type ReviewBinding struct {
	ID             string    `json:"id"`
	ReviewerID     string    `json:"reviewer_id"`
	ReviewedAt     time.Time `json:"reviewed_at"`
	Decision       string    `json:"decision"`
	ManifestSHA256 string    `json:"manifest_sha256"`
	AllowedUses    []Use     `json:"allowed_uses"`
}

type PublishInput struct {
	Manifest BundleManifest
	Review   ReviewBinding
	Sources  map[string]io.Reader
}

type BundleRevision struct {
	Manifest       BundleManifest `json:"manifest"`
	ManifestSHA256 string         `json:"manifest_sha256"`
	Review         ReviewBinding  `json:"review"`
	ReviewSHA256   string         `json:"review_sha256"`
	PublishedAt    time.Time      `json:"published_at"`
	artifactKeys   map[string]string
	manifestJSON   []byte
	reviewJSON     []byte
}

type ResolvedArtifact struct {
	BundleID string
	Revision uint32
	Artifact Artifact
	Reader   io.ReadCloser
}

var (
	ErrRevisionNotFound   = errors.New("library bundle revision not found")
	ErrRevisionConflict   = errors.New("library bundle revision already has different immutable content")
	ErrArtifactNotFound   = errors.New("library artifact not found")
	ErrUseDenied          = errors.New("library artifact use is not permitted")
	ErrArtifactUnverified = errors.New("library artifact stream closed before digest verification")
)

var safeID = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
var sha256Pattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

// IsValidStableID exposes the library owner's identifier syntax to other
// domain owners that store immutable soft references to library revisions.
func IsValidStableID(value string) bool { return safeID.MatchString(value) }

// CanonicalizeManifest returns the normalized manifest bytes and digest used
// by review bindings. Callers must review this digest, not caller JSON bytes.
func CanonicalizeManifest(in BundleManifest) (BundleManifest, []byte, string, error) {
	return canonicalManifest(in)
}

func canonicalManifest(in BundleManifest) (BundleManifest, []byte, string, error) {
	manifest := in
	if manifest.Version != ManifestVersion || !safeID.MatchString(manifest.BundleID) || manifest.Revision == 0 {
		return BundleManifest{}, nil, "", errors.New("bundle manifest identity or version is invalid")
	}
	if len(manifest.Artifacts) == 0 || len(manifest.Artifacts) > MaxBundleArtifacts {
		return BundleManifest{}, nil, "", fmt.Errorf("bundle must contain 1..%d artifacts", MaxBundleArtifacts)
	}
	manifest.Artifacts = append([]Artifact(nil), manifest.Artifacts...)
	sort.Slice(manifest.Artifacts, func(i, j int) bool { return manifest.Artifacts[i].ID < manifest.Artifacts[j].ID })
	seen := make(map[string]bool, len(manifest.Artifacts))
	var total int64
	byID := make(map[string]Artifact, len(manifest.Artifacts))
	for i := range manifest.Artifacts {
		a := &manifest.Artifacts[i]
		if err := validateArtifact(*a); err != nil {
			return BundleManifest{}, nil, "", fmt.Errorf("artifact %q: %w", a.ID, err)
		}
		if seen[a.ID] {
			return BundleManifest{}, nil, "", fmt.Errorf("duplicate artifact id %q", a.ID)
		}
		seen[a.ID] = true
		byID[a.ID] = *a
		total += a.SizeBytes
		if total > MaxBundleBytes {
			return BundleManifest{}, nil, "", fmt.Errorf("bundle exceeds %d-byte aggregate limit", MaxBundleBytes)
		}
		a.Rights = sortedUses(a.Rights)
	}
	for _, artifact := range manifest.Artifacts {
		p := artifact.Provenance
		if p.DerivedFromArtifactID != "" {
			if p.DerivedFromArtifactID == artifact.ID {
				return BundleManifest{}, nil, "", fmt.Errorf("artifact %q cannot derive from itself", artifact.ID)
			}
			parent, ok := byID[p.DerivedFromArtifactID]
			if !ok {
				return BundleManifest{}, nil, "", fmt.Errorf("artifact %q references missing derived-from artifact", artifact.ID)
			}
			if p.SourceSHA256 != "" && p.SourceSHA256 != parent.SHA256 {
				return BundleManifest{}, nil, "", fmt.Errorf("artifact %q source digest differs from its derived-from artifact", artifact.ID)
			}
		}
	}
	if provenanceCycle(manifest.Artifacts) {
		return BundleManifest{}, nil, "", errors.New("artifact provenance contains a cycle")
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		return BundleManifest{}, nil, "", err
	}
	if len(data) > MaxStoredManifestBytes {
		return BundleManifest{}, nil, "", fmt.Errorf("canonical manifest exceeds %d bytes", MaxStoredManifestBytes)
	}
	return manifest, data, digest(data), nil
}

func provenanceCycle(artifacts []Artifact) bool {
	parents := make(map[string]string, len(artifacts))
	for _, a := range artifacts {
		if a.Provenance.DerivedFromArtifactID != "" {
			parents[a.ID] = a.Provenance.DerivedFromArtifactID
		}
	}
	for id := range parents {
		seen := map[string]bool{}
		for current := id; current != ""; current = parents[current] {
			if seen[current] {
				return true
			}
			seen[current] = true
		}
	}
	return false
}

func validateArtifact(a Artifact) error {
	if !safeID.MatchString(a.ID) || !sha256Pattern.MatchString(a.SHA256) || a.SizeBytes <= 0 {
		return errors.New("artifact id, positive byte size, and lowercase SHA-256 are required")
	}
	switch a.Role {
	case RoleGeometryGLB:
		if a.MIME != "model/gltf-binary" || a.SizeBytes > MaxGeometryBytes {
			return errors.New("geometry_glb requires model/gltf-binary and is limited to 128 MiB")
		}
	case RoleTextureImage:
		if (a.MIME != "image/png" && a.MIME != "image/jpeg") || a.SizeBytes > MaxTextureBytes {
			return errors.New("texture_image requires PNG or JPEG and is limited to 24 MiB")
		}
	default:
		return fmt.Errorf("unsupported artifact role %q", a.Role)
	}
	if strings.TrimSpace(a.LicenseID) == "" || len(a.LicenseID) > 128 || strings.TrimSpace(a.Attribution) == "" || len(a.Attribution) > 1024 {
		return errors.New("license id and attribution are required")
	}
	if err := validateProvenance(a.Provenance); err != nil {
		return err
	}
	if a.Provenance.Kind == ProvenanceAcquired && a.Provenance.SourceSHA256 != a.SHA256 {
		return errors.New("acquired provenance digest must equal the artifact digest")
	}
	if len(a.Rights) == 0 {
		return errors.New("at least one per-artifact use right is required")
	}
	seen := map[Use]bool{}
	for _, use := range a.Rights {
		if !validUse(use) || use == UseExternalAIUpload || seen[use] {
			return fmt.Errorf("unsupported or duplicate artifact right %q", use)
		}
		seen[use] = true
	}
	return nil
}

func validateProvenance(p ArtifactProvenance) error {
	if strings.TrimSpace(p.SourceLabel) == "" || len(p.SourceLabel) > 512 {
		return errors.New("provenance source label is required")
	}
	if p.SourceSHA256 != "" && !sha256Pattern.MatchString(p.SourceSHA256) {
		return errors.New("provenance source digest must be lowercase SHA-256")
	}
	switch p.Kind {
	case ProvenanceAuthored:
		if p.SourceSHA256 != "" || p.DerivedFromArtifactID != "" || p.Adaptation != "" {
			return errors.New("authored provenance cannot claim an upstream source or adaptation")
		}
	case ProvenanceAcquired:
		if p.SourceSHA256 == "" || p.DerivedFromArtifactID != "" || p.Adaptation != "" {
			return errors.New("acquired provenance requires the original source digest only")
		}
	case ProvenanceAdapted:
		if p.SourceSHA256 == "" && p.DerivedFromArtifactID == "" || strings.TrimSpace(p.Adaptation) == "" || len(p.Adaptation) > 1024 {
			return errors.New("adapted provenance requires its source and adaptation description")
		}
		if p.DerivedFromArtifactID != "" && !safeID.MatchString(p.DerivedFromArtifactID) {
			return errors.New("derived-from artifact id is invalid")
		}
	default:
		return fmt.Errorf("unsupported provenance kind %q", p.Kind)
	}
	return nil
}

func canonicalReview(in ReviewBinding, manifestSHA string) (ReviewBinding, []byte, string, error) {
	review := in
	if !safeID.MatchString(review.ID) || !safeID.MatchString(review.ReviewerID) || review.ReviewedAt.IsZero() || review.Decision != "approved" {
		return ReviewBinding{}, nil, "", errors.New("review id, reviewer, timestamp, and approved decision are required")
	}
	if review.ManifestSHA256 != manifestSHA {
		return ReviewBinding{}, nil, "", errors.New("review does not bind the canonical manifest digest")
	}
	review.ReviewedAt = review.ReviewedAt.UTC()
	if len(review.AllowedUses) == 0 {
		return ReviewBinding{}, nil, "", errors.New("review must grant at least one use")
	}
	for _, use := range review.AllowedUses {
		if !validUse(use) || use == UseExternalAIUpload {
			return ReviewBinding{}, nil, "", fmt.Errorf("review use %q is unsupported; external AI upload is not available", use)
		}
	}
	review.AllowedUses = sortedUses(review.AllowedUses)
	for i := 1; i < len(review.AllowedUses); i++ {
		if review.AllowedUses[i] == review.AllowedUses[i-1] {
			return ReviewBinding{}, nil, "", errors.New("review uses contain duplicates")
		}
	}
	data, err := json.Marshal(review)
	if err != nil {
		return ReviewBinding{}, nil, "", err
	}
	if len(data) > MaxStoredReviewBytes {
		return ReviewBinding{}, nil, "", fmt.Errorf("canonical review exceeds %d bytes", MaxStoredReviewBytes)
	}
	return review, data, digest(data), nil
}

func validUse(use Use) bool {
	return use == UseRender || use == UseEdit || use == UseRedistribute || use == UseExternalAIUpload
}

func sortedUses(in []Use) []Use {
	out := append([]Use(nil), in...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
