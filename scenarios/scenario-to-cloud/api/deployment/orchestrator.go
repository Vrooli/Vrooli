// DOC: docs/reference/activation-and-reconciliation.md — deployment stages and status transitions
package deployment

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"scenario-to-cloud/bundle"
	"scenario-to-cloud/credentials"
	"scenario-to-cloud/dns"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/internal/stringutil"
	"scenario-to-cloud/persistence"
	"scenario-to-cloud/reach"
	"scenario-to-cloud/reconcile"
	"scenario-to-cloud/release"
	"scenario-to-cloud/secrets"
	"scenario-to-cloud/sshidentity"
	"scenario-to-cloud/vps"
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
	reach             reach.Reach
	releaseBuilder    ReleaseBuilder
	credentials       CredentialBinder
	backups           vps.RecoveryPointRecorder
	secretsFetcher    secrets.Fetcher
	secretsGenerator  secrets.GeneratorFunc
	dnsService        dns.Service
	historyRecorder   HistoryRecorder
	manifestRefresher ManifestRefresher
	logger            func(msg string, fields map[string]interface{})
}

// ReleaseBuilder builds (or returns) the release artifact set for a manifest
// and target platform: bundle, release manifest and native control plane
// beneath releases/<digest>/. Deployments deliver only built releases.
type ReleaseBuilder func(ctx context.Context, manifest domain.CloudManifest, platform release.Platform) (release.Release, error)

// CredentialBinder binds the credential lifecycle to one deployment target
// (SSH or Bridge) so credentials.provision and grants.revoke run through the
// credential authority instead of a private writer.
type CredentialBinder func(ctx context.Context, deploymentID string, target identity.TargetRef, manifest domain.CloudManifest) (*credentials.Service, error)

