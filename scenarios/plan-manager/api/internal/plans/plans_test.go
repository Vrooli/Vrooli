package plans_test

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"plan-manager/internal/plans"
	"plan-manager/internal/testutil/db"
	"plan-manager/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
	repocontract "github.com/vrooli/repo-contract-go"

	apidb "github.com/vrooli/api-core/database"

	localdb "plan-manager/internal/database"
)

func newDB(t *testing.T) (*sql.DB, *mocks.FakeClock) {
	t.Helper()
	d := db.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(plans.Schema),
	))
	clk := mocks.NewFakeClock(time.Date(2026, 6, 25, 12, 0, 0, 0, time.UTC))
	return d, clk
}

func newService(t *testing.T) (plans.Service, *mocks.FakeClock) {
	t.Helper()
	d, clk := newDB(t)
	svc := plans.NewService(plans.Deps{
		Repo:   plans.NewSQLiteRepository(d, clk),
		Clock:  clk,
		Reader: fakeReader{},
		Mirror: newFakeMirrorStore(t.TempDir()),
	})
	return svc, clk
}

func validWorkspaceRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	require.NoError(t, err)
	if _, err := os.Stat(filepath.Join(root, ".vrooli", "repo-contract.json")); err != nil {
		t.Fatalf("valid workspace fixture missing repo contract: %v", err)
	}
	return root
}

type fakeMirrorStore struct {
	root        string
	files       map[string][]byte
	publishErr  error
	publishSeen int
}

func newFakeMirrorStore(root string) *fakeMirrorStore {
	return &fakeMirrorStore{root: root, files: map[string][]byte{}}
}

func (f *fakeMirrorStore) PathFor(_ context.Context, p plans.Plan) (plans.RenderedPlanMirror, error) {
	rel := p.Slug + ".md"
	return plans.RenderedPlanMirror{
		Path:          filepath.Join(f.root, rel),
		RelativePath:  rel,
		RenderVersion: plans.RendererVersion,
		Status:        plans.RenderedMirrorStatusUnknown,
	}, nil
}

func (f *fakeMirrorStore) Read(ctx context.Context, p plans.Plan) ([]byte, plans.RenderedPlanMirror, error) {
	meta, err := f.PathFor(ctx, p)
	if err != nil {
		return nil, meta, err
	}
	data, ok := f.files[meta.Path]
	if !ok {
		meta.Status = plans.RenderedMirrorStatusMissing
		return nil, meta, os.ErrNotExist
	}
	meta.ContentHash = renderedHashForTest(data)
	meta.RenderedAt = p.Mirror.RenderedAt
	if p.Mirror.ContentHash == meta.ContentHash && p.Mirror.RenderVersion == plans.RendererVersion {
		meta.Status = plans.RenderedMirrorStatusFresh
		return data, meta, nil
	}
	meta.Status = plans.RenderedMirrorStatusStale
	return data, meta, nil
}

func (f *fakeMirrorStore) Publish(ctx context.Context, p plans.Plan, markdown []byte, renderedAt string) (plans.RenderedPlanMirror, error) {
	f.publishSeen++
	meta, err := f.PathFor(ctx, p)
	if err != nil {
		return meta, err
	}
	if f.publishErr != nil {
		meta.Status = plans.RenderedMirrorStatusWriteFailed
		meta.LastError = f.publishErr.Error()
		return meta, f.publishErr
	}
	cp := append([]byte(nil), markdown...)
	f.files[meta.Path] = cp
	meta.ContentHash = renderedHashForTest(cp)
	meta.RenderedAt = renderedAt
	meta.Status = plans.RenderedMirrorStatusFresh
	return meta, nil
}

func renderedHashForTest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// TestStorageRoundTripsNewProfessionalFields asserts every new professional
// plan/phase field survives a SQLite save -> get round trip (the JSON document).
func TestStorageRoundTripsNewProfessionalFields(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, plans.Plan{
		Title:                   "Round trip",
		ProblemStatement:        "A problem.",
		TargetOutcome:           "An outcome.",
		Assumptions:             "Some assumptions.",
		TechnicalApproach:       "An approach.",
		ValidationStrategy:      "A strategy.",
		FinalValidationCommands: []string{"vrooli scenario test plan-manager"},
		RisksHazards:            "A risk.",
		ProhibitedApproaches:    "No shims.",
		Phases: []plans.Phase{{
			Title:           "P1",
			Intent:          "do",
			AffectedAreas:   []string{"a/b.go"},
			Steps:           []string{"s1", "s2"},
			ExpectedOutputs: []string{"o1"},
			Validation:      "go test ./...",
			Acceptance:      "passes",
			RisksHazards:    []string{"r1"},
			HandoffNotes:    "next phase needs this",
			Status:          plans.PhaseStatusTodo,
		}},
	})
	require.NoError(t, err)

	got, err := svc.Get(ctx, created.ID, plans.WorkspaceScope{})
	require.NoError(t, err)
	require.Equal(t, "A problem.", got.ProblemStatement)
	require.Equal(t, "An outcome.", got.TargetOutcome)
	require.Equal(t, "Some assumptions.", got.Assumptions)
	require.Equal(t, "An approach.", got.TechnicalApproach)
	require.Equal(t, "A strategy.", got.ValidationStrategy)
	require.Equal(t, []string{"vrooli scenario test plan-manager"}, got.FinalValidationCommands)
	require.Equal(t, "A risk.", got.RisksHazards)
	require.Equal(t, "No shims.", got.ProhibitedApproaches)
	// Work posture is autofilled on save (default greenfield — no scenario signal).
	require.Equal(t, plans.WorkPostureGreenfield, got.WorkPosture)
	require.NotEmpty(t, got.WorkPostureSource)

	require.Len(t, got.Phases, 1)
	ph := got.Phases[0]
	require.Equal(t, []string{"a/b.go"}, ph.AffectedAreas)
	require.Equal(t, []string{"s1", "s2"}, ph.Steps)
	require.Equal(t, []string{"o1"}, ph.ExpectedOutputs)
	require.Equal(t, "go test ./...", ph.Validation)
	require.Equal(t, []string{"r1"}, ph.RisksHazards)
	require.Equal(t, "next phase needs this", ph.HandoffNotes)
}

func TestStorageRoundTripsCanonicalWorkspaceFields(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, plans.Plan{
		Title:         "Scoped plan",
		WorkspaceID:   "ws-1",
		WorkspaceRoot: filepath.Join(t.TempDir(), "workspace"),
	})
	require.NoError(t, err)

	got, err := svc.Get(ctx, created.ID, plans.WorkspaceScope{ID: "ws-1"})
	require.NoError(t, err)
	require.Equal(t, "ws-1", got.WorkspaceID)
	require.Equal(t, created.WorkspaceRoot, got.WorkspaceRoot)
}

func TestWorkspaceScopedListAndLookup(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()
	wsA := plans.WorkspaceScope{Root: filepath.Join(t.TempDir(), "a")}
	wsB := plans.WorkspaceScope{Root: filepath.Join(t.TempDir(), "b")}

	a, err := svc.Create(ctx, plans.Plan{Title: "Shared Title", Slug: "shared", WorkspaceRoot: wsA.Root})
	require.NoError(t, err)
	b, err := svc.Create(ctx, plans.Plan{Title: "Shared Title", Slug: "shared", WorkspaceRoot: wsB.Root})
	require.NoError(t, err)
	require.Equal(t, "shared", a.Slug)
	require.Equal(t, "shared", b.Slug, "slug uniqueness is workspace-scoped")

	listA, err := svc.List(ctx, plans.ListFilter{WorkspaceRoot: wsA.Root})
	require.NoError(t, err)
	require.Len(t, listA, 1)
	require.Equal(t, a.ID, listA[0].ID)

	got, err := svc.Get(ctx, "shared", wsB)
	require.NoError(t, err)
	require.Equal(t, b.ID, got.ID)

	_, err = svc.Get(ctx, a.ID, wsB)
	require.Error(t, err, "scoped lookup must not return a plan from another workspace")
}

