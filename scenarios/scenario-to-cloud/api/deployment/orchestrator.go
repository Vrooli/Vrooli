// DOC: docs/reference/deployment-lifecycle.md — deployment stages and status transitions
package deployment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"scenario-to-cloud/bundle"
	"scenario-to-cloud/dns"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/stringutil"
	"scenario-to-cloud/persistence"
	"scenario-to-cloud/secrets"
	"scenario-to-cloud/ssh"
	"scenario-to-cloud/sshidentity"
	"scenario-to-cloud/vps"
	"scenario-to-cloud/vps/preflight"
)

// progressHubAdapter adapts the deployment package's Hub to the vps.ProgressBroadcaster interface.
type progressHubAdapter struct {
	hub *Hub
}

// Broadcast implements vps.ProgressBroadcaster by converting vps.ProgressEvent to deployment.Event.
func (a *progressHubAdapter) Broadcast(deploymentID string, event vps.ProgressEvent) {
	mainEvent := Event{
		Type:          event.Type,
		Step:          event.Step,
		StepTitle:     event.StepTitle,
		Progress:      event.Progress,
		Message:       event.Message,
		Error:         event.Error,
		ErrorCategory: event.ErrorCategory,
		Retryable:     event.Retryable,
		Hint:          event.Hint,
		Timestamp:     event.Timestamp,
	}
	a.hub.Broadcast(deploymentID, mainEvent)
}

// progressRepoAdapter adapts the persistence.Repository to the vps.ProgressRepo interface.
type progressRepoAdapter struct {
	repo *persistence.Repository
}

// UpdateDeploymentProgress implements vps.ProgressRepo.
func (a *progressRepoAdapter) UpdateDeploymentProgress(ctx context.Context, id, step string, percent float64) error {
	return a.repo.UpdateDeploymentProgress(ctx, id, step, percent)
}

// Orchestrator coordinates the full deployment pipeline.
// It encapsulates the dependencies needed to execute deployments and
// provides a clean separation between HTTP handlers and business logic.
type Orchestrator struct {
	repo              *persistence.Repository
	progressHub       *Hub
	sshRunner         ssh.Runner
	scpRunner         ssh.SCPRunner
	secretsFetcher    secrets.Fetcher
	secretsGenerator  secrets.GeneratorFunc
	dnsService        dns.Service
	historyRecorder   HistoryRecorder
	manifestRefresher ManifestRefresher
	logger            func(msg string, fields map[string]interface{})
}

// OrchestratorConfig holds configuration for creating an Orchestrator.
type OrchestratorConfig struct {
	Repo              *persistence.Repository
	ProgressHub       *Hub
	SSHRunner         ssh.Runner
	SCPRunner         ssh.SCPRunner
	SecretsFetcher    secrets.Fetcher
	SecretsGenerator  secrets.GeneratorFunc
	DNSService        dns.Service
	HistoryRecorder   HistoryRecorder
	ManifestRefresher ManifestRefresher
	Logger            func(msg string, fields map[string]interface{})
}

// NewOrchestrator creates a new orchestrator with the given dependencies.
func NewOrchestrator(cfg OrchestratorConfig) *Orchestrator {
	return &Orchestrator{
		repo:              cfg.Repo,
		progressHub:       cfg.ProgressHub,
		sshRunner:         cfg.SSHRunner,
		scpRunner:         cfg.SCPRunner,
		secretsFetcher:    cfg.SecretsFetcher,
		secretsGenerator:  cfg.SecretsGenerator,
		dnsService:        cfg.DNSService,
		historyRecorder:   cfg.HistoryRecorder,
		manifestRefresher: cfg.ManifestRefresher,
		logger:            cfg.Logger,
	}
}

// RefreshManifest regenerates the manifest from current scenario state.
// This is called by the handler before validation when ForceBundleBuild=true.
// Returns the refreshed manifest or the original if refresh fails/unavailable.
func (o *Orchestrator) RefreshManifest(ctx context.Context, base domain.CloudManifest) (domain.CloudManifest, error) {
	if o.manifestRefresher == nil {
		return base, nil
	}
	return o.manifestRefresher.RefreshManifest(ctx, base)
}

// ExecuteOptions controls which steps run during execution.
type ExecuteOptions struct {
	RunPreflight     bool
	ForceBundleBuild bool
}

