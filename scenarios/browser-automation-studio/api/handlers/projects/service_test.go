package projects

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/vrooli/browser-automation-studio/database"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basprojects "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/projects"
	projectsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/projects/projectsconnect"
)

// ---------------------------------------------------------------------------
// fakes
// ---------------------------------------------------------------------------

type fakeCatalog struct {
	mu              sync.Mutex
	projects        map[uuid.UUID]*database.ProjectIndex
	projectsByName  map[string]*database.ProjectIndex
	projectsByPath  map[string]*database.ProjectIndex
	workflows       map[uuid.UUID][]*database.WorkflowIndex
	stats           map[uuid.UUID]*database.ProjectStats
	deletedWorkflows map[uuid.UUID][]uuid.UUID
	deleted         []uuid.UUID
	getErr          error
	updateErr       error
	deleteErr       error
	createErr       error
	listErr         error
	bulkDeleteErr   error
	hydrateErr      error
}

func newFakeCatalog() *fakeCatalog {
	return &fakeCatalog{
		projects:         map[uuid.UUID]*database.ProjectIndex{},
		projectsByName:   map[string]*database.ProjectIndex{},
		projectsByPath:   map[string]*database.ProjectIndex{},
		workflows:        map[uuid.UUID][]*database.WorkflowIndex{},
		stats:            map[uuid.UUID]*database.ProjectStats{},
		deletedWorkflows: map[uuid.UUID][]uuid.UUID{},
	}
}

func (f *fakeCatalog) CreateProject(_ context.Context, p *database.ProjectIndex, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
	f.projects[p.ID] = p
	f.projectsByName[p.Name] = p
	f.projectsByPath[p.FolderPath] = p
	return nil
}

func (f *fakeCatalog) GetProject(_ context.Context, id uuid.UUID) (*database.ProjectIndex, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.getErr != nil {
		return nil, f.getErr
	}
	p, ok := f.projects[id]
	if !ok {
		return nil, database.ErrNotFound
	}
	return p, nil
}

func (f *fakeCatalog) GetProjectByName(_ context.Context, name string) (*database.ProjectIndex, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.projectsByName[name]
	if !ok {
		return nil, database.ErrNotFound
	}
	return p, nil
}

func (f *fakeCatalog) GetProjectByFolderPath(_ context.Context, path string) (*database.ProjectIndex, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.projectsByPath[path]
	if !ok {
		return nil, database.ErrNotFound
	}
	return p, nil
}

func (f *fakeCatalog) UpdateProject(_ context.Context, p *database.ProjectIndex, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return f.updateErr
	}
	p.UpdatedAt = time.Now()
	f.projects[p.ID] = p
	return nil
}

func (f *fakeCatalog) DeleteProject(_ context.Context, id uuid.UUID, _ bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if _, ok := f.projects[id]; !ok {
		return database.ErrNotFound
	}
	delete(f.projects, id)
	f.deleted = append(f.deleted, id)
	return nil
}

func (f *fakeCatalog) ListProjects(_ context.Context, _, _ int) ([]*database.ProjectIndex, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listErr != nil {
		return nil, f.listErr
	}
	out := make([]*database.ProjectIndex, 0, len(f.projects))
	for _, p := range f.projects {
		out = append(out, p)
	}
	return out, nil
}

func (f *fakeCatalog) GetProjectStats(_ context.Context, id uuid.UUID) (*database.ProjectStats, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if s, ok := f.stats[id]; ok {
		return s, nil
	}
	return &database.ProjectStats{ProjectID: id}, nil
}

func (f *fakeCatalog) GetProjectsStats(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]*database.ProjectStats, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[uuid.UUID]*database.ProjectStats{}
	for _, id := range ids {
		if s, ok := f.stats[id]; ok {
			out[id] = s
		} else {
			out[id] = &database.ProjectStats{ProjectID: id}
		}
	}
	return out, nil
}

func (f *fakeCatalog) ListWorkflowsByProject(_ context.Context, id uuid.UUID, _, _ int) ([]*database.WorkflowIndex, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.workflows[id], nil
}

