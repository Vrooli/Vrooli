// Package evidence owns the storage-independent description and disclosure
// policy for browser-captured material. Storage adapters may retain protected
// raw bytes, but callers receive only this package's manifest metadata.
package evidence

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"strings"

	basevidence "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/evidence"
)

const DefaultMaxArtifactSizeBytes int64 = 5 * 1024 * 1024

// DefaultPolicy is deliberately conservative: captured browser material is
// internal by default and HAR files cannot be exposed from protected storage.
func DefaultPolicy() *basevidence.EvidencePolicy {
	return &basevidence.EvidencePolicy{
		MaxArtifactSizeBytes:        DefaultMaxArtifactSizeBytes,
		DefaultRetentionClass:       basevidence.RetentionClass_RETENTION_CLASS_STANDARD,
		DefaultAccessPolicy:         basevidence.AccessPolicy_ACCESS_POLICY_PROJECT_MEMBERS,
		RedactHar:                   true,
		RedactNetwork:               true,
		RedactedHeaderNames:         []string{"authorization", "cookie", "set-cookie", "proxy-authorization", "x-api-key", "x-auth-token", "x-access-token", "x-refresh-token"},
		RedactedQueryParameterNames: []string{"access_token", "api_key", "apikey", "authorization", "client_secret", "password", "refresh_token", "secret", "session", "session_id", "token"},
	}
}

// Descriptor is the runtime-neutral metadata retained alongside an artifact.
// It intentionally excludes filesystem and object-store locations.
type Descriptor struct {
	Kind           basevidence.ArtifactKind
	MediaType      string
	SizeBytes      int64
	SHA256         string
	Classification basevidence.ContentClassification
	Retention      basevidence.RetentionClass
	Access         basevidence.AccessPolicy
	Redacted       bool
}

// FileIntegrity is the path-free integrity receipt shared by artifact owners.
// It intentionally contains no filesystem location or classification policy.
type FileIntegrity struct {
	SizeBytes int64
	SHA256    string
}

func Describe(kind, mediaType string, payload []byte, policy *basevidence.EvidencePolicy) Descriptor {
	if policy == nil {
		policy = DefaultPolicy()
	}
	kindEnum := KindFor(kind)
	classification, retention, access := ClassificationFor(kindEnum, policy)
	digest := sha256.Sum256(payload)
	return Descriptor{
		Kind: kindEnum, MediaType: strings.TrimSpace(mediaType), SizeBytes: int64(len(payload)),
		SHA256: hex.EncodeToString(digest[:]), Classification: classification,
		Retention: retention, Access: access,
		Redacted: kindEnum == basevidence.ArtifactKind_ARTIFACT_KIND_HAR && policy.RedactHar,
	}
}

// DescribeFile computes integrity metadata without treating the file location
// as evidence metadata. Callers must never serialize path into an exposed
// manifest or replay package.
func DescribeFile(kind, mediaType, path string, policy *basevidence.EvidencePolicy) (Descriptor, error) {
	integrity, err := InspectFileIntegrity(path)
	if err != nil {
		return Descriptor{}, err
	}
	if policy == nil {
		policy = DefaultPolicy()
	}
	kindEnum := KindFor(kind)
	classification, retention, access := ClassificationFor(kindEnum, policy)
	return Descriptor{Kind: kindEnum, MediaType: strings.TrimSpace(mediaType), SizeBytes: integrity.SizeBytes, SHA256: integrity.SHA256, Classification: classification, Retention: retention, Access: access, Redacted: kindEnum == basevidence.ArtifactKind_ARTIFACT_KIND_HAR && policy.RedactHar}, nil
}

// InspectFileIntegrity owns the single streaming file-read/size/digest effect
// used by capture metadata and exact-retention verification.
func InspectFileIntegrity(path string) (FileIntegrity, error) {
	file, err := os.Open(path)
	if err != nil {
		return FileIntegrity{}, err
	}
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	closeErr := file.Close()
	if err != nil {
		return FileIntegrity{}, err
	}
	if closeErr != nil {
		return FileIntegrity{}, closeErr
	}
	return FileIntegrity{SizeBytes: size, SHA256: hex.EncodeToString(hash.Sum(nil))}, nil
}

func KindFor(kind string) basevidence.ArtifactKind {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "screenshot", "image":
		return basevidence.ArtifactKind_ARTIFACT_KIND_SCREENSHOT
	case "video", "video_meta":
		return basevidence.ArtifactKind_ARTIFACT_KIND_VIDEO
	case "trace", "trace_meta":
		return basevidence.ArtifactKind_ARTIFACT_KIND_TRACE
	case "har", "har_meta":
		return basevidence.ArtifactKind_ARTIFACT_KIND_HAR
	case "dom", "dom_snapshot":
		return basevidence.ArtifactKind_ARTIFACT_KIND_DOM
	case "console":
		return basevidence.ArtifactKind_ARTIFACT_KIND_CONSOLE
	case "network":
		return basevidence.ArtifactKind_ARTIFACT_KIND_NETWORK
	case "accessibility", "accessibility_snapshot":
		return basevidence.ArtifactKind_ARTIFACT_KIND_ACCESSIBILITY
	case "performance":
		return basevidence.ArtifactKind_ARTIFACT_KIND_PERFORMANCE
	default:
		return basevidence.ArtifactKind_ARTIFACT_KIND_UNSPECIFIED
	}
}

func ClassificationFor(kind basevidence.ArtifactKind, policy *basevidence.EvidencePolicy) (basevidence.ContentClassification, basevidence.RetentionClass, basevidence.AccessPolicy) {
	if kind == basevidence.ArtifactKind_ARTIFACT_KIND_HAR {
		return basevidence.ContentClassification_CONTENT_CLASSIFICATION_SENSITIVE, basevidence.RetentionClass_RETENTION_CLASS_PROTECTED, basevidence.AccessPolicy_ACCESS_POLICY_PROTECTED_STORAGE_ONLY
	}
	if kind == basevidence.ArtifactKind_ARTIFACT_KIND_NETWORK && policy != nil && policy.RedactNetwork {
		return basevidence.ContentClassification_CONTENT_CLASSIFICATION_SENSITIVE, basevidence.RetentionClass_RETENTION_CLASS_AUDIT, basevidence.AccessPolicy_ACCESS_POLICY_EXECUTION_OWNER
	}
	if policy == nil {
		policy = DefaultPolicy()
	}
	return basevidence.ContentClassification_CONTENT_CLASSIFICATION_INTERNAL, policy.DefaultRetentionClass, policy.DefaultAccessPolicy
}
