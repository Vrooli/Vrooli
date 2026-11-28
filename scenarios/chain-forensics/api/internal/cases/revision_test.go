package cases_test

import (
	"context"
	"testing"
	"time"

	"chain-forensics/internal/cases"
	"github.com/stretchr/testify/require"
	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/scheduletest"
	forensics "github.com/vrooli/chain-forensics"
)

func TestSQLiteCaseRevisionAppendsEvidenceAndRetainsHistory(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(ctx, cases.Schema())
	require.NoError(t, err)
	clock := scheduletest.New(time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC))
	repo := cases.NewSQLiteRepository(handle, clock)
	doc, err := forensics.ExampleCase()
	require.NoError(t, err)
	resolvedGap := "The complaint alleges 54 upstream Silk Road transfers totaling 70,411.46 BTC to the two named addresses. This case captures the later 54-input consolidation spend from those addresses to 1HQ3, but not all upstream transfer transactions."
	otherGap := "Attribution of the final exchange account remains unresolved."
	doc.Gaps = append(doc.Gaps, resolvedGap, otherGap)
	before, err := repo.Import(ctx, doc)
	require.NoError(t, err)
	addedSource := forensics.Source{ID: "revision-source", Name: "Public filing", URL: "https://example.org/filing", Retrieved: "2026-11-12T12:00:00Z", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Citation: "page 4"}
	claim := forensics.Claim{ID: "revision-claim", Text: "The filing identifies an exchange deposit address.", SourceID: addedSource.ID, Kind: forensics.EvidenceLegalClaim}
	addedOccurrence := doc.Occurrences[0]
	addedOccurrence.ID = "revision-occurrence"
	addedOccurrence.SourceID = addedSource.ID
	revisedEdge := forensics.Edge{ID: "revision-edge", From: addedOccurrence.ID, To: doc.Edges[0].To, Grade: forensics.GradeHypothesis, Rule: "source-qualified attribution hypothesis", EvidenceIDs: []string{claim.ID}, Caveats: []string{"Public label does not establish account control."}}
	additions := cases.RevisionAdditions{Summary: "The upstream transfers are now individually reconstructed.", Sources: []forensics.Source{addedSource}, Claims: []forensics.Claim{claim}, Occurrences: []forensics.Occurrence{addedOccurrence}, Edges: []forensics.Edge{revisedEdge}, Gaps: []string{"Exchange account control remains unresolved."}, ResolvedGaps: []string{resolvedGap}}
	updated, receipt, err := repo.Revise(ctx, doc.ID, before.UpdatedAt, additions)
	require.NoError(t, err)
	require.NotEqual(t, before.UpdatedAt, updated.UpdatedAt)
	require.Equal(t, additions.Summary, updated.Case.Summary)
	require.NotContains(t, updated.Case.Gaps, resolvedGap)
	require.Contains(t, updated.Case.Gaps, otherGap)
	require.Equal(t, doc.Sources, updated.Case.Sources[:len(doc.Sources)])
	require.Equal(t, doc.Occurrences, updated.Case.Occurrences[:len(doc.Occurrences)])
	require.Equal(t, addedOccurrence, updated.Case.Occurrences[len(doc.Occurrences)])
	require.Equal(t, doc.Edges, updated.Case.Edges[:len(doc.Edges)])
	require.Equal(t, revisedEdge, updated.Case.Edges[len(updated.Case.Edges)-1])
	require.Equal(t, before.UpdatedAt, receipt.PreviousUpdatedAt)
	require.Equal(t, doc, receipt.PreviousCase)
	revisions, err := repo.ListRevisions(ctx, doc.ID)
	require.NoError(t, err)
	require.Equal(t, []cases.CaseRevision{receipt}, revisions)
	require.Equal(t, doc.Summary, revisions[0].PreviousCase.Summary)
	require.Contains(t, revisions[0].PreviousCase.Gaps, resolvedGap)

	_, _, err = repo.Revise(ctx, doc.ID, before.UpdatedAt, additions)
	require.ErrorAs(t, err, new(cases.ErrStaleRevision))
	afterConflict, err := repo.Get(ctx, doc.ID)
	require.NoError(t, err)
	require.Equal(t, updated, afterConflict)
	revisionsAfterConflict, err := repo.ListRevisions(ctx, doc.ID)
	require.NoError(t, err)
	require.Equal(t, revisions, revisionsAfterConflict)
}

