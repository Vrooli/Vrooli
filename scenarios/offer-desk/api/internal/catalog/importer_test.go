package catalog

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/databasetest"
	offerspb "github.com/vrooli/vrooli/packages/proto/gen/go/offer-desk/v1/offers"
)

func TestImportTreeReportsBrokenReferencesWithoutCopyingNarrative(t *testing.T) { // [REQ:MIG-001] [REQ:MIG-002] [REQ:MIG-003]
	root := filepath.Join(t.TempDir(), "testdata")
	require.NoError(t, os.MkdirAll(root, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "offer.md"), []byte("# Fixture offer\n\n**SKU ID:** `fixture-offer`\n**Status:** `candidate`\n\nThis narrative is intentionally excluded from the imported node.\n\nSee [missing deliverable](missing-deliverable.md).\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "notes.md"), []byte("# Notes\n\nNarrative-only fixture.\n"), 0o644))
	db := database.NewFromPrimary(databasetest.NewSQLite(t))
	s := NewStore(db, nil)
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(func() string { return s.Schema() })))
	report, err := s.ImportTree(context.Background(), root, "operator")
	require.NoError(t, err)
	require.Len(t, report.Files, 2)
	require.Equal(t, 1, report.Files[0].Read)
	require.Equal(t, 1, report.Files[0].Written)
	require.Equal(t, 1, report.Findings)
	var count int
	require.NoError(t, db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM migration_findings`).Scan(&count))
	require.Equal(t, 1, count)
	var narrative string
	require.Error(t, db.QueryRowContext(context.Background(), `SELECT name FROM nodes WHERE name LIKE '%narrative%'`).Scan(&narrative))
}

func TestImportTreeAppliedTwiceWritesOneNodeSet(t *testing.T) { // [REQ:MIG-001]
	root := filepath.Join(t.TempDir(), "testdata")
	require.NoError(t, os.MkdirAll(root, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "offer.md"), []byte("# Fixture offer\n**SKU ID:** `fixture-offer`\n**Status:** active\n"), 0o644))
	s, ctx := testCatalog(t)
	first, err := s.ImportTree(ctx, root, "operator")
	require.NoError(t, err)
	second, err := s.ImportTree(ctx, root, "operator")
	require.NoError(t, err)
	require.Equal(t, 1, first.Files[0].Written)
	require.Zero(t, second.Files[0].Written)
	var nodes int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM nodes`).Scan(&nodes))
	require.Equal(t, 1, nodes)
}

func TestImportTreeRejectsLiveOrUnscopedRoots(t *testing.T) {
	db := database.NewFromPrimary(databasetest.NewSQLite(t))
	s := NewStore(db, nil)
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(func() string { return s.Schema() })))
	_, err := s.ImportTree(context.Background(), t.TempDir(), "operator")
	require.ErrorContains(t, err, "fixture root")
}

func TestOperatorImportResolvesRepositoryRelativeSource(t *testing.T) {
	root, err := resolveImportRoot("docs/monetization", offerspb.SourceMode_SOURCE_MODE_OPERATOR_SUPPLIED)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(root, "catalogs", "CATALOG.md"))
}

func TestImportedStatusRecognizesCorpusShapes(t *testing.T) { // [REQ:MIG-001]
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{name: "list item", body: "- **Status:** active (currently sold)\n", want: "ACTIVE"},
		{name: "inline bold", body: "**Status: candidate. Revisit when triggered.**\n", want: "CANDIDATE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := importedStatus(tc.body)
			require.Equal(t, tc.want, got.String())
		})
	}
}

func TestUnrecognizedStatusIsNotSilentlyIdea(t *testing.T) { // [REQ:MIG-002]
	got := importedStatus("**Status:** maybe\n")
	require.NotEqual(t, "IDEA", got.String())
}

