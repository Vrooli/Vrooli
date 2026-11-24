package releases

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	internalreleases "deployment-manager/internal/releases"
	"github.com/gorilla/mux"
	instancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance"
	releasesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/deployment-manager/v1/releases"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
	_ "modernc.org/sqlite"
)

func TestRequestCloudTeardownPersistsIntentAndReplaysWithoutConfiguredCMConsumer(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "teardown-refusal.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(internalreleases.Schema()); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLRepository(db)
	ctx := context.Background()
	request := teardownTestRequest(t, "hetzner")
	seedAcquireOperation(t, ctx, repo, request)
	operationID := operationIDForRelease(request.ReleaseID)
	if _, err := db.Exec(`UPDATE release_operations SET status=? WHERE id=?`, OperationComplete, operationID); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(repo, nil, nil, nil, nil).WithAuthorization(func(context.Context) error { return nil }).WithDurableOperations(true)

	call := func() (*Operation, int) {
		req := mux.SetURLVars(httptest.NewRequest("POST", "/api/v1/release-operations/"+operationID+"/request-cloud-teardown", nil), map[string]string{"operation_id": operationID})
		response := httptest.NewRecorder()
		h.RequestCloudTeardown(response, req)
		var operation Operation
		if response.Code == 200 {
			if err := json.Unmarshal(response.Body.Bytes(), &operation); err != nil {
				t.Fatalf("decode teardown standing: %v body=%s", err, response.Body.String())
			}
		}
		return &operation, response.Code
	}
	first, status := call()
	if status != 200 || first.CloudTeardownReceipt == nil {
		t.Fatalf("first request status=%d operation=%+v", status, first)
	}
	receipt := first.CloudTeardownReceipt
	if receipt.State != "pending" || receipt.TeardownKey != "dm-release:"+operationID || receipt.Provider != "hetzner" || receipt.ProviderInstanceID != "hetzner-server-fixture-1" || !strings.Contains(receipt.Reason, "awaiting exact Compute Manager") {
		t.Fatalf("pending receipt = %+v", receipt)
	}
	wire := first.Proto().GetCloudTeardownReceipt()
	if wire == nil || wire.GetState() != releasesv1.CloudTeardownState_CLOUD_TEARDOWN_STATE_PENDING || wire.GetAcquisitionReceiptSha256() != request.CloudAcquireReceipt.SourceFingerprint {
		t.Fatalf("generated pending receipt = %+v", wire)
	}
	standing, err := repo.GetOperation(ctx, operationID)
	if err != nil {
		t.Fatal(err)
	}
	var changed durableDeployRequest
	if err := json.Unmarshal(standing.RequestSnapshot, &changed); err != nil {
		t.Fatal(err)
	}
	changed.CloudTeardownReceipt.Reason = "forged replacement"
	changedSnapshot, _ := json.Marshal(changed)
	lease, acquired, err := repo.AcquireOperationLease(ctx, operationID, "teardown-rebind-test", time.Minute)
	if err != nil || !acquired {
		t.Fatalf("rebind lease acquired=%v err=%v", acquired, err)
	}
	leaseCtx := WithOperationLease(ctx, operationID, lease.Owner, lease.Fence)
	if err := repo.UpdateOperationSnapshot(leaseCtx, operationID, changedSnapshot); err == nil {
		t.Fatal("changed teardown receipt replaced the immutable intent")
	}
	if err := repo.ReleaseOperationLease(context.WithoutCancel(leaseCtx), lease); err != nil {
		t.Fatal(err)
	}
	second, status := call()
	if status != 200 || second.CloudTeardownReceipt == nil || !sameCloudTeardownReceipt(receipt, second.CloudTeardownReceipt) {
		t.Fatalf("replayed intent changed: status=%d receipt=%+v", status, second.CloudTeardownReceipt)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	restarted := NewSQLRepository(db)
	standing, err = restarted.GetOperation(ctx, operationID)
	if err != nil {
		t.Fatal(err)
	}
	hydrateOperationCloudComputeRefs(standing)
	if standing.CloudTeardownReceipt == nil || !sameCloudTeardownReceipt(receipt, standing.CloudTeardownReceipt) {
		t.Fatalf("restart lost teardown intent: %+v", standing.CloudTeardownReceipt)
	}
}

func TestRequestCloudTeardownAdoptedTargetIsDetachOnlyAndNeverProviderDestroy(t *testing.T) {
	db, repo := newAcquireSQLRepo(t)
	defer db.Close()
	ctx := context.Background()
	request := teardownTestRequest(t, "adopted-vps")
	seedAcquireOperation(t, ctx, repo, request)
	operationID := operationIDForRelease(request.ReleaseID)
	if _, err := db.Exec(`UPDATE release_operations SET status=? WHERE id=?`, OperationCanceled, operationID); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(repo, nil, nil, nil, nil).WithAuthorization(func(context.Context) error { return nil }).WithDurableOperations(true)
	req := mux.SetURLVars(httptest.NewRequest("POST", "/", nil), map[string]string{"operation_id": operationID})
	response := httptest.NewRecorder()
	h.RequestCloudTeardown(response, req)
	var operation Operation
	if err := json.Unmarshal(response.Body.Bytes(), &operation); err != nil {
		t.Fatalf("decode response: %v body=%s", err, response.Body.String())
	}
	if response.Code != 200 || operation.CloudTeardownReceipt == nil || operation.CloudTeardownReceipt.State != "pending" || operation.CloudTeardownReceipt.Provider != "adopted-vps" || !strings.Contains(operation.CloudTeardownReceipt.Reason, "awaiting exact Compute Manager") {
		t.Fatalf("adopted teardown response=%d operation=%+v", response.Code, operation)
	}
}

type releaseConsumerFixture struct {
	calls   int
	request CloudReleaseRequest
	result  *CloudReleaseResult
	err     error
}

func (f *releaseConsumerFixture) ReleaseCloudCompute(_ context.Context, request CloudReleaseRequest) (*CloudReleaseResult, error) {
	f.calls++
	f.request = request
	return f.result, f.err
}

func TestRequestCloudTeardownUsesStableCMReceiptAndReplaysAfterRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "release-receipt.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(internalreleases.Schema()); err != nil {
		t.Fatal(err)
	}
	repo := NewSQLRepository(db)
	ctx := context.Background()
	request := teardownTestRequest(t, "hetzner")
	seedAcquireOperation(t, ctx, repo, request)
	opID := operationIDForRelease(request.ReleaseID)
	if _, err = db.Exec(`UPDATE release_operations SET status=? WHERE id=?`, OperationComplete, opID); err != nil {
		t.Fatal(err)
	}
	owner := &instancev1.ReleaseReceipt{IdempotencyKey: "dm-release:" + opID, Owner: "deployment-manager", InstanceId: "cm-fixture-1", Provider: "hetzner", ProviderInstanceId: "hetzner-server-fixture-1", Action: "release", State: instancev1.ReleaseState_RELEASE_STATE_RELEASED, Reason: "exact release reconciled"}
	ownerJSON, _ := protojson.Marshal(owner)
	cm := &releaseConsumerFixture{result: &CloudReleaseResult{Receipt: ownerJSON}}
	h := NewHandler(repo, nil, nil, nil, nil).WithAuthorization(func(context.Context) error { return nil }).WithDurableOperations(true).WithCloudComputeReleaser(cm)
	call := func() (*Operation, int) {
		req := mux.SetURLVars(httptest.NewRequest("POST", "/", nil), map[string]string{"operation_id": opID})
		w := httptest.NewRecorder()
		h.RequestCloudTeardown(w, req)
		var op Operation
		if w.Code == 200 {
			if err := json.Unmarshal(w.Body.Bytes(), &op); err != nil {
				t.Fatal(err)
			}
		}
		return &op, w.Code
	}
	first, status := call()
	if status != 200 || first.CloudTeardownReceipt == nil || first.CloudTeardownReceipt.State != "released" || cm.calls != 1 {
		t.Fatalf("status=%d op=%+v cm=%+v", status, first, cm)
	}
	if cm.request.IdempotencyKey != "dm-release:"+opID || cm.request.InstanceID != "cm-fixture-1" || cm.request.Provider != "hetzner" || cm.request.ProviderInstanceID != "hetzner-server-fixture-1" {
		t.Fatalf("CM request=%+v", cm.request)
	}
	second, status := call()
	if status != 200 || !sameCloudTeardownReceipt(first.CloudTeardownReceipt, second.CloudTeardownReceipt) || cm.calls != 1 {
		t.Fatalf("replay changed/called CM: status=%d calls=%d receipt=%+v", status, cm.calls, second.CloudTeardownReceipt)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	got, err := NewSQLRepository(db).GetOperation(ctx, opID)
	if err != nil {
		t.Fatal(err)
	}
	hydrateOperationCloudComputeRefs(got)
	if got.CloudTeardownReceipt == nil || got.CloudTeardownReceipt.State != "released" || len(got.CloudTeardownReceipt.OwnerReceipt) == 0 {
		t.Fatalf("restart lost typed owner receipt: %+v", got.CloudTeardownReceipt)
	}
}