// RunPipeline executes the full deployment with progress tracking.
// The runID parameter uniquely identifies this execution for idempotency tracking.
func (o *Orchestrator) RunPipeline(
	id, runID string,
	manifest domain.CloudManifest,
	existingBundlePath *string,
	providedSecrets map[string]string,
	options ExecuteOptions,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	progress := 0.0
	canonicalIdentity, effectiveManifest := o.resolveAndPersistIdentity(ctx, id, manifest)
	manifest = effectiveManifest

	// Helper to emit and persist progress
	emitProgress := func(eventType, step, stepTitle string, pct float64, errMsg string) {
		event := Event{
			Type:      eventType,
			Step:      step,
			StepTitle: stepTitle,
			Progress:  pct,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		if errMsg != "" {
			event.Error = errMsg
		}

		// Broadcast to SSE clients
		o.progressHub.Broadcast(id, event)

		// Persist to database for reconnection
		if err := o.repo.UpdateDeploymentProgress(ctx, id, step, pct); err != nil {
			o.log("failed to persist progress", map[string]interface{}{"error": err.Error()})
		}
	}

	// Helper for errors (used for bundle_build step which isn't in VPS runners)
	emitError := func(step, stepTitle, errMsg string) {
		event := Event{
			Type:      "deployment_error",
			Step:      step,
			StepTitle: stepTitle,
			Progress:  progress,
			Error:     errMsg,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		o.progressHub.Broadcast(id, event)
	}

	// Log the run_id for traceability
	o.log("deployment pipeline started", map[string]interface{}{
		"deployment_id": id,
		"run_id":        runID,
		"scenario_id":   manifest.Scenario.ID,
	})

	// Record manifest refresh in history when ForceBundleBuild is requested
	// Note: The actual refresh happens in the handler BEFORE validation
	// This just records the event for tracking purposes
	if options.ForceBundleBuild {
		o.appendHistoryEvent(ctx, id, domain.HistoryEvent{
			Type:      domain.EventManifestRefreshed,
			Timestamp: time.Now().UTC(),
			Message:   "Manifest refreshed from current scenario state",
			Success:   boolPtr(true),
		})
		progress += vps.StepWeights["manifest_refresh"]
		emitProgress("step_completed", "manifest_refresh", "Manifest refreshed", progress, "")
	}

	// Fetch and validate secrets
	if err := o.ensureSecretsAvailable(ctx, &manifest, providedSecrets, id, emitError); err != nil {
		return // Error already logged and emitted
	}

	// Step 1: Build bundle (if not already built)
	emitProgress("step_started", "bundle_build", "Building bundle", progress, "")

	bundlePath, err := o.ensureBundleBuilt(ctx, manifest, existingBundlePath, options.ForceBundleBuild, id, emitError)
	if err != nil {
		return // Error already logged and emitted
	}

	progress += vps.StepWeights["bundle_build"]
	emitProgress("step_completed", "bundle_build", "Building bundle", progress, "")

	// Optional preflight checks
	if options.RunPreflight {
		preflightStart := time.Now()
		o.appendHistoryEvent(ctx, id, domain.HistoryEvent{
			Type:      domain.EventPreflightStarted,
			Timestamp: preflightStart.UTC(),
			Message:   "Preflight checks started",
		})

		emitProgress("step_started", "preflight", "Running preflight checks", progress, "")
		preflightResp := preflight.Run(ctx, manifest, o.dnsService, o.sshRunner, preflight.RunOptions{
			ProvidedSecrets: providedSecrets,
		})

		// Automatic remediation: VPS bundle cache can accumulate across repeated redeploys.
		// If preflight fails due to low disk, attempt a single VPS bundle GC pass and re-run preflight.
		if !preflightResp.OK && hasFailingPreflightCheck(preflightResp, domain.PreflightDiskFreeID) {
			if gcApplied := o.tryAutoVPSBundleGC(ctx, id, manifest); gcApplied {
				preflightResp = preflight.Run(ctx, manifest, o.dnsService, o.sshRunner, preflight.RunOptions{
					ProvidedSecrets: providedSecrets,
				})
			}
		}

		preflightJSON, _ := json.Marshal(preflightResp)
		if err := o.repo.UpdateDeploymentPreflightResult(ctx, id, preflightJSON); err != nil {
			o.log("failed to save preflight result", map[string]interface{}{"error": err.Error()})
		}
		o.progressHub.Broadcast(id, Event{
			Type:            "preflight_result",
			Step:            "preflight",
			StepTitle:       "Running preflight checks",
			Progress:        progress,
			PreflightResult: &preflightResp,
			Timestamp:       time.Now().UTC().Format(time.RFC3339),
		})
		if !preflightResp.OK {
			failCount := 0
			for _, check := range preflightResp.Checks {
				if check.Status == domain.PreflightFail {
					failCount++
				}
			}
			errMsg := "Preflight checks failed"
			if failCount > 0 {
				errMsg = fmt.Sprintf("Preflight checks failed (%d issue%s)", failCount, pluralize(failCount))
			}
			o.appendHistoryEvent(ctx, id, domain.HistoryEvent{
				Type:       domain.EventPreflightCompleted,
				Timestamp:  time.Now().UTC(),
				Message:    errMsg,
				Details:    FormatPreflightFailureDetails(preflightResp),
				DurationMs: time.Since(preflightStart).Milliseconds(),
				Success:    boolPtr(false),
			})
			setDeploymentError(o.repo, ctx, id, "preflight", errMsg)
			emitError("preflight", "Running preflight checks", errMsg)
			return
		}
		o.appendHistoryEvent(ctx, id, domain.HistoryEvent{
			Type:       domain.EventPreflightCompleted,
			Timestamp:  time.Now().UTC(),
			Message:    "Preflight checks passed",
			DurationMs: time.Since(preflightStart).Milliseconds(),
			Success:    boolPtr(true),
		})
		progress += vps.StepWeights["preflight"]
		emitProgress("step_completed", "preflight", "Running preflight checks", progress, "")
	}

	// Step 2: VPS Setup
	if err := o.repo.UpdateDeploymentStatus(ctx, id, domain.StatusSetupRunning, nil, nil); err != nil {
		o.log("failed to update status", map[string]interface{}{"error": err.Error()})
	}

	setupStart := time.Now()
	o.appendHistoryEvent(ctx, id, domain.HistoryEvent{
		Type:      domain.EventSetupStarted,
		Timestamp: setupStart.UTC(),
		Message:   "VPS setup started",
	})

	hubAdapter := &progressHubAdapter{hub: o.progressHub}
	repoAdapter := &progressRepoAdapter{repo: o.repo}
	setupResult := vps.RunSetupWithProgress(ctx, manifest, bundlePath, o.sshRunner, o.scpRunner, hubAdapter, repoAdapter, id, &progress)
	setupJSON, _ := json.Marshal(setupResult)
	if err := o.repo.UpdateDeploymentSetupResult(ctx, id, setupJSON); err != nil {
		o.log("failed to save setup result", map[string]interface{}{"error": err.Error()})
	}

	if !setupResult.OK {
		// VPS runner already emitted deployment_error event with correct step
		failedStep := setupResult.FailedStep
		if failedStep == "" {
			failedStep = "vps_setup"
		}
		o.appendHistoryEvent(ctx, id, domain.HistoryEvent{
			Type:       domain.EventSetupCompleted,
			Timestamp:  time.Now().UTC(),
			Message:    "VPS setup failed",
			Details:    setupResult.Error,
			DurationMs: time.Since(setupStart).Milliseconds(),
			Success:    boolPtr(false),
			StepName:   failedStep,
		})
		setDeploymentError(o.repo, ctx, id, failedStep, setupResult.Error)
		return
	}

	o.appendHistoryEvent(ctx, id, domain.HistoryEvent{
		Type:       domain.EventSetupCompleted,
		Timestamp:  time.Now().UTC(),
		Message:    "VPS setup completed",
		DurationMs: time.Since(setupStart).Milliseconds(),
		Success:    boolPtr(true),
	})

	// Step 3: VPS Deploy
	if err := o.repo.UpdateDeploymentStatus(ctx, id, domain.StatusDeploying, nil, nil); err != nil {
		o.log("failed to update status", map[string]interface{}{"error": err.Error()})
	}

	deployStart := time.Now()
	o.appendHistoryEvent(ctx, id, domain.HistoryEvent{
		Type:      domain.EventDeployStarted,
		Timestamp: deployStart.UTC(),
		Message:   "Deployment started",
	})

	deployResult := vps.RunDeployWithProgress(ctx, manifest, o.sshRunner, o.secretsGenerator, providedSecrets, hubAdapter, repoAdapter, id, &progress, vps.DeployOptions{})
	deployJSON, _ := json.Marshal(deployResult)
	if err := o.repo.UpdateDeploymentDeployResult(ctx, id, deployJSON, deployResult.OK); err != nil {
		o.log("failed to save deploy result", map[string]interface{}{"error": err.Error()})
	}

	if !deployResult.OK {
		// VPS runner already emitted deployment_error event with correct step
		failedStep := deployResult.FailedStep
		if failedStep == "" {
			failedStep = "vps_deploy"
		}
		o.appendHistoryEvent(ctx, id, domain.HistoryEvent{
			Type:       domain.EventDeployFailed,
			Timestamp:  time.Now().UTC(),
			Message:    "Deployment failed",
			Details:    deployResult.Error,
			DurationMs: time.Since(deployStart).Milliseconds(),
			Success:    boolPtr(false),
			StepName:   failedStep,
		})
		setDeploymentError(o.repo, ctx, id, failedStep, deployResult.Error)
		return
	}

	o.appendHistoryEvent(ctx, id, domain.HistoryEvent{
		Type:       domain.EventDeployCompleted,
		Timestamp:  time.Now().UTC(),
		Message:    "Deployment completed",
		DurationMs: time.Since(deployStart).Milliseconds(),
		Success:    boolPtr(true),
	})

	// Proactively enforce VPS bundle cache retention so repeated redeploys do not accumulate disk usage.
	// This is best-effort and must not block a successful deployment.
	o.enforceVPSBundleRetentionBestEffort(ctx, id, manifest)

	o.verifyAndPersistIdentity(ctx, id, manifest, canonicalIdentity)

	// Success!
	o.progressHub.Broadcast(id, Event{
		Type:      "completed",
		Progress:  100,
		Message:   "Deployment successful",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// RunStartPipeline starts/resumes a stopped deployment.
// Unlike RunPipeline, this skips bundle building, preflight, and VPS setup steps.
// It only runs the deploy steps needed to restart services.
func (o *Orchestrator) RunStartPipeline(
	id, runID string,
	manifest domain.CloudManifest,
	providedSecrets map[string]string,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	progress := 0.0
	canonicalIdentity, effectiveManifest := o.resolveAndPersistIdentity(ctx, id, manifest)
	manifest = effectiveManifest

	// Log the start operation
	o.log("start pipeline initiated", map[string]interface{}{
		"deployment_id": id,
		"run_id":        runID,
		"scenario_id":   manifest.Scenario.ID,
	})

	// Record start event
	startTime := time.Now()
	o.appendHistoryEvent(ctx, id, domain.HistoryEvent{
		Type:      domain.EventStarted,
		Timestamp: startTime.UTC(),
		Message:   "Deployment start/resume initiated",
	})

	// Create adapters for progress tracking
	hubAdapter := &progressHubAdapter{hub: o.progressHub}
	repoAdapter := &progressRepoAdapter{repo: o.repo}

	// Calculate normalized weights for start steps
	startWeights := vps.CalculateWeightsForSteps(vps.StartSteps)

	// Run deploy steps with start-specific options
	deployResult := vps.RunDeployWithProgress(
		ctx,
		manifest,
		o.sshRunner,
		o.secretsGenerator,
		providedSecrets,
		hubAdapter,
		repoAdapter,
		id,
		&progress,
		vps.DeployOptions{
			StepsToRun:  vps.StartSteps,
			StepWeights: startWeights,
		},
	)

	deployJSON, _ := json.Marshal(deployResult)
	if err := o.repo.UpdateDeploymentDeployResult(ctx, id, deployJSON, deployResult.OK); err != nil {
		o.log("failed to save start result", map[string]interface{}{"error": err.Error()})
	}

	if !deployResult.OK {
		failedStep := deployResult.FailedStep
		if failedStep == "" {
			failedStep = "start"
		}
		o.appendHistoryEvent(ctx, id, domain.HistoryEvent{
			Type:       domain.EventDeployFailed,
			Timestamp:  time.Now().UTC(),
			Message:    "Start failed",
			Details:    deployResult.Error,
			DurationMs: time.Since(startTime).Milliseconds(),
			Success:    boolPtr(false),
			StepName:   failedStep,
		})
		setDeploymentError(o.repo, ctx, id, failedStep, deployResult.Error)
		return
	}

	o.appendHistoryEvent(ctx, id, domain.HistoryEvent{
		Type:       domain.EventDeployCompleted,
		Timestamp:  time.Now().UTC(),
		Message:    "Deployment started successfully",
		DurationMs: time.Since(startTime).Milliseconds(),
		Success:    boolPtr(true),
	})
	o.verifyAndPersistIdentity(ctx, id, manifest, canonicalIdentity)

	// Success!
	o.progressHub.Broadcast(id, Event{
		Type:      "completed",
		Progress:  100,
		Message:   "Deployment started successfully",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// ensureSecretsAvailable fetches secrets from secrets-manager and validates user_prompt secrets.
// Returns an error if secrets cannot be fetched or validated (error already logged and emitted).
func (o *Orchestrator) ensureSecretsAvailable(
	ctx context.Context,
	manifest *domain.CloudManifest,
	providedSecrets map[string]string,
	deploymentID string,
	emitError func(step, stepTitle, errMsg string),
) error {
	// Fetch secrets from secrets-manager BEFORE building bundle
	if manifest.Secrets == nil {
		resources := manifest.Dependencies.Resources
		if manifest.Edge.Caddy.Enabled {
			resources = append(resources, "edge-dns")
		}
		resources = stringutil.OrderedUnique(resources)

		secretsCtx, secretsCancel := context.WithTimeout(ctx, 30*time.Second)
		secretsResp, err := o.secretsFetcher.FetchBundleSecrets(
			secretsCtx,
			manifest.Scenario.ID,
			secrets.DefaultDeploymentTier,
			resources,
		)
		secretsCancel()

		if err != nil {
			o.log("secrets-manager fetch failed", map[string]interface{}{
				"scenario_id": manifest.Scenario.ID,
				"error":       err.Error(),
			})
			errMsg := fmt.Sprintf("secrets-manager unavailable: %v", err)
			setDeploymentError(o.repo, ctx, deploymentID, "secrets_fetch", errMsg)
			emitError("secrets_fetch", "Fetching secrets", err.Error())
			o.appendHistoryEvent(ctx, deploymentID, domain.HistoryEvent{
				Type:      domain.EventDeployFailed,
				Timestamp: time.Now().UTC(),
				Message:   "Secrets fetch failed",
				Details:   errMsg,
				Success:   boolPtr(false),
				StepName:  "secrets_fetch",
			})
			return err
		}

		manifest.Secrets = secrets.BuildManifestSecrets(secretsResp)
		o.log("fetched secrets manifest", map[string]interface{}{
			"scenario_id":   manifest.Scenario.ID,
			"total_secrets": len(secretsResp.BundleSecrets),
		})
	}

	// Validate user_prompt secrets
	if providedSecrets == nil {
		providedSecrets = make(map[string]string)
	}
	if missing, err := vps.ValidateUserPromptSecrets(*manifest, providedSecrets); err != nil {
		o.log("missing required user_prompt secrets", map[string]interface{}{
			"scenario_id": manifest.Scenario.ID,
			"missing":     missing,
		})
		setDeploymentError(o.repo, ctx, deploymentID, "secrets_validate", err.Error())
		emitError("secrets_validate", "Validating secrets", err.Error())
		o.appendHistoryEvent(ctx, deploymentID, domain.HistoryEvent{
			Type:      domain.EventDeployFailed,
			Timestamp: time.Now().UTC(),
			Message:   "Secrets validation failed",
			Details:   err.Error(),
			Success:   boolPtr(false),
			StepName:  "secrets_validate",
		})
		return err
	}

	return nil
}

// ensureBundleBuilt builds a new bundle or returns the existing bundle path.
// Returns the bundle path or an error (error already logged and emitted).
func (o *Orchestrator) ensureBundleBuilt(
	ctx context.Context,
	manifest domain.CloudManifest,
	existingBundlePath *string,
	forceBundleBuild bool,
	deploymentID string,
	emitError func(step, stepTitle, errMsg string),
) (string, error) {
	// Use existing bundle if provided
	if !forceBundleBuild && existingBundlePath != nil && *existingBundlePath != "" {
		return *existingBundlePath, nil
	}

	// Get bundle output directory
	repoRoot, err := bundle.FindRepoRootFromCWD()
	if err != nil {
		setDeploymentError(o.repo, ctx, deploymentID, "bundle_build", err.Error())
		emitError("bundle_build", "Building bundle", err.Error())
		return "", err
	}

	outDir, err := bundle.GetLocalBundlesDir()
	if err != nil {
		setDeploymentError(o.repo, ctx, deploymentID, "bundle_build", err.Error())
		emitError("bundle_build", "Building bundle", err.Error())
		return "", err
	}

	// Clean up old bundles (keep 3 newest)
	o.cleanupOldBundles(outDir, manifest.Scenario.ID)

	// Build the bundle
	buildStart := time.Now()
	artifact, err := bundle.BuildMiniVrooliBundle(repoRoot, outDir, manifest)
	if err != nil {
		setDeploymentError(o.repo, ctx, deploymentID, "bundle_build", err.Error())
		emitError("bundle_build", "Building bundle", err.Error())
		o.appendHistoryEvent(ctx, deploymentID, domain.HistoryEvent{
			Type:       domain.EventBundleBuilt,
			Timestamp:  time.Now().UTC(),
			Message:    "Bundle build failed",
			Details:    err.Error(),
			DurationMs: time.Since(buildStart).Milliseconds(),
			Success:    boolPtr(false),
		})
		return "", err
	}

	o.appendHistoryEvent(ctx, deploymentID, domain.HistoryEvent{
		Type:       domain.EventBundleBuilt,
		Timestamp:  time.Now().UTC(),
		Message:    "Bundle built locally",
		Details:    fmt.Sprintf("Path: %s\nSize: %d bytes", artifact.Path, artifact.SizeBytes),
		DurationMs: time.Since(buildStart).Milliseconds(),
		Success:    boolPtr(true),
		BundleHash: artifact.Sha256,
	})

	// Update database with bundle info
	if err := o.repo.UpdateDeploymentBundle(ctx, deploymentID, artifact.Path, artifact.Sha256, artifact.SizeBytes); err != nil {
		o.log("failed to update bundle info", map[string]interface{}{"error": err.Error()})
	}

	return artifact.Path, nil
}

// cleanupOldBundles removes old bundles for a scenario, keeping the newest N.
func (o *Orchestrator) cleanupOldBundles(bundlesDir, scenarioID string) {
	const retentionCount = 3
	deleted, _, err := bundle.DeleteBundlesForScenario(bundlesDir, scenarioID, retentionCount)
	if err != nil {
		o.log("bundle cleanup warning", map[string]interface{}{
			"scenario_id": scenarioID,
			"error":       err.Error(),
		})
		return
	}
	if len(deleted) > 0 {
		o.log("cleaned old bundles", map[string]interface{}{
			"scenario_id": scenarioID,
			"count":       len(deleted),
		})
	}
}

func (o *Orchestrator) resolveAndPersistIdentity(ctx context.Context, deploymentID string, manifest domain.CloudManifest) (sshidentity.DeploymentSSHIdentity, domain.CloudManifest) {
	resolver := sshidentity.DefaultResolver{}
	var existing *sshidentity.DeploymentSSHIdentity

	if dep, err := o.repo.GetDeployment(ctx, deploymentID); err == nil && dep != nil {
		if parsed, parseErr := sshidentity.FromDeployment(dep); parseErr == nil {
			existing = &parsed
		}
	}

	resolved, err := resolver.Resolve(manifest, existing)
	if err != nil {
		o.log("ssh identity resolve failed", map[string]interface{}{
			"deployment_id": deploymentID,
			"error":         err.Error(),
		})
		resolved = sshidentity.DeploymentSSHIdentity{
			AuthMode:          sshidentity.AuthModeUnknown,
			VerificationState: sshidentity.VerificationUnknown,
		}
	}

	if identityJSON, marshalErr := sshidentity.Marshal(resolved); marshalErr == nil {
		if persistErr := o.repo.UpdateDeploymentSSHIdentity(ctx, deploymentID, identityJSON); persistErr != nil {
			o.log("failed to persist resolved ssh identity", map[string]interface{}{
				"deployment_id": deploymentID,
				"error":         persistErr.Error(),
			})
		}
	}

	return resolved, sshidentity.ApplyToManifest(manifest, resolved)
}

func (o *Orchestrator) verifyAndPersistIdentity(
	ctx context.Context,
	deploymentID string,
	manifest domain.CloudManifest,
	identity sshidentity.DeploymentSSHIdentity,
) {
	verified := identity.Clone()
	if verified.AuthMode == sshidentity.AuthModeExplicitKey {
		cfg := sshidentity.EffectiveSSHConfig(manifest, verified)
		inspector := sshidentity.RemoteAuthorizedKeysInspector{Runner: o.sshRunner}
		state, err := inspector.Inspect(ctx, cfg, verified)
		if err != nil {
			o.log("ssh identity verification failed", map[string]interface{}{
				"deployment_id": deploymentID,
				"error":         err.Error(),
			})
			state = sshidentity.VerificationUnknown
		}
		verified = sshidentity.ApplyVerificationResult(verified, state, time.Now().UTC())
	} else {
		verified = sshidentity.ApplyVerificationResult(verified, sshidentity.VerificationUnknown, time.Now().UTC())
	}

	identityJSON, err := sshidentity.Marshal(verified)
	if err != nil {
		o.log("failed to marshal verified ssh identity", map[string]interface{}{
			"deployment_id": deploymentID,
			"error":         err.Error(),
		})
		return
	}
	if err := o.repo.UpdateDeploymentSSHIdentity(ctx, deploymentID, identityJSON); err != nil {
		o.log("failed to persist verified ssh identity", map[string]interface{}{
			"deployment_id": deploymentID,
			"error":         err.Error(),
		})
	}
}

// appendHistoryEvent persists a history event and logs failures without impacting the request.
func (o *Orchestrator) appendHistoryEvent(ctx context.Context, deploymentID string, event domain.HistoryEvent) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	recorder := o.historyRecorder
	if recorder == nil {
		recorder = o.repo
	}
	if err := recorder.AppendHistoryEvent(ctx, deploymentID, event); err != nil {
		o.log("failed to append history event", map[string]interface{}{
			"deployment_id": deploymentID,
			"type":          event.Type,
			"error":         err.Error(),
		})
	}
}

// log writes a log message using the configured logger.
func (o *Orchestrator) log(msg string, fields map[string]interface{}) {
	if o.logger != nil {
		o.logger(msg, fields)
	}
}

// HistoryRecorder defines the interface for recording deployment history events.
type HistoryRecorder interface {
	AppendHistoryEvent(ctx context.Context, id string, event domain.HistoryEvent) error
}

// setDeploymentError is a helper to set error status on a deployment.
func setDeploymentError(repo interface {
	UpdateDeploymentStatus(ctx context.Context, id string, status domain.DeploymentStatus, errorMsg, errorStep *string) error
}, ctx context.Context, id, step, errMsg string,
) {
	_ = repo.UpdateDeploymentStatus(ctx, id, domain.StatusFailed, &errMsg, &step)
}

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// FormatPreflightFailureDetails builds a human-readable summary of failed preflight checks.
func FormatPreflightFailureDetails(resp domain.PreflightResponse) string {
	if len(resp.Checks) == 0 {
		return "No preflight checks returned"
	}

	var b strings.Builder
	for _, check := range resp.Checks {
		if check.Status != domain.PreflightFail {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		b.WriteString("- ")
		b.WriteString(check.Title)
		if check.Details != "" {
			b.WriteString(": ")
			b.WriteString(check.Details)
		}
		if check.Hint != "" {
			b.WriteString(" (hint: ")
			b.WriteString(check.Hint)
			b.WriteString(")")
		}
	}

	if b.Len() == 0 {
		return "Preflight failed (no failing checks reported)"
	}
	return b.String()
}

func hasFailingPreflightCheck(resp domain.PreflightResponse, checkID string) bool {
	for _, c := range resp.Checks {
		if c.ID == checkID && c.Status == domain.PreflightFail {
			return true
		}
	}
	return false
}

// tryAutoVPSBundleGC attempts to garbage-collect old bundles on the VPS to relieve disk pressure.
// Returns true if a GC pass was applied (and preflight should be re-run).
func (o *Orchestrator) tryAutoVPSBundleGC(ctx context.Context, deploymentID string, manifest domain.CloudManifest) bool {
	if manifest.Target.VPS == nil {
		return false
	}

	// Keep the deployment's currently recorded bundle hash (if present) as an explicit protect.
	var protect []string
	if dep, err := o.repo.GetDeployment(ctx, deploymentID); err == nil && dep != nil {
		if dep.BundleSHA256 != nil && strings.TrimSpace(*dep.BundleSHA256) != "" {
			protect = append(protect, strings.TrimSpace(*dep.BundleSHA256))
		}
	}

	cfg := ssh.ConfigFromManifest(manifest)
	req := domain.VPSBundleGCRequest{
		ScenarioID:     manifest.Scenario.ID,
		KeepLatest:     bundle.DefaultVPSBundleKeepLatest,
		ProtectSHA256:  protect,
		DryRun:         false,
	}

	gcStart := time.Now()
	resp := bundle.GCVPSBundles(ctx, o.sshRunner, cfg, manifest.Target.VPS.Workdir, req)

	// History event is intentionally non-blocking: GC is a best-effort remediation.
	details := fmt.Sprintf("keep_latest=%d deleted=%d deleted_bytes=%d before_bytes=%d after_bytes=%d dry_run=%v",
		req.KeepLatest, resp.DeletedCount, resp.DeletedBytes, resp.TotalBeforeBytes, resp.TotalAfterBytes, resp.DryRun)
	if resp.Error != "" {
		details += "\nerror: " + resp.Error
	}
	success := resp.OK
	o.appendHistoryEvent(ctx, deploymentID, domain.HistoryEvent{
		Type:       domain.EventVPSBundleGC,
		Timestamp:  gcStart.UTC(),
		Message:    "VPS bundle cache GC attempted",
		Details:    details,
		DurationMs: time.Since(gcStart).Milliseconds(),
		Success:    boolPtr(success),
		StepName:   "preflight",
	})

	return resp.OK && resp.DeletedCount > 0
}

func (o *Orchestrator) enforceVPSBundleRetentionBestEffort(ctx context.Context, deploymentID string, manifest domain.CloudManifest) {
	if manifest.Target.VPS == nil {
		return
	}

	// Protect the currently recorded bundle hash (if present). This should also be among the newest,
	// but we keep it explicit to avoid accidental deletion on clock skew or manual rollbacks.
	var protect []string
	if dep, err := o.repo.GetDeployment(ctx, deploymentID); err == nil && dep != nil {
		if dep.BundleSHA256 != nil && strings.TrimSpace(*dep.BundleSHA256) != "" {
			protect = append(protect, strings.TrimSpace(*dep.BundleSHA256))
		}
	}

	cfg := ssh.ConfigFromManifest(manifest)
	req := domain.VPSBundleGCRequest{
		ScenarioID:    manifest.Scenario.ID,
		KeepLatest:    bundle.DefaultVPSBundleKeepLatest,
		ProtectSHA256: protect,
		DryRun:        false,
	}

	gcStart := time.Now()
	resp := bundle.GCVPSBundles(ctx, o.sshRunner, cfg, manifest.Target.VPS.Workdir, req)
	if !resp.OK {
		o.log("vps bundle retention gc warning", map[string]interface{}{
			"deployment_id": deploymentID,
			"scenario_id":   manifest.Scenario.ID,
			"error":         resp.Error,
		})
	}

	// Record in history for auditability; this is operational hygiene, not a deploy stage.
	details := fmt.Sprintf("keep_latest=%d deleted=%d deleted_bytes=%d before_bytes=%d after_bytes=%d",
		req.KeepLatest, resp.DeletedCount, resp.DeletedBytes, resp.TotalBeforeBytes, resp.TotalAfterBytes)
	if resp.Error != "" {
		details += "\nerror: " + resp.Error
	}
	o.appendHistoryEvent(ctx, deploymentID, domain.HistoryEvent{
		Type:       domain.EventVPSBundleGC,
		Timestamp:  gcStart.UTC(),
		Message:    "VPS bundle cache retention enforced",
		Details:    details,
		DurationMs: time.Since(gcStart).Milliseconds(),
		Success:    boolPtr(resp.OK),
	})
}

func boolPtr(value bool) *bool {
	return &value
}
