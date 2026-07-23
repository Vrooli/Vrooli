package runtime

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ArtifactCatalog pins the upstream Vault server archives that may enter the
// managed-service staging path. The desktop/release pipeline separately signs
// its selected staged files; this catalog prevents a managed resource from
// selecting an arbitrary download or architecture at runtime.
type ArtifactCatalog struct {
	SchemaVersion        int                       `json:"schema_version"`
	Version              string                    `json:"version"`
	ChecksumManifestURL  string                    `json:"checksum_manifest_url"`
	ChecksumSignatureURL string                    `json:"checksum_signature_url"`
	Artifacts            map[string]ArtifactTarget `json:"artifacts"`
}

type ArtifactTarget struct {
	URL          string `json:"url"`
	SHA256       string `json:"sha256"`
	BinarySHA256 string `json:"binary_sha256"`
	Archive      string `json:"archive"`
	BinaryPath   string `json:"binary_path"`
}

func ParseArtifactCatalog(data []byte) (ArtifactCatalog, error) {
	var catalog ArtifactCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return ArtifactCatalog{}, fmt.Errorf("parse Vault artifact catalog: %w", err)
	}
	if err := catalog.Validate(); err != nil {
		return ArtifactCatalog{}, err
	}
	return catalog, nil
}

func (c ArtifactCatalog) Validate() error {
	if c.SchemaVersion != 1 || strings.TrimSpace(c.Version) == "" {
		return fmt.Errorf("Vault artifact catalog requires schema version 1 and version")
	}
	for _, raw := range []string{c.ChecksumManifestURL, c.ChecksumSignatureURL} {
		if err := validateHashicorpURL(raw); err != nil {
			return fmt.Errorf("Vault artifact catalog checksum source: %w", err)
		}
	}
	if len(c.Artifacts) == 0 {
		return fmt.Errorf("Vault artifact catalog has no targets")
	}
	for target, artifact := range c.Artifacts {
		if err := validateArtifactTarget(target, artifact); err != nil {
			return err
		}
	}
	return nil
}

func validateArtifactTarget(target string, artifact ArtifactTarget) error {
	if !validPlatform(target) {
		return fmt.Errorf("Vault artifact target %q is invalid", target)
	}
	if err := validateHashicorpURL(artifact.URL); err != nil {
		return fmt.Errorf("Vault artifact %s: %w", target, err)
	}
	if artifact.Archive != "zip" {
		return fmt.Errorf("Vault artifact %s must use a zip archive", target)
	}
	if artifact.BinaryPath == "" || filepath.Base(artifact.BinaryPath) != artifact.BinaryPath {
		return fmt.Errorf("Vault artifact %s has unsafe binary path", target)
	}
	if len(artifact.SHA256) != sha256.Size*2 {
		return fmt.Errorf("Vault artifact %s has invalid SHA-256 length", target)
	}
	if _, err := hex.DecodeString(artifact.SHA256); err != nil {
		return fmt.Errorf("Vault artifact %s has invalid SHA-256: %w", target, err)
	}
	if len(artifact.BinarySHA256) != sha256.Size*2 {
		return fmt.Errorf("Vault artifact %s has invalid binary SHA-256 length", target)
	}
	if _, err := hex.DecodeString(artifact.BinarySHA256); err != nil {
		return fmt.Errorf("Vault artifact %s has invalid binary SHA-256: %w", target, err)
	}
	return nil
}

func validateHashicorpURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host != "releases.hashicorp.com" || strings.TrimSpace(u.Path) == "" {
		return fmt.Errorf("must be an https releases.hashicorp.com URL")
	}
	return nil
}

func validPlatform(value string) bool {
	switch value {
	case "linux-amd64", "linux-arm64", "macos-amd64", "macos-arm64", "windows-amd64":
		return true
	default:
		return false
	}
}

// FetchArtifact downloads a pinned archive, verifies its catalog checksum, and
// extracts only the declared server executable into destination. It accepts an
// injected client for deterministic tests; callers must validate the catalog
// before choosing a target.
func FetchArtifact(ctx context.Context, client *http.Client, artifact ArtifactTarget, destination string) (string, error) {
	if client == nil {
		client = http.DefaultClient
	}
	if artifact.Archive != "zip" || artifact.BinaryPath == "" {
		return "", fmt.Errorf("unsupported Vault artifact declaration")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, artifact.URL, nil)
	if err != nil {
		return "", fmt.Errorf("create Vault artifact request: %w", err)
	}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("download Vault artifact: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download Vault artifact: unexpected status %s", response.Status)
	}
	archive, err := os.CreateTemp("", "vrooli-vault-*.zip")
	if err != nil {
		return "", fmt.Errorf("create Vault artifact staging file: %w", err)
	}
	archivePath := archive.Name()
	defer os.Remove(archivePath)
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(archive, hash), response.Body); err != nil {
		archive.Close()
		return "", fmt.Errorf("write Vault artifact staging file: %w", err)
	}
	if err := archive.Close(); err != nil {
		return "", fmt.Errorf("close Vault artifact staging file: %w", err)
	}
	if hex.EncodeToString(hash.Sum(nil)) != strings.ToLower(artifact.SHA256) {
		return "", fmt.Errorf("Vault artifact checksum mismatch")
	}
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("open Vault artifact archive: %w", err)
	}
	defer reader.Close()
	for _, file := range reader.File {
		if filepath.Clean(file.Name) != artifact.BinaryPath || file.FileInfo().IsDir() {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return "", fmt.Errorf("create Vault binary directory: %w", err)
		}
		source, err := file.Open()
		if err != nil {
			return "", fmt.Errorf("open Vault binary in archive: %w", err)
		}
		defer source.Close()
		temporary := destination + ".tmp"
		output, err := os.OpenFile(temporary, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o700)
		if err != nil {
			return "", fmt.Errorf("create Vault binary: %w", err)
		}
		_, copyErr := io.Copy(output, source)
		closeErr := output.Close()
		if copyErr != nil {
			return "", fmt.Errorf("extract Vault binary: %w", copyErr)
		}
		if closeErr != nil {
			return "", fmt.Errorf("close Vault binary: %w", closeErr)
		}
		if err := verifyBinaryChecksum(temporary, artifact.BinarySHA256); err != nil {
			return "", err
		}
		if err := os.Rename(temporary, destination); err != nil {
			return "", fmt.Errorf("commit Vault binary: %w", err)
		}
		return destination, nil
	}
	return "", fmt.Errorf("Vault artifact archive does not contain %q", artifact.BinaryPath)
}

func verifyBinaryChecksum(path, expected string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read extracted Vault binary: %w", err)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != strings.ToLower(expected) {
		return fmt.Errorf("extracted Vault binary checksum mismatch")
	}
	return nil
}
