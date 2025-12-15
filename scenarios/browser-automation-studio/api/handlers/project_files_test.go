//go:build legacydb
// +build legacydb

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
)

type projectFilesRepo struct {
	mockRepository

	projects  map[uuid.UUID]*database.Project
	workflows map[uuid.UUID]*database.Workflow
	entries   map[string]*database.ProjectEntry
}

func newProjectFilesRepo() *projectFilesRepo {
	return &projectFilesRepo{
		projects:  make(map[uuid.UUID]*database.Project),
		workflows: make(map[uuid.UUID]*database.Workflow),
		entries:   make(map[string]*database.ProjectEntry),
	}
}

func projectEntryKey(projectID uuid.UUID, path string) string {
	return projectID.String() + ":" + path
}

func (r *projectFilesRepo) GetProject(ctx context.Context, id uuid.UUID) (*database.Project, error) {
	p, ok := r.projects[id]
	if !ok {
		return nil, database.ErrNotFound
	}
	clone := *p
	return &clone, nil
}

func (r *projectFilesRepo) DeleteProjectEntries(ctx context.Context, projectID uuid.UUID) error {
	for key := range r.entries {
		if len(key) >= len(projectID.String())+1 && key[:len(projectID.String())] == projectID.String() {
			delete(r.entries, key)
		}
	}
	return nil
}

func (r *projectFilesRepo) UpsertProjectEntry(ctx context.Context, entry *database.ProjectEntry) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	clone := *entry
	r.entries[projectEntryKey(entry.ProjectID, entry.Path)] = &clone
	return nil
}

func (r *projectFilesRepo) GetProjectEntry(ctx context.Context, projectID uuid.UUID, path string) (*database.ProjectEntry, error) {
	entry, ok := r.entries[projectEntryKey(projectID, path)]
	if !ok {
		return nil, database.ErrNotFound
	}
	clone := *entry
	return &clone, nil
}

func (r *projectFilesRepo) DeleteProjectEntry(ctx context.Context, projectID uuid.UUID, path string) error {
	delete(r.entries, projectEntryKey(projectID, path))
	return nil
}

func (r *projectFilesRepo) ListProjectEntries(ctx context.Context, projectID uuid.UUID) ([]*database.ProjectEntry, error) {
	out := make([]*database.ProjectEntry, 0)
	prefix := projectID.String() + ":"
	for key, entry := range r.entries {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			clone := *entry
			out = append(out, &clone)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func (r *projectFilesRepo) CreateWorkflow(ctx context.Context, wf *database.Workflow) error {
	if wf.ID == uuid.Nil {
		wf.ID = uuid.New()
	}
	clone := *wf
	r.workflows[wf.ID] = &clone
	return nil
}

func (r *projectFilesRepo) UpdateWorkflow(ctx context.Context, wf *database.Workflow) error {
	clone := *wf
	r.workflows[wf.ID] = &clone
	return nil
}

func (r *projectFilesRepo) GetWorkflow(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
	wf, ok := r.workflows[id]
	if !ok {
		return nil, database.ErrNotFound
	}
	clone := *wf
	return &clone, nil
}

func (r *projectFilesRepo) DeleteWorkflow(ctx context.Context, id uuid.UUID) error {
	delete(r.workflows, id)
	return nil
}

func withProjectRouteContext(req *http.Request, projectID uuid.UUID) *http.Request {
	ctx := chi.RouteContext(req.Context())
	if ctx == nil {
		ctx = chi.NewRouteContext()
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, ctx))
	}
	ctx.URLParams.Add("id", projectID.String())
	return req
}

func newProjectFilesHandler(repo database.Repository) *Handler {
	log := logrus.New()
	log.SetOutput(io.Discard)
	return &Handler{repo: repo, log: log}
}

