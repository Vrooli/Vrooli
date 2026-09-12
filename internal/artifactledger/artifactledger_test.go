package artifactledger

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func testLedger(t *testing.T) *Ledger {
	t.Helper()
	return NewAt(filepath.Join(t.TempDir(), "removal-receipts")).
		WithClock(func() time.Time { return time.Date(2026, 8, 25, 19, 1, 34, 0, time.UTC) }).
		WithIdentity(func() Identity {
			return Identity{Node: "swarminator", User: "operator", PID: 4242, Process: "abc123"}
		})
}

// removalFixture creates a real artifact, because the seam locks its family
// and refuses to record anything for a path that was never there.
func removalFixture(t *testing.T) Removal {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "vrooli-onboarding")
	if err := os.WriteFile(path, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	return Removal{
		Path:      path,
		Kind:      "binary",
		Component: "cliinstall.RemoveScenarioCLI",
		Predicate: "scenario source deleted by operator command",
	}
}

// The whole package exists because removals left no trace. A successful removal
// must produce a complete, readable pair of records.
func TestGuardRecordsIntentAndOutcome(t *testing.T) {
	ledger := testLedger(t)

	if err := ledger.Guard(removalFixture(t), func() error { return nil }); err != nil {
		t.Fatalf("Guard: %v", err)
	}

	receipts, err := ledger.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if len(receipts) != 2 {
		t.Fatalf("got %d receipts, want an intent and an outcome: %+v", len(receipts), receipts)
	}
	if receipts[0].Outcome != OutcomeIntent || receipts[1].Outcome != OutcomeRemoved {
		t.Fatalf("outcomes = %q, %q; want intent then removed", receipts[0].Outcome, receipts[1].Outcome)
	}
	if receipts[0].ID != receipts[1].ID {
		t.Fatalf("intent and outcome carry different ids (%s, %s); the pair cannot be joined", receipts[0].ID, receipts[1].ID)
	}
	if receipts[0].Predicate != "scenario source deleted by operator command" {
		t.Fatalf("predicate not recorded: %q", receipts[0].Predicate)
	}
	if receipts[0].Component != "cliinstall.RemoveScenarioCLI" {
		t.Fatalf("component not recorded: %q", receipts[0].Component)
	}
	if receipts[0].Schema != Schema {
		t.Fatalf("schema = %q, want %q", receipts[0].Schema, Schema)
	}
}

// The ordering guarantee is the one that survives a crash: a process killed
// between the intent and the unlink must still be attributable.
func TestGuardWritesIntentBeforeRemoving(t *testing.T) {
	ledger := testLedger(t)
	var atRemoval int

	err := ledger.Guard(removalFixture(t), func() error {
		receipts, readErr := ledger.Read()
		if readErr != nil {
			return readErr
		}
		atRemoval = len(receipts)
		return nil
	})
	if err != nil {
		t.Fatalf("Guard: %v", err)
	}
	if atRemoval != 1 {
		t.Fatalf("saw %d receipts at removal time, want 1; the intent was not durable before the unlink", atRemoval)
	}
}

// Attribution is a precondition, not a courtesy. An unwritable ledger must stop
// the removal, because an unattributable deletion is the whole fault class.
func TestGuardRefusesToRemoveWhenTheLedgerCannotBeWritten(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	// The receipt directory cannot be created because a file occupies its path.
	ledger := NewAt(filepath.Join(blocker, "receipts")).
		WithIdentity(func() Identity { return Identity{Node: "n", PID: 1, Process: "p"} })

	removed := false
	err := ledger.Guard(removalFixture(t), func() error { removed = true; return nil })

	if err == nil {
		t.Fatal("Guard succeeded with an unwritable ledger")
	}
	if removed {
		t.Fatal("the artifact was removed even though the removal could not be recorded")
	}
}

