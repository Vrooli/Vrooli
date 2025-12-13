// Package deployments provides deployment orchestration for bundled desktop apps.
package deployments

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"deployment-manager/build"
	"deployment-manager/bundles"
	"deployment-manager/profiles"
	"deployment-manager/shared"
)

// DeployDesktopRequest is the request for orchestrated desktop deployment.
type DeployDesktopRequest struct {
	// ProfileID is the profile to deploy (required)
	ProfileID string `json:"profile_id"`
	// OutputDir is where to place the bundle manifest and binaries
	OutputDir string `json:"output_dir,omitempty"`
	// Platforms to build for (optional, defaults to all)
	Platforms []string `json:"platforms,omitempty"`
	// SkipBuild skips binary compilation
	SkipBuild bool `json:"skip_build,omitempty"`
	// SkipValidation skips pre-flight validation
	SkipValidation bool `json:"skip_validation,omitempty"`
	// SkipPackaging skips invoking scenario-to-desktop (just assembles manifest and builds binaries)
	SkipPackaging bool `json:"skip_packaging,omitempty"`
	// SkipInstallers skips building platform installers (MSI/PKG/AppImage)
	SkipInstallers bool `json:"skip_installers,omitempty"`
	// DeploymentMode is the mode for the desktop app (bundled, external-server, cloud-api)
	DeploymentMode string `json:"deployment_mode,omitempty"`
	// DryRun shows what would be done without doing it
	DryRun bool `json:"dry_run,omitempty"`
	// SigningConfig is the optional signing configuration to apply before building
	// This is passed directly to scenario-to-desktop's signing API
	SigningConfig map[string]interface{} `json:"signing_config,omitempty"`
	// TimeoutSeconds allows callers to override the orchestration timeout window
	TimeoutSeconds int `json:"timeout_seconds,omitempty"`
}

// DeployDesktopResponse is the response from orchestrated deployment.
type DeployDesktopResponse struct {
	Status           string                `json:"status"`
	ProfileID        string                `json:"profile_id"`
	Scenario         string                `json:"scenario"`
	Steps            []OrchestrationStep   `json:"steps"`
	ManifestPath     string                `json:"manifest_path,omitempty"`
	BuildResults     *build.BuildAllResult `json:"build_results,omitempty"`
	DesktopBuildID   string                `json:"desktop_build_id,omitempty"`
	DesktopPath      string                `json:"desktop_path,omitempty"`
	InstallerBuildID string                `json:"installer_build_id,omitempty"`
	Installers       map[string]string     `json:"installers,omitempty"`
	Duration         string                `json:"duration,omitempty"`
	NextSteps        []string              `json:"next_steps,omitempty"`
}

// OrchestrationStep represents a single step in the orchestration.
type OrchestrationStep struct {
	Name     string `json:"name"`
	Status   string `json:"status"` // pending, running, success, failed, skipped, warning
	Duration string `json:"duration,omitempty"`
	Message  string `json:"message,omitempty"`
	Error    string `json:"error,omitempty"`
}

// Orchestrator handles the full desktop deployment workflow.
type Orchestrator struct {
	profileRepo profiles.Repository
	vrooli      string
	log         func(string, map[string]interface{})
}

// NewOrchestrator creates a new deployment orchestrator.
func NewOrchestrator(profileRepo profiles.Repository, log func(string, map[string]interface{})) *Orchestrator {
	vrooli := os.Getenv("VROOLI_ROOT")
	if vrooli == "" {
		vrooli = filepath.Join(os.Getenv("HOME"), "Vrooli")
	}
	return &Orchestrator{
		profileRepo: profileRepo,
		vrooli:      vrooli,
		log:         log,
	}
}

