package project_files //nolint:revive

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/browser-automation-studio/database"
	project_filesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/project_files"
	project_filesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/project_files/project_filesconnect"
)

// ---------------------------------------------------------------------------
// fakes
// ---------------------------------------------------------------------------

type fakeRepo struct {
	mu          sync.Mutex
	projects    map[uuid.UUID]*database.ProjectIndex
	workflows   map[uuid.UUID][]*database.WorkflowIndex
	assets      map[uuid.UUID][]*database.AssetIndex
	createErr   error
	createCalls int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		projects:  map[uuid.UUID]*database.ProjectIndex{},
		workflows: map[uuid.UUID][]*database.WorkflowIndex{},
		assets:    map[uuid.UUID][]*database.AssetIndex{},
	}
}

func (f *fakeRepo) GetProject(_ context.Context, id uuid.UUID) (*database.ProjectIndex, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.projects[id]
	if !ok {
		return nil, database.ErrNotFound
	}
	return p, nil
}

func (f *fakeRepo) ListWorkflowsByProject(_ context.Context, id uuid.UUID, _, _ int) ([]*database.WorkflowIndex, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.workflows[id], nil
}

func (f *fakeRepo) ListAssetsByProject(_ context.Context, id uuid.UUID, _, _ int) ([]*database.AssetIndex, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.assets[id], nil
}

func (f *fakeRepo) CreateWorkflow(_ context.Context, w *database.WorkflowIndex) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.createCalls++
	if f.createErr != nil {
		return f.createErr
	}
	if w.ProjectID == nil {
		return errors.New("project id required")
	}
	f.workflows[*w.ProjectID] = append(f.workflows[*w.ProjectID], w)
	return nil
}

type fakeCatalog struct {
	mu        sync.Mutex
	calls     int
	returnErr error
}

func (f *fakeCatalog) SyncProjectWorkflows(_ context.Context, _ uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.returnErr
}

type fakeOS struct {
	mu          sync.Mutex
	openCalls   []string
	revealCalls []string
	openErr     error
	revealErr   error
}

func (f *fakeOS) OpenFolder(p string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.openCalls = append(f.openCalls, p)
	return f.openErr
}

func (f *fakeOS) RevealInFileManager(p string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.revealCalls = append(f.revealCalls, p)
	return f.revealErr
}

// ---------------------------------------------------------------------------
// test client
// ---------------------------------------------------------------------------

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) { w.t.Log(string(p)); return len(p), nil }

type clientDeps struct {
	repo    ProjectRepo
	catalog Catalog
	osi     OSIntegration
}

func newTestClient(t *testing.T, d clientDeps) project_filesconnect.ProjectFilesServiceClient {
	t.Helper()
	logger := logrus.New()
	logger.SetOutput(testWriter{t})
	if d.osi == nil {
		d.osi = &fakeOS{}
	}
	mount := Module(Deps{Repo: d.repo, Catalog: d.catalog, OS: d.osi, Logger: logger})
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return project_filesconnect.NewProjectFilesServiceClient(srv.Client(), srv.URL)
}

