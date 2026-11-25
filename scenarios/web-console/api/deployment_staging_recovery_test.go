package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
	dmreleasesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases"
	dmreleasesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases/releasesv1connect"
	s2cdeploymentsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments"
	s2cdeploymentsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-cloud/v1/deployments/deploymentsv1connect"
)

const (
	stagingRecoveryOperationID    = "release-op-fixture-release"
	stagingRecoveryDeploymentID   = "fixture-deployment"
	stagingRecoveryCurrentBundle  = "sha256:current-bundle"
	stagingRecoveryCurrentRelease = "sha256:current-release"
	stagingRecoveryLastGood       = "sha256:last-good-release"
	stagingRecoveryReview         = "review-fixture"
	stagingRecoveryKey            = "dm-recovery:release-op-fixture-release"
)

type stagingRecoveryDMFixture struct {
	dmreleasesconnect.UnimplementedReleasesServiceHandler
	operation *dmreleasesv1.ReleaseOperation
	getCalls  int
}

func (f *stagingRecoveryDMFixture) GetOperation(_ context.Context, request *connect.Request[dmreleasesv1.GetReleaseOperationRequest]) (*connect.Response[dmreleasesv1.GetReleaseOperationResponse], error) {
	f.getCalls++
	if request.Msg.GetOperationId() != f.operation.GetOperationId() {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&dmreleasesv1.GetReleaseOperationResponse{Operation: f.operation}), nil
}

type stagingRecoveryS2CFixture struct {
	s2cdeploymentsconnect.UnimplementedDeploymentsServiceHandler
	previewRef    string
	owner         *s2cdeploymentsv1.DeploymentRecoveryOperation
	previewCalls  int
	findCalls     int
	startCalls    int
	lastStart     *s2cdeploymentsv1.StartDeploymentRecoveryRequest
	allowedBearer string
	deniedBearer  string
}

func (f *stagingRecoveryS2CFixture) authorize(header http.Header) error {
	bearer := header.Get("Authorization")
	if f.deniedBearer != "" && bearer == f.deniedBearer {
		return connect.NewError(connect.CodePermissionDenied, nil)
	}
	if f.allowedBearer != "" && bearer != f.allowedBearer {
		return connect.NewError(connect.CodeUnauthenticated, nil)
	}
	return nil
}

func (f *stagingRecoveryS2CFixture) PreviewDeploymentRecovery(_ context.Context, req *connect.Request[s2cdeploymentsv1.PreviewDeploymentRecoveryRequest]) (*connect.Response[s2cdeploymentsv1.DeploymentRecoveryPreview], error) {
	f.previewCalls++
	if err := f.authorize(req.Header()); err != nil {
		return nil, err
	}
	if req.Msg.GetDeploymentId() != stagingRecoveryDeploymentID || req.Msg.GetAction() != "rollback" || req.Msg.GetExpectedCurrentBundleSha256() != stagingRecoveryCurrentBundle || req.Msg.GetExpectedCurrentReleaseDigest() != stagingRecoveryCurrentRelease || req.Msg.GetLastGoodReleaseDigest() != stagingRecoveryLastGood || req.Msg.GetReviewRef() != stagingRecoveryReview || req.Msg.GetDataCompatibility() != "compatible" || req.Msg.GetIdempotencyKey() != stagingRecoveryKey {
		return nil, connect.NewError(connect.CodeInvalidArgument, nil)
	}
	return connect.NewResponse(&s2cdeploymentsv1.DeploymentRecoveryPreview{DeploymentId: stagingRecoveryDeploymentID, Action: "rollback",
		CurrentBundleSha256: stagingRecoveryCurrentBundle, CurrentReleaseDigest: stagingRecoveryCurrentRelease,
		LastGoodReleaseDigest: stagingRecoveryLastGood, RepairBundleSha256: "sha256:last-good-bundle", ReviewRef: stagingRecoveryReview,
		DataCompatibility: "compatible", DataCompatibilityDeclared: true, RouteKind: "bundle_pipeline", ImpactSummary: "Activates the last-good release.",
		PreviewRef: f.previewRef, IdempotencyKey: stagingRecoveryKey}), nil
}