// DeployDesktop handles POST /api/v1/deploy-desktop requests.
func (o *Orchestrator) DeployDesktop(w http.ResponseWriter, r *http.Request) {
	var req DeployDesktopRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"invalid JSON: %v"}`, err), http.StatusBadRequest)
		return
	}

	if req.ProfileID == "" {
		http.Error(w, `{"error":"profile_id is required"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Minute)
	defer cancel()

	start := time.Now()

	response := &DeployDesktopResponse{
		ProfileID: req.ProfileID,
		Steps:     make([]OrchestrationStep, 0),
	}
	scenarioBaseDir := filepath.Join(o.vrooli, "scenarios")

	effectiveTimeout := time.Duration(req.TimeoutSeconds) * time.Second
	if effectiveTimeout <= 0 {
		effectiveTimeout = 10 * time.Minute
	}
	deploymentMode := req.DeploymentMode
	if deploymentMode == "" {
		deploymentMode = "bundled"
	}
	buildPlatforms := resolveBuildPlatforms(req.Platforms)
	installerTargets := resolveInstallerTargets(req.Platforms)

	// Step 1: Load profile
	step := o.startStep("Load profile")
	profile, err := o.profileRepo.Get(ctx, req.ProfileID)
	if err != nil {
		o.failStep(&step, fmt.Sprintf("failed to load profile: %v", err))
		response.Steps = append(response.Steps, step)
		response.Status = "failed"
		o.writeJSON(w, http.StatusBadGateway, response)
		return
	}
	if profile == nil {
		o.failStep(&step, "profile not found")
		response.Steps = append(response.Steps, step)
		response.Status = "failed"
		o.writeJSON(w, http.StatusNotFound, response)
		return
	}
	response.Scenario = profile.Scenario
	o.successStep(&step, fmt.Sprintf("loaded profile for scenario %s", profile.Scenario))
	response.Steps = append(response.Steps, step)

	// Step 2: Validate profile (unless skipped)
	if !req.SkipValidation {
		step = o.startStep("Validate profile")
		if err := o.validateProfile(ctx, req.ProfileID); err != nil {
			o.failStep(&step, err.Error())
			response.Steps = append(response.Steps, step)
			response.Status = "failed"
			o.writeJSON(w, http.StatusBadRequest, response)
			return
		}
		o.successStep(&step, "profile validation passed")
		response.Steps = append(response.Steps, step)
	} else {
		step = o.startStep("Validate profile")
		step.Status = "skipped"
		step.Message = "validation skipped by request"
		response.Steps = append(response.Steps, step)
	}

	// Step 2.5a: Apply signing config if provided
	if len(req.SigningConfig) > 0 {
		step = o.startStep("Apply signing configuration")
		if req.DryRun {
			step.Status = "skipped"
			step.Message = "dry run - would apply signing config"
		} else {
			if err := o.applySigningConfig(ctx, profile.Scenario, req.SigningConfig); err != nil {
				step.Status = "warning"
				step.Message = fmt.Sprintf("failed to apply signing config: %v", err)
				o.log("warn", map[string]interface{}{
					"msg":      "signing config application failed",
					"scenario": profile.Scenario,
					"error":    err.Error(),
				})
			} else {
				o.successStep(&step, "signing configuration applied to scenario-to-desktop")
			}
		}
		response.Steps = append(response.Steps, step)
	}

	// Step 2.5b: Check signing readiness (warning only, non-blocking)
	step = o.startStep("Check signing readiness")
	signingWarnings := o.checkSigningReadiness(ctx, profile.Scenario)
	if len(signingWarnings) > 0 {
		step.Status = "warning"
		step.Message = strings.Join(signingWarnings, "; ")
		o.log("warn", map[string]interface{}{
			"msg":      "signing not fully configured",
			"scenario": profile.Scenario,
			"issues":   signingWarnings,
		})
	} else {
		o.successStep(&step, "signing configuration ready")
	}
	response.Steps = append(response.Steps, step)

	// Step 3: Fetch bundle skeleton and apply profile settings
	step = o.startStep("Assemble manifest")
	manifest, err := bundles.FetchSkeletonBundle(ctx, profile.Scenario)
	if err != nil {
		o.failStep(&step, fmt.Sprintf("failed to fetch bundle skeleton: %v", err))
		response.Steps = append(response.Steps, step)
		response.Status = "failed"
		o.writeJSON(w, http.StatusBadGateway, response)
		return
	}

	// Apply swaps from profile
	profileSwaps, _ := o.profileRepo.GetSwaps(ctx, req.ProfileID)
	for _, ps := range profileSwaps {
		manifest.Swaps = append(manifest.Swaps, bundles.ManifestSwap{
			Original:    ps.From,
			Replacement: ps.To,
			Reason:      ps.Reason,
			Limitations: ps.Limitations,
		})
	}

	// Populate missing asset metadata (checksum/size) so runtime validation passes
	scenarioDir := filepath.Join(scenarioBaseDir, profile.Scenario)
	if err := populateAssetMetadata(manifest, scenarioDir); err != nil {
		step.Status = "warning"
		step.Message = fmt.Sprintf("assembled manifest with %d swaps (asset metadata partial: %v)", len(manifest.Swaps), err)
		o.log("warn", map[string]interface{}{
			"msg":      "asset metadata population incomplete",
			"scenario": profile.Scenario,
			"error":    err.Error(),
		})
	} else {
		step.Message = fmt.Sprintf("assembled manifest with %d swaps", len(manifest.Swaps))
	}
	o.successStep(&step, fmt.Sprintf("assembled manifest with %d swaps", len(manifest.Swaps)))
	response.Steps = append(response.Steps, step)

	// Step 3.5: Omit non-cross-platform CLI services with a warning
	step = o.startStep("Normalize CLI services")
	if manifest != nil {
		pruned, err := pruneNonCrossPlatformCLIs(manifest, filepath.Join(scenarioBaseDir, profile.Scenario))
		if err != nil {
			step.Status = "warning"
			step.Message = fmt.Sprintf("failed to normalize CLI services: %v", err)
			o.log("warn", map[string]interface{}{
				"msg":      "normalize cli services failed",
				"scenario": profile.Scenario,
				"error":    err.Error(),
			})
		} else if len(pruned) > 0 {
			step.Status = "warning"
			step.Message = fmt.Sprintf("omitted %d CLI service(s) not cross-platform: %s", len(pruned), strings.Join(pruned, ", "))
			o.log("warn", map[string]interface{}{
				"msg":       "omitted non-cross-platform cli services",
				"scenario":  profile.Scenario,
				"services":  strings.Join(pruned, ","),
				"remediate": "make cli cross-platform (see test-genie) to include in bundle",
			})
		} else {
			o.successStep(&step, "CLI services are cross-platform or none present")
		}
	} else {
		step.Status = "warning"
		step.Message = "manifest unavailable for CLI normalization"
	}
	response.Steps = append(response.Steps, step)

	// Determine output directory for manifest
	// Store manifest at scenario root so relative paths work correctly
	outputDir := req.OutputDir
	if outputDir == "" {
		outputDir = filepath.Join(o.vrooli, "scenarios", profile.Scenario)
	}

	// Step 4: Write manifest
	step = o.startStep("Export manifest")
	if !req.DryRun {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			o.failStep(&step, fmt.Sprintf("failed to create output dir: %v", err))
			response.Steps = append(response.Steps, step)
			response.Status = "failed"
			o.writeJSON(w, http.StatusInternalServerError, response)
			return
		}

		manifestPath := filepath.Join(outputDir, "bundle.json")
		manifestData, _ := json.MarshalIndent(manifest, "", "  ")
		if err := os.WriteFile(manifestPath, manifestData, 0o644); err != nil {
			o.failStep(&step, fmt.Sprintf("failed to write manifest: %v", err))
			response.Steps = append(response.Steps, step)
			response.Status = "failed"
			o.writeJSON(w, http.StatusInternalServerError, response)
			return
		}
		response.ManifestPath = manifestPath
		o.successStep(&step, fmt.Sprintf("wrote manifest to %s", manifestPath))
	} else {
		step.Status = "skipped"
		step.Message = "dry run - would write manifest"
	}
	response.Steps = append(response.Steps, step)

	// Step 5: Build binaries (unless skipped)
	if !req.SkipBuild {
		step = o.startStep("Build binaries")

		// Find services with build configs
		var buildableServices []bundles.ServiceEntry
		for _, svc := range manifest.Services {
			if svc.Build != nil {
				buildableServices = append(buildableServices, svc)
			}
		}

		if len(buildableServices) == 0 {
			step.Status = "skipped"
			step.Message = "no services with build configuration found"
			response.Steps = append(response.Steps, step)
		} else if req.DryRun {
			step.Status = "skipped"
			step.Message = fmt.Sprintf("dry run - would build %d service(s)", len(buildableServices))
			response.Steps = append(response.Steps, step)
		} else {
			scenarioDir := filepath.Join(o.vrooli, "scenarios", profile.Scenario)
			builder := build.NewBuilder(scenarioDir, o.log)

			allSucceeded := true
			var allResults []build.BuildResult

			if len(buildPlatforms) == 0 {
				o.failStep(&step, "no valid target platforms resolved")
				response.Steps = append(response.Steps, step)
				response.Status = "failed"
				o.writeJSON(w, http.StatusBadRequest, response)
				return
			}

			buildCtx, cancel := context.WithTimeout(ctx, effectiveTimeout)
			defer cancel()

			for _, svc := range buildableServices {
				result, err := builder.BuildAll(buildCtx, svc.ID, svc.Build, buildPlatforms)

				if err != nil {
					o.log("error", map[string]interface{}{
						"msg":     "build failed",
						"service": svc.ID,
						"error":   err.Error(),
					})
					allSucceeded = false
					continue
				}

				allResults = append(allResults, result.Results...)
				if !result.AllSucceeded {
					allSucceeded = false
				}
			}

			response.BuildResults = &build.BuildAllResult{
				Results:      allResults,
				AllSucceeded: allSucceeded,
			}

			if allSucceeded {
				// Update manifest binary paths with actual build outputs
				// Pass the manifest directory so paths are relative to where the manifest lives
				manifestDir := filepath.Dir(response.ManifestPath)
				updateManifestBinaryPaths(manifest, allResults, scenarioDir, manifestDir)

				// Re-write the manifest with updated binary paths
				if response.ManifestPath != "" {
					manifestData, _ := json.MarshalIndent(manifest, "", "  ")
					if err := os.WriteFile(response.ManifestPath, manifestData, 0o644); err != nil {
						o.log("warning", map[string]interface{}{
							"msg":   "failed to update manifest with build paths",
							"error": err.Error(),
						})
					}
				}

				o.successStep(&step, fmt.Sprintf("built %d service(s) for %d platform(s)", len(buildableServices), len(buildPlatforms)))
			} else {
				o.failStep(&step, "some builds failed")
			}
			response.Steps = append(response.Steps, step)
		}
	} else {
		step = o.startStep("Build binaries")
		step.Status = "skipped"
		step.Message = "build skipped by request"
		response.Steps = append(response.Steps, step)
	}

	// Step 6: Generate desktop wrapper via scenario-to-desktop (unless skipped)
	if !req.SkipPackaging {
		step = o.startStep("Generate desktop wrapper")

		if req.DryRun {
			step.Status = "skipped"
			step.Message = "dry run - would generate Electron wrapper via scenario-to-desktop"
			response.Steps = append(response.Steps, step)
		} else {
			desktopClient, err := NewDesktopPackagerClient(o.log)
			if err != nil {
				o.failStep(&step, fmt.Sprintf("failed to create desktop client: %v", err))
				response.Steps = append(response.Steps, step)
				// Continue without packaging - user can do it manually
				o.log("warn", map[string]interface{}{
					"msg":   "scenario-to-desktop not available, skipping packaging",
					"error": err.Error(),
				})
			} else {
				packCtx, cancel := context.WithTimeout(ctx, effectiveTimeout)
				defer cancel()

				genReq := &QuickGenerateRequest{
					ScenarioName:       profile.Scenario,
					TemplateType:       "universal",
					DeploymentMode:     deploymentMode,
					BundleManifestPath: response.ManifestPath,
					Platforms:          installerTargets,
				}

				genResp, err := desktopClient.QuickGenerate(packCtx, genReq)
				if err != nil {
					o.failStep(&step, fmt.Sprintf("desktop generation failed: %v", err))
					response.Steps = append(response.Steps, step)
				} else {
					// Wait for generation to complete
					buildStatus, err := desktopClient.WaitForBuild(packCtx, genResp.BuildID, 3*time.Second)
					if err != nil {
						o.failStep(&step, fmt.Sprintf("desktop generation timed out or failed: %v", err))
						response.Steps = append(response.Steps, step)
					} else {
						response.DesktopBuildID = genResp.BuildID
						response.DesktopPath = buildStatus.OutputPath
						o.successStep(&step, fmt.Sprintf("generated Electron wrapper at %s", buildStatus.OutputPath))
						response.Steps = append(response.Steps, step)
					}
				}
			}
		}
	} else {
		step = o.startStep("Generate desktop wrapper")
		step.Status = "skipped"
		step.Message = "packaging skipped by request"
		response.Steps = append(response.Steps, step)
	}

	// Step 6.5: Validate runtime supervisor presence for bundled mode
	if deploymentMode == "bundled" && response.DesktopPath != "" {
		step = o.startStep("Validate runtime supervisor")
		runtimePath := filepath.Join(response.DesktopPath, "bundle", "runtime")
		info, err := os.Stat(runtimePath)
		if err != nil || !info.IsDir() {
			o.failStep(&step, fmt.Sprintf("runtime supervisor missing at %s", runtimePath))
		} else {
			entries, _ := os.ReadDir(runtimePath)
			if len(entries) == 0 {
				o.failStep(&step, fmt.Sprintf("runtime supervisor directory empty at %s", runtimePath))
			} else {
				o.successStep(&step, "runtime supervisor present")
			}
		}
		response.Steps = append(response.Steps, step)
	}

	// Step 7: Build platform installers (unless skipped)
	if !req.DryRun && !req.SkipPackaging && response.DesktopPath != "" && response.ManifestPath != "" {
		step = o.startStep("Copy binaries into bundle")
		manifestDir := filepath.Dir(response.ManifestPath)
		bundleDir := filepath.Join(response.DesktopPath, "bundle")
		missing, err := copyBuiltBinariesToBundle(manifest, manifestDir, bundleDir, buildPlatforms)
		if err != nil {
			o.failStep(&step, fmt.Sprintf("failed to copy binaries into bundle: %v", err))
		} else if len(missing) > 0 {
			step.Status = "warning"
			step.Message = fmt.Sprintf("copied binaries with %d missing artifact(s): %s", len(missing), strings.Join(missing, ", "))
		} else {
			o.successStep(&step, "copied binaries into bundle/bin for target platforms")
		}
		response.Steps = append(response.Steps, step)
	}

	if !req.SkipInstallers && !req.SkipPackaging && response.DesktopPath != "" {
		step = o.startStep("Build platform installers")

		if req.DryRun {
			step.Status = "skipped"
			step.Message = "dry run - would build MSI/PKG/AppImage installers"
			response.Steps = append(response.Steps, step)
		} else {
			installCtx, cancel := context.WithTimeout(ctx, effectiveTimeout)
			defer cancel()

			installers, err := o.buildInstallersWithPnpm(installCtx, response.DesktopPath, installerTargets)
			if err != nil {
				o.failStep(&step, fmt.Sprintf("installer build failed: %v", err))
			} else {
				response.Installers = installers
				o.successStep(&step, fmt.Sprintf("built installers for %d platform(s)", len(installerTargets)))
			}
			response.Steps = append(response.Steps, step)
		}
	} else if !req.SkipInstallers {
		step = o.startStep("Build platform installers")
		step.Status = "skipped"
		step.Message = "skipped - no desktop wrapper generated"
		response.Steps = append(response.Steps, step)
	} else {
		step = o.startStep("Build platform installers")
		step.Status = "skipped"
		step.Message = "installer build skipped by request"
		response.Steps = append(response.Steps, step)
	}

	// Determine overall status
	allPassed := true
	for _, s := range response.Steps {
		if s.Status == "failed" {
			allPassed = false
			break
		}
	}

	if allPassed {
		response.Status = "success"
		if len(response.Installers) > 0 {
			response.NextSteps = []string{
				"Download installers from the Installers field in this response",
				fmt.Sprintf("Or find them at: %s/dist-electron/", response.DesktopPath),
			}
		} else if response.DesktopPath != "" {
			response.NextSteps = []string{
				fmt.Sprintf("cd %s", response.DesktopPath),
				"pnpm install",
				"pnpm run dist:all  # Build installers for all platforms",
			}
		} else {
			response.NextSteps = []string{
				fmt.Sprintf("cd %s", filepath.Join(o.vrooli, "scenarios", profile.Scenario, "platforms", "electron")),
				"pnpm install",
				"pnpm run dist:all  # Build installers for all platforms",
			}
		}
	} else {
		response.Status = "failed"
	}

	response.Duration = time.Since(start).String()
	o.writeJSON(w, http.StatusOK, response)
}