// A receipt that records a deletion without recording which rule fired restates
// the problem instead of solving it.
func TestGuardRequiresAPredicate(t *testing.T) {
	ledger := testLedger(t)
	removal := removalFixture(t)
	removal.Predicate = "   "

	removed := false
	err := ledger.Guard(removal, func() error { removed = true; return nil })

	if err == nil {
		t.Fatal("Guard accepted a removal with no predicate")
	}
	if removed {
		t.Fatal("removal ran without a recorded rule")
	}
	if !strings.Contains(err.Error(), "predicate") {
		t.Fatalf("error should name the missing predicate, got %q", err)
	}
}

func TestGuardRecordsFailureWithoutClaimingSuccess(t *testing.T) {
	ledger := testLedger(t)
	failure := errors.New("device or resource busy")

	err := ledger.Guard(removalFixture(t), func() error { return failure })
	if !errors.Is(err, failure) {
		t.Fatalf("Guard should surface the removal error, got %v", err)
	}

	receipts, readErr := ledger.Read()
	if readErr != nil {
		t.Fatalf("Read: %v", readErr)
	}
	if got := receipts[len(receipts)-1]; got.Outcome != OutcomeFailed || got.Error == "" {
		t.Fatalf("outcome = %+v, want a failed record carrying the reason", got)
	}
}

// Idempotent removal paths call Guard for artifacts that are already gone. That
// is not a failure, and it must not produce receipts either: a ledger padded
// with records for events that never happened buries the ones that did.
func TestGuardRecordsNothingForAnArtifactThatWasNeverThere(t *testing.T) {
	ledger := testLedger(t)
	removal := removalFixture(t)
	if err := os.Remove(removal.Path); err != nil {
		t.Fatal(err)
	}

	called := false
	err := ledger.Guard(removal, func() error { called = true; return nil })
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Guard on an absent artifact = %v, want fs.ErrNotExist", err)
	}
	if called {
		t.Fatal("the removal ran for an artifact that does not exist")
	}

	receipts, readErr := ledger.Read()
	if readErr != nil {
		t.Fatalf("Read: %v", readErr)
	}
	if len(receipts) != 0 {
		t.Fatalf("got %d receipts for an artifact that was never there: %+v", len(receipts), receipts)
	}
}

// The artifact can still vanish between the lock and the unlink -- another
// agent holding no lock, an external tool. That is a real removal attempt and
// is recorded as absent rather than failed.
func TestGuardRecordsAbsentWhenTheArtifactVanishesMidRemoval(t *testing.T) {
	ledger := testLedger(t)

	err := ledger.Guard(removalFixture(t), func() error { return os.ErrNotExist })
	if err == nil || !os.IsNotExist(err) {
		t.Fatalf("Guard should surface the not-exist error, got %v", err)
	}

	receipts, _ := ledger.Read()
	if got := receipts[len(receipts)-1].Outcome; got != OutcomeAbsent {
		t.Fatalf("outcome = %q, want %q", got, OutcomeAbsent)
	}
}

// agent-manager can be down. A removal during that window is still recorded,
// because an ownership protocol that stops working when a scenario stops is not
// an ownership protocol.
func TestIdentityIsRecordedWithoutAgentManager(t *testing.T) {
	ledger := NewAt(filepath.Join(t.TempDir(), "receipts")).
		WithIdentity(func() Identity {
			return Identity{Node: "swarminator", PID: 99, Process: "nonce"}
		})

	if err := ledger.Guard(removalFixture(t), func() error { return nil }); err != nil {
		t.Fatalf("Guard: %v", err)
	}

	receipts, _ := ledger.Read()
	identity := receipts[0].Identity
	if identity.Node == "" || identity.PID == 0 || identity.Process == "" {
		t.Fatalf("observed identity incomplete: %+v", identity)
	}
	if identity.Attributed() {
		t.Fatal("an identity with no verified provenance must not report itself attributed")
	}
}

