package delivery

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

// Artifact is the durable metadata for a downloadable release artifact.
// It belongs to delivery so admin, entitlement, and update-manifest surfaces
// share one domain model instead of each carrying a transport-local copy.
type Artifact struct {
	ID                int64                  `json:"id"`
	BundleKey         string                 `json:"bundle_key"`
	AppKey            string                 `json:"app_key,omitempty"`
	Provider          string                 `json:"provider"`
	Bucket            string                 `json:"bucket"`
	ObjectKey         string                 `json:"object_key"`
	ETag              string                 `json:"etag,omitempty"`
	SizeBytes         int64                  `json:"size_bytes,omitempty"`
	SHA256            string                 `json:"sha256,omitempty"`
	SHA512            string                 `json:"sha512,omitempty"`
	ReleaseID         string                 `json:"release_id,omitempty"`
	GitCommitHash     string                 `json:"git_commit_hash,omitempty"`
	ContentType       string                 `json:"content_type,omitempty"`
	OriginalFilename  string                 `json:"original_filename,omitempty"`
	Platform          string                 `json:"platform,omitempty"`
	ReleaseVersion    string                 `json:"release_version,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	StableObjectURI   string                 `json:"stable_object_uri,omitempty"`
	SignedDownloadURL string                 `json:"signed_download_url,omitempty"`
	IsCurrent         bool                   `json:"is_current,omitempty"`
}

// ListArtifactsResult is the domain pagination result for artifact inventory
// queries. It is independent of the HTTP transport that serializes it.
type ListArtifactsResult struct {
	Artifacts []Artifact `json:"artifacts"`
	Page      int        `json:"page"`
	PageSize  int        `json:"page_size"`
	Total     int        `json:"total"`
}

// ArtifactScanTargets owns the nullable SQL representation of one canonical
// download_artifacts row. ScanDest matches the standard SELECT column order.
type ArtifactScanTargets struct {
	artifact      Artifact
	appKeyOut     sql.NullString
	provider      sql.NullString
	bucket        sql.NullString
	objectKey     sql.NullString
	etag          sql.NullString
	sizeOut       sql.NullInt64
	shaOut        sql.NullString
	sha512Out     sql.NullString
	releaseIDOut  sql.NullString
	commitHash    sql.NullString
	ctypeOut      sql.NullString
	fnameOut      sql.NullString
	platformOut   sql.NullString
	versionOut    sql.NullString
	metadataBytes []byte
}

func (t *ArtifactScanTargets) ScanDest() []interface{} {
	return []interface{}{
		&t.artifact.ID, &t.artifact.BundleKey, &t.appKeyOut, &t.provider,
		&t.bucket, &t.objectKey, &t.etag, &t.sizeOut, &t.shaOut, &t.sha512Out,
		&t.releaseIDOut, &t.commitHash, &t.ctypeOut, &t.fnameOut, &t.platformOut,
		&t.versionOut, &t.metadataBytes, &t.artifact.CreatedAt, &t.artifact.UpdatedAt,
	}
}

func (t *ArtifactScanTargets) Hydrate() Artifact {
	t.artifact.AppKey = t.appKeyOut.String
	t.artifact.Provider = t.provider.String
	t.artifact.Bucket = t.bucket.String
	t.artifact.ObjectKey = t.objectKey.String
	t.artifact.ETag = t.etag.String
	if t.sizeOut.Valid {
		t.artifact.SizeBytes = t.sizeOut.Int64
	}
	t.artifact.SHA256 = t.shaOut.String
	t.artifact.SHA512 = t.sha512Out.String
	t.artifact.ReleaseID = t.releaseIDOut.String
	t.artifact.GitCommitHash = t.commitHash.String
	t.artifact.ContentType = t.ctypeOut.String
	t.artifact.OriginalFilename = t.fnameOut.String
	t.artifact.Platform = t.platformOut.String
	t.artifact.ReleaseVersion = t.versionOut.String
	if len(t.metadataBytes) > 0 {
		var metadata map[string]interface{}
		if err := json.Unmarshal(t.metadataBytes, &metadata); err == nil {
			t.artifact.Metadata = metadata
		}
	}
	t.artifact.StableObjectURI = StableS3URI(t.artifact.Bucket, t.artifact.ObjectKey)
	return t.artifact
}

// StableS3URI returns the durable, provider-neutral S3 object identifier.
func StableS3URI(bucket, key string) string { return fmt.Sprintf("s3://%s/%s", bucket, key) }