// populateAssetMetadata fills in missing SHA256 and size metadata for assets that exist on disk.
func populateAssetMetadata(manifest *bundles.Manifest, scenarioDir string) error {
	if manifest == nil {
		return fmt.Errorf("manifest is nil")
	}

	// Expand ui-bundle assets to include all built files so the runtime serves proper MIME types.
	_ = expandUIAssets(manifest, scenarioDir) // best-effort

	var firstErr error
	for si := range manifest.Services {
		svc := &manifest.Services[si]
		for ai := range svc.Assets {
			asset := &svc.Assets[ai]
			if asset == nil || asset.Path == "" {
				continue
			}

			// Skip if already populated with a non-placeholder hash
			if asset.SHA256 != "" && asset.SHA256 != "pending" {
				continue
			}

			assetPath := asset.Path
			if !filepath.IsAbs(assetPath) {
				assetPath = filepath.Join(scenarioDir, assetPath)
			}

			info, err := os.Stat(assetPath)
			if err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("stat asset %s: %w", assetPath, err)
				}
				continue
			}

			hash, err := hashFileSHA256(assetPath)
			if err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("hash asset %s: %w", assetPath, err)
				}
				continue
			}

			asset.SHA256 = hash
			asset.SizeBytes = info.Size()
		}
	}

	return firstErr
}

func hashFileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// expandUIAssets ensures ui-bundle services enumerate all built assets under ui/dist.
func expandUIAssets(manifest *bundles.Manifest, scenarioDir string) error {
	if manifest == nil {
		return nil
	}
	var firstErr error
	for si := range manifest.Services {
		svc := &manifest.Services[si]
		if !strings.EqualFold(svc.Type, "ui-bundle") {
			continue
		}

		// Determine ui dist root
		uiRoot := filepath.Join(scenarioDir, "ui", "dist")
		entries, err := os.ReadDir(uiRoot)
		if err != nil {
			if firstErr == nil {
				firstErr = fmt.Errorf("read ui dist: %w", err)
			}
			continue
		}

		var assets []bundles.Asset
		for _, entry := range entries {
			// Walk recursively
			err := filepath.WalkDir(filepath.Join(uiRoot, entry.Name()), func(path string, d os.DirEntry, werr error) error {
				if werr != nil {
					return werr
				}
				if d.IsDir() {
					return nil
				}
				rel, _ := filepath.Rel(scenarioDir, path)
				info, _ := os.Stat(path)
				hash, herr := hashFileSHA256(path)
				if herr != nil {
					if firstErr == nil {
						firstErr = herr
					}
					return nil
				}
				assets = append(assets, bundles.Asset{
					Path:      filepath.ToSlash(rel),
					SHA256:    hash,
					SizeBytes: info.Size(),
				})
				return nil
			})
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}

		// If we found assets, replace the asset list.
		if len(assets) > 0 {
			svc.Assets = assets
		}
	}
	return firstErr
}

