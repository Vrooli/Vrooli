package capacity

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

func TestOperationLedgerReferencePreservesAuthorityAcrossDetachedChecks(t *testing.T) {
	opts := operationRAMFixture(t)
	opts.Admission.OpenStore = nil
	opts.Ledger = LedgerReference(filepath.Join(t.TempDir(), "capacity.db"))
	ctx := context.Background()
	claim, err := ReserveOperationRAM(ctx, opts)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(opts.Ledger)
	if err != nil {
		t.Fatal(err)
	}
	var retained LedgerReference
	if err := json.Unmarshal(data, &retained); err != nil {
		t.Fatal(err)
	}
	if retained != opts.Ledger {
		t.Fatal("retained reference changed capacity authority")
	}
	if err := VerifyOperationRAM(ctx, opts, 30*time.Second); err != nil {
		t.Fatal(err)
	}
	foreign := opts
	foreign.InstanceID = "another-operation"
	if err := VerifyOperationRAM(ctx, foreign, 0); err == nil {
		t.Fatal("foreign operation borrowed the retained lease")
	}
	if err := ReleaseOperationRAM(ctx, foreign); err == nil {
		t.Fatal("foreign operation released the retained lease")
	}
	if err := VerifyOperationRAM(ctx, opts, 0); err != nil {
		t.Fatalf("rejected foreign release disturbed current owner: %v", err)
	}
	if err := ReleaseOperationRAM(ctx, opts); err != nil {
		t.Fatal(err)
	}
	if err := VerifyOperationRAM(ctx, opts, 0); err == nil {
		t.Fatal("released lease still authorized exec")
	}
	store, err := openOperationReservationStore(ctx, opts)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	final, err := store.GetClaim(ctx, claim.ClaimID)
	if err != nil || final.Status != StatusReleased {
		t.Fatalf("release receipt: %+v %v", final, err)
	}
}
