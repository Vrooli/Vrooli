package agentaccess

import (
	"archive/zip"
	"bytes"
	"context"
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
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/browser-automation-studio/automation/session"
	"github.com/vrooli/browser-automation-studio/internal/sessionpermission"
)

const (
	maxExtensionArchiveBytes     = 64 << 20
	maxExtensionUnpackedBytes    = 128 << 20
	maxExtensionManifestBytes    = 1 << 20
	maxExtensionArchiveFileCount = 4096
	managedExtensionStoreName    = "vrooli-bas-managed-extensions"
)

var extensionDigestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

// ManagedExtensionControllerOption configures the BAS-owned extension adapter.
type ManagedExtensionControllerOption func(*managedExtensionControllerConfig)

type managedExtensionControllerConfig struct {
	client *http.Client
	root   string
}

// WithManagedExtensionHTTPClient injects the artifact-owner transport. The
// same client is used only for scenario-to-extension's ZIP download endpoint.
func WithManagedExtensionHTTPClient(client *http.Client) ManagedExtensionControllerOption {
	return func(config *managedExtensionControllerConfig) { config.client = client }
}

// WithManagedExtensionStagingRoot changes the private, API-owned staging root.
// It is intended for owner tests; production uses the system temp directory.
func WithManagedExtensionStagingRoot(root string) ManagedExtensionControllerOption {
	return func(config *managedExtensionControllerConfig) { config.root = root }
}

type stagedExtension struct {
	path     string
	artifact ExtensionArtifact
}

// ManagedExtensionController connects Broker-approved extension lifecycle
// calls to the existing session manager and its exact Playwright-driver lease.
// It has no browser client, CDP endpoint, or lifecycle of its own.
type ManagedExtensionController struct {
	manager     *session.Manager
	artifactURL string
	client      *http.Client
	root        string
	mu          sync.Mutex
	staged      map[string]stagedExtension
}

var _ ExtensionController = (*ManagedExtensionController)(nil)

func NewManagedExtensionController(manager *session.Manager, artifactOwnerURL string, options ...ManagedExtensionControllerOption) (*ManagedExtensionController, error) {
	if manager == nil {
		return nil, errors.New("BAS session manager is required for extension controller")
	}
	parsed, err := url.Parse(strings.TrimSpace(artifactOwnerURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return nil, errors.New("scenario-to-extension URL must be an absolute http(s) origin without credentials or path")
	}
	config := managedExtensionControllerConfig{
		client: &http.Client{Timeout: 30 * time.Second},
		root:   filepath.Join(os.TempDir(), managedExtensionStoreName),
	}
	for _, option := range options {
		option(&config)
	}
	if config.client == nil {
		return nil, errors.New("scenario-to-extension HTTP client is required")
	}
	if !filepath.IsAbs(config.root) {
		return nil, errors.New("managed extension staging root must be absolute")
	}
	if err := os.MkdirAll(config.root, 0o700); err != nil {
		return nil, fmt.Errorf("create managed extension staging root: %w", err)
	}
	if err := os.Chmod(config.root, 0o700); err != nil {
		return nil, fmt.Errorf("secure managed extension staging root: %w", err)
	}
	return &ManagedExtensionController{
		manager: manager, artifactURL: strings.TrimRight(parsed.String(), "/"),
		client: config.client, root: config.root, staged: make(map[string]stagedExtension),
	}, nil
}

func (c *ManagedExtensionController) Install(ctx context.Context, request ExtensionInstallRequest) (ExtensionReceipt, error) {
	return c.lifecycle(ctx, request.Binding, request.Artifact, "install", "installed", "/extensions/install")
}

func (c *ManagedExtensionController) Load(ctx context.Context, request ExtensionLifecycleRequest) (ExtensionReceipt, error) {
	return c.lifecycle(ctx, request.Binding, request.Artifact, "load", "loaded", "/extensions/load")
}

func (c *ManagedExtensionController) Reload(ctx context.Context, request ExtensionLifecycleRequest) (ExtensionReceipt, error) {
	return c.lifecycle(ctx, request.Binding, request.Artifact, "reload", "reloaded", "/extensions/reload")
}

func (c *ManagedExtensionController) Unload(ctx context.Context, request ExtensionLifecycleRequest) (ExtensionReceipt, error) {
	response, receipt, err := c.lifecycleResponse(ctx, request.Binding, request.Artifact, "unload", "unloaded", "/extensions/unload")
	if err != nil {
		return ExtensionReceipt{}, err
	}
	if response.Extension != nil {
		return ExtensionReceipt{}, errors.New("driver reported an extension still present after unload")
	}
	c.removeStaged(request.Binding, request.Artifact)
	return receipt, nil
}

// FinalizeClosedSession removes session-bound extension staging only after the
// canonical BAS session manager has closed the browser that could reference it.
// Failed removals stay indexed so a repeated Broker.Close can retry cleanup.
func (c *ManagedExtensionController) FinalizeClosedSession(ctx context.Context, sessionID string) error {
	if strings.TrimSpace(sessionID) == "" {
		return errors.New("closed BAS session id is required for extension staging cleanup")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, active := c.manager.Get(sessionID); active {
		return errors.New("BAS session owner has not closed before extension staging finalization")
	}
	prefix := sessionID + "\x00"
	c.mu.Lock()
	staged := make(map[string]stagedExtension)
	for key, item := range c.staged {
		if strings.HasPrefix(key, prefix) {
			staged[key] = item
		}
	}
	c.mu.Unlock()
	var cleanupErrors []error
	for key, item := range staged {
		if err := ctx.Err(); err != nil {
			cleanupErrors = append(cleanupErrors, err)
			break
		}
		if err := removeStagedDirectory(item.path); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("remove staged extension artifact for closed session: %w", err))
			continue
		}
		if err := ctx.Err(); err != nil {
			cleanupErrors = append(cleanupErrors, err)
			break
		}
		c.mu.Lock()
		if current, exists := c.staged[key]; exists && current.path == item.path {
			delete(c.staged, key)
		}
		c.mu.Unlock()
	}
	return errors.Join(cleanupErrors...)
}

