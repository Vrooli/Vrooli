package inventory

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	repocontract "github.com/vrooli/repo-contract-go"

	"github.com/vrooli/api-core/uimanifest"

	inventoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ui-health/v1/inventory"
)

// TestScanScenario_RealRepo wires the inventory handler against the live
// repo (no fakes) and scans a known-real react-vite scenario. Regression
// guard for the defaultScenariosRoot off-by-one — if this test ever returns
// 0 surfaces for audio-tools, the inventory pipeline is broken end-to-end.
func TestScanScenario_RealRepo(t *testing.T) {
	repoRoot, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		t.Fatalf("FindRepoRootFromEnvOrCWD: %v", err)
	}
	contract, err := repocontract.LoadDefault(repoRoot)
	if err != nil {
		t.Fatalf("LoadDefault: %v", err)
	}
	scenariosRoot, err := contract.TopLevelDir(repoRoot, "scenarios")
	if err != nil {
		t.Fatalf("TopLevelDir(scenarios): %v", err)
	}

	// Skip if audio-tools isn't checked out in this tree (e.g. shallow
	// worktree). The point of this test is to lock in the wiring, not to
	// require a specific scenario.
	if _, err := os.Stat(filepath.Join(scenariosRoot, "audio-tools", ".vrooli", "service.json")); err != nil {
		t.Skipf("audio-tools not present at %s; skipping live-repo scan", scenariosRoot)
	}

	h := NewConnectHandler(Deps{
		Logger:        log.New(io.Discard, "", 0),
		ManifestLoad:  uimanifest.NewFSLoader(repoRoot),
		ScenariosRoot: scenariosRoot,
	})

	resp, err := h.ScanScenario(context.Background(), connect.NewRequest(&inventoryv1.ScanScenarioRequest{Scenario: "audio-tools"}))
	if err != nil {
		t.Fatalf("ScanScenario: %v", err)
	}
	if got := len(resp.Msg.GetSurfaces()); got == 0 {
		t.Fatalf("expected >0 surfaces for audio-tools; got 0 (the off-by-one bug regressed)")
	}
}

// TestScanScenario_UnknownScenarioReturnsNotFound asserts the inventory
// handler surfaces NotFound (no longer empty success) when the scenario is
// unknown to the manifest loader. This is the observability fix that
// prevents misconfiguration from masquerading as "scan succeeded, 0 results."
func TestScanScenario_UnknownScenarioReturnsNotFound(t *testing.T) {
	tempRepo := t.TempDir()
	// Minimal layout: no scenarios subdir means every scenario name fails.
	h := NewConnectHandler(Deps{
		Logger:        log.New(io.Discard, "", 0),
		ManifestLoad:  uimanifest.NewFSLoader(tempRepo),
		ScenariosRoot: filepath.Join(tempRepo, "scenarios"),
	})

	_, err := h.ScanScenario(context.Background(), connect.NewRequest(&inventoryv1.ScanScenarioRequest{Scenario: "does-not-exist"}))
	if err == nil {
		t.Fatal("expected error for unknown scenario; got nil (the silent-success branch returned)")
	}
	var connectErr *connect.Error
	if !errors.As(err, &connectErr) {
		t.Fatalf("expected *connect.Error; got %T: %v", err, err)
	}
	if connectErr.Code() != connect.CodeNotFound {
		t.Fatalf("expected CodeNotFound; got %s", connectErr.Code())
	}
}

func TestScanIncludesApplicationSourcesOutsideAdoptionSlots(t *testing.T) {
	root := t.TempDir()
	scenario := filepath.Join(root, "scenarios", "sample")
	files := map[string]string{
		".vrooli/service.json":             `{}`,
		"ui/manifest.json":                 `{"slots":{"page":{"dir":"ui/src/pages"}},"files":{"appEntry":{"path":"ui/src/main.tsx"}}}`,
		"ui/src/main.tsx":                  `export const Main = () => <App />`,
		"ui/src/App.tsx":                   `export const App = () => <Catalog />`,
		"ui/src/features/Catalog.tsx":      `export const Catalog = () => <section />`,
		"ui/src/features/Catalog.test.tsx": `test("not production", () => {})`,
		"ui/src/pages/Home.tsx":            `export const Home = () => <main />`,
	}
	for path, source := range files {
		abs := filepath.Join(scenario, path)
		if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(source), 0644); err != nil {
			t.Fatal(err)
		}
	}
	h := NewConnectHandler(Deps{Logger: log.New(io.Discard, "", 0), ManifestLoad: uimanifest.NewFSLoader(root), ScenariosRoot: filepath.Join(root, "scenarios")})
	response, err := h.ScanScenario(context.Background(), connect.NewRequest(&inventoryv1.ScanScenarioRequest{Scenario: "sample"}))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]string{}
	for _, surface := range response.Msg.Surfaces {
		if _, exists := seen[surface.FilePath]; exists {
			t.Fatalf("duplicate source %s", surface.FilePath)
		}
		seen[surface.FilePath] = surface.Slot
	}
	if len(seen) != 4 || seen["ui/src/App.tsx"] == "" || seen["ui/src/features/Catalog.tsx"] == "" || seen["ui/src/pages/Home.tsx"] != "page" {
		t.Fatalf("incomplete or misclassified source inventory: %#v", seen)
	}
}