func TestImportStampsCanonicalWorkspaceFields(t *testing.T) {
	d, clk := newDB(t)
	mirror := newFakeMirrorStore(t.TempDir())
	workspaceRoot := validWorkspaceRoot(t)
	source := filepath.Join(workspaceRoot, "docs", "plans", "legacy.md")
	reader := fakeReader{files: map[string]string{
		source: "# Workspace Import\n\n## Purpose\nAdopt from workspace.\n",
	}}
	svc := plans.NewService(plans.Deps{
		Repo:   plans.NewSQLiteRepository(d, clk),
		Clock:  clk,
		Reader: reader,
		Mirror: mirror,
	})
	ctx := context.Background()

	imported, err := svc.Import(ctx, "docs/plans/legacy.md", "", "", "", plans.WorkspaceScope{ID: "ws-import", Root: workspaceRoot})
	require.NoError(t, err)
	require.Equal(t, "ws-import", imported.WorkspaceID)
	require.Equal(t, workspaceRoot, imported.WorkspaceRoot)
	require.NotNil(t, imported.ImportProvenance)
	require.Equal(t, "ws-import", imported.ImportProvenance.WorkspaceID)
	require.Equal(t, workspaceRoot, imported.ImportProvenance.WorkspaceRoot)

	got, err := svc.Get(ctx, imported.Slug, plans.WorkspaceScope{Root: workspaceRoot})
	require.NoError(t, err)
	require.Equal(t, imported.ID, got.ID)
}

func TestImportRejectsAbsoluteSourceOutsideWorkspaceRoot(t *testing.T) {
	d, clk := newDB(t)
	reader := fakeReader{files: map[string]string{}}
	svc := plans.NewService(plans.Deps{
		Repo:   plans.NewSQLiteRepository(d, clk),
		Clock:  clk,
		Reader: reader,
		Mirror: newFakeMirrorStore(t.TempDir()),
	})
	ctx := context.Background()
	workspaceRoot := validWorkspaceRoot(t)

	_, err := svc.Import(ctx, filepath.Join(t.TempDir(), "external.md"), "", "", "", plans.WorkspaceScope{Root: workspaceRoot})
	require.Error(t, err)
	require.Contains(t, err.Error(), "outside workspace root")
}

func TestImportAllowsRuntimeHomePlansSourceForWorkspace(t *testing.T) {
	d, clk := newDB(t)
	workspaceRoot := validWorkspaceRoot(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	runtimePlans, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyPlans)
	require.NoError(t, err)
	source := filepath.Join(runtimePlans, "legacy.md")
	reader := fakeReader{files: map[string]string{
		source: "# Runtime Legacy\n\n## Purpose\nAdopt from contract-owned runtime home.\n",
	}}
	svc := plans.NewService(plans.Deps{
		Repo:   plans.NewSQLiteRepository(d, clk),
		Clock:  clk,
		Reader: reader,
		Mirror: newFakeMirrorStore(t.TempDir()),
	})
	ctx := context.Background()

	imported, err := svc.Import(ctx, source, "", "", "", plans.WorkspaceScope{Root: workspaceRoot})
	require.NoError(t, err)
	require.Equal(t, "Runtime Legacy", imported.Title)
	require.NotNil(t, imported.ImportProvenance)
	require.Equal(t, source, imported.ImportProvenance.SourcePath)
	require.Equal(t, workspaceRoot, imported.ImportProvenance.WorkspaceRoot)
}

func TestImportRejectsInvalidWorkspaceRoot(t *testing.T) {
	d, clk := newDB(t)
	workspaceRoot := t.TempDir()
	source := filepath.Join(workspaceRoot, "docs", "plans", "legacy.md")
	reader := fakeReader{files: map[string]string{
		source: "# Legacy\n",
	}}
	svc := plans.NewService(plans.Deps{
		Repo:   plans.NewSQLiteRepository(d, clk),
		Clock:  clk,
		Reader: reader,
		Mirror: newFakeMirrorStore(t.TempDir()),
	})
	ctx := context.Background()

	_, err := svc.Import(ctx, source, "", "", "", plans.WorkspaceScope{Root: workspaceRoot})
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid workspace root")
}

// TestContentHashChangesWithNewFields asserts the content hash incorporates the
// new authored fields (changing one changes the hash).
func TestContentHashChangesWithNewFields(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()
	base, err := svc.Create(ctx, plans.Plan{Title: "Hash A", TechnicalApproach: "approach one"})
	require.NoError(t, err)
	other, err := svc.Create(ctx, plans.Plan{Title: "Hash A", TechnicalApproach: "approach two"})
	require.NoError(t, err)
	require.NotEqual(t, base.ContentHash, other.ContentHash, "technical_approach must affect the content hash")
}

func TestCreatePublishesRenderedMirror(t *testing.T) {
	d, clk := newDB(t)
	mirror := newFakeMirrorStore(t.TempDir())
	svc := plans.NewService(plans.Deps{
		Repo:   plans.NewSQLiteRepository(d, clk),
		Clock:  clk,
		Reader: fakeReader{},
		Mirror: mirror,
	})
	ctx := context.Background()

	created, err := svc.Create(ctx, plans.Plan{Title: "Mirror Me", Purpose: "Durable view."})
	require.NoError(t, err)
	require.Equal(t, plans.RenderedMirrorStatusFresh, created.Mirror.Status)
	require.NotEmpty(t, created.Mirror.Path)
	require.Equal(t, "mirror-me.md", created.Mirror.RelativePath)
	require.NotEmpty(t, mirror.files[created.Mirror.Path])

	got, err := svc.Get(ctx, created.ID, plans.WorkspaceScope{})
	require.NoError(t, err)
	require.Equal(t, created.Mirror.ContentHash, got.Mirror.ContentHash)
	require.Equal(t, created.Mirror.Path, got.Mirror.Path)
}

func TestOSMirrorStoreWritesWorkspaceAwareIndex(t *testing.T) {
	d, clk := newDB(t)
	home := t.TempDir()
	workspaceRoot := validWorkspaceRoot(t)
	svc := plans.NewService(plans.Deps{
		Repo:   plans.NewSQLiteRepository(d, clk),
		Clock:  clk,
		Reader: fakeReader{},
		Mirror: plans.NewOSMirrorStore(home),
	})
	ctx := context.Background()

	created, err := svc.Create(ctx, plans.Plan{
		Title:         "Indexed Mirror",
		Purpose:       "Publish index metadata.",
		WorkspaceID:   "ws-index",
		WorkspaceRoot: workspaceRoot,
	})
	require.NoError(t, err)
	mirrorRoot, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyPlans)
	require.NoError(t, err)
	raw, err := os.ReadFile(filepath.Join(mirrorRoot, "_index.json"))
	require.NoError(t, err)
	var idx struct {
		Version int `json:"version"`
		Plans   []struct {
			ID            string `json:"id"`
			Slug          string `json:"slug"`
			Path          string `json:"path"`
			WorkspaceID   string `json:"workspace_id"`
			WorkspaceRoot string `json:"workspace_root"`
		} `json:"plans"`
	}
	require.NoError(t, json.Unmarshal(raw, &idx))
	require.Equal(t, 2, idx.Version)
	require.Len(t, idx.Plans, 1)
	require.Equal(t, created.ID, idx.Plans[0].ID)
	require.Equal(t, created.Slug, idx.Plans[0].Slug)
	require.Equal(t, created.Mirror.Path, idx.Plans[0].Path)
	require.Equal(t, "ws-index", idx.Plans[0].WorkspaceID)
	require.Equal(t, workspaceRoot, idx.Plans[0].WorkspaceRoot)
}

