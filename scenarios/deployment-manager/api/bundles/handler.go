package bundles

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"deployment-manager/codesigning"
	"deployment-manager/codesigning/generation"
	"deployment-manager/profiles"
	"deployment-manager/secrets"
)

// MergeSecretsRequest is the request body for merge-secrets endpoint.
type MergeSecretsRequest struct {
	Scenario string                 `json:"scenario"`
	Tier     string                 `json:"tier"`
	Manifest Manifest               `json:"manifest"`
	Raw      map[string]interface{} `json:"-"`
}

// AssembleBundleRequest is the request body for assemble endpoint.
type AssembleBundleRequest struct {
	Scenario       string `json:"scenario"`
	Tier           string `json:"tier"`
	ProfileID      string `json:"profile_id,omitempty"`
	IncludeSecrets *bool  `json:"include_secrets,omitempty"`
}

// ExportBundleRequest is the request body for bundle export endpoint.
type ExportBundleRequest struct {
	Scenario       string `json:"scenario"`
	Tier           string `json:"tier"`
	ProfileID      string `json:"profile_id,omitempty"`
	IncludeSecrets *bool  `json:"include_secrets,omitempty"`
	OutputDir      string `json:"output_dir,omitempty"`
	StageBundle    *bool  `json:"stage_bundle,omitempty"`
}

// ExportBundleResponse is the response for bundle export endpoint.
type ExportBundleResponse struct {
	Status       string   `json:"status"`
	Schema       string   `json:"schema"`
	Scenario     string   `json:"scenario"`
	Tier         string   `json:"tier"`
	Manifest     Manifest `json:"manifest"`
	Checksum     string   `json:"checksum"`
	GeneratedAt  string   `json:"generated_at"`
	ManifestPath string   `json:"manifest_path,omitempty"`
	OutputDir    string   `json:"output_dir,omitempty"`
}

type stageBundleResult struct {
	ManifestPath string
	Missing      []string
}

// Handler handles bundle-related requests.
type Handler struct {
	secretsClient *secrets.Client
	profileRepo   profiles.Repository
	signingRepo   codesigning.Repository
	signingGen    generation.Generator
	log           func(string, map[string]interface{})
}

// NewHandler creates a new bundles handler.
func NewHandler(secretsClient *secrets.Client, profileRepo profiles.Repository, log func(string, map[string]interface{})) *Handler {
	return &Handler{
		secretsClient: secretsClient,
		profileRepo:   profileRepo,
		signingGen:    generation.NewGenerator(nil),
		log:           log,
	}
}

// NewHandlerWithSigning creates a new bundles handler with signing support.
func NewHandlerWithSigning(secretsClient *secrets.Client, profileRepo profiles.Repository, signingRepo codesigning.Repository, log func(string, map[string]interface{})) *Handler {
	return &Handler{
		secretsClient: secretsClient,
		profileRepo:   profileRepo,
		signingRepo:   signingRepo,
		signingGen:    generation.NewGenerator(nil),
		log:           log,
	}
}

// ValidateBundle validates a desktop bundle manifest.
func (h *Handler) ValidateBundle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to read bundle: %v"}`, err), http.StatusBadRequest)
		return
	}
	if len(body) == 0 {
		http.Error(w, `{"error":"bundle manifest required"}`, http.StatusBadRequest)
		return
	}

	if err := ValidateManifestBytes(body); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"bundle failed validation","details":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "valid",
		"schema": "desktop.v0.1",
	})
}

// MergeBundleSecrets merges secret plans into a bundle manifest.
func (h *Handler) MergeBundleSecrets(w http.ResponseWriter, r *http.Request) {
	var req MergeSecretsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON: %v"}`, err), http.StatusBadRequest)
		return
	}
	if req.Scenario == "" {
		http.Error(w, `{"error":"scenario is required"}`, http.StatusBadRequest)
		return
	}
	if req.Tier == "" {
		req.Tier = "tier-2-desktop"
	}

	// Re-validate manifest before merging.
	rawPayload, err := json.Marshal(req.Manifest)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to marshal manifest: %v"}`, err), http.StatusBadRequest)
		return
	}
	if err := ValidateManifestBytes(rawPayload); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"manifest failed validation","details":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	secretPlans, err := h.secretsClient.FetchBundleSecrets(r.Context(), req.Scenario, req.Tier)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to load secret plans: %v"}`, err), http.StatusBadGateway)
		return
	}

	manifest := req.Manifest
	if err := ApplyBundleSecrets(&manifest, secretPlans); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to merge secrets: %v"}`, err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(manifest)
}

