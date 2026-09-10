package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/authz"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/evidence"
	"scenario-to-cloud/evidencesvc"
	"scenario-to-cloud/internal/httputil"
	"scenario-to-cloud/internal/shellutil"
	"scenario-to-cloud/ramp"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/receiptsigning"
	credentialauthority "github.com/vrooli/vrooli/packages/credential-authority-go"
	credentialauthoritysigning "github.com/vrooli/vrooli/packages/credential-authority-go/receiptsigning"
)

// receiptSigningIdentity is the credential-authority identity that holds the
// cloud receipt signing key ring. The private key never leaves the authority.
const receiptSigningIdentity = "scenario-to-cloud/receipt-signing"

// publicationDeps are the governance seams the publication surface needs.
// Production values are resolved lazily; tests substitute fakes.
type publicationDeps struct {
	governor evidence.Governor
	signer   receiptsigning.ReceiptSigner
	// signerProduction reports whether the signer is a production trust root.
	signerProduction bool
	target           ramp.TargetReader
	// releaseFor resolves the immutable release a deployed bundle belongs to.
	releaseFor func(bundleSHA string) domain.ReceiptRelease
	// bundleFor resolves the bundle sha256 of a stored release digest.
	bundleFor func(releaseDigest string) string
	profile   *evidence.Profile
	now       func() time.Time
	// activate admits the reviewed plan as the activation operation. It is
	// the existing plan-apply path; tests substitute a fake.
	activate func(ctx context.Context, deploymentID, planDigest, requestKey string) (*PlanApplyResult, *apierrors.Error)
}

func (s *Server) publication() *publicationDeps {
	if s.publicationDeps == nil {
		s.publicationDeps = &publicationDeps{}
	}
	d := s.publicationDeps
	if d.governor == nil {
		d.governor = evidence.NewDMGovernor()
	}
	if d.signer == nil {
		d.signer, d.signerProduction = defaultReceiptSigner(s.log)
	}
	if d.target == nil {
		d.target = sshTargetReader{server: s}
	}
	if d.releaseFor == nil {
		d.releaseFor = releaseForBundle
	}
	if d.bundleFor == nil {
		d.bundleFor = bundleForRelease
	}
	if d.profile == nil {
		if profile, err := evidence.LoadProfile(evidence.ProfileCloudLaunchV1); err == nil {
			d.profile = profile
		}
	}
	if d.now == nil {
		d.now = func() time.Time { return time.Now().UTC() }
	}
	if d.activate == nil {
		d.activate = func(ctx context.Context, deploymentID, planDigest, requestKey string) (*PlanApplyResult, *apierrors.Error) {
			return s.applyDeploymentPlan(ctx, deploymentID, PlanApplyRequest{PlanDigest: planDigest, RequestKey: requestKey})
		}
	}
	return d
}

// defaultReceiptSigner binds the credential-authority Ed25519 signer when the
// authority is available and falls back to the development HMAC signer
// (never production eligible) otherwise. The choice is logged once so an
// unsigned-by-trust-root deployment is never silent.
func defaultReceiptSigner(log func(string, map[string]interface{})) (receiptsigning.ReceiptSigner, bool) {
	authority, err := credentialauthority.Default()
	if err == nil {
		signer, signErr := credentialauthoritysigning.New(credentialauthoritysigning.Config{Identity: credentialauthority.Identity(receiptSigningIdentity), Store: authority, AllowedPurposes: []receiptsigning.Purpose{receiptsigning.PurposeCloudEvidenceReceipt}})
		if signErr == nil {
			return signer, true
		}
		err = signErr
	}
	if log != nil {
		log("receipt signing: credential authority unavailable, using the development signer", map[string]interface{}{"error": err.Error()})
	}
	return receiptsigning.NewDevelopmentSigner(), false
}

// releaseForBundle finds the stored release whose bundle is bundleSHA.
func releaseForBundle(bundleSHA string) domain.ReceiptRelease {
	bundleSHA = strings.ToLower(strings.TrimSpace(bundleSHA))
	if bundleSHA == "" {
		return domain.ReceiptRelease{}
	}
	svc, err := releaseService()
	if err != nil {
		return domain.ReceiptRelease{}
	}
	releases, err := svc.Store().List()
	if err != nil {
		return domain.ReceiptRelease{}
	}
	for _, rel := range releases {
		if rel.Complete && strings.EqualFold(rel.Manifest.BundleSHA256, bundleSHA) {
			return domain.ReceiptRelease{ReleaseDigest: rel.Digest, ConfigurationDigest: rel.Manifest.ConfigurationDigest}
		}
	}
	return domain.ReceiptRelease{}
}

