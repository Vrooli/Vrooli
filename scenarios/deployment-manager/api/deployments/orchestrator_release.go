package deployments

import (
	"context"
	"fmt"
	"time"

	"deployment-manager/profiles"
	"deployment-manager/releases"
)

// RunDeploy drives a release through the same pipeline used by DeployDesktop,
// but from a programmatic call site (the releases handler). It returns a
// summary result shaped for the releases.Orchestrator seam.
func (o *Orchestrator) RunDeploy(ctx context.Context, req releases.DeployRequest) (*releases.DeployResult, error) {
	internal := DeployDesktopRequest{
		ProfileID:      req.ProfileID,
		Platforms:      req.Platforms,
		GitCommitHash:  req.GitCommitHash,
		ReleaseID:      req.ReleaseID,
		Channel:        req.Channel,
		ReleaseVersion: req.ReleaseVersion,
	}

	ds := &deployState{
		req: internal,
		response: &DeployDesktopResponse{
			ProfileID: internal.ProfileID,
			Steps:     make([]OrchestrationStep, 0),
		},
		ctx:             ctx,
		scenarioBaseDir: resolveTopLevelScenarioDir(o.vrooli),
	}
	ds.effectiveTimeout = 20 * time.Minute
	ds.deploymentMode = "bundled"
	ds.buildPlatforms = resolveBuildPlatforms(internal.Platforms)
	ds.installerTargets = resolveInstallerTargets(internal.Platforms)

	_ = o.deployLoadProfile(ds)
	o.deployCheckCloudHealth(ds)
	o.deployCheckLPBSReadiness(ds)
	_ = o.deployValidateAndSign(ds)
	_ = o.deployAssembleManifest(ds)
	_ = o.deployBuildBinaries(ds)
	_ = o.deployPackageAndInstall(ds)
	o.deployFinalizeAndPublish(ds)
	o.deployVerifyUpdateEndpoints(ds)

	return summarizeResult(ds), nil
}

// deployCheckCloudHealth probes scenario-to-cloud for LPBS deployment health.
// A failure fails the step and marks the release failed, but does not panic.
func (o *Orchestrator) deployCheckCloudHealth(ds *deployState) {
	if o.cloudClient == nil {
		// No client wired (e.g. in unit tests). Skip silently.
		return
	}
	step := o.startStep("Check LPBS deployment health")
	result, err := o.cloudClient.CheckLPBSHealth(ds.ctx)
	if err != nil {
		o.failStep(&step, fmt.Sprintf("cloud-health probe failed: %v", err))
		ds.response.Steps = append(ds.response.Steps, step)
		o.markReleaseFailed(ds, releases.StatusFailed)
		return
	}
	if !result.Healthy {
		o.failStep(&step, fmt.Sprintf("LPBS cloud deployment unhealthy: %s", result.Details))
		ds.response.Steps = append(ds.response.Steps, step)
		o.markReleaseFailed(ds, releases.StatusFailed)
		return
	}
	o.successStep(&step, "LPBS cloud deployment healthy")
	ds.response.Steps = append(ds.response.Steps, step)
}

// deployCheckLPBSReadiness calls LPBS's deploy-readiness endpoint to confirm
// the app registry, storage, and remote profile are configured.
func (o *Orchestrator) deployCheckLPBSReadiness(ds *deployState) {
	if o.lpbsClient == nil || o.lpbsConfigRepo == nil {
		return
	}
	step := o.startStep("Check LPBS upload readiness")
	cfg, err := o.lpbsConfigRepo.Get(ds.ctx, ds.req.ProfileID)
	if err != nil {
		o.failStep(&step, fmt.Sprintf("load lpbs config: %v", err))
		ds.response.Steps = append(ds.response.Steps, step)
		o.markReleaseFailed(ds, releases.StatusFailed)
		return
	}
	if cfg == nil || cfg.LPBSAppKey == "" {
		step.Status = "skipped"
		step.Message = "no lpbs release config for profile; skipping readiness gate"
		ds.response.Steps = append(ds.response.Steps, step)
		return
	}
	result, err := o.lpbsClient.CheckDeployReadiness(ds.ctx, &LPBSReadinessRequest{
		AppKey:        cfg.LPBSAppKey,
		RemoteProfile: cfg.LPBSRemoteProfile,
		Channel:       effectiveChannel(ds.req.Channel, cfg.DefaultChannel),
	})
	if err != nil {
		o.failStep(&step, fmt.Sprintf("readiness call failed: %v", err))
		ds.response.Steps = append(ds.response.Steps, step)
		o.markReleaseFailed(ds, releases.StatusFailed)
		return
	}
	if !result.Ready {
		msg := result.Error
		if msg == "" {
			msg = fmt.Sprintf("readiness gates pending: %d gate(s) not ready", len(result.Gates))
		}
		o.failStep(&step, msg)
		ds.response.Steps = append(ds.response.Steps, step)
		o.markReleaseFailed(ds, releases.StatusFailed)
		return
	}
	o.successStep(&step, "LPBS upload prerequisites satisfied")
	ds.response.Steps = append(ds.response.Steps, step)
}

