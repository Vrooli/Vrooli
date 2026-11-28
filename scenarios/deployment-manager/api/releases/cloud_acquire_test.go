package releases

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	internalreleases "deployment-manager/internal/releases"
	"github.com/gorilla/mux"
	instancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance"
	"google.golang.org/protobuf/encoding/protojson"
	_ "modernc.org/sqlite"
)

type acquireFixture struct {
	repo   *SQLRepository
	result *CloudAcquireResult
	err    error
	calls  int
}

func (f *acquireFixture) AcquireCloudCompute(ctx context.Context, request CloudAcquireRequest) (*CloudAcquireResult, error) {
	f.calls++
	op, err := f.repo.GetOperation(ctx, operationIDForRelease("release-acquire"))
	if err != nil {
		return nil, err
	}
	var snap durableDeployRequest
	if err := json.Unmarshal(op.RequestSnapshot, &snap); err != nil {
		return nil, err
	}
	plan := snap.CloudAcquirePlan
	if snap.CloudAcquireIdempotencyKey != request.IdempotencyKey || snap.CloudComputeManagerInstanceID != "" || !snap.CloudAcquireCompute || plan == nil || plan.ProfileID != "web-console/staging" || plan.Provider != request.Provider || plan.Region != request.Region || plan.Size != request.Size || plan.AcquireIdempotencyKey != request.IdempotencyKey || plan.QuoteIdempotencyKey != request.IdempotencyKey || plan.CapCurrency != request.CapCurrency || plan.CapMonthlyMinor != request.CapMonthlyMinor || plan.CapTaxBasis != request.CapTaxBasis {
		return nil, errors.New("acquisition began before stable intent was durably stored")
	}
	return f.result, f.err
}

type acquireOrchestrator struct {
	repo   *SQLRepository
	calls  int
	called chan DeployRequest
}

type enrollFixture struct {
	repo   *SQLRepository
	result *CloudEnrollmentResult
	err    error
	calls  int
}

func (f *enrollFixture) EnrollCloudCompute(ctx context.Context, request CloudEnrollmentRequest) (*CloudEnrollmentResult, error) {
	f.calls++
	if request.IdempotencyKey != "dm-enroll:release-op-release-acquire" || request.AcquisitionIdempotencyKey != "dm-acquire:release-op-release-acquire" || request.InstanceID != "cm-fixture-1" {
		return nil, errors.New("enrollment did not bind the stable keys and exact target")
	}
	op, err := f.repo.GetOperation(ctx, operationIDForRelease("release-acquire"))
	if err != nil {
		return nil, err
	}
	var snap durableDeployRequest
	if err := json.Unmarshal(op.RequestSnapshot, &snap); err != nil {
		return nil, err
	}
	var target releaseCloudTarget
	if err := json.Unmarshal(snap.CloudManifest, &target); err != nil {
		return nil, err
	}
	if snap.CloudComputeManagerInstanceID != request.InstanceID || target.Target.VPS.ComputeManagerInstanceID != request.InstanceID || snap.CloudAcquireReceipt == nil {
		return nil, errors.New("Bridge enrollment began before exact target was durably persisted")
	}
	return f.result, f.err
}

func (f *acquireOrchestrator) RunDeploy(ctx context.Context, request DeployRequest) (*DeployResult, error) {
	f.calls++
	if f.called != nil {
		f.called <- request
	}
	op, err := f.repo.GetOperation(ctx, request.OperationID)
	if err != nil {
		return nil, err
	}
	var snap durableDeployRequest
	if err := json.Unmarshal(op.RequestSnapshot, &snap); err != nil {
		return nil, err
	}
	if snap.CloudComputeManagerInstanceID != "cm-fixture-1" || snap.CloudSpendQuoteIdempotencyKey != "dm-acquire:release-op-release-acquire" || snap.CloudAcquireReceipt == nil || len(snap.CloudEnrollmentReceipt) == 0 {
		return nil, errors.New("downstream began before exact CM target, acquisition receipt and enrollment receipt were persisted")
	}
	if err := f.repo.UpdateStatus(ctx, request.ReleaseID, StatusPublishing); err != nil {
		return nil, err
	}
	if err := f.repo.UpdateStatus(ctx, request.ReleaseID, StatusPublished); err != nil {
		return nil, err
	}
	return &DeployResult{Status: "complete"}, nil
}