func (o *Orchestrator) startStep(name string) OrchestrationStep {
	return OrchestrationStep{
		Name:   name,
		Status: "running",
	}
}

func (o *Orchestrator) successStep(step *OrchestrationStep, message string) {
	step.Status = "success"
	step.Message = message
}

func (o *Orchestrator) failStep(step *OrchestrationStep, errMsg string) {
	step.Status = "failed"
	step.Error = errMsg
}

// blockingDependencies lists dependencies that require swaps for desktop deployment.
// These dependencies cannot be bundled directly and need lightweight alternatives.
var blockingDependencies = map[string]string{
	"postgres":    "sqlite",
	"redis":       "in-process",
	"browserless": "playwright-driver",
	"n8n":         "embedded-workflows",
	"qdrant":      "faiss-local",
}

func (o *Orchestrator) validateProfile(ctx context.Context, profileID string) error {
	profile, err := o.profileRepo.Get(ctx, profileID)
	if err != nil {
		return fmt.Errorf("failed to load profile: %w", err)
	}
	if profile == nil {
		return fmt.Errorf("profile not found")
	}
	if profile.Scenario == "" {
		return fmt.Errorf("profile has no scenario configured")
	}

	// Get scenario dependencies from analyzer
	deps, err := shared.GetScenarioDependencies(ctx, profile.Scenario)
	if err != nil {
		// Log but don't fail if analyzer unavailable - allow manual override
		o.log("warn", map[string]interface{}{
			"msg":      "could not fetch scenario dependencies for blocker check",
			"scenario": profile.Scenario,
			"error":    err.Error(),
		})
		// Continue without blocker check if analyzer unavailable
		return nil
	}

	// Get swaps already applied to this profile
	appliedSwaps, _ := o.profileRepo.GetSwaps(ctx, profileID)
	swappedDeps := make(map[string]bool)
	for _, swap := range appliedSwaps {
		swappedDeps[swap.From] = true
	}

	// Check for blocking dependencies without swaps
	var blockers []string
	for _, dep := range deps {
		if suggestedSwap, isBlocking := blockingDependencies[dep]; isBlocking {
			if !swappedDeps[dep] {
				blockers = append(blockers, fmt.Sprintf("%s (swap to %s)", dep, suggestedSwap))
			}
		}
	}

	if len(blockers) > 0 {
		return fmt.Errorf("unresolved blockers for desktop deployment: %s. Run 'deployment-manager swaps list %s' to see available swaps, then apply with 'deployment-manager swaps apply <profile-id> <from> <to>'",
			strings.Join(blockers, ", "), profile.Scenario)
	}

	return nil
}

