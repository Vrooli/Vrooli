package workspaces

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
	workspaceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace/workspace_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	"react-component-library/cli/internal/runtimeenv"
)

type fixedWorkspaceClock time.Time

func (clock fixedWorkspaceClock) Now() time.Time { return time.Time(clock) }

type workspaceEnvironment map[string]string

func (environment workspaceEnvironment) LookupEnv(key string) (string, bool) {
	value, ok := environment[key]
	return value, ok
}

func TestWorkspacesManifestCoverage(t *testing.T) {
	raw := cliapptest.ReadManifest(t)
	cliapp.RequireProtoServiceCoverage(t, raw, workspacev1.File_react_component_library_v1_workspace_workspace_proto, "WorkspaceService")
}

func TestWorkspacesGroupRegistersEveryManifestCommand(t *testing.T) {
	raw := cliapptest.ReadManifest(t)
	group, err := Register(&cliapp.ScenarioApp{}, raw)
	require.NoError(t, err)

	manifest, err := cliapp.ParseManifest(raw)
	require.NoError(t, err)
	declared := manifest.FindGroup(GroupName)
	require.NotNil(t, declared)
	require.Len(t, group.Subcommands, len(declared.Commands))

	for _, manifestCommand := range declared.Commands {
		command := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, manifestCommand.Name)
		require.NotNilf(t, command.RunCtx, "command %s/%s has no bound handler", GroupName, manifestCommand.Name)
		require.NotNilf(t, manifestCommand.Architecture, "command %s/%s declares no architecture", GroupName, manifestCommand.Name)
		require.Equalf(t, manifestCommand.Architecture.Primitive, string(command.PrimitiveEvidence()),
			"command %s/%s primitive evidence does not match its manifest declaration", GroupName, manifestCommand.Name)
	}
}

func TestWorkspacesListForwardsFiltersToConnect(t *testing.T) {
	var got *workspacev1.ListWorkspacesRequest
	mux := http.NewServeMux()
	mux.Handle(workspaceconnect.WorkspaceServiceListWorkspacesProcedure, connect.NewUnaryHandlerSimple(
		workspaceconnect.WorkspaceServiceListWorkspacesProcedure,
		func(_ context.Context, req *workspacev1.ListWorkspacesRequest) (*workspacev1.ListWorkspacesResponse, error) {
			got = req
			return &workspacev1.ListWorkspacesResponse{Workspaces: []*workspacev1.WorkspaceSummary{{
				Id:           "ws-1",
				Name:         "Dashboard",
				Provenance:   workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_AUTHORED,
				Tags:         []string{"ops"},
				PageCount:    2,
				DisplayCount: 3,
				StateCount:   4,
				Revision:     3,
			}}}, nil
		},
	))
	core := cliapptest.NewTestApp(t, mux)
	group, err := Register(core, cliapptest.ReadManifest(t))
	require.NoError(t, err)
	command := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "list")

	out, err := cliapptest.RunCommand(t, command, core, "--provenance", "authored", "--query", "dash", "--limit", "5")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_AUTHORED, got.GetProvenance())
	require.Equal(t, "dash", got.GetQuery())
	require.Equal(t, int32(5), got.GetLimit())
	require.Contains(t, out, "Found 1 workspace(s).")
	require.Contains(t, out, "Dashboard (ws-1)")
	require.Contains(t, out, "pages=2 displays=3 states=4")
}

func TestWorkspacesSaveConflictNamesTheChangedWorkspace(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle(workspaceconnect.WorkspaceServiceSaveWorkspaceProcedure, connect.NewUnaryHandlerSimple(
		workspaceconnect.WorkspaceServiceSaveWorkspaceProcedure,
		func(_ context.Context, _ *workspacev1.SaveWorkspaceRequest) (*workspacev1.SaveWorkspaceResponse, error) {
			return nil, connect.NewError(connect.CodeAborted, errors.New("workspace revision conflict: current revision is 5"))
		},
	))
	core := cliapptest.NewTestApp(t, mux)
	group, err := Register(core, cliapptest.ReadManifest(t))
	require.NoError(t, err)
	command := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "save")

	path := filepath.Join(t.TempDir(), "workspace.json")
	body, err := protojson.Marshal(&workspacev1.Workspace{Id: "ws-1", Name: "Dashboard", Revision: 4})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, body, 0o600))

	_, err = cliapptest.RunCommand(t, command, core, "ws-1", "--base-revision", "4", "--workspace", path)
	require.Error(t, err)
	require.Contains(t, err.Error(), "workspace ws-1 changed since base revision 4")
	require.NotContains(t, err.Error(), "\n", "the conflict must stay a one-line operator message")
}

