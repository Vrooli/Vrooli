package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"scenario-to-cloud/apierrors"
	"scenario-to-cloud/domain"
	"scenario-to-cloud/evidence"
	"scenario-to-cloud/faultinject"
	"scenario-to-cloud/persistence"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/receiptsigning"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

var (
	pubRelease = strings.Repeat("a", 64)
	pubOther   = strings.Repeat("b", 64)
	pubBundle  = strings.Repeat("c", 64)
	pubConfig  = strings.Repeat("d", 64)
	pubNow     = time.Date(2026, 9, 9, 15, 0, 0, 0, time.UTC)
)

type fakeGovernor struct {
	mu       sync.Mutex
	verdicts []*commonv1.TargetVerdict
	reviews  map[string]*evidence.ReviewSnapshot
	prepared *evidence.ReviewSnapshot
	prepErr  error
	getErr   error
}

func (g *fakeGovernor) ReportCoverage(_ context.Context, _ evidence.ReviewIdentity, verdict *commonv1.TargetVerdict) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.verdicts = append(g.verdicts, verdict)
	return nil
}

func (g *fakeGovernor) PrepareReview(context.Context, evidence.ReviewIdentity) (*evidence.ReviewSnapshot, error) {
	if g.prepErr != nil {
		return nil, g.prepErr
	}
	return g.prepared, nil
}

func (g *fakeGovernor) GetReview(_ context.Context, ref string) (*evidence.ReviewSnapshot, error) {
	if g.getErr != nil {
		return nil, g.getErr
	}
	return g.reviews[ref], nil
}

type fakeTargetReader struct {
	receipt  evidence.TargetReceipt
	err      error
	registry *faultinject.Registry
	ctx      context.Context
	reads    int
}

func (f *fakeTargetReader) ReadActiveRelease(context.Context, string) (evidence.TargetReceipt, error) {
	f.reads++
	if f.registry != nil {
		if err := f.registry.Hit(f.ctx, faultinject.TransportReply); err != nil {
			return evidence.TargetReceipt{}, err
		}
	}
	return f.receipt, f.err
}

type fakeActivator struct {
	repo  *persistence.Repository
	calls int
	err   *apierrors.Error
}

func (a *fakeActivator) activate(ctx context.Context, deploymentID, planDigest, requestKey string) (*PlanApplyResult, *apierrors.Error) {
	a.calls++
	if a.err != nil {
		return nil, a.err
	}
	_ = a.repo.UpdateDeploymentStatus(ctx, deploymentID, domain.StatusDeployed, nil, nil)
	return &PlanApplyResult{SchemaVersion: "1", OperationID: "op-publish-" + requestKey, PlanDigest: planDigest, State: "admitted"}, nil
}

func snapshot(key, status, artifact string, targets ...string) *evidence.ReviewSnapshot {
	if len(targets) == 0 {
		targets = []string{"host:203.0.113.10"}
	}
	return &evidence.ReviewSnapshot{Key: key, Status: status, Scenario: "demo", ProfileID: "profile-1", CandidateCommit: "abc123", ArtifactDigest: artifact, Targets: targets, Channel: "stable", PolicyVersion: 2}
}

func newPublicationTestServer(t *testing.T, name string) (*Server, *persistence.Repository, *fakeGovernor, *fakeTargetReader, *fakeActivator) {
	t.Helper()
	srv, repo := newIdentityTestServer(t, name)
	seedDeployment(t, repo, "dep-pub", "demo", "staging", "203.0.113.10", "demo.example")
	if err := repo.UpdateDeploymentBundle(context.Background(), "dep-pub", "/var/lib/demo/bundle.tar.gz", pubBundle, 42); err != nil {
		t.Fatal(err)
	}
	governor := &fakeGovernor{reviews: map[string]*evidence.ReviewSnapshot{}, prepared: snapshot("rr-1", "agent_review", pubRelease)}
	target := &fakeTargetReader{receipt: evidence.TargetReceipt{ActiveRelease: pubRelease, PreviousRelease: pubOther, ActivatedAt: pubNow.Format(time.RFC3339), OperationID: "op-target", Fence: 2}}
	activator := &fakeActivator{repo: repo}
	srv.publicationDeps = &publicationDeps{
		governor: governor, signer: receiptsigning.NewDevelopmentSigner(), target: target,
		releaseFor: func(bundle string) domain.ReceiptRelease {
			if bundle == pubBundle {
				return domain.ReceiptRelease{ReleaseDigest: pubRelease, ConfigurationDigest: pubConfig}
			}
			return domain.ReceiptRelease{}
		},
		now:      func() time.Time { return pubNow },
		activate: activator.activate,
	}
	return srv, repo, governor, target, activator
}