// bundleForRelease finds the bundle sha256 of a complete stored release.
func bundleForRelease(releaseDigest string) string {
	releaseDigest = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(releaseDigest), "sha256:"))
	if releaseDigest == "" {
		return ""
	}
	svc, err := releaseService()
	if err != nil {
		return ""
	}
	rel, err := svc.Store().Load(releaseDigest)
	if err != nil || !rel.Complete {
		return ""
	}
	return rel.Manifest.BundleSHA256
}

// sshTargetReader reads the target's active-release pointer over the
// deployment's SSH binding. The command is built by policy code from the
// manifest workdir; it reads one file and changes nothing.
type sshTargetReader struct{ server *Server }

func (r sshTargetReader) ReadActiveRelease(ctx context.Context, deploymentID string) (evidence.TargetReceipt, error) {
	dc, lerr := r.server.loadDeploymentContext(ctx, deploymentID)
	if lerr != nil {
		return evidence.TargetReceipt{}, lerr
	}
	pointer := shellutil.SafeRemoteJoin(dc.Workdir, ".vrooli", "cloud", "active-release.json")
	result, err := r.server.proberFor(dc).Observe(ctx, "cat", "--", pointer)
	if err != nil {
		return evidence.TargetReceipt{}, fmt.Errorf("read active-release pointer: %w", err)
	}
	if result.ExitCode != 0 {
		return evidence.TargetReceipt{}, fmt.Errorf("read active-release pointer: exit %d: %s", result.ExitCode, strings.TrimSpace(result.Stderr))
	}
	var receipt evidence.TargetReceipt
	if err := json.Unmarshal([]byte(result.Stdout), &receipt); err != nil {
		return evidence.TargetReceipt{}, fmt.Errorf("decode active-release pointer: %w", err)
	}
	if strings.TrimSpace(receipt.ActiveRelease) == "" {
		return evidence.TargetReceipt{}, fmt.Errorf("active-release pointer names no release")
	}
	return receipt, nil
}

// registerPublicationRoutes mounts the evidence and publication surface.
// Mount line for main.go setupRoutes: s.registerPublicationRoutes(api).
func (s *Server) registerPublicationRoutes(api *mux.Router) {
	api.HandleFunc("/releases/{digest}/evidence", s.handleGetReleaseEvidence).Methods("GET")
	api.HandleFunc("/deployments/{id}/evidence", s.handleGetDeploymentEvidence).Methods("GET")
	api.HandleFunc("/deployments/{id}/publication/request", s.handleRequestPublication).Methods("POST")
	api.HandleFunc("/deployments/{id}/publication/apply", s.handleApplyPublication).Methods("POST")
	api.HandleFunc("/deployments/{id}/publication", s.handleGetPublication).Methods("GET")
	if s.repo != nil {
		path, handler := evidencesvc.New(s).Handler()
		s.router.PathPrefix(path).Handler(handler)
	}
}

// --- transport-free operations (shared by REST and Connect) ---

// ReleaseEvidence implements evidencesvc.Publisher.
func (s *Server) ReleaseEvidence(ctx context.Context, releaseDigest, deploymentID string) (*evidence.Summary, error) {
	d := s.publication()
	if d.profile == nil {
		return nil, apierrors.New(apierrors.CodeInternal, "capability profile is unavailable")
	}
	releaseDigest = strings.ToLower(strings.TrimSpace(releaseDigest))
	if releaseDigest == "" {
		return nil, apierrors.New(apierrors.CodeInvalidRequest, "release digest is required")
	}
	targetKey := ""
	if strings.TrimSpace(deploymentID) != "" {
		dep, err := s.repo.GetDeployment(ctx, deploymentID)
		if err != nil {
			return nil, apierrors.Internal("Failed to get deployment", err)
		}
		if dep == nil {
			return nil, deploymentNotFound(deploymentID)
		}
		targetKey = ramp.TargetKey(dep)
	}
	records, err := s.repo.ListEvidenceRecords(ctx, releaseDigest, deploymentID)
	if err != nil {
		return nil, apierrors.Internal("Failed to list evidence records", err)
	}
	summary := evidence.Evaluate(d.profile, records, releaseDigest, targetKey, d.now())
	return &summary, nil
}

