package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	releasesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases"
	releasesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases/releasesv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestStagingProfileRemainsBlockedWithoutExternalInputs(t *testing.T) {
	facade := newDeploymentStagingFacade(func(context.Context) (string, error) {
		t.Fatal("profile status must not contact Deployment Manager")
		return "", nil
	})
	rr := httptest.NewRecorder()
	facade.handleProfile(rr, httptest.NewRequest(http.MethodGet, "/api/v1/deployments/staging", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body struct {
		Profile   stagingProfileView `json:"profile"`
		Readiness stagingReadiness   `json:"readiness"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Readiness.State != "blocked" || body.Profile.Domain != "staging.vrooli.com" || body.Profile.CapMinorUnits != 709 {
		t.Fatalf("response = %+v", body)
	}
}

func TestEvaluateStagingAdmissionBlocksMissingQuoteAndInstance(t *testing.T) {
	readiness := evaluateStagingAdmission(stagingAdmissionFacts{}, "USD", 709)
	if readiness.State != "blocked" || len(readiness.Reasons) != 2 {
		t.Fatalf("readiness = %+v, want blocked for quote and target", readiness)
	}
}

func TestEvaluateStagingAdmissionRefusesOverCapAndCurrencyMismatch(t *testing.T) {
	for name, facts := range map[string]stagingAdmissionFacts{
		"over cap":          {QuoteReference: "fixture-quote", InstanceReference: "fixture-instance", Currency: "USD", MonthlyMinorUnits: 710},
		"currency mismatch": {QuoteReference: "fixture-quote", InstanceReference: "fixture-instance", Currency: "EUR", MonthlyMinorUnits: 100},
	} {
		t.Run(name, func(t *testing.T) {
			readiness := evaluateStagingAdmission(facts, "USD", 709)
			if readiness.State != "blocked" || len(readiness.Reasons) == 0 {
				t.Fatalf("readiness = %+v, expected refusal", readiness)
			}
		})
	}
}

type stagingDMFixture struct {
	releasesconnect.UnimplementedReleasesServiceHandler
	listCalls, operationCalls int
	resumeCalls               int
	resumeStatus              releasesv1.ReleaseOperationStatus
	resumeStep                string
	operationProfileID        string
	operationID               string
	releaseID                 string
}

func (f *stagingDMFixture) List(_ context.Context, req *connect.Request[releasesv1.ListReleasesRequest]) (*connect.Response[releasesv1.ListReleasesResponse], error) {
	f.listCalls++
	if req.Msg.GetProfileId() != "web-console/staging" {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	return connect.NewResponse(&releasesv1.ListReleasesResponse{Releases: []*releasesv1.ReleaseView{{ReleaseId: "fixture-release", ProfileId: "web-console/staging"}}}), nil
}
func (f *stagingDMFixture) GetOperation(_ context.Context, req *connect.Request[releasesv1.GetReleaseOperationRequest]) (*connect.Response[releasesv1.GetReleaseOperationResponse], error) {
	f.operationCalls++
	if req.Msg.GetOperationId() != "release-op-fixture-release" {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	status := f.resumeStatus
	if status == releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_UNSPECIFIED {
		status = releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_AMBIGUOUS
	}
	step := f.resumeStep
	if step == "" {
		step = "blocked"
	}
	profileID := f.operationProfileID
	if profileID == "" {
		profileID = "web-console/staging"
	}
	operationID := f.operationID
	if operationID == "" {
		operationID = req.Msg.GetOperationId()
	}
	releaseID := f.releaseID
	if releaseID == "" {
		releaseID = "fixture-release"
	}
	return connect.NewResponse(&releasesv1.GetReleaseOperationResponse{Operation: &releasesv1.ReleaseOperation{
		OperationId: operationID, ReleaseId: releaseID, ProfileId: profileID,
		Status: status, ActiveStage: "cloud_handoff", Error: "fixture-only ambiguous standing",
		CloudSpendQuoteIdempotencyKey: "fixture-quote-key", CloudComputeManagerInstanceId: "fixture-cm-instance",
		UpdatedAt: timestamppb.Now(), CloudStep: &releasesv1.ReleaseCloudStep{State: step, Reason: "fixture-only verification failed", OwnerOperationId: "s2c-fixture", DeploymentId: "fixture-deployment", RequestKey: "fixture-request-key", RequestDigest: "sha256:fixture", ReceiptJson: `{"outcome":"failed"}`, VerificationReceiptJson: `{"healthy":false,"checked_at":"fixture-only"}`},
	}}), nil
}

func TestStagingReadbackRejectsOperationFromOtherProfile(t *testing.T) {
	fixture := &stagingDMFixture{operationProfileID: "another/profile"}
	path, handler := releasesconnect.NewReleasesServiceHandler(fixture)
	upstreamRouter := mux.NewRouter()
	upstreamRouter.PathPrefix(path).Handler(handler)
	upstream := httptest.NewServer(upstreamRouter)
	defer upstream.Close()
	facade := newDeploymentStagingFacade(func(context.Context) (string, error) { return upstream.URL, nil })
	req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments/staging/operations/release-op-fixture-release", nil)
	req = mux.SetURLVars(req, map[string]string{"operation_id": "release-op-fixture-release"})
	rr := httptest.NewRecorder()
	facade.handleOperation(rr, req)
	if rr.Code != http.StatusNotFound || strings.Contains(rr.Body.String(), "another/profile") || fixture.operationCalls != 1 {
		t.Fatalf("status=%d calls=%d body=%s", rr.Code, fixture.operationCalls, rr.Body.String())
	}
}

func TestStagingReadbackRejectsOperationAndReleaseIdentityMismatch(t *testing.T) {
	for name, fixture := range map[string]*stagingDMFixture{
		"operation id": {operationID: "release-op-other"},
		"release id":   {releaseID: "other-release"},
	} {
		t.Run(name, func(t *testing.T) {
			path, handler := releasesconnect.NewReleasesServiceHandler(fixture)
			upstreamRouter := mux.NewRouter()
			upstreamRouter.PathPrefix(path).Handler(handler)
			upstream := httptest.NewServer(upstreamRouter)
			defer upstream.Close()
			facade := newDeploymentStagingFacade(func(context.Context) (string, error) { return upstream.URL, nil })
			req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments/staging/operations/release-op-fixture-release", nil)
			req = mux.SetURLVars(req, map[string]string{"operation_id": "release-op-fixture-release"})
			rr := httptest.NewRecorder()
			facade.handleOperation(rr, req)
			if rr.Code != http.StatusNotFound || fixture.operationCalls != 1 {
				t.Fatalf("status=%d calls=%d body=%s", rr.Code, fixture.operationCalls, rr.Body.String())
			}
		})
	}
}

func TestStagingOperationListHidesMismatchedOperationBinding(t *testing.T) {
	for name, fixture := range map[string]*stagingDMFixture{
		"operation id": {operationID: "release-op-other"},
		"release id":   {releaseID: "other-release"},
	} {
		t.Run(name, func(t *testing.T) {
			path, handler := releasesconnect.NewReleasesServiceHandler(fixture)
			upstreamRouter := mux.NewRouter()
			upstreamRouter.PathPrefix(path).Handler(handler)
			upstream := httptest.NewServer(upstreamRouter)
			defer upstream.Close()
			facade := newDeploymentStagingFacade(func(context.Context) (string, error) { return upstream.URL, nil })
			rr := httptest.NewRecorder()
			facade.handleOperations(rr, httptest.NewRequest(http.MethodGet, "/api/v1/deployments/staging/operations", nil))
			if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), `"operations":[]`) || fixture.operationCalls != 1 {
				t.Fatalf("status=%d calls=%d body=%s", rr.Code, fixture.operationCalls, rr.Body.String())
			}
		})
	}
}
func (f *stagingDMFixture) ResumeCloudOperation(_ context.Context, req *connect.Request[releasesv1.ResumeCloudOperationRequest]) (*connect.Response[releasesv1.GetReleaseOperationResponse], error) {
	f.resumeCalls++
	if req.Msg.GetOperationId() != "release-op-fixture-release" {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&releasesv1.GetReleaseOperationResponse{Operation: &releasesv1.ReleaseOperation{
		OperationId: req.Msg.GetOperationId(), ReleaseId: "fixture-release", ProfileId: "web-console/staging",
		Status: releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_RUNNING, ActiveStage: "reconcile-cloud", UpdatedAt: timestamppb.Now(),
	}}), nil
}

func TestStagingFacadeReadsDurableProgressIdempotently(t *testing.T) {
	fixture := &stagingDMFixture{}
	path, handler := releasesconnect.NewReleasesServiceHandler(fixture)
	upstreamRouter := mux.NewRouter()
	upstreamRouter.PathPrefix(path).Handler(handler)
	upstream := httptest.NewServer(upstreamRouter)
	defer upstream.Close()
	facade := newDeploymentStagingFacade(func(context.Context) (string, error) { return upstream.URL, nil })
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/deployments/staging/operations", facade.handleOperations).Methods(http.MethodGet)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/deployments/staging/operations", nil)
		req.Header.Set("Authorization", "Bearer operator-fixture")
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("iteration %d status=%d body=%s", i, rr.Code, rr.Body.String())
		}
		if got := rr.Body.String(); !containsAll(got, "AMBIGUOUS", "cloud_handoff", "fixture-only verification failed", "s2c-fixture", "fixture-quote-key", "fixture-cm-instance", "fixture-deployment", "sha256:fixture", "verification_receipt_json", `\"healthy\":false`) {
			t.Fatalf("durable operation fields missing: %s", got)
		}
	}
	if fixture.listCalls != 2 || fixture.operationCalls != 2 || fixture.resumeCalls != 0 {
		t.Fatalf("calls list=%d operation=%d resume=%d; readback must be read-only", fixture.listCalls, fixture.operationCalls, fixture.resumeCalls)
	}
}

func TestStagingReadbackDistinguishesRecoveryDecisionRequired(t *testing.T) {
	op := &releasesv1.ReleaseOperation{OperationId: "fixture-op", ReleaseId: "fixture-release", ProfileId: "web-console/staging",
		Status: releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_FAILED, CloudStep: &releasesv1.ReleaseCloudStep{State: "recovery_decision_required", Reason: "fixture-only verification failed", VerificationReceiptJson: `{"healthy":false}`}}
	view := stagingOperationFromProto(op)
	if view.CloudStep == nil || view.CloudStep.State != "recovery_decision_required" || view.CloudStep.VerificationReceiptJSON != `{"healthy":false}` || view.ResumeAvailable {
		t.Fatalf("recovery status was not preserved as read-only standing: %+v", view)
	}
	if !strings.Contains(view.ResumeReason, "beyond the safe resume boundary") {
		t.Fatalf("recovery must not imply resume is safe: %+v", view)
	}
}

func TestCertificateReadbackUsesOnlyValidDMReceiptAndKeepsRenewalBlocked(t *testing.T) {
	now := time.Now().UTC()
	receipt, err := json.Marshal(map[string]any{"deployment_id": "fixture-deployment", "observed_release_digest": "sha256:" + strings.Repeat("a", 64), "observed_at": now,
		"tls_status": "healthy", "tls_reason": "certificate_valid", "tls_evidence_sha256": "sha256:" + strings.Repeat("c", 64),
		"certificate_lifecycle_state": "valid", "certificate_lifecycle_reason": "renewal_state_not_reported_by_tls_snapshot", "certificate_not_after": now.Add(24 * time.Hour), "certificate_observed_at": now,
		"certificate_renewal_state": "blocked", "certificate_evidence_sha256": "sha256:" + strings.Repeat("b", 64)})
	if err != nil {
		t.Fatal(err)
	}
	view := stagingOperationFromProto(&releasesv1.ReleaseOperation{OperationId: "fixture-operation", ProfileId: "web-console/staging", CloudStep: &releasesv1.ReleaseCloudStep{OperationId: "fixture-operation", DeploymentId: "fixture-deployment", VerificationReceiptJson: string(receipt)}})
	if view.Certificate.State != "valid" || view.Certificate.NotAfter == "" || view.Certificate.SampledAt == "" || view.Certificate.Renewal != "blocked" || view.Certificate.RenewalReason != "renewal_state_not_reported_by_tls_snapshot" {
		t.Fatalf("certificate lifecycle not faithfully projected: %+v", view.Certificate)
	}
	if view.TLS.State != "healthy" || view.TLS.Reason != "certificate_valid" || view.TLS.SampledAt == "" {
		t.Fatalf("generic TLS check not projected independently: %+v", view.TLS)
	}
	staleAt := now.Add(-3 * time.Minute)
	stale, _ := json.Marshal(map[string]any{"deployment_id": "fixture-deployment", "observed_release_digest": "sha256:" + strings.Repeat("a", 64), "observed_at": staleAt,
		"certificate_lifecycle_state": "valid", "certificate_not_after": now.Add(time.Hour), "certificate_observed_at": staleAt, "certificate_renewal_state": "blocked", "certificate_evidence_sha256": "sha256:" + strings.Repeat("b", 64)})
	sampleMismatch, _ := json.Marshal(map[string]any{"deployment_id": "fixture-deployment", "observed_release_digest": "sha256:" + strings.Repeat("a", 64), "observed_at": now.Add(-time.Second),
		"certificate_lifecycle_state": "valid", "certificate_not_after": now.Add(time.Hour), "certificate_observed_at": now, "certificate_renewal_state": "blocked", "certificate_evidence_sha256": "sha256:" + strings.Repeat("b", 64)})
	future, _ := json.Marshal(map[string]any{"deployment_id": "fixture-deployment", "observed_release_digest": "sha256:" + strings.Repeat("a", 64), "observed_at": now.Add(time.Hour),
		"certificate_lifecycle_state": "valid", "certificate_not_after": now.Add(48 * time.Hour), "certificate_observed_at": now.Add(time.Hour), "certificate_renewal_state": "renewed", "certificate_evidence_sha256": "sha256:" + strings.Repeat("b", 64)})
	for _, raw := range []string{`{"certificate_lifecycle_state":"valid"}`, `{bad json}`, string(future), string(stale), string(sampleMismatch)} {
		got := certificateFromReceipt(raw, "fixture-operation", "fixture-operation", "fixture-deployment")
		if got.State != "unavailable" || got.Renewal != "blocked" || got.Reason == "" {
			t.Errorf("invalid receipt %q projected as %+v", raw, got)
		}
	}
	if got := certificateFromReceipt(string(receipt), "fixture-operation", "other-operation", "fixture-deployment"); got.State != "unavailable" || !strings.Contains(got.Reason, "identity") {
		t.Errorf("mismatched operation projected as %+v", got)
	}
	if got := certificateFromReceipt(string(receipt), "fixture-operation", "fixture-operation", "other-deployment"); got.State != "unavailable" || !strings.Contains(got.Reason, "identity") {
		t.Errorf("mismatched deployment projected as %+v", got)
	}
	blocked, _ := json.Marshal(map[string]any{"deployment_id": "fixture-deployment", "certificate_lifecycle_state": "blocked", "certificate_lifecycle_reason": "renewal_state_not_reported_by_tls_snapshot"})
	if got := certificateFromReceipt(string(blocked), "fixture-operation", "fixture-operation", "fixture-deployment"); got.State != "unavailable" || got.Reason != "renewal_state_not_reported_by_tls_snapshot" {
		t.Errorf("backend blocked reason was not preserved: %+v", got)
	}
	if got := tlsFromReceipt(string(blocked), "fixture-operation", "fixture-operation", "fixture-deployment"); got.State != "unavailable" {
		t.Errorf("missing generic TLS facts should remain unavailable independently of certificate state: %+v", got)
	}
}

func TestStagingStartStaysBlockedWithoutProfileOwnedInputs(t *testing.T) {
	facade := newDeploymentStagingFacade(func(context.Context) (string, error) {
		t.Fatal("blocked start must not contact Deployment Manager")
		return "", nil
	})
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/staging/start", strings.NewReader(`{"cloud_compute_manager_instance_id":"attacker-chosen"}`))
		req.Host = "console.example"
		req.Header.Set("Origin", "https://console.example")
		rr := httptest.NewRecorder()
		facade.handleStart(rr, req)
		if rr.Code != http.StatusOK || !containsAll(rr.Body.String(), `"state":"blocked"`, `"deployment_manager_called":false`, "readiness review", "Compute Manager quote") {
			t.Fatalf("iteration %d status=%d body=%s", i, rr.Code, rr.Body.String())
		}
	}
}