func (f *stagingRecoveryS2CFixture) FindDeploymentRecoveryOperation(_ context.Context, req *connect.Request[s2cdeploymentsv1.FindDeploymentRecoveryOperationRequest]) (*connect.Response[s2cdeploymentsv1.DeploymentRecoveryOperation], error) {
	f.findCalls++
	if err := f.authorize(req.Header()); err != nil {
		return nil, err
	}
	if f.owner == nil || req.Msg.GetDeploymentId() != f.owner.GetDeploymentId() || req.Msg.GetIdempotencyKey() != f.owner.GetIdempotencyKey() {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(f.owner), nil
}

func (f *stagingRecoveryS2CFixture) GetDeploymentRecoveryOperation(_ context.Context, req *connect.Request[s2cdeploymentsv1.GetDeploymentRecoveryOperationRequest]) (*connect.Response[s2cdeploymentsv1.DeploymentRecoveryOperation], error) {
	if err := f.authorize(req.Header()); err != nil {
		return nil, err
	}
	if f.owner == nil || req.Msg.GetDeploymentId() != f.owner.GetDeploymentId() || req.Msg.GetOperationId() != f.owner.GetOperationId() {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(f.owner), nil
}

func (f *stagingRecoveryS2CFixture) StartDeploymentRecovery(_ context.Context, req *connect.Request[s2cdeploymentsv1.StartDeploymentRecoveryRequest]) (*connect.Response[s2cdeploymentsv1.DeploymentRecoveryOperation], error) {
	f.startCalls++
	if err := f.authorize(req.Header()); err != nil {
		return nil, err
	}
	f.lastStart = req.Msg
	f.owner = &s2cdeploymentsv1.DeploymentRecoveryOperation{
		SchemaVersion: "scenario-to-cloud/deployment-recovery/v1", DeploymentId: req.Msg.GetDeploymentId(),
		OperationId: "s2c-recovery-fixture", IdempotencyKey: req.Msg.GetIdempotencyKey(), Action: req.Msg.GetAction(),
		Status: "running", ExpectedCurrentBundleSha256: req.Msg.GetExpectedCurrentBundleSha256(),
		LastGoodReleaseDigest: req.Msg.GetLastGoodReleaseDigest(), ReviewRef: req.Msg.GetReviewRef(),
		PreviewRef: req.Msg.GetPreviewRef(), DataCompatibility: req.Msg.GetDataCompatibility(),
	}
	return connect.NewResponse(f.owner), nil
}

func stagingRecoveryDMOperation(profile, state, verification, receipt, recoveryKey, ownerOperationID string) *dmreleasesv1.ReleaseOperation {
	return &dmreleasesv1.ReleaseOperation{
		OperationId: stagingRecoveryOperationID, ReleaseId: "fixture-release", ProfileId: profile,
		Status: dmreleasesv1.ReleaseOperationStatus_RELEASE_OPERATION_STATUS_FAILED,
		CloudStep: &dmreleasesv1.ReleaseCloudStep{
			State: state, DeploymentId: stagingRecoveryDeploymentID, OwnerOperationId: "s2c-cloud-fixture",
			VerificationReceiptJson: verification, ReceiptJson: receipt, RecoveryIdempotencyKey: recoveryKey,
			RecoveryOwnerOperationId: ownerOperationID,
		},
	}
}

func stagingRecoveryDMFixtureOperation(profile, state, verification string) *dmreleasesv1.ReleaseOperation {
	if verification == "" {
		verification = `{"healthy":false,"deployment_id":"fixture-deployment"}`
	}
	return stagingRecoveryDMOperation(profile, state, verification, `{"receipt":{"deployment_id":"fixture-deployment","bundle_sha256":"sha256:current-bundle","release_digest":"sha256:current-release"},"publication":{"review_ref":"review-fixture","release_digest":"sha256:current-release","activated_release_digest":"sha256:current-release","predecessor_release_digest":"sha256:last-good-release","state":"published"}}`, stagingRecoveryKey, "")
}

func stagingRecoveryHuman(request *http.Request, subject, bearer string) *http.Request {
	principal := identity.Principal{Kind: identity.ActorHuman, Subject: subject, Source: identity.SourceScenarioAuthenticator, Verified: true}
	request = request.WithContext(identity.WithPrincipal(request.Context(), principal))
	request.Host = "console.example"
	request.Header.Set("Origin", "https://console.example")
	request.Header.Set("Authorization", bearer)
	return request
}

func newStagingRecoveryFixture(t *testing.T, dm *stagingRecoveryDMFixture, s2c *stagingRecoveryS2CFixture) (*deploymentStagingFacade, *httptest.Server, *httptest.Server) {
	t.Helper()
	dmPath, dmHandler := dmreleasesconnect.NewReleasesServiceHandler(dm)
	dmRouter := mux.NewRouter()
	dmRouter.PathPrefix(dmPath).Handler(dmHandler)
	dmServer := httptest.NewServer(dmRouter)
	s2cPath, s2cHandler := s2cdeploymentsconnect.NewDeploymentsServiceHandler(s2c)
	s2cRouter := mux.NewRouter()
	s2cRouter.PathPrefix(s2cPath).Handler(s2cHandler)
	s2cServer := httptest.NewServer(s2cRouter)
	facade := newDeploymentStagingFacade(func(context.Context) (string, error) { return dmServer.URL, nil }, func(context.Context) (string, error) { return s2cServer.URL, nil })
	t.Cleanup(dmServer.Close)
	t.Cleanup(s2cServer.Close)
	return facade, dmServer, s2cServer
}

func TestRecoveryPreviewRequiresVerifiedHumanAndForwardsThatIdentity(t *testing.T) {
	dm := &stagingRecoveryDMFixture{operation: stagingRecoveryDMFixtureOperation("web-console/staging", "recovery_decision_required", "")}
	s2c := &stagingRecoveryS2CFixture{previewRef: "preview-current", allowedBearer: "Bearer verified-human"}
	facade, _, _ := newStagingRecoveryFixture(t, dm, s2c)
	request := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/staging/operations/"+stagingRecoveryOperationID+"/recovery/preview", strings.NewReader(`{"compatibility_confirmed":true}`))
	request = stagingRecoveryHuman(request, "operator@example.test", "Bearer verified-human")
	request = mux.SetURLVars(request, map[string]string{"operation_id": stagingRecoveryOperationID})
	response := httptest.NewRecorder()
	facade.handleRecoveryPreview(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"preview_ref":"preview-current"`) || !strings.Contains(response.Body.String(), stagingRecoveryLastGood) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if dm.getCalls != 1 || s2c.previewCalls != 1 || s2c.startCalls != 0 || s2c.findCalls != 1 {
		t.Fatalf("DM gets=%d previews=%d finds=%d starts=%d", dm.getCalls, s2c.previewCalls, s2c.findCalls, s2c.startCalls)
	}
	noHuman := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/staging/operations/"+stagingRecoveryOperationID+"/recovery/preview", strings.NewReader(`{"compatibility_confirmed":true}`))
	noHuman.Host = "console.example"
	noHuman.Header.Set("Origin", "https://console.example")
	noHuman.Header.Set("Authorization", "Bearer forged-human")
	noHuman = mux.SetURLVars(noHuman, map[string]string{"operation_id": stagingRecoveryOperationID})
	denied := httptest.NewRecorder()
	facade.handleRecoveryPreview(denied, noHuman)
	if denied.Code != http.StatusUnauthorized || dm.getCalls != 1 || s2c.previewCalls != 1 {
		t.Fatalf("unauthenticated preview status=%d DM=%d S2C=%d body=%s", denied.Code, dm.getCalls, s2c.previewCalls, denied.Body.String())
	}
}

func TestRecoveryRejectsMismatchedProfileVerificationAndBrowserTarget(t *testing.T) {
	tests := []struct {
		name       string
		operation  *dmreleasesv1.ReleaseOperation
		body       string
		wantStatus int
	}{
		{name: "wrong profile", operation: stagingRecoveryDMFixtureOperation("another/profile", "recovery_decision_required", ""), body: `{"compatibility_confirmed":true}`, wantStatus: http.StatusConflict},
		{name: "wrong failed deployment", operation: stagingRecoveryDMFixtureOperation("web-console/staging", "recovery_decision_required", `{"healthy":false,"deployment_id":"other-deployment"}`), body: `{"compatibility_confirmed":true}`, wantStatus: http.StatusConflict},
		{name: "not recovery decision", operation: stagingRecoveryDMFixtureOperation("web-console/staging", "verification_passed", ""), body: `{"compatibility_confirmed":true}`, wantStatus: http.StatusConflict},
		{name: "browser supplied target", operation: stagingRecoveryDMFixtureOperation("web-console/staging", "recovery_decision_required", ""), body: `{"compatibility_confirmed":true,"deployment_id":"attacker-target"}`, wantStatus: http.StatusBadRequest},
		{name: "compatibility not acknowledged", operation: stagingRecoveryDMFixtureOperation("web-console/staging", "recovery_decision_required", ""), body: `{"compatibility_confirmed":false}`, wantStatus: http.StatusPreconditionRequired},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dm := &stagingRecoveryDMFixture{operation: test.operation}
			s2c := &stagingRecoveryS2CFixture{previewRef: "preview-current", allowedBearer: "Bearer verified-human"}
			facade, _, _ := newStagingRecoveryFixture(t, dm, s2c)
			request := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/staging/operations/"+stagingRecoveryOperationID+"/recovery/preview", strings.NewReader(test.body))
			request = stagingRecoveryHuman(request, "operator@example.test", "Bearer verified-human")
			request = mux.SetURLVars(request, map[string]string{"operation_id": stagingRecoveryOperationID})
			response := httptest.NewRecorder()
			facade.handleRecoveryPreview(response, request)
			if response.Code != test.wantStatus || s2c.previewCalls != 0 || s2c.startCalls != 0 {
				t.Fatalf("status=%d want=%d S2C previews=%d starts=%d body=%s", response.Code, test.wantStatus, s2c.previewCalls, s2c.startCalls, response.Body.String())
			}
		})
	}
}

func TestRecoveryConfirmRepreviewsAndStartsOnlyAfterExactHumanConfirmation(t *testing.T) {
	dm := &stagingRecoveryDMFixture{operation: stagingRecoveryDMFixtureOperation("web-console/staging", "recovery_decision_required", "")}
	s2c := &stagingRecoveryS2CFixture{previewRef: "preview-current", allowedBearer: "Bearer verified-human"}
	facade, _, _ := newStagingRecoveryFixture(t, dm, s2c)
	router := mux.NewRouter()
	router.HandleFunc("/api/v1/deployments/staging/operations/{operation_id}/recovery/confirm", facade.handleRecoveryConfirm).Methods(http.MethodPost)
	confirm := func(ref string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/staging/operations/"+stagingRecoveryOperationID+"/recovery/confirm", strings.NewReader(`{"confirmed":true,"confirmed_preview_ref":"`+ref+`"}`))
		request = stagingRecoveryHuman(request, "operator@example.test", "Bearer verified-human")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}
	response := confirm("preview-current")
	if response.Code != http.StatusAccepted || s2c.startCalls != 1 {
		t.Fatalf("first confirmation status=%d starts=%d body=%s", response.Code, s2c.startCalls, response.Body.String())
	}
	start := s2c.lastStart
	if start == nil || start.GetDeploymentId() != stagingRecoveryDeploymentID || start.GetExpectedCurrentBundleSha256() != stagingRecoveryCurrentBundle || start.GetLastGoodReleaseDigest() != stagingRecoveryLastGood || start.GetReviewRef() != stagingRecoveryReview || start.GetPreviewRef() != "preview-current" || start.GetIdempotencyKey() != stagingRecoveryKey || start.GetDataCompatibility() != "compatible" {
		t.Fatalf("generated S2C start request did not bind exact DM identity: %+v", start)
	}
	response = confirm("preview-current")
	if response.Code != http.StatusAccepted || s2c.startCalls != 1 || !strings.Contains(response.Body.String(), "s2c-recovery-fixture") {
		t.Fatalf("duplicate confirmation status=%d starts=%d body=%s", response.Code, s2c.startCalls, response.Body.String())
	}
}

func TestRecoveryConfirmRefusesStalePreviewAndS2CAuthorization(t *testing.T) {
	t.Run("stale preview", func(t *testing.T) {
		dm := &stagingRecoveryDMFixture{operation: stagingRecoveryDMFixtureOperation("web-console/staging", "recovery_decision_required", "")}
		s2c := &stagingRecoveryS2CFixture{previewRef: "preview-new", allowedBearer: "Bearer verified-human"}
		facade, _, _ := newStagingRecoveryFixture(t, dm, s2c)
		request := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/staging/operations/"+stagingRecoveryOperationID+"/recovery/confirm", strings.NewReader(`{"confirmed":true,"confirmed_preview_ref":"preview-old"}`))
		request = stagingRecoveryHuman(request, "operator@example.test", "Bearer verified-human")
		request = mux.SetURLVars(request, map[string]string{"operation_id": stagingRecoveryOperationID})
		response := httptest.NewRecorder()
		facade.handleRecoveryConfirm(response, request)
		if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "stale_preview") || s2c.previewCalls != 1 || s2c.startCalls != 0 {
			t.Fatalf("status=%d previews=%d starts=%d body=%s", response.Code, s2c.previewCalls, s2c.startCalls, response.Body.String())
		}
	})
	t.Run("target grant denied", func(t *testing.T) {
		dm := &stagingRecoveryDMFixture{operation: stagingRecoveryDMFixtureOperation("web-console/staging", "recovery_decision_required", "")}
		s2c := &stagingRecoveryS2CFixture{previewRef: "preview-current", allowedBearer: "Bearer authorized-human", deniedBearer: "Bearer wrong-human"}
		facade, _, _ := newStagingRecoveryFixture(t, dm, s2c)
		request := httptest.NewRequest(http.MethodPost, "/api/v1/deployments/staging/operations/"+stagingRecoveryOperationID+"/recovery/preview", strings.NewReader(`{"compatibility_confirmed":true}`))
		request = stagingRecoveryHuman(request, "another-operator@example.test", "Bearer wrong-human")
		request = mux.SetURLVars(request, map[string]string{"operation_id": stagingRecoveryOperationID})
		response := httptest.NewRecorder()
		facade.handleRecoveryPreview(response, request)
		if response.Code != http.StatusForbidden || s2c.previewCalls != 0 || s2c.startCalls != 0 {
			t.Fatalf("status=%d previews=%d starts=%d body=%s", response.Code, s2c.previewCalls, s2c.startCalls, response.Body.String())
		}
	})
}

func TestRecoveryStatusReattachesByExactStableKeyAfterReload(t *testing.T) {
	dm := &stagingRecoveryDMFixture{operation: stagingRecoveryDMFixtureOperation("web-console/staging", "recovery_decision_required", "")}
	s2c := &stagingRecoveryS2CFixture{previewRef: "preview-current", allowedBearer: "Bearer verified-human"}
	s2c.owner = &s2cdeploymentsv1.DeploymentRecoveryOperation{
		DeploymentId: stagingRecoveryDeploymentID, OperationId: "s2c-recovery-fixture", IdempotencyKey: stagingRecoveryKey,
		Action: "rollback", Status: "running", ExpectedCurrentBundleSha256: stagingRecoveryCurrentBundle,
		LastGoodReleaseDigest: stagingRecoveryLastGood, ReviewRef: stagingRecoveryReview, PreviewRef: "preview-current", DataCompatibility: "compatible",
	}
	facade, _, _ := newStagingRecoveryFixture(t, dm, s2c)
	request := httptest.NewRequest(http.MethodGet, "/api/v1/deployments/staging/operations/"+stagingRecoveryOperationID+"/recovery", nil)
	request = stagingRecoveryHuman(request, "operator@example.test", "Bearer verified-human")
	request = mux.SetURLVars(request, map[string]string{"operation_id": stagingRecoveryOperationID})
	response := httptest.NewRecorder()
	facade.handleRecoveryStatus(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "s2c-recovery-fixture") || s2c.findCalls != 1 || s2c.startCalls != 0 {
		t.Fatalf("status=%d finds=%d starts=%d body=%s", response.Code, s2c.findCalls, s2c.startCalls, response.Body.String())
	}
}