// AssembleBundle assembles a complete bundle manifest for a scenario.
func (h *Handler) AssembleBundle(w http.ResponseWriter, r *http.Request) {
	var req AssembleBundleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON: %v"}`, err), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Scenario) == "" {
		http.Error(w, `{"error":"scenario is required"}`, http.StatusBadRequest)
		return
	}
	if req.Tier == "" {
		req.Tier = "tier-2-desktop"
	}
	includeSecrets := true
	if req.IncludeSecrets != nil {
		includeSecrets = *req.IncludeSecrets
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	manifest, err := FetchSkeletonBundle(ctx, req.Scenario)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to build bundle","details":"%s"}`, err.Error()), http.StatusBadGateway)
		return
	}

	// Apply swaps from profile if provided
	if req.ProfileID != "" && h.profileRepo != nil {
		profileSwaps, err := h.profileRepo.GetSwaps(ctx, req.ProfileID)
		if err != nil && err != profiles.ErrNotFound {
			http.Error(w, fmt.Sprintf(`{"error":"failed to load profile swaps: %v"}`, err), http.StatusBadGateway)
			return
		}
		if len(profileSwaps) > 0 {
			h.applySwapsToManifest(manifest, profileSwaps)
		}
	}

	// Apply signing config from profile if available
	if req.ProfileID != "" {
		if err := h.loadSigningConfig(ctx, req.ProfileID, manifest); err != nil {
			h.log("warn", map[string]interface{}{
				"msg":        "failed to load signing config, continuing without",
				"profile_id": req.ProfileID,
				"error":      err.Error(),
			})
			// Non-fatal: continue without signing config
		}
	}

	if includeSecrets {
		secretPlans, err := h.secretsClient.FetchBundleSecrets(ctx, req.Scenario, req.Tier)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to load secret plans: %v"}`, err), http.StatusBadGateway)
			return
		}
		if err := ApplyBundleSecrets(manifest, secretPlans); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to merge secrets: %v"}`, err), http.StatusBadRequest)
			return
		}
	}

	// Validate assembled manifest to guarantee schema compliance before handing off.
	if scenarioRoot := resolveScenarioRoot(req.Scenario); scenarioRoot != "" {
		_ = populateAssetMetadata(manifest, scenarioRoot)
	}
	payload, _ := json.Marshal(manifest)
	if err := ValidateManifestBytes(payload); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"assembled manifest failed validation","details":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":   "assembled",
		"schema":   "desktop.v0.1",
		"manifest": manifest,
	})
}

// applySwapsToManifest adds profile swaps to the manifest's swap list.
func (h *Handler) applySwapsToManifest(manifest *Manifest, profileSwaps []profiles.Swap) {
	for _, ps := range profileSwaps {
		manifest.Swaps = append(manifest.Swaps, ManifestSwap{
			Original:    ps.From,
			Replacement: ps.To,
			Reason:      ps.Reason,
			Limitations: ps.Limitations,
		})
	}
}

