// Package workspaces is the Connect-RPC handler home for the workspace
// aggregate described by docs/internal/WORKBENCH-PROGRAM.md §3.2. It exposes
// WorkspaceService (proto:
// packages/proto/schemas/react-component-library/v1/workspace) and owns no
// storage rules of its own: identity, revisions, ordering and validation live
// in internal/workspaces, and this package maps typed domain errors onto the
// one Connect error envelope.
package workspaces

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"log"
	"os"
	"strings"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
	workspaceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace/workspace_v1connect"

	"react-component-library/internal/module"
	domain "react-component-library/internal/workspaces"
)

// Handler serves the eight WorkspaceService RPCs over a Repository seam.
type Handler struct {
	repository domain.Repository
	logger     *log.Logger
	assistant  Assistant
}

// NewHandler wires a repository onto the Connect surface. A nil logger uses
// the process default so an internal failure is never silent.
func NewHandler(repository domain.Repository, logger *log.Logger) *Handler {
	if logger == nil {
		logger = log.Default()
	}
	return &Handler{repository: repository, logger: logger}
}

// Module mounts the production workspace service over the scenario's primary
// pool with activity recording disabled.
func Module(db *sql.DB) module.Module {
	return moduleFromRepository(domain.NewSQLiteRepository(db))
}

// ModuleWithRecorder mounts the service with the activity seam wired. main.go
// uses this variant; the repository deliberately does not import the activity
// package.
func ModuleWithRecorder(db *sql.DB, recorder domain.Recorder) module.Module {
	return moduleFromRepository(domain.NewSQLiteRepository(db).WithRecorder(recorder))
}

// ModuleWithRecorderAndAssistant also wires the AI proposal seam. A nil
// assistant is allowed: AssistWorkspace then reports Unavailable instead of
// pretending assistance exists.
func ModuleWithRecorderAndAssistant(db *sql.DB, recorder domain.Recorder, assistant Assistant) module.Module {
	handler := NewHandler(domain.NewSQLiteRepository(db).WithRecorder(recorder), nil).WithAssistant(assistant)
	return moduleFromHandler(handler)
}

// ModuleFromRepository is the injection point for tests and alternate stores.
func ModuleFromRepository(repository domain.Repository, logger *log.Logger) module.Module {
	return moduleFromHandler(NewHandler(repository, logger))
}

func moduleFromRepository(repository domain.Repository) module.Module {
	return moduleFromHandler(NewHandler(repository, nil))
}

func moduleFromHandler(handler *Handler) module.Module {
	connectPath, connectHandler := workspaceconnect.NewWorkspaceServiceHandler(handler)
	return module.Module{
		Name: "workspaces",
		Mount: func(router *mux.Router) {
			mountAssets(router, handler)
			connectx.RegisterServices(router, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
		},
		Endpoints: Endpoints,
	}
}

// SeedFromFile loads the reviewed curated-workspace shelf. A missing file is a
// legitimate fixture state and seeds nothing; a malformed file is an error so a
// content mistake is visible instead of silently emptying the shelf.
func SeedFromFile(ctx context.Context, db *sql.DB, path string) (domain.SeedResult, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return domain.SeedResult{}, nil
	}
	if err != nil {
		return domain.SeedResult{}, err
	}
	return domain.Seed(ctx, db, raw)
}

// Schema re-exports internal/workspaces.Schema for the modules registry.
func Schema() string { return domain.Schema() }

// Migrate re-exports internal/workspaces.Migrate for main.go's boot sequence.
func Migrate(ctx context.Context, db *sql.DB) error { return domain.Migrate(ctx, db) }

func (h *Handler) ListWorkspaces(ctx context.Context, req *connect.Request[workspacev1.ListWorkspacesRequest]) (*connect.Response[workspacev1.ListWorkspacesResponse], error) {
	summaries, err := h.repository.List(ctx, req.Msg.GetProvenance(), req.Msg.GetQuery(), req.Msg.GetLimit())
	if err != nil {
		return nil, h.mapError("list workspaces", err)
	}
	return connect.NewResponse(&workspacev1.ListWorkspacesResponse{Workspaces: summaries}), nil
}

