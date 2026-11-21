package review

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	db "github.com/vrooli/api-core/databasetest"
)

func newReviewDB(t *testing.T) *sql.DB {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	_, err := sqlDB.ExecContext(context.Background(), Schema())
	require.NoError(t, err)
	return sqlDB
}

func newProjectorDB(t *testing.T) *sql.DB {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	_, err := sqlDB.ExecContext(context.Background(), `
CREATE TABLE catalog_gate_evidence (
  id               TEXT PRIMARY KEY,
  asset_id         TEXT NOT NULL,
  target           TEXT NOT NULL,
  gate             TEXT NOT NULL,
  version          TEXT NOT NULL DEFAULT '',
  result           TEXT NOT NULL,
  measurement_json TEXT NOT NULL DEFAULT '',
  source_revision  TEXT NOT NULL,
  rule_set_digest  TEXT NOT NULL DEFAULT '',
  recorded_at      TEXT NOT NULL
)`)
	require.NoError(t, err)
	return sqlDB
}

func seedEvidence(t *testing.T, sqlDB *sql.DB, assetID, target, gate, version, result, measurement, recordedAt string) {
	t.Helper()
	_, err := sqlDB.ExecContext(context.Background(), `
INSERT INTO catalog_gate_evidence(id, asset_id, target, gate, version, result, measurement_json, source_revision, recorded_at)
VALUES (?, ?, ?, ?, ?, ?, ?, 'rev-1', ?)`,
		assetID+"|"+gate+"|"+version+"|"+recordedAt, assetID, target, gate, version, result, measurement, recordedAt)
	require.NoError(t, err)
}

func TestEvidenceProjectorBuildsFindings(t *testing.T) {
	sqlDB := newProjectorDB(t)
	ctx := context.Background()
	seedEvidence(t, sqlDB, "button", "react-vite", "accessibility", "1.0.0", "fail", `{"score":0.42}`, "2026-11-18T09:00:00Z")
	// An older observation for the same identity must collapse into one item.
	seedEvidence(t, sqlDB, "button", "react-vite", "accessibility", "1.0.0", "fail", `{"score":0.30}`, "2026-11-17T09:00:00Z")
	seedEvidence(t, sqlDB, "card", "react-vite", "composition", "2.0.0", "unmeasured", "", "2026-11-18T10:00:00Z")
	seedEvidence(t, sqlDB, "nav", "react-vite", "utility-class", "1.0.0", "error", "not-json", "2026-11-18T11:00:00Z")
	seedEvidence(t, sqlDB, "types", "react-vite", "types", "1.0.0", "PASS", "", "2026-11-18T12:00:00Z")

	projector := NewEvidenceProjector(sqlDB)
	projector.clock = func() time.Time { return time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC) }
	findings, err := projector.Project(ctx)
	require.NoError(t, err)
	require.Len(t, findings, 3, "passing evidence is excluded and repeated failures collapse")

	bySubject := map[string]Finding{}
	for _, finding := range findings {
		bySubject[finding.SubjectID] = finding
	}

	button := bySubject["button"]
	require.Equal(t, findingID("asset", "button", "accessibility", "1.0.0"), button.ID)
	require.Equal(t, "asset", button.SubjectKind)
	require.Equal(t, "accessibility", button.Gate)
	require.Equal(t, SeverityBlocker, button.Severity)
	require.Equal(t, StatusOpen, button.Status)
	require.Equal(t, "docs/concepts/STORY-CONTRACT.md", button.DocsRef)
	require.Contains(t, button.Message, "accessibility")
	require.Contains(t, button.Message, "1.0.0")
	require.Contains(t, button.Message, "fail")
	require.Contains(t, button.CreatedAt, "2026-11-18")
	require.Equal(t, "2026-11-18T09:00:00.000000000Z", button.CreatedAt)

	var evidence map[string]any
	require.NoError(t, json.Unmarshal([]byte(button.EvidenceJSON), &evidence))
	require.Equal(t, "react-vite", evidence["target"])
	require.Equal(t, "1.0.0", evidence["version"])
	require.Equal(t, "2026-11-18T09:00:00Z", evidence["recorded_at"])
	require.Equal(t, 0.42, evidence["measurement"].(map[string]any)["score"])

	require.Equal(t, SeverityHigh, bySubject["card"].Severity)
	require.Equal(t, SeverityMedium, bySubject["nav"].Severity)
	// A non-JSON measurement remains visible rather than vanishing.
	var navEvidence map[string]any
	require.NoError(t, json.Unmarshal([]byte(bySubject["nav"].EvidenceJSON), &navEvidence))
	require.Equal(t, "not-json", navEvidence["measurement"])

	again, err := projector.Project(ctx)
	require.NoError(t, err)
	require.Equal(t, findings, again, "projection ids are deterministic")
}