func TestOperatorCatalogImportReportsBrokenReferencesWithoutCopyingNarrative(t *testing.T) { // [REQ:MIG-001] [REQ:MIG-002] [REQ:MIG-003]
	source := filepath.Join("..", "..", "..", "..", "..", "docs", "monetization")
	root := filepath.Join(t.TempDir(), "monetization")
	require.NoError(t, filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		require.NoError(t, err)
		rel, relErr := filepath.Rel(source, path)
		require.NoError(t, relErr)
		destination := filepath.Join(root, rel)
		if info.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		require.NoError(t, os.MkdirAll(filepath.Dir(destination), 0o755))
		body, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		if rel == filepath.Join("catalogs", "skus", "base", "business.md") {
			body = append(body, []byte("\nSee [missing source](missing-source.md).\n")...)
		}
		return os.WriteFile(destination, body, info.Mode().Perm())
	}))

	s, ctx := testCatalog(t)
	report, err := s.ImportCatalog(ctx, root, offerspb.SourceMode_SOURCE_MODE_OPERATOR_SUPPLIED, false, "operator:test")
	require.NoError(t, err)
	require.False(t, report.Applied)
	require.NotEmpty(t, report.Findings)
	found := false
	for _, finding := range report.Findings {
		if finding.Path == filepath.ToSlash(filepath.Join("catalogs", "skus", "base", "business.md")) && strings.Contains(finding.Reason, "missing-source.md") {
			found = true
			require.False(t, finding.Blocking)
			require.Positive(t, finding.Line)
		}
	}
	require.True(t, found)
	var nodes int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM nodes`).Scan(&nodes))
	require.Zero(t, nodes)
}

func TestPricingMatrixCarriesDeclaredCurrencyAndDistinguishesAbsentZero(t *testing.T) { // [REQ:MIG-001]
	rows, findings := parsePricingRows(strings.TrimSpace(`
| SKU \ Tier | Tier 1 | Tier 2 |
| --- | --- | --- |
| Business bundle | USD 29.00/mo | TBD |
| Lifestyle bundle | USD 0.00/mo | — |
`))
	require.Empty(t, findings)
	byKey := make(map[string]pricingCell, len(rows))
	for _, row := range rows {
		byKey[row.offer+"/"+row.variant] = row
	}

	businessTier1 := byKey["business/tier-1"]
	require.True(t, businessTier1.declared)
	require.EqualValues(t, 2900, businessTier1.priceMinor)
	require.Equal(t, "USD", businessTier1.currency)

	businessTier2 := byKey["business/tier-2"]
	require.False(t, businessTier2.declared)
	require.Zero(t, businessTier2.priceMinor)
	require.Empty(t, businessTier2.currency)

	lifestyleTier1 := byKey["lifestyle/tier-1"]
	require.True(t, lifestyleTier1.declared)
	require.Zero(t, lifestyleTier1.priceMinor)
	require.Equal(t, "USD", lifestyleTier1.currency)
}

func TestPricingMatrixRejectsDeclaredAmountWithoutCurrency(t *testing.T) { // [REQ:MIG-001]
	_, findings := parsePricingRows(strings.TrimSpace(`
| SKU \ Tier | Tier 1 |
| --- | --- |
| Business bundle | 29.00/mo |
`))
	require.Len(t, findings, 1)
	require.True(t, findings[0].Blocking)
	require.Contains(t, findings[0].Reason, "currency")
}

func TestBenchmarkRowsBecomeDimensionDatedFacts(t *testing.T) { // [REQ:GATE-008]
	facts, findings := parseBenchmarkRows(strings.TrimSpace(`
| Comp | Category | Relevant dimension | Value | Source | Date captured | Applicability |
| --- | --- | --- | --- | --- | --- | --- |
| Acme | dev tools | pricing | 29.00 | public pricing page | 2026-01-15 | high |
`))
	require.Empty(t, findings)
	require.Len(t, facts, 1)
	require.Equal(t, "benchmark:acme:pricing", facts[0].Name)
	require.Equal(t, 29.0, facts[0].Value)
	require.Equal(t, "pricing", facts[0].Dimension)
	require.EqualValues(t, 90, facts[0].StaleAfterDays)
	require.Equal(t, "2026-01-15", facts[0].ObservedAt.AsTime().Format("2006-01-02"))
}

func TestImportedStaleBenchmarkRemainsUnknown(t *testing.T) { // [REQ:GATE-003] [REQ:GATE-008]
	s, ctx := testCatalog(t)
	facts, findings := parseBenchmarkRows(strings.TrimSpace(`