func TestWorkspacesAssistForwardsScopeAndReadsRevisionWhenOmitted(t *testing.T) {
	var got *workspacev1.AssistWorkspaceRequest
	var reads int
	mux := http.NewServeMux()
	mux.Handle(workspaceconnect.WorkspaceServiceGetWorkspaceProcedure, connect.NewUnaryHandlerSimple(
		workspaceconnect.WorkspaceServiceGetWorkspaceProcedure,
		func(_ context.Context, req *workspacev1.GetWorkspaceRequest) (*workspacev1.GetWorkspaceResponse, error) {
			reads++
			return &workspacev1.GetWorkspaceResponse{Workspace: &workspacev1.Workspace{Id: req.GetId(), Revision: 7}}, nil
		},
	))
	mux.Handle(workspaceconnect.WorkspaceServiceAssistWorkspaceProcedure, connect.NewUnaryHandlerSimple(
		workspaceconnect.WorkspaceServiceAssistWorkspaceProcedure,
		func(_ context.Context, req *workspacev1.AssistWorkspaceRequest) (*workspacev1.AssistWorkspaceResponse, error) {
			got = req
			return &workspacev1.AssistWorkspaceResponse{
				Revision:     7,
				Operation:    &workspacev1.WorkspaceAssistOperation{Id: "op-1", State: "completed", Provider: "fake", Model: "test-model"},
				ProposalJson: `{"summary":"Rewrite the headline","operations":[{"kind":"setText","nodeId":"hero.title"}]}`,
			}, nil
		},
	))
	core := cliapptest.NewTestApp(t, mux)
	group, err := Register(core, cliapptest.ReadManifest(t))
	require.NoError(t, err)
	command := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "assist")

	out, err := cliapptest.RunCommand(t, command, core,
		"ws-1",
		"--prompt", "Make the headline specific",
		"--scope", "layer",
		"--display", "dp-1",
		"--node", "hero.title",
		"--page", "pg-1",
		"--state", "dp-1=st-1",
	)
	require.NoError(t, err)
	require.Equal(t, 1, reads, "an omitted base revision reads the current one")
	require.NotNil(t, got)
	require.Equal(t, "ws-1", got.GetId())
	require.Equal(t, int32(7), got.GetBaseRevision())
	require.Equal(t, "Make the headline specific", got.GetPrompt())
	require.Equal(t, "layer", got.GetScope().GetKind())
	require.Equal(t, "dp-1", got.GetScope().GetDisplayId())
	require.Equal(t, "hero.title", got.GetScope().GetNodeId())
	require.Equal(t, "pg-1", got.GetPageId())
	require.Len(t, got.GetActiveStates(), 1)
	require.Equal(t, "st-1", got.GetActiveStates()[0].GetStateId())
	require.NotEmpty(t, got.GetIdempotencyKey())

	require.Contains(t, out, "nothing was written")
	require.Contains(t, out, "setText hero.title")
	require.Contains(t, out, "fake · test-model")
}

func TestWorkspacesAssistRejectsAMalformedStateFlag(t *testing.T) {
	core := cliapptest.NewTestApp(t, http.NewServeMux())
	group, err := Register(core, cliapptest.ReadManifest(t))
	require.NoError(t, err)
	command := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "assist")

	_, err = cliapptest.RunCommand(t, command, core, "ws-1", "--prompt", "x", "--state", "nopair")
	require.Error(t, err)
	require.Contains(t, err.Error(), "--state must be display=<state>")
}

