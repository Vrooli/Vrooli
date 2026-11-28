package gocache

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeAudit struct {
	report AuditReport
	err    error
}

func (f fakeAudit) run() (AuditReport, error) { return f.report, f.err }

// useAudit replaces the host audit; the real auditor is exercised by its own
// tests and by the end-to-end build test.
func useAudit(t *testing.T, report AuditReport, err error) {
	t.Helper()
	previous := newStoreAuditor
	newStoreAuditor = func(Store, string, func() time.Time) (storeAuditor, error) {
		return fakeAudit{report: report, err: err}, nil
	}
	t.Cleanup(func() { newStoreAuditor = previous })
}

func clearAudit(t *testing.T) { useAudit(t, AuditReport{}, nil) }

// activate points this test process's go env file at a stand-in helper.
func activate(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	program := filepath.Join(dir, ProgramName)
	if err := os.WriteFile(program, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	envFile := filepath.Join(dir, "go-env")
	if err := os.WriteFile(envFile, []byte("GOFLAGS=-mod=mod\nGOCACHEPROG="+program+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOENV", envFile)
	return program
}

type fixtureEntry struct {
	action, output ID
	bytes          int64
}

func addEntry(t *testing.T, store Store, name string, size int, lastUse time.Time) fixtureEntry {
	t.Helper()
	body := bytes.Repeat([]byte(name+"."), size/(len(name)+1)+1)[:size]
	output := ID(sha256.Sum256(body))
	action := actionID(name)
	outputPath, actionPath := store.entryPath(output, kindOutput), store.entryPath(action, kindAction)
	if err := os.WriteFile(outputPath, body, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(actionPath, encodeEntry(action, Entry{Output: output, Size: int64(size), Time: lastUse}), 0o644); err != nil {
		t.Fatal(err)
	}
	var total int64
	for _, path := range []string{outputPath, actionPath} {
		if err := os.Chtimes(path, lastUse, lastUse); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		total += allocatedBytes(info)
	}
	return fixtureEntry{action: action, output: output, bytes: total}
}

func (e fixtureEntry) present(store Store) bool {
	_, errOutput := os.Stat(store.entryPath(e.output, kindOutput))
	_, errAction := os.Stat(store.entryPath(e.action, kindAction))
	return errOutput == nil && errAction == nil
}

func (e fixtureEntry) gone(store Store) bool {
	_, errOutput := os.Stat(store.entryPath(e.output, kindOutput))
	_, errAction := os.Stat(store.entryPath(e.action, kindAction))
	return errors.Is(errOutput, os.ErrNotExist) && errors.Is(errAction, os.ErrNotExist)
}

func testPolicy(budget int64) Policy { return Policy{BudgetBytes: budget, MinIdle: DefaultMinIdle} }

func mustPlan(t *testing.T, store Store, program string, policy Policy) Plan {
	t.Helper()
	plan, err := MakePlan(PlanOptions{Store: store, Program: program, Policy: policy, Fresh: true})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func mustApply(t *testing.T, store Store, program string, policy Policy, token string, maxBytes int64) Receipt {
	t.Helper()
	receipt, err := Apply(ApplyOptions{Store: store, Program: program, Policy: policy, Token: token, MaxBytes: maxBytes})
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func TestPlanSelectsLeastRecentlyUsedEntriesDownToBudget(t *testing.T) {
	clearAudit(t)
	program := activate(t)
	store := Store{Root: newStoreDir(t)}
	now := time.Now()
	oldest := addEntry(t, store, "oldest", 64<<10, now.Add(-30*time.Hour))
	older := addEntry(t, store, "older", 64<<10, now.Add(-20*time.Hour))
	recent := addEntry(t, store, "recent", 64<<10, now.Add(-10*time.Hour))
	newest := addEntry(t, store, "newest", 64<<10, now.Add(-1*time.Hour))

	plan := mustPlan(t, store, program, testPolicy(recent.bytes+newest.bytes))
	if plan.BlockedReason != "" || plan.Token == "" || plan.Cutoff == nil {
		t.Fatalf("plan = %+v", plan)
	}
	if plan.StoreBytes != oldest.bytes+older.bytes+recent.bytes+newest.bytes || plan.StoreEntries != 8 {
		t.Fatalf("plan measured %d bytes in %d entries", plan.StoreBytes, plan.StoreEntries)
	}
	if plan.ReclaimableBytes != oldest.bytes+older.bytes || plan.ReclaimableEntries != 4 {
		t.Fatalf("plan selects %d bytes in %d entries, want the two oldest (%d bytes)", plan.ReclaimableBytes, plan.ReclaimableEntries, oldest.bytes+older.bytes)
	}
	receipt := mustApply(t, store, program, testPolicy(recent.bytes+newest.bytes), plan.Token, 0)
	if receipt.StopReason != StopComplete || !receipt.PlanExhausted || receipt.FreedBytes != plan.ReclaimableBytes || receipt.RemovedEntries != 4 {
		t.Fatalf("receipt = %+v", receipt)
	}
	if !oldest.gone(store) || !older.gone(store) || !recent.present(store) || !newest.present(store) {
		t.Fatal("apply removed a different set than the plan selected")
	}
}

func TestPlanNeverReachesIntoTheMinimumIdleWindow(t *testing.T) {
	clearAudit(t)
	program := activate(t)
	store := Store{Root: newStoreDir(t)}
	now := time.Now()
	stale := addEntry(t, store, "stale", 4096, now.Add(-30*time.Hour))
	fresh := addEntry(t, store, "fresh", 4096, now.Add(-90*time.Minute))
	plan := mustPlan(t, store, program, testPolicy(0))
	if plan.ReclaimableBytes != stale.bytes {
		t.Fatalf("plan selects %d bytes; only the entry outside the idle window (%d) is eligible", plan.ReclaimableBytes, stale.bytes)
	}
	mustApply(t, store, program, testPolicy(0), plan.Token, 0)
	if !stale.gone(store) || !fresh.present(store) {
		t.Fatal("an entry used inside the minimum idle window was removed")
	}
}

func TestPlanWithinBudgetProposesNothing(t *testing.T) {
	clearAudit(t)
	program := activate(t)
	store := Store{Root: newStoreDir(t)}
	addEntry(t, store, "only", 4096, time.Now().Add(-48*time.Hour))
	plan := mustPlan(t, store, program, testPolicy(1<<30))
	if plan.Token != "" || plan.ReclaimableBytes != 0 || plan.BlockedReason != "" {
		t.Fatalf("within-budget plan = %+v", plan)
	}
}

// The core guarantee: an output a live session handed out survives reclaim
// even when its mtime makes it look idle, and becomes eligible once the
// session ends.
func TestApplyNeverRemovesOutputsALiveSessionHolds(t *testing.T) {
	clearAudit(t)
	program := activate(t)
	store := Store{Root: newStoreDir(t)}
	now := time.Now()
	held := addEntry(t, store, "held", 8192, now.Add(-40*time.Hour))
	free := addEntry(t, store, "free", 8192, now.Add(-40*time.Hour))

	// The session's clock lies in the past so serving the hit does not make
	// the entry look recently used; only the pin protects it.
	session := newSession(store, func() time.Time { return now.Add(-40 * time.Hour) }, io.Discard)
	if session.degraded != "" {
		t.Fatalf("session degraded: %s", session.degraded)
	}
	hit, err := session.get(held.action)
	if err != nil {
		t.Fatalf("get held entry: %v", err)
	}

	plan := mustPlan(t, store, program, testPolicy(0))
	receipt := mustApply(t, store, program, testPolicy(0), plan.Token, 0)
	if !held.present(store) {
		t.Fatal("reclaim removed an output a live session holds")
	}
	if _, err := os.Stat(hit.diskPath); err != nil {
		t.Fatalf("held output path: %v", err)
	}
	if !free.gone(store) || receipt.SkippedPinned != 2 || receipt.LiveSessions != 1 {
		t.Fatalf("receipt = %+v", receipt)
	}

	if err := session.close(); err != nil {
		t.Fatal(err)
	}
	plan = mustPlan(t, store, program, testPolicy(0))
	mustApply(t, store, program, testPolicy(0), plan.Token, 0)
	if !held.gone(store) {
		t.Fatal("entry stayed protected after its session closed")
	}
}

func TestApplyTreatsAJournalWhoseSessionDiedAsVoid(t *testing.T) {
	clearAudit(t)
	program := activate(t)
	store := Store{Root: newStoreDir(t)}
	entry := addEntry(t, store, "orphan", 4096, time.Now().Add(-40*time.Hour))
	state, err := openState(store)
	if err != nil {
		t.Fatal(err)
	}
	defer state.Close()
	journal := filepath.Join(state.sessionsDir(), "4242-dead"+pinJournalSuffix)
	if err := os.WriteFile(journal, []byte(pinJournalHeader+entry.action.String()+"\n"+entry.output.String()+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan := mustPlan(t, store, program, testPolicy(0))
	mustApply(t, store, program, testPolicy(0), plan.Token, 0)
	if !entry.gone(store) {
		t.Fatal("pins of a session that no longer holds its journal lock protected an entry")
	}
	if _, err := os.Stat(journal); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("dead session journal remains: %v", err)
	}
}

func TestApplyRefusesWhileALiveJournalIsUnreadable(t *testing.T) {
	clearAudit(t)
	program := activate(t)
	store := Store{Root: newStoreDir(t)}
	entry := addEntry(t, store, "guarded", 4096, time.Now().Add(-40*time.Hour))
	state, err := openState(store)
	if err != nil {
		t.Fatal(err)
	}
	defer state.Close()
	path := filepath.Join(state.sessionsDir(), "1-future"+pinJournalSuffix)
	if err := os.WriteFile(path, []byte("vrooli-gocache-pins v9\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	holder, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer holder.Close()
	if ok, err := tryLock(holder, true); err != nil || !ok {
		t.Fatalf("hold journal: %v %v", ok, err)
	}
	plan := mustPlan(t, store, program, testPolicy(0))
	receipt := mustApply(t, store, program, testPolicy(0), plan.Token, 0)
	if receipt.StopReason != StopBlocked || !strings.Contains(receipt.BlockedReason, "unknown format") || !entry.present(store) {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestApplySkipsEntriesUsedAfterThePlan(t *testing.T) {
	clearAudit(t)
	program := activate(t)
	store := Store{Root: newStoreDir(t)}
	now := time.Now()
	reused := addEntry(t, store, "reused", 4096, now.Add(-40*time.Hour))
	idle := addEntry(t, store, "idle", 4096, now.Add(-40*time.Hour))
	plan := mustPlan(t, store, program, testPolicy(0))
	for _, path := range []string{store.entryPath(reused.action, kindAction), store.entryPath(reused.output, kindOutput)} {
		if err := os.Chtimes(path, now, now); err != nil {
			t.Fatal(err)
		}
	}
	receipt := mustApply(t, store, program, testPolicy(0), plan.Token, 0)
	if !reused.present(store) || !idle.gone(store) || receipt.RemovedEntries != 2 {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestReclaimRefusesWhileAnUnmediatedGoCommandRuns(t *testing.T) {
	useAudit(t, AuditReport{GoCommands: 1, Unmediated: []ProcessFinding{{PID: 77, Reason: "started before its go env file last changed", Command: "go test ./..."}}}, nil)
	program := activate(t)
	store := Store{Root: newStoreDir(t)}
	entry := addEntry(t, store, "busy", 4096, time.Now().Add(-40*time.Hour))
	plan := mustPlan(t, store, program, testPolicy(0))
	if plan.Token != "" || !strings.Contains(plan.BlockedReason, "pid 77") {
		t.Fatalf("plan = %+v", plan)
	}
	token := encodeToken(plan.StoreID, time.Now().Add(-24*time.Hour), time.Now())
	receipt := mustApply(t, store, program, testPolicy(0), token, 0)
	if receipt.StopReason != StopBlocked || !strings.Contains(receipt.BlockedReason, "pid 77") || !entry.present(store) || receipt.FreedBytes != 0 {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestReclaimRefusesAFailedAudit(t *testing.T) {
	useAudit(t, AuditReport{}, errors.New("proc unreadable"))
	program := activate(t)
	store := Store{Root: newStoreDir(t)}
	entry := addEntry(t, store, "unknown", 4096, time.Now().Add(-40*time.Hour))
	plan := mustPlan(t, store, program, testPolicy(0))
	if plan.Token != "" || !strings.Contains(plan.BlockedReason, "audit failed") {
		t.Fatalf("plan = %+v", plan)
	}
	token := encodeToken(plan.StoreID, time.Now().Add(-24*time.Hour), time.Now())
	if receipt := mustApply(t, store, program, testPolicy(0), token, 0); receipt.StopReason != StopBlocked || !entry.present(store) {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestReclaimRefusesAnInactiveProtocol(t *testing.T) {
	clearAudit(t)
	program := activate(t)
	envFile := filepath.Join(t.TempDir(), "go-env")
	if err := os.WriteFile(envFile, []byte("GOFLAGS=-mod=mod\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOENV", envFile)
	t.Setenv("GOCACHEPROG", program) // this process's own variable does not activate other go commands
	store := Store{Root: newStoreDir(t)}
	entry := addEntry(t, store, "inactive", 4096, time.Now().Add(-40*time.Hour))
	plan := mustPlan(t, store, program, testPolicy(0))
	if plan.Activation.Active || !strings.Contains(plan.BlockedReason, "inactive") {
		t.Fatalf("plan = %+v", plan)
	}
	token := encodeToken(plan.StoreID, time.Now().Add(-24*time.Hour), time.Now())
	if receipt := mustApply(t, store, program, testPolicy(0), token, 0); receipt.StopReason != StopBlocked || !entry.present(store) {
		t.Fatalf("receipt = %+v", receipt)
	}
}

func TestApplyBindsItsTokenToTheExactStore(t *testing.T) {
	clearAudit(t)
	program := activate(t)
	planned := Store{Root: newStoreDir(t)}
	other := Store{Root: newStoreDir(t)}
	addEntry(t, planned, "a", 4096, time.Now().Add(-40*time.Hour))
	victim := addEntry(t, other, "b", 4096, time.Now().Add(-40*time.Hour))
	plan := mustPlan(t, planned, program, testPolicy(0))
	if _, err := Apply(ApplyOptions{Store: other, Program: program, Policy: testPolicy(0), Token: plan.Token}); err == nil || !strings.Contains(err.Error(), "bound to store") {
		t.Fatalf("cross-store apply error = %v", err)
	}
	if !victim.present(other) {
		t.Fatal("a token for one store removed entries from another")
	}
	forged := encodeToken(plan.StoreID, time.Now().Add(-time.Minute), time.Now())
	if _, err := Apply(ApplyOptions{Store: planned, Program: program, Policy: testPolicy(0), Token: forged}); err == nil || !strings.Contains(err.Error(), "minimum idle") {
		t.Fatalf("token inside the idle window error = %v", err)
	}
}

func TestBoundedAppliesResumeOnePlanUntilItIsExhausted(t *testing.T) {
	clearAudit(t)
	program := activate(t)
	store := Store{Root: newStoreDir(t)}
	var want int64
	for i := range 40 {
		want += addEntry(t, store, "bulk"+string(rune('a'+i%26))+string(rune('a'+i/26)), 16<<10, time.Now().Add(-48*time.Hour)).bytes
	}
	policy := testPolicy(0)
	plan := mustPlan(t, store, program, policy)
	if plan.ReclaimableBytes != want {
		t.Fatalf("plan selects %d, want %d", plan.ReclaimableBytes, want)
	}
	var freed int64
	for batch := 0; ; batch++ {
		if batch > 40 {
			t.Fatal("bounded applies did not converge")
		}
		// A recovery loop re-plans before every batch; the plan continues.
		next := mustPlanContinuing(t, store, program, policy)
		if next.Token != plan.Token {
			t.Fatalf("batch %d re-scanned instead of continuing the plan", batch)
		}
		receipt := mustApply(t, store, program, policy, plan.Token, 64<<10)
		freed += receipt.FreedBytes
		if receipt.PlanExhausted {
			break
		}
		if receipt.StopReason != StopMaxBytes || receipt.FreedBytes < 64<<10 {
			t.Fatalf("batch %d receipt = %+v", batch, receipt)
		}
	}
	if freed != want {
		t.Fatalf("bounded applies freed %d, want %d", freed, want)
	}
	if fresh := mustPlanContinuing(t, store, program, policy); fresh.Token != "" || fresh.StoreBytes != 0 {
		t.Fatalf("plan after exhaustion = %+v", fresh)
	}
}

func mustPlanContinuing(t *testing.T, store Store, program string, policy Policy) Plan {
	t.Helper()
	plan, err := MakePlan(PlanOptions{Store: store, Program: program, Policy: policy})
	if err != nil {
		t.Fatal(err)
	}
	return plan
}

func TestApplyRemovesOnlyAbandonedStagedWrites(t *testing.T) {
	clearAudit(t)
	program := activate(t)
	store := Store{Root: newStoreDir(t)}
	trigger := addEntry(t, store, "trigger", 4096, time.Now().Add(-48*time.Hour))
	shard := store.shardDir(trigger.output)
	abandoned := filepath.Join(shard, trigger.output.String()+"-d"+tempMarker+"old")
	inFlight := filepath.Join(shard, trigger.output.String()+"-d"+tempMarker+"new")
	for path, age := range map[string]time.Duration{abandoned: 48 * time.Hour, inFlight: time.Hour} {
		if err := os.WriteFile(path, []byte("partial"), 0o600); err != nil {
			t.Fatal(err)
		}
		at := time.Now().Add(-age)
		if err := os.Chtimes(path, at, at); err != nil {
			t.Fatal(err)
		}
	}
	plan := mustPlan(t, store, program, testPolicy(0))
	mustApply(t, store, program, testPolicy(0), plan.Token, 0)
	if _, err := os.Stat(abandoned); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("abandoned staged write survived")
	}
	if _, err := os.Stat(inFlight); err != nil {
		t.Fatal("a staged write that may still be in flight was removed")
	}
}

func TestDegradedSessionNeverSharesUnpinnedFiles(t *testing.T) {
	cache := newStoreDir(t)
	if err := os.WriteFile(filepath.Join(cache, StateDirName), []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	session := newSession(Store{Root: cache}, time.Now, &stderr)
	if session.degraded == "" || !strings.Contains(stderr.String(), "coordination unavailable") {
		t.Fatalf("session did not report degradation: %q", stderr.String())
	}
	body := []byte("private output")
	output := ID(sha256.Sum256(body))
	path, err := session.put(actionID("p"), output, int64(len(body)), bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(path, cache) {
		t.Fatalf("degraded session published %s into the shared store", path)
	}
	if _, err := session.get(actionID("p")); !errors.Is(err, errMiss) {
		t.Fatalf("degraded session served a hit: %v", err)
	}
	if err := session.close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("private outputs outlived their session")
	}
}

func TestEntryFormatRoundTripsAndRejectsWhatCmdGoRejects(t *testing.T) {
	action := actionID("rt")
	entry := Entry{Output: ID(sha256.Sum256([]byte("o"))), Size: 12345, Time: time.Unix(0, 1_700_000_000_123_456_789)}
	data := encodeEntry(action, entry)
	got, err := decodeEntry(action, data)
	if err != nil || got != entry {
		t.Fatalf("round trip = %+v, %v", got, err)
	}
	for name, corrupt := range map[string][]byte{
		"short":           data[:entrySize-1],
		"wrong action":    encodeEntry(actionID("other"), entry),
		"bad version":     append([]byte("v2"), data[2:]...),
		"missing newline": append(append([]byte{}, data[:entrySize-1]...), ' '),
	} {
		if _, err := decodeEntry(action, corrupt); err == nil {
			t.Errorf("%s entry accepted", name)
		}
	}
}