func TestRenderRepairsMissingMirrorFromCanonicalPlan(t *testing.T) {
	d, clk := newDB(t)
	mirror := newFakeMirrorStore(t.TempDir())
	svc := plans.NewService(plans.Deps{
		Repo:   plans.NewSQLiteRepository(d, clk),
		Clock:  clk,
		Reader: fakeReader{},
		Mirror: mirror,
	})
	ctx := context.Background()
	created, err := svc.Create(ctx, plans.Plan{Title: "Repair Missing", Purpose: "Canonical text."})
	require.NoError(t, err)
	delete(mirror.files, created.Mirror.Path)

	rendered, err := svc.Render(ctx, created.ID, plans.WorkspaceScope{}, plans.RenderOptions{})
	require.NoError(t, err)
	require.True(t, rendered.Repaired)
	require.Equal(t, plans.RenderedMirrorStatusFresh, rendered.Mirror.Status)
	require.Contains(t, rendered.Markdown, "Canonical text.")
	require.NotEmpty(t, mirror.files[created.Mirror.Path])
}

func TestRenderRepairsStaleMirrorFromCanonicalPlan(t *testing.T) {
	d, clk := newDB(t)
	mirror := newFakeMirrorStore(t.TempDir())
	svc := plans.NewService(plans.Deps{
		Repo:   plans.NewSQLiteRepository(d, clk),
		Clock:  clk,
		Reader: fakeReader{},
		Mirror: mirror,
	})
	ctx := context.Background()
	created, err := svc.Create(ctx, plans.Plan{Title: "Repair Stale", Purpose: "Canonical text."})
	require.NoError(t, err)
	mirror.files[created.Mirror.Path] = []byte("# Edited by hand\n")

	rendered, err := svc.Render(ctx, created.ID, plans.WorkspaceScope{}, plans.RenderOptions{})
	require.NoError(t, err)
	require.True(t, rendered.Repaired)
	require.Equal(t, plans.RenderedMirrorStatusFresh, rendered.Mirror.Status)
	require.Contains(t, string(mirror.files[created.Mirror.Path]), "Canonical text.")
	require.NotContains(t, rendered.Markdown, "Edited by hand")
}

func TestReconcileDryRunReportsMirrorRepairWithoutWriting(t *testing.T) {
	d, clk := newDB(t)
	mirror := newFakeMirrorStore(t.TempDir())
	svc := plans.NewService(plans.Deps{
		Repo:   plans.NewSQLiteRepository(d, clk),
		Clock:  clk,
		Reader: fakeReader{},
		Mirror: mirror,
	})
	ctx := context.Background()
	created, err := svc.Create(ctx, plans.Plan{Title: "Dry Repair", Purpose: "Canonical."})
	require.NoError(t, err)
	delete(mirror.files, created.Mirror.Path)
	seen := mirror.publishSeen

	report, err := svc.Reconcile(ctx, plans.ReconcileRequest{DryRun: true, RepairMirrors: true})
	require.NoError(t, err)
	require.Len(t, report.Items, 1)
	require.Equal(t, plans.ReconcileActionMirrorRepairNeeded, report.Items[0].Action)
	require.Equal(t, seen, mirror.publishSeen, "dry-run must not publish mirrors")
	require.Empty(t, mirror.files[created.Mirror.Path])
}

func TestReconcileMirrorRepairHonorsWorkspaceScope(t *testing.T) {
	d, clk := newDB(t)
	mirror := newFakeMirrorStore(t.TempDir())
	svc := plans.NewService(plans.Deps{
		Repo:   plans.NewSQLiteRepository(d, clk),
		Clock:  clk,
		Reader: fakeReader{},
		Mirror: mirror,
	})
	ctx := context.Background()
	wsA := plans.WorkspaceScope{Root: filepath.Join(t.TempDir(), "a")}
	wsB := plans.WorkspaceScope{Root: filepath.Join(t.TempDir(), "b")}

	a, err := svc.Create(ctx, plans.Plan{Title: "Scoped Repair A", Purpose: "Canonical.", WorkspaceRoot: wsA.Root})
	require.NoError(t, err)
	b, err := svc.Create(ctx, plans.Plan{Title: "Scoped Repair B", Purpose: "Canonical.", WorkspaceRoot: wsB.Root})
	require.NoError(t, err)
	delete(mirror.files, a.Mirror.Path)
	delete(mirror.files, b.Mirror.Path)

	report, err := svc.Reconcile(ctx, plans.ReconcileRequest{
		DryRun:        true,
		RepairMirrors: true,
		Workspace:     wsA,
	})
	require.NoError(t, err)
	require.Len(t, report.Items, 1)
	require.Equal(t, a.ID, report.Items[0].PlanID)
	require.Equal(t, plans.ReconcileActionMirrorRepairNeeded, report.Items[0].Action)
}

func TestReconcileAdoptsLegacyPlansNonDestructively(t *testing.T) {
	d, clk := newDB(t)
	mirror := newFakeMirrorStore(t.TempDir())
	source := filepath.Join("docs", "plans", "legacy.md")
	reader := fakeReader{files: map[string]string{
		source: "# Legacy plan\n\n## Purpose\nAdopt me.\n",
	}}
	svc := plans.NewService(plans.Deps{
		Repo:   plans.NewSQLiteRepository(d, clk),
		Clock:  clk,
		Reader: reader,
		Mirror: mirror,
	})
	ctx := context.Background()

	report, err := svc.Reconcile(ctx, plans.ReconcileRequest{AdoptLegacy: true, SourceDocsPlans: true})
	require.NoError(t, err)
	require.Len(t, report.Items, 1)
	require.Equal(t, plans.ReconcileActionImported, report.Items[0].Action)
	require.True(t, report.Items[0].SourceUntouched)
	require.Equal(t, source, report.Items[0].SourcePath)
	require.Equal(t, "# Legacy plan\n\n## Purpose\nAdopt me.\n", reader.files[source], "legacy source must remain untouched")

	imported, err := svc.Get(ctx, report.Items[0].PlanID, plans.WorkspaceScope{})
	require.NoError(t, err)
	require.NotNil(t, imported.ImportProvenance)
	require.Equal(t, source, imported.ImportProvenance.SourcePath)
	require.NotEmpty(t, mirror.files[imported.Mirror.Path])
}

func TestReconcileCleanupAdoptedSourcesRemovesOnlyProvenLegacySources(t *testing.T) {
	d, clk := newDB(t)
	mirror := newFakeMirrorStore(t.TempDir())
	source := filepath.Join("docs", "plans", "legacy.md")
	reader := fakeReader{files: map[string]string{
		source: "# Legacy plan\n\n## Purpose\nAdopt me.\n",
	}}
	svc := plans.NewService(plans.Deps{
		Repo:   plans.NewSQLiteRepository(d, clk),
		Clock:  clk,
		Reader: reader,
		Mirror: mirror,
	})
	ctx := context.Background()

	report, err := svc.Reconcile(ctx, plans.ReconcileRequest{
		AdoptLegacy:           true,
		CleanupAdoptedSources: true,
		SourceDocsPlans:       true,
	})
	require.NoError(t, err)
	require.Len(t, report.Items, 1)
	require.Equal(t, plans.ReconcileActionImported, report.Items[0].Action)
	require.True(t, report.Items[0].SourceRemoved)
	require.False(t, report.Items[0].SourceUntouched)
	require.NotContains(t, reader.files, source)

	imported, err := svc.Get(ctx, report.Items[0].PlanID, plans.WorkspaceScope{})
	require.NoError(t, err)
	require.NotNil(t, imported.ImportProvenance)
	require.Equal(t, source, imported.ImportProvenance.SourcePath)
	require.NotEmpty(t, mirror.files[imported.Mirror.Path])
}