func (h *Handler) GetWorkspace(ctx context.Context, req *connect.Request[workspacev1.GetWorkspaceRequest]) (*connect.Response[workspacev1.GetWorkspaceResponse], error) {
	if err := requireID(req.Msg.GetId()); err != nil {
		return nil, err
	}
	workspace, err := h.repository.Get(ctx, req.Msg.GetId())
	if err != nil {
		return nil, h.mapError("get workspace", err)
	}
	return connect.NewResponse(&workspacev1.GetWorkspaceResponse{Workspace: workspace}), nil
}

func (h *Handler) CreateWorkspace(ctx context.Context, req *connect.Request[workspacev1.CreateWorkspaceRequest]) (*connect.Response[workspacev1.CreateWorkspaceResponse], error) {
	// Curated is a reviewed content state written only by the seed loader.
	// Letting the RPC mint one would create a workspace nobody can edit or
	// delete and would fill the shelf with unreviewed "templates".
	if req.Msg.GetProvenance() == workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			errors.New("curated workspaces come from the reviewed seed; fork one instead"))
	}
	workspace, err := h.repository.Create(ctx, req.Msg)
	if err != nil {
		return nil, h.mapError("create workspace", err)
	}
	return connect.NewResponse(&workspacev1.CreateWorkspaceResponse{Workspace: workspace}), nil
}

func (h *Handler) SaveWorkspace(ctx context.Context, req *connect.Request[workspacev1.SaveWorkspaceRequest]) (*connect.Response[workspacev1.SaveWorkspaceResponse], error) {
	workspace, err := h.repository.Save(ctx, req.Msg)
	if err != nil {
		return nil, h.mapError("save workspace", err)
	}
	return connect.NewResponse(&workspacev1.SaveWorkspaceResponse{Workspace: workspace}), nil
}

func (h *Handler) DeleteWorkspace(ctx context.Context, req *connect.Request[workspacev1.DeleteWorkspaceRequest]) (*connect.Response[workspacev1.DeleteWorkspaceResponse], error) {
	if err := requireID(req.Msg.GetId()); err != nil {
		return nil, err
	}
	if err := h.repository.Delete(ctx, req.Msg.GetId()); err != nil {
		return nil, h.mapError("delete workspace", err)
	}
	return connect.NewResponse(&workspacev1.DeleteWorkspaceResponse{Deleted: true}), nil
}

func (h *Handler) ForkWorkspace(ctx context.Context, req *connect.Request[workspacev1.ForkWorkspaceRequest]) (*connect.Response[workspacev1.ForkWorkspaceResponse], error) {
	if err := requireID(req.Msg.GetId()); err != nil {
		return nil, err
	}
	workspace, err := h.repository.Fork(ctx, req.Msg.GetId(), req.Msg.GetName())
	if err != nil {
		return nil, h.mapError("fork workspace", err)
	}
	return connect.NewResponse(&workspacev1.ForkWorkspaceResponse{Workspace: workspace}), nil
}

func (h *Handler) ValidateWorkspace(ctx context.Context, req *connect.Request[workspacev1.ValidateWorkspaceRequest]) (*connect.Response[workspacev1.ValidateWorkspaceResponse], error) {
	issues, err := h.repository.Validate(ctx, req.Msg.GetWorkspace())
	if err != nil {
		return nil, h.mapError("validate workspace", err)
	}
	return connect.NewResponse(&workspacev1.ValidateWorkspaceResponse{
		Valid:  len(issues) == 0,
		Issues: issues,
	}), nil
}

func (h *Handler) ListWorkspaceRevisions(ctx context.Context, req *connect.Request[workspacev1.ListWorkspaceRevisionsRequest]) (*connect.Response[workspacev1.ListWorkspaceRevisionsResponse], error) {
	if err := requireID(req.Msg.GetId()); err != nil {
		return nil, err
	}
	revisions, err := h.repository.Revisions(ctx, req.Msg.GetId(), req.Msg.GetLimit())
	if err != nil {
		return nil, h.mapError("list workspace revisions", err)
	}
	return connect.NewResponse(&workspacev1.ListWorkspaceRevisionsResponse{Revisions: revisions}), nil
}

