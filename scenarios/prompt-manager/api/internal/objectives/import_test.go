package objectives

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"prompt-manager/internal/memberflow"
	"prompt-manager/internal/testsqlite"

	"github.com/vrooli/api-core/database"
)

func newTestDB(t *testing.T) *database.RoutedDB {
	t.Helper()
	db := testsqlite.Open(t)
	if err := database.EnsureSchemas(context.Background(), db.Primary(), database.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("apply objectives schema: %v", err)
	}
	return db
}

const importFixtureDoc = `# Strategy

## The objectives

| ID | Objective | Class | Served by | Evidence |
|----|-----------|-------|-----------|----------|
| ` + "`T1`" + ` | **Income.** Earn revenue. | terminal | ` + "`team:monetization`" + ` (primary), ` + "`team:marketing-crew`" + ` (supporting) | Command Center ledger |
| ` + "`I1`" + ` | **Enablement.** Build capability. | instrumental | ` + "`team:director-swarm`" + ` (primary) | prompt-manager graph audit |
`

func writeImportDoc(t *testing.T, doc string) (repoRoot, configDir string) {
	t.Helper()
	repoRoot = t.TempDir()
	configDir = t.TempDir()
	path := filepath.Join(repoRoot, memberflow.ObjectivesDocPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir objectives doc: %v", err)
	}
	if err := os.WriteFile(path, []byte(doc), 0o644); err != nil {
		t.Fatalf("write objectives doc: %v", err)
	}
	return repoRoot, configDir
}

func writeTeam(t *testing.T, configDir, teamID, body string) {
	t.Helper()
	dir := filepath.Join(configDir, "teams", teamID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir team %s: %v", teamID, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "team.json"), []byte(body), 0o644); err != nil {
		t.Fatalf("write team %s: %v", teamID, err)
	}
}

// fixtureWithMatchingAcknowledgements writes the doc, reads the legacy
// revisions the memberflow parser derives, and declares each team's ack from
// them so the import starts from a clean, confirmed state.
func fixtureWithMatchingAcknowledgements(t *testing.T) (repoRoot, configDir string) {
	t.Helper()
	repoRoot, configDir = writeImportDoc(t, importFixtureDoc)
	reg, err := memberflow.LoadObjectives(repoRoot)
	if err != nil {
		t.Fatalf("load objectives: %v", err)
	}
	rev := map[string]string{}
	for _, o := range reg.Objectives {
		rev[o.ID] = o.Revision
	}
	writeTeam(t, configDir, "monetization",
		`{"id":"monetization","objectivesServed":[{"id":"T1","role":"primary","acknowledgedRevision":"`+rev["T1"]+`"}]}`)
	writeTeam(t, configDir, "marketing-crew",
		`{"id":"marketing-crew","objectivesServed":[{"id":"T1","role":"supporting","acknowledgedRevision":"`+rev["T1"]+`"}]}`)
	writeTeam(t, configDir, "director-swarm",
		`{"id":"director-swarm","objectivesServed":[{"id":"I1","role":"primary","acknowledgedRevision":"`+rev["I1"]+`"}]}`)
	return repoRoot, configDir
}

func TestImportPreviewFingerprintsAndCarriesAcknowledgements(t *testing.T) {
	repoRoot, configDir := fixtureWithMatchingAcknowledgements(t)

	preview, err := PreviewImportFromRepo(repoRoot, configDir)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if preview.Source.Digest == "" {
		t.Fatal("expected a source digest")
	}
	if len(preview.Source.Files) != 4 {
		t.Fatalf("expected doc + 3 team files in the snapshot, got %d", len(preview.Source.Files))
	}
	if len(preview.Objectives) != 2 {
		t.Fatalf("expected 2 objectives, got %d", len(preview.Objectives))
	}
	if len(preview.Attachments) != 3 {
		t.Fatalf("expected 3 attachments, got %d", len(preview.Attachments))
	}
	if len(preview.CoverageGaps) != 0 {
		t.Fatalf("expected no coverage gaps, got %v", preview.CoverageGaps)
	}
	for _, a := range preview.Attachments {
		if !a.AcknowledgementCarried || a.RestatementPending {
			t.Fatalf("acknowledgement for %s/%s must be carried without restatement, got %+v", a.TeamID, a.ObjectiveID, a)
		}
		if a.LegacyAcknowledgedRevision == a.NewAcknowledgedRevision {
			t.Fatalf("expected the digest scheme to change for %s/%s", a.TeamID, a.ObjectiveID)
		}
	}
	for _, c := range preview.Conflicts {
		if c.Severity == importSeverityError {
			t.Fatalf("unexpected blocking conflict: %+v", c)
		}
	}
}