func (o *Orchestrator) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func resolveBuildPlatforms(inputs []string) []string {
	defaultPlatforms := []string{"linux-x64", "linux-arm64", "darwin-x64", "darwin-arm64", "win-x64"}
	if len(inputs) == 0 {
		return defaultPlatforms
	}

	allowed := make(map[string]bool)
	for _, p := range build.SupportedPlatforms {
		allowed[p.Name] = true
	}

	var result []string
	add := func(name string) {
		if !allowed[name] {
			return
		}
		for _, existing := range result {
			if existing == name {
				return
			}
		}
		result = append(result, name)
	}

	for _, raw := range inputs {
		switch strings.ToLower(raw) {
		case "win", "windows", "win-x64":
			add("win-x64")
		case "mac":
			add("darwin-x64")
			add("darwin-arm64")
		case "darwin-x64":
			add("darwin-x64")
		case "darwin-arm64":
			add("darwin-arm64")
		case "linux":
			add("linux-x64")
			add("linux-arm64")
		case "linux-x64":
			add("linux-x64")
		case "linux-arm64":
			add("linux-arm64")
		default:
			add(strings.ToLower(raw))
		}
	}

	if len(result) == 0 {
		return defaultPlatforms
	}
	return result
}

func resolveInstallerTargets(inputs []string) []string {
	if len(inputs) == 0 {
		return []string{"win", "mac", "linux"}
	}
	targetSet := map[string]bool{}
	add := func(name string) {
		targetSet[name] = true
	}
	for _, raw := range inputs {
		switch strings.ToLower(raw) {
		case "win", "windows", "win-x64":
			add("win")
		case "mac", "darwin", "darwin-arm64", "darwin-x64":
			add("mac")
		case "linux", "linux-x64", "linux-arm64":
			add("linux")
		}
	}
	if len(targetSet) == 0 {
		return []string{"win", "mac", "linux"}
	}
	var out []string
	for _, name := range []string{"win", "mac", "linux"} {
		if targetSet[name] {
			out = append(out, name)
		}
	}
	return out
}

func copyBuiltBinariesToBundle(manifest *bundles.Manifest, manifestDir, bundleDir string, platforms []string) ([]string, error) {
	if manifest == nil {
		return nil, fmt.Errorf("manifest is nil")
	}
	platformSet := make(map[string]bool)
	for _, p := range platforms {
		platformSet[p] = true
	}

	if err := os.MkdirAll(filepath.Join(bundleDir, "bin"), 0o755); err != nil {
		return nil, err
	}

	var missing []string

	for _, svc := range manifest.Services {
		for platform, bin := range svc.Binaries {
			if len(platformSet) > 0 && !platformSet[platform] {
				continue
			}

			src := bin.Path
			if !filepath.IsAbs(src) {
				src = filepath.Join(manifestDir, filepath.FromSlash(bin.Path))
			}

			if _, err := os.Stat(src); err != nil {
				missing = append(missing, fmt.Sprintf("%s:%s", svc.ID, platform))
				continue
			}

			destDir := filepath.Join(bundleDir, "bin", platform)
			if err := os.MkdirAll(destDir, 0o755); err != nil {
				return missing, err
			}

			dest := filepath.Join(destDir, filepath.Base(src))
			if err := copyFilePreserveMode(src, dest); err != nil {
				return missing, err
			}
		}
	}

	return missing, nil
}