func TestReconcileCleanupPrunesFallbackIndexAndStaysIdempotent(t *testing.T) {
	d, clk := newDB(t)
	sourceDir := filepath.Join("docs", "plans")
	source := filepath.Join(sourceDir, "legacy-alias.md")
	reader := fakeReader{files: map[string]string{
		filepath.Join(sourceDir, "_index.json"): `{
  "version": 2,
  "plans": [
    {
      "id": "legacy-1",
      "title": "Legacy Alias",
      "slug": "legacy-alias",
      "path": "docs/plans/legacy-alias.md"
    }
  ]
}`,
		source: "# Legacy Alias\n\n## Purpose\nAdopt me.\n",
	}}
	svc := plans.NewService(plans.Deps{
		Repo:   plans.NewSQLiteRepository(d, clk),
		Clock:  clk,
		Reader: reader,
		Mirror: newFakeMirrorStore(t.TempDir()),
	})
	ctx := context.Background()

	applied, err := svc.Reconcile(ctx, plans.ReconcileRequest{
		AdoptLegacy:           true,
		CleanupAdoptedSources: true,
		SourceDocsPlans:       true,
	})
	require.NoError(t, err)
	require.Len(t, applied.Items, 1)
	require.Equal(t, plans.ReconcileActionImported, applied.Items[0].Action)
	require.True(t, applied.Items[0].SourceRemoved)
	require.NotContains(t, reader.files, source)

	var idx struct {
		Plans []struct {
			Path string `json:"path"`
		} `json:"plans"`
	}
	require.NoError(t, json.Unmarshal([]byte(reader.files[filepath.Join(sourceDir, "_index.json")]), &idx))
	require.Empty(t, idx.Plans, "cleanup must remove stale fallback index records for deleted sources")

	dryRun, err := svc.Reconcile(ctx, plans.ReconcileRequest{
		DryRun:                true,
		AdoptLegacy:           true,
		CleanupAdoptedSources: true,
		SourceDocsPlans:       true,
	})
	require.NoError(t, err)
	for _, item := range dryRun.Items {
		require.NotEqual(t, source, item.SourcePath, "deleted index source must not be rediscovered")
	}
}

func TestReconcileIgnoresIndexedImportedSourceThatIsAlreadyMissing(t *testing.T) {
	d, clk := newDB(t)
	sourceDir := filepath.Join("docs", "plans")
	source := filepath.Join(sourceDir, "removed-alias.md")
	reader := fakeReader{files: map[string]string{
		filepath.Join(sourceDir, "_index.json"): `{
  "version": 2,
  "plans": [
    {
      "id": "legacy-1",
      "title": "Removed Alias",
      "slug": "removed-alias",
      "path": "docs/plans/removed-alias.md"
    }
  ]
}`,
	}}
	svc := plans.NewService(plans.Deps{
		Repo:   plans.NewSQLiteRepository(d, clk),
		Clock:  clk,
		Reader: reader,
		Mirror: newFakeMirrorStore(t.TempDir()),
	})
	ctx := context.Background()
	created, err := svc.Create(ctx, plans.Plan{
		Title: "Removed Alias",
		ImportProvenance: &plans.ImportProvenance{
			SourcePath: source,
		},
	})
	require.NoError(t, err)

	report, err := svc.Reconcile(ctx, plans.ReconcileRequest{
		DryRun:                true,
		AdoptLegacy:           true,
		CleanupAdoptedSources: true,
		SourceDocsPlans:       true,
	})
	require.NoError(t, err)
	for _, item := range report.Items {
		require.NotEqual(t, source, item.SourcePath, "missing indexed source for imported plan must not be reported as cleanup-planned")
	}
	got, err := svc.Get(ctx, created.ID, plans.WorkspaceScope{})
	require.NoError(t, err)
	require.Equal(t, source, got.ImportProvenance.SourcePath, "reconcile should not rewrite import provenance to hide history")
}

func TestReconcileDryRunReportsSlugConflictInsteadOfImportPlanned(t *testing.T) {
	d, clk := newDB(t)
	workspace := plans.WorkspaceScope{Root: validWorkspaceRoot(t)}
	source := filepath.Join(workspace.Root, "docs", "plans", "same-title.md")
	reader := fakeReader{files: map[string]string{
		source: "# Same Title\n\n## Purpose\nDifferent legacy content.\n",
	}}
	svc := plans.NewService(plans.Deps{
		Repo:   plans.NewSQLiteRepository(d, clk),
		Clock:  clk,
		Reader: reader,
		Mirror: newFakeMirrorStore(t.TempDir()),
	})
	ctx := context.Background()
	existing, err := svc.Create(ctx, plans.Plan{Title: "Same Title", Purpose: "Existing canonical content."})
	require.NoError(t, err)
	require.Empty(t, existing.WorkspaceRoot, "fixture plan intentionally lives outside the reconcile workspace")

	dryRun, err := svc.Reconcile(ctx, plans.ReconcileRequest{
		DryRun:                true,
		AdoptLegacy:           true,
		CleanupAdoptedSources: true,
		SourceDocsPlans:       true,
		Workspace:             workspace,
	})
	require.NoError(t, err)
	require.Len(t, dryRun.Items, 1)
	require.Equal(t, plans.ReconcileActionConflict, dryRun.Items[0].Action)
	require.Equal(t, existing.ID, dryRun.Items[0].PlanID)
	require.False(t, dryRun.Items[0].SourceCleanupPlanned)
	require.Contains(t, dryRun.Items[0].Error, `plan slug "same-title" already exists`)

	applied, err := svc.Reconcile(ctx, plans.ReconcileRequest{
		AdoptLegacy:           true,
		CleanupAdoptedSources: true,
		SourceDocsPlans:       true,
		Workspace:             workspace,
	})
	require.NoError(t, err)
	require.Len(t, applied.Items, 1)
	require.Equal(t, plans.ReconcileActionConflict, applied.Items[0].Action)
	require.Contains(t, reader.files, source)
}

func TestReconcileWorkspaceScopeProtectsGlobalCanonicalMirrors(t *testing.T) {
	d, clk := newDB(t)
	workspace := plans.WorkspaceScope{Root: validWorkspaceRoot(t)}
	home := t.TempDir()
	t.Setenv("HOME", home)
	runtimePlans, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyPlans)
	require.NoError(t, err)

	mirror := newFakeMirrorStore(runtimePlans)
	svc := plans.NewService(plans.Deps{
		Repo:   plans.NewSQLiteRepository(d, clk),
		Clock:  clk,
		Reader: fakeReader{files: map[string]string{}},
		Mirror: mirror,
	})
	ctx := context.Background()
	existing, err := svc.Create(ctx, plans.Plan{Title: "Global Canonical", Purpose: "Existing canonical content."})
	require.NoError(t, err)
	require.Empty(t, existing.WorkspaceRoot, "fixture plan intentionally lives outside the reconcile workspace")
	mirrorPath, err := mirror.PathFor(ctx, existing)
	require.NoError(t, err)

	reader := fakeReader{files: map[string]string{
		mirrorPath.Path: "not parseable legacy markdown",
	}}
	svc = plans.NewService(plans.Deps{
		Repo:   plans.NewSQLiteRepository(d, clk),
		Clock:  clk,
		Reader: reader,
		Mirror: mirror,
	})

	report, err := svc.Reconcile(ctx, plans.ReconcileRequest{
		DryRun:                 true,
		AdoptLegacy:            true,
		SourceRuntimeHomePlans: true,
		Workspace:              workspace,
	})
	require.NoError(t, err)
	require.Len(t, report.Items, 1)
	require.Equal(t, plans.ReconcileActionAlreadyCanonical, report.Items[0].Action)
	require.Equal(t, mirrorPath.Path, report.Items[0].SourcePath)
	require.True(t, report.Items[0].SourceUntouched)
	require.False(t, report.Items[0].SourceCleanupPlanned)
}

