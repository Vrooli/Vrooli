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
	"deployment-manager/releases"
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
	// ReleaseID, when set, wires this deployment into an existing release
	// record allocated by POST /api/v1/profiles/{id}/releases/start.
	ReleaseID string `json:"release_id,omitempty"`
	// Channel is the release channel (stable, beta, ...) forwarded to S2D
	// and mapped to LPBS variant_key on apply.
	Channel string `json:"channel,omitempty"`
	// ReleaseVersion mirrors scenario-to-desktop's Config.Version and flows
	// to LPBS so the verify endpoint has an expected version to match.
	ReleaseVersion string `json:"release_version,omitempty"`
	// CandidateID binds the deployment to the immutable build candidate.
	CandidateID string `json:"candidate_id,omitempty"`
	// ArtifactDigest binds the deployment to the candidate's canonical artifact
	// manifest digest. It must survive every owner handoff.
	ArtifactDigest string `json:"artifact_digest,omitempty"`
	// ExpectedArtifactDigests binds each exact target to its finalized bytes.
	ExpectedArtifactDigests map[string]string `json:"expected_artifact_digests,omitempty"`
	// DestinationRevisionID binds publication to the exact destination config.
	DestinationRevisionID string `json:"destination_revision_id,omitempty"`
	// AuthorizationEpoch prevents an old approval from authorizing a new release.
	AuthorizationEpoch uint64 `json:"authorization_epoch,omitempty"`
	// IdempotencyKey makes repeated lifecycle requests resolve to one operation.
	IdempotencyKey     string                      `json:"idempotency_key,omitempty"`
	AuthorizationCheck func(context.Context) error `json:"-"`
	// ReadinessReviewKey is the approved review the release started under;
	// the cloud owner re-checks it before any publication effect.
	ReadinessReviewKey string `json:"readiness_review_key,omitempty"`
	// CloudManifest opts this governed release into scenario-to-cloud's VPS
	// lifecycle. The cloud service remains the owner of secrets and remote
	// effects; DM records the receipt it returns.
	CloudManifest        json.RawMessage `json:"cloud_manifest,omitempty"`
	CloudDeploymentName  string          `json:"cloud_deployment_name,omitempty"`
	CloudBundlePath      string          `json:"cloud_bundle_path,omitempty"`
	CloudBundleSHA256    string          `json:"cloud_bundle_sha256,omitempty"`
	CloudBundleSizeBytes int64           `json:"cloud_bundle_size_bytes,omitempty"`
	CloudRunPreflight    bool            `json:"cloud_run_preflight,omitempty"`
}

