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
	repocontract "github.com/vrooli/repo-contract-go"
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
	// VisualValidation enables screen-recorded smoke test validation before publishing
	VisualValidation bool `json:"visual_validation,omitempty"`
	// GitCommitHash ties this deployment to a specific source commit.
	// When provided, the release gate is checked: all required platforms must be
	// approved for this exact commit before deployment proceeds.
	GitCommitHash string `json:"git_commit_hash,omitempty"`
}

// DeployDesktopResponse is the response from orchestrated deployment.
type DeployDesktopResponse struct {
	Status            string                `json:"status"`
	ProfileID         string                `json:"profile_id"`
	Scenario          string                `json:"scenario"`
	Steps             []OrchestrationStep   `json:"steps"`
	ManifestPath      string                `json:"manifest_path,omitempty"`
	BuildResults      *build.BuildAllResult `json:"build_results,omitempty"`
	DesktopBuildID    string                `json:"desktop_build_id,omitempty"`
	DesktopPath       string                `json:"desktop_path,omitempty"`
	InstallerBuildID  string                `json:"installer_build_id,omitempty"`
	Installers        map[string]string     `json:"installers,omitempty"`
	PublishedVersions []PublishedVersion    `json:"published_versions,omitempty"`
	Duration          string                `json:"duration,omitempty"`
	NextSteps         []string              `json:"next_steps,omitempty"`
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
	profileRepo           profiles.Repository
	approvalsRepo         ApprovalsRepository
	publishedVersionsRepo PublishedVersionsRepository
	vrooli                string
	log                   func(string, map[string]interface{})
}

// NewOrchestrator creates a new deployment orchestrator.
func NewOrchestrator(profileRepo profiles.Repository, log func(string, map[string]interface{})) *Orchestrator {
	return NewOrchestratorWithApprovals(profileRepo, nil, log)
}

// NewOrchestratorWithApprovals creates a new deployment orchestrator with approval gating.
func NewOrchestratorWithApprovals(profileRepo profiles.Repository, approvalsRepo ApprovalsRepository, log func(string, map[string]interface{})) *Orchestrator {
	return NewOrchestratorFull(profileRepo, approvalsRepo, nil, log)
}

// NewOrchestratorFull creates a new deployment orchestrator with all optional repositories.
func NewOrchestratorFull(profileRepo profiles.Repository, approvalsRepo ApprovalsRepository, publishedVersionsRepo PublishedVersionsRepository, log func(string, map[string]interface{})) *Orchestrator {
	vrooli := resolveRepoRoot()
	return &Orchestrator{
		profileRepo:           profileRepo,
		approvalsRepo:         approvalsRepo,
		publishedVersionsRepo: publishedVersionsRepo,
		vrooli:                vrooli,
		log:                   log,
	}
}

// deployState holds mutable state threaded through the deployment phases.
type deployState struct {
	req              DeployDesktopRequest
	response         *DeployDesktopResponse
	profile          *profiles.Profile
	manifest         *bundles.Manifest
	ctx              context.Context
	scenarioBaseDir  string
	outputDir        string
	deploymentMode   string
	buildPlatforms   []string
	installerTargets []string
	effectiveTimeout time.Duration
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

	ds := &deployState{
		req: req,
		response: &DeployDesktopResponse{
			ProfileID: req.ProfileID,
			Steps:     make([]OrchestrationStep, 0),
		},
		ctx:             ctx,
		scenarioBaseDir: resolveTopLevelScenarioDir(o.vrooli),
	}

	ds.effectiveTimeout = time.Duration(req.TimeoutSeconds) * time.Second
	if ds.effectiveTimeout <= 0 {
		ds.effectiveTimeout = 10 * time.Minute
	}
	ds.deploymentMode = req.DeploymentMode
	if ds.deploymentMode == "" {
		ds.deploymentMode = "bundled"
	}
	ds.buildPlatforms = resolveBuildPlatforms(req.Platforms)
	ds.installerTargets = resolveInstallerTargets(req.Platforms)

	if code := o.deployLoadProfile(ds); code != 0 {
		o.writeJSON(w, code, ds.response)
		return
	}

	if code := o.deployValidateAndSign(ds); code != 0 {
		o.writeJSON(w, code, ds.response)
		return
	}

	if code := o.deployAssembleManifest(ds); code != 0 {
		o.writeJSON(w, code, ds.response)
		return
	}

	if code := o.deployBuildBinaries(ds); code != 0 {
		o.writeJSON(w, code, ds.response)
		return
	}

	if code := o.deployPackageAndInstall(ds); code != 0 {
		o.writeJSON(w, code, ds.response)
		return
	}

	o.deployFinalizeAndPublish(ds)

	ds.response.Duration = time.Since(start).String()
	o.writeJSON(w, http.StatusOK, ds.response)
}