func (f *fakeCatalog) DeleteProjectWorkflows(_ context.Context, projectID uuid.UUID, ids []uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.bulkDeleteErr != nil {
		return f.bulkDeleteErr
	}
	f.deletedWorkflows[projectID] = append(f.deletedWorkflows[projectID], ids...)
	return nil
}

func (f *fakeCatalog) HydrateProject(_ context.Context, p *database.ProjectIndex) (*basprojects.Project, error) {
	if f.hydrateErr != nil {
		return nil, f.hydrateErr
	}
	return &basprojects.Project{
		Id:         p.ID.String(),
		Name:       p.Name,
		FolderPath: p.FolderPath,
	}, nil
}

func (f *fakeCatalog) ListWorkflows(_ context.Context, req *basapi.ListWorkflowsRequest) (*basapi.ListWorkflowsResponse, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if req.ProjectId == nil {
		return &basapi.ListWorkflowsResponse{}, nil
	}
	id, _ := uuid.Parse(*req.ProjectId)
	wfs := f.workflows[id]
	out := make([]*basapi.WorkflowSummary, 0, len(wfs))
	for _, w := range wfs {
		out = append(out, &basapi.WorkflowSummary{Id: w.ID.String(), Name: w.Name})
	}
	return &basapi.ListWorkflowsResponse{Workflows: out}, nil
}

type fakeExecutor struct {
	mu       sync.Mutex
	executed []uuid.UUID
	err      error
}

func (f *fakeExecutor) ExecuteWorkflow(_ context.Context, id uuid.UUID, _ map[string]any) (*database.ExecutionIndex, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	f.executed = append(f.executed, id)
	return &database.ExecutionIndex{ID: uuid.New(), Status: "pending"}, nil
}

type fakePaths struct {
	mu        sync.Mutex
	prepared  []string
	prepareErr error
	madeAll   []string
	makeErr   error
}

func (f *fakePaths) Prepare(p string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.prepareErr != nil {
		return "", f.prepareErr
	}
	f.prepared = append(f.prepared, p)
	return p, nil
}

func (f *fakePaths) MakeAll(p string, _ uint32) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.makeErr != nil {
		return f.makeErr
	}
	f.madeAll = append(f.madeAll, p)
	return nil
}

// ---------------------------------------------------------------------------
// test client
// ---------------------------------------------------------------------------

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) { w.t.Log(string(p)); return len(p), nil }

type clientDeps struct {
	catalog  Catalog
	executor Executor
	paths    PathPreparer
}