func (f *resumableOperationRepoFake) UpdateOperationSnapshot(_ context.Context, operationID string, snapshot json.RawMessage) error {
	if f.operation == nil || f.operation.ID != operationID {
		return errors.New("operation not found")
	}
	f.operation.RequestSnapshot = append(json.RawMessage(nil), snapshot...)
	return nil
}

type alwaysCloudAdmission struct{}

func (alwaysCloudAdmission) VerifyCloudComputeAdmission(context.Context, string, string) error {
	return nil
}

func TestAcquireCloudComputePersistsExactTargetBeforeDownstream(t *testing.T) {
	db, repo := newAcquireSQLRepo(t)
	defer db.Close()
	ctx := context.Background()
	request := acquireDeployRequest()
	seedAcquireOperation(t, ctx, repo, request)
	lease, acquired, err := repo.AcquireOperationLease(ctx, operationIDForRelease(request.ReleaseID), "worker-acquire-test", time.Minute)
	if err != nil || !acquired {
		t.Fatalf("acquire operation lease: acquired=%v err=%v", acquired, err)
	}
	ctx = WithOperationLease(ctx, lease.OperationID, lease.Owner, lease.Fence)
	acquirer := &acquireFixture{repo: repo, result: fixtureAcquireResult(request)}
	enroller := &enrollFixture{repo: repo, result: fixtureEnrollmentResult("ready")}
	orch := &acquireOrchestrator{repo: repo}
	h := NewHandler(repo, nil, nil, orch, nil).WithCloudComputeAdmissionVerifier(alwaysCloudAdmission{}).WithCloudComputeAcquirer(acquirer).WithCloudComputeEnroller(enroller).WithDurableOperations(true)
	h.executeReleaseWithLease(ctx, operationIDForRelease(request.ReleaseID), request.ReleaseID, "", request, nil)
	op, err := repo.GetOperation(ctx, operationIDForRelease(request.ReleaseID))
	if err != nil {
		t.Fatal(err)
	}
	var snap durableDeployRequest
	if err := json.Unmarshal(op.RequestSnapshot, &snap); err != nil {
		t.Fatal(err)
	}
	var manifest releaseCloudTarget
	if err := json.Unmarshal(snap.CloudManifest, &manifest); err != nil {
		t.Fatal(err)
	}
	if acquirer.calls != 1 || enroller.calls != 1 || orch.calls != 1 || op.Status != OperationComplete || manifest.Target.VPS.ComputeManagerInstanceID != "cm-fixture-1" {
		t.Fatalf("acquire=%d enroll=%d downstream=%d operation=%+v target=%q", acquirer.calls, enroller.calls, orch.calls, op, manifest.Target.VPS.ComputeManagerInstanceID)
	}
	hydrateOperationCloudComputeRefs(op)
	wire := op.Proto().GetCloudAcquireReceipt()
	if wire == nil || wire.GetOperationId() != op.ID || wire.GetProfileId() != request.ProfileID || wire.GetAcquisitionIdempotencyKey() != request.CloudAcquireIdempotencyKey || wire.GetComputeManagerInstanceId() != "cm-fixture-1" || wire.GetSourceReceipt().GetState() != "acquired" || wire.GetSourceReceipt().GetSpendReceipt().GetIdempotencyKey() != request.CloudAcquireIdempotencyKey {
		t.Fatalf("typed generated acquisition receipt = %+v", wire)
	}
}

func TestAcquirePlanSnapshotRefusesPolicyOrQuoteRebinding(t *testing.T) {
	request := acquireDeployRequest()
	snapshot, err := snapshotForDeployRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	var saved durableDeployRequest
	if err := json.Unmarshal(snapshot, &saved); err != nil {
		t.Fatal(err)
	}
	if err := validateCloudAcquirePlan(saved.CloudAcquirePlan, saved.ProfileID, saved.CloudAcquireRegion, saved.CloudAcquireIdempotencyKey, saved.CloudSpendQuoteIdempotencyKey); err != nil {
		t.Fatalf("saved A2 plan refused: %v", err)
	}
	changed := *saved.CloudAcquirePlan
	changed.CapMonthlyMinor++
	if err := validateCloudAcquirePlan(&changed, saved.ProfileID, saved.CloudAcquireRegion, saved.CloudAcquireIdempotencyKey, saved.CloudSpendQuoteIdempotencyKey); err == nil {
		t.Fatal("rebound cap was accepted")
	}
	if err := validateCloudAcquirePlan(saved.CloudAcquirePlan, saved.ProfileID, saved.CloudAcquireRegion, saved.CloudAcquireIdempotencyKey, "other-quote-key"); err == nil {
		t.Fatal("rebound quote key was accepted")
	}
}