func TestRequestCloudTeardownMismatchedAndAbsentCMReceiptStayBlocked(t *testing.T) {
	for _, tc := range []struct {
		name    string
		receipt *instancev1.ReleaseReceipt
		err     error
	}{{name: "mismatch", receipt: &instancev1.ReleaseReceipt{IdempotencyKey: "wrong", Owner: "deployment-manager", InstanceId: "cm-fixture-1", Provider: "hetzner", ProviderInstanceId: "hetzner-server-fixture-1", Action: "release", State: instancev1.ReleaseState_RELEASE_STATE_RELEASED}}, {name: "absent", err: context.DeadlineExceeded}} {
		t.Run(tc.name, func(t *testing.T) {
			db, repo := newAcquireSQLRepo(t)
			defer db.Close()
			ctx := context.Background()
			request := teardownTestRequest(t, "hetzner")
			seedAcquireOperation(t, ctx, repo, request)
			opID := operationIDForRelease(request.ReleaseID)
			_, _ = db.Exec(`UPDATE release_operations SET status=? WHERE id=?`, OperationComplete, opID)
			var raw []byte
			if tc.receipt != nil {
				raw, _ = protojson.Marshal(tc.receipt)
			}
			cm := &releaseConsumerFixture{result: &CloudReleaseResult{Receipt: raw}, err: tc.err}
			h := NewHandler(repo, nil, nil, nil, nil).WithAuthorization(func(context.Context) error { return nil }).WithDurableOperations(true).WithCloudComputeReleaser(cm)
			req := mux.SetURLVars(httptest.NewRequest("POST", "/", nil), map[string]string{"operation_id": opID})
			w := httptest.NewRecorder()
			h.RequestCloudTeardown(w, req)
			var op Operation
			if err := json.Unmarshal(w.Body.Bytes(), &op); err != nil {
				t.Fatal(err)
			}
			if w.Code != 200 || op.CloudTeardownReceipt == nil || op.CloudTeardownReceipt.State == "released" {
				t.Fatalf("invalid receipt passed: code=%d op=%+v body=%s", w.Code, op, w.Body.String())
			}
		})
	}
}