// deployLoadProfile loads the profile and checks the release gate.
func (o *Orchestrator) deployLoadProfile(ds *deployState) int {
	step := o.startStep("Load profile")
	profile, err := o.profileRepo.Get(ds.ctx, ds.req.ProfileID)
	if err != nil {
		o.failStep(&step, fmt.Sprintf("failed to load profile: %v", err))
		ds.response.Steps = append(ds.response.Steps, step)
		ds.response.Status = "failed"
		return http.StatusBadGateway
	}
	if profile == nil {
		o.failStep(&step, "profile not found")
		ds.response.Steps = append(ds.response.Steps, step)
		ds.response.Status = "failed"
		return http.StatusNotFound
	}
	ds.profile = profile
	ds.response.Scenario = profile.Scenario
	o.successStep(&step, fmt.Sprintf("loaded profile for scenario %s", profile.Scenario))
	ds.response.Steps = append(ds.response.Steps, step)

	// Release gate check
	if ds.req.GitCommitHash != "" && o.approvalsRepo != nil {
		step = o.startStep("Check release gate")
		gate, gateErr := o.approvalsRepo.CheckReleaseGate(ds.ctx, ds.req.ProfileID, ds.req.GitCommitHash)
		if gateErr != nil {
			o.failStep(&step, fmt.Sprintf("release gate check failed: %v", gateErr))
			ds.response.Steps = append(ds.response.Steps, step)
			ds.response.Status = "failed"
			return http.StatusInternalServerError
		}
		if !gate.Ready {
			msg := fmt.Sprintf("release gate blocked for commit %s:", ds.req.GitCommitHash)
			for _, p := range gate.Platforms {
				if p.Status != ApprovalStatusApproved {
					msg += fmt.Sprintf(" %s=%s", p.Platform, p.Status)
				}
			}
			o.failStep(&step, msg)
			ds.response.Steps = append(ds.response.Steps, step)
			ds.response.Status = "blocked"
			return http.StatusPreconditionFailed
		}
		o.successStep(&step, fmt.Sprintf("all required platforms approved for commit %s", ds.req.GitCommitHash))
		ds.response.Steps = append(ds.response.Steps, step)
	}

	return 0
}

// deployValidateAndSign validates the profile and applies signing config.
func (o *Orchestrator) deployValidateAndSign(ds *deployState) int {
	if !ds.req.SkipValidation {
		step := o.startStep("Validate profile")
		if err := o.validateProfile(ds.ctx, ds.req.ProfileID); err != nil {
			o.failStep(&step, err.Error())
			ds.response.Steps = append(ds.response.Steps, step)
			ds.response.Status = "failed"
			return http.StatusBadRequest
		}
		o.successStep(&step, "profile validation passed")
		ds.response.Steps = append(ds.response.Steps, step)
	} else {
		step := o.startStep("Validate profile")
		step.Status = "skipped"
		step.Message = "validation skipped by request"
		ds.response.Steps = append(ds.response.Steps, step)
	}

	// Apply signing config if provided
	if len(ds.req.SigningConfig) > 0 {
		step := o.startStep("Apply signing configuration")
		if ds.req.DryRun {
			step.Status = "skipped"
			step.Message = "dry run - would apply signing config"
		} else {
			if err := o.applySigningConfig(ds.ctx, ds.profile.Scenario, ds.req.SigningConfig); err != nil {
				step.Status = "warning"
				step.Message = fmt.Sprintf("failed to apply signing config: %v", err)
				o.log("warn", map[string]interface{}{
					"msg":      "signing config application failed",
					"scenario": ds.profile.Scenario,
					"error":    err.Error(),
				})
			} else {
				o.successStep(&step, "signing configuration applied to scenario-to-desktop")
			}
		}
		ds.response.Steps = append(ds.response.Steps, step)
	}

	// Check signing readiness
	step := o.startStep("Check signing readiness")
	signingWarnings := o.checkSigningReadiness(ds.ctx, ds.profile.Scenario)
	if len(signingWarnings) > 0 {
		step.Status = "warning"
		step.Message = strings.Join(signingWarnings, "; ")
		o.log("warn", map[string]interface{}{
			"msg":      "signing not fully configured",
			"scenario": ds.profile.Scenario,
			"issues":   signingWarnings,
		})
	} else {
		o.successStep(&step, "signing configuration ready")
	}
	ds.response.Steps = append(ds.response.Steps, step)

	return 0
}