// Recover routes an exact release recovery request to the owning cloud
// service. Deployment-manager keeps the identity and authorization checks;
// the owner returns the effect receipt.
func (o *Orchestrator) Recover(ctx context.Context, request *releases.RecoveryRequest, deploymentID, expectedBundleSHA string) (*releases.RecoveryReceipt, error) {
	client, ok := o.cloudClient.(CloudRecoveryClient)
	if request == nil {
		return nil, fmt.Errorf("recovery request is required")
	}
	if o.releaseIdentity != nil && request.DestinationRevisionID != "" {
		record, err := o.releaseIdentity.GetDestinationRevision(ctx, request.DestinationRevisionID)
		if err != nil || record == nil {
			return nil, fmt.Errorf("load recovery destination %q: %w", request.DestinationRevisionID, err)
		}
		if strings.EqualFold(record.Revision.Kind, "lpbs") {
			lpbsRecovery, supported := o.lpbsClient.(LPBSRecoveryClient)
			if !supported {
				return nil, fmt.Errorf("lpbs recovery owner is unavailable")
			}
			if o.lpbsConfigRepo == nil {
				return nil, fmt.Errorf("lpbs release configuration is unavailable")
			}
			config, err := o.lpbsConfigRepo.Get(ctx, request.ProfileID)
			if err != nil || config == nil || strings.TrimSpace(config.LPBSAppKey) == "" {
				return nil, fmt.Errorf("lpbs release configuration is unavailable: %w", err)
			}
			revision, err := parseChannelRevision(record.Revision.ExpectedChannelRevision)
			if err != nil {
				return nil, fmt.Errorf("lpbs destination revision has no numeric expected channel revision: %w", err)
			}
			ownerRequest := &LPBSRecoveryRequest{AppKey: config.LPBSAppKey, VariantKey: lpbsVariantKey(request.Channel), ExpectedRevision: revision, ExpectedPredecessor: request.ExpectedPredecessor, Action: request.Action, DataCompatibility: request.DataCompatibility, ArtifactIDs: request.RepairArtifactIDs, CandidateID: request.CandidateID, DestinationRevisionID: request.DestinationRevisionID, Halted: true, Reason: request.Confirmation, Confirmation: request.Confirmation, DryRun: request.DryRun}
			var ownerReceipt *LPBSRecoveryReceipt
			if request.Action == "halt" {
				ownerReceipt, err = lpbsRecovery.HaltChannel(ctx, ownerRequest)
			} else {
				recoveryOwner, supported := o.lpbsClient.(LPBSChannelRecoveryClient)
				if !supported {
					return nil, fmt.Errorf("lpbs recovery action %q is unsupported by the owner", request.Action)
				}
				ownerReceipt, err = recoveryOwner.RecoverChannel(ctx, ownerRequest)
			}
			if err != nil {
				return nil, err
			}
			if ownerReceipt == nil || (request.Action != "halt" && ownerReceipt.Action != request.Action) || ownerReceipt.AppKey != config.LPBSAppKey || ownerReceipt.VariantKey != lpbsVariantKey(request.Channel) || strings.TrimSpace(ownerReceipt.ExternalReceipt) == "" || ownerReceipt.ObservedAt.IsZero() {
				return nil, fmt.Errorf("lpbs recovery owner returned an incomplete or mismatched receipt")
			}
			if request.Action != "halt" && (ownerReceipt.CandidateID != request.CandidateID || ownerReceipt.DestinationRevisionID != request.DestinationRevisionID) {
				return nil, fmt.Errorf("lpbs recovery receipt does not preserve candidate and destination identity")
			}
			return &releases.RecoveryReceipt{ReceiptID: fmt.Sprintf("%s:lpbs:%s", request.ReleaseID, ownerReceipt.ExternalReceipt), ReleaseID: request.ReleaseID, CandidateID: request.CandidateID, DestinationRevisionID: request.DestinationRevisionID, DeploymentID: record.Revision.DestinationID, Action: request.Action, Outcome: ownerReceipt.Outcome, Health: ownerReceipt.Health, ExternalReceipt: ownerReceipt.ExternalReceipt, ObservedAt: ownerReceipt.ObservedAt, DryRun: ownerReceipt.DryRun}, nil
		}
	}
	if !ok {
		return nil, fmt.Errorf("cloud recovery owner is unavailable")
	}
	idempotencyKey := strings.TrimSpace(request.IdempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = strings.Join([]string{request.ReleaseID, request.Action, expectedBundleSHA, request.RepairBundleSHA256}, ":")
	}
	ownerReceipt, err := client.RecoverCloud(ctx, &CloudRecoveryRequest{
		DeploymentID: deploymentID, Action: request.Action, ExpectedBundleSHA: expectedBundleSHA,
		RepairBundleSHA: request.RepairBundleSHA256, DataCompatibility: request.DataCompatibility,
		IdempotencyKey: idempotencyKey, Confirmation: request.Confirmation, DryRun: request.DryRun,
		ReviewKey: request.ReviewKey, PreviewRef: request.PreviewRef,
	})
	if err != nil {
		return nil, err
	}
	if ownerReceipt == nil {
		return nil, fmt.Errorf("cloud recovery owner returned no receipt")
	}
	if request.Action != "halt" && strings.TrimSpace(ownerReceipt.BundleSHA256) == "" {
		return nil, fmt.Errorf("cloud recovery owner returned no repaired bundle identity")
	}
	receipt := &releases.RecoveryReceipt{
		ReceiptID: fmt.Sprintf("%s:%s", request.ReleaseID, ownerReceipt.ExternalReceipt), ReleaseID: request.ReleaseID,
		CandidateID: request.CandidateID, DestinationRevisionID: request.DestinationRevisionID, DeploymentID: ownerReceipt.DeploymentID,
		Action: ownerReceipt.Action, Outcome: ownerReceipt.Outcome, Health: ownerReceipt.Health,
		BundleSHA256: ownerReceipt.BundleSHA256, ExternalReceipt: ownerReceipt.ExternalReceipt, ObservedAt: ownerReceipt.ObservedAt, DryRun: ownerReceipt.DryRun,
		PreviewRef: ownerReceipt.PreviewRef,
	}
	return receipt, nil
}