func TestRequestCloudTeardownRejectsIncompleteReleaseBeforeIntent(t *testing.T) {
	db, repo := newAcquireSQLRepo(t)
	defer db.Close()
	ctx := context.Background()
	request := teardownTestRequest(t, "hetzner")
	seedAcquireOperation(t, ctx, repo, request)
	operationID := operationIDForRelease(request.ReleaseID)
	h := NewHandler(repo, nil, nil, nil, nil).WithAuthorization(func(context.Context) error { return nil }).WithDurableOperations(true)
	req := mux.SetURLVars(httptest.NewRequest("POST", "/", nil), map[string]string{"operation_id": operationID})
	response := httptest.NewRecorder()
	h.RequestCloudTeardown(response, req)
	standing, err := repo.GetOperation(ctx, operationID)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot durableDeployRequest
	if err := json.Unmarshal(standing.RequestSnapshot, &snapshot); err != nil {
		t.Fatal(err)
	}
	if response.Code != 409 || snapshot.CloudTeardownReceipt != nil {
		t.Fatalf("incomplete release status=%d teardown=%+v", response.Code, snapshot.CloudTeardownReceipt)
	}
}

func TestRequestCloudTeardownRefusesMissingTypedAcquisitionReceipt(t *testing.T) {
	db, repo := newAcquireSQLRepo(t)
	defer db.Close()
	ctx := context.Background()
	request := acquireDeployRequest()
	seedAcquireOperation(t, ctx, repo, request)
	operationID := operationIDForRelease(request.ReleaseID)
	if _, err := db.Exec(`UPDATE release_operations SET status=? WHERE id=?`, OperationComplete, operationID); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(repo, nil, nil, nil, nil).WithAuthorization(func(context.Context) error { return nil }).WithDurableOperations(true)
	req := mux.SetURLVars(httptest.NewRequest("POST", "/", nil), map[string]string{"operation_id": operationID})
	response := httptest.NewRecorder()
	h.RequestCloudTeardown(response, req)
	standing, err := repo.GetOperation(ctx, operationID)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot durableDeployRequest
	if err := json.Unmarshal(standing.RequestSnapshot, &snapshot); err != nil {
		t.Fatal(err)
	}
	if response.Code != 409 || snapshot.CloudTeardownReceipt != nil {
		t.Fatalf("missing typed target status=%d teardown=%+v", response.Code, snapshot.CloudTeardownReceipt)
	}
}

func teardownTestRequest(t *testing.T, provider string) DeployRequest {
	t.Helper()
	request := acquireDeployRequest()
	request.CloudComputeManagerInstanceID = "cm-fixture-1"
	request.CloudManifest, _ = bindCloudManifestInstance(request.CloudManifest, request.CloudComputeManagerInstanceID)
	request.CloudSpendQuoteIdempotencyKey = request.CloudAcquireIdempotencyKey
	source := fixtureAcquireResult(request).Receipt
	response := &instancev1.GetAcquireReceiptResponse{}
	if err := protojson.Unmarshal(source, response); err != nil {
		t.Fatal(err)
	}
	response.Instance.Provider = provider
	response.Instance.ProviderInstanceId = "hetzner-server-fixture-1"
	response.Instance.ExpiresAt = timestamppb.New(time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC))
	response.SpendReceipt.Currency, response.SpendReceipt.CapCurrency, response.SpendReceipt.TaxBasis, response.SpendReceipt.CapTaxBasis = "USD", "USD", "pre-tax", "pre-tax"
	response.SpendReceipt.TotalMinorPerMonth, response.SpendReceipt.CapMonthlyMinor = 709, 709
	response.SpendReceipt.PriceProvenanceFingerprint = "sha256:fixture-price"
	source, _ = protojson.Marshal(response)
	var err error
	request.CloudAcquireReceipt, err = newCloudAcquireReceipt(operationIDForRelease(request.ReleaseID), request, source, time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	return request
}