// OrchestratorConfig holds configuration for creating an Orchestrator.
type OrchestratorConfig struct {
	Repo              *persistence.Repository
	ProgressHub       *Hub
	Reach             reach.Reach
	ReleaseBuilder    ReleaseBuilder
	Credentials       CredentialBinder
	Backups           vps.RecoveryPointRecorder
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
		reach:             cfg.Reach,
		releaseBuilder:    cfg.ReleaseBuilder,
		credentials:       cfg.Credentials,
		backups:           cfg.Backups,
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
			setDeploymentError(ctx, o.repo, deploymentID, "secrets_fetch", errMsg)
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
		setDeploymentError(ctx, o.repo, deploymentID, "secrets_validate", err.Error())
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

// EnsureBundle builds the release archive when none is recorded (or when
// forced) and persists its identity on the deployment. It is cloud-local:
// nothing reaches the target.
func (o *Orchestrator) EnsureBundle(ctx context.Context, deploymentID string, manifest domain.CloudManifest, existingBundlePath *string, force bool) (string, error) {
	emit := func(step, stepTitle, errMsg string) {
		o.progressHub.Broadcast(deploymentID, Event{Type: "deployment_error", Step: step, StepTitle: stepTitle, Error: errMsg, Timestamp: time.Now().UTC().Format(time.RFC3339)})
	}
	return o.ensureBundleBuilt(ctx, manifest, existingBundlePath, force, deploymentID, emit)
}

// ensureBundleBuilt builds the release artifact set (bundle, release
// manifest, native control plane) or returns the recorded bundle path. The
// target platform is negotiated through reach when a target is bound; a
// fresh host defaults to linux/amd64, the platform every release supports.
func (o *Orchestrator) ensureBundleBuilt(
	ctx context.Context,
	manifest domain.CloudManifest,
	existingBundlePath *string,
	forceBundleBuild bool,
	deploymentID string,
	emitError func(step, stepTitle, errMsg string),
) (string, error) {
	if !forceBundleBuild && existingBundlePath != nil && *existingBundlePath != "" {
		return *existingBundlePath, nil
	}
	if o.releaseBuilder == nil {
		err := fmt.Errorf("release builder is not configured; a deployment delivers only built releases")
		setDeploymentError(ctx, o.repo, deploymentID, "bundle_build", err.Error())
		emitError("bundle_build", "Building release", err.Error())
		return "", err
	}
	platform := release.Platform{GOOS: "linux", GOARCH: "amd64"}
	if o.reach != nil {
		if dep, err := o.repo.GetDeployment(ctx, deploymentID); err == nil && dep != nil && !dep.Target.IsZero() {
			target := dep.Target
			if target.Locator.Workdir == "" && manifest.Target.VPS != nil {
				target.Locator = identity.TargetLocator{Host: manifest.Target.VPS.Host, Port: manifest.Target.VPS.Port, User: manifest.Target.VPS.User, Workdir: manifest.Target.VPS.Workdir}
			}
			if caps, err := o.reach.Negotiate(ctx, target); caps.Platform != "" {
				if goos, goarch, ok := strings.Cut(caps.Platform, "/"); ok {
					platform = release.Platform{GOOS: goos, GOARCH: goarch}
				}
			} else if err != nil {
				o.log("target platform not negotiated; building the default control plane", map[string]interface{}{"deployment_id": deploymentID, "error": err.Error()})
			}
		}
	}
	buildStart := time.Now()
	rel, err := o.releaseBuilder(ctx, manifest, platform)
	if err != nil {
		setDeploymentError(ctx, o.repo, deploymentID, "bundle_build", err.Error())
		emitError("bundle_build", "Building release", err.Error())
		o.appendHistoryEvent(ctx, deploymentID, domain.HistoryEvent{
			Type: domain.EventBundleBuilt, Timestamp: time.Now().UTC(), Message: "Release build failed", Details: err.Error(),
			DurationMs: time.Since(buildStart).Milliseconds(), Success: boolPtr(false),
		})
		return "", err
	}
	bundlePath := rel.BundlePath()
	size := int64(0)
	if info, statErr := os.Stat(bundlePath); statErr == nil {
		size = info.Size()
	}
	o.appendHistoryEvent(ctx, deploymentID, domain.HistoryEvent{
		Type: domain.EventBundleBuilt, Timestamp: time.Now().UTC(), Message: "Release built locally",
		Details:    fmt.Sprintf("Release: %s\nPath: %s\nPlatform: %s", rel.Digest, bundlePath, platform),
		DurationMs: time.Since(buildStart).Milliseconds(), Success: boolPtr(true), BundleHash: rel.Manifest.BundleSHA256,
	})
	if err := o.repo.UpdateDeploymentBundle(ctx, deploymentID, bundlePath, rel.Manifest.BundleSHA256, size); err != nil {
		o.log("failed to update bundle info", map[string]interface{}{"error": err.Error()})
	}
	return bundlePath, nil
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

	boundKey, keyErr := credentials.ResolveSSHKeyPath(ctx, o.repo, deploymentID)
	if keyErr != nil {
		o.log("resolve ssh key binding failed", map[string]interface{}{"deployment_id": deploymentID, "error": keyErr.Error()})
	}
	resolved, err := resolver.Resolve(boundKey, existing)
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

	return resolved, manifest
}

func (o *Orchestrator) verifyAndPersistIdentity(
	ctx context.Context,
	deploymentID string,
	manifest domain.CloudManifest,
	identity sshidentity.DeploymentSSHIdentity,
) {
	verified := identity.Clone()
	if verified.AuthMode == sshidentity.AuthModeExplicitKey && o.reach != nil {
		state, err := vps.VerifyAuthorizedKey(ctx, vps.Prober{Reach: o.reach, Target: o.targetFor(ctx, deploymentID, manifest)}, verified)
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
func setDeploymentError(ctx context.Context, repo interface {
	UpdateDeploymentStatus(ctx context.Context, id string, status domain.DeploymentStatus, errorMsg, errorStep *string) error
}, id, step, errMsg string,
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
	// Releases a recovery point references and every retained (active or
	// predecessor) release survive cleanup regardless of age or lease state.
	if points, err := o.repo.ListRecoveryPoints(ctx, deploymentID); err == nil {
		var nonTerminal []string
		if ops, oerr := o.repo.ListOperationsByDeployment(ctx, deploymentID); oerr == nil {
			for _, op := range ops {
				if !op.State.IsTerminal() {
					nonTerminal = append(nonTerminal, op.ID)
				}
			}
		}
		protection := reconcile.Protect(strings.Join(protect, ""), "", points, nonTerminal)
		protect = append(protect, protection.ReleaseDigests...)
	}

	req := domain.VPSBundleGCRequest{
		ScenarioID:    manifest.Scenario.ID,
		KeepLatest:    bundle.DefaultVPSBundleKeepLatest,
		ProtectSHA256: protect,
		DryRun:        false,
	}

	gcStart := time.Now()
	resp := bundle.GCTargetReleases(ctx, o.reach, o.targetFor(ctx, deploymentID, manifest), deploymentID, manifest.Scenario.ID, req)

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

	req := domain.VPSBundleGCRequest{
		ScenarioID:    manifest.Scenario.ID,
		KeepLatest:    bundle.DefaultVPSBundleKeepLatest,
		ProtectSHA256: protect,
		DryRun:        false,
	}

	gcStart := time.Now()
	resp := bundle.GCTargetReleases(ctx, o.reach, o.targetFor(ctx, deploymentID, manifest), deploymentID, manifest.Scenario.ID, req)
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

// targetFor is the reach target for a deployment: the recorded binding, or
// the manifest locator for a record that predates target bindings.
func (o *Orchestrator) targetFor(ctx context.Context, deploymentID string, manifest domain.CloudManifest) identity.TargetRef {
	if dep, err := o.repo.GetDeployment(ctx, deploymentID); err == nil && dep != nil && !dep.Target.IsZero() {
		target := dep.Target
		if target.Locator.Workdir == "" && manifest.Target.VPS != nil {
			target.Locator.Workdir = manifest.Target.VPS.Workdir
		}
		return target
	}
	return domain.TargetRefFromManifest(manifest)
}
