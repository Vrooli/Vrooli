package main

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/gorilla/mux"
)

const (
	maxBrowserExtensionArtifactBytes = 64 << 20
	maxBrowserExtensionManifestBytes = 1 << 20
)

// BrowserExtensionArtifactMetadata is derived from manifest.json inside the
// exact digest-verified release ZIP. It deliberately contains no filesystem
// paths or caller-provided extension ID.
type BrowserExtensionArtifactMetadata struct {
	ArtifactRef     string   `json:"artifact_ref"`
	ArtifactSHA256  string   `json:"artifact_sha256"`
	ManifestSHA256  string   `json:"manifest_sha256"`
	ExtensionID     string   `json:"extension_id"`
	Version         string   `json:"version"`
	Permissions     []string `json:"permissions"`
	HostPermissions []string `json:"host_permissions"`
	PermissionProbe string   `json:"permission_probe,omitempty"`
}

type extensionManifestForBrowserArtifact struct {
	ManifestVersion         int      `json:"manifest_version"`
	Name                    string   `json:"name"`
	Version                 string   `json:"version"`
	Key                     string   `json:"key"`
	Permissions             []string `json:"permissions"`
	OptionalPermissions     []string `json:"optional_permissions"`
	HostPermissions         []string `json:"host_permissions"`
	OptionalHostPermissions []string `json:"optional_host_permissions"`
}

// prepareBrowserExtensionManifest gives unpacked Chromium installs a stable
// artifact-derived ID. Only the public key is embedded; the private key is
// discarded. Existing manifest keys are preserved and validated.
func prepareBrowserExtensionManifest(root string) error {
	path := root + string(os.PathSeparator) + "manifest.json"
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read manifest for browser identity: %w", err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(raw, &document); err != nil {
		return fmt.Errorf("manifest.json is invalid JSON: %w", err)
	}
	var key string
	if encoded := document["key"]; len(encoded) > 0 {
		if err := json.Unmarshal(encoded, &key); err != nil {
			return errors.New("manifest key must be a base64 public key")
		}
		if _, err := chromeExtensionID(key); err != nil {
			return fmt.Errorf("manifest key is invalid: %w", err)
		}
		return nil
	}
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("generate extension identity key: %w", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("encode extension public key: %w", err)
	}
	encoded, err := json.Marshal(base64.StdEncoding.EncodeToString(der))
	if err != nil {
		return err
	}
	document["key"] = encoded
	updated, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return fmt.Errorf("encode extension manifest identity: %w", err)
	}
	updated = append(updated, '\n')
	if err := os.WriteFile(path, updated, 0o644); err != nil {
		return fmt.Errorf("write extension manifest identity: %w", err)
	}
	return nil
}

// chromeExtensionID derives Chromium's 32-character a-p ID from a manifest
// key's base64 DER SubjectPublicKeyInfo. Only RSA public keys are accepted.
func chromeExtensionID(encodedPublicKey string) (string, error) {
	der, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encodedPublicKey))
	if err != nil || len(der) == 0 {
		return "", errors.New("manifest key is not valid base64")
	}
	publicKey, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return "", fmt.Errorf("parse manifest public key: %w", err)
	}
	rsaKey, ok := publicKey.(*rsa.PublicKey)
	if !ok || rsaKey.N == nil || rsaKey.N.BitLen() < 2048 {
		return "", errors.New("manifest public key must be RSA with at least 2048 bits")
	}
	digest := sha256.Sum256(der)
	const alphabet = "abcdefghijklmnop"
	var id strings.Builder
	for _, value := range digest[:16] {
		id.WriteByte(alphabet[value>>4])
		id.WriteByte(alphabet[value&0x0f])
	}
	return id.String(), nil
}