// deploymentRelease resolves the release digest a deployment currently
// runs, from the explicit override or the deployed bundle.
func (s *Server) deploymentRelease(dep *domain.Deployment, override string) string {
	if v := strings.ToLower(strings.TrimSpace(override)); v != "" {
		return v
	}
	if dep == nil || dep.BundleSHA256 == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(s.publication().releaseFor(*dep.BundleSHA256).ReleaseDigest))
}

// RequestPublication implements evidencesvc.Publisher: it evaluates the
// profile, reports coverage to the governance owner, asks for the exact
// review and stores the review reference. It performs no target effect.
func (s *Server) RequestPublication(ctx context.Context, deploymentID, requestKey string, identity evidence.ReviewIdentity) (*evidence.Publication, *evidence.Summary, error) {
	d := s.publication()
	requestKey = strings.TrimSpace(requestKey)
	if requestKey == "" {
		return nil, nil, apierrors.New(apierrors.CodeInvalidRequest, "request_key is required")
	}
	dep, err := s.repo.GetDeployment(ctx, deploymentID)
	if err != nil {
		return nil, nil, apierrors.Internal("Failed to get deployment", err)
	}
	if dep == nil {
		return nil, nil, deploymentNotFound(deploymentID)
	}
	targetKey := ramp.TargetKey(dep)
	if identity.ScenarioID == "" {
		identity.ScenarioID = dep.ScenarioID
	}
	if identity.Environment == "" {
		identity.Environment = dep.Environment
	}
	if len(identity.TargetSet) == 0 {
		identity.TargetSet = []string{targetKey}
	}
	if identity.ReleaseDigest == "" {
		identity.ReleaseDigest = s.deploymentRelease(dep, "")
	}
	if identity.ConfigurationDigest == "" && dep.BundleSHA256 != nil {
		identity.ConfigurationDigest = d.releaseFor(*dep.BundleSHA256).ConfigurationDigest
	}
	canonical, err := identity.Canonical()
	if err != nil {
		return nil, nil, apierrors.New(apierrors.CodeInvalidRequest, err.Error())
	}
	if !containsString(canonical.TargetSet, targetKey) {
		return nil, nil, apierrors.New(apierrors.CodeInvalidRequest, "review identity target set does not include this deployment's target").WithDetail("target_key", targetKey)
	}
	if canonical.ScenarioID != dep.ScenarioID {
		return nil, nil, apierrors.New(apierrors.CodeInvalidRequest, "review identity scenario does not match the deployment")
	}
	digest, _ := canonical.Digest()
	if existing, err := s.repo.GetPublicationByRequestKey(ctx, deploymentID, requestKey); err != nil {
		return nil, nil, apierrors.Internal("Failed to read publication", err)
	} else if existing != nil {
		if existing.IdentityDigest != digest {
			return nil, nil, apierrors.New(apierrors.CodeRequestKeyConflict, "request_key is bound to a different review identity").WithDetail("identity_digest", existing.IdentityDigest)
		}
		summary, serr := s.ReleaseEvidence(ctx, existing.ReleaseDigest, deploymentID)
		if serr != nil {
			return nil, nil, serr
		}
		return existing, summary, nil
	}
	summary, serr := s.ReleaseEvidence(ctx, canonical.ReleaseDigest, deploymentID)
	if serr != nil {
		return nil, nil, serr
	}
	now := d.now()
	pub := evidence.Publication{
		SchemaVersion: evidence.PublicationSchemaVersion, ID: uuid.New().String(), DeploymentID: deploymentID, RequestKey: requestKey,
		Identity: canonical, IdentityDigest: digest, ReleaseDigest: canonical.ReleaseDigest, State: evidence.StateRequested, TargetKey: targetKey, RequestedAt: now, UpdatedAt: now,
	}
	stored, _, err := s.repo.CreatePublication(ctx, pub)
	if err != nil {
		return nil, nil, apierrors.Internal("Failed to record publication", err)
	}
	verdict, err := summary.CoverageVerdict(stored.ID, now)
	if err != nil {
		return nil, nil, apierrors.Internal("Failed to build coverage verdict", err)
	}
	if err := d.governor.ReportCoverage(ctx, canonical, verdict); err != nil {
		stored.Refuse(evidence.RefusalReviewUnavailable, err.Error(), now)
		_ = s.repo.UpdatePublication(ctx, *stored)
		return stored, summary, apierrors.New(apierrors.CodeGovernanceUnavailable, "Governance owner did not accept the coverage report").WithDetail("cause", err.Error()).WithRetryable(true)
	}
	review, err := d.governor.PrepareReview(ctx, canonical)
	if err != nil {
		stored.Refuse(evidence.RefusalReviewUnavailable, err.Error(), now)
		_ = s.repo.UpdatePublication(ctx, *stored)
		return stored, summary, apierrors.New(apierrors.CodeGovernanceUnavailable, "Governance owner did not prepare the review").WithDetail("cause", err.Error()).WithRetryable(true)
	}
	if mismatch := review.MatchesIdentity(canonical); mismatch != nil {
		stored.Refuse(evidence.RefusalReviewMismatch, mismatch.Error(), now)
		_ = s.repo.UpdatePublication(ctx, *stored)
		return stored, summary, apierrors.New(apierrors.CodePublicationRefused, "Governance owner prepared a review for a different identity").WithDetail("cause", mismatch.Error())
	}
	stored.ReviewRef = review.Key
	stored.State = evidence.StateReviewPrepared
	if review.Status == evidence.ReviewApproved {
		stored.State = evidence.StateApproved
	}
	stored.UpdatedAt = now
	if err := s.repo.UpdatePublication(ctx, *stored); err != nil {
		return nil, nil, apierrors.Internal("Failed to update publication", err)
	}
	return stored, summary, nil
}