func (c *ManagedExtensionController) Validate(ctx context.Context, request ExtensionLifecycleRequest) (ExtensionValidation, error) {
	response, receipt, err := c.lifecycleResponse(ctx, request.Binding, request.Artifact, "validate", "valid", "/extensions/validate")
	if err != nil {
		return ExtensionValidation{}, err
	}
	checks := map[string]bool{
		"installed": response.Extension != nil,
		"enabled":   response.Extension != nil && response.Extension.Enabled,
		"id":        response.Extension != nil && response.Extension.ID == request.Artifact.ExtensionID,
		"version":   response.Extension != nil && response.Extension.Version == request.Artifact.Version,
	}
	valid := true
	for _, passed := range checks {
		valid = valid && passed
	}
	return ExtensionValidation{Receipt: receipt, Valid: valid, Checks: checks}, nil
}

func (c *ManagedExtensionController) Inventory(ctx context.Context, binding ExtensionSessionBinding) (ExtensionInventoryReceipt, error) {
	if err := validateExtensionBinding(binding); err != nil {
		return ExtensionInventoryReceipt{}, err
	}
	c.mu.Lock()
	var inventoryArtifacts []map[string]any
	prefix := stagedExtensionBindingKey(binding) + "\x00"
	for key, staged := range c.staged {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		a := staged.artifact
		inventoryArtifacts = append(inventoryArtifacts, map[string]any{
			"directory_path": staged.path,
			"artifact": map[string]any{
				"extension_id": a.ExtensionID, "artifact_ref": a.ArtifactRef,
				"artifact_sha256": a.Digest, "manifest_sha256": a.ManifestSHA256,
				"version": a.Version, "permissions": a.Permissions,
				"host_permissions": a.HostPermissions, "permission_probe": a.PermissionProbe,
			},
		})
	}
	c.mu.Unlock()
	method := http.MethodGet
	var body []byte
	if len(inventoryArtifacts) > 0 {
		method = http.MethodPost
		var err error
		body, err = json.Marshal(map[string]any{"session_id": binding.SessionID, "profile_id": binding.ProfileID, "artifacts": inventoryArtifacts})
		if err != nil {
			return ExtensionInventoryReceipt{}, errors.New("could not encode verified managed extension inventory request")
		}
	}
	resp, err := c.manager.ManagedExtensionRequest(ctx, binding.SessionID, binding.OwnerID, binding.LeaseID, binding.ProfileID, binding.PermissionRef, binding.CallerID, method, "/extensions/inventory", body)
	if err != nil {
		return ExtensionInventoryReceipt{}, fmt.Errorf("read managed browser extension inventory: %w", err)
	}
	var result struct {
		SessionID  string `json:"session_id"`
		Extensions []struct {
			ID                 string                                        `json:"id"`
			Name               string                                        `json:"name"`
			Version            string                                        `json:"version"`
			Enabled            bool                                          `json:"enabled"`
			Path               string                                        `json:"path"`
			Permissions        []string                                      `json:"permissions,omitempty"`
			HostPermissions    []string                                      `json:"host_permissions,omitempty"`
			PermissionEvidence sessionpermission.ExtensionPermissionEvidence `json:"permission_evidence"`
		} `json:"extensions"`
	}
	if err := decodeDriverResponse(resp, &result); err != nil {
		return ExtensionInventoryReceipt{}, err
	}
	if result.SessionID != binding.SessionID {
		return ExtensionInventoryReceipt{}, errors.New("driver extension inventory belongs to another BAS session")
	}
	entries := make([]sessionpermission.ExtensionInventoryEntry, 0, len(result.Extensions))
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, item := range result.Extensions {
		// These are only present if the driver owner has a live-grant source.
		// Chromium's Extensions.getExtensions currently omits permissions, so
		// absent values stay absent; verified manifest declarations are separate.
		entry := sessionpermission.ExtensionInventoryEntry{
			ID: item.ID, Name: item.Name, Version: item.Version, Enabled: item.Enabled,
			Permissions:        append([]string(nil), item.Permissions...),
			HostPermissions:    append([]string(nil), item.HostPermissions...),
			PermissionEvidence: item.PermissionEvidence,
		}
		if item.Path != "" {
			stagedArtifact, found := stagedExtensionForID(c.staged, binding, item.ID)
			if found && filepath.Clean(item.Path) == filepath.Clean(stagedArtifact.path) {
				if item.PermissionEvidence.Source == sessionpermission.ExtensionPermissionSourceChromeGetAll && stagedArtifact.artifact.PermissionProbe != extensionPermissionProbeChromeGetAllV1 {
					return ExtensionInventoryReceipt{}, errors.New("driver reported the Chrome permission probe for an artifact that does not contain it")
				}
				if item.PermissionEvidence.Status == sessionpermission.ExtensionPermissionEvidenceEffective && item.PermissionEvidence.Source != sessionpermission.ExtensionPermissionSourceChromeGetAll {
					return ExtensionInventoryReceipt{}, errors.New("driver reported effective managed permissions from an unsupported source")
				}
				entry.DeclaredPermissions = append([]string(nil), stagedArtifact.artifact.Permissions...)
				entry.DeclaredHostPermissions = append([]string(nil), stagedArtifact.artifact.HostPermissions...)
				entry.ArtifactSHA256 = stagedArtifact.artifact.Digest
				entry.ManifestSHA256 = stagedArtifact.artifact.ManifestSHA256
				entry.ArtifactVerified = true
			}
		}
		entries = append(entries, entry)
	}
	return ExtensionInventoryReceipt{Binding: binding, ObservedAt: time.Now().UTC(), Extensions: entries}, nil
}