func TestAcquireMissingDurablePlanPersistsBlockedStateBeforeCMCall(t *testing.T) {
	db, repo := newAcquireSQLRepo(t)
	defer db.Close()
	ctx := context.Background()
	request := acquireDeployRequest()
	seedAcquireOperation(t, ctx, repo, request)
	operationID := operationIDForRelease(request.ReleaseID)
	lease, acquired, err := repo.AcquireOperationLease(ctx, operationID, "worker-plan-missing-test", time.Minute)
	if err != nil || !acquired {
		t.Fatalf("operation lease acquired=%v err=%v", acquired, err)
	}
	leaseCtx := WithOperationLease(ctx, operationID, lease.Owner, lease.Fence)
	var snapshot durableDeployRequest
	operation, err := repo.GetOperation(ctx, operationID)
	if err != nil || json.Unmarshal(operation.RequestSnapshot, &snapshot) != nil {
		t.Fatalf("read operation snapshot: operation=%+v err=%v", operation, err)
	}
	snapshot.CloudAcquirePlan = nil
	raw, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateOperationSnapshot(leaseCtx, operationID, raw); err != nil {
		t.Fatalf("remove plan for fail-closed test: %v", err)
	}
	request.CloudAcquirePlan = nil
	acquirer := &acquireFixture{repo: repo, result: fixtureAcquireResult(request)}
	orch := &acquireOrchestrator{repo: repo}
	h := NewHandler(repo, nil, nil, orch, nil).WithCloudComputeAcquirer(acquirer).WithDurableOperations(true)
	h.executeReleaseWithLease(leaseCtx, operationID, request.ReleaseID, "", request, nil)
	operation, err = repo.GetOperation(ctx, operationID)
	if err != nil {
		t.Fatal(err)
	}
	if operation.Status != OperationFailed || operation.ActiveStage != "acquire-blocked" || acquirer.calls != 0 || orch.calls != 0 {
		t.Fatalf("operation=%+v CM calls=%d downstream calls=%d", operation, acquirer.calls, orch.calls)
	}
}

func TestAcquisitionReceiptMustMatchSnapshottedA2Policy(t *testing.T) {
	request := acquireDeployRequest()
	request.CloudSpendQuoteIdempotencyKey = request.CloudAcquireIdempotencyKey
	request.CloudComputeManagerInstanceID = "cm-fixture-1"
	request.CloudManifest, _ = bindCloudManifestInstance(request.CloudManifest, request.CloudComputeManagerInstanceID)
	base := fixtureAcquireResult(request).Receipt
	for _, tc := range []struct {
		name   string
		mutate func(*instancev1.GetAcquireReceiptResponse)
	}{
		{name: "provider", mutate: func(r *instancev1.GetAcquireReceiptResponse) {
			r.Instance.Provider = "digitalocean"
			r.SpendReceipt.Provider = "digitalocean"
		}},
		{name: "region", mutate: func(r *instancev1.GetAcquireReceiptResponse) { r.Instance.Region = "ash" }},
		{name: "size", mutate: func(r *instancev1.GetAcquireReceiptResponse) { r.Instance.Size = "cx33" }},
		{name: "cap", mutate: func(r *instancev1.GetAcquireReceiptResponse) { r.SpendReceipt.CapMonthlyMinor = 710 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wire := &instancev1.GetAcquireReceiptResponse{}
			if err := protojson.Unmarshal(base, wire); err != nil {
				t.Fatal(err)
			}
			tc.mutate(wire)
			encoded, err := protojson.Marshal(wire)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := newCloudAcquireReceipt(operationIDForRelease(request.ReleaseID), request, encoded, time.Now().UTC()); err == nil {
				t.Fatal("receipt with changed provider policy was accepted")
			}
		})
	}
}