// mapError is the single mapping from domain errors to Connect codes. A
// conflict is ABORTED rather than FAILED_PRECONDITION because the request was
// well-formed and retryable: the client lost an optimistic-concurrency race and
// must rebase on the revision carried by ConflictError.Current.
func (h *Handler) mapError(operation string, err error) error {
	var invalid *domain.InvalidError
	if errors.As(err, &invalid) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}
	var conflict *domain.ConflictError
	if errors.As(err, &conflict) {
		return connect.NewError(connect.CodeAborted, err)
	}
	if errors.Is(err, domain.ErrCurated) {
		return connect.NewError(connect.CodeFailedPrecondition, err)
	}
	if errors.Is(err, domain.ErrNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	h.logger.Printf("workspaces: %s: %v", operation, err)
	return connect.NewError(connect.CodeInternal, errors.New("workspace storage failure"))
}

func requireID(id string) error {
	if strings.TrimSpace(id) == "" {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}
	return nil
}

// connectEndpoints is the machine-readable WorkspaceService surface consumed by
// `make endpoints`. Every proto RPC has exactly one descriptor; the parity test
// in internal/modules asserts that invariant. Asset uploads are REST (their
// payload is binary) and are appended to Endpoints in assets.go.
var connectEndpoints = []module.EndpointDescriptor{
	{
		ID:          "workspaces_list",
		Path:        workspaceconnect.WorkspaceServiceListWorkspacesProcedure,
		Method:      "POST",
		Summary:     "List workspaces",
		Description: "Returns workspace summaries newest-updated first, with page/display/state counts. Optional provenance filters the collection; query is a case-insensitive substring over name, summary and tags.",
		Category:    "workspaces",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"provenance": "WorkspaceProvenance (optional)", "query": "string (optional substring)", "limit": "int32 (0 = server default, capped)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"workspaces": "array<WorkspaceSummary>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "Unknown provenance filter"}, {Status: 500, Code: "internal", Description: "Storage read failure"}},
	},
	{
		ID:          "workspaces_get",
		Path:        workspaceconnect.WorkspaceServiceGetWorkspaceProcedure,
		Method:      "POST",
		Summary:     "Get a workspace aggregate",
		Description: "Returns the workspace with its pages, displays and states in stored order.",
		Category:    "workspaces",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"workspace": "Workspace"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "id is required"}, {Status: 404, Code: "not_found", Description: "No workspace with that id"}},
	},
	{
		ID:          "workspaces_create",
		Path:        workspaceconnect.WorkspaceServiceCreateWorkspaceProcedure,
		Method:      "POST",
		Summary:     "Create an authored workspace",
		Description: "Persists a workspace at revision 1, assigning ids to any page/display/state that lacks one. An empty page list seeds one page, one desktop display and one default state.",
		Category:    "workspaces",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"name": "string (required)", "summary": "string", "provenance": "WorkspaceProvenance", "tags": "array<string>", "binding_scenario": "string", "binding_page": "string", "pages": "array<WorkspacePage>", "theme_json": "opaque JSON", "options_json": "opaque JSON"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"workspace": "Workspace"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "name is missing, provenance is curated (seed only), or the aggregate fails validation; issues carry paths"}},
	},
	{
		ID:          "workspaces_save",
		Path:        workspaceconnect.WorkspaceServiceSaveWorkspaceProcedure,
		Method:      "POST",
		Summary:     "Replace a workspace aggregate",
		Description: "Replaces pages/displays/states in one transaction, normalizes ordering to 0..n-1, bumps the revision and appends a protojson revision snapshot. base_revision is required; a stale base returns ABORTED and writes nothing.",
		Category:    "workspaces",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"workspace": "Workspace (required)", "base_revision": "int32 (required)", "actor": "string (recorded on the snapshot)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"workspace": "Workspace"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing workspace/base_revision or an aggregate validation issue"},
			{Status: 404, Code: "not_found", Description: "No workspace with that id"},
			{Status: 409, Code: "aborted", Description: "base_revision did not match the stored revision"},
		},
	},
	{
		ID:          "workspaces_delete",
		Path:        workspaceconnect.WorkspaceServiceDeleteWorkspaceProcedure,
		Method:      "POST",
		Summary:     "Delete a workspace",
		Description: "Deletes the aggregate, its nested rows and its revision history in one transaction.",
		Category:    "workspaces",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"deleted": "bool"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "id is required"}, {Status: 404, Code: "not_found", Description: "No workspace with that id"}},
	},
	{
		ID:          "workspaces_fork",
		Path:        workspaceconnect.WorkspaceServiceForkWorkspaceProcedure,
		Method:      "POST",
		Summary:     "Fork a workspace",
		Description: "Copies the aggregate with fresh page/display/state ids, sets provenance FORKED and origin_workspace_id, and starts the fork at revision 1. An empty name means \"Copy of <origin name>\".",
		Category:    "workspaces",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)", "name": "string (optional)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"workspace": "Workspace"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "id is required"}, {Status: 404, Code: "not_found", Description: "No workspace with that id"}},
	},
	{
		ID:          "workspaces_validate",
		Path:        workspaceconnect.WorkspaceServiceValidateWorkspaceProcedure,
		Method:      "POST",
		Summary:     "Validate a workspace without writing",
		Description: "Checks the required name, aggregate-wide id uniqueness, concrete devices and per-display state-name uniqueness, and returns path-specific issues.",
		Category:    "workspaces",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"workspace": "Workspace"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"valid": "bool", "issues": "array<ValidationIssue>"}},
	},
	{
		ID:          "workspaces_assist",
		Path:        workspaceconnect.WorkspaceServiceAssistWorkspaceProcedure,
		Method:      "POST",
		Summary:     "Propose a patch for one workspace page",
		Description: "Ask the AI gateway for a validated patch against one page, at an explicit revision and with the state on screen selected per display. The proposal is returned as opaque JSON and never written; the studio applies it as one undoable command or discards it.",
		Category:    "workspaces",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)", "base_revision": "int32 (required)", "page_id": "string (optional; first page)", "active_states": "array<{display_id,state_id}>", "prompt": "string (1..2000 chars)", "scope": "{kind: design|frame|layer, display_id?, node_id?}", "idempotency_key": "string (8..200, required)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"revision": "int32", "operation": "WorkspaceAssistOperation", "proposal_json": "opaque validated patch JSON"}},
		Errors: []module.ErrorDesc{
			{Status: 400, Code: "invalid_argument", Description: "Missing id/key, a bad scope, an unknown page/state, or an unreadable composition"},
			{Status: 404, Code: "not_found", Description: "No workspace with that id"},
			{Status: 409, Code: "aborted", Description: "base_revision did not match the stored revision"},
			{Status: 422, Code: "failed_precondition", Description: "The model returned a patch this page cannot apply"},
			{Status: 503, Code: "unavailable", Description: "The AI gateway is not configured or could not be reached"},
		},
	},
	{
		ID:          "workspaces_revisions",
		Path:        workspaceconnect.WorkspaceServiceListWorkspaceRevisionsProcedure,
		Method:      "POST",
		Summary:     "List workspace revision snapshots",
		Description: "Returns revision snapshots newest-first, bounded by limit (0 = server default).",
		Category:    "workspaces",
		Request:     &module.Schema{Type: "object", Properties: map[string]string{"id": "string (required)", "limit": "int32 (0 = server default, capped)"}},
		Response:    &module.Schema{Type: "object", Properties: map[string]string{"revisions": "array<WorkspaceRevision>"}},
		Errors:      []module.ErrorDesc{{Status: 400, Code: "invalid_argument", Description: "id is required"}, {Status: 404, Code: "not_found", Description: "No workspace with that id"}},
	},
}
