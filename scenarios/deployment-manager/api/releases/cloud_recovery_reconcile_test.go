package releases

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

type cloudRecoveryReconcileRepo struct {
	*mockRepo
	operation *Operation
	step      *CloudStep
	fence     uint64
}

func (r *cloudRecoveryReconcileRepo) GetOperation(_ context.Context, id string) (*Operation, error) {
	if r.operation == nil || r.operation.ID != id {
		return nil, context.Canceled
	}
	return r.operation, nil
}
func (r *cloudRecoveryReconcileRepo) GetCloudStep(_ context.Context, id string) (*CloudStep, error) {
	if r.step == nil || r.step.OperationID != id {
		return nil, context.Canceled
	}
	copy := *r.step
	return &copy, nil
}
func (r *cloudRecoveryReconcileRepo) RecordCloudStep(_ context.Context, step CloudStep) error {
	copy := step
	r.step = &copy
	return nil
}
func (r *cloudRecoveryReconcileRepo) AcquireOperationLease(_ context.Context, id, owner string, _ time.Duration) (*OperationLease, bool, error) {
	r.fence++
	return &OperationLease{OperationID: id, Owner: owner, Fence: r.fence, ExpiresAt: time.Now().Add(time.Minute)}, true, nil
}
func (r *cloudRecoveryReconcileRepo) ReleaseOperationLease(context.Context, *OperationLease) error {
	return nil
}
func (r *cloudRecoveryReconcileRepo) UpdateOperationFenced(_ context.Context, _ *OperationLease, status, stage, reason string) error {
	r.operation.Status, r.operation.ActiveStage, r.operation.Error = status, stage, reason
	return nil
}
func (r *cloudRecoveryReconcileRepo) AppendOperationEvent(context.Context, *OperationLease, string, string, string) error {
	return nil
}
func (r *cloudRecoveryReconcileRepo) RecordRecoveryReceipt(_ context.Context, receipt RecoveryReceipt) error {
	for _, prior := range r.recovery {
		if prior.ReleaseID == receipt.ReleaseID && prior.ExternalReceipt == receipt.ExternalReceipt {
			return nil
		}
	}
	r.recovery = append(r.recovery, receipt)
	return nil
}

type cloudRecoveryReadbackFake struct {
	findResult *CloudRecoveryOperation
	getResult  *CloudRecoveryOperation
	findCalls  int
	getCalls   int
	getID      string
	key        string
}

func (f *cloudRecoveryReadbackFake) FindCloudRecoveryOperation(_ context.Context, deploymentID, key string) (*CloudRecoveryOperation, error) {
	f.findCalls++
	f.key = key
	if f.findResult == nil {
		return nil, nil
	}
	copy := *f.findResult
	return &copy, nil
}
func (f *cloudRecoveryReadbackFake) GetCloudRecoveryOperation(_ context.Context, deploymentID, id string) (*CloudRecoveryOperation, error) {
	f.getCalls++
	f.getID = id
	if f.getResult == nil {
		return nil, nil
	}
	copy := *f.getResult
	return &copy, nil
}

type cloudRecoveryReconcileOrchestrator struct {
	*mockOrch
	*cloudRecoveryReadbackFake
}