// ExportBundle assembles and exports a signed bundle manifest with checksum.
// This is the endpoint for generating production-ready bundle.json files.
func (h *Handler) ExportBundle(w http.ResponseWriter, r *http.Request) {
	var req ExportBundleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON: %v"}`, err), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Scenario) == "" {
		http.Error(w, `{"error":"scenario is required"}`, http.StatusBadRequest)
		return
	}
	if req.Tier == "" {
		req.Tier = "tier-2-desktop"
	}
	includeSecrets := true
	if req.IncludeSecrets != nil {
		includeSecrets = *req.IncludeSecrets
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	manifest, err := FetchSkeletonBundle(ctx, req.Scenario)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to build bundle","details":"%s"}`, err.Error()), http.StatusBadGateway)
		return
	}

	// Apply swaps from profile if provided
	if req.ProfileID != "" && h.profileRepo != nil {
		profileSwaps, err := h.profileRepo.GetSwaps(ctx, req.ProfileID)
		if err != nil && err != profiles.ErrNotFound {
			http.Error(w, fmt.Sprintf(`{"error":"failed to load profile swaps: %v"}`, err), http.StatusBadGateway)
			return
		}
		if len(profileSwaps) > 0 {
			h.applySwapsToManifest(manifest, profileSwaps)
		}
	}

	// Apply signing config from profile if available
	if req.ProfileID != "" {
		if err := h.loadSigningConfig(ctx, req.ProfileID, manifest); err != nil {
			h.log("warn", map[string]interface{}{
				"msg":        "failed to load signing config, continuing without",
				"profile_id": req.ProfileID,
				"error":      err.Error(),
			})
			// Non-fatal: continue without signing config
		}
	}

	if includeSecrets {
		secretPlans, err := h.secretsClient.FetchBundleSecrets(ctx, req.Scenario, req.Tier)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to load secret plans: %v"}`, err), http.StatusBadGateway)
			return
		}
		if err := ApplyBundleSecrets(manifest, secretPlans); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to merge secrets: %v"}`, err), http.StatusBadRequest)
			return
		}
	}

	// Validate assembled manifest before export.
	if scenarioRoot := resolveScenarioRoot(req.Scenario); scenarioRoot != "" {
		_ = populateAssetMetadata(manifest, scenarioRoot)
	}
	payload, err := json.Marshal(manifest)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"failed to serialize manifest: %v"}`, err), http.StatusInternalServerError)
		return
	}
	if err := ValidateManifestBytes(payload); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"assembled manifest failed validation","details":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	// Calculate SHA256 checksum of the manifest content.
	hash := sha256.Sum256(payload)
	checksum := hex.EncodeToString(hash[:])

	generatedAt := time.Now().UTC().Format(time.RFC3339)

	response := ExportBundleResponse{
		Status:      "exported",
		Schema:      "desktop.v0.1",
		Scenario:    req.Scenario,
		Tier:        req.Tier,
		Manifest:    *manifest,
		Checksum:    checksum,
		GeneratedAt: generatedAt,
	}

	stageBundle := false
	if req.StageBundle != nil {
		stageBundle = *req.StageBundle
	}
	if req.OutputDir != "" && req.StageBundle == nil {
		stageBundle = true
	}

	if stageBundle {
		scenarioRoot := resolveScenarioRoot(req.Scenario)
		if scenarioRoot == "" {
			http.Error(w, `{"error":"scenario root not found"}`, http.StatusBadRequest)
			return
		}
		stageResult, err := stageBundleArtifacts(manifest, scenarioRoot, req.OutputDir)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"failed to stage bundle","details":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}
		if len(stageResult.Missing) > 0 {
			http.Error(w, fmt.Sprintf(`{"error":"bundle staging missing artifacts","details":"%s"}`, strings.Join(stageResult.Missing, ", ")), http.StatusUnprocessableEntity)
			return
		}
		response.ManifestPath = stageResult.ManifestPath
		response.OutputDir = req.OutputDir
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

// GenerateSigningConfigRequest is the request body for signing config generation.
type GenerateSigningConfigRequest struct {
	ProfileID    string   `json:"profile_id"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// GenerateSigningConfigResponse is the response for signing config generation.
type GenerateSigningConfigResponse struct {
	Status                string                 `json:"status"`
	ProfileID             string                 `json:"profile_id"`
	ElectronBuilderConfig map[string]interface{} `json:"electron_builder_config,omitempty"`
	Files                 map[string]string      `json:"files,omitempty"`
	Message               string                 `json:"message,omitempty"`
}