func TestEvidenceProjectorMissingTableIsEmpty(t *testing.T) {
	sqlDB := db.NewSQLite(t)
	projector := NewEvidenceProjector(sqlDB)
	findings, err := projector.Project(context.Background())
	require.NoError(t, err)
	require.Empty(t, findings)

	service := NewService(newReviewDB(t), projector)
	service.WithTTL(0)
	page, err := service.List(context.Background(), Filter{}, 10, 0)
	require.NoError(t, err)
	require.Empty(t, page.Findings)
	require.Zero(t, page.Total)
}

type fakeProjector struct {
	findings []Finding
	err      error
	calls    int
}

func (f *fakeProjector) Project(context.Context) ([]Finding, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return append([]Finding(nil), f.findings...), nil
}

func newTestService(t *testing.T, projector Projector) (*Service, *fakeProjector) {
	t.Helper()
	sqlDB := newReviewDB(t)
	fake, _ := projector.(*fakeProjector)
	service := NewService(sqlDB, projector)
	tick := 0
	service.WithClock(func() time.Time {
		tick++
		return time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC).Add(time.Duration(tick) * time.Second)
	})
	service.WithTTL(0)
	service.random = bytes.NewReader(bytes.Repeat([]byte{0x1d, 0x2e, 0x3f, 0x4a}, 128))
	return service, fake
}

func projectedFinding(id, severity string, createdAt time.Time) Finding {
	return Finding{
		ID:           id,
		SubjectKind:  "asset",
		SubjectID:    "asset-" + id,
		Gate:         "accessibility",
		Severity:     severity,
		Status:       StatusOpen,
		Message:      "gate failed",
		Remediation:  "fix it",
		EvidenceJSON: `{"target":"react-vite"}`,
		DocsRef:      "docs/concepts/STORY-CONTRACT.md",
		CreatedAt:    createdAt.Format(time.RFC3339),
	}
}

func TestRefreshPreservesStatusOwnerAndDispositions(t *testing.T) {
	base := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)
	projector := &fakeProjector{findings: []Finding{projectedFinding("finding-1", SeverityBlocker, base)}}
	service, fake := newTestService(t, projector)
	ctx := context.Background()

	page, err := service.List(ctx, Filter{}, 10, 0)
	require.NoError(t, err)
	require.Len(t, page.Findings, 1)
	require.Equal(t, 1, page.Total)
	require.Equal(t, 1, fake.calls)

	triaged, err := service.RecordDisposition(ctx, "finding-1", DecisionAcknowledge, "known issue", "ada")
	require.NoError(t, err)
	require.Equal(t, StatusAcknowledged, triaged.Status)
	require.Len(t, triaged.Dispositions, 1)

	assigned, err := service.Assign(ctx, "finding-1", "lin")
	require.NoError(t, err)
	require.Equal(t, "lin", assigned.Owner)

	// A gate rerun updates the projection-owned message but must not erase the
	// acknowledgement, the owner, or the disposition history.
	fake.findings[0].Message = "gate failed again"
	fake.findings[0].Severity = SeverityHigh
	page, err = service.List(ctx, Filter{}, 10, 0)
	require.NoError(t, err)
	require.Len(t, page.Findings, 1)
	require.Equal(t, "gate failed again", page.Findings[0].Message)
	require.Equal(t, SeverityHigh, page.Findings[0].Severity)
	require.Equal(t, StatusAcknowledged, page.Findings[0].Status)
	require.Equal(t, "lin", page.Findings[0].Owner)
	require.Len(t, page.Findings[0].Dispositions, 1)
	require.Equal(t, DecisionAcknowledge, page.Findings[0].Dispositions[0].Decision)
}