type driverExtensionReceipt struct {
	SessionID string `json:"session_id"`
	Extension *struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Version string `json:"version"`
		Enabled bool   `json:"enabled"`
		Path    string `json:"path"`
	} `json:"extension"`
}

func (c *ManagedExtensionController) lifecycle(ctx context.Context, binding ExtensionSessionBinding, artifact ExtensionArtifact, operation, state, suffix string) (ExtensionReceipt, error) {
	_, receipt, err := c.lifecycleResponse(ctx, binding, artifact, operation, state, suffix)
	return receipt, err
}

func (c *ManagedExtensionController) lifecycleResponse(ctx context.Context, binding ExtensionSessionBinding, artifact ExtensionArtifact, operation, state, suffix string) (driverExtensionReceipt, ExtensionReceipt, error) {
	if err := validateExtensionBinding(binding); err != nil {
		return driverExtensionReceipt{}, ExtensionReceipt{}, err
	}
	if binding.Backend != string(sessionpermission.BackendManaged) {
		return driverExtensionReceipt{}, ExtensionReceipt{}, errors.New("managed extension lifecycle is unsupported outside the BAS managed backend")
	}
	if err := validateVerifiedArtifact(artifact); err != nil {
		return driverExtensionReceipt{}, ExtensionReceipt{}, err
	}
	path, err := c.stageArtifact(ctx, binding, artifact)
	if err != nil {
		return driverExtensionReceipt{}, ExtensionReceipt{}, err
	}
	body, err := json.Marshal(map[string]any{
		"directory_path": path,
		"artifact": map[string]any{
			"extension_id": artifact.ExtensionID, "artifact_ref": artifact.ArtifactRef,
			"artifact_sha256": artifact.Digest, "manifest_sha256": artifact.ManifestSHA256,
			"version": artifact.Version, "permissions": artifact.Permissions,
			"host_permissions": artifact.HostPermissions,
			"permission_probe": artifact.PermissionProbe,
		},
	})
	if err != nil {
		return driverExtensionReceipt{}, ExtensionReceipt{}, err
	}
	resp, err := c.manager.ManagedExtensionRequest(ctx, binding.SessionID, binding.OwnerID, binding.LeaseID, binding.ProfileID, binding.PermissionRef, binding.CallerID, http.MethodPost, suffix, body)
	if err != nil {
		return driverExtensionReceipt{}, ExtensionReceipt{}, fmt.Errorf("BAS managed extension %s failed: %w", operation, err)
	}
	var driverResult driverExtensionReceipt
	if err := decodeDriverResponse(resp, &driverResult); err != nil {
		return driverExtensionReceipt{}, ExtensionReceipt{}, err
	}
	if driverResult.SessionID != binding.SessionID {
		return driverExtensionReceipt{}, ExtensionReceipt{}, errors.New("driver extension receipt belongs to another BAS session")
	}
	if operation != "unload" {
		if driverResult.Extension == nil || driverResult.Extension.ID != artifact.ExtensionID || driverResult.Extension.Version != artifact.Version || !driverResult.Extension.Enabled || filepath.Clean(driverResult.Extension.Path) != filepath.Clean(path) {
			return driverExtensionReceipt{}, ExtensionReceipt{}, errors.New("driver extension receipt does not match the verified installed artifact")
		}
	}
	c.mu.Lock()
	c.staged[stagedExtensionKey(binding, artifact)] = stagedExtension{path: path, artifact: artifact}
	c.mu.Unlock()
	receipt := ExtensionReceipt{
		SessionID: binding.SessionID, Backend: binding.Backend, ProfileID: binding.ProfileID,
		PermissionRef: binding.PermissionRef, OwnerID: binding.OwnerID, LeaseID: binding.LeaseID,
		CallerID: binding.CallerID, ArtifactRef: artifact.ArtifactRef, Operation: operation,
		ExtensionID: artifact.ExtensionID, Version: artifact.Version, Digest: artifact.Digest,
		ManifestSHA256: artifact.ManifestSHA256, State: state, OccurredAt: time.Now().UTC(),
	}
	return driverResult, receipt, nil
}