// GenerateSigningConfig generates electron-builder signing configuration and supporting files.
// This endpoint takes a profile ID and generates the electron-builder config,
// entitlements.plist, and notarize script based on the profile's signing settings.
// POST /api/v1/bundles/signing-config
func (h *Handler) GenerateSigningConfig(w http.ResponseWriter, r *http.Request) {
	var req GenerateSigningConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON: %v"}`, err), http.StatusBadRequest)
		return
	}

	if req.ProfileID == "" {
		http.Error(w, `{"error":"profile_id is required"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Check if signing repository is available
	if h.signingRepo == nil {
		http.Error(w, `{"error":"signing configuration not available"}`, http.StatusServiceUnavailable)
		return
	}

	// Fetch signing config from repository
	signingConfig, err := h.signingRepo.Get(ctx, req.ProfileID)
	if err != nil {
		h.log("error", map[string]interface{}{
			"msg":        "failed to get signing config",
			"profile_id": req.ProfileID,
			"error":      err.Error(),
		})
		http.Error(w, `{"error":"failed to retrieve signing configuration"}`, http.StatusInternalServerError)
		return
	}

	if signingConfig == nil || !signingConfig.Enabled {
		h.writeJSON(w, http.StatusOK, GenerateSigningConfigResponse{
			Status:    "disabled",
			ProfileID: req.ProfileID,
			Message:   "Code signing is not enabled for this profile",
		})
		return
	}

	// Generate electron-builder config
	ebConfig, err := generation.GenerateElectronBuilderJSON(signingConfig, nil)
	if err != nil {
		h.log("error", map[string]interface{}{
			"msg":   "failed to generate electron-builder config",
			"error": err.Error(),
		})
		http.Error(w, `{"error":"failed to generate electron-builder config"}`, http.StatusInternalServerError)
		return
	}

	// Generate all supporting files (entitlements, notarize script)
	generatedFiles, err := h.signingGen.GenerateAll(signingConfig)
	if err != nil {
		h.log("error", map[string]interface{}{
			"msg":   "failed to generate signing files",
			"error": err.Error(),
		})
		http.Error(w, `{"error":"failed to generate signing files"}`, http.StatusInternalServerError)
		return
	}

	// Convert byte slices to strings for JSON response
	files := make(map[string]string)
	for path, content := range generatedFiles {
		files[path] = string(content)
	}

	h.log("info", map[string]interface{}{
		"msg":         "generated signing config",
		"profile_id":  req.ProfileID,
		"files_count": len(files),
	})

	h.writeJSON(w, http.StatusOK, GenerateSigningConfigResponse{
		Status:                "generated",
		ProfileID:             req.ProfileID,
		ElectronBuilderConfig: ebConfig,
		Files:                 files,
		Message:               fmt.Sprintf("Generated %d signing file(s)", len(files)),
	})
}

// writeJSON writes a JSON response.
func (h *Handler) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// LoadSigningConfig loads signing configuration for a profile and applies it to the manifest.
// This is called during AssembleBundle and ExportBundle when a profile has signing configured.
func (h *Handler) loadSigningConfig(ctx context.Context, profileID string, manifest *Manifest) error {
	if h.signingRepo == nil || profileID == "" {
		return nil
	}

	signingConfig, err := h.signingRepo.Get(ctx, profileID)
	if err != nil {
		return fmt.Errorf("failed to load signing config: %w", err)
	}

	if signingConfig != nil && signingConfig.Enabled {
		manifest.CodeSigning = signingConfig
	}

	return nil
}

// resolveScenarioRoot returns the absolute scenario directory.
func resolveScenarioRoot(scenario string) string {
	if scenario == "" {
		return ""
	}
	root := os.Getenv("VROOLI_ROOT")
	if root == "" {
		home, _ := os.UserHomeDir()
		root = filepath.Join(home, "Vrooli")
	}
	return filepath.Join(root, "scenarios", scenario)
}

// populateAssetMetadata fills in missing/pending asset hashes and sizes using files on disk.
func populateAssetMetadata(manifest *Manifest, scenarioRoot string) error {
	if manifest == nil {
		return nil
	}
	// Expand ui-bundle assets to include file entries instead of directories.
	_ = expandUIAssets(manifest, scenarioRoot)

	for svcIdx, svc := range manifest.Services {
		for assetIdx, asset := range svc.Assets {
			if asset.SHA256 != "" && asset.SHA256 != "pending" && asset.SizeBytes > 0 {
				continue
			}
			abs := filepath.Join(scenarioRoot, filepath.FromSlash(asset.Path))
			info, err := os.Stat(abs)
			if err != nil || info.IsDir() {
				continue
			}
			hash, size, hErr := hashFile(abs)
			if hErr != nil {
				continue
			}
			manifest.Services[svcIdx].Assets[assetIdx].SHA256 = hash
			manifest.Services[svcIdx].Assets[assetIdx].SizeBytes = size
		}
	}
	return nil
}

func hashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func expandUIAssets(manifest *Manifest, scenarioRoot string) error {
	if manifest == nil {
		return nil
	}
	var firstErr error
	for svcIdx, svc := range manifest.Services {
		if !strings.EqualFold(svc.Type, "ui-bundle") {
			continue
		}
		uiRoot := filepath.Join(scenarioRoot, "ui", "dist")
		entries, err := os.ReadDir(uiRoot)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("read ui dist: %w", err)
			}
			continue
		}
		var assets []Asset
		for _, entry := range entries {
			err := filepath.WalkDir(filepath.Join(uiRoot, entry.Name()), func(path string, d os.DirEntry, werr error) error {
				if werr != nil {
					return werr
				}
				if d.IsDir() {
					return nil
				}
				rel, _ := filepath.Rel(scenarioRoot, path)
				hash, size, herr := hashFile(path)
				if herr != nil {
					if firstErr == nil {
						firstErr = herr
					}
					return nil
				}
				assets = append(assets, Asset{
					Path:      filepath.ToSlash(rel),
					SHA256:    hash,
					SizeBytes: size,
				})
				return nil
			})
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}
		if len(assets) > 0 {
			manifest.Services[svcIdx].Assets = assets
		}
	}
	return firstErr
}

func stageBundleArtifacts(manifest *Manifest, scenarioRoot, outputDir string) (stageBundleResult, error) {
	if manifest == nil {
		return stageBundleResult{}, fmt.Errorf("manifest is nil")
	}
	if strings.TrimSpace(outputDir) == "" {
		return stageBundleResult{}, fmt.Errorf("output_dir is required for bundle staging")
	}

	absOutput, err := filepath.Abs(outputDir)
	if err != nil {
		return stageBundleResult{}, fmt.Errorf("resolve output_dir: %w", err)
	}
	if err := os.MkdirAll(absOutput, 0o755); err != nil {
		return stageBundleResult{}, fmt.Errorf("create output_dir: %w", err)
	}

	var missing []string
	seen := make(map[string]bool)

	copyEntry := func(relPath string, isBinary bool) error {
		cleanRel, err := sanitizeRelPath(relPath)
		if err != nil {
			return err
		}
		if seen[cleanRel] {
			return nil
		}
		seen[cleanRel] = true
		src := filepath.Join(scenarioRoot, cleanRel)
		info, err := os.Stat(src)
		if err != nil {
			missing = append(missing, cleanRel)
			return nil
		}
		dest := filepath.Join(absOutput, cleanRel)
		if info.IsDir() {
			if isBinary {
				return fmt.Errorf("binary path is a directory: %s", cleanRel)
			}
			if err := copyDir(src, dest); err != nil {
				return err
			}
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		return copyFilePreserveMode(src, dest)
	}

	for _, svc := range manifest.Services {
		for _, bin := range svc.Binaries {
			if bin.Path == "" {
				continue
			}
			if err := copyEntry(bin.Path, true); err != nil {
				return stageBundleResult{}, err
			}
		}
		for _, asset := range svc.Assets {
			if asset.Path == "" {
				continue
			}
			if err := copyEntry(asset.Path, false); err != nil {
				return stageBundleResult{}, err
			}
		}
	}

	manifestPath := filepath.Join(absOutput, "bundle.json")
	payload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return stageBundleResult{}, fmt.Errorf("serialize manifest: %w", err)
	}
	if err := os.WriteFile(manifestPath, payload, 0o644); err != nil {
		return stageBundleResult{}, fmt.Errorf("write bundle.json: %w", err)
	}

	return stageBundleResult{
		ManifestPath: manifestPath,
		Missing:      missing,
	}, nil
}

func sanitizeRelPath(relPath string) (string, error) {
	if filepath.IsAbs(relPath) {
		return "", fmt.Errorf("path must be relative: %s", relPath)
	}
	clean := filepath.Clean(filepath.FromSlash(relPath))
	if clean == "." || clean == "" {
		return "", fmt.Errorf("invalid path: %s", relPath)
	}
	if strings.HasPrefix(clean, "..") {
		return "", fmt.Errorf("path escapes root: %s", relPath)
	}
	return clean, nil
}

func copyDir(src, dest string) error {
	return filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFilePreserveMode(path, target)
	})
}

func copyFilePreserveMode(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if info, err := os.Stat(src); err == nil {
		mode = info.Mode().Perm()
	}
	return os.WriteFile(dst, data, mode)
}