func TestStagingStartRequiresSameOrigin(t *testing.T) {
	facade := newDeploymentStagingFacade(func(context.Context) (string, error) {
		t.Fatal("cross-origin request must not resolve Deployment Manager")
		return "", nil
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/staging/start", nil)
	req.Host = "console.example"
	req.Header.Set("Origin", "https://attacker.example")
	rr := httptest.NewRecorder()
	facade.handleStart(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}

func TestStagingResumeReReadsAndDelegatesOnlyEligibleOperation(t *testing.T) {
	fixture := &stagingDMFixture{resumeStatus: releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_FAILED, resumeStep: "operation_admitted"}
	path, handler := releasesconnect.NewReleasesServiceHandler(fixture)
	upstreamRouter := mux.NewRouter()
	upstreamRouter.PathPrefix(path).Handler(handler)
	upstream := httptest.NewServer(upstreamRouter)
	defer upstream.Close()
	facade := newDeploymentStagingFacade(func(context.Context) (string, error) { return upstream.URL, nil })
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/deployments/staging/operations/{operation_id}/resume", facade.handleResume).Methods(http.MethodPost)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/staging/operations/release-op-fixture-release/resume", nil)
	req.Host = "console.example"
	req.Header.Set("Origin", "https://console.example")
	req = mux.SetURLVars(req, map[string]string{"operation_id": "release-op-fixture-release"})
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted || !strings.Contains(rr.Body.String(), `"status":"RELEASE_OPERATION_STATUS_RUNNING"`) || fixture.operationCalls != 1 || fixture.resumeCalls != 1 {
		t.Fatalf("status=%d get=%d resume=%d body=%s", rr.Code, fixture.operationCalls, fixture.resumeCalls, rr.Body.String())
	}
}

func TestStagingResumeRefusesUnsafeStandingBeforeResumeRPC(t *testing.T) {
	fixture := &stagingDMFixture{resumeStatus: releasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_FAILED, resumeStep: "downstream_started"}
	path, handler := releasesconnect.NewReleasesServiceHandler(fixture)
	upstreamRouter := mux.NewRouter()
	upstreamRouter.PathPrefix(path).Handler(handler)
	upstream := httptest.NewServer(upstreamRouter)
	defer upstream.Close()
	facade := newDeploymentStagingFacade(func(context.Context) (string, error) { return upstream.URL, nil })
	req := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/staging/operations/release-op-fixture-release/resume", nil)
	req.Host = "console.example"
	req.Header.Set("Origin", "https://console.example")
	req = mux.SetURLVars(req, map[string]string{"operation_id": "release-op-fixture-release"})
	rr := httptest.NewRecorder()
	facade.handleResume(rr, req)
	if rr.Code != http.StatusConflict || fixture.operationCalls != 1 || fixture.resumeCalls != 0 || !strings.Contains(rr.Body.String(), "safe resume boundary") {
		t.Fatalf("status=%d get=%d resume=%d body=%s", rr.Code, fixture.operationCalls, fixture.resumeCalls, rr.Body.String())
	}
}

func containsAll(text string, values ...string) bool {
	for _, value := range values {
		if !strings.Contains(text, value) {
			return false
		}
	}
	return true
}