// deployVerifyUpdateEndpoints calls LPBS's verify endpoint per platform
// after publish. On any mismatch the release is marked verify_failed.
func (o *Orchestrator) deployVerifyUpdateEndpoints(ds *deployState) {
	if o.lpbsClient == nil || o.lpbsConfigRepo == nil || ds.req.ReleaseID == "" {
		return
	}
	cfg, err := o.lpbsConfigRepo.Get(ds.ctx, ds.req.ProfileID)
	if err != nil || cfg == nil || cfg.LPBSAppKey == "" {
		return
	}
	step := o.startStep("Verify update endpoints")
	channel := effectiveChannel(ds.req.Channel, cfg.DefaultChannel)
	version := ds.req.ReleaseVersion

	var evidence []releases.VerificationItem
	allMatch := true
	for _, pv := range ds.response.PublishedVersions {
		platform := pv.Platform
		expected := version
		if expected == "" {
			expected = pv.Version
		}
		result, verr := o.lpbsClient.Verify(ds.ctx, &LPBSVerifyRequest{
			AppKey:          cfg.LPBSAppKey,
			Channel:         channel,
			Platform:        platform,
			ExpectedVersion: expected,
		})
		item := releases.VerificationItem{
			Platform:        platform,
			Channel:         channel,
			ExpectedVersion: expected,
			CheckedAt:       time.Now().UTC(),
		}
		if verr != nil {
			item.Error = verr.Error()
			allMatch = false
		} else if result != nil {
			item.ObservedVersion = result.ObservedVersion
			item.SHA512Match = result.SHA512Match
			item.Match = result.Match
			if result.Error != "" {
				item.Error = result.Error
			}
			if !result.Match {
				allMatch = false
			}
		}
		evidence = append(evidence, item)
	}

	if o.releasesRepo != nil {
		if serr := o.releasesRepo.SetVerificationEvidence(ds.ctx, ds.req.ReleaseID, evidence); serr != nil {
			o.log("warn", map[string]interface{}{
				"msg":   "persist verification evidence failed",
				"error": serr.Error(),
			})
		}
		if allMatch {
			_ = o.releasesRepo.UpdateStatus(ds.ctx, ds.req.ReleaseID, releases.StatusPublished)
			_ = o.releasesRepo.MarkSuperseded(ds.ctx, ds.req.ProfileID, channel, ds.req.ReleaseID)
		} else {
			_ = o.releasesRepo.UpdateStatus(ds.ctx, ds.req.ReleaseID, releases.StatusVerifyFailed)
		}
	}

	if allMatch {
		o.successStep(&step, fmt.Sprintf("verified %d platform(s) against channel %s", len(evidence), channel))
	} else {
		o.failStep(&step, fmt.Sprintf("verification failed for one or more platforms on channel %s", channel))
		ds.response.Status = "verify_failed"
	}
	ds.response.Steps = append(ds.response.Steps, step)
}

// markReleaseFailed transitions the release record to failed, if wired.
func (o *Orchestrator) markReleaseFailed(ds *deployState, status string) {
	if o.releasesRepo == nil || ds.req.ReleaseID == "" {
		return
	}
	_ = o.releasesRepo.UpdateStatus(ds.ctx, ds.req.ReleaseID, status)
}

// effectiveChannel picks an explicit request channel over the profile default.
func effectiveChannel(requested, defaultChannel string) string {
	if requested != "" {
		return requested
	}
	if defaultChannel != "" {
		return defaultChannel
	}
	return "stable"
}

// loadLPBSConfigForPublish loads the LPBS coords that `publishToLPBS` needs.
// Missing config is allowed (publish step degrades to a warning).
func (o *Orchestrator) loadLPBSConfigForPublish(ctx context.Context, profileID string) *profiles.LPBSReleaseConfig {
	if o.lpbsConfigRepo == nil {
		return nil
	}
	cfg, err := o.lpbsConfigRepo.Get(ctx, profileID)
	if err != nil {
		o.log("warn", map[string]interface{}{
			"msg":   "load lpbs config failed",
			"error": err.Error(),
		})
		return nil
	}
	return cfg
}

// summarizeResult converts the internal deployState into the releases-package
// result shape expected by the releases.Handler.
func summarizeResult(ds *deployState) *releases.DeployResult {
	result := &releases.DeployResult{
		Status:    ds.response.Status,
		ReleaseID: ds.req.ReleaseID,
	}
	for _, s := range ds.response.Steps {
		result.Steps = append(result.Steps, releases.Step{
			Name:    s.Name,
			Status:  s.Status,
			Message: s.Message,
			Error:   s.Error,
		})
	}
	for _, pv := range ds.response.PublishedVersions {
		result.PublishedVersions = append(result.PublishedVersions, releases.PublishedVersionRef{
			Platform:      pv.Platform,
			Version:       pv.Version,
			GitCommitHash: pv.GitCommitHash,
			ArtifactID:    pv.ArtifactID,
		})
	}
	return result
}

// Compile-time check that the Orchestrator satisfies the releases.Orchestrator seam.
var _ releases.Orchestrator = (*Orchestrator)(nil)