func makeProject(t *testing.T, repo *fakeRepo) (uuid.UUID, string) {
	t.Helper()
	id := uuid.New()
	dir := t.TempDir()
	repo.projects[id] = &database.ProjectIndex{
		ID:         id,
		Name:       "test",
		FolderPath: dir,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	return id, dir
}

// ---------------------------------------------------------------------------
// helper-function tests
// ---------------------------------------------------------------------------

func TestNormalizeProjectRelPath(t *testing.T) {
	cases := []struct {
		in, want string
		ok       bool
	}{
		{"", "", false},
		{".", "", false},
		{"workflows/a.json", "workflows/a.json", true},
		{"/workflows/a.json", "workflows/a.json", true},
		{"workflows\\a.json", "workflows/a.json", true},
		{"../escape", "", false},
		{"  workflows/a.json  ", "workflows/a.json", true},
	}
	for _, c := range cases {
		got, ok := normalizeProjectRelPath(c.in)
		if ok != c.ok {
			t.Errorf("normalize(%q) ok=%v want %v", c.in, ok, c.ok)
		}
		if ok && got != c.want {
			t.Errorf("normalize(%q) = %q want %q", c.in, got, c.want)
		}
	}
}

func TestSafeJoinProjectPath(t *testing.T) {
	root := "/tmp/project"
	if _, err := safeJoinProjectPath("", "x"); err == nil {
		t.Fatal("empty root must error")
	}
	if _, err := safeJoinProjectPath(root, "../escape"); err == nil {
		t.Fatal("traversal must error")
	}
	got, err := safeJoinProjectPath(root, "a/b.json")
	require.NoError(t, err)
	require.Equal(t, "/tmp/project/a/b.json", got)
}

func TestWorkflowFolderPathFromRelPath(t *testing.T) {
	cases := map[string]string{
		"a.json":         "/",
		"sub/a.json":     "/sub",
		"sub/two/a.json": "/sub/two",
	}
	for in, want := range cases {
		if got := workflowFolderPathFromRelPath(in); got != want {
			t.Errorf("workflowFolderPathFromRelPath(%q)=%q want %q", in, got, want)
		}
	}
}

// ---------------------------------------------------------------------------
// Module wiring
// ---------------------------------------------------------------------------

func TestModule_PanicsOnMissingDeps(t *testing.T) {
	logger := logrus.New()
	repo := newFakeRepo()
	cat := &fakeCatalog{}

	for _, c := range []struct {
		name string
		deps Deps
	}{
		{"missing logger", Deps{Repo: repo, Catalog: cat}},
		{"missing repo", Deps{Catalog: cat, Logger: logger}},
		{"missing catalog", Deps{Repo: repo, Logger: logger}},
	} {
		t.Run(c.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Fatal("expected panic")
				}
			}()
			Module(c.deps)
		})
	}
}

func TestModule_ReturnsServiceMount(t *testing.T) {
	mount := Module(Deps{
		Repo:    newFakeRepo(),
		Catalog: &fakeCatalog{},
		Logger:  logrus.New(),
	})
	require.NotEmpty(t, mount.Path)
	require.NotNil(t, mount.Handler)
}

// ---------------------------------------------------------------------------
// GetProjectFileTree
// ---------------------------------------------------------------------------