// deployAssembleManifest fetches the bundle skeleton, applies swaps, and writes the manifest.
func (o *Orchestrator) deployAssembleManifest(ds *deployState) int {
	step := o.startStep("Assemble manifest")
	manifest, err := bundles.FetchSkeletonBundle(ds.ctx, ds.profile.Scenario)
	if err != nil {
		o.failStep(&step, fmt.Sprintf("failed to fetch bundle skeleton: %v", err))
		ds.response.Steps = append(ds.response.Steps, step)
		ds.response.Status = "failed"
		return http.StatusBadGateway
	}

	// Apply swaps from profile
	profileSwaps, _ := o.profileRepo.GetSwaps(ds.ctx, ds.req.ProfileID)
	for _, ps := range profileSwaps {
		manifest.Swaps = append(manifest.Swaps, bundles.ManifestSwap{
			Original:    ps.From,
			Replacement: ps.To,
			Reason:      ps.Reason,
			Limitations: ps.Limitations,
		})
	}

	// Populate missing asset metadata
	scenarioDir := filepath.Join(ds.scenarioBaseDir, ds.profile.Scenario)
	if err := populateAssetMetadata(manifest, scenarioDir); err != nil {
		step.Status = "warning"
		step.Message = fmt.Sprintf("assembled manifest with %d swaps (asset metadata partial: %v)", len(manifest.Swaps), err)
		o.log("warn", map[string]interface{}{
			"msg":      "asset metadata population incomplete",
			"scenario": ds.profile.Scenario,
			"error":    err.Error(),
		})
	}
	o.successStep(&step, fmt.Sprintf("assembled manifest with %d swaps", len(manifest.Swaps)))
	ds.response.Steps = append(ds.response.Steps, step)

	// Normalize CLI services
	step = o.startStep("Normalize CLI services")
	pruned, err := pruneNonCrossPlatformCLIs(manifest, filepath.Join(ds.scenarioBaseDir, ds.profile.Scenario))
	if err != nil {
		step.Status = "warning"
		step.Message = fmt.Sprintf("failed to normalize CLI services: %v", err)
		o.log("warn", map[string]interface{}{
			"msg":      "normalize cli services failed",
			"scenario": ds.profile.Scenario,
			"error":    err.Error(),
		})
	} else if len(pruned) > 0 {
		step.Status = "warning"
		step.Message = fmt.Sprintf("omitted %d CLI service(s) not cross-platform: %s", len(pruned), strings.Join(pruned, ", "))
		o.log("warn", map[string]interface{}{
			"msg":       "omitted non-cross-platform cli services",
			"scenario":  ds.profile.Scenario,
			"services":  strings.Join(pruned, ","),
			"remediate": "make cli cross-platform (see test-genie) to include in bundle",
		})
	} else {
		o.successStep(&step, "CLI services are cross-platform or none present")
	}
	ds.response.Steps = append(ds.response.Steps, step)

	ds.manifest = manifest
	ds.outputDir = ds.req.OutputDir
	if ds.outputDir == "" {
		ds.outputDir = resolveScenarioDir(o.vrooli, ds.profile.Scenario)
	}

	// Write manifest
	step = o.startStep("Export manifest")
	if !ds.req.DryRun {
		if err := os.MkdirAll(ds.outputDir, 0o755); err != nil {
			o.failStep(&step, fmt.Sprintf("failed to create output dir: %v", err))
			ds.response.Steps = append(ds.response.Steps, step)
			ds.response.Status = "failed"
			return http.StatusInternalServerError
		}

		manifestPath := filepath.Join(ds.outputDir, "bundle.json")
		manifestData, _ := json.MarshalIndent(manifest, "", "  ")
		if err := os.WriteFile(manifestPath, manifestData, 0o644); err != nil {
			o.failStep(&step, fmt.Sprintf("failed to write manifest: %v", err))
			ds.response.Steps = append(ds.response.Steps, step)
			ds.response.Status = "failed"
			return http.StatusInternalServerError
		}
		ds.response.ManifestPath = manifestPath
		o.successStep(&step, fmt.Sprintf("wrote manifest to %s", manifestPath))
	} else {
		step.Status = "skipped"
		step.Message = "dry run - would write manifest"
	}
	ds.response.Steps = append(ds.response.Steps, step)

	return 0
}

