package deployments

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"deployment-manager/profiles"
	"deployment-manager/readiness"
	"deployment-manager/releases"
)

// RunDeploy drives a release through the same pipeline used by DeployDesktop,
// but from a programmatic call site (the releases handler). It returns a
// summary result shaped for the releases.Orchestrator seam.
func (o *Orchestrator) RunDeploy(ctx context.Context, req releases.DeployRequest) (*releases.DeployResult, error) {
	internal := DeployDesktopRequest{
		ProfileID:             req.ProfileID,
		Platforms:             req.Platforms,
		GitCommitHash:         req.GitCommitHash,
		ArtifactDigest:        req.ArtifactDigest,
		ReleaseID:             req.ReleaseID,
		Channel:               req.Channel,
		ReleaseVersion:        req.ReleaseVersion,
		CandidateID:           req.CandidateID,
		DestinationRevisionID: req.DestinationRevisionID,
		AuthorizationEpoch:    req.AuthorizationEpoch,
		IdempotencyKey:        req.IdempotencyKey,
		AuthorizationCheck:    req.AuthorizationCheck,
		ReadinessReviewKey:    req.ReadinessReviewKey,
		CloudManifest:         req.CloudManifest,
		CloudDeploymentName:   req.CloudDeploymentName,
		CloudBundlePath:       req.CloudBundlePath,
		CloudBundleSHA256:     req.CloudBundleSHA256,
		CloudBundleSizeBytes:  req.CloudBundleSizeBytes,
		CloudRunPreflight:     req.CloudRunPreflight,
	}
	if req.ReleaseID != "" {
		if o.releaseIdentity == nil {
			return nil, fmt.Errorf("release identity repository is required for release execution")
		}
		candidate, err := o.validateCandidateTargets(ctx, req)
		if err != nil {
			return nil, err
		}
		if err := o.validateReleaseDestination(ctx, req); err != nil {
			return nil, err
		}
		internal.ExpectedArtifactDigests = candidateArtifactDigests(candidate)
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

	if code := o.deployLoadProfile(ds); code != 0 {
		return summarizeResult(ds), nil
	}
	if code := o.deployCloud(ds); code != 0 {
		return summarizeResult(ds), nil
	}
	o.deployCheckCloudHealth(ds)
	if releaseStageFailed(ds) {
		return summarizeResult(ds), nil
	}
	o.deployCheckLPBSReadiness(ds)
	if releaseStageFailed(ds) {
		return summarizeResult(ds), nil
	}
	if code := o.deployValidateAndSign(ds); code != 0 {
		return summarizeResult(ds), nil
	}
	if code := o.deployAssembleManifest(ds); code != 0 {
		return summarizeResult(ds), nil
	}
	if code := o.deployBuildBinaries(ds); code != 0 {
		return summarizeResult(ds), nil
	}
	if code := o.deployPackageAndInstall(ds); code != 0 {
		return summarizeResult(ds), nil
	}
	o.finalizeAndVerifyRelease(ds)

	return summarizeResult(ds), nil
}

// finalizeAndVerifyRelease keeps verification behind the publication gate.
// A refused or failed finalization must not trigger an owner read or allow a
// later verification result to obscure the failed publication stage.
func (o *Orchestrator) finalizeAndVerifyRelease(ds *deployState) {
	o.deployFinalizeAndPublish(ds)
	if releaseStageFailed(ds) {
		return
	}
	o.deployVerifyUpdateEndpoints(ds)
}

func (o *Orchestrator) validateCandidateTargets(ctx context.Context, req releases.DeployRequest) (*releases.CandidateRecord, error) {
	if req.CandidateID == "" {
		return nil, fmt.Errorf("candidate identity is required for release execution")
	}
	record, err := o.releaseIdentity.GetCandidate(ctx, req.CandidateID)
	if err != nil || record == nil {
		return nil, fmt.Errorf("load release candidate %q: %w", req.CandidateID, err)
	}
	expected := make(map[string]struct{}, len(record.Candidate.Artifacts))
	for _, artifact := range record.Candidate.Artifacts {
		expected[artifact.Target.ID] = struct{}{}
	}
	requested := make(map[string]struct{}, len(req.Platforms))
	for _, platform := range req.Platforms {
		requested[platform] = struct{}{}
	}
	if len(expected) != len(requested) {
		return nil, fmt.Errorf("release target set does not match candidate %q", req.CandidateID)
	}
	for target := range expected {
		if _, ok := requested[target]; !ok {
			return nil, fmt.Errorf("release target %q is not present in candidate %q", target, req.CandidateID)
		}
	}
	if strings.TrimSpace(req.ArtifactDigest) == "" || req.ArtifactDigest != record.ArtifactManifestDigest {
		return nil, fmt.Errorf("release artifact digest does not match candidate %q", req.CandidateID)
	}
	return record, nil
}

func (o *Orchestrator) validateReleaseDestination(ctx context.Context, req releases.DeployRequest) error {
	if strings.TrimSpace(req.DestinationRevisionID) == "" {
		return fmt.Errorf("destination identity is required for release execution")
	}
	record, err := o.releaseIdentity.GetDestinationRevision(ctx, req.DestinationRevisionID)
	if err != nil {
		return fmt.Errorf("load release destination %q: %w", req.DestinationRevisionID, err)
	}
	if record == nil {
		return fmt.Errorf("release destination %q is not registered", req.DestinationRevisionID)
	}
	if record.ID != req.DestinationRevisionID {
		return fmt.Errorf("stored release destination identity %q does not match requested identity %q", record.ID, req.DestinationRevisionID)
	}
	channel := effectiveChannel(req.Channel, "")
	if strings.TrimSpace(record.Revision.Channel) != channel {
		return fmt.Errorf("release destination %q is bound to channel %q, not %q", req.DestinationRevisionID, record.Revision.Channel, channel)
	}
	return nil
}

func candidateArtifactDigests(record *releases.CandidateRecord) map[string]string {
	if record == nil {
		return nil
	}
	digests := make(map[string]string, len(record.Candidate.Artifacts))
	for _, artifact := range record.Candidate.Artifacts {
		digests[artifact.Target.ID] = artifact.Digest
	}
	return digests
}

func releaseStageFailed(ds *deployState) bool {
	if ds == nil || ds.response == nil {
		return true
	}
	if ds.response.Status == "failed" || ds.response.Status == "blocked" {
		return true
	}
	for _, step := range ds.response.Steps {
		if step.Status == "failed" {
			return true
		}
	}
	return false
}

// deployCheckCloudHealth asks scenario-to-cloud whether the exact cloud
// deployment this release targets is healthy, current and running the
// expected bundle. The deployment is identified by the receipt id when the
// cloud step ran, otherwise by the manifest's scenario + domain/environment
// selector; never by a product-specific slug. A failed gate fails the step
// and marks the release failed, but does not panic.
func (o *Orchestrator) deployCheckCloudHealth(ds *deployState) {
	const stepName = "Check cloud deployment health"
	request := DeploymentHealthRequest{
		Selector:              cloudManifestSelector(ds.req.CloudManifest),
		ExpectedReleaseDigest: ds.req.CloudBundleSHA256,
	}
	if ds.response.CloudReceipt != nil {
		request.DeploymentID = ds.response.CloudReceipt.DeploymentID
		if request.ExpectedReleaseDigest == "" {
			request.ExpectedReleaseDigest = ds.response.CloudReceipt.BundleSHA256
		}
	}
	if request.DeploymentID == "" && request.Selector.IsZero() {
		step := o.startStep(stepName)
		step.Status = "skipped"
		step.Message = "release has no cloud deployment identity; cloud health not asserted"
		ds.response.Steps = append(ds.response.Steps, step)
		return
	}
	if o.cloudClient == nil {
		if ds.req.ReleaseID != "" {
			step := o.startStep(stepName)
			o.failStep(&step, "cloud-health client is not configured for commercial publication")
			ds.response.Steps = append(ds.response.Steps, step)
			o.markReleaseFailed(ds, releases.StatusFailed)
		}
		return
	}
	step := o.startStep(stepName)
	result, err := o.cloudClient.CheckDeploymentHealth(ds.ctx, request)
	if err != nil {
		o.failStep(&step, fmt.Sprintf("cloud-health probe failed: %v", err))
		ds.response.Steps = append(ds.response.Steps, step)
		o.markReleaseFailed(ds, releases.StatusFailed)
		return
	}
	if result == nil {
		o.failStep(&step, "cloud-health probe returned no response")
		ds.response.Steps = append(ds.response.Steps, step)
		o.markReleaseFailed(ds, releases.StatusFailed)
		return
	}
	if !result.Healthy {
		o.failStep(&step, "cloud deployment not proven healthy: "+result.Explain())
		ds.response.Steps = append(ds.response.Steps, step)
		o.markReleaseFailed(ds, releases.StatusFailed)
		return
	}
	message := fmt.Sprintf("cloud deployment %s healthy and current (observed %s", result.DeploymentID, result.ObservedAt.UTC().Format(time.RFC3339))
	if result.ObservedReleaseDigest != "" {
		message += ", release " + result.ObservedReleaseDigest
	}
	o.successStep(&step, message+")")
	ds.response.Steps = append(ds.response.Steps, step)
}

// deployCheckLPBSReadiness calls LPBS's deploy-readiness endpoint to confirm
// the app registry, storage, and remote profile are configured.
func (o *Orchestrator) deployCheckLPBSReadiness(ds *deployState) {
	if cloudScenario := cloudManifestScenario(ds.req.CloudManifest); cloudScenario != "" && cloudScenario != "landing-page-business-suite" {
		step := o.startStep("Check LPBS upload readiness")
		step.Status = "skipped"
		step.Message = "LPBS upload readiness does not apply to cloud scenario " + cloudScenario
		ds.response.Steps = append(ds.response.Steps, step)
		return
	}
	if o.lpbsClient == nil || o.lpbsConfigRepo == nil {
		if ds.req.ReleaseID != "" {
			step := o.startStep("Check LPBS upload readiness")
			o.failStep(&step, "LPBS readiness client is not configured for commercial publication")
			ds.response.Steps = append(ds.response.Steps, step)
			o.markReleaseFailed(ds, releases.StatusFailed)
		}
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
		if ds.req.ReleaseID != "" {
			o.failStep(&step, "LPBS release configuration is missing app_key")
			ds.response.Steps = append(ds.response.Steps, step)
			o.markReleaseFailed(ds, releases.StatusFailed)
			return
		}
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
	if result == nil {
		o.failStep(&step, "readiness call returned no response")
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

// deployCloud delegates the remote lifecycle to scenario-to-cloud and only
// advances after its durable receipt validates. DM never reports the create
// or execute response as a deployment effect.
func (o *Orchestrator) deployCloud(ds *deployState) int {
	if len(strings.TrimSpace(string(ds.req.CloudManifest))) == 0 {
		return 0
	}
	step := o.startStep("Deploy cloud target")
	if strings.TrimSpace(ds.req.ReleaseID) == "" || strings.TrimSpace(ds.req.CandidateID) == "" || strings.TrimSpace(ds.req.DestinationRevisionID) == "" {
		o.failStep(&step, "cloud deployment requires a governed release, candidate, and destination identity")
		ds.response.Steps = append(ds.response.Steps, step)
		ds.response.Status = "blocked"
		return http.StatusPreconditionFailed
	}
	if o.releasesRepo == nil {
		o.failStep(&step, "cloud deployment requires a durable release repository")
		ds.response.Steps = append(ds.response.Steps, step)
		ds.response.Status = "failed"
		return http.StatusServiceUnavailable
	}
	if _, ok := o.releasesRepo.(releases.PublicationReceiptRepository); !ok {
		o.failStep(&step, "cloud deployment requires a durable publication-receipt repository")
		ds.response.Steps = append(ds.response.Steps, step)
		ds.response.Status = "failed"
		return http.StatusServiceUnavailable
	}
	client, ok := o.cloudClient.(CloudDeploymentClient)
	if !ok {
		o.failStep(&step, "cloud deployment client is not configured")
		ds.response.Steps = append(ds.response.Steps, step)
		ds.response.Status = "failed"
		o.markReleaseFailed(ds, releases.StatusFailed)
		return http.StatusServiceUnavailable
	}
	if strings.TrimSpace(ds.req.ReleaseID) != "" {
		if ds.req.AuthorizationCheck == nil {
			o.failStep(&step, "canonical release authorization is required before cloud deployment")
			ds.response.Steps = append(ds.response.Steps, step)
			ds.response.Status = "blocked"
			return http.StatusPreconditionFailed
		}
		if err := ds.req.AuthorizationCheck(ds.ctx); err != nil {
			o.failStep(&step, fmt.Sprintf("current release authorization failed before cloud deployment: %v", err))
			ds.response.Steps = append(ds.response.Steps, step)
			ds.response.Status = "blocked"
			return http.StatusPreconditionFailed
		}
	}
	receipt, err := client.DeployCloud(ds.ctx, &CloudDeploymentRequest{
		Name: ds.req.CloudDeploymentName, Manifest: ds.req.CloudManifest,
		BundlePath: ds.req.CloudBundlePath, BundleSHA256: ds.req.CloudBundleSHA256,
		BundleSizeBytes: ds.req.CloudBundleSizeBytes, RunPreflight: ds.req.CloudRunPreflight,
		Review: cloudReviewIdentity(ds.req),
	})
	if err != nil {
		o.failStep(&step, err.Error())
		ds.response.Steps = append(ds.response.Steps, step)
		ds.response.Status = "failed"
		o.markReleaseFailed(ds, releases.StatusFailed)
		return http.StatusBadGateway
	}
	if receipt == nil {
		o.failStep(&step, "cloud deployment client returned no receipt")
		ds.response.Steps = append(ds.response.Steps, step)
		ds.response.Status = "failed"
		o.markReleaseFailed(ds, releases.StatusFailed)
		return http.StatusBadGateway
	}
	if err := validateCloudReceipt(*receipt, receipt.DeploymentID, &CloudDeploymentRequest{
		Manifest: ds.req.CloudManifest, BundleSHA256: ds.req.CloudBundleSHA256,
	}); err != nil {
		o.failStep(&step, err.Error())
		ds.response.Steps = append(ds.response.Steps, step)
		ds.response.Status = "failed"
		o.markReleaseFailed(ds, releases.StatusFailed)
		return http.StatusBadGateway
	}
	ds.response.CloudReceipt = receipt
	// Governance gate: the owner's per-cell evidence must be complete for the
	// exact release and target the receipt names. A green build, a healthy
	// observation or the receipt alone never promotes (GOV-01, GOV-02).
	if err := checkCloudEvidence(receipt); err != nil {
		o.failStep(&step, "cloud evidence refuses promotion: "+err.Error())
		ds.response.Steps = append(ds.response.Steps, step)
		ds.response.Status = "failed"
		o.markReleaseFailed(ds, releases.StatusFailed)
		return http.StatusPreconditionFailed
	}
	markAmbiguous := func(message string, status int) int {
		o.failStep(&step, message)
		ds.response.Steps = append(ds.response.Steps, step)
		ds.response.Status = "ambiguous"
		o.markReleaseFailed(ds, releases.StatusAmbiguous)
		return status
	}
	if err := o.releasesRepo.SetDeploymentID(ds.ctx, ds.req.ReleaseID, receipt.DeploymentID); err != nil {
		o.failStep(&step, fmt.Sprintf("persist cloud deployment identity: %v", err))
		ds.response.Steps = append(ds.response.Steps, step)
		ds.response.Status = "ambiguous"
		o.markReleaseFailed(ds, releases.StatusAmbiguous)
		return http.StatusBadGateway
	}
	receiptRepo := o.releasesRepo.(releases.PublicationReceiptRepository)
	if err := receiptRepo.RecordPublicationReceipt(ds.ctx, ds.req.ReleaseID, cloudPublicationReceipt(ds.req, receipt)); err != nil {
		return markAmbiguous(fmt.Sprintf("persist cloud deployment receipt: %v", err), http.StatusBadGateway)
	}
	o.successStep(&step, "cloud deployment receipt verified for "+receipt.DestinationID)
	ds.response.Steps = append(ds.response.Steps, step)
	return 0
}

// cloudReviewIdentity projects the release request onto the cloud owner's
// review identity. The target set is left to the owner (its target key);
// the policy version is the active readiness checklist version.
func cloudReviewIdentity(req DeployDesktopRequest) *CloudReviewIdentity {
	if strings.TrimSpace(req.ProfileID) == "" || strings.TrimSpace(req.GitCommitHash) == "" {
		return nil
	}
	return &CloudReviewIdentity{
		ReviewKey: req.ReadinessReviewKey, ProfileID: req.ProfileID, CandidateCommit: req.GitCommitHash,
		ArtifactDigest: req.ArtifactDigest, Channel: req.Channel, PolicyVersion: readiness.ChecklistVersion,
		CandidateID: req.CandidateID, DestinationRevisionID: req.DestinationRevisionID, AuthorizationEpoch: req.AuthorizationEpoch,
	}
}

// cloudPublicationReceipt records the ACTUAL activated release from the
// owner's publication (target receipt) when one exists, otherwise the
// receipt's own release binding, plus the predecessor the owner resolved
// from its published history and the evidence drill-down reference.
func cloudPublicationReceipt(req DeployDesktopRequest, receipt *CloudDeploymentReceipt) releases.PublicationReceipt {
	artifact := receipt.ReleaseDigest
	if artifact == "" {
		artifact = receipt.BundleSHA256
	}
	predecessor := ""
	reviewKey := req.ReadinessReviewKey
	if pub := receipt.Publication; pub != nil {
		if pub.ActivatedReleaseDigest != "" && pub.State == "published" {
			artifact = pub.ActivatedReleaseDigest
		}
		predecessor = pub.PredecessorReleaseDigest
		if pub.ReviewRef != "" {
			reviewKey = pub.ReviewRef
		}
	}
	evidenceRef := ""
	if receipt.Evidence != nil {
		evidenceRef = "scenario-to-cloud:/api/v1/deployments/" + receipt.DeploymentID + "/evidence?release_digest=" + receipt.Evidence.ReleaseDigest
	}
	return releases.PublicationReceipt{
		CandidateID: req.CandidateID, DestinationRevisionID: req.DestinationRevisionID,
		TargetID: "cloud:" + receipt.ScenarioID, ArtifactDigest: artifact,
		DestinationObject: receipt.DestinationID, Producer: receipt.ProducerRef,
		ExternalReceipt: receipt.ExternalReceipt, Outcome: releases.ReceiptVerified,
		ObservedAt: receipt.ObservedAt, ReviewKey: reviewKey, PredecessorArtifactDigest: predecessor, EvidenceRef: evidenceRef,
	}
}

func cloudManifestScenario(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var envelope struct {
		Scenario struct {
			ID string `json:"id"`
		} `json:"scenario"`
	}
	if json.Unmarshal(raw, &envelope) != nil {
		return ""
	}
	return strings.TrimSpace(envelope.Scenario.ID)
}

// deployVerifyUpdateEndpoints calls LPBS's verify endpoint per platform
// after publish. On any mismatch the release is marked verify_failed.
func (o *Orchestrator) deployVerifyUpdateEndpoints(ds *deployState) {
	if ds.req.ReleaseID == "" {
		return
	}
	step := o.startStep("Verify update endpoints")
	if o.lpbsClient == nil || o.lpbsConfigRepo == nil {
		o.failStep(&step, "LPBS verification is not configured for commercial publication")
		ds.response.Status = "verify_failed"
		ds.response.Steps = append(ds.response.Steps, step)
		o.persistReleaseStatus(ds, releases.StatusVerifyFailed)
		return
	}
	cfg, err := o.lpbsConfigRepo.Get(ds.ctx, ds.req.ProfileID)
	if err != nil {
		o.failStep(&step, fmt.Sprintf("load LPBS verification config: %v", err))
		ds.response.Status = "verify_failed"
		ds.response.Steps = append(ds.response.Steps, step)
		o.persistReleaseStatus(ds, releases.StatusVerifyFailed)
		return
	}
	if cfg == nil || strings.TrimSpace(cfg.LPBSAppKey) == "" {
		o.failStep(&step, "LPBS verification configuration is missing app_key")
		ds.response.Status = "verify_failed"
		ds.response.Steps = append(ds.response.Steps, step)
		o.persistReleaseStatus(ds, releases.StatusVerifyFailed)
		return
	}
	channel := effectiveChannel(ds.req.Channel, cfg.DefaultChannel)
	version := ds.req.ReleaseVersion

	var evidence []releases.VerificationItem
	allMatch := exactPlatformSet(ds.req.Platforms, ds.response.PublishedVersions)
	if !allMatch {
		o.failStep(&step, fmt.Sprintf("published target set does not match requested target set (%s)", strings.Join(ds.req.Platforms, ",")))
		ds.response.Status = "verify_failed"
		ds.response.Steps = append(ds.response.Steps, step)
		if o.releasesRepo != nil {
			o.persistReleaseStatus(ds, releases.StatusVerifyFailed)
		}
		return
	}
	for _, pv := range ds.response.PublishedVersions {
		platform := pv.Platform
		expected := version
		if expected == "" {
			expected = pv.Version
		}
		item := releases.VerificationItem{
			Platform:        platform,
			Channel:         channel,
			ExpectedVersion: expected,
			CheckedAt:       time.Now().UTC(),
		}
		if pv.SHA512 == "" {
			item.Error = "published artifact has no sha512 identity"
			allMatch = false
		} else {
			result, verr := o.lpbsClient.Verify(ds.ctx, &LPBSVerifyRequest{
				AppKey:          cfg.LPBSAppKey,
				Channel:         channel,
				Platform:        platform,
				ExpectedVersion: expected,
				ExpectedSHA512:  pv.SHA512,
				Deep:            true,
			})
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
				if !result.Match || !result.SHA512Match || result.ObservedVersion != expected {
					allMatch = false
				}
			} else {
				item.Error = "verification returned no result"
				allMatch = false
			}
		}
		evidence = append(evidence, item)
	}

	if o.releasesRepo != nil {
		if serr := o.releasesRepo.SetVerificationEvidence(ds.ctx, ds.req.ReleaseID, evidence); serr != nil {
			o.failStep(&step, fmt.Sprintf("persist verification evidence: %v", serr))
			ds.response.Status = "verify_failed"
			o.persistReleaseStatus(ds, releases.StatusVerifyFailed)
			ds.response.Steps = append(ds.response.Steps, step)
			return
		}
		if allMatch {
			if err := o.releasesRepo.UpdateStatus(ds.ctx, ds.req.ReleaseID, releases.StatusPublished); err != nil {
				o.failStep(&step, fmt.Sprintf("persist published status: %v", err))
				allMatch = false
				ds.response.Status = "ambiguous"
			} else if err := o.releasesRepo.MarkSuperseded(ds.ctx, ds.req.ProfileID, channel, ds.req.ReleaseID); err != nil {
				o.failStep(&step, fmt.Sprintf("persist superseded release state: %v", err))
				allMatch = false
				ds.response.Status = "ambiguous"
				o.persistReleaseStatus(ds, releases.StatusAmbiguous)
			}
		} else {
			if err := o.releasesRepo.UpdateStatus(ds.ctx, ds.req.ReleaseID, releases.StatusVerifyFailed); err != nil {
				o.failStep(&step, fmt.Sprintf("persist verification failure status: %v", err))
				ds.response.Status = "ambiguous"
				o.persistReleaseStatus(ds, releases.StatusAmbiguous)
			}
		}
	}

	if allMatch {
		o.successStep(&step, fmt.Sprintf("verified %d platform(s) against channel %s", len(evidence), channel))
	} else {
		o.failStep(&step, fmt.Sprintf("verification failed for one or more platforms on channel %s", channel))
		if ds.response.Status != "ambiguous" {
			ds.response.Status = "verify_failed"
		}
	}
	if allMatch {
		ds.response.Status = "published"
	}
	ds.response.Steps = append(ds.response.Steps, step)
}

func exactPlatformSet(requested []string, published []PublishedVersion) bool {
	if len(requested) == 0 || len(requested) != len(published) {
		return false
	}
	want := make(map[string]struct{}, len(requested))
	for _, platform := range requested {
		platform = strings.TrimSpace(platform)
		if platform == "" {
			return false
		}
		if _, exists := want[platform]; exists {
			return false
		}
		want[platform] = struct{}{}
	}
	for _, version := range published {
		if _, exists := want[version.Platform]; !exists {
			return false
		}
		delete(want, version.Platform)
	}
	return len(want) == 0
}

// markReleaseFailed transitions the release record to failed, if wired.
func (o *Orchestrator) markReleaseFailed(ds *deployState, status string) {
	if o.releasesRepo == nil || ds.req.ReleaseID == "" {
		return
	}
	o.persistReleaseStatus(ds, status)
}

func (o *Orchestrator) persistReleaseStatus(ds *deployState, status string) {
	if o.releasesRepo == nil || ds == nil || ds.req.ReleaseID == "" {
		return
	}
	if err := o.releasesRepo.UpdateStatus(ds.ctx, ds.req.ReleaseID, status); err != nil {
		o.log("error", map[string]interface{}{
			"msg":        "persist release status failed",
			"release_id": ds.req.ReleaseID,
			"status":     status,
			"error":      err.Error(),
		})
		if ds.response != nil && ds.response.Status != "published" {
			ds.response.Status = "ambiguous"
		}
	}
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

// loadLPBSConfigForPublish loads the exact LPBS coordinates that
// publishToLPBS must bind before it launches an owner effect.
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
	if ds.response.CloudReceipt != nil {
		result.CloudDeploymentID = ds.response.CloudReceipt.DeploymentID
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
