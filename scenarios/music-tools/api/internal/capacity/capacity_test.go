package capacity

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeBroker struct {
	verdict  Verdict
	claims   []string
	releases []ClaimRef
}

func (f *fakeBroker) Claim(_ context.Context, owner string, _, _ uint64) (Verdict, error) {
	f.claims = append(f.claims, owner)
	return f.verdict, nil
}

func (f *fakeBroker) Release(_ context.Context, ref ClaimRef) error {
	f.releases = append(f.releases, ref)
	return nil
}

func TestAdvisoryReclaimDegrades(t *testing.T) {
	if !ShouldDegrade(Verdict{Granted: true, ReclaimBytes: 1, Enforce: "advisory"}) {
		t.Fatal("advisory reclaim should degrade")
	}
}

func TestAdmitReleasesClaimAndReturnsRung(t *testing.T) {
	fake := &fakeBroker{verdict: Verdict{Generation: 11, ClaimID: "claim-1", Granted: true, AppliedRung: "offload-dit"}}
	release, meta, err := Admit(context.Background(), func(error) {}, fake, "job-1", 7, 6)
	if err != nil {
		t.Fatal(err)
	}
	if len(fake.claims) != 1 || fake.claims[0] != "job-1" || meta["capacity_admitted_rung"] != "offload-dit" || meta["applied_rung"] != "" {
		t.Fatalf("claim=%v meta=%v", fake.claims, meta)
	}
	release()
	if len(fake.releases) != 1 || fake.releases[0] != (ClaimRef{ClaimID: "claim-1", Generation: 11}) {
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
			response := fmt.Sprintf(`{"verdict":{"kind":%q,"step":"offload-dit","granted_bytes":6442450944},"claim":{"claim_id":"bounded-job","generation":11},"enforce":"advisory"}`, tc.kind)
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
			broker := &fakeBroker{verdict: Verdict{Granted: true, ClaimID: "lease", Generation: 11, AppliedRung: rung}}
			if release, meta, err := Admit(context.Background(), func(error) {}, broker, "job", 7, 6); err == nil || release != nil || meta != nil {
				t.Fatalf("invalid admission: release=%v meta=%v err=%v", release != nil, meta, err)
			}
			if len(broker.releases) != 1 || broker.releases[0] != (ClaimRef{ClaimID: "lease", Generation: 11}) {
				t.Fatalf("releases=%v", broker.releases)
			}
		})
	}
}

func TestAdvisoryReclaimRecordsEffectiveReducedAdmission(t *testing.T) {
	broker := &fakeBroker{verdict: Verdict{Granted: true, ClaimID: "lease", Generation: 11, AppliedRung: "full", ReclaimBytes: 1, Enforce: "advisory"}}
	release, meta, err := Admit(context.Background(), func(error) {}, broker, "job", 7, 6)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if meta["capacity_admitted_rung"] != "offload-dit" || meta["capacity_degrade"] != "offload-dit" || meta["applied_rung"] != "" {
		t.Fatalf("metadata overstates actual/effective posture: %v", meta)
	}
}

func (f *fakeBroker) Heartbeat(context.Context, ClaimRef) error { return nil }

type renewalFailureBroker struct {
	fakeBroker
	attempted chan struct{}
}

func (f *renewalFailureBroker) Heartbeat(context.Context, ClaimRef) error {
	close(f.attempted)
	return fmt.Errorf("lost lease")
}

func TestLeaseRenewalFailureCancelsOwnedWorkAndReleaseIsOnce(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)
	broker := &renewalFailureBroker{attempted: make(chan struct{})}
	release := maintain(ctx, cancel, broker, ClaimRef{ClaimID: "claim", Generation: 11}, time.Millisecond)
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

func TestCLIAdmissionRetainsReturnedGeneration(t *testing.T) {
	command := filepath.Join(t.TempDir(), "broker")
	script := `#!/bin/sh
printf '%s\n' '{"verdict":{"kind":"grant","step":"full"},"claim":{"claim_id":"owned-claim","generation":37}}'
`
	if err := os.WriteFile(command, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)
	release, meta, err := Admit(ctx, cancel, CLICapacityBroker{Command: command}, "job", 7, 6)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if meta["capacity_claim_id"] != "owned-claim" || meta["capacity_claim_generation"] != "37" {
		t.Fatalf("acquired lease generation lost: %v", meta)
	}
}

func TestAdmissionRejectsMissingGeneration(t *testing.T) {
	broker := &fakeBroker{verdict: Verdict{Granted: true, ClaimID: "owned-claim", AppliedRung: "full"}}
	release, _, err := Admit(t.Context(), func(error) {}, broker, "job", 7, 6)
	if release != nil {
		defer release()
	}
	if err == nil || !strings.Contains(err.Error(), "generation") {
		t.Fatalf("admitted claim without its generation: %v", err)
	}
}

