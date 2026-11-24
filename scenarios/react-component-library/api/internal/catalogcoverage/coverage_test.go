package catalogcoverage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for range 8 {
		if info, err := os.Stat(filepath.Join(dir, "scenarios", "react-component-library", "catalog")); err == nil && info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Skip("catalog not found from working directory")
	return ""
}

func live(t *testing.T) ([]Asset, []Implementation) {
	t.Helper()
	root := repoRoot(t)
	assets, err := LoadCatalog(filepath.Join(root, "scenarios", "react-component-library", "catalog"))
	if err != nil {
		t.Fatalf("load catalog: %v", err)
	}
	impls, err := LoadImplementations(filepath.Join(root, "scenarios", "react-component-library", "library"))
	if err != nil {
		t.Fatalf("load implementations: %v", err)
	}
	return assets, impls
}

func TestLoadCatalogReadsEveryAsset(t *testing.T) {
	assets, _ := live(t)
	// Relational, not a recorded population: compare against what is actually on
	// disk so the test keeps meaning as the catalog grows or shrinks.
	root := repoRoot(t)
	onDisk, err := filepath.Glob(filepath.Join(root, "scenarios", "react-component-library", "catalog", "assets", "*", "*.json"))
	if err != nil {
		t.Fatalf("glob catalog assets: %v", err)
	}
	if len(onDisk) == 0 {
		t.Fatal("no catalog asset files found on disk; the scan found nothing to load")
	}
	if len(assets) != len(onDisk) {
		t.Fatalf("loaded %d assets from %d catalog files; every asset file must be read", len(assets), len(onDisk))
	}
	for _, a := range assets {
		if a.ID == "" || a.Domain == "" || a.Priority == "" || a.Delivery == "" {
			t.Fatalf("asset %+v is missing required identity fields", a)
		}
	}
}