func appendPassingEvidence(t *testing.T, repo *persistence.Repository, release, target string, skip map[string]evidence.Disposition) {
	t.Helper()
	profile, err := evidence.LoadProfile(evidence.ProfileCloudLaunchV1)
	if err != nil {
		t.Fatal(err)
	}
	for _, cell := range profile.Cells {
		rec := evidence.Record{
			SchemaVersion: evidence.RecordSchemaVersion, ID: uuid.New().String(), ProfileID: profile.ID, CaseID: cell.CaseID, Lane: cell.Lane, Disposition: evidence.DispositionPassed,
			Binding:     evidence.Binding{ProducerRef: evidence.ProducerRef, DeploymentID: "dep-pub", ReleaseDigest: release, TargetKey: target, OperationID: "op-" + cell.CaseID},
			ReceiptRefs: []string{"cloud-target:op-" + cell.CaseID + ":receipt"}, ObservedAt: pubNow.Add(-time.Minute), RecordedAt: pubNow.Add(-time.Minute),
		}
		if d, ok := skip[cell.String()]; ok {
			rec.Disposition, rec.ReceiptRefs, rec.Reason = d, nil, string(d)+" by fixture"
			rec.Binding.OperationID, rec.Binding.ObservationID = "", "obs-"+cell.CaseID
		}
		if err := rec.Validate(profile); err != nil {
			t.Fatal(err)
		}
		if err := repo.AppendEvidenceRecord(context.Background(), rec); err != nil {
			t.Fatal(err)
		}
	}
}

func callJSON(t *testing.T, srv *Server, method, path string, body any) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.router.ServeHTTP(rec, req)
	var decoded map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &decoded)
	return rec, decoded
}

func requestBody(release string) map[string]any {
	return map[string]any{"request_key": "req-1", "identity": map[string]any{
		"scenario_id": "demo", "profile_id": "profile-1", "candidate_commit": "abc123", "release_digest": release,
		"configuration_digest": pubConfig, "environment": "staging", "channel": "stable", "policy_version": 2,
	}}
}

func publicationErrorCode(decoded map[string]any) (string, map[string]any) {
	wrapper, _ := decoded["error"].(map[string]any)
	code, _ := wrapper["code"].(string)
	details, _ := wrapper["details"].(map[string]any)
	return code, details
}

