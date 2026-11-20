package workspaces

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	db "github.com/vrooli/api-core/databasetest"

	workspacev1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspace"
)

func newTestRepository(t *testing.T) *SQLiteRepository {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	if _, err := sqlDB.ExecContext(context.Background(), Schema()); err != nil {
		t.Fatalf("apply schema: %v", err)
	}
	repo := NewSQLiteRepository(sqlDB)
	repo.now = func() time.Time { return time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC) }
	return repo
}

func desktopDisplay(width, height int32, states ...*workspacev1.WorkspaceState) *workspacev1.WorkspaceDisplay {
	return &workspacev1.WorkspaceDisplay{
		Device: workspacev1.DisplayDevice_DISPLAY_DEVICE_DESKTOP,
		Width:  width,
		Height: height,
		States: states,
	}
}

func TestCreateSeedsDefaultComposition(t *testing.T) {
	repo := newTestRepository(t)

	workspace, err := repo.Create(context.Background(), &workspacev1.CreateWorkspaceRequest{Name: "  Harbor dashboard  "})
	require.NoError(t, err)

	require.True(t, strings.HasPrefix(workspace.Id, "ws_"))
	require.Equal(t, "Harbor dashboard", workspace.Name)
	require.Equal(t, workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_AUTHORED, workspace.Provenance)
	require.Equal(t, int32(1), workspace.Revision)
	require.Equal(t, "2026-11-19T12:00:00.000Z", workspace.CreatedAt)
	require.Equal(t, workspace.CreatedAt, workspace.UpdatedAt)
	require.JSONEq(t, `{}`, workspace.ThemeJson)
	require.JSONEq(t, `{}`, workspace.OptionsJson)

	require.Len(t, workspace.Pages, 1)
	page := workspace.Pages[0]
	require.True(t, strings.HasPrefix(page.Id, "pg_"))
	require.Equal(t, "Page 1", page.Name)
	require.Len(t, page.Displays, 1)
	display := page.Displays[0]
	require.True(t, strings.HasPrefix(display.Id, "dp_"))
	require.Equal(t, workspacev1.DisplayDevice_DISPLAY_DEVICE_DESKTOP, display.Device)
	require.Equal(t, int32(1180), display.Width)
	require.Equal(t, int32(780), display.Height)
	require.Len(t, display.States, 1)
	state := display.States[0]
	require.True(t, strings.HasPrefix(state.Id, "st_"))
	require.Equal(t, "Default", state.Name)
	require.Equal(t, workspacev1.StateStatus_STATE_STATUS_DRAFT, state.Status)
	require.JSONEq(t, `{}`, state.ArgsJson)

	revisions, err := repo.Revisions(context.Background(), workspace.Id, 0)
	require.NoError(t, err)
	require.Len(t, revisions, 1, "create records its revision 1 snapshot")
}