func TestGateDefinitionsExposeAllThreeTiersAndRealBrowserInputs(t *testing.T) {
	root := repoRoot(t)
	definitions, err := LoadGateDefinitions(filepath.Join(root, "scenarios", "react-component-library", "catalog", "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]int{}
	byID := map[string]GateDefinition{}
	for _, definition := range definitions {
		seen[definition.Tier]++
		byID[definition.ID] = definition
	}
	if len(definitions) != 64 {
		t.Fatalf("full gate corpus has %d definitions; expected the unchanged 64-gate corpus", len(definitions))
	}
	for tier, want := range map[string]int{"release-integrity": 23, "asset-quality": 35, "consumer-use": 6} {
		if seen[tier] != want {
			t.Errorf("full-corpus tier %q has %d gates, want %d", tier, seen[tier], want)
		}
	}
	for _, tier := range []string{"release-integrity", "asset-quality", "consumer-use"} {
		if seen[tier] == 0 {
			t.Errorf("tier %q has no gates", tier)
		}
	}
	responsive := byID["responsive"]
	if len(responsive.RunnerConfig.RequiredViewportWidthsPx) != 2 || responsive.RunnerConfig.RequiredViewportWidthsPx[0] != 320 || responsive.RunnerConfig.RequiredViewportWidthsPx[1] != 390 {
		t.Fatalf("responsive gate lacks the 320/390 px browser requirement: %+v", responsive.RunnerConfig)
	}
	if !byID["accessibility"].RunnerConfig.RequireAxeResults {
		t.Fatal("accessibility gate must require an actual axe-core result")
	}
}

func TestOptOutReasonsAreCountedInCoverage(t *testing.T) {
	assets := []Asset{{ID: "controls.button", Name: "Button", Kind: "component", Maturity: "verified", Targets: []string{"react-vite"}, QualityGateOptOuts: []QualityGateOptOut{{Gate: "responsive", Reason: "This source-only token asset has no rendered surface."}}}}
	impls := []Implementation{{Name: "Button", Root: "components", CatalogID: "controls.button"}}
	report := ComputeWithEvidence(assets, impls, nil, []GateDefinition{{ID: "responsive", Tier: "asset-quality", Rung: RungScaffolded, Blocking: true, AppliesTo: []string{"component"}}})
	if report.QualityGateOptOutCount != 1 {
		t.Fatalf("quality gate opt-out count = %d, want 1", report.QualityGateOptOutCount)
	}
}

func TestFullCatalogGateClassificationAndOptOutCoverage(t *testing.T) {
	assets, impls := live(t)
	root := repoRoot(t)
	catalogRoot := filepath.Join(root, "scenarios", "react-component-library", "catalog")
	definitions, err := LoadGateDefinitions(filepath.Join(catalogRoot, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(definitions) != 64 {
		t.Fatalf("loaded %d current gates, want all 64", len(definitions))
	}
	var optOutCount int
	for _, asset := range assets {
		optOutCount += len(asset.QualityGateOptOuts)
		got, err := ResolveRuleSetFor(catalogRoot, asset, definitions)
		if err != nil {
			t.Errorf("resolve current gate classification for %s: %v", asset.ID, err)
			continue
		}
		want, err := legacyRuleBindings(catalogRoot, asset, definitions)
		if err != nil {
			t.Errorf("resolve pre-tier classification for %s: %v", asset.ID, err)
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("gate classification changed for %s: got %+v, pre-tier %+v", asset.ID, got, want)
		}
	}
	report := ComputeWithEvidence(assets, impls, nil, definitions)
	if report.QualityGateOptOutCount != optOutCount {
		t.Fatalf("full-catalog opt-out report count = %d, parsed declarations = %d", report.QualityGateOptOutCount, optOutCount)
	}
}

// legacyRuleBindings models the pre-tier flat classifier. The three-tier
// projection must preserve every asset/gate binding before any retired path
// is removed from the corpus.
func legacyRuleBindings(catalogRoot string, asset Asset, definitions []GateDefinition) ([]RuleBinding, error) {
	overrides, err := loadAssetRuleOverrides(catalogRoot, asset.ID)
	if err != nil {
		return nil, err
	}
	optOut := map[string]bool{}
	for _, item := range overrides.OptOuts {
		optOut[item.Gate] = true
	}
	added := map[string]bool{}
	for _, gate := range overrides.Gates {
		added[gate] = true
	}
	out := []RuleBinding{}
	for _, definition := range definitions {
		if optOut[definition.ID] {
			continue
		}
		if added[definition.ID] {
			out = append(out, RuleBinding{GateID: definition.ID, Source: RuleSourceAsset, DeclaredIn: assetDeclarationPath(catalogRoot, asset.ID)})
			continue
		}
		if !slices.Contains(definition.AppliesTo, asset.Kind) {
			continue
		}
		source := RuleSourceKind
		if appliesToAllKinds(definition.AppliesTo) {
			source = RuleSourceUniversal
		}
		if definition.Attribution == "corpus" {
			source = RuleSourceCorpus
		}
		out = append(out, RuleBinding{GateID: definition.ID, Source: source, DeclaredIn: filepath.Join("scenarios", "react-component-library", "catalog", "config.json")})
	}
	slices.SortFunc(out, func(a, b RuleBinding) int { return strings.Compare(a.GateID, b.GateID) })
	return out, nil
}

// TestEveryRowIsAccountedFor is the integrity property of the join: no asset and
// no implementation may vanish, and nothing may be counted twice.
func TestEveryRowIsAccountedFor(t *testing.T) {
	assets, impls := live(t)
	rep := Compute(assets, impls)

	planned := rep.Totals[BucketPlannedBuilt] + rep.Totals[BucketPlannedUnbuilt]
	if planned != len(assets) {
		t.Errorf("planned rows %d != catalog assets %d", planned, len(assets))
	}
	accountedImpls := rep.Totals[BucketPlannedBuilt] + rep.Totals[BucketSupplemental]
	if accountedImpls != len(impls) {
		t.Errorf("built+supplemental %d != implementations %d", accountedImpls, len(impls))
	}
	if len(rep.Rows) != planned+rep.Totals[BucketSupplemental] {
		t.Errorf("row count %d does not equal planned %d plus supplemental %d",
			len(rep.Rows), planned, rep.Totals[BucketSupplemental])
	}
}

func TestDomainAndPriorityRollupsSumToPlanned(t *testing.T) {
	assets, impls := live(t)
	rep := Compute(assets, impls)
	planned := rep.Totals[BucketPlannedBuilt] + rep.Totals[BucketPlannedUnbuilt]

	var byDomain, byPriority int
	for _, c := range rep.ByDomain {
		byDomain += c.Planned
	}
	for _, c := range rep.ByPriority {
		byPriority += c.Planned
	}
	if byDomain != planned {
		t.Errorf("domain rollup %d != planned %d", byDomain, planned)
	}
	if byPriority != planned {
		t.Errorf("priority rollup %d != planned %d", byPriority, planned)
	}
}

// TestNextWorkIsDeterministicAndLeveraged asserts the ordering contract that
// makes an agent loop reproducible: highest downstream impact first, then
// priority, then id, with no ties left to map iteration order.
func TestNextWorkIsDeterministicAndLeveraged(t *testing.T) {
	assets, impls := live(t)
	rep := Compute(assets, impls)
	first := NextWork(rep, 10)
	if len(first) == 0 {
		t.Fatal("expected unbuilt assets")
	}
	for i := 1; i < len(first); i++ {
		a, b := first[i-1], first[i]
		if a.BlocksDownstream < b.BlocksDownstream {
			t.Fatalf("next-work not ordered by leverage: %s(%d) before %s(%d)",
				a.AssetID, a.BlocksDownstream, b.AssetID, b.BlocksDownstream)
		}
	}
	second := NextWork(rep, 10)
	for i := range first {
		if first[i].AssetID != second[i].AssetID {
			t.Fatal("next-work ordering is not deterministic across calls")
		}
	}
	t.Logf("next target: %s (blocks %d)", first[0].AssetID, first[0].BlocksDownstream)
}

func TestBlockedCountsIgnoreSuggests(t *testing.T) {
	assets := []Asset{
		{ID: "a.one", Requires: []string{"a.two"}, Suggests: []string{"a.three"}},
		{ID: "a.two"},
		{ID: "a.three"},
	}
	counts := blockedCounts(assets)
	if counts["a.two"] != 1 {
		t.Errorf("required asset should be blocked-by 1, got %d", counts["a.two"])
	}
	if counts["a.three"] != 0 {
		t.Errorf("suggested asset must never count as blocking, got %d", counts["a.three"])
	}
}

func TestDanglingCatalogIDIsSurfaced(t *testing.T) {
	assets := []Asset{{ID: "a.one", Domain: "a", Priority: "P0"}}
	impls := []Implementation{{Name: "Ghost", Root: "components", CatalogID: "a.missing"}}
	rep := Compute(assets, impls)
	var found bool
	for _, row := range rep.Rows {
		if row.Implementation == "Ghost" && row.Domain == "(dangling catalogId)" {
			found = true
		}
	}
	if !found {
		t.Fatal("an implementation pointing at a nonexistent catalog id must be surfaced, not silently supplemental")
	}
}

func TestCoverageDerivesMaturityFromEvidence(t *testing.T) {
	assets := []Asset{{ID: "controls.button", Name: "Button", Kind: "component", Domain: "controls", Priority: "P0", Maturity: "verified", Targets: []string{"react-vite"}}}
	impls := []Implementation{{Name: "Button", Root: "components", CatalogID: "controls.button"}}
	gates := []GateDefinition{
		{ID: "types", Rung: RungScaffolded, Blocking: true, AppliesTo: []string{"component"}},
		{ID: "api", Rung: RungImplemented, Blocking: true, AppliesTo: []string{"component"}},
		{ID: "visual", Rung: RungVerified, Blocking: true, AppliesTo: []string{"component"}},
	}
	noEvidence := ComputeWithEvidence(assets, impls, nil, gates)
	if got := noEvidence.Rows[0].Achieved; got != RungScaffolded {
		t.Fatalf("linked implementation without passing gates achieved %q, want scaffolded", got)
	}
	partial := ComputeWithEvidence(assets, impls, []GateEvidence{{AssetID: "controls.button", Target: "react-vite", Gate: "types", Result: "pass"}, {AssetID: "controls.button", Target: "react-vite", Gate: "api", Result: "pass"}}, gates)
	if got := partial.Rows[0].Achieved; got != RungImplemented {
		t.Fatalf("partial evidence achieved %q, want implemented", got)
	}
}

func TestCoverageUsesNewestEvidenceRegardlessOfInputOrder(t *testing.T) {
	asset := Asset{ID: "controls.button", Name: "Button", Kind: "component", Maturity: "implemented", Targets: []string{"react-vite"}}
	impl := Implementation{Name: "Button", CatalogID: asset.ID, Latest: "1.0.0"}
	gates := []GateDefinition{{ID: "api", Rung: RungImplemented, Blocking: true, AppliesTo: []string{"component"}}}
	newerFailure := GateEvidence{AssetID: asset.ID, Target: "react-vite", Gate: "api", Version: "1.0.0", Result: "fail", RecordedAt: "2026-02-02T00:00:00Z"}
	olderPass := GateEvidence{AssetID: asset.ID, Target: "react-vite", Gate: "api", Version: "1.0.0", Result: "pass", RecordedAt: "2026-02-01T00:00:00Z"}
	for _, evidence := range [][]GateEvidence{{olderPass, newerFailure}, {newerFailure, olderPass}} {
		report := ComputeWithEvidence([]Asset{asset}, []Implementation{impl}, evidence, gates)
		row := report.Rows[0]
		if row.Achieved != RungScaffolded || row.AssetScore != 0 || len(row.FailedGates) != 1 {
			t.Fatalf("newest failure was not authoritative (order %v): %+v", evidence, row)
		}
		if report.PassEvidence != 0 || report.FailEvidence != 1 {
			t.Fatalf("evidence census should retain only the newest observation for a version: pass=%d fail=%d", report.PassEvidence, report.FailEvidence)
		}
	}
}

func TestGateTierRollupKeepsMeasuredPassesAndFailuresVisible(t *testing.T) {
	assets := []Asset{{ID: "controls.button", Name: "Button", Kind: "component", Domain: "controls", Priority: "P0", Maturity: "verified", Targets: []string{"react-vite"}}}
	impls := []Implementation{{Name: "Button", Root: "components", CatalogID: "controls.button"}}
	gates := []GateDefinition{
		{ID: "types", Tier: "release-integrity", Rung: RungScaffolded, Blocking: true, AppliesTo: []string{"component"}},
		{ID: "responsive", Tier: "asset-quality", Rung: RungScaffolded, Blocking: true, AppliesTo: []string{"component"}},
		{ID: "consumer-pin", Tier: "consumer-use", Rung: RungScaffolded, Blocking: true, AppliesTo: []string{"component"}},
	}
	evidence := []GateEvidence{
		{AssetID: "controls.button", Target: "react-vite", Gate: "types", Result: "pass"},
		{AssetID: "controls.button", Target: "react-vite", Gate: "responsive", Result: "fail"},
		{AssetID: "controls.button", Target: "react-vite", Gate: "consumer-pin", Result: "unmeasured"},
	}
	report := ComputeWithEvidence(assets, impls, evidence, gates)
	for tier, want := range map[string]ScoreBreakdown{
		"release-integrity": {Passed: 1, Applicable: 1, Score: 1},
		"asset-quality":     {Passed: 0, Applicable: 1, Score: 0},
		"consumer-use":      {Passed: 0, Applicable: 1, Score: 0},
	} {
		if got := report.ByTier[tier]; got != want {
			t.Errorf("tier %q = %+v, want %+v", tier, got, want)
		}
	}
}

func TestUnmeasuredEvidenceIsVisibleAndCannotRaiseScore(t *testing.T) {
	assets := []Asset{{ID: "controls.button", Name: "Button", Kind: "component", Maturity: "verified", Targets: []string{"react-vite"}}}
	impls := []Implementation{{Name: "Button", Root: "components", CatalogID: "controls.button"}}
	gates := []GateDefinition{{ID: "rtl", Rung: RungScaffolded, Blocking: true, AppliesTo: []string{"component"}}}
	report := ComputeWithEvidence(assets, impls, []GateEvidence{{AssetID: "controls.button", Target: "react-vite", Gate: "rtl", Result: "unmeasured"}}, gates)
	if report.Rows[0].AssetScore != 0 || report.Rows[0].Achieved != RungScaffolded {
		t.Fatalf("unmeasured evidence raised maturity: row=%+v", report.Rows[0])
	}
	if report.UnmeasuredEvidence != 1 || report.PassEvidence != 0 || report.FailEvidence != 0 {
		t.Fatalf("evidence census = pass:%d fail:%d unmeasured:%d", report.PassEvidence, report.FailEvidence, report.UnmeasuredEvidence)
	}
	if report.Maturity.MandatoryGateCoverage.Numerator != 0 || report.Maturity.MandatoryGateCoverage.Denominator != 1 {
		t.Fatalf("mandatory coverage = %+v", report.Maturity.MandatoryGateCoverage)
	}
}

func TestCoverageReachesTargetWhenEveryBlockingGatePasses(t *testing.T) {
	assets := []Asset{{ID: "controls.button", Name: "Button", Kind: "component", Domain: "controls", Priority: "P0", Maturity: "production-ready", Targets: []string{"react-vite"}}}
	impls := []Implementation{{Name: "Button", Root: "components", CatalogID: "controls.button"}}
	gates := []GateDefinition{
		{ID: "types", Rung: RungScaffolded, Blocking: true, AppliesTo: []string{"component"}},
		{ID: "api", Rung: RungImplemented, Blocking: true, AppliesTo: []string{"component"}},
		{ID: "visual", Rung: RungVerified, Blocking: true, AppliesTo: []string{"component"}},
		{ID: "stress", Rung: RungProductionReady, Blocking: true, AppliesTo: []string{"component"}},
	}
	evidence := []GateEvidence{
		{AssetID: "controls.button", Target: "react-vite", Gate: "types", Result: "pass"},
		{AssetID: "controls.button", Target: "react-vite", Gate: "api", Result: "pass"},
		{AssetID: "controls.button", Target: "react-vite", Gate: "visual", Result: "pass"},
		{AssetID: "controls.button", Target: "react-vite", Gate: "stress", Result: "pass"},
	}
	report := ComputeWithEvidence(assets, impls, evidence, gates)
	if got := report.Rows[0].Achieved; got != RungProductionReady {
		t.Fatalf("achieved = %q, want production-ready", got)
	}
}

func TestCoverageCanExceedDeclaredTargetWhenHigherRungGatesPass(t *testing.T) {
	assets := []Asset{{ID: "navigation.navigation-tree", Name: "NavigationTree", Kind: "component", Domain: "navigation", Priority: "P1", Maturity: "verified", Targets: []string{"react-vite"}}}
	impls := []Implementation{{Name: "NavigationTree", Root: "components", CatalogID: "navigation.navigation-tree"}}
	gates := []GateDefinition{
		{ID: "types", Rung: RungScaffolded, Blocking: true, AppliesTo: []string{"component"}},
		{ID: "api", Rung: RungImplemented, Blocking: true, AppliesTo: []string{"component"}},
		{ID: "visual", Rung: RungVerified, Blocking: true, AppliesTo: []string{"component"}},
		{ID: "stress", Rung: RungProductionReady, Blocking: true, AppliesTo: []string{"component"}},
	}
	evidence := []GateEvidence{
		{AssetID: "navigation.navigation-tree", Target: "react-vite", Gate: "types", Result: "pass"},
		{AssetID: "navigation.navigation-tree", Target: "react-vite", Gate: "api", Result: "pass"},
		{AssetID: "navigation.navigation-tree", Target: "react-vite", Gate: "visual", Result: "pass"},
		{AssetID: "navigation.navigation-tree", Target: "react-vite", Gate: "stress", Result: "pass"},
	}
	report := ComputeWithEvidence(assets, impls, evidence, gates)
	if got := report.Rows[0].Achieved; got != RungProductionReady {
		t.Fatalf("achieved = %q, want production-ready above declared verified target", got)
	}
}

func TestCoverageMetricsExposeAuditableIndependentDenominators(t *testing.T) {
	assets := []Asset{
		{ID: "controls.button", Name: "Button", Kind: "component", Domain: "controls", Priority: "P0", Maturity: "production-ready", Targets: []string{"react-vite"}},
		{ID: "controls.input", Name: "Input", Kind: "component", Domain: "controls", Priority: "P1", Maturity: "verified", Targets: []string{"react-vite"}},
	}
	impls := []Implementation{{Name: "Button", CatalogID: "controls.button"}}
	gates := []GateDefinition{{ID: "types", Rung: RungScaffolded, Blocking: true, AppliesTo: []string{"component"}}}
	report := ComputeWithEvidence(assets, impls, []GateEvidence{{AssetID: "controls.button", Target: "react-vite", Gate: "types", Result: "pass"}}, gates)
	if report.Maturity.CatalogCompletion.Numerator != 1 || report.Maturity.CatalogCompletion.Denominator != 2 {
		t.Fatalf("catalog completion = %#v", report.Maturity.CatalogCompletion)
	}
	if report.Maturity.MandatoryGateCoverage.Numerator != 1 || report.Maturity.MandatoryGateCoverage.Denominator != 2 {
		t.Fatalf("mandatory gate coverage = %#v", report.Maturity.MandatoryGateCoverage)
	}
	if report.Maturity.ProductionReadyCoverage.Denominator != 2 {
		t.Fatalf("production-ready denominator = %#v", report.Maturity.ProductionReadyCoverage)
	}
}

func TestWeightedAssetScoreIsPerAssetAndCorpusGatesDoNotInflateIt(t *testing.T) {
	assets := []Asset{
		{ID: "data-display.one", Name: "One", Kind: "component", Priority: "P0", Maturity: "verified", Targets: []string{"react-vite"}, PinnedWeight: 3},
		{ID: "data-display.two", Name: "Two", Kind: "component", Priority: "P1", Maturity: "verified", Targets: []string{"react-vite"}, PinnedWeight: 1},
	}
	impls := []Implementation{{Name: "One", CatalogID: "data-display.one"}, {Name: "Two", CatalogID: "data-display.two"}}
	gates := []GateDefinition{
		{ID: "types", Rung: RungScaffolded, Blocking: true, Attribution: "attributable", AppliesTo: []string{"component"}},
		{ID: "visual", Rung: RungVerified, Blocking: true, Attribution: "attributable", AppliesTo: []string{"component"}},
		{ID: "tokens", Rung: RungVerified, Blocking: true, Attribution: "corpus", AppliesTo: []string{"component"}},
	}
	evidence := []GateEvidence{
		{AssetID: "data-display.one", Target: "react-vite", Gate: "types", Result: "pass"},
		{AssetID: "data-display.one", Target: "react-vite", Gate: "visual", Result: "pass"},
		{AssetID: "data-display.two", Target: "react-vite", Gate: "types", Result: "pass"},
		{AssetID: "data-display.two", Target: "react-vite", Gate: "visual", Result: "fail"},
		{AssetID: "", Target: "", Gate: "tokens", Result: "pass"},
	}
	report := ComputeWithEvidence(assets, impls, evidence, gates)
	one, two := report.Rows[0], report.Rows[1]
	if one.AssetScore != 1 || two.AssetScore != 0.5 {
		t.Fatalf("asset scores = %v, %v; want 1 and .5", one.AssetScore, two.AssetScore)
	}
	if report.Score != 87.5 {
		t.Fatalf("weighted score = %v, want 87.5", report.Score)
	}
	if len(report.ByGate) != 2 {
		t.Fatalf("corpus gate appeared in attributable breakdown: %#v", report.ByGate)
	}
}

func TestWeightedScoreIsolationKeepsPassingAssetPassing(t *testing.T) {
	assets := []Asset{{ID: "a.one", Kind: "component", Maturity: "verified", Targets: []string{"react-vite"}}, {ID: "a.two", Kind: "component", Maturity: "verified", Targets: []string{"react-vite"}}}
	impls := []Implementation{{Name: "One", CatalogID: "a.one"}, {Name: "Two", CatalogID: "a.two"}}
	gates := []GateDefinition{{ID: "api", Rung: RungImplemented, Blocking: true, AppliesTo: []string{"component"}}}
	report := ComputeWithEvidence(assets, impls, []GateEvidence{{AssetID: "a.one", Target: "react-vite", Gate: "api", Result: "pass"}, {AssetID: "a.two", Target: "react-vite", Gate: "api", Result: "fail"}}, gates)
	if report.Rows[0].AssetScore != 1 || report.Rows[1].AssetScore != 0 {
		t.Fatalf("one asset failure leaked across rows: %#v", report.Rows)
	}
}

func TestMissingOrVacuousExperienceCannotEarnMaturity(t *testing.T) {
	asset := []Asset{{ID: "controls.button", Name: "Button", Kind: "component", Maturity: "verified", Targets: []string{"react-vite"}}}
	gates := []GateDefinition{{ID: "types", Rung: RungScaffolded, Blocking: true, AppliesTo: []string{"component"}}}
	evidence := []GateEvidence{{AssetID: "controls.button", Target: "react-vite", Gate: "types", Result: "pass"}}
	for _, impl := range []Implementation{
		{Name: "Button", CatalogID: "controls.button", ExperienceStateKnown: true},
		{Name: "Button", CatalogID: "controls.button", ExperienceStateKnown: true, ExperienceRegistered: true, ExperienceVacuous: true},
	} {
		got := ComputeWithEvidence(asset, []Implementation{impl}, evidence, gates).Rows[0].Achieved
		if got != RungScaffolded {
			t.Fatalf("experience state %#v earned %q, want scaffolded", impl, got)
		}
	}
}

func TestCoverageDropsOneRungWhenLastGateEvidenceIsRemoved(t *testing.T) {
	assets := []Asset{{ID: "controls.button", Name: "Button", Kind: "component", Maturity: "production-ready", Targets: []string{"react-vite"}}}
	impls := []Implementation{{Name: "Button", Root: "components", CatalogID: "controls.button"}}
	gates := []GateDefinition{
		{ID: "types", Rung: RungScaffolded, Blocking: true, AppliesTo: []string{"component"}},
		{ID: "api", Rung: RungImplemented, Blocking: true, AppliesTo: []string{"component"}},
		{ID: "visual", Rung: RungVerified, Blocking: true, AppliesTo: []string{"component"}},
		{ID: "stress", Rung: RungProductionReady, Blocking: true, AppliesTo: []string{"component"}},
	}
	all := []GateEvidence{{AssetID: "controls.button", Target: "react-vite", Gate: "types", Result: "pass"}, {AssetID: "controls.button", Target: "react-vite", Gate: "api", Result: "pass"}, {AssetID: "controls.button", Target: "react-vite", Gate: "visual", Result: "pass"}, {AssetID: "controls.button", Target: "react-vite", Gate: "stress", Result: "pass"}}
	withoutLast := all[:len(all)-1]
	if got := ComputeWithEvidence(assets, impls, withoutLast, gates).Rows[0].Achieved; got != RungVerified {
		t.Fatalf("achieved after removing final gate = %q, want verified", got)
	}
}

func TestSkippedEvidenceNeverRaisesMaturity(t *testing.T) {
	assets := []Asset{{ID: "controls.button", Name: "Button", Kind: "component", Maturity: "verified", Targets: []string{"react-vite"}}}
	impls := []Implementation{{Name: "Button", Root: "components", CatalogID: "controls.button"}}
	gates := []GateDefinition{{ID: "types", Rung: RungScaffolded, Blocking: true, AppliesTo: []string{"component"}}, {ID: "api", Rung: RungImplemented, Blocking: true, AppliesTo: []string{"component"}}}
	evidence := []GateEvidence{{AssetID: "controls.button", Target: "react-vite", Gate: "types", Result: "pass"}, {AssetID: "controls.button", Target: "react-vite", Gate: "api", Result: "skipped"}}
	if got := ComputeWithEvidence(assets, impls, evidence, gates).Rows[0].Achieved; got != RungScaffolded {
		t.Fatalf("achieved with skipped gate = %q, want scaffolded", got)
	}
}

func TestNextWorkPrefersBuiltMaturityGap(t *testing.T) {
	assets := []Asset{
		{ID: "foundation.tokens", Name: "Tokens", Kind: "foundation", Domain: "foundations", Priority: "P0", Maturity: "implemented", Targets: []string{"react-vite"}},
		{ID: "controls.button", Name: "Button", Kind: "component", Domain: "controls", Priority: "P1", Maturity: "production-ready", Targets: []string{"react-vite"}},
	}
	impls := []Implementation{{Name: "Button", Root: "components", CatalogID: "controls.button"}}
	rep := Compute(assets, impls)
	work := NextWork(rep, 1)
	if len(work) != 1 || work[0].AssetID != "controls.button" {
		t.Fatalf("next work = %+v, want built-but-below-target button", work)
	}
}

func TestDisplayNameNormalization(t *testing.T) {
	for _, test := range []struct{ raw, want string }{
		{`"Button"`, "Button"},
		{`{"default":"Button"}`, "Button"},
		{`{"fr":"Bouton","en":"Button"}`, "Button"},
		{`{"fr":"Bouton","de":"Taste"}`, "Taste"},
	} {
		var name DisplayName
		if err := json.Unmarshal([]byte(test.raw), &name); err != nil || string(name) != test.want {
			t.Errorf("%s: %q, %v", test.raw, name, err)
		}
	}
	for _, raw := range []string{`null`, `""`, `{}`, `[]`, `42`, `{"en":""}`, `{"en":42}`} {
		var name DisplayName
		if err := json.Unmarshal([]byte(raw), &name); err == nil {
			t.Errorf("accepted invalid name %s", raw)
		}
	}
}

func TestCatalogRejectsDuplicateIdentity(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "assets", "controls")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"one", "two"} {
		if err := os.WriteFile(filepath.Join(dir, name+".json"), []byte(`{"kind":"catalog-asset","asset":{"id":"controls.button","name":"Button","kind":"component"}}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := LoadCatalog(root); err == nil {
		t.Fatal("duplicate identity accepted")
	}
}