func TestWriteProjectWorkflowFileWritesDiskAndIndex(t *testing.T) {
	root := t.TempDir()
	projectID := uuid.New()
	repo := newProjectFilesRepo()
	repo.projects[projectID] = &database.Project{ID: projectID, Name: "p", FolderPath: root}
	h := newProjectFilesHandler(repo)

	payload := map[string]any{
		"path": "actions/auth/login.action.json",
		"workflow": map[string]any{
			"name": "login",
			"type": "action",
			"flow_definition": map[string]any{
				"nodes": []any{},
				"edges": []any{},
			},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/files/write", bytes.NewReader(body))
	req = withProjectRouteContext(req, projectID)
	w := httptest.NewRecorder()

	h.WriteProjectWorkflowFile(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	abs := filepath.Join(root, filepath.FromSlash("actions/auth/login.action.json"))
	raw, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("expected workflow file to exist: %v", err)
	}

	var onDisk ProjectWorkflowFileOnDisk
	if err := json.Unmarshal(raw, &onDisk); err != nil {
		t.Fatalf("expected valid workflow json: %v", err)
	}
	if onDisk.ID == "" {
		t.Fatalf("expected workflow id to be written to disk")
	}
	if onDisk.Type != "action" {
		t.Fatalf("expected type action, got %q", onDisk.Type)
	}

	entry, err := repo.GetProjectEntry(context.Background(), projectID, "actions/auth/login.action.json")
	if err != nil {
		t.Fatalf("expected project entry: %v", err)
	}
	if entry.Kind != database.ProjectEntryKindWorkflowFile || entry.WorkflowID == nil {
		t.Fatalf("expected workflow_file entry with workflow id")
	}
}

func TestWriteProjectWorkflowFileValidatesCaseExpectation(t *testing.T) {
	root := t.TempDir()
	projectID := uuid.New()
	repo := newProjectFilesRepo()
	repo.projects[projectID] = &database.Project{ID: projectID, Name: "p", FolderPath: root}
	h := newProjectFilesHandler(repo)

	payload := map[string]any{
		"path": "cases/signup_success.case.json",
		"workflow": map[string]any{
			"name": "signup_success",
			"type": "case",
			"flow_definition": map[string]any{
				"nodes": []any{
					map[string]any{
						"id":   "wait-for-ready",
						"type": "wait",
						"data": map[string]any{
							"label":     "Wait for ready",
							"selector":  "#ready",
							"timeoutMs": 1000,
							"waitType":  "element",
						},
					},
				},
				"edges": []any{},
			},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/files/write", bytes.NewReader(body))
	req = withProjectRouteContext(req, projectID)
	w := httptest.NewRecorder()

	h.WriteProjectWorkflowFile(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["code"] != "CASE_EXPECTATION_MISSING" {
		t.Fatalf("expected code CASE_EXPECTATION_MISSING, got %v", resp["code"])
	}
}

func TestWriteProjectWorkflowFileReturnsWarningsForAssertionsInActions(t *testing.T) {
	root := t.TempDir()
	projectID := uuid.New()
	repo := newProjectFilesRepo()
	repo.projects[projectID] = &database.Project{ID: projectID, Name: "p", FolderPath: root}
	h := newProjectFilesHandler(repo)

	payload := map[string]any{
		"path": "actions/demo.action.json",
		"workflow": map[string]any{
			"name": "demo",
			"type": "action",
			"flow_definition": map[string]any{
				"nodes": []any{
					map[string]any{
						"id":   "assert-ready",
						"type": "assert",
						"data": map[string]any{
							"label":          "Assert ready exists",
							"selector":       "#ready",
							"assertMode":     "exists",
							"timeoutMs":      1000,
							"failureMessage": "Expected #ready",
						},
					},
				},
				"edges": []any{},
			},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/files/write", bytes.NewReader(body))
	req = withProjectRouteContext(req, projectID)
	w := httptest.NewRecorder()

	h.WriteProjectWorkflowFile(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	warnings, ok := resp["warnings"].([]any)
	if !ok {
		t.Fatalf("expected warnings array, got %T", resp["warnings"])
	}
	if len(warnings) == 0 {
		t.Fatalf("expected at least one warning")
	}
}

func TestResyncProjectFilesIndexesDiskAndRepairsIDs(t *testing.T) {
	root := t.TempDir()
	projectID := uuid.New()
	repo := newProjectFilesRepo()
	repo.projects[projectID] = &database.Project{ID: projectID, Name: "p", FolderPath: root}
	h := newProjectFilesHandler(repo)

	if err := os.MkdirAll(filepath.Join(root, "cases"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Missing id on purpose; resync should repair it.
	wf := ProjectWorkflowFileOnDisk{
		Version:        "v1",
		ID:             "",
		Name:           "signup_success",
		Type:           "case",
		FlowDefinition: map[string]any{"nodes": []any{}, "edges": []any{}},
	}
	raw, _ := json.MarshalIndent(wf, "", "  ")
	raw = append(raw, '\n')
	if err := os.WriteFile(filepath.Join(root, "cases/signup_success.case.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.txt"), []byte("asset\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+projectID.String()+"/files/resync", nil)
	req = withProjectRouteContext(req, projectID)
	w := httptest.NewRecorder()

	h.ResyncProjectFiles(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Ensure ID repaired on disk.
	updated, err := os.ReadFile(filepath.Join(root, "cases/signup_success.case.json"))
	if err != nil {
		t.Fatal(err)
	}
	var repaired ProjectWorkflowFileOnDisk
	if err := json.Unmarshal(updated, &repaired); err != nil {
		t.Fatalf("expected valid json: %v", err)
	}
	if repaired.ID == "" {
		t.Fatalf("expected resync to populate id")
	}

	entries, _ := repo.ListProjectEntries(context.Background(), projectID)
	if len(entries) < 2 {
		t.Fatalf("expected entries indexed, got %d", len(entries))
	}
	_, err = repo.GetProjectEntry(context.Background(), projectID, "README.txt")
	if err != nil {
		t.Fatalf("expected README asset indexed: %v", err)
	}
	_, err = repo.GetProjectEntry(context.Background(), projectID, "cases/signup_success.case.json")
	if err != nil {
		t.Fatalf("expected workflow file indexed: %v", err)
	}
}