| Comp | Category | Relevant dimension | Value | Source | Date captured | Applicability |
| --- | --- | --- | --- | --- | --- | --- |
| Acme | dev tools | pricing | 29.00 | public pricing page | 2025-01-15 | high |
`))
	require.Empty(t, findings)
	require.Len(t, facts, 1)
	node, err := s.CreateNode(ctx, offerspb.NodeKind_OFFER, "Benchmark-gated offer", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	_, err = s.AddTrigger(ctx, &offerspb.Trigger{NodeId: node.Id, FactName: facts[0].Name, Operator: ">=", Threshold: 20})
	require.NoError(t, err)
	_, err = s.Transition(ctx, node.Id, offerspb.Status_CANDIDATE, "operator")
	require.NoError(t, err)
	_, err = s.AddFact(ctx, facts[0])
	require.NoError(t, err)
	evals, err := s.Evaluate(ctx, false)
	require.NoError(t, err)
	require.Len(t, evals, 1)
	require.Equal(t, offerspb.Verdict_UNKNOWN, evals[0].Verdict)
	require.Contains(t, evals[0].Explanation, "stale")
	nodes, err := s.ListNodes(ctx, offerspb.NodeKind_NODE_KIND_UNSPECIFIED, offerspb.Status_STATUS_UNSPECIFIED)
	require.NoError(t, err)
	require.Equal(t, offerspb.Status_CANDIDATE, nodes[0].Status)
}

func TestCatalogImportPersistsPricingPresenceAndBenchmarkFacts(t *testing.T) { // [REQ:MIG-001] [REQ:GATE-008]
	source := filepath.Join("..", "..", "..", "..", "..", "docs", "monetization")
	root := filepath.Join(t.TempDir(), "monetization")
	require.NoError(t, filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		require.NoError(t, err)
		rel, relErr := filepath.Rel(source, path)
		require.NoError(t, relErr)
		destination := filepath.Join(root, rel)
		if info.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		require.NoError(t, os.MkdirAll(filepath.Dir(destination), 0o755))
		body, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		return os.WriteFile(destination, body, info.Mode().Perm())
	}))
	require.NoError(t, os.WriteFile(filepath.Join(root, "strategy", "PRICING.md"), []byte(`