func (bm *BuildManager) VerifyBrowserExtensionArtifact(buildID, expectedDigest string) (BrowserExtensionArtifactMetadata, error) {
	if strings.TrimSpace(buildID) == "" || strings.TrimSpace(expectedDigest) == "" {
		return BrowserExtensionArtifactMetadata{}, errors.New("artifact_ref and expected_digest are required")
	}
	bm.mu.RLock()
	stored, ok := bm.builds[buildID]
	var build *ExtensionBuild
	if ok && stored != nil {
		build = cloneExtensionBuild(stored)
	}
	bm.mu.RUnlock()
	if !ok || build == nil || build.Status != ExtensionBuildReady || build.ArtifactPath == "" {
		return BrowserExtensionArtifactMetadata{}, errors.New("ready extension artifact not found")
	}
	if expectedDigest != build.ArtifactSHA256 {
		return BrowserExtensionArtifactMetadata{}, errors.New("expected digest does not match the artifact owner's receipt")
	}
	pathInfo, err := os.Lstat(build.ArtifactPath)
	if err != nil || !pathInfo.Mode().IsRegular() {
		return BrowserExtensionArtifactMetadata{}, errors.New("release artifact is unavailable or not a regular file")
	}
	file, err := os.Open(build.ArtifactPath)
	if err != nil {
		return BrowserExtensionArtifactMetadata{}, errors.New("release artifact is unavailable")
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil || !stat.Mode().IsRegular() || stat.Size() <= 0 || stat.Size() > maxBrowserExtensionArtifactBytes || stat.Size() != build.ArtifactBytes {
		return BrowserExtensionArtifactMetadata{}, errors.New("release artifact is not a bounded regular file")
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, io.LimitReader(file, maxBrowserExtensionArtifactBytes+1)); err != nil {
		return BrowserExtensionArtifactMetadata{}, fmt.Errorf("hash release artifact: %w", err)
	}
	if got := hex.EncodeToString(hash.Sum(nil)); got != build.ArtifactSHA256 {
		return BrowserExtensionArtifactMetadata{}, errors.New("release artifact failed integrity verification")
	}
	archive, err := zip.NewReader(file, stat.Size())
	if err != nil {
		return BrowserExtensionArtifactMetadata{}, errors.New("release artifact is not a valid ZIP")
	}
	var manifestFile *zip.File
	probeFiles := map[string]*zip.File{}
	for _, entry := range archive.File {
		if entry.Name == "manifest.json" {
			if manifestFile != nil {
				return BrowserExtensionArtifactMetadata{}, errors.New("release artifact contains multiple manifests")
			}
			manifestFile = entry
		}
		if entry.Name == browserPermissionProbePath || entry.Name == "__vrooli_bas/permission-report.js" {
			probeFiles[entry.Name] = entry
		}
	}
	if manifestFile == nil || manifestFile.UncompressedSize64 == 0 || manifestFile.UncompressedSize64 > maxBrowserExtensionManifestBytes {
		return BrowserExtensionArtifactMetadata{}, errors.New("release artifact has no bounded manifest.json")
	}
	reader, err := manifestFile.Open()
	if err != nil {
		return BrowserExtensionArtifactMetadata{}, errors.New("release artifact manifest cannot be opened")
	}
	manifestBytes, readErr := io.ReadAll(io.LimitReader(reader, maxBrowserExtensionManifestBytes+1))
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil || len(manifestBytes) > maxBrowserExtensionManifestBytes {
		return BrowserExtensionArtifactMetadata{}, errors.New("release artifact manifest cannot be read within size limit")
	}
	var manifest extensionManifestForBrowserArtifact
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return BrowserExtensionArtifactMetadata{}, errors.New("release artifact manifest is invalid JSON")
	}
	if manifest.ManifestVersion != 3 || strings.TrimSpace(manifest.Name) == "" || strings.TrimSpace(manifest.Version) == "" {
		return BrowserExtensionArtifactMetadata{}, errors.New("release artifact manifest does not meet the MV3 identity contract")
	}
	manifestHash := sha256.Sum256(manifestBytes)
	manifestDigest := hex.EncodeToString(manifestHash[:])
	if build.ManifestSHA256 == "" || manifestDigest != build.ManifestSHA256 {
		return BrowserExtensionArtifactMetadata{}, errors.New("release artifact manifest differs from the artifact owner's validated manifest")
	}
	extensionID, err := chromeExtensionID(manifest.Key)
	if err != nil {
		return BrowserExtensionArtifactMetadata{}, fmt.Errorf("derive extension id from release artifact: %w", err)
	}
	permissionProbe := ""
	if len(probeFiles) != 0 {
		if len(probeFiles) != 2 {
			return BrowserExtensionArtifactMetadata{}, errors.New("release artifact has an incomplete BAS permission-report contract")
		}
		for path, expected := range map[string][]byte{
			browserPermissionProbePath:          []byte(browserPermissionProbeHTML),
			"__vrooli_bas/permission-report.js": []byte(browserPermissionProbeJS),
		} {
			reader, openErr := probeFiles[path].Open()
			if openErr != nil {
				return BrowserExtensionArtifactMetadata{}, errors.New("release artifact BAS permission-report contract cannot be read")
			}
			content, readErr := io.ReadAll(io.LimitReader(reader, int64(len(expected))+1))
			closeErr := reader.Close()
			if readErr != nil || closeErr != nil || !bytes.Equal(content, expected) {
				return BrowserExtensionArtifactMetadata{}, errors.New("release artifact BAS permission-report contract does not match the supported owner source")
			}
		}
		permissionProbe = "chrome.permissions.getAll.v1"
	}
	return BrowserExtensionArtifactMetadata{
		ArtifactRef: buildID, ArtifactSHA256: build.ArtifactSHA256,
		ManifestSHA256: manifestDigest, ExtensionID: extensionID, Version: manifest.Version,
		Permissions:     sortedUnique(append(manifest.Permissions, manifest.OptionalPermissions...)),
		HostPermissions: sortedUnique(append(manifest.HostPermissions, manifest.OptionalHostPermissions...)),
		PermissionProbe: permissionProbe,
	}, nil
}

func sortedUnique(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			set[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func browserExtensionArtifactMetadataHandler(w http.ResponseWriter, r *http.Request) {
	buildID := mux.Vars(r)["build_id"]
	metadata, err := buildManager.VerifyBrowserExtensionArtifact(buildID, r.URL.Query().Get("expected_digest"))
	if err != nil {
		respondWithError(w, http.StatusConflict, err.Error())
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	writeJSON(w, metadata, "verified browser extension artifact metadata")
}
