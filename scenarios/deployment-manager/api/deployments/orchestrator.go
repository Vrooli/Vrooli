// Package deployments provides deployment orchestration for bundled desktop apps.
package deployments

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
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
}

// DeployDesktopResponse is the response from orchestrated deployment.
type DeployDesktopResponse struct {
	Status          string                   `json:"status"`
	ProfileID       string                   `json:"profile_id"`
	Scenario        string                   `json:"scenario"`
	Steps           []OrchestrationStep      `json:"steps"`
	ManifestPath    string                   `json:"manifest_path,omitempty"`
	BuildResults    *build.BuildAllResult    `json:"build_results,omitempty"`
	DesktopBuildID  string                   `json:"desktop_build_id,omitempty"`
	DesktopPath     string                   `json:"desktop_path,omitempty"`
	InstallerBuildID string                  `json:"installer_build_id,omitempty"`
	Installers      map[string]string        `json:"installers,omitempty"`
	Duration        string                   `json:"duration,omitempty"`
	NextSteps       []string                 `json:"next_steps,omitempty"`
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
	o.successStep(&step, fmt.Sprintf("assembled manifest with %d swaps", len(manifest.Swaps)))
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

			for _, svc := range buildableServices {
				result, err := builder.BuildAll(ctx, svc.ID, svc.Build, req.Platforms)

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

				o.successStep(&step, fmt.Sprintf("built %d service(s) for %d platform(s)", len(buildableServices), len(allResults)/len(buildableServices)))
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
				deploymentMode := req.DeploymentMode
				if deploymentMode == "" {
					deploymentMode = "bundled"
				}

				genReq := &QuickGenerateRequest{
					ScenarioName:       profile.Scenario,
					TemplateType:       "universal",
					DeploymentMode:     deploymentMode,
					BundleManifestPath: response.ManifestPath,
				}

				genResp, err := desktopClient.QuickGenerate(ctx, genReq)
				if err != nil {
					o.failStep(&step, fmt.Sprintf("desktop generation failed: %v", err))
					response.Steps = append(response.Steps, step)
				} else {
					// Wait for generation to complete
					buildStatus, err := desktopClient.WaitForBuild(ctx, genResp.BuildID, 3*time.Second)
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

	// Step 7: Build platform installers (unless skipped)
	if !req.SkipInstallers && !req.SkipPackaging && response.DesktopPath != "" {
		step = o.startStep("Build platform installers")

		if req.DryRun {
			step.Status = "skipped"
			step.Message = "dry run - would build MSI/PKG/AppImage installers"
			response.Steps = append(response.Steps, step)
		} else {
			desktopClient, err := NewDesktopPackagerClient(o.log)
			if err != nil {
				o.failStep(&step, fmt.Sprintf("failed to create desktop client: %v", err))
				response.Steps = append(response.Steps, step)
			} else {
				platforms := req.Platforms
				if len(platforms) == 0 {
					platforms = []string{"win", "mac", "linux"}
				}

				buildReq := &ScenarioBuildRequest{
					Platforms: platforms,
					Clean:     false,
				}

				buildResp, err := desktopClient.BuildScenario(ctx, profile.Scenario, buildReq)
				if err != nil {
					o.failStep(&step, fmt.Sprintf("installer build failed to start: %v", err))
					response.Steps = append(response.Steps, step)
				} else {
					// Wait for build to complete
					buildStatus, err := desktopClient.WaitForBuild(ctx, buildResp.BuildID, 10*time.Second)
					if err != nil {
						o.failStep(&step, fmt.Sprintf("installer build failed: %v", err))
						response.Steps = append(response.Steps, step)
					} else {
						response.InstallerBuildID = buildResp.BuildID
						response.Installers = buildStatus.Artifacts
						o.successStep(&step, fmt.Sprintf("built installers for %d platform(s)", len(platforms)))
						response.Steps = append(response.Steps, step)
					}
				}
			}
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
