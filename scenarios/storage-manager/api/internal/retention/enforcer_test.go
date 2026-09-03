package retention

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	coreStorage "github.com/vrooli/api-core/storage"
	"github.com/vrooli/vrooli/packages/artifactledger"
)

func TestEnforceDirectoryBudgetUsesBuiltinProvider(t *testing.T) {
	root := contractFixture(t)
	resourceDir := filepath.Join(root, "resources", "demo")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	budgeted := filepath.Join(resourceDir, "cache")
	if err := os.MkdirAll(budgeted, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"old", "new"} {
		if err := os.WriteFile(filepath.Join(budgeted, name), []byte("1234"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	inventory := coreStorage.OwnerInventory{
		RepoRoot: root,
		Owners: []coreStorage.OwnerManifest{{
			Kind:         coreStorage.OwnerResource,
			ID:           "demo",
			ManifestPath: filepath.Join(resourceDir, "resource.json"),
			StorageEntries: []coreStorage.StorageEntry{{
				Name: "cache", Path: coreStorage.PortablePath{Value: "cache"}, Kind: "dir", Regenerable: true,
				Budget: &coreStorage.BudgetDeclaration{MaxBytes: "4B"},
			}},
		}},
	}

	results, err := (Enforcer{RepoRoot: root, Platform: coreStorage.PlatformLinux}).Enforce(context.Background(), inventory)
	if err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	result, ok := results["demo"]
	if !ok || result.Error != "" {
		t.Fatalf("result = %+v, want successful owner result", result)
	}
	if result.Deleted != 1 || result.Freed != 4 {
		t.Fatalf("result = %+v, want one 4-byte deletion", result)
	}
	entries, err := os.ReadDir(budgeted)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("remaining entries = %d, want 1", len(entries))
	}
}

func TestEnforceEmitsBudgetExceededOnSecondSustainedCycle(t *testing.T) {
	root := contractFixture(t)
	resourceDir := filepath.Join(root, "resources", "demo")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	budgeted := filepath.Join(resourceDir, "durable")
	if err := os.MkdirAll(budgeted, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(budgeted, "data"), []byte("1234567890"), 0o644); err != nil {
		t.Fatal(err)
	}
	inventory := coreStorage.OwnerInventory{RepoRoot: root, Owners: []coreStorage.OwnerManifest{{
		Kind: coreStorage.OwnerResource, ID: "demo", ManifestPath: filepath.Join(resourceDir, "resource.json"),
		StorageEntries: []coreStorage.StorageEntry{{Name: "durable", Path: coreStorage.PortablePath{Value: "durable"}, Kind: "dir", Regenerable: false, Budget: &coreStorage.BudgetDeclaration{MaxBytes: "4B"}}},
	}}}
	cycles := map[string]int{}
	var events []map[string]any
	e := Enforcer{RepoRoot: root, Platform: coreStorage.PlatformLinux, OverBudgetCycles: cycles, BudgetEvent: func(_ context.Context, _ string, payload map[string]any) error {
		events = append(events, payload)
		return nil
	}}
	if _, err := e.Enforce(context.Background(), inventory); err != nil {
		t.Fatal(err)
	}
	if len(events) != 0 {
		t.Fatalf("first over-budget cycle emitted %v, want none", events)
	}
	if _, err := e.Enforce(context.Background(), inventory); err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0]["owner"] != "demo" || events[0]["entry"] != "durable" {
		t.Fatalf("budget events = %#v, want one typed second-cycle event", events)
	}
}

func TestEnforceSkipsUnsupportedPlatformEntries(t *testing.T) {
	owner := coreStorage.OwnerManifest{
		Kind: coreStorage.OwnerResource, ID: "demo", ManifestPath: "/repo/resources/demo/resource.json",
		StorageEntries: []coreStorage.StorageEntry{{Name: "models", Kind: "dir", Path: coreStorage.PortablePath{ByOS: map[coreStorage.Platform]*string{coreStorage.PlatformLinux: nil, coreStorage.PlatformMacOS: nil, coreStorage.PlatformWindows: nil}}, Budget: &coreStorage.BudgetDeclaration{MaxBytes: "1MiB"}}},
	}
	repoRoot := contractFixture(t)
	results, err := (Enforcer{RepoRoot: repoRoot, Platform: coreStorage.PlatformLinux}).Enforce(context.Background(), coreStorage.OwnerInventory{Owners: []coreStorage.OwnerManifest{owner}})
	if err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("results = %+v, want no applicable result", results)
	}
}