// RecoveryControls reports the controls supported by the owner bound to an
// exact destination revision. It is deliberately capability based: a route
// must not advertise an action merely because the request type accepts it.
func (o *Orchestrator) RecoveryControls(ctx context.Context, destinationRevisionID string) ([]string, error) {
	if o == nil || o.releaseIdentity == nil {
		return nil, fmt.Errorf("release identity repository is unavailable")
	}
	record, err := o.releaseIdentity.GetDestinationRevision(ctx, strings.TrimSpace(destinationRevisionID))
	if err != nil || record == nil {
		if err != nil {
			return nil, fmt.Errorf("load recovery destination %q: %w", destinationRevisionID, err)
		}
		return nil, fmt.Errorf("load recovery destination %q: not found", destinationRevisionID)
	}
	if strings.EqualFold(record.Revision.Kind, "lpbs") {
		controls := make([]string, 0, 4)
		if _, supported := o.lpbsClient.(LPBSRecoveryClient); supported {
			controls = append(controls, "halt")
		}
		if _, supported := o.lpbsClient.(LPBSChannelRecoveryClient); supported {
			controls = append(controls, "withdraw", "rollback", "forward_repair")
		}
		return controls, nil
	}
	if _, supported := o.cloudClient.(CloudRecoveryClient); supported {
		return []string{"halt", "rollback", "forward_repair"}, nil
	}
	return nil, nil
}

// ObserveRelease obtains current health from the owner bound to a cloud
// destination. LPBS keeps its dedicated byte-and-version verifier; cloud
// reconciliation must use the cloud health contract instead of pretending
// that an LPBS check applies to every destination kind.
func (o *Orchestrator) ObserveRelease(ctx context.Context, release *releases.Release) (*releases.OwnerObservation, error) {
	if o == nil || release == nil {
		return nil, fmt.Errorf("release observation requires a release")
	}
	if o.releaseIdentity == nil {
		return nil, fmt.Errorf("release identity repository is unavailable")
	}
	record, err := o.releaseIdentity.GetDestinationRevision(ctx, strings.TrimSpace(release.DestinationRevisionID))
	if err != nil || record == nil {
		if err != nil {
			return nil, fmt.Errorf("load release destination %q: %w", release.DestinationRevisionID, err)
		}
		return nil, fmt.Errorf("load release destination %q: not found", release.DestinationRevisionID)
	}
	if strings.EqualFold(record.Revision.Kind, "lpbs") {
		return nil, fmt.Errorf("LPBS releases require the LPBS verifier")
	}
	client, ok := o.cloudClient.(CloudHealthClient)
	if !ok || strings.TrimSpace(release.DeploymentID) == "" {
		return nil, fmt.Errorf("cloud health owner or deployment identity is unavailable")
	}
	expected := expectedCloudReleaseDigest(release)
	result, err := client.CheckDeploymentHealth(ctx, DeploymentHealthRequest{DeploymentID: release.DeploymentID, ExpectedReleaseDigest: expected})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, fmt.Errorf("cloud health owner returned no observation")
	}
	if strings.TrimSpace(result.DeploymentID) != strings.TrimSpace(release.DeploymentID) {
		return nil, fmt.Errorf("cloud health owner observed deployment %q, expected %q", result.DeploymentID, release.DeploymentID)
	}
	if result.ObservedAt.IsZero() {
		return nil, fmt.Errorf("cloud health owner observation has no observed_at")
	}
	return &releases.OwnerObservation{
		DeploymentID:          result.DeploymentID,
		TargetID:              "cloud:" + result.DeploymentID,
		ExpectedReleaseDigest: expected,
		ObservedReleaseDigest: result.ObservedReleaseDigest,
		Healthy:               result.Healthy,
		ObservedAt:            result.ObservedAt,
		Detail:                result.Explain(),
	}, nil
}