// A claim is not attribution. plan-manager already refuses to attribute on an
// environment claim; this keeps the ledger consistent with that decision.
func TestClaimedIdentityIsNeverTreatedAsAttribution(t *testing.T) {
	identity := Identity{
		Node:    "swarminator",
		PID:     7,
		Process: "nonce",
		Claimed: Claimed{Session: "swarm-session-1", Sandbox: "sandbox-9"},
	}
	if identity.Attributed() {
		t.Fatal("an environment claim was treated as verified attribution")
	}

	identity.Verified = &Verified{RunID: "run-x", Actor: "agent"}
	if !identity.Attributed() {
		t.Fatal("verified provenance was not treated as attribution")
	}
}

func TestVerifiedProvenanceIsRecordedWhenSupplied(t *testing.T) {
	ledger := testLedger(t)
	removal := removalFixture(t)
	removal.Verified = &Verified{RunID: "run-x", Actor: "agent", Source: "agent-manager"}

	if err := ledger.Guard(removal, func() error { return nil }); err != nil {
		t.Fatalf("Guard: %v", err)
	}

	receipts, _ := ledger.Read()
	if !receipts[0].Identity.Attributed() {
		t.Fatalf("verified provenance was not carried into the receipt: %+v", receipts[0].Identity)
	}
	if receipts[0].Identity.Verified.RunID != "run-x" {
		t.Fatalf("run id = %q, want run-x", receipts[0].Identity.Verified.RunID)
	}
}

// The agent identity token authenticates its bearer, which makes it a
// credential. It must never reach an evidence artifact, and the surest way to
// guarantee that is to never read it.
func TestLedgerNeverReadsTheIdentityToken(t *testing.T) {
	var requested []string
	original := LookupEnv
	t.Cleanup(func() { LookupEnv = original })
	LookupEnv = func(key string) (string, bool) {
		requested = append(requested, key)
		return "SECRET-VALUE-MUST-NOT-APPEAR", true
	}

	identity := CurrentIdentity()

	for _, key := range requested {
		if strings.Contains(key, "IDENTITY_TOKEN") {
			t.Fatalf("the ledger read %s; a credential must never reach a receipt", key)
		}
		if strings.Contains(key, "AGENT_MANAGER_RUN") {
			t.Fatalf("the ledger read %s, a retired signal plan-manager refuses to attribute on", key)
		}
	}
	if identity.Verified != nil {
		t.Fatal("an environment value produced a verified identity")
	}
}