func TestWorkspaceDocumentReadersAcceptSupportedShapesAndRejectMalformedInput(t *testing.T) {
	t.Run("workspace proto JSON", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "workspace.json")
		require.NoError(t, os.WriteFile(path, []byte(`{"id":"ws-1","name":"Harbor"}`), 0o600))
		got, err := readWorkspace(path)
		require.NoError(t, err)
		require.Equal(t, "ws-1", got.GetId())
	})
	t.Run("invalid workspace JSON", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "workspace.json")
		require.NoError(t, os.WriteFile(path, []byte(`{"id":`), 0o600))
		_, err := readWorkspace(path)
		require.ErrorContains(t, err, "decode --workspace")
	})
	t.Run("bare page array", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "pages.json")
		require.NoError(t, os.WriteFile(path, []byte(`[{"id":"pg-1","name":"Home"}]`), 0o600))
		got, err := readWorkspacePages(path)
		require.NoError(t, err)
		require.Len(t, got, 1)
		require.Equal(t, "pg-1", got[0].GetId())
	})
	t.Run("wrapped page array", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "pages.json")
		require.NoError(t, os.WriteFile(path, []byte(`{"pages":[{"id":"pg-2"}]}`), 0o600))
		got, err := readWorkspacePages(path)
		require.NoError(t, err)
		require.Equal(t, "pg-2", got[0].GetId())
	})
	tests := []struct{ name, body, want string }{
		{"empty", "  ", "is empty"},
		{"invalid JSON", "{", "decode --pages"},
		{"invalid page proto", `[{"unknown":"field"}]`, "entry 0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "pages.json")
			require.NoError(t, os.WriteFile(path, []byte(tt.body), 0o600))
			_, err := readWorkspacePages(path)
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestWorkspaceFormattingCoversNestedCompositionAndFallbacks(t *testing.T) {
	require.Equal(t, []string{"(nil workspace)"}, formatWorkspaceLines(nil))
	require.Equal(t, "(nil workspace)", formatWorkspaceSummary(nil))
	workspace := seedTestWorkspace(workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED)
	workspace.Name = " "
	workspace.Summary = "A summary"
	workspace.ThemeJson = `{"color":"blue"}`
	workspace.OptionsJson = `{"density":"compact"}`
	workspace.BindingPage = "home"
	workspace.Pages = append(workspace.Pages, nil)
	workspace.Pages[0].Route = ""
	workspace.Pages[0].PageTypeId = ""
	workspace.Pages[0].Displays[0].Device = workspacev1.DisplayDevice_DISPLAY_DEVICE_TABLET
	workspace.Pages[0].Displays[0].States[0].Status = workspacev1.StateStatus_STATE_STATUS_READY
	workspace.Pages[0].Displays[0].States = append(workspace.Pages[0].Displays[0].States, nil)
	workspace.Pages[0].Displays = append(workspace.Pages[0].Displays, nil)
	rows := formatWorkspaceLines(workspace)
	require.Contains(t, rows[0], "(unnamed) (ws_harbor) | curated")
	require.Contains(t, rows[0], "binding=home")
	require.Contains(t, rows[1], "summary: A summary")
	require.Contains(t, rows[2], "theme_json:")
	require.Contains(t, rows[3], "options_json:")
	require.Contains(t, strings.Join(rows, "\n"), `route=(none) type=(none)`)
	require.Contains(t, strings.Join(rows, "\n"), "device=tablet")
	require.Contains(t, strings.Join(rows, "\n"), "status=ready")
	require.Equal(t, "only-page", bindingLabel("", "only-page"))
	require.Equal(t, "only-scenario", bindingLabel("only-scenario", ""))
	require.Equal(t, "", bindingLabel("", ""))
	require.Equal(t, "fallback", nonEmpty("  ", "fallback"))
}

func TestWorkspaceFlagParsersCoverAliasesAndInvalidValues(t *testing.T) {
	for raw, want := range map[string]workspacev1.WorkspaceProvenance{
		"":                              workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_UNSPECIFIED,
		"workspace-provenance-authored": workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_AUTHORED,
		"curated":                       workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED,
		"imported":                      workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_IMPORTED,
		"forked":                        workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_FORKED,
	} {
		got, err := parseProvenance(raw)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
	_, err := parseProvenance("unknown")
	require.ErrorContains(t, err, "--provenance must be one of")
	require.Equal(t, []string{"a", "b"}, cleanValues([]string{" a ", "", "  ", "b"}))
	require.Equal(t, "forked", provenanceLabel(workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_FORKED))
	require.Equal(t, "unspecified", provenanceLabel(999))
	require.Equal(t, "unspecified", deviceLabel(999))
	require.Equal(t, "phone", deviceLabel(workspacev1.DisplayDevice_DISPLAY_DEVICE_PHONE))
	require.Equal(t, "draft", stateStatusLabel(workspacev1.StateStatus_STATE_STATUS_DRAFT))
	require.Equal(t, "unspecified", stateStatusLabel(999))
}

func TestWorkspaceReportsDescribeMutationsAndValidation(t *testing.T) {
	h := &handlers{}
	workspace := seedTestWorkspace(workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_AUTHORED)
	ctx := cliapp.NewTestRunContext(cliapp.TestRunContextOptions{})
	get := h.getReport(ctx, &workspacev1.GetWorkspaceResponse{Workspace: workspace})
	require.Contains(t, get.Summary[0], "Harbor (ws_harbor) at revision 9")
	require.Contains(t, get.RetrievalHints[0], "--base-revision 9")
	created := h.createReport(ctx, &workspacev1.CreateWorkspaceResponse{Workspace: workspace})
	require.Contains(t, created.Result[0], "Created workspace Harbor")
	saved := h.saveReport(ctx, &workspacev1.SaveWorkspaceResponse{Workspace: workspace})
	require.Contains(t, saved.NextCommand[0], "workspaces revisions ws_harbor")
	forked := h.forkReport(ctx, &workspacev1.ForkWorkspaceResponse{Workspace: workspace})
	require.Contains(t, forked.Result[0], "Forked workspace ws_curated_harbor")
	ctx = cliapp.NewTestRunContext(cliapp.TestRunContextOptions{Schema: cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "id", Required: true}},
	}, Positionals: map[string]string{"id": "ws_harbor"}})
	for _, deleted := range []bool{true, false} {
		report := h.deleteReport(ctx, &workspacev1.DeleteWorkspaceResponse{Deleted: deleted})
		if deleted {
			require.Contains(t, report.Result[0], "and its revisions were removed")
		} else {
			require.Contains(t, report.Result[0], "was not present")
		}
	}
	revisions := h.revisionsReport(ctx, &workspacev1.ListWorkspaceRevisionsResponse{Revisions: []*workspacev1.WorkspaceRevision{
		{Revision: 8, SnapshotJson: "{}"}, nil, {Revision: 7, Actor: "operator", CreatedAt: "now", SnapshotJson: `{"id":"ws_harbor"}`},
	}})
	require.Contains(t, revisions.Summary[0], "Found 2 revision(s)")
	require.Contains(t, revisions.Results[0], "actor=(unattributed)")
	require.Contains(t, revisions.Results[1], "actor=operator")
	validation := h.validateReport(ctx, &workspacev1.ValidateWorkspaceResponse{
		Valid: false, Issues: []*workspacev1.ValidationIssue{{Path: "pages[0]", Message: "missing display"}, nil},
	})
	require.Contains(t, validation.Summary[0], "2 validation issue(s)")
	require.Equal(t, []string{"pages[0]: missing display"}, validation.Results)
	valid := h.validateReport(ctx, &workspacev1.ValidateWorkspaceResponse{Valid: true})
	require.Equal(t, "Workspace is valid.", valid.Summary[0])
}

