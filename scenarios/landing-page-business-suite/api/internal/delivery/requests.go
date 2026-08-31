package delivery

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path"
	"strings"
	"time"
)

// StorageSettingsSnapshot is the safe-to-serialize view of persisted storage
// configuration. Credential values are deliberately represented only by
// presence flags.
type StorageSettingsSnapshot struct {
	Provider                 string `json:"provider"`
	Bucket                   string `json:"bucket,omitempty"`
	Region                   string `json:"region,omitempty"`
	Endpoint                 string `json:"endpoint,omitempty"`
	ForcePathStyle           bool   `json:"force_path_style"`
	DefaultPrefix            string `json:"default_prefix,omitempty"`
	SignedURLTTLSeconds      int    `json:"signed_url_ttl_seconds"`
	PublicBaseURL            string `json:"public_base_url,omitempty"`
	AccessKeyIDSet           bool   `json:"access_key_id_set"`
	SecretAccessKeySet       bool   `json:"secret_access_key_set"`
	SessionTokenSet          bool   `json:"session_token_set"`
	CredentialsFromAuthority bool   `json:"credentials_from_authority"`
	SettingsRowAvailable     bool   `json:"settings_row_available"`
}

type StorageSettingsUpdate struct {
	Provider            *string `json:"provider"`
	Bucket              *string `json:"bucket"`
	Region              *string `json:"region"`
	Endpoint            *string `json:"endpoint"`
	ForcePathStyle      *bool   `json:"force_path_style"`
	DefaultPrefix       *string `json:"default_prefix"`
	SignedURLTTLSeconds *int    `json:"signed_url_ttl_seconds"`
	PublicBaseURL       *string `json:"public_base_url"`
	AccessKeyID         *string `json:"access_key_id"`
	SecretAccessKey     *string `json:"secret_access_key"`
	SessionToken        *string `json:"session_token"`
}

type PresignUploadRequest struct {
	Filename       string                 `json:"filename"`
	ContentType    string                 `json:"content_type"`
	AppKey         string                 `json:"app_key"`
	Platform       string                 `json:"platform"`
	ReleaseVersion string                 `json:"release_version"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type PresignUploadResponse struct {
	UploadURL       string            `json:"upload_url"`
	RequiredHeaders map[string]string `json:"required_headers"`
	Bucket          string            `json:"bucket"`
	ObjectKey       string            `json:"object_key"`
	ExpiresAt       time.Time         `json:"expires_at"`
	StableObjectURI string            `json:"stable_object_uri"`
}

type CommitArtifactRequest struct {
	Bucket           string                 `json:"bucket"`
	ObjectKey        string                 `json:"object_key"`
	OriginalFilename string                 `json:"original_filename"`
	ContentType      string                 `json:"content_type"`
	AppKey           string                 `json:"app_key"`
	Platform         string                 `json:"platform"`
	ReleaseVersion   string                 `json:"release_version"`
	SHA256           string                 `json:"sha256"`
	SHA512           string                 `json:"sha512"`
	ReleaseID        string                 `json:"release_id"`
	GitCommitHash    string                 `json:"git_commit_hash"`
	Metadata         map[string]interface{} `json:"metadata"`
	SetAsCurrent     bool                   `json:"set_as_current"`
}

// BuildObjectKey creates a traversal-safe, collision-resistant object key for
// an upload. It belongs to delivery because key shape is storage policy rather
// than HTTP policy.
func BuildObjectKey(settings StorageSettings, bundleKey string, req PresignUploadRequest) (string, error) {
	prefix := strings.Trim(strings.TrimSpace(settings.DefaultPrefix), "/")
	filename := SanitizeObjectFilename(req.Filename)
	nonce, err := RandomHex(6)
	if err != nil {
		return "", err
	}
	segments := []string{}
	if prefix != "" {
		segments = append(segments, prefix)
	}
	segments = append(segments, bundleKey)
	if app := strings.TrimSpace(req.AppKey); app != "" {
		segments = append(segments, app)
	}
	if platform := strings.TrimSpace(req.Platform); platform != "" {
		segments = append(segments, platform)
	}
	if version := strings.TrimSpace(req.ReleaseVersion); version != "" {
		segments = append(segments, version)
	}
	segments = append(segments, fmt.Sprintf("%d-%s-%s", time.Now().UTC().Unix(), nonce, filename))
	return strings.Join(segments, "/"), nil
}

// RandomHex returns n cryptographically random bytes as lowercase hex.
func RandomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// SanitizeObjectFilename removes path traversal and normalizes an object name.
func SanitizeObjectFilename(filename string) string {
	filename = strings.TrimSpace(filename)
	filename = path.Base(filename)
	if filename == "." || filename == "/" {
		return "artifact.bin"
	}
	filename = strings.ReplaceAll(filename, " ", "-")
	filename = strings.ReplaceAll(filename, "..", ".")
	filename = strings.Trim(filename, "/")
	if filename == "" {
		return "artifact.bin"
	}
	return filename
}
