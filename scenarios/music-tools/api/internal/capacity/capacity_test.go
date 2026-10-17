package capacity

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestAdvisoryReclaimDegrades(t *testing.T) {
	if !ShouldDegrade(Verdict{Granted: true, ReclaimBytes: 1, Enforce: "advisory"}) {
		t.Fatal("advisory reclaim should degrade")
	}
}

func TestAdmitReleasesClaimAndReturnsRung(t *testing.T) {
	fake := &fakeBroker{verdict: Verdict{ClaimID: "claim-1", Granted: true, AppliedRung: "offload-dit"}}
	release, meta, err := Admit(context.Background(), func(error) {}, fake, "job-1", 7, 6)
	if err != nil {
		t.Fatal(err)
	}
	if len(fake.claims) != 1 || fake.claims[0] != "job-1" || meta["capacity_admitted_rung"] != "offload-dit" || meta["applied_rung"] != "" {
		t.Fatalf("claim=%v meta=%v", fake.claims, meta)
	}
	release()
	if len(fake.releases) != 1 || fake.releases[0] != "claim-1" {
		t.Fatalf("releases=%v", fake.releases)
	}
}

func TestAdmitRejectsDeniedClaim(t *testing.T) {
	fake := &fakeBroker{verdict: Verdict{Reason: "insufficient free VRAM"}}
	if _, _, err := Admit(context.Background(), func(error) {}, fake, "job-1", 7, 6); err == nil {
		t.Fatal("denied claim was admitted")
	}
}

func TestCLIBrokerHonorsAdmissionVerdicts(t *testing.T) {
	for _, tc := range []struct {
		kind     string
		admitted bool
	}{
		{"grant", true},
		{"degrade", true},
		{"queue", false},
		{"deny", false},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			command := filepath.Join(t.TempDir(), "broker")
			response := fmt.Sprintf(`{"verdict":{"kind":%q,"step":"offload-dit","granted_bytes":6442450944},"claim":{"claim_id":"bounded-job"},"enforce":"advisory"}`, tc.kind)
			script := "#!/bin/sh\ncat <<'BROKER_RESPONSE'\n" + response + "\nBROKER_RESPONSE\n"
			if err := os.WriteFile(command, []byte(script), 0o700); err != nil {
				t.Fatal(err)
			}
			broker := CLICapacityBroker{Command: command}
			release, meta, err := Admit(context.Background(), func(error) {}, broker, "music-job", 7516192768, 6442450944)
			if tc.admitted {
				if err != nil {
					t.Fatalf("admitted %s verdict rejected: %v", tc.kind, err)
				}
				defer release()
				if meta["capacity_claim_id"] != "bounded-job" || meta["capacity_admitted_rung"] != "offload-dit" || meta["capacity_degrade"] != "offload-dit" || meta["applied_rung"] != "" {
					t.Fatalf("lost reduced-memory admission facts: %v", meta)
				}
			} else if err == nil {
				release()
				t.Fatalf("non-admitted %s verdict accepted", tc.kind)
			}
		})
	}
}

func TestAdmissionRejectsUnknownOrMissingRungAndReleasesClaim(t *testing.T) {
	for _, rung := range []string{"unknown", ""} {
		t.Run(rung, func(t *testing.T) {
			broker := &fakeBroker{verdict: Verdict{Granted: true, ClaimID: "lease", AppliedRung: rung}}
			if release, meta, err := Admit(context.Background(), func(error) {}, broker, "job", 7, 6); err == nil || release != nil || meta != nil {
				t.Fatalf("invalid admission: release=%v meta=%v err=%v", release != nil, meta, err)
			}
			if len(broker.releases) != 1 || broker.releases[0] != "lease" {
				t.Fatalf("releases=%v", broker.releases)
			}
		})
	}
}

func TestAdvisoryReclaimRecordsEffectiveReducedAdmission(t *testing.T) {
	broker := &fakeBroker{verdict: Verdict{Granted: true, ClaimID: "lease", AppliedRung: "full", ReclaimBytes: 1, Enforce: "advisory"}}
	release, meta, err := Admit(context.Background(), func(error) {}, broker, "job", 7, 6)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if meta["capacity_admitted_rung"] != "offload-dit" || meta["capacity_degrade"] != "offload-dit" || meta["applied_rung"] != "" {
		t.Fatalf("metadata overstates actual/effective posture: %v", meta)
	}
}

func (f *fakeBroker) Heartbeat(context.Context, string) error { return nil }

type renewalFailureBroker struct {
	fakeBroker
	attempted chan struct{}
}

func (f *renewalFailureBroker) Heartbeat(context.Context, string) error {
	close(f.attempted)
	return fmt.Errorf("lost lease")
}

func TestLeaseRenewalFailureCancelsOwnedWorkAndReleaseIsOnce(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	broker := &renewalFailureBroker{attempted: make(chan struct{})}
	release := maintain(ctx, cancel, broker, "claim", time.Millisecond)
	defer release()
	select {
	case <-ctx.Done():
	case <-time.After(time.Second):
		t.Fatal("lease loss did not cancel work")
	}
	if !strings.Contains(context.Cause(ctx).Error(), "lease renewal failed") {
		t.Fatalf("cause %v", context.Cause(ctx))
	}
	release()
	release()
	if len(broker.releases) != 1 {
		t.Fatalf("release count %d", len(broker.releases))
	}
}

func TestAdmissionRequiresARealClaimIdentity(t *testing.T) {
	broker := &fakeBroker{verdict: Verdict{Granted: true, AppliedRung: "full"}}
	if release, _, err := Admit(context.Background(), func(error) {}, broker, "job", 7, 6); err == nil {
		release()
		t.Fatal("unowned admission accepted")
	}
}
