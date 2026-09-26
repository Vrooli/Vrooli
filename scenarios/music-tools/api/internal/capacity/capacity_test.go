package capacity

import (
	"context"
	"testing"
)

type fakeBroker struct {
	verdict  Verdict
	claims   []string
	releases []string
}

func (f *fakeBroker) Claim(_ context.Context, owner string, _, _ uint64) (Verdict, error) {
	f.claims = append(f.claims, owner)
	return f.verdict, nil
}
func (f *fakeBroker) Release(_ context.Context, id string) error {
	f.releases = append(f.releases, id)
	return nil
}

func TestFreeVRAMUsesFreeNotTotal(t *testing.T) {
	if got := FreeVRAM(16, 10); got != 6 {
		t.Fatalf("got %d", got)
	}
}

func TestFreeVRAMFromInventoryUsesUsedBytes(t *testing.T) {
	raw := []byte(`{"gpus":[{"index":0,"vram_bytes":"17171480576","vram_used_bytes":"14534311936"}]}`)
	if got, err := FreeVRAMFromInventory(raw, 0); err != nil || got != 2637168640 {
		t.Fatalf("free=%d err=%v", got, err)
	}
}
func TestAdvisoryReclaimDegrades(t *testing.T) {
	if !ShouldDegrade(Verdict{Granted: true, ReclaimBytes: 1, Enforce: "advisory"}, 6) {
		t.Fatal("advisory reclaim should degrade")
	}
}

func TestAdmitReleasesClaimAndReturnsRung(t *testing.T) {
	fake := &fakeBroker{verdict: Verdict{ClaimID: "claim-1", Granted: true, AppliedRung: "offload-dit"}}
	release, meta, err := Admit(context.Background(), fake, "job-1", 7, 6)
	if err != nil {
		t.Fatal(err)
	}
	if len(fake.claims) != 1 || fake.claims[0] != "job-1" || meta["applied_rung"] != "offload-dit" {
		t.Fatalf("claim=%v meta=%v", fake.claims, meta)
	}
	release()
	if len(fake.releases) != 1 || fake.releases[0] != "claim-1" {
		t.Fatalf("releases=%v", fake.releases)
	}
}

func TestAdmitRejectsDeniedClaim(t *testing.T) {
	fake := &fakeBroker{verdict: Verdict{Reason: "insufficient free VRAM"}}
	if _, _, err := Admit(context.Background(), fake, "job-1", 7, 6); err == nil {
		t.Fatal("denied claim was admitted")
	}
}
