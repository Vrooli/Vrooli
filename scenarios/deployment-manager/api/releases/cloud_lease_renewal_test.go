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

type refusedLeaseRenewer struct {
	calls   int
	receipt json.RawMessage
}

func (f *refusedLeaseRenewer) RenewCloudLease(_ context.Context, request CloudLeaseRenewalRequest) (*CloudLeaseRenewalResult, error) {
	f.calls++
	if f.receipt == nil {
		now := timestamppb.Now()
		owner := &instancev1.RenewalReceipt{IdempotencyKey: request.IdempotencyKey, Owner: "deployment-manager", InstanceId: request.InstanceID, Provider: request.Provider, ProviderInstanceId: request.ProviderInstanceID, PriorExpiresAt: timestamppb.New(request.OldExpiresAt), RequestedUntil: timestamppb.New(request.RequestedUntil), SpendIdempotencyKey: request.QuoteKey, QuoteFingerprint: request.QuoteFingerprint, State: instancev1.RenewalState_RENEWAL_STATE_REFUSED, Reason: "provider adapter has no exact lease-renewal capability; no provider call was made", CreatedAt: now, UpdatedAt: now}
		encoded, _ := protojson.Marshal(owner)
		f.receipt = encoded
	}
	return &CloudLeaseRenewalResult{Receipt: f.receipt}, nil
}

func TestCloudLeaseRenewalConsumerPersistsRefusalAndReplaysExactOwnerReceipt(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "lease-renewal.db")
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
	opID := operationIDForRelease(request.ReleaseID)
	if _, err := db.Exec(`UPDATE release_operations SET status=? WHERE id=?`, OperationComplete, opID); err != nil {
		t.Fatal(err)
	}
	owner := &refusedLeaseRenewer{}
	h := NewHandler(repo, nil, nil, nil, nil).WithAuthorization(func(context.Context) error { return nil }).WithCloudLeaseRenewer(owner).WithDurableOperations(true)
	call := func() *Operation {
		req := mux.SetURLVars(httptest.NewRequest("POST", "/api/v1/release-operations/"+opID+"/request-cloud-lease-renewal", nil), map[string]string{"operation_id": opID})
		w := httptest.NewRecorder()
		h.RequestCloudLeaseRenewal(w, req)
		if w.Code != 200 {
			t.Fatalf("renewal standing status=%d body=%s", w.Code, w.Body.String())
		}
		var op Operation
		if err := json.Unmarshal(w.Body.Bytes(), &op); err != nil {
			t.Fatal(err)
		}
		return &op
	}
	first := call()
	r := first.CloudLeaseRenewalReceipt
	if r == nil || r.State != "blocked" || r.RenewalKey != "dm-lease-renew:"+opID || r.QuoteFingerprint == "" || r.OwnerReceipt != nil || !strings.Contains(r.Reason, "No fresh incremental renewal quote") || owner.calls != 0 {
		t.Fatalf("lease standing = %+v", r)
	}
	wire := first.Proto().GetCloudLeaseRenewalReceipt()
	if wire == nil || wire.GetState() != releasesv1.CloudLeaseRenewalState_CLOUD_LEASE_RENEWAL_STATE_BLOCKED || wire.GetOldExpiresAt() == nil || wire.GetComputeManagerReceipt() != nil {
		t.Fatalf("typed readback = %+v", wire)
	}
	readback := operationFromProto(first.Proto())
	if readback.CloudLeaseRenewalReceipt == nil || !sameCloudLeaseRenewalReceipt(r, readback.CloudLeaseRenewalReceipt) {
		t.Fatalf("typed operation round-trip lost exact owner receipt: %+v", readback.CloudLeaseRenewalReceipt)
	}
	second := call()
	if !sameCloudLeaseRenewalReceipt(r, second.CloudLeaseRenewalReceipt) || owner.calls != 0 {
		t.Fatalf("duplicate request changed receipt: %+v", second.CloudLeaseRenewalReceipt)
	}
	standing, err := repo.GetOperation(ctx, opID)
	if err != nil {
		t.Fatal(err)
	}
	var changed durableDeployRequest
	if err := json.Unmarshal(standing.RequestSnapshot, &changed); err != nil {
		t.Fatal(err)
	}
	changed.CloudLeaseRenewalReceipt.QuoteFingerprint = "sha256:rebound"
	snapshot, _ := json.Marshal(changed)
	lease, acquired, err := repo.AcquireOperationLease(ctx, opID, "lease-renew-rebind-test", time.Minute)
	if err != nil || !acquired {
		t.Fatalf("fence acquired=%v err=%v", acquired, err)
	}
	leaseCtx := WithOperationLease(ctx, opID, lease.Owner, lease.Fence)
	if err := repo.UpdateOperationSnapshot(leaseCtx, opID, snapshot); err == nil {
		t.Fatal("cost/quote fact rebind was accepted")
	}
	_ = repo.ReleaseOperationLease(context.WithoutCancel(leaseCtx), lease)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	restarted, err := NewSQLRepository(db).GetOperation(ctx, opID)
	if err != nil {
		t.Fatal(err)
	}
	hydrateOperationCloudComputeRefs(restarted)
	if !sameCloudLeaseRenewalReceipt(r, restarted.CloudLeaseRenewalReceipt) {
		t.Fatalf("restart lost standing: %+v", restarted.CloudLeaseRenewalReceipt)
	}
}

func TestCloudLeaseRenewalStandingExpiredStillBlocked(t *testing.T) {
	request := teardownTestRequest(t, "hetzner")
	durable := durableFromDeployRequest(request)
	dueNow := time.Date(2026, 11, 25, 11, 40, 0, 0, time.UTC)
	due := cloudLeaseRenewalDueStanding(operationIDForRelease(request.ReleaseID), durable, dueNow)
	if due == nil || due.State != "due" {
		t.Fatalf("operation readback should show due standing inside the renewal window: %+v", due)
	}
	r, err := pendingCloudLeaseRenewal(operationIDForRelease(request.ReleaseID), durable, time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	durable.CloudLeaseRenewalReceipt = r
	if !strings.Contains(r.Reason, "expired") || validateCloudLeaseRenewalReceipt(operationIDForRelease(request.ReleaseID), durable) != nil {
		t.Fatalf("expired standing = %+v", r)
	}
	r.NewExpiresAt = time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	durable.CloudLeaseRenewalReceipt = r
	if err := validateCloudLeaseRenewalReceipt(operationIDForRelease(request.ReleaseID), durable); err == nil {
		t.Fatal("new expiry without CM receipt was accepted")
	}
}

func durableFromDeployRequest(request DeployRequest) durableDeployRequest {
	snapshot, _ := snapshotForDeployRequest(request)
	var durable durableDeployRequest
	_ = json.Unmarshal(snapshot, &durable)
	durable.CloudLeaseRenewalReceipt = cloneCloudLeaseRenewalReceipt(request.CloudLeaseRenewalReceipt)
	return durable
}