func expectedCloudReleaseDigest(release *releases.Release) string {
	if release == nil {
		return ""
	}
	for i := len(release.PublicationReceipts) - 1; i >= 0; i-- {
		receipt := release.PublicationReceipts[i]
		if (strings.HasPrefix(receipt.TargetID, "cloud:") || receipt.Producer == cloudReceiptProducerRef) && strings.TrimSpace(receipt.ArtifactDigest) != "" {
			return receipt.ArtifactDigest
		}
	}
	return release.ArtifactDigest
}

// DeployDesktopResponse is the response from orchestrated deployment.
type DeployDesktopResponse struct {
	Status            string                  `json:"status"`
	ProfileID         string                  `json:"profile_id"`
	Scenario          string                  `json:"scenario"`
	Steps             []OrchestrationStep     `json:"steps"`
	ManifestPath      string                  `json:"manifest_path,omitempty"`
	BuildResults      *build.BuildAllResult   `json:"build_results,omitempty"`
	DesktopBuildID    string                  `json:"desktop_build_id,omitempty"`
	DesktopPath       string                  `json:"desktop_path,omitempty"`
	InstallerBuildID  string                  `json:"installer_build_id,omitempty"`
	Installers        map[string]string       `json:"installers,omitempty"`
	PublishedVersions []PublishedVersion      `json:"published_versions,omitempty"`
	CloudReceipt      *CloudDeploymentReceipt `json:"cloud_receipt,omitempty"`
	Duration          string                  `json:"duration,omitempty"`
	NextSteps         []string                `json:"next_steps,omitempty"`
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
	releasesRepo          releases.Repository
	releaseIdentity       releases.IdentityRepository
	lpbsConfigRepo        profiles.LPBSReleaseConfigRepository
	publishPipelineRunner publishPipelineRunner
	cloudClient           CloudHealthClient
	lpbsClient            LPBSReleaseClient
	commandRunner         commandRunner
	vrooli                string
	log                   func(string, map[string]interface{})
}

// NewOrchestrator creates a new deployment orchestrator.
func NewOrchestrator(profileRepo profiles.Repository, log func(string, map[string]interface{})) *Orchestrator {
	return NewOrchestratorWithApprovals(profileRepo, nil, log)
}

// NewOrchestratorWithApprovals creates a new deployment orchestrator with approval gating.
func NewOrchestratorWithApprovals(profileRepo profiles.Repository, approvalsRepo ApprovalsRepository, log func(string, map[string]interface{})) *Orchestrator {
	return NewOrchestratorFull(profileRepo, approvalsRepo, nil, nil, nil, nil, nil, log)
}

// NewOrchestratorFull creates a new deployment orchestrator with all optional repositories.
func NewOrchestratorFull(
	profileRepo profiles.Repository,
	approvalsRepo ApprovalsRepository,
	publishedVersionsRepo PublishedVersionsRepository,
	releasesRepo releases.Repository,
	lpbsConfigRepo profiles.LPBSReleaseConfigRepository,
	cloudClient CloudHealthClient,
	lpbsClient LPBSReleaseClient,
	log func(string, map[string]interface{}),
) *Orchestrator {
	vrooli := resolveRepoRoot()
	return &Orchestrator{
		profileRepo:           profileRepo,
		approvalsRepo:         approvalsRepo,
		publishedVersionsRepo: publishedVersionsRepo,
		releasesRepo:          releasesRepo,
		releaseIdentity:       identityRepository(releasesRepo),
		lpbsConfigRepo:        lpbsConfigRepo,
		cloudClient:           cloudClient,
		lpbsClient:            lpbsClient,
		commandRunner:         processCommandRunner{},
		vrooli:                vrooli,
		log:                   log,
	}
}