// A ledger that silently drops what it cannot parse is not evidence.
func TestReadReportsMalformedLinesRatherThanSkipping(t *testing.T) {
	dir := t.TempDir()
	ledger := NewAt(dir)
	if err := os.WriteFile(filepath.Join(dir, "2026-08-25.jsonl"), []byte("{\"schema\":\"x\"}\nnot-json\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := ledger.Read(); err == nil {
		t.Fatal("Read silently accepted a malformed ledger line")
	}
}

func TestReadOnMissingLedgerIsNotAnError(t *testing.T) {
	receipts, err := NewAt(filepath.Join(t.TempDir(), "absent")).Read()
	if err != nil {
		t.Fatalf("Read on a fresh host should be empty, not an error: %v", err)
	}
	if len(receipts) != 0 {
		t.Fatalf("got %d receipts from an absent ledger", len(receipts))
	}
}

func TestReadUsesLexicographicLedgerFileOrder(t *testing.T) {
	dir := t.TempDir()
	for name, id := range map[string]string{"z-last.jsonl": "last", "a-first.jsonl": "first", "m-middle.jsonl": "middle"} {
		data := []byte(fmt.Sprintf("{\"id\":%q}\n", id))
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	receipts, err := NewAt(dir).Read()
	if err != nil {
		t.Fatal(err)
	}
	if got := []string{receipts[0].ID, receipts[1].ID, receipts[2].ID}; !slices.Equal(got, []string{"first", "middle", "last"}) {
		t.Fatalf("receipt order = %v", got)
	}
}

// [REQ:VROOLI-ARTIFACT-SEAM]
// A decision made outside the lock is a hint, not an authorization. Between
// deciding an artifact was reclaimable and deleting it, another agent may have
// reinstalled the scenario that owns it -- the check-then-act window that made
// the reaper unsafe under concurrent agents.
func TestGuardAbandonsWhenRevalidationFailsUnderTheLock(t *testing.T) {
	ledger := testLedger(t)
	removal := removalFixture(t)
	removal.Verify = func() error { return errors.New("owner module reappeared") }

	removed := false
	err := ledger.Guard(removal, func() error { removed = true; return nil })
	if !errors.Is(err, ErrAbandoned) {
		t.Fatalf("Guard = %v, want ErrAbandoned so a caller cannot count it as reclaimed", err)
	}
	if removed {
		t.Fatal("the artifact was removed after its authorization was withdrawn")
	}
	if _, err := os.Stat(removal.Path); err != nil {
		t.Fatalf("the artifact should still exist: %v", err)
	}

	receipts, _ := ledger.Read()
	if len(receipts) != 1 || receipts[0].Outcome != OutcomeAbandoned {
		t.Fatalf("want a single abandoned record, got %+v", receipts)
	}
	if receipts[0].Error != "owner module reappeared" {
		t.Fatalf("abandoned record does not say why: %q", receipts[0].Error)
	}
}

// Re-validation is worthless if it runs before the lock is held.
func TestGuardRevalidatesWhileHoldingTheLock(t *testing.T) {
	ledger := testLedger(t)
	removal := removalFixture(t)

	heldDuringVerify := false
	original := lockArtifact
	t.Cleanup(func() { lockArtifact = original })
	held := false
	lockArtifact = func(subject string) (func(), error) {
		release, err := original(subject)
		if err != nil {
			return nil, err
		}
		held = true
		return func() { held = false; release() }, nil
	}
	removal.Verify = func() error { heldDuringVerify = held; return nil }

	if err := ledger.Guard(removal, func() error { return nil }); err != nil {
		t.Fatalf("Guard: %v", err)
	}
	if !heldDuringVerify {
		t.Fatal("the predicate was re-validated without the lock held")
	}
}

// A CLI is installed as a triple and the installer locks the binary path. A
// sidecar removal must take that same lock, or install and removal would not
// exclude each other while both appeared to be locking.
func TestGuardLocksTheSubjectFamilyNotTheIndividualFile(t *testing.T) {
	ledger := testLedger(t)
	binary := removalFixture(t)
	sidecar := binary
	sidecar.Path = binary.Path + ".build.meta"
	sidecar.Kind = "build-metadata"
	sidecar.Subject = binary.Path
	if err := os.WriteFile(sidecar.Path, []byte("meta"), 0o600); err != nil {
		t.Fatal(err)
	}

	var locked []string
	original := lockArtifact
	t.Cleanup(func() { lockArtifact = original })
	lockArtifact = func(subject string) (func(), error) {
		locked = append(locked, subject)
		return original(subject)
	}

	if err := ledger.Guard(sidecar, func() error { return os.Remove(sidecar.Path) }); err != nil {
		t.Fatalf("Guard: %v", err)
	}
	if len(locked) != 1 || locked[0] != binary.Path {
		t.Fatalf("locked %v, want the binary path %q that the installer locks", locked, binary.Path)
	}
}

// The seam and the installer must agree on the lock file, or both lock and
// neither excludes the other.
func TestSeamLocksTheSamePathTheInstallerLocks(t *testing.T) {
	dir := t.TempDir()
	artifact := filepath.Join(dir, "some-cli")
	if err := os.WriteFile(artifact, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := lockArtifact(artifact); err != nil {
		t.Fatalf("lockArtifact: %v", err)
	}
	// buildinfo.AcquireBinaryInstallLock locks <executable>.lock. Asserting the
	// filename here keeps the two in agreement without importing buildinfo,
	// which would be an import cycle.
	if _, err := os.Stat(artifact + ".lock"); err != nil {
		t.Fatalf("the seam did not lock <artifact>.lock, which is what the installer locks: %v", err)
	}
}