| SKU \ Tier | Tier 1 | Tier 2 | Tier 3 | Tier 4 |
| --- | --- | --- | --- | --- |
| Business bundle | USD 29.00/mo | TBD | TBD | — |
| Lifestyle bundle | USD 0.00/mo | TBD | TBD | — |
| Property services (add-on) | TBD | TBD | TBD | — |
| Elder care (add-on) | TBD | TBD | TBD | — |
| Family with kids (add-on) | TBD | TBD | TBD | — |
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "strategy", "TIERS.md"), []byte(`
### Tier 1 — Bundle apps (active)
### Tier 2 — Self-hosted full Vrooli runtime (candidate)
### Tier 3 — Hosted cloud Vrooli (candidate)
### Tier 4 — Hardware appliance (north-star)
`), 0o644))
	for _, fixtureStatus := range []struct {
		path   string
		status string
	}{
		{path: "catalogs/skus/base/business.md", status: "active"},
		{path: "catalogs/skus/base/lifestyle.md", status: "candidate"},
	} {
		path := filepath.Join(root, fixtureStatus.path)
		body, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		bodyText := string(body)
		bodyText = strings.Replace(bodyText, "**SKU ID:** `business`\n", "**SKU ID:** `business`\n**Status:** `"+fixtureStatus.status+"`\n", 1)
		bodyText = strings.Replace(bodyText, "**SKU ID:** `lifestyle`\n", "**SKU ID:** `lifestyle`\n**Status:** `"+fixtureStatus.status+"`\n", 1)
		require.NoError(t, os.WriteFile(path, []byte(bodyText), 0o644))
	}
	require.NoError(t, os.WriteFile(filepath.Join(root, "evidence", "BENCHMARKS.md"), []byte(`
| Comp | Category | Relevant dimension | Value | Source | Date captured | Applicability |
| --- | --- | --- | --- | --- | --- | --- |
| Acme | dev tools | pricing | 29.00 | public pricing page | 2026-01-15 | high |
`), 0o644))

	s, ctx := testCatalog(t)
	report, err := s.ImportCatalog(ctx, root, offerspb.SourceMode_SOURCE_MODE_OPERATOR_SUPPLIED, true, "operator:test")
	require.NoError(t, err)
	require.True(t, report.Applied)
	var sellsAt, declaredPrices, declaredZero, facts int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM edges WHERE kind='sells_at'`).Scan(&sellsAt))
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM edges WHERE kind='sells_at' AND intended_price_declared=1`).Scan(&declaredPrices))
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM edges WHERE kind='sells_at' AND intended_price_declared=1 AND intended_price_minor=0`).Scan(&declaredZero))
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM facts WHERE name='benchmark:acme:pricing'`).Scan(&facts))
	require.Equal(t, 8, sellsAt)
	require.Equal(t, 2, declaredPrices)
	require.Equal(t, 1, declaredZero)
	require.Equal(t, 1, facts)
}