func TestTypedAcquisitionReceiptSurvivesSQLRestartAndRejectsRebinding(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "receipt-restart.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(internalreleases.Schema()); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLRepository(db)
	ctx := context.Background()
	request := acquireDeployRequest()
	seedAcquireOperation(t, ctx, repo, request)
	operationID := operationIDForRelease(request.ReleaseID)
	lease, acquired, err := repo.AcquireOperationLease(ctx, operationID, "receipt-restart-worker", time.Minute)
	if err != nil || !acquired {
		t.Fatalf("lease acquired=%v err=%v", acquired, err)
	}
	leaseCtx := WithOperationLease(ctx, operationID, lease.Owner, lease.Fence)
	request.CloudComputeManagerInstanceID = "cm-fixture-1"
	request.CloudSpendQuoteIdempotencyKey = request.CloudAcquireIdempotencyKey
	request.CloudManifest, err = bindCloudManifestInstance(request.CloudManifest, request.CloudComputeManagerInstanceID)
	if err != nil {
		t.Fatal(err)
	}
	request.CloudAcquireReceipt, err = newCloudAcquireReceipt(operationID, request, fixtureAcquireResult(request).Receipt, time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := snapshotForDeployRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateOperationSnapshot(leaseCtx, operationID, snapshot); err != nil {
		t.Fatalf("persist typed receipt: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	repo = NewSQLRepository(db)
	standing, err := repo.GetOperation(ctx, operationID)
	if err != nil {
		t.Fatal(err)
	}
	restoredRequest, err := deployRequestFromSnapshot(standing.RequestSnapshot, nil)
	if err != nil || restoredRequest.CloudAcquirePlan == nil || restoredRequest.CloudAcquirePlan.Provider != CloudA2Provider || restoredRequest.CloudAcquirePlan.Size != CloudA2Size || restoredRequest.CloudAcquirePlan.CapMonthlyMinor != CloudA2CapMonthlyMinor {
		t.Fatalf("restarted operation lost immutable acquire plan: plan=%+v err=%v", restoredRequest.CloudAcquirePlan, err)
	}
	hydrateOperationCloudComputeRefs(standing)
	wire := standing.Proto().GetCloudAcquireReceipt()
	if wire == nil || wire.GetSourceReceipt().GetInstance().GetId() != "cm-fixture-1" || wire.GetSourceReceipt().GetSpendReceipt().GetAdmissionDecision() != "approved" || wire.GetSourceReceipt().GetSpendReceipt().GetQuoteState() != "approved" || wire.GetSourceReceipt().GetState() != "acquired" {
		t.Fatalf("restarted typed receipt = %+v", wire)
	}
	if err := repo.UpdateOperationSnapshot(WithOperationLease(ctx, operationID, "stale-owner", lease.Fence+1), operationID, snapshot); err == nil {
		t.Fatal("stale fence rewrote the acquisition receipt")
	}

	var durable durableDeployRequest
	if err := json.Unmarshal(standing.RequestSnapshot, &durable); err != nil {
		t.Fatal(err)
	}
	changed := durable
	changed.CloudAcquireReceipt = cloneCloudAcquireReceipt(durable.CloudAcquireReceipt)
	changed.CloudAcquireReceipt.SourceFingerprint = "sha256:changed"
	changedSnapshot, err := json.Marshal(changed)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateOperationSnapshot(leaseCtx, operationID, changedSnapshot); err == nil {
		t.Fatal("changed generated CM receipt was accepted on replay")
	}

	changed = durable
	changed.ProfileID = "other/profile"
	changedSnapshot, err = json.Marshal(changed)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateOperationSnapshot(leaseCtx, operationID, changedSnapshot); err == nil {
		t.Fatal("receipt was rebound to another DM profile")
	}

	changed = durable
	var manifest map[string]interface{}
	if err := json.Unmarshal(durable.CloudManifest, &manifest); err != nil {
		t.Fatal(err)
	}
	target := manifest["target"].(map[string]interface{})
	vps := target["vps"].(map[string]interface{})
	vps["compute_manager_instance_id"] = "cm-other"
	changed.CloudManifest, err = json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	changed.CloudComputeManagerInstanceID = "cm-other"
	changedSnapshot, err = json.Marshal(changed)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.UpdateOperationSnapshot(leaseCtx, operationID, changedSnapshot); err == nil {
		t.Fatal("receipt was rebound to another stored target")
	}
}

func TestAcquireCloudComputeWaitsForPersistedEnrollmentReadiness(t *testing.T) {
	for _, tc := range []struct {
		name, state, stage string
		err                error
	}{
		{name: "awaiting enrollment", state: "awaiting_enrollment", stage: "awaiting_enrollment", err: ErrCloudEnrollmentPending},
		{name: "enrolling", state: "enrolling", stage: "enrolling", err: ErrCloudEnrollmentInProgress},
		{name: "ambiguous readback", state: "ambiguous", stage: "enrollment-reconcile", err: ErrCloudEnrollmentAmbiguous},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, repo := newAcquireSQLRepo(t)
			defer db.Close()
			ctx := context.Background()
			request := acquireDeployRequest()
			seedAcquireOperation(t, ctx, repo, request)
			lease, acquired, err := repo.AcquireOperationLease(ctx, operationIDForRelease(request.ReleaseID), "worker-enrollment-test", time.Minute)
			if err != nil || !acquired {
				t.Fatalf("lease acquired=%v err=%v", acquired, err)
			}
			ctx = WithOperationLease(ctx, lease.OperationID, lease.Owner, lease.Fence)
			acquirer := &acquireFixture{repo: repo, result: fixtureAcquireResult(request)}
			enroller := &enrollFixture{repo: repo, result: fixtureEnrollmentResult(tc.state), err: tc.err}
			orch := &acquireOrchestrator{repo: repo}
			h := NewHandler(repo, nil, nil, orch, nil).WithCloudComputeAdmissionVerifier(alwaysCloudAdmission{}).WithCloudComputeAcquirer(acquirer).WithCloudComputeEnroller(enroller).WithDurableOperations(true)
			h.executeReleaseWithLease(ctx, operationIDForRelease(request.ReleaseID), request.ReleaseID, "", request, nil)
			op, err := repo.GetOperation(ctx, operationIDForRelease(request.ReleaseID))
			if err != nil {
				t.Fatal(err)
			}
			var snap durableDeployRequest
			if err := json.Unmarshal(op.RequestSnapshot, &snap); err != nil {
				t.Fatal(err)
			}
			if op.Status != OperationAmbiguous || op.ActiveStage != tc.stage || orch.calls != 0 || snap.CloudComputeManagerInstanceID != "cm-fixture-1" || len(snap.CloudEnrollmentReceipt) == 0 || enroller.calls != 1 {
				t.Fatalf("operation=%+v target=%q orchestrator=%d enrollment=%d", op, snap.CloudComputeManagerInstanceID, orch.calls, enroller.calls)
			}
		})
	}
}

func TestResumePendingEnrollmentReusesAcquisitionAndEnrollmentKeys(t *testing.T) {
	base := newMockRepo()
	request := acquireDeployRequest()
	request.CloudComputeManagerInstanceID = "cm-fixture-1"
	request.CloudSpendQuoteIdempotencyKey = request.CloudAcquireIdempotencyKey
	request.CloudManifest, _ = bindCloudManifestInstance(request.CloudManifest, "cm-fixture-1")
	request.CloudAcquireReceipt, _ = newCloudAcquireReceipt(operationIDForRelease(request.ReleaseID), request, fixtureAcquireResult(request).Receipt, time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC))
	snapshot, err := snapshotForDeployRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	base.store[request.ReleaseID] = &Release{ID: request.ReleaseID, ProfileID: request.ProfileID, Status: StatusAmbiguous}
	repo := &resumableOperationRepoFake{mockRepo: base, operation: &Operation{ID: operationIDForRelease(request.ReleaseID), ReleaseID: request.ReleaseID, ProfileID: request.ProfileID, Status: OperationAmbiguous, ActiveStage: "awaiting_enrollment", RequestSnapshot: snapshot}}
	acquirer := &resumeAcquireFixture{result: fixtureAcquireResult(request)}
	enroller := &resumeEnrollmentFixture{repo: repo, result: fixtureEnrollmentResult("ready")}
	orch := &resumeReadyOrchestrator{called: make(chan DeployRequest, 1)}
	h := NewHandler(repo, nil, nil, orch, nil).WithExecutionAuthorization(func(context.Context) error { return nil }).WithCloudComputeAdmissionVerifier(alwaysCloudAdmission{}).WithCloudComputeAcquirer(acquirer).WithCloudComputeEnroller(enroller).WithDurableOperations(true)
	req := mux.SetURLVars(httptest.NewRequest(http.MethodPost, "/api/v1/release-operations/"+repo.operation.ID+"/resume-cloud", nil), map[string]string{"operation_id": repo.operation.ID})
	response := httptest.NewRecorder()
	h.ResumeCloudOperation(response, req)
	if response.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	select {
	case got := <-orch.called:
		if got.CloudAcquireIdempotencyKey != request.CloudAcquireIdempotencyKey || got.CloudEnrollmentIdempotencyKey != request.CloudEnrollmentIdempotencyKey || got.CloudComputeManagerInstanceID != request.CloudComputeManagerInstanceID {
			t.Fatalf("resumed request=%+v", got)
		}
	case <-time.After(time.Second):
		t.Fatalf("resume did not pass the saved target to downstream; operation=%+v", repo.operation)
	}
}