func validCloudRecoveryReconcileFixture(t *testing.T) (*cloudRecoveryReconcileRepo, string, *CloudRecoveryOperation) {
	t.Helper()
	releaseID := "release-human-cloud-rollback"
	operationID := operationIDForRelease(releaseID)
	cloudReceipt := json.RawMessage(`{"receipt":{"deployment_id":"dep-human","bundle_sha256":"sha256:current-bundle","release_digest":"sha256:current-release"},"publication":{"review_ref":"review-current","release_digest":"sha256:current-release","activated_release_digest":"sha256:current-release","predecessor_release_digest":"sha256:last-good","state":"published"}}`)
	repo := &cloudRecoveryReconcileRepo{
		mockRepo:  newMockRepo(),
		operation: &Operation{ID: operationID, ReleaseID: releaseID, ProfileID: "web-console/staging", Status: OperationFailed, ActiveStage: "cloud_health"},
		step: &CloudStep{
			OperationID: operationID, RequestKey: "dm-cloud:" + operationID, RequestDigest: "sha256:request", State: "recovery_decision_required",
			DeploymentID: "dep-human", Receipt: cloudReceipt, VerificationReceipt: json.RawMessage(`{"healthy":false,"deployment_id":"dep-human","reason_code":"release_mismatch"}`),
			RecoveryIdempotencyKey: CloudRecoveryIdempotencyKey(operationID), RecoveryState: "awaiting_human_recovery", RecoveryReason: "health verification failed",
		},
	}
	repo.store[releaseID] = &Release{ID: releaseID, ProfileID: "web-console/staging", CandidateID: "candidate-current", DestinationRevisionID: "destination-current", DeploymentID: "dep-human", Status: StatusFailed}
	repo.publications = []PublicationReceipt{{
		CandidateID: "candidate-current", DestinationRevisionID: "destination-current", TargetID: "cloud:demo",
		ArtifactDigest: "sha256:current-release", DestinationObject: "dep-human", Producer: "scenario-to-cloud",
		ExternalReceipt: "deploy-receipt", Outcome: ReceiptVerified, ObservedAt: time.Now().Add(-time.Minute), ReviewKey: "review-current",
		PredecessorArtifactDigest: "sha256:last-good",
	}}
	owner := &CloudRecoveryOperation{
		DeploymentID: "dep-human", OperationID: "s2c-recovery-human", IdempotencyKey: CloudRecoveryIdempotencyKey(operationID),
		Action: "rollback", Status: "pending", ExpectedCurrentBundleSHA256: "current-bundle",
		LastGoodReleaseDigest: "sha256:last-good", ReviewRef: "review-current", PreviewRef: "preview-human",
		DataCompatibility: "compatible",
	}
	return repo, operationID, owner
}

func reconcileCloudRecoveryRequest(handler *Handler, operationID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/release-operations/"+operationID+"/reconcile-cloud-recovery", nil)
	req = mux.SetURLVars(req, map[string]string{"operation_id": operationID})
	response := httptest.NewRecorder()
	handler.ReconcileCloudRecoveryOperation(response, req)
	return response
}

func TestHumanCloudRecoveryReadbackAttachesByKeyThenExactIDAcrossHandlerRestart(t *testing.T) {
	repo, operationID, owner := validCloudRecoveryReconcileFixture(t)
	reader := &cloudRecoveryReadbackFake{findResult: owner}
	orch := &cloudRecoveryReconcileOrchestrator{mockOrch: &mockOrch{}, cloudRecoveryReadbackFake: reader}
	auth := func(context.Context) error { return nil }
	handler := NewHandler(repo, nil, nil, orch, nil).WithDurableOperations(true).WithReadAuthorization(auth)

	response := reconcileCloudRecoveryRequest(handler, operationID)
	if response.Code != http.StatusOK {
		t.Fatalf("first reconciliation status=%d body=%s", response.Code, response.Body.String())
	}
	if reader.findCalls != 1 || reader.key != CloudRecoveryIdempotencyKey(operationID) || reader.getCalls != 0 {
		t.Fatalf("first lookup did not use the stable DM key: %+v", reader)
	}
	if repo.step.RecoveryState != "owner_pending" || repo.step.RecoveryOwnerOperationID != owner.OperationID || repo.step.RecoveryPreviewRef != owner.PreviewRef || repo.operation.Status != OperationAmbiguous {
		t.Fatalf("pending operation was not durably attached: step=%+v operation=%+v", repo.step, repo.operation)
	}

	owner.Status = "succeeded"
	owner.ReceiptJSON = json.RawMessage(`{"schema_version":1,"deployment_id":"dep-human","operation_id":"s2c-recovery-human","action":"rollback","outcome":"rolled_back","health":"healthy","bundle_sha256":"repair-bundle","external_receipt":"s2c-recovery-receipt","observed_at":"2026-11-24T12:00:00Z","dry_run":false,"preview_ref":"preview-human","review_key":"review-current","release_digest":"sha256:last-good"}`)
	reader.getResult = owner
	// A new handler models a DM process restart; SQL-backed CloudStep is the
	// same durable repository and already contains the exact owner operation ID.
	handler = NewHandler(repo, nil, nil, orch, nil).WithDurableOperations(true).WithReadAuthorization(auth)
	response = reconcileCloudRecoveryRequest(handler, operationID)
	if response.Code != http.StatusOK {
		t.Fatalf("terminal reconciliation status=%d body=%s", response.Code, response.Body.String())
	}
	if reader.findCalls != 1 || reader.getCalls != 1 || reader.getID != owner.OperationID {
		t.Fatalf("restart did not reattach to the exact owner operation: %+v", reader)
	}
	if repo.step.RecoveryState != "succeeded" || string(repo.step.RecoveryReceipt) != string(owner.ReceiptJSON) || repo.operation.Status != OperationComplete || repo.operation.ActiveStage != "recovery_settled" {
		t.Fatalf("terminal recovery did not settle DM standing: step=%+v operation=%+v", repo.step, repo.operation)
	}
	if len(repo.recovery) != 1 || !repo.recovery[0].Valid() || repo.recovery[0].ExternalReceipt != "s2c-recovery-receipt" {
		t.Fatalf("DM recovery receipt settlement = %+v", repo.recovery)
	}
}