func (c *ManagedExtensionController) stageArtifact(ctx context.Context, binding ExtensionSessionBinding, artifact ExtensionArtifact) (string, error) {
	key := stagedExtensionKey(binding, artifact)
	c.mu.Lock()
	if staged, ok := c.staged[key]; ok {
		c.mu.Unlock()
		return staged.path, nil
	}
	c.mu.Unlock()
	archiveBytes, err := c.downloadArtifact(ctx, artifact)
	if err != nil {
		return "", err
	}
	if err := verifyExtensionArchive(archiveBytes, artifact); err != nil {
		return "", err
	}
	dir := filepath.Join(c.root, artifact.Digest, bindingPathKey(binding))
	if err := os.MkdirAll(filepath.Dir(dir), 0o700); err != nil {
		return "", fmt.Errorf("create artifact staging directory: %w", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("clear prior artifact staging directory: %w", err)
	}
	if err := os.Mkdir(dir, 0o700); err != nil {
		return "", fmt.Errorf("create artifact staging directory: %w", err)
	}
	if err := extractExtensionArchive(archiveBytes, dir); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	if err := os.Chmod(dir, 0o500); err != nil {
		return "", fmt.Errorf("make staged artifact read-only: %w", err)
	}
	staged := stagedExtension{path: dir, artifact: artifact}
	c.mu.Lock()
	c.staged[key] = staged
	c.mu.Unlock()
	return dir, nil
}

func (c *ManagedExtensionController) removeStaged(binding ExtensionSessionBinding, artifact ExtensionArtifact) {
	key := stagedExtensionKey(binding, artifact)
	c.mu.Lock()
	staged, ok := c.staged[key]
	delete(c.staged, key)
	c.mu.Unlock()
	if ok {
		_ = removeStagedDirectory(staged.path)
	}
}

func removeStagedDirectory(root string) error {
	if _, err := os.Stat(root); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		mode := os.FileMode(0o600)
		if entry.IsDir() {
			mode = 0o700
		}
		return os.Chmod(path, mode)
	})
	removeErr := os.RemoveAll(root)
	return errors.Join(walkErr, removeErr)
}