func TestDefaultAssistIdempotencyKeyUsesInjectedClock(t *testing.T) {
	fixedNow := time.Date(2026, time.November, 20, 12, 34, 56, 0, time.UTC)
	h := handlers{deps: runtimeenv.Dependencies{
		Clock:       fixedWorkspaceClock(fixedNow),
		Environment: workspaceEnvironment{},
	}}
	require.Equal(t, "assist-1795178096000000000-iq6q8m", h.defaultIdempotencyKey())
}

func seedTestWorkspace(provenance workspacev1.WorkspaceProvenance) *workspacev1.Workspace {
	return &workspacev1.Workspace{
		Id:                "ws_harbor",
		Name:              "Harbor",
		Summary:           "A threshold, not a funnel.",
		Provenance:        provenance,
		Revision:          9,
		CreatedAt:         "2026-11-18T10:00:00.000Z",
		UpdatedAt:         "2026-11-19T10:00:00.000Z",
		OriginWorkspaceId: "ws_curated_harbor",
		Pages: []*workspacev1.WorkspacePage{{
			Id:       "pg_welcome",
			Name:     "Welcome",
			SpecJson: `{"id":"m.page","type":"page","props":{},"children":[]}`,
			Displays: []*workspacev1.WorkspaceDisplay{{
				Id:     "dp_welcome_desktop",
				Device: workspacev1.DisplayDevice_DISPLAY_DEVICE_DESKTOP,
				Width:  1180,
				Height: 780,
				States: []*workspacev1.WorkspaceState{{
					Id:       "st_welcome_default",
					Name:     "Default",
					ArgsJson: `{"spec":{"id":"m.page","type":"page","props":{},"children":[]}}`,
				}},
			}},
		}},
	}
}

