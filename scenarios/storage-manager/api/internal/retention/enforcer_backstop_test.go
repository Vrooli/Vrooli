package retention

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	coreStorage "github.com/vrooli/api-core/storage"
)

type fakeOwnerReclaimer struct {
	calls   []string
	receipt json.RawMessage
	err     error
	onCall  func()
}

func (f *fakeOwnerReclaimer) Reclaim(_ context.Context, owner, operation string) (json.RawMessage, error) {
	f.calls = append(f.calls, owner+" "+operation)
	if f.onCall != nil {
		f.onCall()
	}
	return f.receipt, f.err
}

type budgetEvent struct {
	kind    string
	payload map[string]any
}

// keeperFixture declares one non-regenerable class-data entry for a scenario
// whose live database sits in the lifecycle data directory, the layout every
// lifecycle-launched SQLitePath scenario uses.
func keeperFixture(t *testing.T, budget string, reclaim *coreStorage.ReclaimDeclaration) (string, coreStorage.OwnerInventory, string, string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SCENARIO_NAME", "")
	t.Setenv("VROOLI_SCENARIO", "")
	t.Setenv("SCENARIO_DATA_DIR", "")
	root := contractFixture(t)
	scenarioDir := filepath.Join(root, "scenarios", "keeper")
	live := filepath.Join(coreStorage.LifecycleDataDir(scenarioDir), "keeper.db")
	if err := os.MkdirAll(filepath.Dir(live), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(live, make([]byte, 16), 0o644); err != nil {
		t.Fatal(err)
	}
	owner := coreStorage.OwnerManifest{
		Kind: coreStorage.OwnerScenario, ID: "keeper", ManifestPath: filepath.Join(scenarioDir, ".vrooli", "service.json"),
		StorageEntries: []coreStorage.StorageEntry{{
			Name: "data", Kind: "dir", Class: coreStorage.ClassData, Regenerable: false,
			Budget: &coreStorage.BudgetDeclaration{MaxBytes: budget}, Reclaim: reclaim,
		}},
	}
	classRoot, err := coreStorage.ResolveOwnerStoragePath(root, owner, owner.StorageEntries[0], coreStorage.HostPlatform(), coreStorage.PlatformSeams{})
	if err != nil {
		t.Fatal(err)
	}
	return root, coreStorage.OwnerInventory{RepoRoot: root, Owners: []coreStorage.OwnerManifest{owner}}, live, classRoot
}

func entryResult(t *testing.T, results map[string]Result, owner, entry string) Result {
	t.Helper()
	for _, result := range results[owner].EntryResults {
		if result.Entry == entry {
			return result
		}
	}
	t.Fatalf("no result for %s/%s in %+v", owner, entry, results)
	return Result{}
}

// TestEnforceMeasuresTheLiveLifecycleDatabase is the regression for budgets
// that measured only the class root while the live database grew elsewhere.
func TestEnforceMeasuresTheLiveLifecycleDatabase(t *testing.T) {
	root, inventory, live, classRoot := keeperFixture(t, "8B", nil)
	if err := os.MkdirAll(classRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(classRoot, "old.db"), make([]byte, 4), 0o644); err != nil {
		t.Fatal(err)
	}

	results, err := (Enforcer{RepoRoot: root, Platform: coreStorage.HostPlatform()}).Enforce(context.Background(), inventory)
	if err != nil {
		t.Fatal(err)
	}

	result := entryResult(t, results, "keeper", "data")
	if result.UsedBytes != 20 || result.OverBytes != 12 || len(result.Locations) != 2 {
		t.Fatalf("result = %+v, want 20 bytes over two locations", result)
	}
	if _, err := os.Stat(live); err != nil {
		t.Fatalf("the live database must never be touched: %v", err)
	}
}

func TestMeasureEntryExcludesANestedSiblingEntry(t *testing.T) {
	root, inventory, _, classRoot := keeperFixture(t, "1GiB", nil)
	owner := inventory.Owners[0]
	owner.StorageEntries = append(owner.StorageEntries, coreStorage.StorageEntry{
		Name: "legacy_history", Kind: "dir", Class: coreStorage.ClassData, Subpath: "legacy-history",
		Budget: &coreStorage.BudgetDeclaration{MaxBytes: "1GiB"},
	})
	legacy := filepath.Join(classRoot, "legacy-history")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "archived.db"), make([]byte, 100), 0o644); err != nil {
		t.Fatal(err)
	}

	data, err := MeasureEntry(root, owner, owner.StorageEntries[0], coreStorage.HostPlatform())
	if err != nil {
		t.Fatal(err)
	}
	archived, err := MeasureEntry(root, owner, owner.StorageEntries[1], coreStorage.HostPlatform())
	if err != nil {
		t.Fatal(err)
	}
	if data.Bytes != 16 {
		t.Fatalf("data counted %d bytes, want only the live 16; the archive belongs to its own entry", data.Bytes)
	}
	if archived.Bytes != 100 {
		t.Fatalf("legacy_history = %d bytes, want 100", archived.Bytes)
	}
}