// TestReleaseEvidenceKeepsEveryDispositionDistinct [REQ:STC-P0-035] proves
// P17-O06 on the wire: missing, unavailable and passed cells are reported
// with their own word and receipt references (GOV-01, GOV-07).
func TestReleaseEvidenceKeepsEveryDispositionDistinct(t *testing.T) {
	srv, repo, _, _, _ := newPublicationTestServer(t, "handlers-publication-evidence")
	rec, decoded := callJSON(t, srv, http.MethodGet, "/api/v1/releases/"+pubRelease+"/evidence?deployment_id=dep-pub", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	summary := decoded["evidence"].(map[string]any)
	if summary["passed"] != false || len(summary["blocking_cells"].([]any)) != int(summary["required_cells"].(float64)) {
		t.Fatalf("empty evidence must block every cell: %v", summary)
	}
	appendPassingEvidence(t, repo, pubRelease, "host:203.0.113.10", map[string]evidence.Disposition{"GOV-07/api": evidence.DispositionUnavailable})
	_, decoded = callJSON(t, srv, http.MethodGet, "/api/v1/deployments/dep-pub/evidence", nil)
	summary = decoded["evidence"].(map[string]any)
	if summary["passed"] != false || summary["release_digest"] != pubRelease || summary["target_key"] != "host:203.0.113.10" {
		t.Fatalf("summary = %v", summary)
	}
	dispositions := map[string]int{}
	for _, raw := range summary["cells"].([]any) {
		cell := raw.(map[string]any)
		dispositions[cell["disposition"].(string)]++
		if cell["disposition"] == "passed" && len(cell["receipt_refs"].([]any)) == 0 {
			t.Fatalf("passed cell without receipt refs: %v", cell)
		}
		if cell["case_id"] == "GOV-07" && cell["lane"] == "api" && (cell["disposition"] != "unavailable" || cell["observation_id"] != "obs-GOV-07") {
			t.Fatalf("GOV-07 cell = %v", cell)
		}
	}
	if dispositions["unavailable"] != 1 || dispositions["passed"] != int(summary["required_cells"].(float64))-1 {
		t.Fatalf("dispositions = %v", dispositions)
	}
	// Evidence for another release does not leak into this one (GOV-02).
	_, other := callJSON(t, srv, http.MethodGet, "/api/v1/releases/"+pubOther+"/evidence?deployment_id=dep-pub", nil)
	if other["evidence"].(map[string]any)["passed"] != false {
		t.Fatalf("foreign release must not pass: %v", other)
	}
}

// TestPublicationRequestReportsCoverageAndBindsTheReview [REQ:STC-P0-035]
// proves P17 steps 6-7: the coverage verdict reaches the governance owner
// with one ref per cell and the review reference is stored under the exact
// identity digest; the same request key replays without a second report.
func TestPublicationRequestReportsCoverageAndBindsTheReview(t *testing.T) {
	srv, repo, governor, _, _ := newPublicationTestServer(t, "handlers-publication-request")
	appendPassingEvidence(t, repo, pubRelease, "host:203.0.113.10", nil)
	rec, decoded := callJSON(t, srv, http.MethodPost, "/api/v1/deployments/dep-pub/publication/request", requestBody(pubRelease))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	pub := decoded["publication"].(map[string]any)
	if pub["state"] != evidence.StateReviewPrepared || pub["review_ref"] != "rr-1" || pub["target_key"] != "host:203.0.113.10" || pub["identity_digest"] == "" {
		t.Fatalf("publication = %v", pub)
	}
	if len(governor.verdicts) != 1 || governor.verdicts[0].Disposition != commonv1.Disposition_DISPOSITION_PASSED || governor.verdicts[0].Target.Platform != "host:203.0.113.10" || len(governor.verdicts[0].Refs) == 0 {
		t.Fatalf("verdicts = %v", governor.verdicts)
	}
	replay, _ := callJSON(t, srv, http.MethodPost, "/api/v1/deployments/dep-pub/publication/request", requestBody(pubRelease))
	if replay.Code != http.StatusOK || len(governor.verdicts) != 1 {
		t.Fatalf("replay status = %d verdicts=%d", replay.Code, len(governor.verdicts))
	}
	conflict, body := callJSON(t, srv, http.MethodPost, "/api/v1/deployments/dep-pub/publication/request", requestBody(pubOther))
	if code, _ := publicationErrorCode(body); conflict.Code != http.StatusConflict || code != apierrors.CodeRequestKeyConflict {
		t.Fatalf("different identity under the same key = %d %s", conflict.Code, conflict.Body.String())
	}
	governor.prepErr = errors.New("deployment-manager unreachable")
	down, body := callJSON(t, srv, http.MethodPost, "/api/v1/deployments/dep-pub/publication/request", map[string]any{"request_key": "req-down", "identity": requestBody(pubRelease)["identity"]})
	if code, _ := publicationErrorCode(body); down.Code != http.StatusServiceUnavailable || code != apierrors.CodeGovernanceUnavailable {
		t.Fatalf("governance outage = %d %s", down.Code, down.Body.String())
	}
}

// TestPublicationApplyRefusesUnlessTheExactApprovalStands [REQ:STC-P0-035]
// proves GOV-03 (approved bytes replaced), GOV-06 (approval revoked or
// superseded before the mutation), the evidence gate at apply time, and
// that a refusal performs no activation (P17-A02).
func TestPublicationApplyRefusesUnlessTheExactApprovalStands(t *testing.T) {
	srv, repo, governor, _, activator := newPublicationTestServer(t, "handlers-publication-refusals")
	appendPassingEvidence(t, repo, pubRelease, "host:203.0.113.10", map[string]evidence.Disposition{"GOV-07/api": evidence.DispositionUnavailable})
	if rec, _ := callJSON(t, srv, http.MethodPost, "/api/v1/deployments/dep-pub/publication/request", requestBody(pubRelease)); rec.Code != http.StatusOK {
		t.Fatalf("request status = %d %s", rec.Code, rec.Body.String())
	}
	apply := func(reviewRef string) (*httptest.ResponseRecorder, string, map[string]any) {
		rec, body := callJSON(t, srv, http.MethodPost, "/api/v1/deployments/dep-pub/publication/apply", map[string]any{"request_key": "req-1", "review_ref": reviewRef, "plan_digest": "sha256:plan"})
		code, details := publicationErrorCode(body)
		return rec, code, details
	}
	governor.reviews["rr-1"] = snapshot("rr-1", evidence.ReviewApproved, pubRelease)
	rec, code, details := apply("rr-1")
	if rec.Code != http.StatusConflict || code != apierrors.CodePublicationRefused || details["refusal"] != evidence.RefusalEvidenceIncomplete {
		t.Fatalf("incomplete evidence must refuse: %d %s", rec.Code, rec.Body.String())
	}
	// Withdraw the unavailable record with a fresh passed rerun so the cell passes.
	withdraw := func() {
		profile, _ := evidence.LoadProfile(evidence.ProfileCloudLaunchV1)
		cell := evidence.CellID{CaseID: "GOV-07", Lane: "api"}
		rec := evidence.Record{SchemaVersion: 1, ID: uuid.New().String(), ProfileID: profile.ID, CaseID: cell.CaseID, Lane: cell.Lane, Disposition: evidence.DispositionPassed, Binding: evidence.Binding{ProducerRef: evidence.ProducerRef, DeploymentID: "dep-pub", ReleaseDigest: pubRelease, TargetKey: "host:203.0.113.10", OperationID: "op-rerun"}, ReceiptRefs: []string{"cloud-target:op-rerun:receipt"}, ObservedAt: pubNow, RecordedAt: pubNow}
		if err := repo.AppendEvidenceRecord(context.Background(), rec); err != nil {
			t.Fatal(err)
		}
	}
	withdraw()
	// The publication was refused; a new request under a new key is needed.
	body := requestBody(pubRelease)
	body["request_key"] = "req-2"
	if rec, _ := callJSON(t, srv, http.MethodPost, "/api/v1/deployments/dep-pub/publication/request", body); rec.Code != http.StatusOK {
		t.Fatalf("second request status = %d %s", rec.Code, rec.Body.String())
	}
	apply2 := func(reviewRef string) (*httptest.ResponseRecorder, string, map[string]any) {
		rec, body := callJSON(t, srv, http.MethodPost, "/api/v1/deployments/dep-pub/publication/apply", map[string]any{"request_key": "req-2", "review_ref": reviewRef, "plan_digest": "sha256:plan"})
		code, details := publicationErrorCode(body)
		return rec, code, details
	}
	cases := []struct {
		name    string
		review  *evidence.ReviewSnapshot
		getErr  error
		ref     string
		refusal string
	}{
		{"wrong review ref", snapshot("rr-1", evidence.ReviewApproved, pubRelease), nil, "rr-9", evidence.RefusalReviewMismatch},
		{"not approved", snapshot("rr-1", "agent_review", pubRelease), nil, "rr-1", evidence.RefusalReviewNotApproved},
		{"superseded (GOV-06)", snapshot("rr-1", evidence.ReviewSuperseded, pubRelease), nil, "rr-1", evidence.RefusalReviewRevoked},
		{"approved bytes replaced (GOV-03)", snapshot("rr-1", evidence.ReviewApproved, pubOther), nil, "rr-1", evidence.RefusalReviewMismatch},
		{"approved for another target (GOV-02)", snapshot("rr-1", evidence.ReviewApproved, pubRelease, "machine:other"), nil, "rr-1", evidence.RefusalReviewMismatch},
		{"owner unreachable", nil, errors.New("timeout"), "rr-1", evidence.RefusalReviewUnavailable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Each refusal leaves the publication refused; reset it to review_prepared
			// so the next case exercises the same decision path.
			stored, _ := repo.GetPublicationByRequestKey(context.Background(), "dep-pub", "req-2")
			stored.State, stored.Refusal = evidence.StateReviewPrepared, nil
			_ = repo.UpdatePublication(context.Background(), *stored)
			governor.reviews["rr-1"], governor.getErr = tc.review, tc.getErr
			rec, code, details := apply2(tc.ref)
			if rec.Code != http.StatusConflict || code != apierrors.CodePublicationRefused || details["refusal"] != tc.refusal {
				t.Fatalf("%s: %d %s", tc.name, rec.Code, rec.Body.String())
			}
			if activator.calls != 0 {
				t.Fatalf("%s: refusal must not activate (calls=%d)", tc.name, activator.calls)
			}
			refused, _ := repo.GetPublicationByRequestKey(context.Background(), "dep-pub", "req-2")
			if refused.State != evidence.StateRefused || refused.Refusal == nil || refused.Refusal.Code != tc.refusal {
				t.Fatalf("%s: stored publication = %+v", tc.name, refused)
			}
		})
	}
	_ = rec
}