func copyFilePreserveMode(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	mode := os.FileMode(0o755)
	if info, err := os.Stat(src); err == nil {
		mode = info.Mode().Perm()
	}
	return os.WriteFile(dst, data, mode)
}

func (o *Orchestrator) buildInstallersWithPnpm(ctx context.Context, desktopPath string, platforms []string) (map[string]string, error) {
	if len(platforms) == 0 {
		platforms = []string{"win", "mac", "linux"}
	}

	packageManager := "pnpm"
	if _, err := exec.LookPath(packageManager); err != nil {
		o.log("warn", map[string]interface{}{
			"msg":   "pnpm not found, falling back to npm",
			"error": err.Error(),
		})
		packageManager = "npm"
	}

	if err := runCommandLogged(ctx, packageManager, []string{"install"}, desktopPath, o.log); err != nil {
		return nil, err
	}

	distDir := filepath.Join(desktopPath, "dist-electron")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		return nil, err
	}

	installers := make(map[string]string)
	for _, platform := range platforms {
		cmd := []string{"run", fmt.Sprintf("dist:%s", platform)}
		if err := runCommandLogged(ctx, packageManager, cmd, desktopPath, o.log); err != nil {
			return installers, fmt.Errorf("%s build failed: %w", platform, err)
		}

		artifact, err := findInstallerArtifact(distDir, platform)
		if err != nil {
			o.log("warn", map[string]interface{}{
				"msg":      "installer built but artifact not located",
				"platform": platform,
				"error":    err.Error(),
			})
			continue
		}
		installers[platform] = artifact
	}

	return installers, nil
}

func runCommandLogged(ctx context.Context, bin string, args []string, dir string, log func(string, map[string]interface{})) error {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	log("info", map[string]interface{}{
		"msg":  "command completed",
		"cmd":  fmt.Sprintf("%s %s", bin, strings.Join(args, " ")),
		"dir":  dir,
		"ok":   err == nil,
		"logs": string(output),
	})
	if err != nil {
		return fmt.Errorf("%s %s failed: %v", bin, strings.Join(args, " "), err)
	}
	return nil
}

func findInstallerArtifact(distDir, platform string) (string, error) {
	var patterns []string
	switch platform {
	case "win":
		patterns = []string{"*.exe", "*.msi"}
	case "mac":
		patterns = []string{"*.dmg", "*.pkg", "*.zip"}
	case "linux":
		patterns = []string{"*.AppImage", "*.deb", "*.tar.gz"}
	default:
		return "", fmt.Errorf("unknown platform %s", platform)
	}

	var candidates []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(filepath.Join(distDir, pattern))
		if err == nil {
			candidates = append(candidates, matches...)
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("no installer artifacts matched for %s", platform)
	}

	sort.Slice(candidates, func(i, j int) bool {
		infoI, _ := os.Stat(candidates[i])
		infoJ, _ := os.Stat(candidates[j])
		if infoI == nil || infoJ == nil {
			return candidates[i] < candidates[j]
		}
		return infoI.ModTime().After(infoJ.ModTime())
	})

	return candidates[0], nil
}

// pruneNonCrossPlatformCLIs removes CLI services that are not cross-platform (e.g., no Go/Rust sources to compile).
// It returns the IDs of services pruned and leaves non-CLI services untouched.
func pruneNonCrossPlatformCLIs(manifest *bundles.Manifest, scenarioDir string) ([]string, error) {
	if manifest == nil {
		return nil, nil
	}

	var kept []bundles.ServiceEntry
	var pruned []string

	for _, svc := range manifest.Services {
		if !isCLIService(svc) {
			kept = append(kept, svc)
			continue
		}

		if isCrossPlatformCLIBuild(svc, scenarioDir) {
			kept = append(kept, svc)
			continue
		}

		pruned = append(pruned, svc.ID)
	}

	manifest.Services = kept
	return pruned, nil
}

func isCLIService(svc bundles.ServiceEntry) bool {
	id := strings.ToLower(svc.ID)
	if strings.Contains(id, "cli") {
		return true
	}
	if svc.Build != nil {
		base := strings.ToLower(filepath.Base(svc.Build.SourceDir))
		if base == "cli" {
			return true
		}
	}
	return false
}