func TestImportAppliesThenReplaysIdempotently(t *testing.T) {
	ctx := context.Background()
	repoRoot, configDir := fixtureWithMatchingAcknowledgements(t)
	preview, err := PreviewImportFromRepo(repoRoot, configDir)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	svc := newTestService(t, nil)

	receipt, err := svc.Import(ctx, preview)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if receipt.AlreadyApplied {
		t.Fatal("first import must not report already-applied")
	}
	if !receipt.SnapshotStored || receipt.ObjectivesImported != 2 || receipt.AttachmentsImported != 3 {
		t.Fatalf("unexpected receipt %+v", receipt)
	}
	if receipt.AcknowledgementsCarried != 3 || receipt.RestatementPending != 0 {
		t.Fatalf("expected 3 carried acknowledgements and 0 pending, got %+v", receipt)
	}
	if _, ok, err := svc.repo.GetImportSnapshot(ctx, preview.Source.Digest); err != nil || !ok {
		t.Fatalf("expected recoverable snapshot, ok=%v err=%v", ok, err)
	}

	objs, err := svc.ListObjectives(ctx)
	if err != nil {
		t.Fatalf("list objectives: %v", err)
	}
	if len(objs) != 2 {
		t.Fatalf("expected 2 objectives after import, got %d", len(objs))
	}
	atts, err := svc.ListTeamAttachments(ctx, "monetization")
	if err != nil {
		t.Fatalf("list attachments: %v", err)
	}
	if len(atts) != 1 || atts[0].RestatementPending {
		t.Fatalf("expected a single non-pending attachment, got %+v", atts)
	}

	replay, err := svc.Import(ctx, preview)
	if err != nil {
		t.Fatalf("replay import: %v", err)
	}
	if !replay.AlreadyApplied {
		t.Fatal("replaying the same source digest must be a no-op")
	}
	objs, _ = svc.ListObjectives(ctx)
	if len(objs) != 2 {
		t.Fatalf("replay must not duplicate objectives, got %d", len(objs))
	}
	var total int
	for _, teamID := range []string{"monetization", "marketing-crew", "director-swarm"} {
		a, _ := svc.ListTeamAttachments(ctx, teamID)
		total += len(a)
	}
	if total != 3 {
		t.Fatalf("replay must not duplicate attachments, got %d", total)
	}
}

func TestImportRefusesBlockingConflictWithoutMutation(t *testing.T) {
	ctx := context.Background()
	repoRoot, configDir := writeImportDoc(t, importFixtureDoc)
	writeTeam(t, configDir, "monetization",
		`{"id":"monetization","objectivesServed":[{"id":"I9","acknowledgedRevision":"deadbeef0000"}]}`)
	preview, err := PreviewImportFromRepo(repoRoot, configDir)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	var blocking bool
	for _, c := range preview.Conflicts {
		if c.Kind == "unknown-objective" && c.Severity == importSeverityError {
			blocking = true
		}
	}
	if !blocking {
		t.Fatalf("expected an unknown-objective blocking conflict, got %+v", preview.Conflicts)
	}

	svc := newTestService(t, nil)
	if _, err := svc.Import(ctx, preview); !errors.Is(err, ErrImportConflict) {
		t.Fatalf("expected import conflict refusal, got %v", err)
	}
	objs, _ := svc.ListObjectives(ctx)
	if len(objs) != 0 {
		t.Fatalf("a refused import must not mutate objectives, got %d", len(objs))
	}
	if _, ok, _ := svc.repo.GetImportSnapshot(ctx, preview.Source.Digest); ok {
		t.Fatal("a refused import must not store a snapshot")
	}
}

func TestImportKeepsAlreadyStaleAcknowledgementPending(t *testing.T) {
	ctx := context.Background()
	repoRoot, configDir := writeImportDoc(t, importFixtureDoc)
	writeTeam(t, configDir, "monetization",
		`{"id":"monetization","objectivesServed":[{"id":"T1","role":"primary","acknowledgedRevision":"000000000000"}]}`)
	preview, err := PreviewImportFromRepo(repoRoot, configDir)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	svc := newTestService(t, nil)
	receipt, err := svc.Import(ctx, preview)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if receipt.AcknowledgementsCarried != 0 || receipt.RestatementPending != 1 {
		t.Fatalf("expected the stale ack to stay pending, got %+v", receipt)
	}
	atts, _ := svc.ListTeamAttachments(ctx, "monetization")
	if len(atts) != 1 || !atts[0].RestatementPending {
		t.Fatalf("expected a pending restatement, got %+v", atts)
	}
}