func TestReconcileCleanupDryRunReportsPlannedRemovalWithoutRemoving(t *testing.T) {
	d, clk := newDB(t)
	source := filepath.Join("docs", "plans", "legacy.md")
	reader := fakeReader{files: map[string]string{
		source: "# Legacy plan\n\n## Purpose\nAdopt me.\n",
	}}
	svc := plans.NewService(plans.Deps{Repo: plans.NewSQLiteRepository(d, clk), Clock: clk, Reader: reader})
	ctx := context.Background()

	report, err := svc.Reconcile(ctx, plans.ReconcileRequest{
		DryRun:                true,
		AdoptLegacy:           true,
		CleanupAdoptedSources: true,
		SourceDocsPlans:       true,
	})
	require.NoError(t, err)
	require.Len(t, report.Items, 1)
	require.Equal(t, plans.ReconcileActionImportPlanned, report.Items[0].Action)
	require.True(t, report.Items[0].SourceCleanupPlanned)
	require.True(t, report.Items[0].SourceUntouched)
	require.False(t, report.Items[0].SourceRemoved)
	require.Contains(t, reader.files, source)
}

func TestReconcileCleanupNeverRemovesParseFailuresOrCanonicalMirrors(t *testing.T) {
	d, clk := newDB(t)
	mirror := newFakeMirrorStore(filepath.Join("docs", "plans"))
	reader := fakeReader{files: map[string]string{
		filepath.Join("docs", "plans", "bad.md"): "# Bad\n\n## Phases\n\n### Phase A — malformed\n",
	}}
	svc := plans.NewService(plans.Deps{
		Repo:   plans.NewSQLiteRepository(d, clk),
		Clock:  clk,
		Reader: reader,
		Mirror: mirror,
	})
	ctx := context.Background()
	created, err := svc.Create(ctx, plans.Plan{Title: "Canonical Mirror", Purpose: "Protect mirror."})
	require.NoError(t, err)
	reader.files[created.Mirror.Path] = string(mirror.files[created.Mirror.Path])

	report, err := svc.Reconcile(ctx, plans.ReconcileRequest{
		AdoptLegacy:           true,
		CleanupAdoptedSources: true,
		SourceDocsPlans:       true,
	})
	require.NoError(t, err)
	paths := map[string]plans.ReconcileItem{}
	for _, item := range report.Items {
		paths[item.SourcePath] = item
	}
	badPath := filepath.Join("docs", "plans", "bad.md")
	require.Equal(t, plans.ReconcileActionParseFailed, paths[badPath].Action)
	require.True(t, paths[badPath].SourceUntouched)
	require.Contains(t, reader.files, badPath)
	require.Equal(t, plans.ReconcileActionAlreadyCanonical, paths[created.Mirror.Path].Action)
	require.True(t, paths[created.Mirror.Path].SourceUntouched)
	require.False(t, paths[created.Mirror.Path].SourceCleanupPlanned)
	require.Contains(t, reader.files, created.Mirror.Path)
}

func TestReconcileAdoptsWorkspaceRelativeLegacyPlans(t *testing.T) {
	d, clk := newDB(t)
	mirror := newFakeMirrorStore(t.TempDir())
	workspaceRoot := validWorkspaceRoot(t)
	source := filepath.Join(workspaceRoot, "docs", "plans", "workspace-legacy.md")
	reader := fakeReader{files: map[string]string{
		source: "# Workspace Legacy\n\n## Purpose\nAdopt from the scoped workspace.\n",
	}}
	svc := plans.NewService(plans.Deps{
		Repo:   plans.NewSQLiteRepository(d, clk),
		Clock:  clk,
		Reader: reader,
		Mirror: mirror,
	})
	ctx := context.Background()

	report, err := svc.Reconcile(ctx, plans.ReconcileRequest{
		AdoptLegacy:     true,
		SourceDocsPlans: true,
		Workspace:       plans.WorkspaceScope{Root: workspaceRoot},
	})
	require.NoError(t, err)
	require.Len(t, report.Items, 1)
	require.Equal(t, plans.ReconcileActionImported, report.Items[0].Action)
	require.Equal(t, source, report.Items[0].SourcePath)
}

type fakeReader struct{ files map[string]string }

func (f fakeReader) ReadFile(path string) ([]byte, error) {
	if v, ok := f.files[path]; ok {
		return []byte(v), nil
	}
	return nil, os.ErrNotExist
}

func (f fakeReader) WriteFile(path string, data []byte) error {
	if f.files == nil {
		f.files = map[string]string{}
	}
	f.files[path] = string(data)
	return nil
}

func (f fakeReader) ListMarkdownFiles(dir string) ([]string, error) {
	var out []string
	prefix := filepath.Clean(dir) + string(os.PathSeparator)
	for path := range f.files {
		clean := filepath.Clean(path)
		if filepath.Dir(clean) == filepath.Clean(dir) && strings.HasPrefix(clean, prefix) && strings.EqualFold(filepath.Ext(clean), ".md") {
			out = append(out, clean)
		}
	}
	return out, nil
}

func (f fakeReader) RemoveFile(path string) error {
	if _, ok := f.files[path]; !ok {
		return os.ErrNotExist
	}
	delete(f.files, path)
	return nil
}

func samplePlan() plans.Plan {
	return plans.Plan{
		Title:            "Improve widget",
		Purpose:          "Make widgets better.",
		Scope:            "In: widget core. Out: gadgets.",
		Constraints:      "Greenfield within the module.",
		NonGoals:         "No gadget changes.",
		DefinitionOfDone: "Tests green; baseline diff exit 0.",
		References: []plans.Reference{
			{Kind: plans.ReferenceCode, Target: "internal/widget/core.go"},
			{Kind: plans.ReferenceReq, Target: "OT-P0-001"},
		},
		RegressionAnchor: plans.RegressionAnchor{
			Strategy: "scenario_baseline", Scenario: "widget-svc", BaselineName: "impl",
			Commands: []string{"git-control-tower baseline diff --scenario widget-svc --name impl --wait"},
		},
		RelevantContext: []plans.RelevantContextItem{{
			Kind:         plans.RelevantContextSearch,
			Scope:        plans.RelevantContextScopeGlobal,
			Label:        "Recall prior widget work",
			Reason:       "Recover prior implementation context.",
			Command:      "search-hub query widget --type record,doc",
			Required:     true,
			RepeatPolicy: plans.RelevantContextOnResume,
			Source:       plans.RelevantContextSourceAuthored,
			Status:       plans.RelevantContextStatusReady,
		}},
		Phases: []plans.Phase{
			{
				Title: "Anchor", Intent: "Capture baseline", Acceptance: "Anchor recorded", Status: plans.PhaseStatusTodo,
				RequiredReading: []string{"docs/TESTING.md"}, Reminders: []string{"Never git stash"},
				RelevantContext: []plans.RelevantContextItem{{
					Kind:         plans.RelevantContextDoc,
					Scope:        plans.RelevantContextScopePhase,
					Label:        "Testing protocol",
					Target:       "docs/TESTING.md",
					Required:     true,
					RepeatPolicy: plans.RelevantContextPhaseEntry,
					Source:       plans.RelevantContextSourceAuthored,
					Status:       plans.RelevantContextStatusReady,
				}},
			},
			{
				Title: "Implement", Intent: "Build it", Acceptance: "Builds", Status: plans.PhaseStatusTodo,
				References: []plans.Reference{{Kind: plans.ReferenceCode, Target: "internal/widget/new.go", Future: true}},
			},
		},
	}
}