func TestCLIBrokerUsesCapturedGenerationAndCanonicalReleaseProtocol(t *testing.T) {
	dir := t.TempDir()
	command, calls := filepath.Join(dir, "broker"), filepath.Join(dir, "calls")
	script := fmt.Sprintf(`#!/bin/sh
printf '%%s\n' "$*" >> %q
case "$2" in
claim) printf '%%s\n' '{"verdict":{"kind":"grant","step":"full"},"claim":{"claim_id":"owned-claim","generation":37}}' ;;
heartbeat)
 if [ "$#" -ne 9 ] || [ "$3" != '--claim-id' ] || [ "$4" != 'owned-claim' ] || [ "$5" != '--generation' ] || [ "$6" != '37' ] || [ "$7" != '--ttl' ] || [ "$8" != '10m' ] || [ "$9" != '--json' ]; then
  printf 'stale_generation: expected owned-claim/37\n' >&2
  exit 7
 fi
 printf '%%s\n' '{"claim_id":"owned-claim","generation":37}' ;;
release)
 if [ "$#" -ne 4 ] || [ "$3" != '--claim-id' ] || [ "$4" != 'owned-claim' ]; then
  printf 'release protocol is captured-ID only\n' >&2
  exit 8
 fi ;;
*) exit 9 ;;
esac
`, calls)
	if err := os.WriteFile(command, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	broker := CLICapacityBroker{Command: command}
	v, err := broker.Claim(t.Context(), "job", 7, 6)
	if err != nil {
		t.Fatal(err)
	}
	ref := ClaimRef{ClaimID: v.ClaimID, Generation: v.Generation}
	for range 2 {
		if err := broker.Heartbeat(t.Context(), ref); err != nil {
			t.Fatal(err)
		}
	}
	stale := ref
	stale.Generation--
	if err := broker.Heartbeat(t.Context(), stale); err == nil || !strings.Contains(err.Error(), "stale_generation") {
		t.Fatalf("stale lease accepted or cause hidden: %v", err)
	}
	if err := broker.Release(t.Context(), ref); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 5 || !strings.HasPrefix(lines[0], "capacity claim ") || lines[1] != lines[2] || lines[4] != "capacity release --claim-id owned-claim" {
		t.Fatalf("unexpected extra lookup/mutation or changed identity: %s", data)
	}
}

type recordingLeaseBroker struct {
	mu         sync.Mutex
	beats      chan ClaimRef
	references []ClaimRef
	released   []ClaimRef
}

func (b *recordingLeaseBroker) Claim(context.Context, string, uint64, uint64) (Verdict, error) {
	return Verdict{Granted: true, ClaimID: "owned", Generation: 61, AppliedRung: "full"}, nil
}
func (b *recordingLeaseBroker) Heartbeat(_ context.Context, ref ClaimRef) error {
	b.mu.Lock()
	b.references = append(b.references, ref)
	b.mu.Unlock()
	select {
	case b.beats <- ref:
	default:
	}
	return nil
}
func (b *recordingLeaseBroker) Release(_ context.Context, ref ClaimRef) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.released = append(b.released, ref)
	return nil
}

func TestMaintainedLeaseNeverSubstitutesIdentityAndReleasesOnce(t *testing.T) {
	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)
	broker := &recordingLeaseBroker{beats: make(chan ClaimRef, 8)}
	v, err := broker.Claim(ctx, "job", 7, 6)
	if err != nil {
		t.Fatal(err)
	}
	ref := ClaimRef{ClaimID: v.ClaimID, Generation: v.Generation}
	release := maintain(ctx, cancel, broker, ref, time.Millisecond)
	defer release()
	for range 2 {
		select {
		case got := <-broker.beats:
			if got != ref {
				t.Fatalf("heartbeat substituted lease: %+v", got)
			}
		case <-time.After(time.Second):
			t.Fatal("heartbeat not observed")
		}
	}
	release()
	release()
	broker.mu.Lock()
	defer broker.mu.Unlock()
	if len(broker.released) != 1 || broker.released[0] != ref {
		t.Fatalf("release changed ownership/count: %+v", broker.released)
	}
	for _, got := range broker.references {
		if got != ref {
			t.Fatalf("maintain changed lease: %+v", got)
		}
	}
	if ctx.Err() != nil {
		t.Fatalf("valid heartbeats canceled work: %v", context.Cause(ctx))
	}
}

func TestCLIBrokerRefusesMissingGenerationBeforeCommandAndBoundsFailureStderr(t *testing.T) {
	command, marker := filepath.Join(t.TempDir(), "broker"), filepath.Join(t.TempDir(), "called")
	message := "stale_generation witness: " + strings.Repeat("x", 9000)
	script := fmt.Sprintf("#!/bin/sh\ntouch %q\nprintf '%%s' '%s' >&2\nexit 7\n", marker, message)
	if err := os.WriteFile(command, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	broker := CLICapacityBroker{Command: command}
	for _, generation := range []int64{0, -1} {
		ref := ClaimRef{ClaimID: "owned", Generation: generation}
		if err := broker.Heartbeat(t.Context(), ref); err == nil {
			t.Fatal("invalid generation heartbeat executed")
		}
		if err := broker.Release(t.Context(), ref); err == nil {
			t.Fatal("release accepted an uncaptured identity")
		}
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("invalid lease invoked command: %v", err)
	}
	err := broker.Heartbeat(t.Context(), ClaimRef{ClaimID: "owned", Generation: 37})
	if err == nil || !strings.Contains(err.Error(), "stale_generation witness") || !strings.Contains(err.Error(), "stderr truncated") || len(err.Error()) > 4400 {
		t.Fatalf("failure lost diagnostic or exceeded budget: %v", err)
	}
}