type resumeEnrollmentFixture struct {
	repo   *resumableOperationRepoFake
	result *CloudEnrollmentResult
}

func (f *resumeEnrollmentFixture) EnrollCloudCompute(ctx context.Context, request CloudEnrollmentRequest) (*CloudEnrollmentResult, error) {
	if request.IdempotencyKey != "dm-enroll:release-op-release-acquire" || request.AcquisitionIdempotencyKey != "dm-acquire:release-op-release-acquire" || request.InstanceID != "cm-fixture-1" {
		return nil, errors.New("resume changed enrollment identity")
	}
	var snap durableDeployRequest
	if json.Unmarshal(f.repo.operation.RequestSnapshot, &snap) != nil || snap.CloudComputeManagerInstanceID != request.InstanceID {
		return nil, errors.New("resume enrolled before restoring exact target")
	}
	return f.result, nil
}

type resumeAcquireFixture struct{ result *CloudAcquireResult }

func (f *resumeAcquireFixture) AcquireCloudCompute(_ context.Context, request CloudAcquireRequest) (*CloudAcquireResult, error) {
	if request.IdempotencyKey != f.result.QuoteIdempotencyKey {
		return nil, errors.New("resume changed acquisition key")
	}
	return f.result, nil
}

type resumeReadyOrchestrator struct{ called chan DeployRequest }