// [REQ:PM-STORE-001]
func TestRepositorySaveGetRoundTrip(t *testing.T) {
	d, clk := newDB(t)
	repo := plans.NewSQLiteRepository(d, clk)
	ctx := context.Background()

	in := samplePlan()
	in.ID = "plan-1"
	in.Slug = "improve-widget"
	in.Status = plans.PlanStatusDraft
	in.ContentHash = "abc"
	require.NoError(t, repo.Save(ctx, in))

	got, ok, err := repo.Get(ctx, "plan-1")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, in.Title, got.Title)
	require.Equal(t, in.Purpose, got.Purpose)
	require.Len(t, got.Phases, 2)
	require.Equal(t, "Anchor", got.Phases[0].Title)
	require.Len(t, got.References, 2)
	require.True(t, got.Phases[1].References[0].Future)
	require.Equal(t, "scenario_baseline", got.RegressionAnchor.Strategy)
	require.Len(t, got.RelevantContext, 1)
	require.Equal(t, plans.RelevantContextSearch, got.RelevantContext[0].Kind)
	require.Equal(t, "Testing protocol", got.Phases[0].RelevantContext[0].Label)

	// Resolves by slug too.
	bySlug, ok, err := repo.Get(ctx, "improve-widget")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "plan-1", bySlug.ID)

	// Missing => ok=false, no error.
	_, ok, err = repo.Get(ctx, "nope")
	require.NoError(t, err)
	require.False(t, ok)
}

func TestServiceCreateComputesIdentityFields(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()

	p, err := svc.Create(ctx, samplePlan())
	require.NoError(t, err)
	require.NotEmpty(t, p.ID)
	require.Equal(t, "improve-widget", p.Slug)
	require.Equal(t, plans.PlanStatusDraft, p.Status) // all phases todo
	require.NotEmpty(t, p.ContentHash)
	require.NotEmpty(t, p.CreatedAt)
	for _, ph := range p.Phases {
		require.NotEmpty(t, ph.ID, "phases get ids assigned")
	}

	// Slug uniqueness disambiguation.
	p2, err := svc.Create(ctx, samplePlan())
	require.NoError(t, err)
	require.Equal(t, "improve-widget-2", p2.Slug)

	// Empty title rejected.
	_, err = svc.Create(ctx, plans.Plan{})
	require.Error(t, err)
}

func TestContentHashStableAndSensitive(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, samplePlan())
	require.NoError(t, err)

	// Same authored content => same hash (re-create a twin, compare hash of authored payload).
	twin := samplePlan()
	b, err := svc.Create(ctx, twin)
	require.NoError(t, err)
	require.Equal(t, a.ContentHash, b.ContentHash, "identical authored content hashes identically")

	// Mutating authored content changes the hash.
	updated := a
	updated.Purpose = "Different purpose"
	got, err := svc.Update(ctx, updated)
	require.NoError(t, err)
	require.NotEqual(t, a.ContentHash, got.ContentHash)

	updated = a
	updated.RelevantContext[0].Reason = "Different context reason"
	got, err = svc.Update(ctx, updated)
	require.NoError(t, err)
	require.NotEqual(t, a.ContentHash, got.ContentHash)
}

// [REQ:PM-STORE-002]
func TestStatusTransitionLegality(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()
	p, err := svc.Create(ctx, samplePlan())
	require.NoError(t, err)
	require.Equal(t, plans.PlanStatusDraft, p.Status)

	// Move first phase to active => plan active.
	ph0 := p.Phases[0]
	ph0.Status = plans.PhaseStatusActive
	p, err = svc.UpdatePhase(ctx, p.ID, ph0)
	require.NoError(t, err)
	require.Equal(t, plans.PlanStatusActive, p.Status)

	// All phases done => plan complete.
	for _, ph := range p.Phases {
		ph.Status = plans.PhaseStatusDone
		p, err = svc.UpdatePhase(ctx, p.ID, ph)
		require.NoError(t, err)
	}
	require.Equal(t, plans.PlanStatusComplete, p.Status)

	// Archive is terminal and survives a phase update.
	p, err = svc.Archive(ctx, p.ID, plans.WorkspaceScope{})
	require.NoError(t, err)
	require.Equal(t, plans.PlanStatusArchived, p.Status)
	ph := p.Phases[0]
	ph.Status = plans.PhaseStatusActive
	p, err = svc.UpdatePhase(ctx, p.ID, ph)
	require.NoError(t, err)
	require.Equal(t, plans.PlanStatusArchived, p.Status, "archived is sticky")
}

// TestUpdatePreservesPhaseIdentityOnReKey pins the orphan guard: an Update whose
// incoming phases dropped their IDs (a caller round-tripping through a surface
// that doesn't echo IDs) must NOT re-key the phases — they're matched back to the
// existing phases by title so executions/decisions/findings that reference those
// phase ids are not orphaned.
func TestUpdatePreservesPhaseIdentityOnReKey(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()
	p, err := svc.Create(ctx, samplePlan())
	require.NoError(t, err)
	require.Len(t, p.Phases, 2)
	anchorID, implementID := p.Phases[0].ID, p.Phases[1].ID
	require.NotEmpty(t, anchorID)

	updated := p
	updated.Phases = append([]plans.Phase(nil), p.Phases...)
	for i := range updated.Phases {
		updated.Phases[i].ID = "" // simulate a round-trip that dropped phase IDs
	}
	got, err := svc.Update(ctx, updated)
	require.NoError(t, err)
	require.Len(t, got.Phases, 2)
	require.Equal(t, anchorID, got.Phases[0].ID, "phase identity preserved by title match, not re-keyed")
	require.Equal(t, implementID, got.Phases[1].ID)
}

// TestUpdatePreservesImportProvenance proves a normal authored-field update that
// omits governance lineage does not drop it: import provenance and preserved
// legacy sections survive when the caller does not echo them.
func TestUpdatePreservesImportProvenance(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()
	seed := samplePlan()
	seed.ImportProvenance = &plans.ImportProvenance{
		SourcePath:     "docs/plans/legacy.md",
		OriginalFormat: plans.OriginalFormatLegacyMarkdown,
		Note:           "Adopted from legacy 13-section markdown.",
	}
	seed.PreservedLegacySections = []plans.LegacySection{{
		Heading:            "Open Questions",
		Content:            "Q: do we need X?",
		PreservationReason: plans.PreservationReasonUnmapped,
	}}
	p, err := svc.Create(ctx, seed)
	require.NoError(t, err)
	require.NotNil(t, p.ImportProvenance)

	// A normal update that edits an authored field but omits provenance fields.
	updated := p
	updated.ImportProvenance = nil
	updated.PreservedLegacySections = nil
	updated.Purpose = "An edited purpose."
	got, err := svc.Update(ctx, updated)
	require.NoError(t, err)
	require.Equal(t, "An edited purpose.", got.Purpose)
	require.NotNil(t, got.ImportProvenance, "import provenance must survive an authored-field update")
	require.Equal(t, "docs/plans/legacy.md", got.ImportProvenance.SourcePath)
	require.Len(t, got.PreservedLegacySections, 1, "preserved legacy sections must survive an authored-field update")
	require.Equal(t, "Open Questions", got.PreservedLegacySections[0].Heading)
}

func TestAddPhaseAppendsAndOrders(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()
	p, err := svc.Create(ctx, samplePlan())
	require.NoError(t, err)

	p, err = svc.AddPhase(ctx, p.ID, plans.Phase{Title: "Validate", Intent: "Run baselines"})
	require.NoError(t, err)
	require.Len(t, p.Phases, 3)
	require.Equal(t, 3, p.Phases[2].Order)
	require.Equal(t, plans.PhaseStatusTodo, p.Phases[2].Status)
}