// ApplyPublication implements evidencesvc.Publisher: it re-checks the
// approval with the governance owner immediately before admitting the
// activation, refuses on any mismatch, and records the operation.
func (s *Server) ApplyPublication(ctx context.Context, deploymentID, requestKey, reviewRef, planDigest string) (*evidence.Publication, *evidence.Summary, error) {
	d := s.publication()
	requestKey, reviewRef, planDigest = strings.TrimSpace(requestKey), strings.TrimSpace(reviewRef), strings.TrimSpace(planDigest)
	if requestKey == "" || reviewRef == "" || planDigest == "" {
		return nil, nil, apierrors.New(apierrors.CodeInvalidRequest, "request_key, review_ref and plan_digest are required")
	}
	pub, err := s.repo.GetPublicationByRequestKey(ctx, deploymentID, requestKey)
	if err != nil {
		return nil, nil, apierrors.Internal("Failed to read publication", err)
	}
	if pub == nil {
		return nil, nil, apierrors.New(apierrors.CodePublicationRefused, "No publication was requested under this request_key").WithDetail("refusal", evidence.RefusalReviewUnavailable)
	}
	summary, serr := s.ReleaseEvidence(ctx, pub.ReleaseDigest, deploymentID)
	if serr != nil {
		return nil, nil, serr
	}
	now := d.now()
	refuse := func(code, reason string) (*evidence.Publication, *evidence.Summary, error) {
		pub.Refuse(code, reason, now)
		_ = s.repo.UpdatePublication(ctx, *pub)
		return pub, summary, apierrors.New(apierrors.CodePublicationRefused, "Publication refused: "+reason).WithDetail("refusal", code).WithDetail("review_ref", pub.ReviewRef)
	}
	switch pub.State {
	case evidence.StatePublished:
		return pub, summary, nil
	case evidence.StateActivating:
		if err := s.reconcilePublication(ctx, pub); err != nil {
			return nil, nil, err
		}
		return pub, summary, nil
	case evidence.StateFailed:
		return pub, summary, apierrors.New(apierrors.CodePublicationRefused, "Publication already failed; request a new publication").WithDetail("refusal", pub.Refusal)
	}
	if pub.ReviewRef == "" || pub.ReviewRef != reviewRef {
		return refuse(evidence.RefusalReviewMismatch, fmt.Sprintf("review_ref %s is not the review bound to this request (%s)", reviewRef, pub.ReviewRef))
	}
	if !summary.Passed {
		return refuse(evidence.RefusalEvidenceIncomplete, "required cells are not passed: "+strings.Join(summary.BlockingCells, ", "))
	}
	review, reviewErr := d.governor.GetReview(ctx, reviewRef)
	if refusal := evidence.Decide(pub, review, reviewErr); refusal != nil {
		return refuse(refusal.Code, refusal.Reason)
	}
	pub.State = evidence.StateApproved
	pub.PlanDigest = planDigest
	pub.UpdatedAt = now
	if err := s.repo.UpdatePublication(ctx, *pub); err != nil {
		return nil, nil, apierrors.Internal("Failed to update publication", err)
	}
	applied, aerr := d.activate(ctx, deploymentID, planDigest, "publish:"+requestKey)
	if aerr != nil {
		if aerr.Code == apierrors.CodePlanDigestMismatch || aerr.Code == apierrors.CodePlanStale {
			return refuse(evidence.RefusalPlanMismatch, aerr.Message)
		}
		return nil, nil, aerr
	}
	pub.OperationID = applied.OperationID
	pub.State = evidence.StateActivating
	pub.UpdatedAt = now
	if err := s.repo.UpdatePublication(ctx, *pub); err != nil {
		return nil, nil, apierrors.Internal("Failed to update publication", err)
	}
	if err := s.reconcilePublication(ctx, pub); err != nil {
		return nil, nil, err
	}
	return pub, summary, nil
}