func stagedExtensionForID(staged map[string]stagedExtension, binding ExtensionSessionBinding, extensionID string) (stagedExtension, bool) {
	prefix := stagedExtensionBindingKey(binding) + "\x00" + extensionID + "\x00"
	for key, artifact := range staged {
		if strings.HasPrefix(key, prefix) {
			return artifact, true
		}
	}
	return stagedExtension{}, false
}

func (c *ManagedExtensionController) downloadArtifact(ctx context.Context, artifact ExtensionArtifact) ([]byte, error) {
	endpoint := c.artifactURL + "/api/v1/extension/download/" + url.PathEscape(artifact.ArtifactRef)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	client := *c.client
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download verified extension artifact: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("scenario-to-extension artifact download returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxExtensionArchiveBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxExtensionArchiveBytes {
		return nil, errors.New("scenario-to-extension returned an empty or oversized artifact")
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != artifact.Digest {
		return nil, errors.New("downloaded artifact bytes differ from the verified scenario-to-extension digest")
	}
	return data, nil
}

type extensionManifest struct {
	ManifestVersion         int      `json:"manifest_version"`
	Name                    string   `json:"name"`
	Version                 string   `json:"version"`
	Key                     string   `json:"key"`
	Permissions             []string `json:"permissions"`
	OptionalPermissions     []string `json:"optional_permissions"`
	HostPermissions         []string `json:"host_permissions"`
	OptionalHostPermissions []string `json:"optional_host_permissions"`
}

func verifyExtensionArchive(data []byte, artifact ExtensionArtifact) error {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil || len(archive.File) == 0 || len(archive.File) > maxExtensionArchiveFileCount {
		return errors.New("verified artifact is not a bounded extension ZIP")
	}
	var manifestBytes []byte
	permissionProbeFound, permissionProbeScriptFound := false, false
	for _, entry := range archive.File {
		if entry.Name == "__vrooli_bas/permission-report.html" {
			permissionProbeFound = true
		}
		if entry.Name == "__vrooli_bas/permission-report.js" {
			permissionProbeScriptFound = true
		}
		if entry.Name != "manifest.json" {
			continue
		}
		if manifestBytes != nil || entry.UncompressedSize64 == 0 || entry.UncompressedSize64 > maxExtensionManifestBytes {
			return errors.New("artifact contains a duplicate or oversized manifest")
		}
		reader, err := entry.Open()
		if err != nil {
			return errors.New("artifact manifest cannot be opened")
		}
		manifestBytes, err = io.ReadAll(io.LimitReader(reader, maxExtensionManifestBytes+1))
		closeErr := reader.Close()
		if err != nil || closeErr != nil || len(manifestBytes) > maxExtensionManifestBytes {
			return errors.New("artifact manifest cannot be read within bounds")
		}
	}
	if manifestBytes == nil {
		return errors.New("artifact has no root manifest.json")
	}
	if artifact.PermissionProbe == extensionPermissionProbeChromeGetAllV1 && (!permissionProbeFound || !permissionProbeScriptFound) {
		return errors.New("artifact owner declared a runtime permission probe that is incomplete in the verified ZIP")
	}
	if artifact.PermissionProbe == "" && (permissionProbeFound || permissionProbeScriptFound) {
		return errors.New("artifact contains a runtime permission probe not declared by its artifact owner")
	}
	manifestDigest := sha256.Sum256(manifestBytes)
	if hex.EncodeToString(manifestDigest[:]) != artifact.ManifestSHA256 {
		return errors.New("artifact manifest digest differs from the verifier receipt")
	}
	var manifest extensionManifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return errors.New("artifact manifest is invalid JSON")
	}
	if manifest.ManifestVersion != 3 || manifest.Name == "" || manifest.Version != artifact.Version {
		return errors.New("artifact manifest identity differs from the verifier receipt")
	}
	id, err := extensionIDFromKey(manifest.Key)
	if err != nil || id != artifact.ExtensionID {
		return errors.New("artifact manifest public key does not match the verifier extension id")
	}
	permissions := sortedUnique(append(append([]string{}, manifest.Permissions...), manifest.OptionalPermissions...))
	hostPermissions := sortedUnique(append(append([]string{}, manifest.HostPermissions...), manifest.OptionalHostPermissions...))
	if !equalStrings(permissions, sortedUnique(artifact.Permissions)) || !equalStrings(hostPermissions, sortedUnique(artifact.HostPermissions)) {
		return errors.New("artifact manifest permission set differs from the verifier receipt")
	}
	return nil
}

