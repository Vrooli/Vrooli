package catalogcoverage

import (
	"os"
	"path/filepath"
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
	if len(assets) < 400 {
		t.Fatalf("expected the full catalog, got %d assets", len(assets))
	}
	for _, a := range assets {
		if a.ID == "" || a.Domain == "" || a.Priority == "" || a.Delivery == "" {
			t.Fatalf("asset %+v is missing required identity fields", a)
		}
	}
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