func isCrossPlatformCLIBuild(svc bundles.ServiceEntry, scenarioDir string) bool {
	if svc.Build == nil {
		return false
	}

	switch svc.Build.Type {
	case "go":
		sourceDir := filepath.Join(scenarioDir, svc.Build.SourceDir)
		matches, _ := filepath.Glob(filepath.Join(sourceDir, "*.go"))
		return len(matches) > 0
	case "rust":
		sourceDir := filepath.Join(scenarioDir, svc.Build.SourceDir)
		if _, err := os.Stat(filepath.Join(sourceDir, "Cargo.toml")); err == nil {
			return true
		}
		matches, _ := filepath.Glob(filepath.Join(sourceDir, "*.rs"))
		return len(matches) > 0
	default:
		return false
	}
}

// updateManifestBinaryPaths updates manifest service binaries to point to actual build outputs.
// This ensures scenario-to-desktop can find the compiled binaries.
// The manifestDir parameter should be the directory containing the manifest file.
func updateManifestBinaryPaths(manifest *bundles.Manifest, results []build.BuildResult, scenarioDir, manifestDir string) {
	// Build a map of platform -> output path from build results
	platformPaths := make(map[string]string)
	for _, r := range results {
		if r.Success && r.OutputPath != "" {
			// Make path relative to manifest directory (not scenario directory)
			// This is necessary because scenario-to-desktop resolves paths relative to the manifest location
			relPath, err := filepath.Rel(manifestDir, r.OutputPath)
			if err != nil {
				// Fall back to path relative to scenario directory
				relPath = r.OutputPath
				if strings.HasPrefix(r.OutputPath, scenarioDir) {
					relPath = strings.TrimPrefix(r.OutputPath, scenarioDir)
					relPath = strings.TrimPrefix(relPath, string(filepath.Separator))
				}
			}
			// Normalize to forward slashes for JSON
			relPath = filepath.ToSlash(relPath)
			platformPaths[r.Platform] = relPath
		}
	}

	// Update manifest services with actual binary paths
	for i := range manifest.Services {
		svc := &manifest.Services[i]
		if svc.Build == nil {
			continue // Skip services without build config
		}

		// Update binary paths for each platform
		for platform := range svc.Binaries {
			if newPath, ok := platformPaths[platform]; ok {
				svc.Binaries[platform] = bundles.ServiceBinary{
					Path: newPath,
					Args: svc.Binaries[platform].Args,
					Env:  svc.Binaries[platform].Env,
					Cwd:  svc.Binaries[platform].Cwd,
				}
			}
		}

		// Also update darwin-arm64 and linux-arm64 if we built them
		if newPath, ok := platformPaths["darwin-arm64"]; ok {
			svc.Binaries["darwin-arm64"] = bundles.ServiceBinary{Path: newPath}
		}
		if newPath, ok := platformPaths["linux-arm64"]; ok {
			svc.Binaries["linux-arm64"] = bundles.ServiceBinary{Path: newPath}
		}

		// Clear build config since binaries are now pre-compiled
		// This prevents scenario-to-desktop from trying to recompile
		svc.Build = nil
	}
}

// applySigningConfig applies the provided signing configuration to scenario-to-desktop.
// This allows deploy-desktop to configure signing in a single command.
func (o *Orchestrator) applySigningConfig(ctx context.Context, scenarioName string, config map[string]interface{}) error {
	desktopClient, err := NewDesktopPackagerClient(o.log)
	if err != nil {
		return fmt.Errorf("scenario-to-desktop unavailable: %w", err)
	}

	return desktopClient.SetSigningConfig(ctx, scenarioName, config)
}

// checkSigningReadiness checks if signing is configured for the scenario.
// Returns a list of warnings (empty if signing is ready or not applicable).
// This is non-blocking - it only returns warnings, never errors that stop the build.
func (o *Orchestrator) checkSigningReadiness(ctx context.Context, scenarioName string) []string {
	var warnings []string

	// Try to create a desktop client to check signing
	desktopClient, err := NewDesktopPackagerClient(o.log)
	if err != nil {
		// scenario-to-desktop not available - can't check signing, but don't warn
		// The actual packaging step will handle this
		o.log("debug", map[string]interface{}{
			"msg":   "could not check signing readiness - scenario-to-desktop unavailable",
			"error": err.Error(),
		})
		return nil
	}

	// Check signing readiness
	readiness, err := desktopClient.CheckSigningReadiness(ctx, scenarioName)
	if err != nil {
		// API error - log but don't warn user (might be network issue)
		o.log("debug", map[string]interface{}{
			"msg":      "signing readiness check failed",
			"scenario": scenarioName,
			"error":    err.Error(),
		})
		return nil
	}

	// If signing is not ready, collect warnings
	if !readiness.Ready {
		if len(readiness.Issues) > 0 {
			warnings = append(warnings, readiness.Issues...)
		} else {
			warnings = append(warnings, "Code signing not configured - installers will be unsigned")
		}

		// Add platform-specific warnings
		for platform, status := range readiness.Platforms {
			if !status.Ready && status.Reason != "" {
				warnings = append(warnings, fmt.Sprintf("%s: %s", platform, status.Reason))
			}
		}
	}

	return warnings
}