// [REQ:PM-STORE-001]
func TestRenderMarkdownDeterministic(t *testing.T) {
	p := samplePlan()
	p.Status = plans.PlanStatusActive
	p.ContentHash = "deadbeefcafebabe0000"
	first := plans.RenderMarkdown(p)
	second := plans.RenderMarkdown(p)
	require.Equal(t, first, second, "render is deterministic")

	require.Contains(t, first, "# Improve widget")
	require.Contains(t, first, "## Purpose")
	require.Contains(t, first, "[CODE: internal/widget/core.go]")
	require.Contains(t, first, "[REQ: OT-P0-001]")
	require.Contains(t, first, "### Phase 1 — Anchor")
	require.Contains(t, first, "_(future)_", "future references are annotated")
	require.Contains(t, first, "## Regression Anchor")
}

func TestSupersessionResolution(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()
	oldPlan, err := svc.Create(ctx, samplePlan())
	require.NoError(t, err)
	newPlan, err := svc.Create(ctx, samplePlan())
	require.NoError(t, err)

	superseding, err := svc.LinkSupersession(ctx, newPlan.ID, oldPlan.ID)
	require.NoError(t, err)
	require.Contains(t, superseding.Supersedes, oldPlan.ID)

	// The superseded plan records the reverse edge.
	gotOld, err := svc.Get(ctx, oldPlan.ID, plans.WorkspaceScope{})
	require.NoError(t, err)
	require.Contains(t, gotOld.SupersededBy, newPlan.ID)

	// Graph query returns the edge.
	edges, err := svc.GetGraph(ctx, newPlan.ID)
	require.NoError(t, err)
	require.Len(t, edges, 1)
	require.Equal(t, newPlan.ID, edges[0].FromPlanID)
	require.Equal(t, oldPlan.ID, edges[0].ToPlanID)
	require.Equal(t, "supersedes", edges[0].Kind)

	// Whole-graph query (empty id) also returns it.
	all, err := svc.GetGraph(ctx, "")
	require.NoError(t, err)
	require.Len(t, all, 1)
}

// [REQ:PM-GRAPH-001]
func TestContentHashDerivesSupersession(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()
	oldPlan, err := svc.Create(ctx, samplePlan())
	require.NoError(t, err)

	newPlan, err := svc.Create(ctx, samplePlan())
	require.NoError(t, err)
	require.Equal(t, oldPlan.ContentHash, newPlan.ContentHash)
	require.Contains(t, newPlan.Supersedes, oldPlan.ID, "duplicate authored content records the newer plan as superseding the older one")

	gotOld, err := svc.Get(ctx, oldPlan.ID, plans.WorkspaceScope{})
	require.NoError(t, err)
	require.Contains(t, gotOld.SupersededBy, newPlan.ID)

	edges, err := svc.GetGraph(ctx, newPlan.ID)
	require.NoError(t, err)
	require.Len(t, edges, 1)
	require.Equal(t, plans.EdgeKindSupersedes, edges[0].Kind)
	require.Equal(t, newPlan.ID, edges[0].FromPlanID)
	require.Equal(t, oldPlan.ID, edges[0].ToPlanID)
}

// [REQ:PM-GRAPH-001]
func TestLinkDependencyCreatesGraphEdge(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()
	dependency, err := svc.Create(ctx, samplePlan())
	require.NoError(t, err)
	dependingInput := samplePlan()
	dependingInput.Title = "Downstream widget"
	dependingInput.Purpose = "Build on the first plan."
	depending, err := svc.Create(ctx, dependingInput)
	require.NoError(t, err)

	got, err := svc.LinkDependency(ctx, depending.ID, dependency.ID)
	require.NoError(t, err)
	require.Equal(t, depending.ID, got.ID)

	edges, err := svc.GetGraph(ctx, depending.ID)
	require.NoError(t, err)
	require.Len(t, edges, 1)
	require.Equal(t, depending.ID, edges[0].FromPlanID)
	require.Equal(t, dependency.ID, edges[0].ToPlanID)
	require.Equal(t, plans.EdgeKindDependsOn, edges[0].Kind)

	_, err = svc.LinkDependency(ctx, depending.ID, depending.ID)
	require.Error(t, err)
}

func TestListFilters(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()
	a, err := svc.Create(ctx, samplePlan())
	require.NoError(t, err)
	_, err = svc.Create(ctx, samplePlan())
	require.NoError(t, err)
	_, err = svc.Archive(ctx, a.ID, plans.WorkspaceScope{})
	require.NoError(t, err)

	// Default list excludes archived.
	active, err := svc.List(ctx, plans.ListFilter{})
	require.NoError(t, err)
	require.Len(t, active, 1)

	// IncludeArchived returns all.
	all, err := svc.List(ctx, plans.ListFilter{IncludeArchived: true})
	require.NoError(t, err)
	require.Len(t, all, 2)

	// Status filter.
	archived, err := svc.List(ctx, plans.ListFilter{Status: plans.PlanStatusArchived})
	require.NoError(t, err)
	require.Len(t, archived, 1)
}

// [REQ:PM-GRAPH-001]
func TestTemplates(t *testing.T) {
	svc, _ := newService(t)
	ctx := context.Background()
	templates, err := svc.ListTemplates(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, templates)

	p, err := svc.CreateFromTemplate(ctx, "cli", "My CLI feature", "")
	require.NoError(t, err)
	require.Equal(t, "my-cli-feature", p.Slug)
	require.NotEmpty(t, p.Phases)
	require.Equal(t, "Contract", p.Phases[0].Title)

	_, err = svc.CreateFromTemplate(ctx, "does-not-exist", "x", "")
	require.Error(t, err)
}

func TestParsePlanMarkdown(t *testing.T) {
	md := strings.Join([]string{
		"# Adopt the gizmo",
		"",
		"## Purpose",
		"Bring the gizmo under management.",
		"",
		"## Scope",
		"In: gizmo. Out: doohickey. See [CODE: internal/gizmo/gizmo.go] and [REQ: OT-P0-002].",
		"",
		"## Phases",
		"",
		"### Phase 1 — Wire it",
		"- Intent: Connect the gizmo",
		"- Acceptance: It connects",
		"- Status: done",
		"",
		"### Phase 2 — Validate",
		"- Intent: Check it",
		"- Status: todo",
		"",
	}, "\n")

	p, err := plans.ParsePlanMarkdown(md)
	require.NoError(t, err)
	require.Equal(t, "Adopt the gizmo", p.Title)
	require.Contains(t, p.Purpose, "under management")
	require.Len(t, p.References, 2)
	require.Equal(t, plans.ReferenceCode, p.References[0].Kind)
	require.Equal(t, "internal/gizmo/gizmo.go", p.References[0].Target)
	require.Equal(t, plans.ReferenceReq, p.References[1].Kind)
	require.Len(t, p.Phases, 2)
	require.Equal(t, "Wire it", p.Phases[0].Title)
	require.Equal(t, plans.PhaseStatusDone, p.Phases[0].Status)
	require.Equal(t, "Connect the gizmo", p.Phases[0].Intent)

	// Empty markdown / no title are rejected.
	_, err = plans.ParsePlanMarkdown("")
	require.Error(t, err)
	_, err = plans.ParsePlanMarkdown("no heading here")
	require.Error(t, err)
}