func TestImportPreservesEvidenceSourceAndMeasurability(t *testing.T) {
	ctx := context.Background()
	repoRoot, configDir := fixtureWithMatchingAcknowledgements(t)
	preview, err := PreviewImportFromRepo(repoRoot, configDir)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	svc := newTestService(t, nil)
	if _, err := svc.Import(ctx, preview); err != nil {
		t.Fatalf("import: %v", err)
	}

	// T1 and I1 declare an evidence source in the fixture doc. The import must
	// preserve it and derive measurability from it; otherwise every imported
	// objective is reported unmeasurable even though it names an instrument.
	for _, id := range []string{"T1", "I1"} {
		o, ok, err := svc.GetObjective(ctx, id)
		if err != nil || !ok {
			t.Fatalf("get %s: ok=%v err=%v", id, ok, err)
		}
		if strings.TrimSpace(o.EvidenceSource) == "" {
			t.Fatalf("%s: evidence source was dropped by the import", id)
		}
		if !o.HasEvidence {
			t.Fatalf("%s: evidence source is present but HasEvidence is false", id)
		}
	}
	result, err := svc.Validate(ctx)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	for _, f := range result.Findings {
		if f.Rule == "objective_unmeasurable" && (f.ObjectiveID == "T1" || f.ObjectiveID == "I1") {
			t.Fatalf("expected %s to be measurable, got finding %+v", f.ObjectiveID, f)
		}
	}
}

// faultRepo injects a failure partway through an import and runs the import
// outside a real transaction, so partial writes persist and the retry must
// converge. It models a store whose WithTx falls back to direct execution.
type faultRepo struct {
	Repository
	failMethod string
	failAt     int
	calls      int
}

func (f *faultRepo) WithTx(ctx context.Context, fn func(Repository) error) error {
	return fn(f)
}

func (f *faultRepo) PutAttachment(ctx context.Context, a Attachment) error {
	f.calls++
	if f.failMethod == "PutAttachment" && f.calls == f.failAt {
		return errors.New("injected interruption")
	}
	return f.Repository.PutAttachment(ctx, a)
}

func TestImportRecoversAfterInterruption(t *testing.T) {
	ctx := context.Background()
	repoRoot, configDir := fixtureWithMatchingAcknowledgements(t)
	preview, err := PreviewImportFromRepo(repoRoot, configDir)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}

	real := NewRepository(newTestDB(t))
	interrupted := NewService(&faultRepo{Repository: real, failMethod: "PutAttachment", failAt: 1}, nil)
	if _, err := interrupted.Import(ctx, preview); err == nil {
		t.Fatal("expected the injected interruption to fail the import")
	}
	objs, _ := real.ListObjectives(ctx)
	if len(objs) == 0 {
		t.Fatal("expected partial objectives to persist after interruption (no transaction)")
	}
	if _, ok, _ := real.GetImportReceipt(ctx, preview.Source.Digest); ok {
		t.Fatal("an interrupted import must not leave a receipt")
	}

	clean := NewService(real, nil)
	receipt, err := clean.Import(ctx, preview)
	if err != nil {
		t.Fatalf("retry after interruption: %v", err)
	}
	if receipt.AlreadyApplied || receipt.ObjectivesImported != 2 || receipt.AttachmentsImported != 3 {
		t.Fatalf("unexpected retry receipt %+v", receipt)
	}
	objs, _ = real.ListObjectives(ctx)
	if len(objs) != 2 {
		t.Fatalf("retry must converge on 2 objectives, got %d", len(objs))
	}
	var total int
	for _, teamID := range []string{"monetization", "marketing-crew", "director-swarm"} {
		a, _ := real.ListAttachments(ctx, teamID)
		total += len(a)
	}
	if total != 3 {
		t.Fatalf("retry must converge on 3 attachments, got %d", total)
	}
	if _, ok, _ := real.GetImportSnapshot(ctx, preview.Source.Digest); !ok {
		t.Fatal("expected the snapshot to be stored by the successful retry")
	}
}