func TestDispositionTransitions(t *testing.T) {
	base := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)
	projector := &fakeProjector{findings: []Finding{projectedFinding("finding-1", SeverityBlocker, base)}}
	service, _ := newTestService(t, projector)
	ctx := context.Background()
	_, err := service.List(ctx, Filter{}, 10, 0)
	require.NoError(t, err)

	steps := []struct {
		decision string
		status   string
	}{
		{DecisionAcknowledge, StatusAcknowledged},
		{DecisionFix, StatusOpen},
		{DecisionAcceptRisk, StatusAccepted},
		{DecisionRetire, StatusRetired},
	}
	for _, step := range steps {
		finding, err := service.RecordDisposition(ctx, "finding-1", step.decision, "reason for "+step.decision, "ada")
		require.NoError(t, err)
		require.Equal(t, step.status, finding.Status)
	}

	finding, err := service.Get(ctx, "finding-1")
	require.NoError(t, err)
	require.Len(t, finding.Dispositions, 4)
	require.Equal(t, DecisionRetire, finding.Dispositions[0].Decision)
	require.Equal(t, DecisionAcceptRisk, finding.Dispositions[1].Decision)
	require.Equal(t, DecisionFix, finding.Dispositions[2].Decision)
	require.Equal(t, DecisionAcknowledge, finding.Dispositions[3].Decision)

	_, err = service.RecordDisposition(ctx, "finding-1", "accept_risk", "", "ada")
	require.ErrorIs(t, err, ErrInvalid)
	_, err = service.RecordDisposition(ctx, "finding-1", "accept_risk", "reason", "")
	require.ErrorIs(t, err, ErrInvalid)
	_, err = service.RecordDisposition(ctx, "finding-1", "later", "reason", "ada")
	require.ErrorIs(t, err, ErrInvalid)
	_, err = service.RecordDisposition(ctx, "missing", DecisionFix, "reason", "ada")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestSummaryCountsAndListFilters(t *testing.T) {
	base := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)
	projector := &fakeProjector{findings: []Finding{
		projectedFinding("f1", SeverityBlocker, base),
		projectedFinding("f2", SeverityHigh, base.Add(time.Minute)),
		projectedFinding("f3", SeverityHigh, base.Add(2*time.Minute)),
		projectedFinding("f4", SeverityMedium, base.Add(3*time.Minute)),
	}}
	fixed := projectedFinding("f5", SeverityLow, base.Add(4*time.Minute))
	fixed.Status = StatusFixed
	projector.findings = append(projector.findings, fixed)

	service, _ := newTestService(t, projector)
	ctx := context.Background()
	_, err := service.List(ctx, Filter{}, 10, 0)
	require.NoError(t, err)

	_, err = service.RecordDisposition(ctx, "f3", DecisionAcknowledge, "triaged", "ada")
	require.NoError(t, err)
	_, err = service.RecordDisposition(ctx, "f4", DecisionAcceptRisk, "risk accepted", "ada")
	require.NoError(t, err)

	summary, err := service.Summary(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, summary.Open)
	require.Equal(t, 1, summary.Blockers)
	require.Equal(t, 2, summary.High)
	require.Equal(t, 1, summary.Acknowledged)
	require.Equal(t, 1, summary.Accepted)
	require.Equal(t, 1, summary.Fixed)

	blockers, err := service.List(ctx, Filter{Severity: SeverityBlocker}, 10, 0)
	require.NoError(t, err)
	require.Len(t, blockers.Findings, 1)
	require.Equal(t, 1, blockers.Total)
	require.Equal(t, "f1", blockers.Findings[0].ID)

	search, err := service.List(ctx, Filter{Query: "gate failed"}, 10, 0)
	require.NoError(t, err)
	require.Len(t, search.Findings, 5)
	require.Equal(t, 5, search.Total)

	limited, err := service.List(ctx, Filter{}, 2, 0)
	require.NoError(t, err)
	require.Len(t, limited.Findings, 2)
	require.Equal(t, 5, limited.Total)
	require.Equal(t, defaultListLimit, clampLimit(0))
	require.Equal(t, maxListLimit, clampLimit(5000))

	_, err = service.List(ctx, Filter{Status: "later"}, 10, 0)
	require.ErrorIs(t, err, ErrInvalid)
	_, err = service.List(ctx, Filter{Severity: "urgent"}, 10, 0)
	require.ErrorIs(t, err, ErrInvalid)
	_, err = service.List(ctx, Filter{}, 10, -1)
	require.ErrorIs(t, err, ErrInvalid)
}