func TestExportSeedPromotesAWorkspaceIntoTheReviewedShelf(t *testing.T) {
	rawManifest := cliapptest.ReadManifest(t)
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "api"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "catalog"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "library"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "api", "go.mod"), []byte("module example\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "catalog", "config.json"), []byte("{}"), 0o644))
	seedFile := filepath.Join(root, "library", "workspace-seeds.json")
	require.NoError(t, os.WriteFile(seedFile, []byte(`{
  "schemaVersion": 1,
  "seedVersion": 4,
  "description": "reviewed",
  "workspaces": [
    {"id": "ws_curated_other", "name": "Other"}
  ]
}
`), 0o644))
	t.Chdir(root)

	var workspace *workspacev1.Workspace
	mux := http.NewServeMux()
	mux.Handle(workspaceconnect.WorkspaceServiceGetWorkspaceProcedure, connect.NewUnaryHandlerSimple(
		workspaceconnect.WorkspaceServiceGetWorkspaceProcedure,
		func(_ context.Context, req *workspacev1.GetWorkspaceRequest) (*workspacev1.GetWorkspaceResponse, error) {
			require.Equal(t, "ws_harbor", req.GetId())
			return &workspacev1.GetWorkspaceResponse{Workspace: workspace}, nil
		},
	))
	core := cliapptest.NewTestApp(t, mux)
	fixedNow := time.Date(2026, time.November, 20, 12, 34, 56, 0, time.UTC)
	deps := runtimeenv.Dependencies{
		Clock:       fixedWorkspaceClock(fixedNow),
		Environment: workspaceEnvironment{"VROOLI_SCENARIO_ROOT": root},
	}
	group, err := registerWithRuntime(core, rawManifest, deps)
	require.NoError(t, err)
	command := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "export-seed")

	workspace = seedTestWorkspace(workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_FORKED)
	out, err := cliapptest.RunCommand(t, command, core,
		"ws_harbor", "--seed-id", "ws_curated_harbor",
	)
	require.NoError(t, err)
	require.Contains(t, out, "Added")
	require.Contains(t, out, "seedVersion 5")
	require.Contains(t, out, "Nothing was restarted")

	raw, err := os.ReadFile(seedFile)
	require.NoError(t, err)
	var payload struct {
		SchemaVersion int               `json:"schemaVersion"`
		SeedVersion   int               `json:"seedVersion"`
		Workspaces    []json.RawMessage `json:"workspaces"`
	}
	require.NoError(t, json.Unmarshal(raw, &payload))
	require.Equal(t, 1, payload.SchemaVersion)
	require.Equal(t, 5, payload.SeedVersion)
	require.Len(t, payload.Workspaces, 2)

	var exported workspacev1.Workspace
	require.NoError(t, protojson.Unmarshal(payload.Workspaces[1], &exported))
	require.Equal(t, "ws_curated_harbor", exported.GetId())
	require.Equal(t, "Harbor", exported.GetName())
	require.Equal(t, workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED, exported.GetProvenance())
	require.Equal(t, int32(1), exported.GetRevision())
	require.Equal(t, "2026-11-20T12:34:56.000Z", exported.GetCreatedAt())
	require.Equal(t, exported.GetCreatedAt(), exported.GetUpdatedAt())
	require.Empty(t, exported.GetOriginWorkspaceId())
	require.Empty(t, exported.GetBindingScenario())
	require.Len(t, exported.GetPages(), 1)
	require.Equal(t, "pg_welcome", exported.GetPages()[0].GetId())

	// Re-exporting the same slot replaces it instead of duplicating the shelf.
	out, err = cliapptest.RunCommand(t, command, core,
		"ws_harbor", "--seed-id", "ws_curated_harbor",
	)
	require.NoError(t, err)
	require.Contains(t, out, "Updated")
	require.Contains(t, out, "seedVersion 6")
	require.NoError(t, json.Unmarshal(mustRead(t, seedFile), &payload))
	require.Len(t, payload.Workspaces, 2)
}

