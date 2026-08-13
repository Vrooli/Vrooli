package main

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
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

const (
	maxVaultArchiveBytes        = 512 << 20
	hashicorpReleaseKeyURL      = "https://keybase.io/hashicorp/pgp_keys.asc"
	hashicorpReleaseFingerprint = "C874011F0AB405110D02105534365D9472D7468F"
)

type vaultArtifactCatalog struct {
	Version              string                   `json:"version"`
	ChecksumManifestURL  string                   `json:"checksum_manifest_url"`
	ChecksumSignatureURL string                   `json:"checksum_signature_url"`
	Artifacts            map[string]vaultArtifact `json:"artifacts"`
}

type vaultArtifact struct {
	URL          string `json:"url"`
	SHA256       string `json:"sha256"`
	BinarySHA256 string `json:"binary_sha256"`
	Archive      string `json:"archive"`
	BinaryPath   string `json:"binary_path"`
}

func init() {
	registerManagedServiceArtifactStager("vault", stageVaultManagedServiceArtifact)
}

func stageVaultManagedServiceArtifact(ctx context.Context, root, outDir string, platform resourcedeployment.Platform) (stagedManagedServiceArtifact, error) {
	if err := stageVaultServer(ctx, root, outDir, platform.String()); err != nil {
		return stagedManagedServiceArtifact{}, err
	}
	version, err := vaultArtifactVersion(root)
	if err != nil {
		return stagedManagedServiceArtifact{}, err
	}
	name := "vault_" + artifactOS(platform.OS) + "_" + platform.Arch
	if platform.OS == "windows" {
		name += ".exe"
	}
	return stagedManagedServiceArtifact{Version: version, File: name}, nil
}

func vaultArtifactVersion(root string) (string, error) {
	data, err := os.ReadFile(filepath.Join(root, "resources", "vault", "artifacts.json"))
	if err != nil {
		return "", fmt.Errorf("read Vault artifact catalog: %w", err)
	}
	var catalog vaultArtifactCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return "", fmt.Errorf("parse Vault artifact catalog: %w", err)
	}
	if strings.TrimSpace(catalog.Version) == "" {
		return "", fmt.Errorf("Vault artifact catalog version is required")
	}
	return catalog.Version, nil
}