func extensionIDFromKey(encoded string) (string, error) {
	der, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil || len(der) == 0 {
		return "", errors.New("manifest key is not base64")
	}
	publicKey, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return "", err
	}
	if key, ok := publicKey.(*rsa.PublicKey); !ok || key.N == nil || key.N.BitLen() < 2048 {
		return "", errors.New("manifest key must be RSA with at least 2048 bits")
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

func extractExtensionArchive(data []byte, destination string) error {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return errors.New("artifact is not a valid extension ZIP")
	}
	base := filepath.Clean(destination)
	var total uint64
	for _, entry := range archive.File {
		name := filepath.FromSlash(entry.Name)
		clean := filepath.Clean(name)
		if clean == "." || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || strings.Contains(entry.Name, "\\") {
			return errors.New("artifact contains an unsafe ZIP entry path")
		}
		mode := entry.Mode()
		if mode&os.ModeSymlink != 0 || (!mode.IsRegular() && !mode.IsDir()) {
			return errors.New("artifact contains a non-regular ZIP entry")
		}
		total += entry.UncompressedSize64
		if total > maxExtensionUnpackedBytes {
			return errors.New("artifact expands beyond the allowed unpacked size")
		}
		path := filepath.Join(base, clean)
		if !strings.HasPrefix(path, base+string(filepath.Separator)) {
			return errors.New("artifact ZIP entry escaped its staging directory")
		}
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0o700); err != nil {
				return fmt.Errorf("create extension directory: %w", err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return fmt.Errorf("create extension asset directory: %w", err)
		}
		reader, err := entry.Open()
		if err != nil {
			return errors.New("open extension asset from ZIP")
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o400)
		if err != nil {
			_ = reader.Close()
			return fmt.Errorf("create staged extension asset: %w", err)
		}
		_, copyErr := io.Copy(file, io.LimitReader(reader, int64(entry.UncompressedSize64)+1))
		closeErr := file.Close()
		readerErr := reader.Close()
		if copyErr != nil || closeErr != nil || readerErr != nil {
			return errors.New("write staged extension asset")
		}
		if err := os.Chmod(path, 0o400); err != nil {
			return fmt.Errorf("secure staged extension asset: %w", err)
		}
	}
	return nil
}

func decodeDriverResponse(resp *http.Response, target any) error {
	if resp == nil || resp.Body == nil {
		return errors.New("Playwright driver returned an empty extension response")
	}
	defer resp.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode Playwright driver extension response: %w", err)
	}
	return nil
}

func validateExtensionBinding(binding ExtensionSessionBinding) error {
	if binding.SessionID == "" || binding.Backend == "" || binding.ProfileID == "" || binding.PermissionRef == "" || binding.OwnerID == "" || binding.LeaseID == "" || binding.CallerID == "" {
		return errors.New("extension session binding is incomplete")
	}
	return nil
}

func validateVerifiedArtifact(artifact ExtensionArtifact) error {
	if artifact.ArtifactRef == "" || artifact.ExtensionID == "" || !extensionDigestPattern.MatchString(artifact.Digest) || !extensionDigestPattern.MatchString(artifact.ManifestSHA256) || artifact.Version == "" {
		return errors.New("verified extension artifact metadata is incomplete")
	}
	if artifact.PermissionProbe != "" && artifact.PermissionProbe != extensionPermissionProbeChromeGetAllV1 {
		return errors.New("verified extension artifact names an unsupported permission probe")
	}
	return nil
}

func stagedExtensionKey(binding ExtensionSessionBinding, artifact ExtensionArtifact) string {
	return stagedExtensionBindingKey(binding) + "\x00" + artifact.ExtensionID + "\x00" + artifact.Digest
}

func bindingPathKey(binding ExtensionSessionBinding) string {
	hash := sha256.Sum256([]byte(stagedExtensionBindingKey(binding)))
	return hex.EncodeToString(hash[:])
}

func stagedExtensionBindingKey(binding ExtensionSessionBinding) string {
	return strings.Join([]string{binding.SessionID, binding.OwnerID, binding.LeaseID, binding.ProfileID, binding.PermissionRef, binding.CallerID}, "\x00")
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

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