func identityRepository(repository releases.Repository) releases.IdentityRepository {
	identity, _ := repository.(releases.IdentityRepository)
	return identity
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
	if strings.TrimSpace(req.GitCommitHash) == "" {
		http.Error(w, `{"error":"git_commit_hash is required","reason":"commit_identifier_required"}`, http.StatusBadRequest)
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
	o.writeJSON(w, deploymentResponseHTTPStatus(ds.response), ds.response)
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
	// Release-bound executions carry the canonical readiness authorization
	// check, which is revalidated immediately before publication. Re-running
	// the legacy commit/platform approval projection here would create a second
	// decision boundary and could block an otherwise valid exact release.
	if o.approvalsRepo != nil && ds.req.AuthorizationCheck == nil {
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
				o.failStep(&step, fmt.Sprintf("failed to apply signing config: %v", err))
				o.log("warn", map[string]interface{}{
					"msg":      "signing config application failed",
					"scenario": ds.profile.Scenario,
					"error":    err.Error(),
				})
				ds.response.Steps = append(ds.response.Steps, step)
				ds.response.Status = "failed"
				return http.StatusBadGateway
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
		if ds.req.ReleaseID != "" {
			o.failStep(&step, "commercial release signing readiness failed: "+strings.Join(signingWarnings, "; "))
			ds.response.Steps = append(ds.response.Steps, step)
			ds.response.Status = "failed"
			return http.StatusPreconditionFailed
		}
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

	// Apply swaps from profile. A commercial release cannot continue with an
	// incomplete profile because the resulting artifact would no longer be
	// attributable to the reviewed configuration.
	profileSwaps, err := o.profileRepo.GetSwaps(ds.ctx, ds.req.ProfileID)
	if err != nil {
		o.failStep(&step, fmt.Sprintf("failed to load profile swaps: %v", err))
		ds.response.Steps = append(ds.response.Steps, step)
		ds.response.Status = "failed"
		return http.StatusBadGateway
	}
	for _, ps := range profileSwaps {
		manifest.Swaps = append(manifest.Swaps, bundles.ManifestSwap{
			Original:    ps.From,
			Replacement: ps.To,
			Reason:      ps.Reason,
			Limitations: ps.Limitations,
		})
	}

	// Populate missing asset metadata. A commercial release must fail closed if
	// the exact asset set cannot be enumerated or hashed.
	scenarioDir := filepath.Join(ds.scenarioBaseDir, ds.profile.Scenario)
	if err := populateAssetMetadata(manifest, scenarioDir); err != nil {
		if ds.req.ReleaseID != "" {
			o.failStep(&step, fmt.Sprintf("failed to prepare exact asset metadata: %v", err))
			ds.response.Steps = append(ds.response.Steps, step)
			ds.response.Status = "failed"
			return http.StatusBadGateway
		}
		step.Status = "warning"
		step.Message = fmt.Sprintf("assembled manifest with %d swaps (asset metadata partial: %v)", len(manifest.Swaps), err)
		o.log("warn", map[string]interface{}{
			"msg":      "asset metadata population incomplete",
			"scenario": ds.profile.Scenario,
			"error":    err.Error(),
		})
	}
	if step.Status == "running" {
		o.successStep(&step, fmt.Sprintf("assembled manifest with %d swaps", len(manifest.Swaps)))
	}
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
		manifestData, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			o.failStep(&step, fmt.Sprintf("failed to encode manifest: %v", err))
			ds.response.Steps = append(ds.response.Steps, step)
			ds.response.Status = "failed"
			return http.StatusInternalServerError
		}
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
	var buildErrors []string
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
			buildErrors = append(buildErrors, fmt.Sprintf("%s: %v", svc.ID, err))
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
			manifestData, err := json.MarshalIndent(ds.manifest, "", "  ")
			if err != nil {
				o.failStep(&step, fmt.Sprintf("failed to encode built manifest: %v", err))
			} else if err := os.WriteFile(ds.response.ManifestPath, manifestData, 0o644); err != nil {
				o.failStep(&step, fmt.Sprintf("failed to persist built manifest: %v", err))
			}
		}
		if step.Status != "failed" {
			o.successStep(&step, fmt.Sprintf("built %d service(s) for %d platform(s)", len(buildableServices), len(ds.buildPlatforms)))
		}
	} else {
		message := "some builds failed"
		if len(buildErrors) > 0 {
			message += ": " + strings.Join(buildErrors, "; ")
		}
		o.failStep(&step, message)
	}
	ds.response.Steps = append(ds.response.Steps, step)

	if !allSucceeded {
		ds.response.Status = "failed"
		return http.StatusBadGateway
	}
	return 0
}