func TestExportSeedRefusesCuratedSourcesAndNakedIds(t *testing.T) {
	mux := http.NewServeMux()
	var workspace *workspacev1.Workspace
	mux.Handle(workspaceconnect.WorkspaceServiceGetWorkspaceProcedure, connect.NewUnaryHandlerSimple(
		workspaceconnect.WorkspaceServiceGetWorkspaceProcedure,
		func(_ context.Context, _ *workspacev1.GetWorkspaceRequest) (*workspacev1.GetWorkspaceResponse, error) {
			return &workspacev1.GetWorkspaceResponse{Workspace: workspace}, nil
		},
	))
	core := cliapptest.NewTestApp(t, mux)
	group, err := Register(core, cliapptest.ReadManifest(t))
	require.NoError(t, err)
	command := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "export-seed")

	workspace = seedTestWorkspace(workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED)
	_, err = cliapptest.RunCommand(t, command, core, "ws_harbor")
	require.ErrorContains(t, err, "already a curated template")

	workspace = seedTestWorkspace(workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_AUTHORED)
	_, err = cliapptest.RunCommand(t, command, core, "ws_harbor")
	require.ErrorContains(t, err, "--seed-id is required")

	_, err = cliapptest.RunCommand(t, command, core, "ws_harbor", "--seed-id", "harbor")
	require.ErrorContains(t, err, "must start with ws_curated_")
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	return raw
}

func TestExportSeedRefusesACompositionTheStudioCannotAddress(t *testing.T) {
	mux := http.NewServeMux()
	var workspace *workspacev1.Workspace
	mux.Handle(workspaceconnect.WorkspaceServiceGetWorkspaceProcedure, connect.NewUnaryHandlerSimple(
		workspaceconnect.WorkspaceServiceGetWorkspaceProcedure,
		func(_ context.Context, _ *workspacev1.GetWorkspaceRequest) (*workspacev1.GetWorkspaceResponse, error) {
			return &workspacev1.GetWorkspaceResponse{Workspace: workspace}, nil
		},
	))
	core := cliapptest.NewTestApp(t, mux)
	group, err := Register(core, cliapptest.ReadManifest(t))
	require.NoError(t, err)
	command := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "export-seed")

	workspace = seedTestWorkspace(workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_AUTHORED)
	workspace.Pages[0].Displays[0].States[0].ArgsJson = `{"spec":{"id":"root","type":"page","props":{},"children":[{"id":"same","type":"text","props":{}},{"id":"same","type":"text","props":{}}]}}`
	_, err = cliapptest.RunCommand(t, command, core, "ws_harbor", "--seed-id", "ws_curated_harbor")
	require.ErrorContains(t, err, `layer id "same"`)

	workspace = seedTestWorkspace(workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_AUTHORED)
	workspace.Pages[0].SpecJson = "{}"
	workspace.Pages[0].Displays[0].States[0].ArgsJson = "{}"
	_, err = cliapptest.RunCommand(t, command, core, "ws_harbor", "--seed-id", "ws_curated_harbor")
	require.ErrorContains(t, err, "none of its states has a composition")
}
