package impact

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/ai-provider-go/policy"
)

func TestRealRegistriesContextFlipAndRestore(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repo := filepath.Clean(filepath.Join(filepath.Dir(file), "../../.."))
	ctx, err := policy.LoadRightsContextFile(filepath.Join(repo, "scenarios/ai-gateway/config/model-rights-context.json"))
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := Load(DefaultRegistries(repo), ctx, time.Unix(1, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	oneMillion := ctx
	oneMillionValue := 1000000.0
	oneMillion.RevenueStatus = "exact_value"
	oneMillion.AnnualRevenueUSD = &oneMillionValue
	flipped, err := Load(DefaultRegistries(repo), oneMillion, time.Unix(1, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	thresholdReport, err := Compare(DefaultRegistries(repo), baseline.Context, flipped.Context, time.Unix(1, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	thresholdChanges := changedPaths(thresholdReport)
	if len(thresholdChanges) != 1 {
		t.Fatalf("$1M comparison changed %d paths, want only Stable Audio's $1M music operation: %+v", len(thresholdChanges), thresholdChanges)
	}
	change := thresholdChanges[0]
	if change.Provider != "music" || change.ID != "stabilityai/stable-audio-3-medium" || !strings.Contains(change.Path, "#/1/operations/compose") || change.Before != policy.RightsAllowed || change.After != policy.RightsDenied {
		t.Fatalf("threshold impact = %+v; want Stable Audio compose allowed -> denied", change)
	}

	twentyMillion := ctx
	twentyMillionValue := 20000000.0
	twentyMillion.RevenueStatus = "exact_value"
	twentyMillion.AnnualRevenueUSD = &twentyMillionValue
	twentyMillionReport, err := Compare(DefaultRegistries(repo), baseline.Context, twentyMillion, time.Unix(1, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	twentyMillionChanges := changedPaths(twentyMillionReport)
	if len(twentyMillionChanges) != 2 {
		t.Fatalf("$20M comparison changed %d paths, want the Stable Audio and MiniMax compose operations: %+v", len(twentyMillionChanges), twentyMillionChanges)
	}
	changedModels := map[string]bool{}
	for _, changed := range twentyMillionChanges {
		if changed.Provider != "music" || !strings.Contains(changed.Path, "/operations/compose") || changed.Before != policy.RightsAllowed || changed.After != policy.RightsDenied {
			t.Errorf("$20M rights change = %+v; want a music compose path allowed -> denied", changed)
		}
		changedModels[changed.ID] = true
	}
	if !changedModels["stabilityai/stable-audio-3-medium"] || !changedModels["MiniMaxAI/MiniMax-Music3"] {
		t.Fatalf("$20M change omitted one of the sourced conditional models: %+v", changedModels)
	}

	ctx.AnnualRevenueUSD = nil
	restored, err := Load(DefaultRegistries(repo), ctx, time.Unix(1, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	restoreReport, err := Compare(DefaultRegistries(repo), flipped.Context, restored.Context, time.Unix(1, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	restoreChanges := changedPaths(restoreReport)
	if len(restoreChanges) != 1 {
		t.Fatalf("restoration changed %d paths, want Stable Audio compose only: %+v", len(restoreChanges), restoreChanges)
	}
	if restoreChanges[0].Provider != change.Provider || restoreChanges[0].ID != change.ID || restoreChanges[0].Path != change.Path || restoreChanges[0].Before != policy.RightsDenied || restoreChanges[0].After != policy.RightsAllowed {
		t.Fatalf("restoration impact = %+v; want same Stable Audio compose path denied -> allowed", restoreChanges[0])
	}
	twentyMillionRestore, err := Compare(DefaultRegistries(repo), twentyMillion, restored.Context, time.Unix(1, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if changes := changedPaths(twentyMillionRestore); len(changes) != 2 {
		t.Fatalf("restoring from $20M changed %d paths, want both sourced models restored: %+v", len(changes), changes)
	}

	providers := map[string]int{}
	for _, m := range baseline.Models {
		providers[m.Provider]++
	}
	// The inventory must cover every entry the registries declare; the
	// expected counts come from the registry files, not from a literal that
	// drifts each time a model is added.
	wantTotal := 0
	for _, registry := range DefaultRegistries(repo) {
		want := sourceEntryCount(t, registry.Path)
		wantTotal += want
		if providers[registry.Provider] != want {
			t.Errorf("real registry inventory has %d %s models, want all %d source entries", providers[registry.Provider], registry.Provider, want)
		}
	}
	if len(baseline.Models) != wantTotal {
		t.Fatalf("real registry inventory has %d models, want all %d source entries", len(baseline.Models), wantTotal)
	}
	for _, model := range baseline.Models {
		if len(model.Paths) == 0 {
			t.Errorf("real registry model %s/%s has no selection/operation path", model.Provider, model.ID)
		}
		for _, path := range model.Paths {
			if path.Decision.Status != policy.RightsAllowed && path.Decision.Status != policy.RightsDenied && path.Decision.Status != policy.RightsUnknown {
				t.Errorf("real registry path %s has invalid rights status %q", path.Path, path.Decision.Status)
			}
		}
	}
	baseByKey, flipByKey, restoreByKey := decisionsByKey(baseline), decisionsByKey(flipped), decisionsByKey(restored)
	if len(baseByKey) != len(flipByKey) || len(baseByKey) != len(restoreByKey) {
		t.Fatalf("inventory changed across context flip: %d, %d, %d", len(baseByKey), len(flipByKey), len(restoreByKey))
	}
	changed := 0
	for key, before := range baseByKey {
		after, ok := flipByKey[key]
		if !ok {
			t.Fatalf("flipped report missing %s", key)
		}
		back, ok := restoreByKey[key]
		if !ok {
			t.Fatalf("restored report missing %s", key)
		}
		if before != back {
			t.Errorf("restoring pre-revenue context did not restore %s: %q vs %q", key, before, back)
		}
		if before != after {
			changed++
		}
	}
	if changed != 1 {
		t.Errorf("expected only the sourced Stable Audio compose path to change at $1M; found %d", changed)
	}
	for _, model := range baseline.Models {
		for _, path := range model.Paths {
			if path.Use != policy.RightsUseMonetizedUse {
				t.Errorf("registry inventory reported a packaging path without a declared model_artifact: %s", path.Path)
			}
		}
	}
	baseReasons, flipReasons := reasonsByKey(baseline), reasonsByKey(flipped)
	for key, reason := range baseReasons {
		if baseByKey[key] != flipByKey[key] {
			continue
		}
		if flipReasons[key] != reason {
			t.Errorf("non-conditional real-registry decision changed reason for %s: %s -> %s", key, reason, flipReasons[key])
		}
	}
}

func TestRealOllamaMonthlyActiveUserPathsFlipAndRestore(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repo := filepath.Clean(filepath.Join(filepath.Dir(file), "../../.."))
	ctx, err := policy.LoadRightsContextFile(filepath.Join(repo, "scenarios/ai-gateway/config/model-rights-context.json"))
	if err != nil {
		t.Fatal(err)
	}
	const overThreshold = "700000001"
	flipped := ctx
	flipped.MonthlyActiveUsers = cloneFacts(ctx.MonthlyActiveUsers)
	flipped.MonthlyActiveUsersEvidence = cloneFacts(ctx.MonthlyActiveUsersEvidence)
	for _, period := range []string{"2023-06", "2024-06", "2024-08"} {
		flipped.MonthlyActiveUsers[period] = overThreshold
		flipped.MonthlyActiveUsersEvidence[period] = "hypothetical LM5 threshold-flip comparison"
	}
	registries := DefaultRegistries(repo)
	at := time.Date(2026, 11, 14, 22, 0, 0, 0, time.UTC)
	refused, err := Compare(registries, ctx, flipped, at)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Compare(registries, flipped, ctx, at)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{
		"codellama:7b": 1,
		"llama3.1:8b":  2,
		"llama3.2:1b":  1,
		"llama3.2:3b":  3,
	}
	changedPathCount := 0
	changedRoutes := map[string]bool{}
	for _, model := range refused.Models {
		for _, path := range model.Paths {
			if !path.Changed {
				continue
			}
			if model.Provider != "ollama" {
				t.Errorf("unexpected non-Ollama MAU path changed: %s/%s %s", model.Provider, model.ID, path.Path)
				continue
			}
			if _, ok := want[model.ID]; !ok {
				t.Errorf("unexpected Ollama MAU path changed: %s %s", model.ID, path.Path)
				continue
			}
			if path.PreviousStatus != policy.RightsAllowed || path.Decision.Status != policy.RightsDenied {
				t.Errorf("Ollama MAU path %s/%s changed %s -> %s, want allowed -> denied", model.ID, path.Path, path.PreviousStatus, path.Decision.Status)
			}
			want[model.ID]--
			changedRoutes[model.ID+"|"+path.Path] = true
			changedPathCount++
		}
	}
	for id, remaining := range want {
		if remaining != 0 {
			t.Errorf("threshold flip affected %d fewer paths for Ollama model %s", remaining, id)
		}
	}
	if changedPathCount != 7 {
		t.Fatalf("MAU threshold flip changed %d paths, want exactly 7 real registry rows", changedPathCount)
	}
	restoredRoutes := map[string]bool{}
	for _, model := range restored.Models {
		for _, path := range model.Paths {
			if !path.Changed {
				continue
			}
			key := model.ID + "|" + path.Path
			if model.Provider != "ollama" || !changedRoutes[key] || path.PreviousStatus != policy.RightsDenied || path.Decision.Status != policy.RightsAllowed {
				t.Errorf("restored Ollama path %s/%s is not denied -> allowed: %+v", model.ID, path.Path, path)
			}
			restoredRoutes[key] = true
		}
	}
	if len(restoredRoutes) != len(changedRoutes) {
		t.Fatalf("restoration affected %d paths, want the same %d paths", len(restoredRoutes), len(changedRoutes))
	}
	for route := range changedRoutes {
		if !restoredRoutes[route] {
			t.Errorf("restoration omitted path %s", route)
		}
	}
}

func TestRealRegistriesCombinedContextFlipRestoresEveryAffectedRoute(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repo := filepath.Clean(filepath.Join(filepath.Dir(file), "../../.."))
	base, err := policy.LoadRightsContextFile(filepath.Join(repo, "scenarios/ai-gateway/config/model-rights-context.json"))
	if err != nil {
		t.Fatal(err)
	}
	target := base
	target.MonthlyActiveUsers = cloneFacts(base.MonthlyActiveUsers)
	target.MonthlyActiveUsersEvidence = cloneFacts(base.MonthlyActiveUsersEvidence)
	target.RevenueStatus = "at_or_above_threshold"
	for _, period := range []string{"2023-06", "2024-06", "2024-08"} {
		target.MonthlyActiveUsers[period] = "700000001"
		target.MonthlyActiveUsersEvidence[period] = "hypothetical combined LM5 impact comparison"
	}
	at := time.Date(2026, 11, 14, 23, 0, 0, 0, time.UTC)
	registries := DefaultRegistries(repo)
	refused, err := Compare(registries, base, target, at)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Compare(registries, target, base, at)
	if err != nil {
		t.Fatal(err)
	}
	// Route paths are produced from the real registry; assert the model/use
	// identity and exact path set below, including the two music compose paths.
	wantRoutes := map[string]bool{}
	for _, item := range []struct {
		provider, model string
		count           int
	}{
		{"ollama", "codellama:7b", 1}, {"ollama", "llama3.1:8b", 2},
		{"ollama", "llama3.2:1b", 1}, {"ollama", "llama3.2:3b", 3},
		{"music", "MiniMaxAI/MiniMax-Music3", 1}, {"music", "stabilityai/stable-audio-3-medium", 1},
	} {
		seen := 0
		for _, model := range refused.Models {
			if model.Provider != item.provider || model.ID != item.model {
				continue
			}
			for _, path := range model.Paths {
				if !path.Changed {
					continue
				}
				if path.PreviousStatus != policy.RightsAllowed || path.Decision.Status != policy.RightsDenied {
					t.Errorf("%s/%s %s changed %s -> %s, want allowed -> denied", model.Provider, model.ID, path.Path, path.PreviousStatus, path.Decision.Status)
				}
				wantRoutes[model.Provider+"|"+model.ID+"|"+path.Path] = true
				seen++
			}
		}
		if seen != item.count {
			t.Errorf("combined flip changed %d paths for %s/%s, want %d", seen, item.provider, item.model, item.count)
		}
	}
	if len(wantRoutes) != 9 {
		t.Fatalf("combined flip identified %d exact changed routes, want 9: %+v", len(wantRoutes), wantRoutes)
	}
	restoredRoutes := map[string]bool{}
	for _, model := range restored.Models {
		for _, path := range model.Paths {
			if !path.Changed {
				continue
			}
			key := model.Provider + "|" + model.ID + "|" + path.Path
			if !wantRoutes[key] || path.PreviousStatus != policy.RightsDenied || path.Decision.Status != policy.RightsAllowed {
				t.Errorf("restoration changed unexpected route or status: %s %s -> %s", key, path.PreviousStatus, path.Decision.Status)
			}
			restoredRoutes[key] = true
		}
	}
	if len(restoredRoutes) != len(wantRoutes) {
		t.Fatalf("restoration returned %d paths, want same %d routes", len(restoredRoutes), len(wantRoutes))
	}
}

func cloneFacts(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

type statusChange struct {
	Provider string
	ID       string
	Path     string
	Before   policy.RightsStatus
	After    policy.RightsStatus
}

func changedPaths(report Report) []statusChange {
	var out []statusChange
	for _, model := range report.Models {
		for _, path := range model.Paths {
			if path.Changed {
				out = append(out, statusChange{Provider: model.Provider, ID: model.ID, Path: path.Path, Before: path.PreviousStatus, After: path.Decision.Status})
			}
		}
	}
	return out
}

func TestConditionalGrantPathChangesAndRestores(t *testing.T) {
	base := policy.RightsContext{ID: "business", Version: "1", EntityScope: "entity-A", RevenueStatus: "pre_revenue", AsOf: "2026-11-14", Source: "fixture"}
	threshold := base
	threshold.RevenueStatus = "at_or_above_threshold"
	artifact := map[string]any{
		"artifact_id": "fixture/model", "revision": "r1", "sha256": strings.Repeat("a", 64), "license_id": "fixture-license", "classification": "conditional",
		"evidence":           map[string]any{"url": "https://example.test/license", "revision": "r1", "sha256": strings.Repeat("b", 64), "observed_at": "2026-11-14"},
		"conditional_grants": []any{map[string]any{"id": "under-threshold", "uses": []any{"local_inference"}, "conditions": []any{map[string]any{"id": "revenue", "kind": "annual_revenue", "comparator": "lt", "threshold": 10000000, "unit": "USD", "entity_scope": "entity-A", "interpretation": "under threshold"}}}},
	}
	registry := Registry{Provider: "fixture", Path: "fixture.json", Label: "fixture.json", Document: map[string]any{"roles": map[string]any{"select": map[string]any{"model": "model"}}, "models": map[string]any{"model": map[string]any{"rights": artifact}}}}
	denied, err := Compare([]Registry{registry}, base, threshold, time.Unix(1, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	allowedAgain, err := Compare([]Registry{registry}, threshold, base, time.Unix(1, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range []struct {
		report   Report
		from, to policy.RightsStatus
	}{{denied, policy.RightsAllowed, policy.RightsDenied}, {allowedAgain, policy.RightsDenied, policy.RightsAllowed}} {
		found := false
		for _, model := range pair.report.Models {
			for _, path := range model.Paths {
				if strings.Contains(path.Path, "#/roles/select") && path.Changed && path.PreviousStatus == pair.from && path.Decision.Status == pair.to {
					found = true
				}
			}
		}
		if !found {
			t.Errorf("fixture context change did not report role path %s -> %s", pair.from, pair.to)
		}
	}
}

func decisionsByKey(report Report) map[string]string {
	out := map[string]string{}
	for _, model := range report.Models {
		for _, path := range model.Paths {
			key := model.Provider + "/" + model.ID + "|" + string(path.Use) + "|" + path.Path
			out[key] = string(path.Decision.Status)
		}
	}
	return out
}

func reasonsByKey(report Report) map[string]string {
	out := map[string]string{}
	for _, model := range report.Models {
		for _, path := range model.Paths {
			key := model.Provider + "/" + model.ID + "|" + string(path.Use) + "|" + path.Path
			out[key] = path.Decision.Reason
		}
	}
	return out
}

// sourceEntryCount counts the models a registry file declares, independent of
// the impact loader: a "models" object or array, or a root array.
func sourceEntryCount(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Models json.RawMessage `json:"models"`
	}
	if err := json.Unmarshal(data, &doc); err == nil && len(doc.Models) > 0 {
		var object map[string]json.RawMessage
		if json.Unmarshal(doc.Models, &object) == nil {
			return len(object)
		}
		var array []json.RawMessage
		if json.Unmarshal(doc.Models, &array) == nil {
			return len(array)
		}
	}
	var root []json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("%s: no models object, array or root array: %v", path, err)
	}
	return len(root)
}