// TestPublicationRecordsTheTargetReceiptAndReconcilesLostReplies
// [REQ:STC-P0-035] proves GOV-04 and GOV-05 / P17-A03: the published release
// and predecessor come from the target pointer; a lost reply (faultinject
// transport_reply) is reconciled by reading the target again, never by a
// second activation; a pointer naming another release fails the publication.
func TestPublicationRecordsTheTargetReceiptAndReconcilesLostReplies(t *testing.T) {
	srv, repo, governor, target, activator := newPublicationTestServer(t, "handlers-publication-publish")
	appendPassingEvidence(t, repo, pubRelease, "host:203.0.113.10", nil)
	governor.reviews["rr-1"] = snapshot("rr-1", evidence.ReviewApproved, pubRelease)
	if rec, _ := callJSON(t, srv, http.MethodPost, "/api/v1/deployments/dep-pub/publication/request", requestBody(pubRelease)); rec.Code != http.StatusOK {
		t.Fatalf("request status = %d %s", rec.Code, rec.Body.String())
	}
	registry := faultinject.New()
	testCtx := faultinject.WithRegistry(database.WithTestMode(context.Background()), registry)
	if err := registry.Arm(testCtx, faultinject.TransportReply, faultinject.Behaviour{Kind: faultinject.KindDropReply, Once: true}); err != nil {
		t.Fatal(err)
	}
	target.registry, target.ctx = registry, testCtx
	rec, decoded := callJSON(t, srv, http.MethodPost, "/api/v1/deployments/dep-pub/publication/apply", map[string]any{"request_key": "req-1", "review_ref": "rr-1", "plan_digest": "sha256:plan"})
	if rec.Code != http.StatusOK {
		t.Fatalf("apply status = %d %s", rec.Code, rec.Body.String())
	}
	pub := decoded["publication"].(map[string]any)
	if pub["state"] != evidence.StateActivating || pub["operation_id"] != "op-publish-publish:req-1" || activator.calls != 1 {
		t.Fatalf("lost reply must leave the publication activating without a second effect: %v calls=%d", pub, activator.calls)
	}
	// Replayed apply and GET both reconcile from the target; no new activation.
	rec, decoded = callJSON(t, srv, http.MethodPost, "/api/v1/deployments/dep-pub/publication/apply", map[string]any{"request_key": "req-1", "review_ref": "rr-1", "plan_digest": "sha256:plan"})
	pub = decoded["publication"].(map[string]any)
	if rec.Code != http.StatusOK || pub["state"] != evidence.StatePublished || activator.calls != 1 || target.reads != 2 {
		t.Fatalf("reconciled publication = %v calls=%d reads=%d", pub, activator.calls, target.reads)
	}
	if pub["activated_release_digest"] != pubRelease || pub["predecessor_release_digest"] != pubOther || pub["target_receipt"].(map[string]any)["receipt_digest"] == "" {
		t.Fatalf("publication must record the target's own release and predecessor: %v", pub)
	}
	rec, decoded = callJSON(t, srv, http.MethodGet, "/api/v1/deployments/dep-pub/publication?request_key=req-1", nil)
	if rec.Code != http.StatusOK || decoded["publication"].(map[string]any)["state"] != evidence.StatePublished {
		t.Fatalf("get = %d %s", rec.Code, rec.Body.String())
	}
	// The signed deployment receipt now binds the release, target and the
	// publishing operation; asking for another release is refused.
	if err := repo.UpdateDeploymentDeployResult(context.Background(), "dep-pub", json.RawMessage(`{"ok":true}`), true); err != nil {
		t.Fatal(err)
	}
	rec, decoded = callJSON(t, srv, http.MethodGet, "/api/v1/deployments/dep-pub/receipt?release_digest="+pubRelease, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("receipt = %d %s", rec.Code, rec.Body.String())
	}
	receipt := decoded["receipt"].(map[string]any)
	if receipt["producer_ref"] != domain.ReceiptProducerRef || receipt["release_digest"] != pubRelease || receipt["target_key"] != "host:203.0.113.10" || receipt["operation_id"] != "op-publish-publish:req-1" || receipt["signature"] == nil {
		t.Fatalf("receipt = %v", receipt)
	}
	rec, decoded = callJSON(t, srv, http.MethodGet, "/api/v1/deployments/dep-pub/receipt?release_digest="+pubOther, nil)
	if code, _ := publicationErrorCode(decoded); rec.Code != http.StatusUnprocessableEntity || code != apierrors.CodeReceiptInvalid {
		t.Fatalf("mismatched receipt request = %d %s", rec.Code, rec.Body.String())
	}

	// A target that reports a different active release fails the publication.
	body := requestBody(pubRelease)
	body["request_key"] = "req-mismatch"
	if rec, _ := callJSON(t, srv, http.MethodPost, "/api/v1/deployments/dep-pub/publication/request", body); rec.Code != http.StatusOK {
		t.Fatalf("request status = %d %s", rec.Code, rec.Body.String())
	}
	target.registry = nil
	target.receipt = evidence.TargetReceipt{ActiveRelease: pubOther}
	rec, decoded = callJSON(t, srv, http.MethodPost, "/api/v1/deployments/dep-pub/publication/apply", map[string]any{"request_key": "req-mismatch", "review_ref": "rr-1", "plan_digest": "sha256:plan"})
	pub = decoded["publication"].(map[string]any)
	if rec.Code != http.StatusOK || pub["state"] != evidence.StateFailed || pub["refusal"].(map[string]any)["code"] != evidence.RefusalTargetMismatch {
		t.Fatalf("mismatched target release = %d %v", rec.Code, pub)
	}
}