// The install root is protected because the repository contract says so, not
// because this package names it. A budget declared over ~/.vrooli/bin -- the
// exact shape the coding_agent_shims safeguard used to ship -- must be refused
// without deleting anything.
func TestEnforceRefusesBudgetOverContractProtectedRuntimeHomeEntry(t *testing.T) {
	repoRoot := contractFixture(t)
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory on this host: %v", err)
	}
	binDir := filepath.Join(home, ".vrooli", "bin")

	owner := coreStorage.OwnerManifest{
		Kind: coreStorage.OwnerSafeguard, ID: "coding_agent_shims",
		ManifestPath: filepath.Join(repoRoot, "internal", "safeguards", "coding-agent-shims", "safeguard.json"),
		StorageEntries: []coreStorage.StorageEntry{{
			Name: "shims", Kind: "dir", Regenerable: true,
			Path:   coreStorage.PortablePath{Value: binDir},
			Budget: &coreStorage.BudgetDeclaration{MaxBytes: "64MiB"},
		}},
	}

	results, err := (Enforcer{RepoRoot: repoRoot, Platform: coreStorage.Platform(runtime.GOOS)}).
		Enforce(context.Background(), coreStorage.OwnerInventory{RepoRoot: repoRoot, Owners: []coreStorage.OwnerManifest{owner}})
	if err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	result, ok := results["coding_agent_shims"]
	if !ok {
		t.Fatalf("results = %+v, want a governed result for the safeguard", results)
	}
	if !result.Refused || result.Deleted != 0 {
		t.Fatalf("result = %+v, want a refusal that deleted nothing", result)
	}
	if !strings.Contains(result.Reason, "protected") {
		t.Fatalf("reason = %q, want it to name the protection", result.Reason)
	}
	if len(result.EntryResults) != 1 || !result.EntryResults[0].Refused {
		t.Fatalf("entry results = %+v, want one refused entry", result.EntryResults)
	}
}

// A cycle that cannot enumerate what it must not delete must delete nothing.
// The previous implementation returned an empty protected set on every failure
// path, which read at the call site as "this host has nothing to protect".
func TestEnforceFailsClosedWhenProtectedRootsCannotBeResolved(t *testing.T) {
	missingRepo := filepath.Join(t.TempDir(), "not-a-checkout")
	inventory := coreStorage.OwnerInventory{
		RepoRoot: missingRepo,
		Owners: []coreStorage.OwnerManifest{{
			Kind: coreStorage.OwnerResource, ID: "demo",
			ManifestPath:   filepath.Join(missingRepo, "resources", "demo", "resource.json"),
			StorageEntries: []coreStorage.StorageEntry{{Name: "cache", Path: coreStorage.PortablePath{Value: "cache"}, Kind: "dir", Regenerable: true, Budget: &coreStorage.BudgetDeclaration{MaxBytes: "1B"}}},
		}},
	}
	results, err := (Enforcer{RepoRoot: missingRepo, Platform: coreStorage.PlatformLinux}).Enforce(context.Background(), inventory)
	if err == nil {
		t.Fatalf("Enforce = %+v, want an error when protected roots cannot be resolved", results)
	}
	if results != nil {
		t.Fatalf("results = %+v, want none", results)
	}
}

