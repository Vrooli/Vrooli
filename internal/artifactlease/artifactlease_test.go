package artifactlease

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var (
	base       = time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	testOwner  = Owner{Node: "swarminator", User: "operator"}
	testModule = "/fixture/scenarios/example/cli"
)

// artifactFixture creates a stand-in installed binary in a temp directory. No
// real scenario, CLI, or install root is involved anywhere in this package's
// tests.
func artifactFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "example-cli")
	if err := os.WriteFile(path, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestClaimRecordsOwnershipAndBumpsGeneration(t *testing.T) {
	artifact := artifactFixture(t)

	first, err := Claim(artifact, testOwner, testModule, time.Hour, base)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if first.Generation != 1 {
		t.Fatalf("first generation = %d, want 1", first.Generation)
	}

	second, err := Claim(artifact, testOwner, testModule, time.Hour, base.Add(time.Minute))
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if second.Generation != 2 {
		t.Fatalf("second generation = %d, want 2; a replace must advance the generation", second.Generation)
	}

	loaded, found, err := Load(artifact)
	if err != nil || !found {
		t.Fatalf("Load: %v found=%v", err, found)
	}
	if loaded.Owner.Node != "swarminator" || loaded.OwnerModule != testModule {
		t.Fatalf("lease did not persist ownership: %+v", loaded)
	}
}

// The guard this replaces asked "is the binary running?", which a CLI never is.
// An observed absence must record, never delete.
func TestNoteOwnerMissingRecordsWithoutReclaiming(t *testing.T) {
	artifact := artifactFixture(t)
	if _, err := Claim(artifact, testOwner, testModule, 0, base); err != nil {
		t.Fatal(err)
	}

	lease, err := NoteOwnerMissing(artifact, base.Add(time.Hour))
	if err != nil {
		t.Fatalf("NoteOwnerMissing: %v", err)
	}
	if lease.OwnerMissingSince == "" || lease.Observations != 1 {
		t.Fatalf("absence not recorded: %+v", lease)
	}
	if _, err := os.Stat(artifact); err != nil {
		t.Fatalf("recording an absence must not touch the artifact: %v", err)
	}
}

// The grace window measures how long the absence has lasted, not how recently
// it was last noticed. Re-stamping the timestamp on every sighting would reset
// the clock forever and the artifact would never become reclaimable.
func TestRepeatedObservationsDoNotResetTheClock(t *testing.T) {
	artifact := artifactFixture(t)
	if _, err := Claim(artifact, testOwner, testModule, 0, base); err != nil {
		t.Fatal(err)
	}

	first, _ := NoteOwnerMissing(artifact, base.Add(time.Hour))
	later, _ := NoteOwnerMissing(artifact, base.Add(5*time.Hour))

	if later.OwnerMissingSince != first.OwnerMissingSince {
		t.Fatalf("absence timestamp moved from %q to %q", first.OwnerMissingSince, later.OwnerMissingSince)
	}
	if later.Observations != 2 {
		t.Fatalf("observations = %d, want 2", later.Observations)
	}
}

// [REQ:VROOLI-ARTIFACT-LEASE]
// This is the behaviour the whole phase exists for: a scenario that disappears
// and is recreated inside the grace window keeps its CLI. Concurrent agents
// regenerate scenarios routinely, and treating that instant as authority to
// delete is what reclaimed freshly built binaries.
func TestOwnerReturningInsideGraceClearsTheAbsence(t *testing.T) {
	artifact := artifactFixture(t)
	if _, err := Claim(artifact, testOwner, testModule, 0, base); err != nil {
		t.Fatal(err)
	}
	if _, err := NoteOwnerMissing(artifact, base.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := NoteOwnerMissing(artifact, base.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}

	// The scenario is regenerated, which reinstalls its CLI.
	if _, err := Claim(artifact, testOwner, testModule, 0, base.Add(3*time.Hour)); err != nil {
		t.Fatal(err)
	}

	lease, found, err := Load(artifact)
	if err != nil || !found {
		t.Fatalf("Load: %v found=%v", err, found)
	}
	if lease.OwnerMissingSince != "" || lease.Observations != 0 {
		t.Fatalf("a reinstall must clear the recorded absence: %+v", lease)
	}

	// Even long after the original absence began, the artifact is protected.
	eligibility := EvaluateReclaim(lease, true, base.Add(96*time.Hour), DefaultGrace)
	if eligibility.Reclaimable {
		t.Fatal("a reinstalled artifact was still reclaimable on the strength of an old absence")
	}
}

func TestNoteOwnerPresentClearsAnAbsence(t *testing.T) {
	artifact := artifactFixture(t)
	if _, err := Claim(artifact, testOwner, testModule, 0, base); err != nil {
		t.Fatal(err)
	}
	if _, err := NoteOwnerMissing(artifact, base.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := NoteOwnerPresent(artifact); err != nil {
		t.Fatalf("NoteOwnerPresent: %v", err)
	}

	lease, _, _ := Load(artifact)
	if lease.OwnerMissingSince != "" || lease.Observations != 0 {
		t.Fatalf("absence survived the owner reappearing: %+v", lease)
	}
}

func TestEvaluateReclaimRequiresTheFullGracePath(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		lease       Lease
		found       bool
		now         time.Time
		reclaimable bool
		reason      string
	}{
		{
			name:   "no lease at all",
			found:  false,
			now:    base,
			reason: "not been observed",
		},
		{
			name:   "lease still held",
			lease:  Lease{Generation: 3, ExpiresAt: base.Add(time.Hour).Format(time.RFC3339Nano), OwnerMissingSince: base.Add(-48 * time.Hour).Format(time.RFC3339Nano), Observations: 9},
			found:  true,
			now:    base,
			reason: "held until",
		},
		{
			name:   "never observed missing",
			lease:  Lease{Generation: 1},
			found:  true,
			now:    base,
			reason: "never been observed missing",
		},
		{
			name:   "only one observation",
			lease:  Lease{Generation: 1, OwnerMissingSince: base.Add(-48 * time.Hour).Format(time.RFC3339Nano), Observations: 1},
			found:  true,
			now:    base,
			reason: "observed 1 time",
		},
		{
			name:   "inside the grace window",
			lease:  Lease{Generation: 1, OwnerMissingSince: base.Add(-time.Hour).Format(time.RFC3339Nano), Observations: 5},
			found:  true,
			now:    base,
			reason: "missing for 1h0m0s",
		},
		{
			name:        "absent long enough and seen repeatedly",
			lease:       Lease{Generation: 7, OwnerMissingSince: base.Add(-48 * time.Hour).Format(time.RFC3339Nano), Observations: 4},
			found:       true,
			now:         base,
			reclaimable: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			got := EvaluateReclaim(testCase.lease, testCase.found, testCase.now, DefaultGrace)
			if got.Reclaimable != testCase.reclaimable {
				t.Fatalf("Reclaimable = %v, want %v (reason %q)", got.Reclaimable, testCase.reclaimable, got.Reason)
			}
			if !testCase.reclaimable {
				if got.Reason == "" {
					t.Fatal("a refusal with no reason turns reaper behaviour into folklore")
				}
				if !strings.Contains(got.Reason, testCase.reason) {
					t.Fatalf("reason = %q, want it to mention %q", got.Reason, testCase.reason)
				}
			}
		})
	}
}

// A host whose clock is corrected backwards must not suddenly find every
// artifact eligible.
func TestClockMovingBackwardsDoesNotMakeEverythingReclaimable(t *testing.T) {
	lease := Lease{
		Generation:        2,
		OwnerMissingSince: base.Format(time.RFC3339Nano),
		Observations:      5,
	}

	got := EvaluateReclaim(lease, true, base.Add(-72*time.Hour), DefaultGrace)
	if got.Reclaimable {
		t.Fatal("a backwards clock made a fresh absence look long expired")
	}
}

// An artifact installed before this protocol has no lease. That must not read
// as "unowned, reclaim freely" -- the absence clock starts when it is first
// observed, exactly as it would for a new artifact.
func TestArtifactPredatingTheProtocolGetsAGracePeriod(t *testing.T) {
	artifact := artifactFixture(t)

	lease, err := NoteOwnerMissing(artifact, base)
	if err != nil {
		t.Fatalf("NoteOwnerMissing: %v", err)
	}
	if got := EvaluateReclaim(lease, true, base, DefaultGrace); got.Reclaimable {
		t.Fatal("an artifact with no prior lease was immediately reclaimable")
	}
	if _, err := NoteOwnerMissing(artifact, base.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}

	reloaded, found, _ := Load(artifact)
	if got := EvaluateReclaim(reloaded, found, base.Add(25*time.Hour), DefaultGrace); !got.Reclaimable {
		t.Fatalf("still not reclaimable after the full grace path: %q", got.Reason)
	}
}

// A lease that cannot be parsed is not permission to reclaim.
func TestUnreadableLeaseIsNotPermissionToReclaim(t *testing.T) {
	artifact := artifactFixture(t)
	if err := os.WriteFile(Path(artifact), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, found, err := Load(artifact)
	if err == nil {
		t.Fatal("a corrupt lease decoded without error")
	}
	if found {
		t.Fatal("a corrupt lease reported itself present, which would let a caller act on a zero value")
	}
}

// A corrupt lease must not make an artifact permanently uninstallable.
func TestClaimReplacesACorruptLease(t *testing.T) {
	artifact := artifactFixture(t)
	if err := os.WriteFile(Path(artifact), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}

	lease, err := Claim(artifact, testOwner, testModule, time.Hour, base)
	if err != nil {
		t.Fatalf("Claim over a corrupt lease: %v", err)
	}
	if lease.Generation != 1 {
		t.Fatalf("generation = %d, want a fresh 1", lease.Generation)
	}
}

func TestRemoveIsIdempotent(t *testing.T) {
	artifact := artifactFixture(t)
	if _, err := Claim(artifact, testOwner, testModule, time.Hour, base); err != nil {
		t.Fatal(err)
	}
	if err := Remove(artifact); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if err := Remove(artifact); err != nil {
		t.Fatalf("second Remove should be a no-op: %v", err)
	}
}