// deployBuildBinaries compiles service binaries for target platforms.
func (o *Orchestrator) deployBuildBinaries(ds *deployState) int {
	if ds.req.SkipBuild {
		step := o.startStep("Build binaries")
		step.Status = "skipped"
		step.Message = "build skipped by request"
		ds.response.Steps = append(ds.response.Steps, step)
		return 0
	}

	step := o.startStep("Build binaries")

	var buildableServices []bundles.ServiceEntry
	for _, svc := range ds.manifest.Services {
		if svc.Build != nil {
			buildableServices = append(buildableServices, svc)
		}
	}

	if len(buildableServices) == 0 {
		step.Status = "skipped"
		step.Message = "no services with build configuration found"
		ds.response.Steps = append(ds.response.Steps, step)
		return 0
	}
	if ds.req.DryRun {
		step.Status = "skipped"
		step.Message = fmt.Sprintf("dry run - would build %d service(s)", len(buildableServices))
		ds.response.Steps = append(ds.response.Steps, step)
		return 0
	}

	scenarioDir := resolveScenarioDir(o.vrooli, ds.profile.Scenario)
	builder := build.NewBuilder(scenarioDir, o.log)

	if len(ds.buildPlatforms) == 0 {
		o.failStep(&step, "no valid target platforms resolved")
		ds.response.Steps = append(ds.response.Steps, step)
		ds.response.Status = "failed"
		return http.StatusBadRequest
	}

	buildCtx, cancel := context.WithTimeout(ds.ctx, ds.effectiveTimeout)
	defer cancel()

	allSucceeded := true
	var allResults []build.BuildResult
	for _, svc := range buildableServices {
		result, err := builder.BuildAll(buildCtx, svc.ID, svc.Build, ds.buildPlatforms)
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

	ds.response.BuildResults = &build.BuildAllResult{
		Results:      allResults,
		AllSucceeded: allSucceeded,
	}

	if allSucceeded {
		manifestDir := filepath.Dir(ds.response.ManifestPath)
		updateManifestBinaryPaths(ds.manifest, allResults, scenarioDir, manifestDir)

		if ds.response.ManifestPath != "" {
			manifestData, _ := json.MarshalIndent(ds.manifest, "", "  ")
			if err := os.WriteFile(ds.response.ManifestPath, manifestData, 0o644); err != nil {
				o.log("warning", map[string]interface{}{
					"msg":   "failed to update manifest with build paths",
					"error": err.Error(),
				})
			}
		}
		o.successStep(&step, fmt.Sprintf("built %d service(s) for %d platform(s)", len(buildableServices), len(ds.buildPlatforms)))
	} else {
		o.failStep(&step, "some builds failed")
	}
	ds.response.Steps = append(ds.response.Steps, step)

	return 0
}

// deployPackageAndInstall generates the desktop wrapper and builds installers.
func (o *Orchestrator) deployPackageAndInstall(ds *deployState) int {
	o.deployGenerateWrapper(ds)
	o.deployValidateRuntime(ds)
	o.deployCopyBinaries(ds)
	o.deployBuildInstallers(ds)
	o.deployVisualValidation(ds)
	return 0
}

func (o *Orchestrator) deployGenerateWrapper(ds *deployState) {
	if ds.req.SkipPackaging {
		step := o.startStep("Generate desktop wrapper")
		step.Status = "skipped"
		step.Message = "packaging skipped by request"
		ds.response.Steps = append(ds.response.Steps, step)
		return
	}

	step := o.startStep("Generate desktop wrapper")
	if ds.req.DryRun {
		step.Status = "skipped"
		step.Message = "dry run - would generate Electron wrapper via scenario-to-desktop"
		ds.response.Steps = append(ds.response.Steps, step)
		return
	}

	desktopClient, err := NewDesktopPackagerClient(o.log)
	if err != nil {
		o.failStep(&step, fmt.Sprintf("failed to create desktop client: %v", err))
		ds.response.Steps = append(ds.response.Steps, step)
		o.log("warn", map[string]interface{}{
			"msg":   "scenario-to-desktop not available, skipping packaging",
			"error": err.Error(),
		})
		return
	}

	packCtx, cancel := context.WithTimeout(ds.ctx, ds.effectiveTimeout)
	defer cancel()

	genReq := &QuickGenerateRequest{
		ScenarioName:       ds.profile.Scenario,
		TemplateType:       "universal",
		DeploymentMode:     ds.deploymentMode,
		BundleManifestPath: ds.response.ManifestPath,
		Platforms:          ds.installerTargets,
	}

	genResp, err := desktopClient.QuickGenerate(packCtx, genReq)
	if err != nil {
		o.failStep(&step, fmt.Sprintf("desktop generation failed: %v", err))
		ds.response.Steps = append(ds.response.Steps, step)
		return
	}

	buildStatus, err := desktopClient.WaitForBuild(packCtx, genResp.BuildID, 3*time.Second)
	if err != nil {
		o.failStep(&step, fmt.Sprintf("desktop generation timed out or failed: %v", err))
		ds.response.Steps = append(ds.response.Steps, step)
		return
	}

	ds.response.DesktopBuildID = genResp.BuildID
	ds.response.DesktopPath = buildStatus.OutputPath
	o.successStep(&step, fmt.Sprintf("generated Electron wrapper at %s", buildStatus.OutputPath))
	ds.response.Steps = append(ds.response.Steps, step)
}

func (o *Orchestrator) deployValidateRuntime(ds *deployState) {
	if ds.deploymentMode != "bundled" || ds.response.DesktopPath == "" {
		return
	}

	step := o.startStep("Validate runtime supervisor")
	runtimePath := filepath.Join(ds.response.DesktopPath, "bundle", "runtime")
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
	ds.response.Steps = append(ds.response.Steps, step)
}

func (o *Orchestrator) deployCopyBinaries(ds *deployState) {
	if ds.req.DryRun || ds.req.SkipPackaging || ds.response.DesktopPath == "" || ds.response.ManifestPath == "" {
		return
	}

	step := o.startStep("Copy binaries into bundle")
	manifestDir := filepath.Dir(ds.response.ManifestPath)
	bundleDir := filepath.Join(ds.response.DesktopPath, "bundle")
	missing, err := copyBuiltBinariesToBundle(ds.manifest, manifestDir, bundleDir, ds.buildPlatforms)
	if err != nil {
		o.failStep(&step, fmt.Sprintf("failed to copy binaries into bundle: %v", err))
	} else if len(missing) > 0 {
		step.Status = "warning"
		step.Message = fmt.Sprintf("copied binaries with %d missing artifact(s): %s", len(missing), strings.Join(missing, ", "))
	} else {
		o.successStep(&step, "copied binaries into bundle/bin for target platforms")
	}
	ds.response.Steps = append(ds.response.Steps, step)
}

func (o *Orchestrator) deployBuildInstallers(ds *deployState) {
	if ds.req.SkipInstallers {
		step := o.startStep("Build platform installers")
		step.Status = "skipped"
		step.Message = "installer build skipped by request"
		ds.response.Steps = append(ds.response.Steps, step)
		return
	}

	if ds.req.SkipPackaging || ds.response.DesktopPath == "" {
		step := o.startStep("Build platform installers")
		step.Status = "skipped"
		step.Message = "skipped - no desktop wrapper generated"
		ds.response.Steps = append(ds.response.Steps, step)
		return
	}

	step := o.startStep("Build platform installers")
	if ds.req.DryRun {
		step.Status = "skipped"
		step.Message = "dry run - would build MSI/PKG/AppImage installers"
		ds.response.Steps = append(ds.response.Steps, step)
		return
	}

	installCtx, cancel := context.WithTimeout(ds.ctx, ds.effectiveTimeout)
	defer cancel()

	installers, err := o.buildInstallersWithPnpm(installCtx, ds.response.DesktopPath, ds.installerTargets)
	if err != nil {
		o.failStep(&step, fmt.Sprintf("installer build failed: %v", err))
	} else {
		ds.response.Installers = installers
		o.successStep(&step, fmt.Sprintf("built installers for %d platform(s)", len(ds.installerTargets)))
	}
	ds.response.Steps = append(ds.response.Steps, step)
}

func (o *Orchestrator) deployVisualValidation(ds *deployState) {
	if !ds.req.VisualValidation || ds.req.SkipValidation || ds.response.DesktopPath == "" {
		return
	}

	step := o.startStep("Visual validation")
	if ds.req.DryRun {
		step.Status = "skipped"
		step.Message = "dry run - would run visual validation with screen recording"
	} else {
		step.Status = "warning"
		step.Message = "visual validation: review video at /api/v1/validations endpoint"
		ds.response.NextSteps = append(ds.response.NextSteps,
			"Review the recorded smoke test video via the validation API",
			"Approve or reject at POST /api/v1/validations/{id}/review",
		)
	}
	ds.response.Steps = append(ds.response.Steps, step)
}

// deployFinalizeAndPublish publishes and determines overall status.
func (o *Orchestrator) deployFinalizeAndPublish(ds *deployState) {
	if o.publishedVersionsRepo != nil && !ds.req.DryRun {
		step := o.startStep("Publish to LPBS")
		o.publishToLPBS(ds.ctx, ds.profile, ds.req, ds.response, &step)
		ds.response.Steps = append(ds.response.Steps, step)
	}

	allPassed := true
	for _, s := range ds.response.Steps {
		if s.Status == "failed" {
			allPassed = false
			break
		}
	}

	if allPassed {
		ds.response.Status = "success"
		if len(ds.response.Installers) > 0 {
			ds.response.NextSteps = []string{
				"Download installers from the Installers field in this response",
				fmt.Sprintf("Or find them at: %s/dist-electron/", ds.response.DesktopPath),
			}
		} else if ds.response.DesktopPath != "" {
			ds.response.NextSteps = []string{
				fmt.Sprintf("cd %s", ds.response.DesktopPath),
				"pnpm install",
				"pnpm run dist:all  # Build installers for all platforms",
			}
		} else {
			ds.response.NextSteps = []string{
				fmt.Sprintf("cd %s", filepath.Join(resolveScenarioDir(o.vrooli, ds.profile.Scenario), "platforms", "electron")),
				"pnpm install",
				"pnpm run dist:all  # Build installers for all platforms",
			}
		}
	} else {
		ds.response.Status = "failed"
	}
}

func resolveRepoRoot() string {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		return ""
	}
	return root
}

func resolveTopLevelScenarioDir(repoRoot string) string {
	contract, err := repocontract.LoadDefault(repoRoot)
	if err != nil {
		return ""
	}
	resolved, err := contract.TopLevelDir(repoRoot, "scenarios")
	if err != nil {
		return ""
	}
	return resolved
}

func resolveScenarioDir(repoRoot, scenario string) string {
	resolved, err := repocontract.ResolveScenarioPath(repoRoot, strings.TrimSpace(scenario))
	if err != nil {
		return ""
	}
	return resolved
}

// blockingDependencies lists dependencies that require swaps for desktop deployment.
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

	deps, err := shared.GetScenarioDependencies(ctx, profile.Scenario)
	if err != nil {
		o.log("warn", map[string]interface{}{
			"msg":      "could not fetch scenario dependencies for blocker check",
			"scenario": profile.Scenario,
			"error":    err.Error(),
		})
		return nil
	}

	appliedSwaps, _ := o.profileRepo.GetSwaps(ctx, profileID)
	swappedDeps := make(map[string]bool)
	for _, swap := range appliedSwaps {
		swappedDeps[swap.From] = true
	}

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

func (o *Orchestrator) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