func TestOwnerReclaimIsRequestedOnlyAfterASustainedBreachAndEscalatesOnRepeat(t *testing.T) {
	root, inventory, _, _ := keeperFixture(t, "8B", &coreStorage.ReclaimDeclaration{Operation: "/api/v1/storage/reclaim"})
	now := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)
	owner := &fakeOwnerReclaimer{receipt: json.RawMessage(`{"reclaimed_bytes":0}`)}
	var events []budgetEvent
	enforcer := Enforcer{
		RepoRoot: root, Platform: coreStorage.HostPlatform(), OverBudgetCycles: map[string]int{},
		OwnerReclaim: owner, OwnerReclaimAttempts: map[string]time.Time{}, Now: func() time.Time { return now },
		BudgetEvent: func(_ context.Context, kind string, payload map[string]any) error {
			events = append(events, budgetEvent{kind, payload})
			return nil
		},
	}
	cycle := func() Result {
		results, err := enforcer.Enforce(context.Background(), inventory)
		if err != nil {
			t.Fatal(err)
		}
		return entryResult(t, results, "keeper", "data")
	}

	if first := cycle(); first.OwnerReclaim != nil || len(owner.calls) != 0 {
		t.Fatalf("one over-budget cycle must not ask the owner: %+v calls=%v", first, owner.calls)
	}
	second := cycle()
	if len(owner.calls) != 1 || owner.calls[0] != "keeper /api/v1/storage/reclaim" {
		t.Fatalf("calls = %v, want one request to the declared operation", owner.calls)
	}
	if second.OwnerReclaim == nil || string(second.OwnerReclaim.Receipt) != `{"reclaimed_bytes":0}` || !second.OwnerReclaim.StillOver {
		t.Fatalf("outcome = %+v, want the owner's receipt and a still-over measurement", second.OwnerReclaim)
	}
	if second.Escalated {
		t.Fatal("a first request is not judged by an immediate re-measure; the owner may reclaim asynchronously")
	}
	if third := cycle(); len(owner.calls) != 1 || third.OwnerReclaim != nil {
		t.Fatalf("a request inside the interval must wait: calls=%v", owner.calls)
	}

	now = now.Add(DefaultOwnerReclaimInterval + time.Minute)
	repeat := cycle()
	if len(owner.calls) != 2 || !repeat.Escalated {
		t.Fatalf("a breach standing after a repeated request must escalate: calls=%v result=%+v", owner.calls, repeat)
	}
	if last := events[len(events)-1]; last.kind != "storage.budget.owner_reclaim_insufficient" || last.payload["owner"] != "keeper" {
		t.Fatalf("events = %+v, want an owner_reclaim_insufficient escalation", events)
	}
}

func TestOwnerReclaimThatFreesSpaceEndsTheBreach(t *testing.T) {
	root, inventory, live, _ := keeperFixture(t, "8B", &coreStorage.ReclaimDeclaration{Operation: "/api/v1/storage/reclaim"})
	owner := &fakeOwnerReclaimer{onCall: func() {
		if err := os.WriteFile(live, make([]byte, 4), 0o644); err != nil {
			t.Fatal(err)
		}
	}}
	attempts := map[string]time.Time{}
	enforcer := Enforcer{RepoRoot: root, Platform: coreStorage.HostPlatform(), OverBudgetCycles: map[string]int{}, OwnerReclaim: owner, OwnerReclaimAttempts: attempts}
	for range 2 {
		if _, err := enforcer.Enforce(context.Background(), inventory); err != nil {
			t.Fatal(err)
		}
	}
	results, err := enforcer.Enforce(context.Background(), inventory)
	if err != nil {
		t.Fatal(err)
	}
	result := entryResult(t, results, "keeper", "data")
	if len(owner.calls) != 1 || result.OverBytes != 0 || result.Escalated {
		t.Fatalf("calls=%v result=%+v, want one request and the breach cleared", owner.calls, result)
	}
	if _, pending := attempts["keeper/data"]; pending {
		t.Fatal("a cleared breach must reset its reclaim memory so the next breach starts fresh")
	}
}

func TestOwnerReclaimFailureEscalatesImmediately(t *testing.T) {
	root, inventory, _, _ := keeperFixture(t, "8B", &coreStorage.ReclaimDeclaration{Operation: "/api/v1/storage/reclaim"})
	owner := &fakeOwnerReclaimer{err: errors.New("owner scenario unreachable")}
	enforcer := Enforcer{RepoRoot: root, Platform: coreStorage.HostPlatform(), OverBudgetCycles: map[string]int{}, OwnerReclaim: owner, OwnerReclaimAttempts: map[string]time.Time{}}
	var result Result
	for range 2 {
		results, err := enforcer.Enforce(context.Background(), inventory)
		if err != nil {
			t.Fatal(err)
		}
		result = entryResult(t, results, "keeper", "data")
	}
	if !result.Escalated || result.OwnerReclaim == nil || result.OwnerReclaim.Error == "" {
		t.Fatalf("result = %+v, want an escalated failed request", result)
	}
}

func TestEntryWithoutAReclaimOperationNeverCallsTheOwner(t *testing.T) {
	root, inventory, _, _ := keeperFixture(t, "8B", nil)
	owner := &fakeOwnerReclaimer{}
	enforcer := Enforcer{RepoRoot: root, Platform: coreStorage.HostPlatform(), OverBudgetCycles: map[string]int{}, OwnerReclaim: owner, OwnerReclaimAttempts: map[string]time.Time{}}
	for range 3 {
		if _, err := enforcer.Enforce(context.Background(), inventory); err != nil {
			t.Fatal(err)
		}
	}
	if len(owner.calls) != 0 {
		t.Fatalf("calls = %v, want none without a declared operation", owner.calls)
	}
}

func TestRecordCycleSurfacesOverBudgetAndEscalatedEntries(t *testing.T) {
	RecordCycle(time.Now(), map[string]Result{
		"keeper": {Owner: "keeper", EntryResults: []Result{
			{Owner: "keeper", Entry: "data", UsedBytes: 20, OverBytes: 12, Escalated: true},
			{Owner: "keeper", Entry: "logs", UsedBytes: 1},
		}},
	}, nil)
	status, ok := LatestCycle()
	if !ok || len(status.Entries) != 2 || len(status.OverBudget) != 1 || len(status.Escalated) != 1 || status.Escalated[0].Entry != "data" {
		t.Fatalf("status = %+v", status)
	}
}