// A wildly wrong ceiling must alarm rather than empty its directory.
func TestEnforceRefusesWhenBudgetWouldRemoveNearlyEverything(t *testing.T) {
	repoRoot := contractFixture(t)
	budgeted := filepath.Join(repoRoot, "resources", "demo", "cache")
	if err := os.MkdirAll(budgeted, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := range 20 {
		name := filepath.Join(budgeted, "entry-"+strconv.Itoa(i))
		if err := os.WriteFile(name, []byte("0123456789"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	inventory := coreStorage.OwnerInventory{
		RepoRoot: repoRoot,
		Owners: []coreStorage.OwnerManifest{{
			Kind: coreStorage.OwnerResource, ID: "demo",
			ManifestPath:   filepath.Join(repoRoot, "resources", "demo", "resource.json"),
			StorageEntries: []coreStorage.StorageEntry{{Name: "cache", Path: coreStorage.PortablePath{Value: "cache"}, Kind: "dir", Regenerable: true, Budget: &coreStorage.BudgetDeclaration{MaxBytes: "1B"}}},
		}},
	}
	results, err := (Enforcer{RepoRoot: repoRoot, Platform: coreStorage.Platform(runtime.GOOS)}).Enforce(context.Background(), inventory)
	if err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	result := results["demo"]
	if !result.Refused || result.Deleted != 0 {
		t.Fatalf("result = %+v, want a refusal that deleted nothing", result)
	}
	entries, err := os.ReadDir(budgeted)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 20 {
		t.Fatalf("remaining entries = %d, want all 20 kept", len(entries))
	}
}

// contractFixture stages a repository root carrying the real contract.
//
// Copying the shipped contract rather than hand-writing a minimal one is
// deliberate: these tests assert that protection is derived from the contract,
// so a fixture that drifts from it would assert nothing about production.
func contractFixture(t *testing.T) string {
	t.Helper()
	source := repoContractPath(t)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read shipped contract: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "repo-contract.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// repoContractPath walks up from the test's working directory to the checkout.
func repoContractPath(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		candidate := filepath.Join(dir, ".vrooli", "repo-contract.json")
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no .vrooli/repo-contract.json above %s", dir)
		}
		dir = parent
	}
}

// Pruning deletes files. When an owner declares it cannot rebuild an entry, the
// deleted bytes are the only copy, so the budget must alarm without destroying.
// vrooli-memory's append-only journal, ollama's downloaded model weights, and
// deployment-manager's data all sit behind this refusal.
func TestEnforceRefusesToPruneNonRegenerableEntryButStillReportsOverage(t *testing.T) {
	root := contractFixture(t)
	scenarioDir := filepath.Join(root, "scenarios", "keeper")
	journal := filepath.Join(scenarioDir, "data")
	if err := os.MkdirAll(journal, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"journal.db", "journal.db-wal"} {
		if err := os.WriteFile(filepath.Join(journal, name), []byte("12345678"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	inventory := coreStorage.OwnerInventory{
		RepoRoot: root,
		Owners: []coreStorage.OwnerManifest{{
			Kind:         coreStorage.OwnerScenario,
			ID:           "keeper",
			ManifestPath: filepath.Join(scenarioDir, ".vrooli", "service.json"),
			StorageEntries: []coreStorage.StorageEntry{{
				Name: "data", Path: coreStorage.PortablePath{Value: "data"}, Kind: "dir", Regenerable: false,
				Budget: &coreStorage.BudgetDeclaration{MaxBytes: "4B", MaxAge: "1h"},
			}},
		}},
	}

	results, err := (Enforcer{RepoRoot: root, Platform: coreStorage.PlatformLinux}).Enforce(context.Background(), inventory)
	if err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	result, ok := results["keeper"]
	if !ok {
		t.Fatal("expected a governed result for the non-regenerable owner")
	}
	if !result.Refused {
		t.Fatalf("expected refusal, got %+v", result)
	}
	if result.Deleted != 0 || result.Freed != 0 {
		t.Fatalf("a non-regenerable entry must never be pruned, got %+v", result)
	}
	if result.Error != "" {
		t.Fatalf("refusal is a governed outcome, not an error: %q", result.Error)
	}
	if result.UsedBytes != 16 {
		t.Fatalf("refusal must still measure usage, got %d", result.UsedBytes)
	}
	if result.OverBytes != 12 {
		t.Fatalf("refusal must still report overage so the budget keeps alarming, got %d", result.OverBytes)
	}
	for _, name := range []string{"journal.db", "journal.db-wal"} {
		if _, statErr := os.Stat(filepath.Join(journal, name)); statErr != nil {
			t.Fatalf("%s was deleted despite regenerable=false: %v", name, statErr)
		}
	}
}

// Every retention deletion must leave a receipt. Without one, "what emptied
// this directory" is unanswerable after the fact, which is the state that made
// the original install-root disappearance impossible to attribute.
func TestEnforceWritesARemovalReceiptForEveryPrunedEntry(t *testing.T) {
	repoRoot := contractFixture(t)
	budgeted := filepath.Join(repoRoot, "resources", "demo", "cache")
	if err := os.MkdirAll(budgeted, 0o755); err != nil {
		t.Fatal(err)
	}
	// Twenty entries of ten bytes against a hundred-byte ceiling prunes the ten
	// oldest: real work, and well inside the blast-radius cap.
	base := time.Now().Add(-24 * time.Hour)
	for i := range 20 {
		name := filepath.Join(budgeted, "entry-"+strconv.Itoa(i))
		if err := os.WriteFile(name, []byte("0123456789"), 0o644); err != nil {
			t.Fatal(err)
		}
		stamp := base.Add(time.Duration(i) * time.Minute)
		if err := os.Chtimes(name, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}

	ledgerDir := filepath.Join(t.TempDir(), "receipts")
	ledger := artifactledger.NewAt(ledgerDir)

	inventory := coreStorage.OwnerInventory{
		RepoRoot: repoRoot,
		Owners: []coreStorage.OwnerManifest{{
			Kind: coreStorage.OwnerResource, ID: "demo",
			ManifestPath:   filepath.Join(repoRoot, "resources", "demo", "resource.json"),
			StorageEntries: []coreStorage.StorageEntry{{Name: "cache", Path: coreStorage.PortablePath{Value: "cache"}, Kind: "dir", Regenerable: true, Budget: &coreStorage.BudgetDeclaration{MaxBytes: "100B"}}},
		}},
	}
	results, err := (Enforcer{RepoRoot: repoRoot, Platform: coreStorage.Platform(runtime.GOOS), Ledger: ledger}).
		Enforce(context.Background(), inventory)
	if err != nil {
		t.Fatalf("Enforce: %v", err)
	}
	result := results["demo"]
	if result.Deleted == 0 {
		t.Fatalf("result = %+v, want deletions", result)
	}

	receipts, err := ledger.Read()
	if err != nil {
		t.Fatalf("read receipts: %v", err)
	}
	removed := map[string]bool{}
	for _, receipt := range receipts {
		if receipt.Outcome != artifactledger.OutcomeRemoved {
			continue
		}
		if receipt.Predicate == "" || receipt.Component == "" {
			t.Fatalf("receipt %+v names no rule or component", receipt)
		}
		removed[receipt.Path] = true
	}
	if len(removed) != result.Deleted {
		t.Fatalf("receipts recorded %d removals, enforcer deleted %d", len(removed), result.Deleted)
	}
}