func TestParsePlanMarkdownRejectsMalformedMachineMarkup(t *testing.T) {
	cases := []struct {
		name string
		md   string
	}{
		{
			name: "empty_reference_target",
			md:   "# Bad\n\n## References\n\n[CODE:]\n",
		},
		{
			name: "malformed_phase_heading",
			md:   "# Bad\n\n## Phases\n\n### Phase One - Missing numeric order\n- Intent: x\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := plans.ParsePlanMarkdown(tc.md)
			require.Error(t, err)
			var invalid plans.ErrInvalidPlan
			require.True(t, errors.As(err, &invalid))
		})
	}
}

// [REQ:PM-STORE-001]
func TestImportAndMigrate(t *testing.T) {
	d, clk := newDB(t)
	reader := fakeReader{files: map[string]string{
		"docs/plans/foo.md": "# Foo plan\n\n## Purpose\nDo foo.\n\n### Phase 1 — Start\n- Intent: begin\n",
	}}
	svc := plans.NewService(plans.Deps{Repo: plans.NewSQLiteRepository(d, clk), Clock: clk, Reader: reader})
	ctx := context.Background()

	// Import from a fallback source path.
	imported, err := svc.Import(ctx, "docs/plans/foo.md", "", "", "", plans.WorkspaceScope{})
	require.NoError(t, err)
	require.Equal(t, "Foo plan", imported.Title)
	require.Equal(t, "foo-plan", imported.Slug)
	require.Len(t, imported.Phases, 1)

	// Import inline markdown (no reader needed).
	inline, err := svc.Import(ctx, "", "# Bar plan\n\n## Purpose\nDo bar.\n", "", "", plans.WorkspaceScope{})
	require.NoError(t, err)
	require.Equal(t, "Bar plan", inline.Title)

	overridden, err := svc.Import(ctx, "", "# Original\n\n## Purpose\nDo original.\n", "Override Title", "override-slug", plans.WorkspaceScope{})
	require.NoError(t, err)
	require.Equal(t, "Override Title", overridden.Title)
	require.Equal(t, "override-slug", overridden.Slug)

	// Migrate is idempotent and returns the canonical record.
	migrated, err := svc.Migrate(ctx, imported.ID)
	require.NoError(t, err)
	require.Equal(t, imported.ID, migrated.ID)

	// Importing with neither markdown nor a resolvable path errors.
	_, err = svc.Import(ctx, "", "", "", "", plans.WorkspaceScope{})
	require.Error(t, err)
}

func TestImportRelativeSourceUsesWorkspaceRoot(t *testing.T) {
	d, clk := newDB(t)
	workspaceRoot := validWorkspaceRoot(t)
	source := filepath.Join(workspaceRoot, "docs", "plans", "foo.md")
	reader := fakeReader{files: map[string]string{
		source: "# Foo plan\n\n## Purpose\nDo foo.\n",
	}}
	svc := plans.NewService(plans.Deps{Repo: plans.NewSQLiteRepository(d, clk), Clock: clk, Reader: reader})
	ctx := context.Background()

	imported, err := svc.Import(ctx, "docs/plans/foo.md", "", "", "", plans.WorkspaceScope{Root: workspaceRoot})
	require.NoError(t, err)
	require.Equal(t, "Foo plan", imported.Title)
	require.NotNil(t, imported.ImportProvenance)
	require.Equal(t, source, imported.ImportProvenance.SourcePath)
	require.Equal(t, workspaceRoot, imported.ImportProvenance.WorkspaceRoot)

	rendered, err := svc.Render(ctx, imported.ID, plans.WorkspaceScope{}, plans.RenderOptions{})
	require.NoError(t, err)
	require.Contains(t, rendered.Markdown, "- Workspace root: `"+workspaceRoot+"`")

	roundTripped, err := plans.ParsePlanMarkdown(rendered.Markdown)
	require.NoError(t, err)
	require.NotNil(t, roundTripped.ImportProvenance)
	require.Equal(t, workspaceRoot, roundTripped.ImportProvenance.WorkspaceRoot)
}

func TestImportMigratesLegacyRequiredReadingToRelevantContext(t *testing.T) {
	d, clk := newDB(t)
	reader := fakeReader{files: map[string]string{
		"docs/plans/legacy-context.md": strings.Join([]string{
			"# Legacy Context",
			"",
			"## Required Reading",
			"",
			"- prompt-manager skill read api-steer",
			"- docs/concepts/PLAN-MODEL.md",
			"",
			"## Phases",
			"",
			"### Phase 1 — Implement",
			"- Intent: preserve setup",
			"",
			"**Required Reading:**",
			"- [REQ: PM-CTX-001]",
			"- cli: vrooli scenario requirements validate plan-manager",
			"",
		}, "\n"),
	}}
	svc := plans.NewService(plans.Deps{Repo: plans.NewSQLiteRepository(d, clk), Clock: clk, Reader: reader})
	ctx := context.Background()

	imported, err := svc.Import(ctx, "docs/plans/legacy-context.md", "", "", "", plans.WorkspaceScope{})
	require.NoError(t, err)
	require.Len(t, imported.RelevantContext, 2)
	require.Equal(t, plans.RelevantContextSkill, imported.RelevantContext[0].Kind)
	require.Equal(t, plans.RelevantContextSourceMigrated, imported.RelevantContext[0].Source)
	require.Equal(t, plans.RelevantContextDoc, imported.RelevantContext[1].Kind)
	require.Len(t, imported.Phases, 1)
	require.Equal(t, []string{"[REQ: PM-CTX-001]", "cli: vrooli scenario requirements validate plan-manager"}, imported.Phases[0].RequiredReading)
	require.Len(t, imported.Phases[0].RelevantContext, 2)
	require.Equal(t, plans.RelevantContextReqRef, imported.Phases[0].RelevantContext[0].Kind)
	require.Equal(t, "PM-CTX-001", imported.Phases[0].RelevantContext[0].Target)
	require.Equal(t, plans.RelevantContextCommand, imported.Phases[0].RelevantContext[1].Kind)

	rendered, err := svc.Render(ctx, imported.ID, plans.WorkspaceScope{}, plans.RenderOptions{})
	require.NoError(t, err)
	require.Contains(t, rendered.Markdown, "## Global Execution Setup")
	require.Contains(t, rendered.Markdown, "**Phase Context Setup:**")
	require.Contains(t, rendered.Markdown, "_(required, migrated)_")
	require.NotContains(t, rendered.Markdown, "## Required Reading")
	require.NotContains(t, rendered.Markdown, "**Required Reading:**")
}

func TestMigrateImportsIndexedFallbackPlan(t *testing.T) {
	d, clk := newDB(t)
	reader := fakeReader{files: map[string]string{
		"docs/plans/_index.json": `{
			"version": 1,
			"plans": [
				{
					"id": "legacy-plan",
					"title": "Legacy Plan",
					"slug": "legacy-plan",
					"path": "docs/plans/legacy-plan.md",
					"created_at": "2026-06-25T12:00:00Z",
					"updated_at": "2026-06-25T12:00:00Z",
					"archived": false
				}
			]
		}`,
		"docs/plans/legacy-plan.md": "# Legacy Plan\n\n## Purpose\nAdopt this from the old store.\n\n## Phases\n\n### Phase 1 — Start\n- Intent: migrate\n",
	}}
	svc := plans.NewService(plans.Deps{Repo: plans.NewSQLiteRepository(d, clk), Clock: clk, Reader: reader})
	ctx := context.Background()

	migrated, err := svc.Migrate(ctx, "legacy-plan")
	require.NoError(t, err)
	require.Equal(t, "Legacy Plan", migrated.Title)
	require.Equal(t, "legacy-plan", migrated.Slug)
	require.Len(t, migrated.Phases, 1)

	got, err := svc.Get(ctx, migrated.ID, plans.WorkspaceScope{})
	require.NoError(t, err)
	require.Equal(t, migrated.ID, got.ID)
	require.Equal(t, "Adopt this from the old store.", got.Purpose)
}