func (f *resumeReadyOrchestrator) RunDeploy(_ context.Context, request DeployRequest) (*DeployResult, error) {
	f.called <- request
	return &DeployResult{Status: "complete"}, nil
}

func TestAcquireCloudComputeRefusalAndAmbiguityBlockDownstream(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		err                   error
		wantStatus, wantStage string
	}{
		{name: "admission refused", err: errors.New("stored quote over approved cap"), wantStatus: OperationFailed, wantStage: "acquire-blocked"},
		{name: "owner readback unavailable", err: errors.Join(ErrCloudAcquireAmbiguous, errors.New("receipt unavailable")), wantStatus: OperationAmbiguous, wantStage: "acquire-reconcile"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, repo := newAcquireSQLRepo(t)
			defer db.Close()
			ctx := context.Background()
			request := acquireDeployRequest()
			seedAcquireOperation(t, ctx, repo, request)
			lease, acquired, err := repo.AcquireOperationLease(ctx, operationIDForRelease(request.ReleaseID), "worker-acquire-test", time.Minute)
			if err != nil || !acquired {
				t.Fatalf("acquire operation lease: acquired=%v err=%v", acquired, err)
			}
			ctx = WithOperationLease(ctx, lease.OperationID, lease.Owner, lease.Fence)
			acquirer := &acquireFixture{repo: repo, err: tc.err}
			orch := &acquireOrchestrator{repo: repo}
			h := NewHandler(repo, nil, nil, orch, nil).WithCloudComputeAcquirer(acquirer).WithDurableOperations(true)
			h.executeReleaseWithLease(ctx, operationIDForRelease(request.ReleaseID), request.ReleaseID, "", request, nil)
			op, err := repo.GetOperation(ctx, operationIDForRelease(request.ReleaseID))
			if err != nil {
				t.Fatal(err)
			}
			if op.Status != tc.wantStatus || op.ActiveStage != tc.wantStage || orch.calls != 0 {
				t.Fatalf("operation=%+v downstream calls=%d", op, orch.calls)
			}
		})
	}
}
func newAcquireSQLRepo(t *testing.T) (*sql.DB, *SQLRepository) {
	t.Helper()
	db, err := sql.Open("sqlite", "file:dm-acquire-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(internalreleases.Schema()); err != nil {
		db.Close()
		t.Fatal(err)
	}
	return db, NewSQLRepository(db)
}
func acquireDeployRequest() DeployRequest {
	plan, _ := canonicalCloudAcquirePlan("web-console/staging", "nbg1", "dm-acquire:release-op-release-acquire")
	return DeployRequest{ProfileID: "web-console/staging", ReleaseID: "release-acquire", IdempotencyKey: "start-key", CloudAcquireCompute: true, CloudAcquireRegion: "nbg1", CloudAcquireIdempotencyKey: "dm-acquire:release-op-release-acquire", CloudSpendQuoteIdempotencyKey: "dm-acquire:release-op-release-acquire", CloudAcquirePlan: &plan, CloudEnrollmentIdempotencyKey: "dm-enroll:release-op-release-acquire", CloudManifest: json.RawMessage(`{"environment":"staging","scenario":{"id":"web-console"},"edge":{"domain":"staging.vrooli.com"},"target":{"type":"vps","vps":{}}}`)}
}