func TestListPagesAndCountsTheFilter(t *testing.T) {
	base := time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)
	projected := make([]Finding, 0, 7)
	for index := 0; index < 7; index++ {
		finding := projectedFinding(fmt.Sprintf("f%d", index), SeverityHigh, base.Add(time.Duration(index)*time.Minute))
		finding.Message = "alpha gate failed"
		projected = append(projected, finding)
	}
	projected[0].Severity = SeverityBlocker
	projected[3].Status = StatusAcknowledged
	projected[4].Status = StatusAcknowledged

	service, _ := newTestService(t, &fakeProjector{findings: projected})
	ctx := context.Background()

	first, err := service.List(ctx, Filter{}, 3, 0)
	require.NoError(t, err)
	require.Equal(t, 7, first.Total)
	require.Len(t, first.Findings, 3)
	require.Equal(t, []string{"f6", "f5", "f4"}, findingIDs(first.Findings), "newest-first page")

	second, err := service.List(ctx, Filter{}, 3, 3)
	require.NoError(t, err)
	require.Equal(t, 7, second.Total)
	require.Equal(t, []string{"f3", "f2", "f1"}, findingIDs(second.Findings))

	seen := map[string]bool{}
	for _, finding := range append(append([]Finding{}, first.Findings...), second.Findings...) {
		require.False(t, seen[finding.ID], "pages must not overlap")
		seen[finding.ID] = true
	}

	combined, err := service.List(ctx, Filter{Severity: SeverityHigh, Status: StatusAcknowledged, Query: "alpha"}, 2, 0)
	require.NoError(t, err)
	require.Equal(t, 2, combined.Total, "total describes the filter, not the page")
	require.Len(t, combined.Findings, 2)
	for _, finding := range combined.Findings {
		require.Equal(t, SeverityHigh, finding.Severity)
		require.Equal(t, StatusAcknowledged, finding.Status)
	}

	beyond, err := service.List(ctx, Filter{}, 5, 50)
	require.NoError(t, err)
	require.Empty(t, beyond.Findings)
	require.Equal(t, 7, beyond.Total, "an offset past the end keeps the true total")

	_, err = service.List(ctx, Filter{}, 5, -1)
	require.ErrorIs(t, err, ErrInvalid)
}

func findingIDs(findings []Finding) []string {
	ids := make([]string, 0, len(findings))
	for _, finding := range findings {
		ids = append(ids, finding.ID)
	}
	return ids
}

func TestGetMissingAndAssignment(t *testing.T) {
	projector := &fakeProjector{findings: []Finding{projectedFinding("finding-1", SeverityMedium, time.Now().UTC())}}
	service, _ := newTestService(t, projector)
	ctx := context.Background()
	_, err := service.List(ctx, Filter{}, 10, 0)
	require.NoError(t, err)

	_, err = service.Get(ctx, "nope")
	require.ErrorIs(t, err, ErrNotFound)
	_, err = service.Get(ctx, "  ")
	require.ErrorIs(t, err, ErrInvalid)
	_, err = service.Assign(ctx, "nope", "ada")
	require.ErrorIs(t, err, ErrNotFound)
	_, err = service.Assign(ctx, "", "ada")
	require.ErrorIs(t, err, ErrInvalid)

	cleared, err := service.Assign(ctx, "finding-1", "  ")
	require.NoError(t, err)
	require.Empty(t, cleared.Owner)
}

type blockingProjector struct {
	started chan struct{}
	release chan struct{}
}

func (b *blockingProjector) Project(ctx context.Context) ([]Finding, error) {
	close(b.started)
	select {
	case <-b.release:
	case <-ctx.Done():
	}
	return []Finding{projectedFinding("fresh", SeverityMedium, time.Now().UTC())}, nil
}

// A backlog refresh must not make the person wait. Once a projection exists,
// the next TTL window refreshes behind the read and the previous queue stays
// queryable.
func TestExpiredProjectionRefreshesBehindTheRead(t *testing.T) {
	sqlDB := newReviewDB(t)
	projector := &blockingProjector{started: make(chan struct{}), release: make(chan struct{})}
	service := NewService(sqlDB, projector).WithTTL(time.Minute)
	service.lastRefresh = time.Now().Add(-time.Hour)

	done := make(chan error, 1)
	go func() {
		_, err := service.List(context.Background(), Filter{}, 10, 0)
		done <- err
	}()

	select {
	case <-projector.started:
	case <-time.After(2 * time.Second):
		t.Fatal("background projection never started")
	}
	select {
	case err := <-done:
		require.NoError(t, err, "the read must not wait for the projection")
	case <-time.After(2 * time.Second):
		t.Fatal("List blocked on the background projection")
	}
	close(projector.release)
	require.Eventually(t, func() bool { return !service.refreshing.Load() }, 2*time.Second, 5*time.Millisecond)
}