func TestHumanCloudRecoveryReadbackFailsClosedOnCrossDeployment(t *testing.T) {
	repo, operationID, owner := validCloudRecoveryReconcileFixture(t)
	owner.DeploymentID = "different-deployment"
	reader := &cloudRecoveryReadbackFake{findResult: owner}
	orch := &cloudRecoveryReconcileOrchestrator{mockOrch: &mockOrch{}, cloudRecoveryReadbackFake: reader}
	handler := NewHandler(repo, nil, nil, orch, nil).WithReadAuthorization(func(context.Context) error { return nil })
	response := reconcileCloudRecoveryRequest(handler, operationID)
	if response.Code != http.StatusOK || repo.step.RecoveryState != "identity_mismatch" || repo.operation.Status != OperationAmbiguous || len(repo.recovery) != 0 {
		t.Fatalf("cross-deployment owner operation was not refused: status=%d step=%+v operation=%+v receipts=%+v body=%s", response.Code, repo.step, repo.operation, repo.recovery, response.Body.String())
	}
}

func TestHumanCloudRecoveryMissingOperationRemainsAwaitingHuman(t *testing.T) {
	repo, operationID, _ := validCloudRecoveryReconcileFixture(t)
	reader := &cloudRecoveryReadbackFake{}
	orch := &cloudRecoveryReconcileOrchestrator{mockOrch: &mockOrch{}, cloudRecoveryReadbackFake: reader}
	handler := NewHandler(repo, nil, nil, orch, nil).WithReadAuthorization(func(context.Context) error { return nil })
	response := reconcileCloudRecoveryRequest(handler, operationID)
	if response.Code != http.StatusOK || repo.step.RecoveryState != "awaiting_human_recovery" || reader.findCalls != 1 || len(repo.recovery) != 0 {
		t.Fatalf("missing owner operation did not remain visible and safe: status=%d step=%+v calls=%d body=%s", response.Code, repo.step, reader.findCalls, response.Body.String())
	}
}

func TestHumanCloudRecoveryReadbackRequiresAuthorization(t *testing.T) {
	repo, operationID, _ := validCloudRecoveryReconcileFixture(t)
	reader := &cloudRecoveryReadbackFake{}
	orch := &cloudRecoveryReconcileOrchestrator{mockOrch: &mockOrch{}, cloudRecoveryReadbackFake: reader}
	handler := NewHandler(repo, nil, nil, orch, nil).WithReadAuthorization(func(context.Context) error {
		return errors.New("not authorized")
	})
	response := reconcileCloudRecoveryRequest(handler, operationID)
	if response.Code != http.StatusUnauthorized || reader.findCalls != 0 || reader.getCalls != 0 || repo.step.RecoveryState != "awaiting_human_recovery" {
		t.Fatalf("unauthorized recovery readback reached S2C or changed standing: status=%d finds=%d gets=%d state=%s", response.Code, reader.findCalls, reader.getCalls, repo.step.RecoveryState)
	}
}

func TestHumanCloudRecoveryReadbackBlocksOperationReleaseProfileMismatch(t *testing.T) {
	repo, operationID, owner := validCloudRecoveryReconcileFixture(t)
	repo.operation.ProfileID = "web-console/production"
	reader := &cloudRecoveryReadbackFake{findResult: owner}
	orch := &cloudRecoveryReconcileOrchestrator{mockOrch: &mockOrch{}, cloudRecoveryReadbackFake: reader}
	handler := NewHandler(repo, nil, nil, orch, nil).WithReadAuthorization(func(context.Context) error { return nil })
	response := reconcileCloudRecoveryRequest(handler, operationID)
	if response.Code != http.StatusOK || reader.findCalls != 0 || repo.step.RecoveryState != "identity_mismatch" || repo.operation.Status != OperationAmbiguous {
		t.Fatalf("profile mismatch was not blocked before owner readback: status=%d finds=%d step=%+v operation=%+v body=%s", response.Code, reader.findCalls, repo.step, repo.operation, response.Body.String())
	}
}