func fixtureAcquireResult(request DeployRequest) *CloudAcquireResult {
	source := &instancev1.GetAcquireReceiptResponse{
		Instance:     &instancev1.Instance{Id: "cm-fixture-1", Provider: "hetzner", ProviderInstanceId: "hetzner-server-fixture-1", Region: "nbg1", Size: "cx23", Tenant: "deployment-manager"},
		SpendReceipt: &instancev1.SpendReceipt{IdempotencyKey: request.CloudAcquireIdempotencyKey, InstanceId: "cm-fixture-1", Provider: "hetzner", Region: "nbg1", Size: "cx23", AdmissionDecision: "approved", QuoteState: "approved", CapCurrency: "USD", CapMonthlyMinor: 709, CapTaxBasis: "pre-tax"},
		State:        "acquired",
	}
	receipt, _ := protojson.Marshal(source)
	return &CloudAcquireResult{QuoteIdempotencyKey: request.CloudAcquireIdempotencyKey, InstanceID: "cm-fixture-1", Receipt: receipt}
}

func fixtureEnrollmentResult(state string) *CloudEnrollmentResult {
	receipt, _ := json.Marshal(map[string]string{"idempotencyKey": "dm-enroll:release-op-release-acquire", "acquisitionIdempotencyKey": "dm-acquire:release-op-release-acquire", "instanceId": "cm-fixture-1", "state": state, "bridgeMachineId": "bridge-1", "bridgeOperationId": "bridge-op-1"})
	return &CloudEnrollmentResult{State: state, InstanceID: "cm-fixture-1", BridgeMachineID: "bridge-1", BridgeOperationID: "bridge-op-1", Receipt: receipt}
}
func seedAcquireOperation(t *testing.T, ctx context.Context, repo *SQLRepository, request DeployRequest) {
	t.Helper()
	if _, err := repo.db.ExecContext(ctx, `INSERT INTO releases (id,profile_id,git_commit_hash,release_version,channel,status) VALUES (?,?,?,?,?,?)`, request.ReleaseID, request.ProfileID, "commit-1", "1.0.0", "stable", StatusPending); err != nil {
		t.Fatal(err)
	}
	snapshot, err := snapshotForDeployRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.InsertOperation(ctx, &Operation{ID: operationIDForRelease(request.ReleaseID), ReleaseID: request.ReleaseID, ProfileID: request.ProfileID, IdempotencyKey: request.IdempotencyKey, RequestSnapshot: snapshot, Status: OperationQueued}); err != nil {
		t.Fatal(err)
	}
}
