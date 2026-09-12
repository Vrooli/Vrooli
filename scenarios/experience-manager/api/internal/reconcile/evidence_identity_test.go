package reconcile

import (
	"context"
	"strings"
	"testing"

	"experience-manager/internal/spec"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	testdb "github.com/vrooli/api-core/databasetest"
)

func TestEvidenceIdentityBindsContractAndSnapshotWithoutInventingLegacyProvenance(t *testing.T) {
	snapshot := Snapshot{Contract: snapshotContract, Root: AXNode{Role: "button", Name: "Send"}}
	source := strings.Repeat("a", 64)
	raw := withEvidenceIdentity(`{"metric":"presence"}`, source, snapshot)
	identity := IdentityFromMeasurement(raw)
	require.Equal(t, source, identity.ContractHash)
	require.Equal(t, EvaluatorVersion, identity.EvaluatorVersion)
	require.Regexp(t, identityHash, identity.SnapshotHash)
	require.Equal(t, identity, IdentityFromMeasurement(withEvidenceIdentity(`{}`, source, snapshot)))
	snapshot.Root.Name = "Retry send"
	require.NotEqual(t, identity.SnapshotHash, IdentityFromMeasurement(withEvidenceIdentity(`{}`, source, snapshot)).SnapshotHash)
	require.Empty(t, IdentityFromMeasurement(`{"metric":"presence"}`).ContractHash)
	require.Empty(t, IdentityFromMeasurement(withEvidenceIdentity(`{}`, source, Snapshot{})).SnapshotHash)
	require.Empty(t, IdentityFromMeasurement(withEvidenceIdentity(`{}`, "", snapshot)).ContractHash)
}
func TestEvidenceIdentitiesSurvivePersistenceAndDoNotOverwriteChangedObservations(t *testing.T) {
	db := testdb.NewSQLite(t)
	ctx := context.Background()
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	a := Evidence{Scenario: "demo", PageID: "home", StateID: "default", ClaimID: "send", Verdict: "passed", CheckedAt: "2026-09-05T00:00:00Z", MeasurementJSON: withEvidenceIdentity(`{}`, strings.Repeat("a", 64), Snapshot{Contract: snapshotContract, Root: AXNode{Role: "button", Name: "Send"}})}
	a.ID = evidenceID(a)
	require.NoError(t, repo.SaveEvidence(ctx, a))
	b := a
	b.MeasurementJSON = withEvidenceIdentity(`{}`, strings.Repeat("b", 64), Snapshot{Contract: snapshotContract, Root: AXNode{Role: "button", Name: "Send"}})
	b.ID = evidenceID(b)
	require.NotEqual(t, a.ID, b.ID)
	require.NoError(t, repo.SaveEvidence(ctx, b))
	rows, err := repo.ListEvidence(ctx, EvidenceFilter{Scenario: "demo", PageID: "home"})
	require.NoError(t, err)
	require.Len(t, rows, 2)
	hashes := map[string]bool{}
	for _, r := range rows {
		hashes[IdentityFromMeasurement(r.MeasurementJSON).ContractHash] = true
	}
	require.True(t, hashes[strings.Repeat("a", 64)])
	require.True(t, hashes[strings.Repeat("b", 64)])
	changed := a
	changed.Verdict = "failed"
	require.Error(t, repo.SaveEvidence(ctx, changed), "changed facts must receive a different identity")
	_, err = db.ExecContext(ctx, "UPDATE reconcile_evidence SET verdict='failed' WHERE id=?", a.ID)
	require.NoError(t, err)
	_, err = repo.ListEvidence(ctx, EvidenceFilter{Scenario: "demo", PageID: "home"})
	require.Error(t, err, "tampered historical facts must not be returned as verified evidence")

}
func TestComponentReconciliationRetainsOriginalContractIdentity(t *testing.T) {
	component := spec.ComponentDocument{SourceHash: strings.Repeat("c", 64)}
	require.Equal(t, component.SourceHash, componentAsPage(component).SourceHash)
}

func TestReconciliationPersistsTheParsedSourceIdentity(t *testing.T) {
	db := testdb.NewSQLite(t)
	ctx := context.Background()
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)))
	repo := NewSQLiteRepository(db)
	check := Check{Repository: repo}
	page := spec.PageDocument{SourceHash: strings.Repeat("d", 64), Page: spec.PageIdentity{ID: "home"}}
	snapshot := Snapshot{Contract: snapshotContract, URL: "http://example.invalid/", Root: AXNode{Role: "button", Name: "Send"}}
	findings := check.persistEvidence(ctx, "demo", "experience/pages/home.json", page, CaptureTarget{Scenario: "demo", Route: "/", StateID: "default"}, snapshot, []Evidence{{ClaimID: "send", ClaimType: "element-present", Verdict: "passed"}})
	require.Empty(t, findings)
	rows, err := repo.ListEvidence(ctx, EvidenceFilter{Scenario: "demo", PageID: "home"})
	require.NoError(t, err)
	require.Len(t, rows, 1)
	identity := IdentityFromMeasurement(rows[0].MeasurementJSON)
	require.Equal(t, page.SourceHash, identity.ContractHash)
	require.NotEmpty(t, identity.SnapshotHash)
}

func TestEvidenceAndViewportArePublishedTogether(t *testing.T) {
	db := testdb.NewSQLite(t)
	ctx := context.Background()
	require.NoError(t, apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)))
	_, err := db.ExecContext(ctx, `CREATE TRIGGER reject_viewport BEFORE INSERT ON reconcile_evidence_viewports BEGIN SELECT RAISE(ABORT, 'fixture viewport failure'); END`)
	require.NoError(t, err)
	repo := NewSQLiteRepository(db)
	e := Evidence{ID: "test", Scenario: "demo", PageID: "home"}
	require.Error(t, repo.SaveEvidence(ctx, e))
	var count int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM reconcile_evidence WHERE id='test'").Scan(&count))
	require.Zero(t, count, "partial evidence escaped after viewport write failure")
}

// Build a measurement fixture through the same observation/envelope boundary.
func withEvidenceIdentity(raw, contractHash string, snapshot Snapshot) string {
	return withPreparedEvidenceIdentity(raw, evidenceIdentityFor(contractHash, snapshot))
}