func TestSQLiteCaseRevisionRejectsInvalidEvidenceWithoutMutation(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(ctx, cases.Schema())
	require.NoError(t, err)
	repo := cases.NewSQLiteRepository(handle, scheduletest.New(time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)))
	doc, err := forensics.ExampleCase()
	require.NoError(t, err)
	before, err := repo.Import(ctx, doc)
	require.NoError(t, err)
	_, _, err = repo.Revise(ctx, doc.ID, before.UpdatedAt, cases.RevisionAdditions{Claims: []forensics.Claim{{ID: "dangling", Text: "unsupported", SourceID: "missing", Kind: forensics.EvidenceLegalClaim}}})
	require.ErrorAs(t, err, new(cases.ErrInvalid))
	_, _, err = repo.Revise(ctx, doc.ID, before.UpdatedAt, cases.RevisionAdditions{Summary: "must not apply", ResolvedGaps: []string{"near-match gap that is not present"}})
	require.ErrorAs(t, err, new(cases.ErrInvalid))
	after, err := repo.Get(ctx, doc.ID)
	require.NoError(t, err)
	require.Equal(t, before, after)
	revisions, err := repo.ListRevisions(ctx, doc.ID)
	require.NoError(t, err)
	require.Empty(t, revisions)
}

func TestSQLiteCaseRevisionExplicitlySupersedesGradeAndRetainsPriorSnapshot(t *testing.T) {
	ctx := context.Background()
	handle := db.NewSQLite(t)
	_, err := handle.ExecContext(ctx, cases.Schema())
	require.NoError(t, err)
	repo := cases.NewSQLiteRepository(handle, scheduletest.New(time.Date(2026, 11, 12, 12, 0, 0, 0, time.UTC)))
	doc, err := forensics.ExampleCase()
	require.NoError(t, err)
	before, err := repo.Import(ctx, doc)
	require.NoError(t, err)
	oldEdge := doc.Edges[0]
	replacement := oldEdge
	replacement.ID = "superseding-edge"
	replacement.Grade = forensics.GradeHypothesis
	replacement.Rule = "reassessed after provider provenance review"
	replacement.EvidenceIDs = []string{oldEdge.From}
	replacement.Caveats = append(append([]string(nil), oldEdge.Caveats...), "The original corroboration relied on an unverified provider attribution.")
	updated, receipt, err := repo.Revise(ctx, doc.ID, before.UpdatedAt, cases.RevisionAdditions{EdgeRevisions: []cases.EdgeRevision{{SupersedesEdgeID: oldEdge.ID, Replacement: replacement}}})
	require.NoError(t, err)
	require.NotContains(t, updated.Case.Edges, oldEdge)
	require.Contains(t, updated.Case.Edges, replacement)
	require.Contains(t, receipt.PreviousCase.Edges, oldEdge)
	revisions, err := repo.ListRevisions(ctx, doc.ID)
	require.NoError(t, err)
	require.Len(t, revisions, 1)
	require.Contains(t, revisions[0].PreviousCase.Edges, oldEdge)
	require.Equal(t, "superseding-edge", revisions[0].Additions.EdgeRevisions[0].Replacement.ID)

	_, _, err = repo.Revise(ctx, doc.ID, updated.UpdatedAt, cases.RevisionAdditions{EdgeRevisions: []cases.EdgeRevision{{SupersedesEdgeID: oldEdge.ID, Replacement: replacement}}})
	require.ErrorAs(t, err, new(cases.ErrInvalid))
	oldGradeReuse := replacement
	oldGradeReuse.ID = oldEdge.ID
	_, _, err = repo.Revise(ctx, doc.ID, updated.UpdatedAt, cases.RevisionAdditions{EdgeRevisions: []cases.EdgeRevision{{SupersedesEdgeID: replacement.ID, Replacement: oldGradeReuse}}})
	require.ErrorAs(t, err, new(cases.ErrInvalid))
}
