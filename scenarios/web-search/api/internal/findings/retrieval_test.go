package findings_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"web-search/internal/findings"
)

func TestAddPreservesSourceRetrievalTime(t *testing.T) {
	repo, _ := newRepo(t)
	retrieved := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	f, err := repo.Add(context.Background(), findings.NewFinding{Claim: "historical claim", Query: "q", Source: findings.SourceL2, RetrievedAt: retrieved, Citations: []findings.NewCitation{{URL: "https://example.org/source", RetrievedAt: retrieved}}}, "agent")
	require.NoError(t, err)
	require.Equal(t, retrieved, f.RetrievalDate)
	require.Equal(t, retrieved, f.Citations[0].RetrievedAt)
}

func TestAddRollsBackFindingWhenCitationWriteFails(t *testing.T) {
	repo, db := newRepo(t)
	ctx := context.Background()
	_, err := db.ExecContext(ctx, `CREATE TRIGGER fail_finding_citation BEFORE INSERT ON finding_citations BEGIN SELECT RAISE(ABORT, 'forced citation failure'); END`)
	require.NoError(t, err)
	_, err = repo.Add(ctx, findings.NewFinding{Claim: "must rollback", Citations: []findings.NewCitation{{URL: "https://example.org"}}}, "agent")
	require.Error(t, err)
	var count int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM findings WHERE claim = ?`, "must rollback").Scan(&count))
	require.Zero(t, count)
}

func TestAddRollsBackFindingAndCitationsWhenAuditWriteFails(t *testing.T) {
	repo, db := newRepo(t)
	ctx := context.Background()
	_, err := db.ExecContext(ctx, `CREATE TRIGGER fail_finding_audit BEFORE INSERT ON finding_audit BEGIN SELECT RAISE(ABORT, 'forced audit failure'); END`)
	require.NoError(t, err)
	_, err = repo.Add(ctx, findings.NewFinding{Claim: "audit rollback", Citations: []findings.NewCitation{{URL: "https://example.org"}}}, "agent")
	require.Error(t, err)
	var findingsCount, citationsCount int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM findings WHERE claim = ?`, "audit rollback").Scan(&findingsCount))
	require.NoError(t, db.QueryRowContext(ctx, `SELECT COUNT(*) FROM finding_citations WHERE url = ?`, "https://example.org").Scan(&citationsCount))
	require.Zero(t, findingsCount)
	require.Zero(t, citationsCount)
}