func TestGetProjectFileTree_InvalidProjectID(t *testing.T) {
	c := newTestClient(t, clientDeps{repo: newFakeRepo(), catalog: &fakeCatalog{}})
	_, err := c.GetProjectFileTree(context.Background(), connect.NewRequest(&project_filesv1.GetProjectFileTreeRequest{
		ProjectId: "not-a-uuid",
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestGetProjectFileTree_ProjectNotFound(t *testing.T) {
	c := newTestClient(t, clientDeps{repo: newFakeRepo(), catalog: &fakeCatalog{}})
	_, err := c.GetProjectFileTree(context.Background(), connect.NewRequest(&project_filesv1.GetProjectFileTreeRequest{
		ProjectId: uuid.NewString(),
	}))
	requireConnectCode(t, err, connect.CodeNotFound)
}

func TestGetProjectFileTree_HappyPath(t *testing.T) {
	repo := newFakeRepo()
	cat := &fakeCatalog{}
	c := newTestClient(t, clientDeps{repo: repo, catalog: cat})
	projectID, dir := makeProject(t, repo)

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "subdir"), 0o755))
	wfID := uuid.New()
	repo.workflows[projectID] = []*database.WorkflowIndex{{
		ID: wfID, ProjectID: &projectID, Name: "wf", FolderPath: "/", FilePath: "wf.json", Version: 1,
	}}
	repo.assets[projectID] = []*database.AssetIndex{{
		ID: uuid.New(), ProjectID: projectID, FilePath: "subdir/x.png", MimeType: "image/png", FileSize: 1,
	}}

	resp, err := c.GetProjectFileTree(context.Background(), connect.NewRequest(&project_filesv1.GetProjectFileTreeRequest{
		ProjectId: projectID.String(),
	}))
	require.NoError(t, err)
	entries := resp.Msg.GetEntries()
	require.NotEmpty(t, entries)
	// Order: kind ascending, then path. Verify both a folder and a workflow surface.
	kinds := map[project_filesv1.ProjectEntryKind]int{}
	for _, e := range entries {
		kinds[e.GetKind()]++
	}
	require.Greater(t, kinds[project_filesv1.ProjectEntryKind_PROJECT_ENTRY_KIND_FOLDER], 0)
	require.Equal(t, 1, kinds[project_filesv1.ProjectEntryKind_PROJECT_ENTRY_KIND_WORKFLOW_FILE])
	require.Equal(t, 1, kinds[project_filesv1.ProjectEntryKind_PROJECT_ENTRY_KIND_ASSET_FILE])
	require.Equal(t, 1, cat.calls, "GetProjectFileTree must trigger a SyncProjectWorkflows")
}

// ---------------------------------------------------------------------------
// MkdirProjectPath
// ---------------------------------------------------------------------------

func TestMkdirProjectPath_HappyPath(t *testing.T) {
	repo := newFakeRepo()
	c := newTestClient(t, clientDeps{repo: repo, catalog: &fakeCatalog{}})
	projectID, dir := makeProject(t, repo)

	resp, err := c.MkdirProjectPath(context.Background(), connect.NewRequest(&project_filesv1.MkdirProjectPathRequest{
		ProjectId: projectID.String(),
		Path:      "new/folder",
	}))
	require.NoError(t, err)
	require.Equal(t, "created", resp.Msg.GetStatus())
	require.Equal(t, "new/folder", resp.Msg.GetPath())
	info, err := os.Stat(filepath.Join(dir, "new", "folder"))
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

func TestMkdirProjectPath_InvalidPath(t *testing.T) {
	repo := newFakeRepo()
	c := newTestClient(t, clientDeps{repo: repo, catalog: &fakeCatalog{}})
	projectID, _ := makeProject(t, repo)
	_, err := c.MkdirProjectPath(context.Background(), connect.NewRequest(&project_filesv1.MkdirProjectPathRequest{
		ProjectId: projectID.String(),
		Path:      "../escape",
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

// ---------------------------------------------------------------------------
// WriteProjectWorkflowFile + ReadProjectFile (round-trip)
// ---------------------------------------------------------------------------

func TestWriteAndReadWorkflowFile_RoundTrip(t *testing.T) {
	repo := newFakeRepo()
	cat := &fakeCatalog{}
	c := newTestClient(t, clientDeps{repo: repo, catalog: cat})
	projectID, _ := makeProject(t, repo)

	writeResp, err := c.WriteProjectWorkflowFile(context.Background(), connect.NewRequest(&project_filesv1.WriteProjectWorkflowFileRequest{
		ProjectId: projectID.String(),
		Path:      "hello.json",
		Workflow: &project_filesv1.ProjectWorkflowFileWrite{
			Name:        "hello",
			Description: "hi",
			Tags:        []string{"smoke"},
		},
	}))
	require.NoError(t, err)
	require.Equal(t, "hello.json", writeResp.Msg.GetPath())
	require.NotEmpty(t, writeResp.Msg.GetWorkflowId())
	require.Equal(t, 1, repo.createCalls)

	readResp, err := c.ReadProjectFile(context.Background(), connect.NewRequest(&project_filesv1.ReadProjectFileRequest{
		ProjectId: projectID.String(),
		Path:      "workflows/hello.json",
	}))
	require.NoError(t, err)
	require.NotNil(t, readResp.Msg.GetWorkflow())
	require.Equal(t, "hello", readResp.Msg.GetWorkflow().GetName())
}

func TestWriteWorkflowFile_RejectsNonJSON(t *testing.T) {
	repo := newFakeRepo()
	c := newTestClient(t, clientDeps{repo: repo, catalog: &fakeCatalog{}})
	projectID, _ := makeProject(t, repo)
	_, err := c.WriteProjectWorkflowFile(context.Background(), connect.NewRequest(&project_filesv1.WriteProjectWorkflowFileRequest{
		ProjectId: projectID.String(),
		Path:      "workflows/oops.txt",
		Workflow:  &project_filesv1.ProjectWorkflowFileWrite{Name: "n"},
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestWriteWorkflowFile_AlreadyExists(t *testing.T) {
	repo := newFakeRepo()
	c := newTestClient(t, clientDeps{repo: repo, catalog: &fakeCatalog{}})
	projectID, dir := makeProject(t, repo)
	target := filepath.Join(dir, "workflows", "existing.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0o755))
	require.NoError(t, os.WriteFile(target, []byte("{}"), 0o644))

	_, err := c.WriteProjectWorkflowFile(context.Background(), connect.NewRequest(&project_filesv1.WriteProjectWorkflowFileRequest{
		ProjectId: projectID.String(),
		Path:      "workflows/existing.json",
		Workflow:  &project_filesv1.ProjectWorkflowFileWrite{Name: "x"},
	}))
	requireConnectCode(t, err, connect.CodeAlreadyExists)
}

func TestReadProjectFile_OnlyJSON(t *testing.T) {
	repo := newFakeRepo()
	c := newTestClient(t, clientDeps{repo: repo, catalog: &fakeCatalog{}})
	projectID, _ := makeProject(t, repo)
	_, err := c.ReadProjectFile(context.Background(), connect.NewRequest(&project_filesv1.ReadProjectFileRequest{
		ProjectId: projectID.String(),
		Path:      "workflows/a.txt",
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestReadProjectFile_NotFound(t *testing.T) {
	repo := newFakeRepo()
	c := newTestClient(t, clientDeps{repo: repo, catalog: &fakeCatalog{}})
	projectID, _ := makeProject(t, repo)
	_, err := c.ReadProjectFile(context.Background(), connect.NewRequest(&project_filesv1.ReadProjectFileRequest{
		ProjectId: projectID.String(),
		Path:      "workflows/missing.json",
	}))
	requireConnectCode(t, err, connect.CodeNotFound)
}

// ---------------------------------------------------------------------------
// MoveProjectFile / DeleteProjectFile
// ---------------------------------------------------------------------------

func TestMoveProjectFile_HappyPath(t *testing.T) {
	repo := newFakeRepo()
	cat := &fakeCatalog{}
	c := newTestClient(t, clientDeps{repo: repo, catalog: cat})
	projectID, dir := makeProject(t, repo)
	src := filepath.Join(dir, "a.txt")
	require.NoError(t, os.WriteFile(src, []byte("x"), 0o644))

	resp, err := c.MoveProjectFile(context.Background(), connect.NewRequest(&project_filesv1.MoveProjectFileRequest{
		ProjectId: projectID.String(),
		FromPath:  "a.txt",
		ToPath:    "sub/b.txt",
	}))
	require.NoError(t, err)
	require.Equal(t, "moved", resp.Msg.GetStatus())
	_, err = os.Stat(filepath.Join(dir, "sub", "b.txt"))
	require.NoError(t, err)
	require.Equal(t, 1, cat.calls)
}

func TestMoveProjectFile_InvalidFromPath(t *testing.T) {
	repo := newFakeRepo()
	c := newTestClient(t, clientDeps{repo: repo, catalog: &fakeCatalog{}})
	projectID, _ := makeProject(t, repo)
	_, err := c.MoveProjectFile(context.Background(), connect.NewRequest(&project_filesv1.MoveProjectFileRequest{
		ProjectId: projectID.String(),
		FromPath:  "../x",
		ToPath:    "y",
	}))
	requireConnectCode(t, err, connect.CodeInvalidArgument)
}

func TestDeleteProjectFile_HappyPath(t *testing.T) {
	repo := newFakeRepo()
	c := newTestClient(t, clientDeps{repo: repo, catalog: &fakeCatalog{}})
	projectID, dir := makeProject(t, repo)
	target := filepath.Join(dir, "doomed.txt")
	require.NoError(t, os.WriteFile(target, []byte("bye"), 0o644))

	_, err := c.DeleteProjectFile(context.Background(), connect.NewRequest(&project_filesv1.DeleteProjectFileRequest{
		ProjectId: projectID.String(),
		Path:      "doomed.txt",
	}))
	require.NoError(t, err)
	_, err = os.Stat(target)
	require.True(t, os.IsNotExist(err))
}

// ---------------------------------------------------------------------------
// ResyncProjectFiles
// ---------------------------------------------------------------------------

func TestResyncProjectFiles_HappyPath(t *testing.T) {
	repo := newFakeRepo()
	cat := &fakeCatalog{}
	c := newTestClient(t, clientDeps{repo: repo, catalog: cat})
	projectID, dir := makeProject(t, repo)
	repo.workflows[projectID] = []*database.WorkflowIndex{{ID: uuid.New(), ProjectID: &projectID, FilePath: "w.json"}}
	repo.assets[projectID] = []*database.AssetIndex{{ID: uuid.New(), ProjectID: projectID, FilePath: "a.png"}}

	resp, err := c.ResyncProjectFiles(context.Background(), connect.NewRequest(&project_filesv1.ResyncProjectFilesRequest{
		ProjectId: projectID.String(),
	}))
	require.NoError(t, err)
	require.Equal(t, int32(1), resp.Msg.GetWorkflowsIndexed())
	require.Equal(t, int32(1), resp.Msg.GetAssetsIndexed())
	require.Equal(t, int32(2), resp.Msg.GetEntriesIndexed())
	require.Equal(t, dir, resp.Msg.GetProjectRoot())
	require.Equal(t, 1, cat.calls)
}

func TestResyncProjectFiles_CatalogError(t *testing.T) {
	repo := newFakeRepo()
	cat := &fakeCatalog{returnErr: errors.New("boom")}
	c := newTestClient(t, clientDeps{repo: repo, catalog: cat})
	projectID, _ := makeProject(t, repo)
	_, err := c.ResyncProjectFiles(context.Background(), connect.NewRequest(&project_filesv1.ResyncProjectFilesRequest{
		ProjectId: projectID.String(),
	}))
	requireConnectCode(t, err, connect.CodeInternal)
}

// ---------------------------------------------------------------------------
// RevealProjectPath / OpenProjectFolder
// ---------------------------------------------------------------------------

func TestRevealProjectPath_File(t *testing.T) {
	repo := newFakeRepo()
	osi := &fakeOS{}
	c := newTestClient(t, clientDeps{repo: repo, catalog: &fakeCatalog{}, osi: osi})
	projectID, dir := makeProject(t, repo)
	target := filepath.Join(dir, "note.txt")
	require.NoError(t, os.WriteFile(target, []byte("hi"), 0o644))

	resp, err := c.RevealProjectPath(context.Background(), connect.NewRequest(&project_filesv1.RevealProjectPathRequest{
		ProjectId: projectID.String(),
		Path:      "note.txt",
	}))
	require.NoError(t, err)
	require.Equal(t, "revealed", resp.Msg.GetStatus())
	require.True(t, strings.HasSuffix(resp.Msg.GetPath(), "note.txt"))
	require.Equal(t, 1, len(osi.revealCalls))
	require.Equal(t, 0, len(osi.openCalls))
}

func TestRevealProjectPath_Dir(t *testing.T) {
	repo := newFakeRepo()
	osi := &fakeOS{}
	c := newTestClient(t, clientDeps{repo: repo, catalog: &fakeCatalog{}, osi: osi})
	projectID, dir := makeProject(t, repo)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "subdir"), 0o755))

	resp, err := c.RevealProjectPath(context.Background(), connect.NewRequest(&project_filesv1.RevealProjectPathRequest{
		ProjectId: projectID.String(),
		Path:      "subdir",
	}))
	require.NoError(t, err)
	require.Equal(t, "opened", resp.Msg.GetStatus())
	require.Equal(t, 1, len(osi.openCalls))
	require.Equal(t, 0, len(osi.revealCalls))
}

func TestRevealProjectPath_NotFound(t *testing.T) {
	repo := newFakeRepo()
	c := newTestClient(t, clientDeps{repo: repo, catalog: &fakeCatalog{}, osi: &fakeOS{}})
	projectID, _ := makeProject(t, repo)
	_, err := c.RevealProjectPath(context.Background(), connect.NewRequest(&project_filesv1.RevealProjectPathRequest{
		ProjectId: projectID.String(),
		Path:      "ghost",
	}))
	requireConnectCode(t, err, connect.CodeNotFound)
}

func TestOpenProjectFolder_HappyPath(t *testing.T) {
	repo := newFakeRepo()
	osi := &fakeOS{}
	c := newTestClient(t, clientDeps{repo: repo, catalog: &fakeCatalog{}, osi: osi})
	projectID, dir := makeProject(t, repo)

	resp, err := c.OpenProjectFolder(context.Background(), connect.NewRequest(&project_filesv1.OpenProjectFolderRequest{
		ProjectId: projectID.String(),
	}))
	require.NoError(t, err)
	require.Equal(t, "opened", resp.Msg.GetStatus())
	require.Equal(t, dir, resp.Msg.GetPath())
	require.Equal(t, 1, len(osi.openCalls))
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func requireConnectCode(t *testing.T, err error, want connect.Code) {
	t.Helper()
	require.Error(t, err)
	var ce *connect.Error
	if !errors.As(err, &ce) {
		t.Fatalf("expected *connect.Error, got %T: %v", err, err)
	}
	require.Equal(t, want, ce.Code(), "unexpected connect code: %s", ce.Message())
}
