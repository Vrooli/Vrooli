package plans

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	testkitgo "github.com/vrooli/repo-contract-go/repocontracttest"
)

type fakePlanManager struct {
	err       error
	list      []PlanRecord
	gotListWS WorkspaceScope
	gotGetWS  WorkspaceScope
	rendered  RenderedPlan
}

type fakeMirrorReader struct {
	list      []PlanRecord
	content   string
	err       error
	gotRef    string
	gotListWS WorkspaceScope
	gotFindWS WorkspaceScope
	gotReadWS WorkspaceScope
}

func (f *fakeMirrorReader) List(_ context.Context, workspace WorkspaceScope, _ bool) ([]PlanRecord, error) {
	f.gotListWS = workspace
	if f.err != nil {
		return nil, f.err
	}
	return f.list, nil
}

func (f *fakeMirrorReader) Find(_ context.Context, workspace WorkspaceScope, ref string) (PlanRecord, error) {
	f.gotFindWS = workspace
	f.gotRef = ref
	if f.err != nil {
		return PlanRecord{}, f.err
	}
	for _, p := range f.list {
		if p.ID == ref || p.Slug == ref {
			return p, nil
		}
	}
	return PlanRecord{}, errors.New("not found")
}

func (f *fakeMirrorReader) Read(ctx context.Context, workspace WorkspaceScope, ref string) (PlanRecord, string, error) {
	f.gotReadWS = workspace
	p, err := f.Find(ctx, workspace, ref)
	if err != nil {
		return PlanRecord{}, "", err
	}
	return p, f.content, nil
}

func (f *fakePlanManager) ListPlans(_ context.Context, workspace WorkspaceScope, _ bool) ([]PlanRecord, error) {
	f.gotListWS = workspace
	if f.err != nil {
		return nil, f.err
	}
	return f.list, nil
}

func (f *fakePlanManager) GetPlan(_ context.Context, workspace WorkspaceScope, ref string) (PlanRecord, error) {
	f.gotGetWS = workspace
	if f.err != nil {
		return PlanRecord{}, f.err
	}
	for _, p := range f.list {
		if p.ID == ref || p.Slug == ref {
			return p, nil
		}
	}
	if f.rendered.Plan.ID == ref || f.rendered.Plan.Slug == ref {
		return f.rendered.Plan, nil
	}
	return PlanRecord{}, errors.New("not found")
}

func (f *fakePlanManager) RenderMarkdown(context.Context, WorkspaceScope, string) (RenderedPlan, error) {
	if f.err != nil {
		return RenderedPlan{}, f.err
	}
	return f.rendered, nil
}

func newPlanWorkspace(t *testing.T) string {
	t.Helper()
	fixture := testkitgo.NewRepoFixture(t)
	fixture.WriteRepoContract(t)
	return fixture.Root
}

func TestServiceListUsesWorkspaceScope(t *testing.T) {
	workspace := newPlanWorkspace(t)
	pm := &fakePlanManager{}
	svc := Service{Root: newPlanWorkspace(t), Home: t.TempDir(), PlanManager: &fakePlanManager{}}
	svc.PlanManager = pm

	if _, err := svc.List(ListRequest{Workspace: workspace}); err != nil {
		t.Fatalf("List: %v", err)
	}
	if pm.gotListWS.Root != workspace {
		t.Fatalf("workspace root = %q, want %q", pm.gotListWS.Root, workspace)
	}
}