func TestImportCatalogAppliedTwiceWritesOneNodeSet(t *testing.T) { // [REQ:MIG-001]
	root := filepath.Join(t.TempDir(), "monetization")
	source := filepath.Join("..", "..", "..", "..", "..", "docs", "monetization")
	require.NoError(t, filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		require.NoError(t, err)
		rel, relErr := filepath.Rel(source, path)
		require.NoError(t, relErr)
		destination := filepath.Join(root, rel)
		if info.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		require.NoError(t, os.MkdirAll(filepath.Dir(destination), 0o755))
		body, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		return os.WriteFile(destination, body, info.Mode().Perm())
	}))
	require.NoError(t, filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || filepath.Ext(path) != ".md" {
			return err
		}
		body, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if !strings.Contains(string(body), "**Status:") {
			body = append(body, []byte("\n**Status:** active\n")...)
		}
		return os.WriteFile(path, body, info.Mode().Perm())
	}))
	s, ctx := testCatalog(t)
	first, err := s.ImportCatalog(ctx, root, offerspb.SourceMode_SOURCE_MODE_OPERATOR_SUPPLIED, true, "operator:test")
	require.NoError(t, err)
	second, err := s.ImportCatalog(ctx, root, offerspb.SourceMode_SOURCE_MODE_OPERATOR_SUPPLIED, true, "operator:test")
	require.NoError(t, err)
	require.True(t, first.Applied)
	require.True(t, second.Applied)
	var nodes int
	require.NoError(t, s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM nodes`).Scan(&nodes))
	var blocking int
	for _, finding := range second.Findings {
		if finding.Blocking {
			blocking++
		}
	}
	require.Zero(t, blocking)
	for _, file := range second.Files {
		require.Zero(t, file.Written, file.Path)
	}
	require.NotZero(t, nodes)
}

func TestOperatorImportAcceptsRetiredProseStateHandoff(t *testing.T) { // [REQ:MIG-003]
	source := filepath.Join("..", "..", "..", "..", "..", "docs", "monetization")
	root := filepath.Join(t.TempDir(), "monetization")
	require.NoError(t, filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		require.NoError(t, err)
		rel, relErr := filepath.Rel(source, path)
		require.NoError(t, relErr)
		destination := filepath.Join(root, rel)
		if info.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		require.NoError(t, os.MkdirAll(filepath.Dir(destination), 0o755))
		body, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		return os.WriteFile(destination, body, info.Mode().Perm())
	}))

	s, ctx := testCatalog(t)
	report, err := s.ImportCatalog(ctx, root, offerspb.SourceMode_SOURCE_MODE_OPERATOR_SUPPLIED, false, "operator:retired-prose")
	require.NoError(t, err)
	for _, finding := range report.Findings {
		require.False(t, finding.Blocking, "%s: %s", finding.Path, finding.Reason)
		require.NotContains(t, finding.Reason, "lifecycle status marker is absent")
		require.NotContains(t, finding.Reason, "pricing matrix is missing the supported edge row")
	}
	for _, file := range report.Files {
		if file.Path == "strategy/PRICING.md" || file.Path == "strategy/TIERS.md" {
			require.Zero(t, file.Read, file.Path)
			require.Zero(t, file.Written, file.Path)
		}
	}
}

func verificationSource(t *testing.T) string {
	t.Helper()
	source := filepath.Join("..", "..", "..", "..", "..", "docs", "monetization")
	root := filepath.Join(t.TempDir(), "monetization")
	require.NoError(t, filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		require.NoError(t, err)
		rel, relErr := filepath.Rel(source, path)
		require.NoError(t, relErr)
		destination := filepath.Join(root, rel)
		if info.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		require.NoError(t, os.MkdirAll(filepath.Dir(destination), 0o755))
		body, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		if filepath.Ext(path) == ".md" && !strings.Contains(string(body), "**Status:") {
			body = append(body, []byte("\n**Status:** active\n")...)
		}
		return os.WriteFile(destination, body, info.Mode().Perm())
	}))
	return root
}

func TestCatalogVerifyReportsDriftWhenLiveCountExceedsSource(t *testing.T) { // [REQ:MIG-004]
	root := verificationSource(t)
	s, ctx := testCatalog(t)
	_, err := s.ImportCatalog(ctx, root, offerspb.SourceMode_SOURCE_MODE_OPERATOR_SUPPLIED, true, "operator:test")
	require.NoError(t, err)
	_, err = s.CreateNode(ctx, offerspb.NodeKind_OFFER, "untracked-extra", offerspb.Status_IDEA, "", "")
	require.NoError(t, err)
	report, err := s.VerifyCatalog(ctx, root, offerspb.SourceMode_SOURCE_MODE_OPERATOR_SUPPLIED)
	require.NoError(t, err)
	require.False(t, report.Reconciled)
	require.Positive(t, report.TotalDrift)
	require.NotEmpty(t, report.ExtraNodeIds)
}

func TestCatalogVerifyPassesOnAReconciledCatalog(t *testing.T) { // [REQ:MIG-004]
	root := verificationSource(t)
	s, ctx := testCatalog(t)
	_, err := s.ImportCatalog(ctx, root, offerspb.SourceMode_SOURCE_MODE_OPERATOR_SUPPLIED, true, "operator:test")
	require.NoError(t, err)
	report, err := s.VerifyCatalog(ctx, root, offerspb.SourceMode_SOURCE_MODE_OPERATOR_SUPPLIED)
	require.NoError(t, err)
	require.True(t, report.Reconciled)
	require.Zero(t, report.TotalDrift)
	require.Empty(t, report.DuplicateIdentities)
	require.Empty(t, report.OrphanEdgeIds)
}

func TestCatalogVerifyDetectsAnOrphanEdge(t *testing.T) { // [REQ:MIG-004]
	root := verificationSource(t)
	s, ctx := testCatalog(t)
	_, err := s.ImportCatalog(ctx, root, offerspb.SourceMode_SOURCE_MODE_OPERATOR_SUPPLIED, true, "operator:test")
	require.NoError(t, err)
	const orphanID = "orphan-edge-for-verify"
	_, err = s.db.ExecContext(ctx, `INSERT INTO edges(id,from_id,to_id,kind,intended_price_minor,currency,intended_price_declared) VALUES(?,?,?,?,?,?,?)`, orphanID, "missing-from", "missing-to", "feeds", 0, "", 0)
	require.NoError(t, err)
	report, err := s.VerifyCatalog(ctx, root, offerspb.SourceMode_SOURCE_MODE_OPERATOR_SUPPLIED)
	require.NoError(t, err)
	require.False(t, report.Reconciled)
	require.Contains(t, report.OrphanEdgeIds, orphanID)
}

// TestCatalogVerifyCountsBlockingFindingsFromAnyPath pins the wrong-root case.
//
// Pointing the verifier at a subdirectory of the declared root raises a blocking
// "declared source file is missing" finding for every manifest path plus an
// "undeclared source file" finding for everything it does find. None of those
// paths appears in the per-file report, so the old drift loop — which only
// counted a blocking finding whose path matched a file already in the report —
// dropped all of them and the wrong root reported reconciled=true.
func TestCatalogVerifyCountsBlockingFindingsFromAnyPath(t *testing.T) { // [REQ:MIG-004]
	root := verificationSource(t)
	s, ctx := testCatalog(t)
	_, err := s.ImportCatalog(ctx, root, offerspb.SourceMode_SOURCE_MODE_OPERATOR_SUPPLIED, true, "operator:test")
	require.NoError(t, err)

	wrongRoot := filepath.Join(root, "catalogs")
	report, err := s.VerifyCatalog(ctx, wrongRoot, offerspb.SourceMode_SOURCE_MODE_OPERATOR_SUPPLIED)
	require.NoError(t, err)
	require.False(t, report.Reconciled, "a root that resolves none of the declared source files must not reconcile")
	require.Positive(t, report.TotalDrift, "blocking findings must count toward drift regardless of path")
	require.False(t, report.Comparable)
	// A wrong root and a compressed source both skip the comparison for opposite
	// reasons, so they must not share a message: reassuring text here is how a
	// mis-aimed verification gets read as a pass.
	require.Contains(t, report.NotComparableReason, "check --source-path")
	require.NotContains(t, report.NotComparableReason, "this is expected")
}

// TestCatalogVerifyReportsNotComparableWhenSourcesAreCompressed pins the state
// the monetization cutover actually left behind.
//
// Once sources are compressed to judgment-only prose and their records move into
// this scenario, a dry-run import yields no countable record, so the
// source-versus-live count comparison has nothing to compare. That is the
// correct steady state — but before Comparable existed it was reported as a bare
// reconciled=true, indistinguishable from an import that never ran.
func TestCatalogVerifyReportsNotComparableWhenSourcesAreCompressed(t *testing.T) { // [REQ:MIG-004]
	root := verificationSource(t)
	s, ctx := testCatalog(t)
	_, err := s.ImportCatalog(ctx, root, offerspb.SourceMode_SOURCE_MODE_OPERATOR_SUPPLIED, true, "operator:test")
	require.NoError(t, err)

	full, err := s.VerifyCatalog(ctx, root, offerspb.SourceMode_SOURCE_MODE_OPERATOR_SUPPLIED)
	require.NoError(t, err)
	require.True(t, full.Comparable, "a source declaring records must be comparable")
	require.Empty(t, full.NotComparableReason)

	// Compress every declared source the way the retirement disposition did:
	// strip the state marker the parser keys on, keep the prose.
	require.NoError(t, filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		require.NoError(t, walkErr)
		if info.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		body, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		return os.WriteFile(path, []byte(strings.ReplaceAll(string(body), "**Status:", "**Retired-status:")), info.Mode().Perm())
	}))

	compressed, err := s.VerifyCatalog(ctx, root, offerspb.SourceMode_SOURCE_MODE_OPERATOR_SUPPLIED)
	require.NoError(t, err)
	require.False(t, compressed.Comparable, "a compressed source declares no records and cannot be compared")
	require.NotEmpty(t, compressed.NotComparableReason, "the skipped comparison must state why")
}