func TestCreateAssignsIDsAndNormalizesOrder(t *testing.T) {
	repo := newTestRepository(t)

	workspace, err := repo.Create(context.Background(), &workspacev1.CreateWorkspaceRequest{
		Name: "Multi",
		Pages: []*workspacev1.WorkspacePage{
			{
				Name: "First", SortOrder: 9,
				Displays: []*workspacev1.WorkspaceDisplay{
					{
						Device: workspacev1.DisplayDevice_DISPLAY_DEVICE_PHONE, SortOrder: 7,
						States: []*workspacev1.WorkspaceState{{Name: "Empty"}, {Name: "Loading"}},
					},
					{
						Device: workspacev1.DisplayDevice_DISPLAY_DEVICE_DESKTOP, SortOrder: 2,
						States: []*workspacev1.WorkspaceState{{Name: "Default"}},
					},
				},
			},
			{
				Name: "Second", SortOrder: 4,
				Displays: []*workspacev1.WorkspaceDisplay{
					{
						Device: workspacev1.DisplayDevice_DISPLAY_DEVICE_TABLET,
						States: []*workspacev1.WorkspaceState{{Name: "Error", Status: workspacev1.StateStatus_STATE_STATUS_READY}},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, workspace.Pages, 2)

	seen := map[string]bool{}
	for pageIndex, page := range workspace.Pages {
		require.Equal(t, int32(pageIndex), page.SortOrder)
		require.True(t, strings.HasPrefix(page.Id, "pg_"))
		seen[page.Id] = true
		for displayIndex, display := range page.Displays {
			require.Equal(t, int32(displayIndex), display.SortOrder)
			require.True(t, strings.HasPrefix(display.Id, "dp_"))
			seen[display.Id] = true
			for stateIndex, state := range display.States {
				require.Equal(t, int32(stateIndex), state.SortOrder)
				require.True(t, strings.HasPrefix(state.Id, "st_"))
				seen[state.Id] = true
			}
		}
	}
	require.Len(t, seen, 2+3+4, "every nested node gets a distinct id")

	got, err := repo.Get(context.Background(), workspace.Id)
	require.NoError(t, err)
	require.Equal(t, workspace.Name, got.Name)
	require.Equal(t, int32(0), got.Pages[0].SortOrder)
	require.Equal(t, "Error", got.Pages[1].Displays[0].States[0].Name)
	require.Equal(t, workspacev1.StateStatus_STATE_STATUS_READY, got.Pages[1].Displays[0].States[0].Status)
}

func TestGetRoundTripsAggregateFields(t *testing.T) {
	repo := newTestRepository(t)
	workspace, err := repo.Create(context.Background(), &workspacev1.CreateWorkspaceRequest{
		Name:            "Harbor",
		Summary:         "Onboarding flow",
		Provenance:      workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED,
		Tags:            []string{"onboarding", "harbor"},
		BindingScenario: "agent-inbox",
		BindingPage:     "inbox",
		ThemeJson:       `{"ink":"#111111","accent":"#ff0000"}`,
		OptionsJson:     `{"radius":24}`,
	})
	require.NoError(t, err)

	got, err := repo.Get(context.Background(), workspace.Id)
	require.NoError(t, err)
	require.Equal(t, "Onboarding flow", got.Summary)
	require.Equal(t, workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED, got.Provenance)
	require.Equal(t, []string{"onboarding", "harbor"}, got.Tags)
	require.Equal(t, "agent-inbox", got.BindingScenario)
	require.Equal(t, "inbox", got.BindingPage)
	require.JSONEq(t, `{"ink":"#111111","accent":"#ff0000"}`, got.ThemeJson)
	require.JSONEq(t, `{"radius":24}`, got.OptionsJson)
}

func TestGetMissingReturnsNotFound(t *testing.T) {
	repo := newTestRepository(t)
	_, err := repo.Get(context.Background(), "ws_missing")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestSaveReplacesNestedRowsBumpsRevisionAndSnapshots(t *testing.T) {
	repo := newTestRepository(t)
	created, err := repo.Create(context.Background(), &workspacev1.CreateWorkspaceRequest{
		Name: "Draft",
		Pages: []*workspacev1.WorkspacePage{{
			Name: "Only",
			Displays: []*workspacev1.WorkspaceDisplay{
				desktopDisplay(1440, 900, &workspacev1.WorkspaceState{Name: "Default"}),
			},
		}},
	})
	require.NoError(t, err)

	repo.now = func() time.Time { return time.Date(2026, 11, 19, 13, 5, 0, 0, time.UTC) }
	created.Name = "Published"
	// A client may submit new nodes without ids; Save assigns them, just like
	// Create does.
	created.Pages[0].Displays[0].States[0].Id = ""
	created.Pages = append(created.Pages, &workspacev1.WorkspacePage{
		Name: "Onboarding",
		Displays: []*workspacev1.WorkspaceDisplay{
			desktopDisplay(1180, 780, &workspacev1.WorkspaceState{Name: "Empty"}),
			{
				Device: workspacev1.DisplayDevice_DISPLAY_DEVICE_PHONE, Width: 390, Height: 844,
				States: []*workspacev1.WorkspaceState{{Name: "Default"}, {Name: "Loading"}},
			},
		},
	})

	saved, err := repo.Save(context.Background(), &workspacev1.SaveWorkspaceRequest{
		Workspace:    created,
		BaseRevision: created.Revision,
		Actor:        "operator@example.test",
	})
	require.NoError(t, err)

	require.Equal(t, int32(2), saved.Revision)
	require.Equal(t, "Published", saved.Name)
	require.Equal(t, "2026-11-19T13:05:00.000Z", saved.UpdatedAt)
	require.Equal(t, created.CreatedAt, saved.CreatedAt)
	require.Len(t, saved.Pages, 2)
	require.Len(t, saved.Pages[1].Displays, 2)
	require.True(t, strings.HasPrefix(saved.Pages[0].Displays[0].States[0].Id, "st_"),
		"a save assigns ids to submitted nodes that lack one")

	reloaded, err := repo.Get(context.Background(), saved.Id)
	require.NoError(t, err)
	require.True(t, proto.Equal(saved, reloaded), "the aggregate must round-trip unchanged")

	revisions, err := repo.Revisions(context.Background(), saved.Id, 0)
	require.NoError(t, err)
	require.Len(t, revisions, 2)
	require.Equal(t, int32(2), revisions[0].Revision, "newest revision first")
	require.Equal(t, int32(1), revisions[1].Revision)
	require.Equal(t, "operator@example.test", revisions[0].Actor)
	require.Contains(t, revisions[0].SnapshotJson, `"revision":2`)
	require.Contains(t, revisions[0].SnapshotJson, `"name":"Published"`)
}

func TestSaveRejectsStaleRevisionWithoutWriting(t *testing.T) {
	repo := newTestRepository(t)
	created, err := repo.Create(context.Background(), &workspacev1.CreateWorkspaceRequest{Name: "Base"})
	require.NoError(t, err)
	base := created.Revision

	first := cloneWorkspace(created)
	first.Name = "First"
	_, err = repo.Save(context.Background(), &workspacev1.SaveWorkspaceRequest{Workspace: first, BaseRevision: base})
	require.NoError(t, err)

	stale := cloneWorkspace(created)
	stale.Name = "Loser"
	_, err = repo.Save(context.Background(), &workspacev1.SaveWorkspaceRequest{Workspace: stale, BaseRevision: base})
	require.ErrorIs(t, err, ErrConflict)
	var conflict *ConflictError
	require.ErrorAs(t, err, &conflict)
	require.NotNil(t, conflict.Current)
	require.Equal(t, int32(2), conflict.Current.Revision)
	require.Equal(t, "First", conflict.Current.Name)

	unchanged, err := repo.Get(context.Background(), created.Id)
	require.NoError(t, err)
	require.Equal(t, "First", unchanged.Name)
	require.Equal(t, int32(2), unchanged.Revision)
}

func TestSaveRequiresBaseRevisionAndDocument(t *testing.T) {
	repo := newTestRepository(t)
	created, err := repo.Create(context.Background(), &workspacev1.CreateWorkspaceRequest{Name: "Base"})
	require.NoError(t, err)

	_, err = repo.Save(context.Background(), &workspacev1.SaveWorkspaceRequest{Workspace: created})
	require.ErrorIs(t, err, ErrInvalid)

	_, err = repo.Save(context.Background(), nil)
	require.ErrorIs(t, err, ErrInvalid)

	_, err = repo.Save(context.Background(), &workspacev1.SaveWorkspaceRequest{
		Workspace:    &workspacev1.Workspace{Id: "ws_missing", Name: "Missing"},
		BaseRevision: 1,
	})
	require.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteCascadesNestedRowsAndRevisions(t *testing.T) {
	repo := newTestRepository(t)
	created, err := repo.Create(context.Background(), &workspacev1.CreateWorkspaceRequest{Name: "Disposable"})
	require.NoError(t, err)
	stateID := created.Pages[0].Displays[0].States[0].Id

	require.NoError(t, repo.Delete(context.Background(), created.Id))
	_, err = repo.Get(context.Background(), created.Id)
	require.ErrorIs(t, err, ErrNotFound)

	for table, column := range map[string]string{
		"workspace_pages":     "workspace_id",
		"workspace_revisions": "workspace_id",
	} {
		var count int
		require.NoError(t, repo.db.QueryRowContext(context.Background(),
			`SELECT COUNT(*) FROM `+table+` WHERE `+column+` = ?`, created.Id).Scan(&count))
		require.Zero(t, count, "%s must not outlive the workspace", table)
	}
	var states int
	require.NoError(t, repo.db.QueryRowContext(context.Background(),
		`SELECT COUNT(*) FROM workspace_states WHERE id = ?`, stateID).Scan(&states))
	require.Zero(t, states)

	require.ErrorIs(t, repo.Delete(context.Background(), created.Id), ErrNotFound)
}

func TestForkCopiesAggregateWithNewIdentity(t *testing.T) {
	repo := newTestRepository(t)
	created, err := repo.Create(context.Background(), &workspacev1.CreateWorkspaceRequest{
		Name:       "Curated template",
		Provenance: workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED,
		Tags:       []string{"landing"},
		Pages: []*workspacev1.WorkspacePage{{
			Name:     "Landing",
			Route:    "/",
			SpecJson: `{"root":{"kind":"page"}}`,
			Displays: []*workspacev1.WorkspaceDisplay{
				desktopDisplay(1180, 780, &workspacev1.WorkspaceState{Name: "Default", ArgsJson: `{"hero":"none"}`}),
			},
		}},
	})
	require.NoError(t, err)

	fork, err := repo.Fork(context.Background(), created.Id, "")
	require.NoError(t, err)

	require.True(t, strings.HasPrefix(fork.Id, "ws_"))
	require.NotEqual(t, created.Id, fork.Id)
	require.Equal(t, "Copy of Curated template", fork.Name)
	require.Equal(t, workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_FORKED, fork.Provenance)
	require.Equal(t, created.Id, fork.OriginWorkspaceId)
	require.Equal(t, int32(1), fork.Revision)
	require.Equal(t, []string{"landing"}, fork.Tags)

	require.Len(t, fork.Pages, 1)
	require.NotEqual(t, created.Pages[0].Id, fork.Pages[0].Id)
	require.Equal(t, "Landing", fork.Pages[0].Name)
	require.JSONEq(t, `{"root":{"kind":"page"}}`, fork.Pages[0].SpecJson)
	require.NotEqual(t, created.Pages[0].Displays[0].Id, fork.Pages[0].Displays[0].Id)
	require.NotEqual(t, created.Pages[0].Displays[0].States[0].Id, fork.Pages[0].Displays[0].States[0].Id)
	require.JSONEq(t, `{"hero":"none"}`, fork.Pages[0].Displays[0].States[0].ArgsJson)

	// The origin is untouched.
	origin, err := repo.Get(context.Background(), created.Id)
	require.NoError(t, err)
	require.Equal(t, workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED, origin.Provenance)
	require.Equal(t, created.Pages[0].Id, origin.Pages[0].Id)
}

func TestForkUsesRequestedName(t *testing.T) {
	repo := newTestRepository(t)
	created, err := repo.Create(context.Background(), &workspacev1.CreateWorkspaceRequest{Name: "Origin"})
	require.NoError(t, err)

	fork, err := repo.Fork(context.Background(), created.Id, "  My fork  ")
	require.NoError(t, err)
	require.Equal(t, "My fork", fork.Name)
}

func TestListOrdersFiltersAndCounts(t *testing.T) {
	repo := newTestRepository(t)
	repo.now = func() time.Time { return time.Date(2026, 11, 19, 9, 0, 0, 0, time.UTC) }
	older, err := repo.Create(context.Background(), &workspacev1.CreateWorkspaceRequest{
		Name: "Older harbor", Tags: []string{"onboarding"},
	})
	require.NoError(t, err)

	repo.now = func() time.Time { return time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC) }
	newer, err := repo.Create(context.Background(), &workspacev1.CreateWorkspaceRequest{
		Name: "Newer", Summary: "billing flow",
		Provenance: workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED,
		Pages: []*workspacev1.WorkspacePage{{
			Name: "A",
			Displays: []*workspacev1.WorkspaceDisplay{
				desktopDisplay(1180, 780, &workspacev1.WorkspaceState{Name: "Default"}),
				{
					Device: workspacev1.DisplayDevice_DISPLAY_DEVICE_PHONE,
					States: []*workspacev1.WorkspaceState{{Name: "Default"}, {Name: "Empty"}},
				},
			},
		}},
	})
	require.NoError(t, err)

	all, err := repo.List(context.Background(), workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_UNSPECIFIED, "", 0)
	require.NoError(t, err)
	require.Len(t, all, 2)
	require.Equal(t, newer.Id, all[0].Id)
	require.Equal(t, older.Id, all[1].Id)
	require.Equal(t, int32(1), all[0].PageCount)
	require.Equal(t, int32(2), all[0].DisplayCount)
	require.Equal(t, int32(3), all[0].StateCount)

	curated, err := repo.List(context.Background(), workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_CURATED, "", 0)
	require.NoError(t, err)
	require.Len(t, curated, 1)
	require.Equal(t, newer.Id, curated[0].Id)

	byName, err := repo.List(context.Background(), workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_UNSPECIFIED, "HARBOR", 0)
	require.NoError(t, err)
	require.Len(t, byName, 1)
	require.Equal(t, older.Id, byName[0].Id)

	byTag, err := repo.List(context.Background(), workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_UNSPECIFIED, "onboarding", 0)
	require.NoError(t, err)
	require.Len(t, byTag, 1)
	require.Equal(t, older.Id, byTag[0].Id)

	bySummary, err := repo.List(context.Background(), workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_UNSPECIFIED, "billing", 0)
	require.NoError(t, err)
	require.Len(t, bySummary, 1)
	require.Equal(t, newer.Id, bySummary[0].Id)

	limited, err := repo.List(context.Background(), workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_UNSPECIFIED, "", 1)
	require.NoError(t, err)
	require.Len(t, limited, 1)

	_, err = repo.List(context.Background(), workspacev1.WorkspaceProvenance(99), "", 0)
	require.ErrorIs(t, err, ErrInvalid)
}

func TestListTreatsWildcardsAsLiterals(t *testing.T) {
	repo := newTestRepository(t)
	literal, err := repo.Create(context.Background(), &workspacev1.CreateWorkspaceRequest{Name: "100% Coverage"})
	require.NoError(t, err)
	_, err = repo.Create(context.Background(), &workspacev1.CreateWorkspaceRequest{Name: "100x Coverage"})
	require.NoError(t, err)

	matched, err := repo.List(context.Background(), workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_UNSPECIFIED, "100%", 0)
	require.NoError(t, err)
	require.Len(t, matched, 1, "%% must be a literal, not a wildcard")
	require.Equal(t, literal.Id, matched[0].Id)

	none, err := repo.List(context.Background(), workspacev1.WorkspaceProvenance_WORKSPACE_PROVENANCE_UNSPECIFIED, "100_", 0)
	require.NoError(t, err)
	require.Empty(t, none, "_ must be a literal, not a single-character wildcard")
}

func TestRevisionsAreBoundedAndRequireWorkspace(t *testing.T) {
	repo := newTestRepository(t)
	created, err := repo.Create(context.Background(), &workspacev1.CreateWorkspaceRequest{Name: "History"})
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		doc := cloneWorkspace(created)
		doc.Name = "History " + string(rune('A'+i))
		_, err := repo.Save(context.Background(), &workspacev1.SaveWorkspaceRequest{
			Workspace:    doc,
			BaseRevision: doc.Revision,
		})
		require.NoError(t, err)
		created, err = repo.Get(context.Background(), created.Id)
		require.NoError(t, err)
	}

	revisions, err := repo.Revisions(context.Background(), created.Id, 2)
	require.NoError(t, err)
	require.Len(t, revisions, 2)
	require.Equal(t, int32(4), revisions[0].Revision)
	require.Equal(t, int32(3), revisions[1].Revision)

	_, err = repo.Revisions(context.Background(), "ws_missing", 0)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestValidateReturnsPathSpecificIssues(t *testing.T) {
	repo := newTestRepository(t)

	issues, err := repo.Validate(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, issues, 1)
	require.Equal(t, "workspace", issues[0].Path)

	workspace := &workspacev1.Workspace{
		Id:   "ws_a",
		Name: "  ",
		Pages: []*workspacev1.WorkspacePage{
			{
				Id: "pg_a",
				Displays: []*workspacev1.WorkspaceDisplay{
					{
						Id:     "dp_a",
						Device: workspacev1.DisplayDevice_DISPLAY_DEVICE_UNSPECIFIED,
						States: []*workspacev1.WorkspaceState{
							{Id: "st_a", Name: "Loading"},
							{Id: "st_b", Name: "Loading"},
						},
					},
					{Id: "dp_a", Device: workspacev1.DisplayDevice_DISPLAY_DEVICE_PHONE},
				},
			},
			{Id: "pg_a"},
		},
	}
	issues, err = repo.Validate(context.Background(), workspace)
	require.NoError(t, err)

	paths := make(map[string]bool, len(issues))
	for _, issue := range issues {
		paths[issue.Path] = true
	}
	require.True(t, paths["name"])
	require.True(t, paths["pages[0].displays[0].device"])
	require.True(t, paths["pages[0].displays[0].states[1].name"])
	require.True(t, paths["pages[0].displays[1].id"])
	require.True(t, paths["pages[1].id"])

	invalid := invalidFromIssues(issues)
	require.ErrorIs(t, invalid, ErrInvalid)
	var typed *InvalidError
	require.ErrorAs(t, invalid, &typed)
	require.Len(t, typed.Issues, len(issues))
}

func TestCreateRejectsInvalidAggregates(t *testing.T) {
	repo := newTestRepository(t)

	_, err := repo.Create(context.Background(), &workspacev1.CreateWorkspaceRequest{Name: "   "})
	require.ErrorIs(t, err, ErrInvalid)

	_, err = repo.Create(context.Background(), &workspacev1.CreateWorkspaceRequest{
		Name: "Bad device",
		Pages: []*workspacev1.WorkspacePage{{
			Name: "Page",
			Displays: []*workspacev1.WorkspaceDisplay{{
				Device: workspacev1.DisplayDevice_DISPLAY_DEVICE_UNSPECIFIED,
			}},
		}},
	})
	require.ErrorIs(t, err, ErrInvalid)

	_, err = repo.Create(context.Background(), &workspacev1.CreateWorkspaceRequest{
		Name: "Duplicate state names",
		Pages: []*workspacev1.WorkspacePage{{
			Name: "Page",
			Displays: []*workspacev1.WorkspaceDisplay{{
				Device: workspacev1.DisplayDevice_DISPLAY_DEVICE_DESKTOP,
				States: []*workspacev1.WorkspaceState{{Name: "Default"}, {Name: "Default"}},
			}},
		}},
	})
	require.ErrorIs(t, err, ErrInvalid)

	var count int
	require.NoError(t, repo.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM workspaces`).Scan(&count))
	require.Zero(t, count, "a rejected create writes nothing")
}

type recordingRecorder struct {
	actions []string
	before  [][]byte
	after   [][]byte
	err     error
}

func (r *recordingRecorder) Record(_ context.Context, action, _ string, before, after []byte) error {
	r.actions = append(r.actions, action)
	r.before = append(r.before, before)
	r.after = append(r.after, after)
	return r.err
}

func TestRecorderReceivesCreateSaveForkDelete(t *testing.T) {
	repo := newTestRepository(t)
	recorder := &recordingRecorder{}
	repo.WithRecorder(recorder)
	ctx := context.Background()

	created, err := repo.Create(ctx, &workspacev1.CreateWorkspaceRequest{Name: "Recorded"})
	require.NoError(t, err)

	doc := cloneWorkspace(created)
	doc.Name = "Recorded v2"
	saved, err := repo.Save(ctx, &workspacev1.SaveWorkspaceRequest{Workspace: doc, BaseRevision: doc.Revision})
	require.NoError(t, err)

	_, err = repo.Fork(ctx, saved.Id, "")
	require.NoError(t, err)
	require.NoError(t, repo.Delete(ctx, created.Id))

	require.Equal(t, []string{"workspace.created", "workspace.saved", "workspace.forked", "workspace.deleted"}, recorder.actions)

	var createdSnapshot map[string]any
	require.NoError(t, json.Unmarshal(recorder.after[0], &createdSnapshot))
	require.Equal(t, "Recorded", createdSnapshot["name"])

	var beforeSave map[string]any
	require.NoError(t, json.Unmarshal(recorder.before[1], &beforeSave))
	require.Equal(t, "Recorded", beforeSave["name"])
	var afterSave map[string]any
	require.NoError(t, json.Unmarshal(recorder.after[1], &afterSave))
	require.Equal(t, "Recorded v2", afterSave["name"])

	require.NotEmpty(t, recorder.before[3])
	require.Nil(t, recorder.after[3])
}

func TestRecorderFailureIsReportedAfterCommit(t *testing.T) {
	repo := newTestRepository(t)
	repo.WithRecorder(&recordingRecorder{err: errors.New("activity store offline")})

	_, err := repo.Create(context.Background(), &workspacev1.CreateWorkspaceRequest{Name: "Committed"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "after commit")

	var count int
	require.NoError(t, repo.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM workspaces`).Scan(&count))
	require.Equal(t, 1, count, "the mutation is committed; only the activity record failed")
}

func TestNilRecorderIsAllowed(t *testing.T) {
	repo := newTestRepository(t)
	require.Nil(t, repo.recorder)

	created, err := repo.Create(context.Background(), &workspacev1.CreateWorkspaceRequest{Name: "Quiet"})
	require.NoError(t, err)
	require.Equal(t, "Quiet", created.Name)
}

func TestCreateWithUnknownProvenanceRejected(t *testing.T) {
	repo := newTestRepository(t)
	_, err := repo.Create(context.Background(), &workspacev1.CreateWorkspaceRequest{
		Name:       "Bad provenance",
		Provenance: workspacev1.WorkspaceProvenance(99),
	})
	require.ErrorIs(t, err, ErrInvalid)
}