// GetPublication implements evidencesvc.Publisher. It reconciles an
// activating publication against the target receipt before answering.
func (s *Server) GetPublication(ctx context.Context, deploymentID, requestKey string) (*evidence.Publication, *evidence.Summary, error) {
	var pub *evidence.Publication
	if strings.TrimSpace(requestKey) != "" {
		stored, err := s.repo.GetPublicationByRequestKey(ctx, deploymentID, strings.TrimSpace(requestKey))
		if err != nil {
			return nil, nil, apierrors.Internal("Failed to read publication", err)
		}
		pub = stored
	} else {
		list, err := s.repo.ListPublications(ctx, deploymentID)
		if err != nil {
			return nil, nil, apierrors.Internal("Failed to list publications", err)
		}
		if len(list) > 0 {
			pub = &list[0]
		}
	}
	if pub == nil {
		return nil, nil, apierrors.New(apierrors.CodeOperationNotFound, "No publication exists for this deployment").WithDetail("deployment_id", deploymentID)
	}
	if pub.State == evidence.StateActivating {
		if err := s.reconcilePublication(ctx, pub); err != nil {
			return nil, nil, err
		}
	}
	summary, serr := s.ReleaseEvidence(ctx, pub.ReleaseDigest, deploymentID)
	if serr != nil {
		return pub, nil, nil
	}
	return pub, summary, nil
}

// reconcilePublication resolves an activating publication from the
// deployment record and the target's own active-release pointer. It never
// re-dispatches the activation: a lost reply is reconciled by reading the
// target, so the effect happens at most once per request key.
func (s *Server) reconcilePublication(ctx context.Context, pub *evidence.Publication) *apierrors.Error {
	if pub.State != evidence.StateActivating {
		return nil
	}
	d := s.publication()
	dep, err := s.repo.GetDeployment(ctx, pub.DeploymentID)
	if err != nil {
		return apierrors.Internal("Failed to get deployment", err)
	}
	if dep == nil {
		return deploymentNotFound(pub.DeploymentID)
	}
	now := d.now()
	switch dep.Status {
	case domain.StatusFailed, domain.StatusStopped:
		reason := "deployment ended " + string(dep.Status)
		if dep.ErrorMessage != nil && *dep.ErrorMessage != "" {
			reason += ": " + *dep.ErrorMessage
		}
		pub.State = evidence.StateFailed
		pub.Refusal = &evidence.Refusal{Code: "activation_failed", Reason: reason}
		pub.UpdatedAt = now
	case domain.StatusDeployed:
		receipt, readErr := d.target.ReadActiveRelease(ctx, pub.DeploymentID)
		if readErr != nil {
			// Still activating from the record's point of view; the pointer is
			// the only proof, so the state stays activating and the cause is
			// reported without any second effect.
			pub.Refusal = &evidence.Refusal{Code: "target_receipt_unavailable", Reason: readErr.Error()}
			pub.UpdatedAt = now
		} else {
			history, herr := s.repo.ListPublications(ctx, pub.DeploymentID)
			if herr != nil {
				return apierrors.Internal("Failed to list publications", herr)
			}
			evidence.RecordActivation(pub, receipt, history, now)
		}
	default:
		return nil
	}
	if err := s.repo.UpdatePublication(ctx, *pub); err != nil {
		return apierrors.Internal("Failed to update publication", err)
	}
	return nil
}

func containsString(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}

// --- REST handlers ---

func (s *Server) handleGetReleaseEvidence(w http.ResponseWriter, r *http.Request) {
	summary, err := s.ReleaseEvidence(r.Context(), mux.Vars(r)["digest"], r.URL.Query().Get("deployment_id"))
	if err != nil {
		apierrors.Write(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"schema_version": evidence.SummarySchemaVersion, "evidence": summary})
}

