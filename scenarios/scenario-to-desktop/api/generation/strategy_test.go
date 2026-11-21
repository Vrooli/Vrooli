package generation

import (
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSelectedStrategyHostRequirementsAreScoped(t *testing.T) {
	strategies := PackagingStrategies().List()
	var electronTools, wailsTools []string
	for _, strategy := range strategies {
		switch strategy.ID {
		case DefaultStrategyID:
			electronTools = strategy.Capabilities.HostTools
		case WailsStrategyID:
			wailsTools = strategy.Capabilities.HostTools
		}
	}
	if len(electronTools) != 0 {
		t.Fatalf("Electron inherited strategy-only host tools: %v", electronTools)
	}
	if !reflect.DeepEqual(wailsTools, []string{"webkitgtk", "webkit2gtk-driver"}) {
		t.Fatalf("Wails host tools = %v", wailsTools)
	}
	if err := ValidateSelectedStrategyHostTools(DefaultStrategyID, "linux-x64"); err != nil {
		t.Fatalf("Electron host preflight should not require WebKitGTK: %v", err)
	}
}

func TestPackagedEvidenceProviderUsesPublishedReportContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/packaged/report" || r.URL.Query().Get("scenario") != "demo" || r.URL.Query().Get("ramp") != "desktop" || r.URL.Query().Get("strategy") != DefaultStrategyID {
			t.Fatalf("request=%s?%s", r.URL.Path, r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"groups":[{"scenario":"demo","ramp":"desktop","strategy":"electron-sidecar","platform":"linux-x64","condition":"cold","comparable":true,"samples":3,"metrics":{"app_interactive_ms":{"p50":1000}}}]}`))
	}))
	defer server.Close()

	report, err := NewPackagedEvidenceProvider(server.URL)(context.Background(), "demo", DefaultStrategyID)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Groups) != 1 || !report.Groups[0].Comparable || report.Groups[0].Samples != 3 || report.Groups[0].Metrics["app_interactive_ms"] == nil {
		t.Fatalf("report=%+v", report)
	}
}

func TestResolvePackagingSelectionRecordsOverrideAndFacts(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "api"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "api", "go.mod"), []byte("module sample\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "service.json"), []byte(`{"components":{"api":{"role":"api","build":{"kind":"go_module","dir":"api"}}},"dependencies":{"resources":{"sqlite":{}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	record, err := ResolvePackagingSelection(root, "", "linux-x64")
	if err != nil {
		t.Fatal(err)
	}
	if record.Selected.StrategyID != DefaultStrategyID || record.Selected.Override || record.PolicyVersion != SelectionPolicyV1 {
		t.Fatalf("selection=%+v", record)
	}
	if len(record.StackFacts.Components) != 1 || record.StackFacts.Components[0].BuildKind != "go_module" || len(record.StackFacts.Resources) != 1 {
		t.Fatalf("facts=%+v", record.StackFacts)
	}
	overridden, err := ResolvePackagingSelection(root, DefaultStrategyID, "linux-x64")
	if err != nil || !overridden.Selected.Override || overridden.Selected.StrategyID != DefaultStrategyID {
		t.Fatalf("override selection=%+v err=%v", overridden, err)
	}
}

func TestStrategyRoutesListAndExplainJSON(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "demo", ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "demo", ".vrooli", "service.json"), []byte(`{"components":{"api":{"role":"api","build":{"kind":"go_module","dir":"api"}}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	RegisterStrategyRoutes(router, root)
	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest("GET", "/api/v1/strategies", nil))
	if list.Code != 200 {
		t.Fatalf("list status=%d body=%s", list.Code, list.Body.String())
	}
	var listed map[string]any
	if err := json.Unmarshal(list.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	strategies, ok := listed["strategies"].([]any)
	if !ok || len(strategies) != 3 || listed["default_strategy"] != DefaultStrategyID || listed["policy_version"] != SelectionPolicyV1 {
		t.Fatalf("list=%+v", listed)
	}
	defaults := 0
	for _, raw := range strategies {
		if entry, ok := raw.(map[string]any); ok && entry["default"] == true {
			defaults++
		}
	}
	if defaults != 1 {
		t.Fatalf("expected exactly one default strategy, got %d", defaults)
	}
	explain := httptest.NewRecorder()
	router.ServeHTTP(explain, httptest.NewRequest("GET", "/api/v1/strategies/demo/explain?strategy=electron-sidecar", nil))
	if explain.Code != 200 {
		t.Fatalf("explain status=%d body=%s", explain.Code, explain.Body.String())
	}
	var result PackagingSelectionRecord
	if err := json.Unmarshal(explain.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Selected.StrategyID != DefaultStrategyID || !result.Selected.Override || len(result.Eligibility) != 3 {
		t.Fatalf("explain=%+v", result)
	}
	for _, eligibility := range result.Eligibility {
		if eligibility.Strategy.ID == DefaultStrategyID && !eligibility.Eligible {
			t.Fatalf("default unexpectedly ineligible: %+v", eligibility)
		}
	}
}

func TestWailsEligibilityRequiresGoAPIAndEmbedContract(t *testing.T) {
	root := t.TempDir()
	api := filepath.Join(root, "api")
	if err := os.MkdirAll(filepath.Join(api, "desktop"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".vrooli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(api, "go.mod"), []byte("module example/api\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	service := `{"components":{"api":{"role":"api","build":{"kind":"go_module","dir":"api"}}}}`
	if err := os.WriteFile(filepath.Join(root, ".vrooli", "service.json"), []byte(service), 0o644); err != nil {
		t.Fatal(err)
	}
	eligible := PackagingStrategies().Evaluate(ReadStackFacts(root), "linux-x64")
	for _, result := range eligible {
		if result.Strategy.ID == WailsStrategyID && result.Eligible {
			t.Fatal("Wails became eligible without the API embed descriptor")
		}
		if result.Strategy.ID == DefaultStrategyID && !result.Eligible {
			t.Fatalf("Electron sidecar default became ineligible: %+v", result)
		}
	}
	if err := os.WriteFile(filepath.Join(api, "desktop", "embed.go"), []byte("package desktop\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	facts := ReadStackFacts(root)
	if len(facts.Components) != 1 || !facts.Components[0].EmbeddableServer {
		t.Fatalf("embed fact=%+v", facts.Components)
	}
	eligible = PackagingStrategies().Evaluate(facts, "linux-x64")
	for _, result := range eligible {
		if result.Strategy.ID == WailsStrategyID && !result.Eligible {
			t.Fatalf("Wails rejected an API with Go module and embed contract: %+v", result)
		}
	}
	if strategy, ok := StrategyForShell("electron"); !ok || strategy.ID != DefaultStrategyID {
		t.Fatalf("legacy Electron shell did not resolve to default: %+v, %v", strategy, ok)
	}
}

func TestStrategyExplainIncludesPublishedPackagedEvidence(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	router := mux.NewRouter()
	RegisterStrategyRoutes(router, root, func(_ context.Context, scenario, strategy string) (*packagedEvidenceReport, error) {
		if scenario != "demo" || strategy != DefaultStrategyID {
			t.Fatalf("provider args=%q,%q", scenario, strategy)
		}
		return &packagedEvidenceReport{Groups: []packagedEvidenceGroup{{Scenario: "demo", Ramp: "desktop", Strategy: DefaultStrategyID, Platform: "linux-x64", Condition: "cold", Comparable: true, Samples: 3}}}, nil
	})
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/strategies/demo/explain", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var result strategyExplainResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.EvidenceStatus != "available" || result.PackagedEvidence == nil || len(result.PackagedEvidence.Groups) != 1 || result.PackagedEvidence.Groups[0].Samples != 3 {
		t.Fatalf("evidence response=%+v", result)
	}
}

func TestCopyTemplateTreeMaterializesInRootStrategyAlias(t *testing.T) {
	root := t.TempDir()
	vanilla := filepath.Join(root, "vanilla")
	if err := os.MkdirAll(vanilla, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(vanilla, "main.ts"), []byte("baseline"), 0o644); err != nil {
		t.Fatal(err)
	}
	strategies := filepath.Join(root, "strategies")
	if err := os.MkdirAll(strategies, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../vanilla", filepath.Join(strategies, "electron-sidecar")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	destination := filepath.Join(t.TempDir(), "snapshot")
	if err := copyTemplateTree(root, destination); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(filepath.Join(destination, "strategies", "electron-sidecar", "main.ts"))
	if err != nil || string(contents) != "baseline" {
		t.Fatalf("copied strategy alias=%q err=%v", contents, err)
	}
}

func TestStrategyOutputNamePreservesElectronOutputPath(t *testing.T) {
	for _, id := range []string{DefaultStrategyID, "electron"} {
		got, err := StrategyOutputName(id)
		if err != nil || got != "electron" {
			t.Fatalf("StrategyOutputName(%q)=(%q,%v)", id, got, err)
		}
	}
	if _, err := StrategyOutputName("unregistered"); err == nil {
		t.Fatal("expected unregistered strategy rejection")
	}
}

func TestOptimizedProfileToggleAttributionIsIndependentAndBaselineUnchanged(t *testing.T) {
	profiles := []ElectronOptimizationProfile{
		{SharedRuntimeReadiness: true},
		{SecureCustomSchemeUI: true},
		{V8SnapshotCodeCache: true},
		{ElectronFuses: true},
	}
	identities := map[string]bool{}
	for _, profile := range profiles {
		if err := validateOptimizationProfile(OptimizedStrategyID, &profile); err != nil {
			t.Fatalf("single toggle rejected: %+v: %v", profile, err)
		}
		if identities[profile.Identity()] {
			t.Fatalf("toggle identity collided: %s", profile.Identity())
		}
		identities[profile.Identity()] = true
	}
	if err := validateOptimizationProfile(DefaultStrategyID, nil); err != nil {
		t.Fatalf("sidecar default rejected: %v", err)
	}
	if err := validateOptimizationProfile(DefaultStrategyID, &profiles[0]); err == nil {
		t.Fatal("sidecar accepted an optimization toggle")
	}
	if err := validateOptimizationProfile(OptimizedStrategyID, nil); err == nil {
		t.Fatal("optimized profile accepted unspecified toggles")
	}
	if got, err := StrategyOutputName(DefaultStrategyID); err != nil || got != "electron" {
		t.Fatalf("baseline output=%q err=%v", got, err)
	}
	if got, err := StrategyOutputName(OptimizedStrategyID); err != nil || got != OptimizedStrategyID {
		t.Fatalf("optimized output=%q err=%v", got, err)
	}
}