func TestClassifyPlanManagerStatusUsesConnectErrorCode(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   error
	}{
		{name: "not found", status: 400, body: `{"code":"not_found","message":"missing"}`, want: ErrPlanManagerNotFound},
		{name: "invalid", status: 500, body: `{"code":"invalid_argument","message":"bad"}`, want: ErrPlanManagerInvalid},
		{name: "conflict", status: 500, body: `{"code":"already_exists","message":"dup"}`, want: ErrPlanManagerConflict},
		{name: "unavailable", status: 503, body: `{"code":"unavailable","message":"down"}`, want: ErrPlanManagerUnavailable},
		{name: "timeout", status: 504, body: `{"code":"deadline_exceeded","message":"slow"}`, want: ErrPlanManagerTimeout},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := classifyPlanManagerStatus("http://plan-manager", tt.status, tt.body)
			if !errors.Is(err, tt.want) {
				t.Fatalf("classifyPlanManagerStatus() = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestServiceReadFallbackUsesMirrorFilesWhenPlanManagerUnavailable(t *testing.T) {
	home := t.TempDir()
	plansDir := filepath.Join(home, ".vrooli", "plans")
	if err := os.MkdirAll(plansDir, 0o755); err != nil {
		t.Fatalf("mkdir plans dir: %v", err)
	}
	path := filepath.Join(plansDir, "fallback-plan.md")
	if err := os.WriteFile(path, []byte("# Fallback Plan\n\nReadable while API is down.\n"), 0o644); err != nil {
		t.Fatalf("write mirror: %v", err)
	}
	svc := Service{
		Home:        home,
		PlanManager: &fakePlanManager{err: ErrPlanManagerUnavailable},
		Now:         func() time.Time { return time.Date(2026, 6, 30, 12, 0, 0, 0, time.UTC) },
	}

	listed, err := svc.List(ListRequest{})
	if err != nil {
		t.Fatalf("List fallback: %v", err)
	}
	if len(listed.Plans) != 1 || listed.Plans[0].Slug != "fallback-plan" {
		t.Fatalf("fallback list = %#v", listed.Plans)
	}
	if !listed.Degraded || listed.Source != readSourceMirror || !strings.Contains(listed.Warning, "Plan Manager unavailable") {
		t.Fatalf("fallback list provenance = source %q degraded %v warning %q", listed.Source, listed.Degraded, listed.Warning)
	}
	shown, err := svc.Show(ShowRequest{Ref: "fallback-plan"})
	if err != nil {
		t.Fatalf("Show fallback: %v", err)
	}
	if !strings.Contains(shown.Content, "Readable while API is down.") {
		t.Fatalf("shown content = %q", shown.Content)
	}
	if !shown.Degraded || shown.Source != readSourceMirror {
		t.Fatalf("fallback show provenance = source %q degraded %v", shown.Source, shown.Degraded)
	}
	pathed, err := svc.Path(ShowRequest{Ref: "fallback-plan"})
	if err != nil {
		t.Fatalf("Path fallback: %v", err)
	}
	if pathed.Path != path {
		t.Fatalf("fallback path = %q, want %q", pathed.Path, path)
	}
	if !pathed.Degraded || pathed.Source != readSourceMirror {
		t.Fatalf("fallback path provenance = source %q degraded %v", pathed.Source, pathed.Degraded)
	}
}

func TestServiceReadFallbackFiltersMirrorsByWorkspace(t *testing.T) {
	home := t.TempDir()
	workspace := newPlanWorkspace(t)
	otherWorkspace := newPlanWorkspace(t)
	plansDir := filepath.Join(home, ".vrooli", "plans")
	if err := os.MkdirAll(plansDir, 0o755); err != nil {
		t.Fatalf("mkdir plans dir: %v", err)
	}
	matchingPath := filepath.Join(plansDir, "matching.md")
	otherPath := filepath.Join(plansDir, "other.md")
	unscopedPath := filepath.Join(plansDir, "unscoped.md")
	for path, content := range map[string]string{
		matchingPath: "# Matching\n",
		otherPath:    "# Other\n",
		unscopedPath: "# Unscoped\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
	}
	writeJSONFile(t, filepath.Join(plansDir, "_index.json"), indexFile{Version: 2, Plans: []PlanRecord{
		{ID: "match", Slug: "matching", Title: "Matching", Path: matchingPath, WorkspaceRoot: workspace},
		{ID: "other", Slug: "other", Title: "Other", Path: otherPath, WorkspaceRoot: otherWorkspace},
		{ID: "unscoped", Slug: "unscoped", Title: "Unscoped", Path: unscopedPath},
	}})
	svc := Service{Home: home, PlanManager: &fakePlanManager{err: ErrPlanManagerUnavailable}}

	listed, err := svc.List(ListRequest{Workspace: workspace})
	if err != nil {
		t.Fatalf("List fallback: %v", err)
	}
	if len(listed.Plans) != 1 || listed.Plans[0].Slug != "matching" {
		t.Fatalf("scoped fallback list = %#v", listed.Plans)
	}
	shown, err := svc.Show(ShowRequest{Ref: "matching", Workspace: workspace})
	if err != nil {
		t.Fatalf("Show matching fallback: %v", err)
	}
	if shown.Content != "# Matching\n" {
		t.Fatalf("matching content = %q", shown.Content)
	}
	if _, err := svc.Show(ShowRequest{Ref: "unscoped", Workspace: workspace}); err == nil {
		t.Fatalf("Show unscoped fallback succeeded, want scoped miss")
	}
	if _, err := svc.Show(ShowRequest{Ref: "other", Workspace: workspace}); err == nil {
		t.Fatalf("Show other fallback succeeded, want scoped miss")
	}
}

// TestServiceReadFallbackTreatsUnversionedIndexAsAbsent: an _index.json with no
// version was not written by Plan Manager's mirror projection; the fallback
// ignores it (no in-place normalization) and rebuilds from the mirror files.
func TestServiceReadFallbackTreatsUnversionedIndexAsAbsent(t *testing.T) {
	home := t.TempDir()
	plansDir := filepath.Join(home, ".vrooli", "plans")
	if err := os.MkdirAll(plansDir, 0o755); err != nil {
		t.Fatalf("mkdir plans dir: %v", err)
	}
	planPath := filepath.Join(plansDir, "real-plan.md")
	if err := os.WriteFile(planPath, []byte("# Real Plan\n"), 0o644); err != nil {
		t.Fatalf("write plan: %v", err)
	}
	// Unversioned index pointing at a phantom plan: it must be ignored, not
	// normalized to version 1 and trusted.
	writeJSONFile(t, filepath.Join(plansDir, "_index.json"), indexFile{Plans: []PlanRecord{
		{ID: "phantom", Slug: "phantom", Title: "Phantom", Path: filepath.Join(plansDir, "phantom.md")},
	}})
	svc := Service{Home: home, PlanManager: &fakePlanManager{err: ErrPlanManagerUnavailable}}

	listed, err := svc.List(ListRequest{})
	if err != nil {
		t.Fatalf("List fallback: %v", err)
	}
	if len(listed.Plans) != 1 || listed.Plans[0].Slug != "real-plan" {
		t.Fatalf("unversioned index must take the scan-rebuild path, got %#v", listed.Plans)
	}
}

func TestServiceReadFallbackUsesInjectedMirrorReader(t *testing.T) {
	mirror := &fakeMirrorReader{
		list:    []PlanRecord{{ID: "plan-1", Slug: "fallback-plan", Title: "Fallback Plan", Path: "/mirror/fallback-plan.md"}},
		content: "# Fallback Plan\n",
	}
	svc := Service{
		PlanManager:  &fakePlanManager{err: ErrPlanManagerTimeout},
		MirrorReader: mirror,
	}

	shown, err := svc.Show(ShowRequest{Ref: "fallback-plan"})
	if err != nil {
		t.Fatalf("Show fallback: %v", err)
	}
	if mirror.gotRef != "fallback-plan" {
		t.Fatalf("mirror ref = %q, want fallback-plan", mirror.gotRef)
	}
	if shown.Content != "# Fallback Plan\n" || shown.Source != readSourceMirror || !shown.Degraded {
		t.Fatalf("shown = %#v", shown)
	}
}

func writeJSONFile(t *testing.T, path string, value any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write json: %v", err)
	}
}

func TestServiceShowPrefersPlanManagerMetadataAndRender(t *testing.T) {
	pm := &fakePlanManager{
		list: []PlanRecord{{
			ID: "plan-1", Slug: "plan-one", Title: "Plan One", Path: "/mirror/plan-one.md",
		}},
		rendered: RenderedPlan{
			Plan:    PlanRecord{Path: "/mirror/plan-one.md"},
			Content: "# Plan One\n",
		},
	}
	svc := Service{Home: t.TempDir(), PlanManager: pm}

	shown, err := svc.Show(ShowRequest{Ref: "plan-one"})
	if err != nil {
		t.Fatalf("Show: %v", err)
	}
	if shown.Plan.Title != "Plan One" || shown.Plan.Path != "/mirror/plan-one.md" {
		t.Fatalf("shown plan = %#v", shown.Plan)
	}
	if shown.Content != "# Plan One\n" {
		t.Fatalf("content = %q", shown.Content)
	}
}

func TestServiceReadDoesNotFallbackForCanonicalPlanManagerErrors(t *testing.T) {
	home := t.TempDir()
	plansDir := filepath.Join(home, ".vrooli", "plans")
	if err := os.MkdirAll(plansDir, 0o755); err != nil {
		t.Fatalf("mkdir plans dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(plansDir, "stale-plan.md"), []byte("# Stale Plan\n"), 0o644); err != nil {
		t.Fatalf("write mirror: %v", err)
	}
	svc := Service{Home: home, PlanManager: &fakePlanManager{err: ErrPlanManagerNotFound}}

	if _, err := svc.Show(ShowRequest{Ref: "stale-plan"}); !errors.Is(err, ErrPlanManagerNotFound) {
		t.Fatalf("Show err = %v, want Plan Manager not found", err)
	}
	if _, err := svc.Path(ShowRequest{Ref: "stale-plan"}); !errors.Is(err, ErrPlanManagerNotFound) {
		t.Fatalf("Path err = %v, want Plan Manager not found", err)
	}
}