// deployPackageAndInstall generates the desktop wrapper and builds installers.
func (o *Orchestrator) deployPackageAndInstall(ds *deployState) int {
	o.deployGenerateWrapper(ds)
	if releaseStageFailed(ds) {
		ds.response.Status = "failed"
		return http.StatusBadGateway
	}
	o.deployValidateRuntime(ds)
	if releaseStageFailed(ds) {
		ds.response.Status = "failed"
		return http.StatusBadGateway
	}
	o.deployCopyBinaries(ds)
	if releaseStageFailed(ds) {
		ds.response.Status = "failed"
		return http.StatusBadGateway
	}
	o.deployBuildInstallers(ds)
	if releaseStageFailed(ds) {
		ds.response.Status = "failed"
		return http.StatusBadGateway
	}
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
		o.failStep(&step, fmt.Sprintf("copied binaries with %d missing artifact(s): %s", len(missing), strings.Join(missing, ", ")))
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
	if !ds.req.DryRun && strings.TrimSpace(ds.req.ReleaseID) != "" && o.publishedVersionsRepo == nil {
		step := o.startStep("Publish to LPBS")
		o.failStep(&step, "commercial publication requires a published-version repository")
		ds.response.Steps = append(ds.response.Steps, step)
		ds.response.Status = "failed"
		return
	}
	if o.publishedVersionsRepo != nil && !ds.req.DryRun {
		step := o.startStep("Publish to LPBS")
		if ds.req.AuthorizationCheck == nil {
			o.failStep(&step, "canonical release authorization is required before publication")
			ds.response.Steps = append(ds.response.Steps, step)
			ds.response.Status = "blocked"
			return
		}
		if err := ds.req.AuthorizationCheck(ds.ctx); err != nil {
			o.failStep(&step, fmt.Sprintf("current release authorization failed before publication: %v", err))
			ds.response.Steps = append(ds.response.Steps, step)
			ds.response.Status = "blocked"
			return
		}
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
		if ds.response.Status != "ambiguous" {
			ds.response.Status = "failed"
		}
	}
}

func deploymentResponseHTTPStatus(response *DeployDesktopResponse) int {
	if response == nil {
		return http.StatusInternalServerError
	}
	switch response.Status {
	case "blocked":
		return http.StatusPreconditionFailed
	case "failed", "verify_failed":
		return http.StatusBadGateway
	case "ambiguous":
		return http.StatusConflict
	default:
		return http.StatusOK
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
		return fmt.Errorf("could not fetch scenario dependencies for blocker check: %w", err)
	}

	appliedSwaps, err := o.profileRepo.GetSwaps(ctx, profileID)
	if err != nil {
		return fmt.Errorf("failed to load profile swaps: %w", err)
	}
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