var verifyVaultChecksumProvenance = func(ctx context.Context, catalog vaultArtifactCatalog) error {
	for label, rawURL := range map[string]string{"checksum manifest": catalog.ChecksumManifestURL, "checksum signature": catalog.ChecksumSignatureURL} {
		parsed, err := url.Parse(rawURL)
		if err != nil || parsed.Scheme != "https" || parsed.Host != "releases.hashicorp.com" || !strings.HasPrefix(parsed.Path, "/vault/") {
			return fmt.Errorf("%s URL must be an HTTPS HashiCorp Vault release URL", label)
		}
	}
	dir, err := os.MkdirTemp("", "vrooli-vault-provenance-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	key, err := os.Create(filepath.Join(dir, "hashicorp.asc"))
	if err != nil {
		return err
	}
	if err := fetchVaultArchive(ctx, hashicorpReleaseKeyURL, key); err != nil {
		key.Close()
		return err
	}
	if err := key.Close(); err != nil {
		return err
	}
	keyring := filepath.Join(dir, "keyring.gpg")
	if output, err := exec.CommandContext(ctx, "gpg", "--batch", "--yes", "--dearmor", "--output", keyring, key.Name()).CombinedOutput(); err != nil {
		return fmt.Errorf("dearmor HashiCorp release key: %w: %s", err, strings.TrimSpace(string(output)))
	}
	output, err := exec.CommandContext(ctx, "gpg", "--show-keys", "--with-colons", key.Name()).Output()
	if err != nil {
		return fmt.Errorf("read HashiCorp release key: %w", err)
	}
	if !strings.Contains(strings.ToUpper(string(output)), hashicorpReleaseFingerprint) {
		return fmt.Errorf("HashiCorp release key fingerprint does not match pinned fingerprint")
	}
	manifest, err := os.Create(filepath.Join(dir, "SHA256SUMS"))
	if err != nil {
		return err
	}
	if err := fetchVaultArchive(ctx, catalog.ChecksumManifestURL, manifest); err != nil {
		manifest.Close()
		return err
	}
	if err := manifest.Close(); err != nil {
		return err
	}
	signature, err := os.Create(filepath.Join(dir, "SHA256SUMS.sig"))
	if err != nil {
		return err
	}
	if err := fetchVaultArchive(ctx, catalog.ChecksumSignatureURL, signature); err != nil {
		signature.Close()
		return err
	}
	if err := signature.Close(); err != nil {
		return err
	}
	if output, err := exec.CommandContext(ctx, "gpgv", "--keyring", keyring, signature.Name(), manifest.Name()).CombinedOutput(); err != nil {
		return fmt.Errorf("verify HashiCorp checksum signature: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// stageVaultServer is a release-only, Go-native staging step. It accepts only
// the checked-in HashiCorp catalog target, verifies the downloaded archive and
// extracted binary, and leaves the final Vrooli release signer to cover the
// staged result with its release signature.
func stageVaultServer(ctx context.Context, root, outDir, target string) error {
	target = strings.ToLower(strings.TrimSpace(target))
	if !isVaultTarget(target) {
		return fmt.Errorf("unsupported Vault target %q", target)
	}
	data, err := os.ReadFile(filepath.Join(root, "resources", "vault", "artifacts.json"))
	if err != nil {
		return fmt.Errorf("read Vault artifact catalog: %w", err)
	}
	var catalog vaultArtifactCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return fmt.Errorf("parse Vault artifact catalog: %w", err)
	}
	if err := verifyVaultChecksumProvenance(ctx, catalog); err != nil {
		return fmt.Errorf("verify Vault upstream checksum provenance: %w", err)
	}
	artifact, ok := catalog.Artifacts[target]
	if !ok {
		return fmt.Errorf("Vault artifact catalog has no target %s", target)
	}
	if err := artifact.validate(); err != nil {
		return fmt.Errorf("Vault artifact catalog target %s: %w", target, err)
	}

	archive, err := os.CreateTemp("", "vrooli-vault-archive-*")
	if err != nil {
		return fmt.Errorf("create Vault archive staging file: %w", err)
	}
	archivePath := archive.Name()
	defer os.Remove(archivePath)
	if err := fetchVaultArchive(ctx, artifact.URL, archive); err != nil {
		archive.Close()
		return err
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("close Vault archive: %w", err)
	}
	if err := verifySHA256File(archivePath, artifact.SHA256); err != nil {
		return fmt.Errorf("verify Vault archive: %w", err)
	}

	server, err := extractVaultBinary(archivePath, artifact.BinaryPath)
	if err != nil {
		return err
	}
	serverPath := server.Name()
	defer os.Remove(serverPath)
	if err := server.Close(); err != nil {
		return fmt.Errorf("close extracted Vault server: %w", err)
	}
	if err := verifySHA256File(serverPath, artifact.BinarySHA256); err != nil {
		return fmt.Errorf("verify extracted Vault server: %w", err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create release output: %w", err)
	}
	platform, arch, _ := strings.Cut(target, "-")
	platform = artifactOS(platform)
	name := "vault_" + platform + "_" + arch
	if platform == "windows" {
		name += ".exe"
	}
	destination := filepath.Join(outDir, name)
	if err := copyFile(serverPath, destination, 0o755); err != nil {
		return fmt.Errorf("stage Vault server: %w", err)
	}
	return nil
}

func (a vaultArtifact) validate() error {
	u, err := url.Parse(a.URL)
	if err != nil || u.Scheme != "https" || u.Host != "releases.hashicorp.com" || !strings.HasPrefix(u.Path, "/vault/") {
		return fmt.Errorf("url must be an HTTPS HashiCorp Vault release URL")
	}
	if a.Archive != "zip" || filepath.Base(a.BinaryPath) != a.BinaryPath || a.BinaryPath == "" {
		return fmt.Errorf("only a root-level zip binary is supported")
	}
	if err := validSHA256(a.SHA256); err != nil {
		return fmt.Errorf("archive sha256: %w", err)
	}
	return validSHA256(a.BinarySHA256)
}

func validSHA256(value string) error {
	if len(value) != sha256.Size*2 {
		return fmt.Errorf("must be a %d-character SHA-256 digest", sha256.Size*2)
	}
	_, err := hex.DecodeString(value)
	return err
}

func isVaultTarget(target string) bool {
	switch target {
	case "linux-amd64", "linux-arm64", "macos-amd64", "macos-arm64", "windows-amd64":
		return true
	default:
		return false
	}
}

var fetchVaultArchive = func(ctx context.Context, rawURL string, destination *os.File) error {
	client := &http.Client{Timeout: 2 * time.Minute}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("download Vault archive: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download Vault archive: unexpected HTTP status %s", response.Status)
	}
	if response.ContentLength > maxVaultArchiveBytes {
		return fmt.Errorf("download Vault archive: content length exceeds %d bytes", maxVaultArchiveBytes)
	}
	written, err := io.Copy(destination, io.LimitReader(response.Body, maxVaultArchiveBytes+1))
	if err != nil {
		return fmt.Errorf("write Vault archive: %w", err)
	}
	if written > maxVaultArchiveBytes {
		return fmt.Errorf("download Vault archive exceeds %d bytes", maxVaultArchiveBytes)
	}
	return nil
}

func extractVaultBinary(archivePath, binaryPath string) (*os.File, error) {
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, fmt.Errorf("open Vault archive: %w", err)
	}
	defer archive.Close()
	for _, entry := range archive.File {
		if entry.Name != binaryPath || entry.FileInfo().IsDir() {
			continue
		}
		reader, err := entry.Open()
		if err != nil {
			return nil, fmt.Errorf("open Vault binary in archive: %w", err)
		}
		defer reader.Close()
		file, err := os.CreateTemp("", "vrooli-vault-server-*")
		if err != nil {
			return nil, err
		}
		written, err := io.Copy(file, io.LimitReader(reader, maxVaultArchiveBytes+1))
		if err != nil {
			file.Close()
			os.Remove(file.Name())
			return nil, fmt.Errorf("extract Vault binary: %w", err)
		}
		if written > maxVaultArchiveBytes {
			file.Close()
			os.Remove(file.Name())
			return nil, fmt.Errorf("Vault binary exceeds %d bytes", maxVaultArchiveBytes)
		}
		return file, nil
	}
	return nil, fmt.Errorf("Vault archive does not contain %q", binaryPath)
}

var verifySHA256File = func(path, expected string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return err
	}
	if !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), expected) {
		return fmt.Errorf("checksum mismatch")
	}
	return nil
}

func copyFile(source, destination string, mode os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