func newTestClient(t *testing.T, d clientDeps) projectsconnect.ProjectsServiceClient {
	t.Helper()
	logger := logrus.New()
	logger.SetOutput(testWriter{t})
	if d.paths == nil {
		d.paths = &fakePaths{}
	}
	if d.executor == nil {
		d.executor = &fakeExecutor{}
	}
	mount := Module(Deps{Catalog: d.catalog, Executor: d.executor, Paths: d.paths, Logger: logger})
	mux := http.NewServeMux()
	mux.Handle(mount.Path, mount.Handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return projectsconnect.NewProjectsServiceClient(srv.Client(), srv.URL)
}

// ---------------------------------------------------------------------------
// Module wiring
// ---------------------------------------------------------------------------

func TestModuleRequiresLogger(t *testing.T) {
	require.PanicsWithValue(t, "projects.Module requires Deps.Logger", func() {
		Module(Deps{})
	})
}

func TestModuleRequiresCatalog(t *testing.T) {
	require.PanicsWithValue(t, "projects.Module requires Deps.Catalog", func() {
		Module(Deps{Logger: logrus.New()})
	})
}

func TestModuleRequiresExecutor(t *testing.T) {
	require.PanicsWithValue(t, "projects.Module requires Deps.Executor", func() {
		Module(Deps{Logger: logrus.New(), Catalog: newFakeCatalog()})
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func TestNormalizeProjectRelPath(t *testing.T) {
	cases := []struct {
		in, want string
		ok       bool
	}{
		{"", "", false},
		{".", "", false},
		{"workflows/a", "workflows/a", true},
		{"/workflows/a", "workflows/a", true},
		{"../escape", "", false},
	}
	for _, c := range cases {
		got, ok := normalizeProjectRelPath(c.in)
		require.Equalf(t, c.ok, ok, "case %q", c.in)
		require.Equalf(t, c.want, got, "case %q", c.in)
	}
}

// ---------------------------------------------------------------------------
// CreateProject
// ---------------------------------------------------------------------------

func TestCreateProjectHappy(t *testing.T) {
	cat := newFakeCatalog()
	paths := &fakePaths{}
	c := newTestClient(t, clientDeps{catalog: cat, paths: paths})

	resp, err := c.CreateProject(context.Background(), connect.NewRequest(&basprojects.CreateProjectRequest{
		Name:       "alpha",
		FolderPath: "/tmp/projects/alpha",
	}))
	require.NoError(t, err)
	require.Equal(t, "alpha", resp.Msg.GetProject().GetName())
	require.NotEmpty(t, resp.Msg.GetProject().GetId())
	require.Equal(t, []string{"/tmp/projects/alpha"}, paths.prepared)
}

func TestCreateProjectMissingName(t *testing.T) {
	c := newTestClient(t, clientDeps{catalog: newFakeCatalog()})
	_, err := c.CreateProject(context.Background(), connect.NewRequest(&basprojects.CreateProjectRequest{
		FolderPath: "/tmp/x",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestCreateProjectMissingFolder(t *testing.T) {
	c := newTestClient(t, clientDeps{catalog: newFakeCatalog()})
	_, err := c.CreateProject(context.Background(), connect.NewRequest(&basprojects.CreateProjectRequest{
		Name: "n",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestCreateProjectAlreadyExistsByName(t *testing.T) {
	cat := newFakeCatalog()
	existingID := uuid.New()
	cat.projects[existingID] = &database.ProjectIndex{ID: existingID, Name: "dup", FolderPath: "/tmp/other"}
	cat.projectsByName["dup"] = cat.projects[existingID]
	c := newTestClient(t, clientDeps{catalog: cat})
	_, err := c.CreateProject(context.Background(), connect.NewRequest(&basprojects.CreateProjectRequest{
		Name: "dup", FolderPath: "/tmp/new",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeAlreadyExists, connect.CodeOf(err))
}

func TestCreateProjectPresetRecommended(t *testing.T) {
	cat := newFakeCatalog()
	paths := &fakePaths{}
	c := newTestClient(t, clientDeps{catalog: cat, paths: paths})

	_, err := c.CreateProject(context.Background(), connect.NewRequest(&basprojects.CreateProjectRequest{
		Name:       "beta",
		FolderPath: "/tmp/beta",
		Preset:     basprojects.PresetKind_PRESET_KIND_RECOMMENDED,
	}))
	require.NoError(t, err)
	require.Len(t, paths.madeAll, 4) // actions, flows, cases, assets
}

func TestCreateProjectPresetCustomInvalidPath(t *testing.T) {
	cat := newFakeCatalog()
	c := newTestClient(t, clientDeps{catalog: cat})
	_, err := c.CreateProject(context.Background(), connect.NewRequest(&basprojects.CreateProjectRequest{
		Name:        "gamma",
		FolderPath:  "/tmp/gamma",
		Preset:      basprojects.PresetKind_PRESET_KIND_CUSTOM,
		PresetPaths: []string{"../escape"},
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

// ---------------------------------------------------------------------------
// ListProjects / GetProject
// ---------------------------------------------------------------------------

func TestListProjects(t *testing.T) {
	cat := newFakeCatalog()
	for i := 0; i < 3; i++ {
		id := uuid.New()
		cat.projects[id] = &database.ProjectIndex{ID: id, Name: "p", FolderPath: "/tmp/p"}
	}
	c := newTestClient(t, clientDeps{catalog: cat})
	resp, err := c.ListProjects(context.Background(), connect.NewRequest(&basprojects.ListProjectsRequest{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetProjects(), 3)
}

func TestGetProjectHappy(t *testing.T) {
	cat := newFakeCatalog()
	id := uuid.New()
	cat.projects[id] = &database.ProjectIndex{ID: id, Name: "x", FolderPath: "/tmp/x"}
	c := newTestClient(t, clientDeps{catalog: cat})
	resp, err := c.GetProject(context.Background(), connect.NewRequest(&basprojects.GetProjectRequest{Id: id.String()}))
	require.NoError(t, err)
	require.Equal(t, id.String(), resp.Msg.GetProject().GetId())
}

func TestGetProjectNotFound(t *testing.T) {
	c := newTestClient(t, clientDeps{catalog: newFakeCatalog()})
	_, err := c.GetProject(context.Background(), connect.NewRequest(&basprojects.GetProjectRequest{Id: uuid.New().String()}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestGetProjectInvalidID(t *testing.T) {
	c := newTestClient(t, clientDeps{catalog: newFakeCatalog()})
	_, err := c.GetProject(context.Background(), connect.NewRequest(&basprojects.GetProjectRequest{Id: "not-a-uuid"}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

// ---------------------------------------------------------------------------
// UpdateProject / DeleteProject
// ---------------------------------------------------------------------------

func TestUpdateProjectHappy(t *testing.T) {
	cat := newFakeCatalog()
	id := uuid.New()
	cat.projects[id] = &database.ProjectIndex{ID: id, Name: "old", FolderPath: "/tmp/old"}
	c := newTestClient(t, clientDeps{catalog: cat})
	resp, err := c.UpdateProject(context.Background(), connect.NewRequest(&basprojects.UpdateProjectRequest{
		Id:   id.String(),
		Name: "new",
	}))
	require.NoError(t, err)
	require.Equal(t, "new", resp.Msg.GetProject().GetName())
	require.Equal(t, "new", cat.projects[id].Name)
}

func TestUpdateProjectNotFound(t *testing.T) {
	c := newTestClient(t, clientDeps{catalog: newFakeCatalog()})
	_, err := c.UpdateProject(context.Background(), connect.NewRequest(&basprojects.UpdateProjectRequest{
		Id: uuid.New().String(), Name: "x",
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestDeleteProjectHappy(t *testing.T) {
	cat := newFakeCatalog()
	id := uuid.New()
	cat.projects[id] = &database.ProjectIndex{ID: id, Name: "x", FolderPath: "/tmp/x"}
	c := newTestClient(t, clientDeps{catalog: cat})
	resp, err := c.DeleteProject(context.Background(), connect.NewRequest(&basprojects.DeleteProjectRequest{
		Id: id.String(), DeleteFiles: true,
	}))
	require.NoError(t, err)
	require.True(t, resp.Msg.GetFilesDeleted())
	require.Contains(t, cat.deleted, id)
}

func TestDeleteProjectNotFound(t *testing.T) {
	cat := newFakeCatalog()
	cat.deleteErr = database.ErrNotFound
	c := newTestClient(t, clientDeps{catalog: cat})
	_, err := c.DeleteProject(context.Background(), connect.NewRequest(&basprojects.DeleteProjectRequest{
		Id: uuid.New().String(),
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

// ---------------------------------------------------------------------------
// ListProjectWorkflows / BulkDelete / ExecuteAll
// ---------------------------------------------------------------------------

func TestListProjectWorkflows(t *testing.T) {
	cat := newFakeCatalog()
	id := uuid.New()
	cat.projects[id] = &database.ProjectIndex{ID: id, Name: "x", FolderPath: "/tmp/x"}
	cat.workflows[id] = []*database.WorkflowIndex{
		{ID: uuid.New(), Name: "wf1"},
		{ID: uuid.New(), Name: "wf2"},
	}
	c := newTestClient(t, clientDeps{catalog: cat})
	resp, err := c.ListProjectWorkflows(context.Background(), connect.NewRequest(&basprojects.ListProjectWorkflowsRequest{
		ProjectId: id.String(),
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetWorkflows(), 2)
}

func TestBulkDeleteProjectWorkflowsHappy(t *testing.T) {
	cat := newFakeCatalog()
	id := uuid.New()
	cat.projects[id] = &database.ProjectIndex{ID: id}
	wid1, wid2 := uuid.New(), uuid.New()
	c := newTestClient(t, clientDeps{catalog: cat})
	resp, err := c.BulkDeleteProjectWorkflows(context.Background(), connect.NewRequest(&basprojects.BulkDeleteProjectWorkflowsRequest{
		ProjectId:   id.String(),
		WorkflowIds: []string{wid1.String(), wid2.String()},
	}))
	require.NoError(t, err)
	require.Equal(t, int32(2), resp.Msg.GetDeletedCount())
	require.Len(t, cat.deletedWorkflows[id], 2)
}

func TestBulkDeleteEmpty(t *testing.T) {
	cat := newFakeCatalog()
	id := uuid.New()
	cat.projects[id] = &database.ProjectIndex{ID: id}
	c := newTestClient(t, clientDeps{catalog: cat})
	_, err := c.BulkDeleteProjectWorkflows(context.Background(), connect.NewRequest(&basprojects.BulkDeleteProjectWorkflowsRequest{
		ProjectId: id.String(),
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestBulkDeleteBadWorkflowID(t *testing.T) {
	cat := newFakeCatalog()
	id := uuid.New()
	cat.projects[id] = &database.ProjectIndex{ID: id}
	c := newTestClient(t, clientDeps{catalog: cat})
	_, err := c.BulkDeleteProjectWorkflows(context.Background(), connect.NewRequest(&basprojects.BulkDeleteProjectWorkflowsRequest{
		ProjectId:   id.String(),
		WorkflowIds: []string{"not-a-uuid"},
	}))
	require.Error(t, err)
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

func TestExecuteAllEmptyProject(t *testing.T) {
	cat := newFakeCatalog()
	id := uuid.New()
	cat.projects[id] = &database.ProjectIndex{ID: id}
	c := newTestClient(t, clientDeps{catalog: cat})
	resp, err := c.ExecuteAllProjectWorkflows(context.Background(), connect.NewRequest(&basprojects.ExecuteAllProjectWorkflowsRequest{
		ProjectId: id.String(),
	}))
	require.NoError(t, err)
	require.Empty(t, resp.Msg.GetExecutions())
}

func TestExecuteAllMixedResults(t *testing.T) {
	cat := newFakeCatalog()
	id := uuid.New()
	cat.projects[id] = &database.ProjectIndex{ID: id}
	cat.workflows[id] = []*database.WorkflowIndex{
		{ID: uuid.New(), Name: "wf1"},
		{ID: uuid.New(), Name: "wf2"},
	}
	exec := &fakeExecutor{}
	c := newTestClient(t, clientDeps{catalog: cat, executor: exec})
	resp, err := c.ExecuteAllProjectWorkflows(context.Background(), connect.NewRequest(&basprojects.ExecuteAllProjectWorkflowsRequest{
		ProjectId: id.String(),
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetExecutions(), 2)
	for _, r := range resp.Msg.GetExecutions() {
		require.Equal(t, "pending", r.GetStatus())
		require.NotEmpty(t, r.GetExecutionId())
	}
	require.Len(t, exec.executed, 2)
}

func TestExecuteAllReportsPerWorkflowFailure(t *testing.T) {
	cat := newFakeCatalog()
	id := uuid.New()
	cat.projects[id] = &database.ProjectIndex{ID: id}
	cat.workflows[id] = []*database.WorkflowIndex{{ID: uuid.New(), Name: "wf"}}
	exec := &fakeExecutor{err: errors.New("boom")}
	c := newTestClient(t, clientDeps{catalog: cat, executor: exec})
	resp, err := c.ExecuteAllProjectWorkflows(context.Background(), connect.NewRequest(&basprojects.ExecuteAllProjectWorkflowsRequest{
		ProjectId: id.String(),
	}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.GetExecutions(), 1)
	require.Equal(t, "failed", resp.Msg.GetExecutions()[0].GetStatus())
	require.Equal(t, "boom", resp.Msg.GetExecutions()[0].GetError())
}