func (s *Server) handleGetDeploymentEvidence(w http.ResponseWriter, r *http.Request) {
	id, dep := s.FetchDeploymentOnly(w, r)
	if dep == nil {
		return
	}
	releaseDigest := s.deploymentRelease(dep, r.URL.Query().Get("release_digest"))
	if releaseDigest == "" {
		apierrors.Write(w, apierrors.New(apierrors.CodeInvalidRequest, "Deployment runs no stored release; pass release_digest").WithDetail("deployment_id", id))
		return
	}
	summary, err := s.ReleaseEvidence(r.Context(), releaseDigest, id)
	if err != nil {
		apierrors.Write(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"schema_version": evidence.SummarySchemaVersion, "evidence": summary})
}

type publicationRequestBody struct {
	RequestKey string                  `json:"request_key"`
	Identity   evidence.ReviewIdentity `json:"identity"`
}

func (s *Server) handleRequestPublication(w http.ResponseWriter, r *http.Request) {
	id, dep := s.FetchDeploymentOnly(w, r)
	if dep == nil {
		return
	}
	if denied := s.authz.RequireEffect(r.Context(), id, authz.EffectWorkloadMutation); denied != nil {
		apierrors.Write(w, denied)
		return
	}
	var body publicationRequestBody
	if !httputil.DecodeRequestBody(w, r, &body) {
		return
	}
	pub, summary, err := s.RequestPublication(r.Context(), id, body.RequestKey, body.Identity)
	if err != nil {
		if typed := apierrors.As(err); typed != nil && pub != nil {
			err = typed.WithDetail("publication", pub)
		}
		apierrors.Write(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"schema_version": evidence.PublicationSchemaVersion, "publication": pub, "evidence": summary})
}

type publicationApplyBody struct {
	RequestKey string `json:"request_key"`
	ReviewRef  string `json:"review_ref"`
	PlanDigest string `json:"plan_digest"`
}

func (s *Server) handleApplyPublication(w http.ResponseWriter, r *http.Request) {
	id, dep := s.FetchDeploymentOnly(w, r)
	if dep == nil {
		return
	}
	if denied := s.authz.RequireEffect(r.Context(), id, authz.EffectHost); denied != nil {
		apierrors.Write(w, denied)
		return
	}
	var body publicationApplyBody
	if !httputil.DecodeRequestBody(w, r, &body) {
		return
	}
	pub, summary, err := s.ApplyPublication(r.Context(), id, body.RequestKey, body.ReviewRef, body.PlanDigest)
	if err != nil {
		if typed := apierrors.As(err); typed != nil && pub != nil {
			err = typed.WithDetail("publication", pub)
		}
		apierrors.Write(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"schema_version": evidence.PublicationSchemaVersion, "publication": pub, "evidence": summary})
}

func (s *Server) handleGetPublication(w http.ResponseWriter, r *http.Request) {
	id, dep := s.FetchDeploymentOnly(w, r)
	if dep == nil {
		return
	}
	pub, summary, err := s.GetPublication(r.Context(), id, r.URL.Query().Get("request_key"))
	if err != nil {
		apierrors.Write(w, err)
		return
	}
	httputil.WriteJSON(w, http.StatusOK, map[string]any{"schema_version": evidence.PublicationSchemaVersion, "publication": pub, "evidence": summary})
}

// signedDeploymentReceipt builds, binds and signs the deployment receipt.
func (s *Server) signedDeploymentReceipt(ctx context.Context, dep *domain.Deployment) (*domain.CloudDeploymentReceipt, error) {
	d := s.publication()
	rel := domain.ReceiptRelease{}
	if dep.BundleSHA256 != nil {
		rel = d.releaseFor(*dep.BundleSHA256)
	}
	if list, err := s.repo.ListPublications(ctx, dep.ID); err == nil {
		for _, pub := range list {
			if pub.State == evidence.StatePublished && pub.OperationID != "" {
				rel.OperationID = pub.OperationID
				break
			}
		}
	}
	receipt, err := domain.BuildCloudDeploymentReceipt(dep, dep.UpdatedAt, rel)
	if err != nil {
		return nil, err
	}
	if err := receipt.Sign(ctx, d.signer); err != nil {
		return nil, err
	}
	return receipt, nil
}
