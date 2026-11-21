package designs

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	db "github.com/vrooli/api-core/databasetest"
)

func newTestRepository(t *testing.T) *sqliteRepository {
	t.Helper()
	sqlDB := db.NewSQLite(t)
	_, err := sqlDB.ExecContext(context.Background(), Schema())
	require.NoError(t, err)
	return NewSQLiteRepository(sqlDB).(*sqliteRepository)
}

func TestCreateAssignsIdentityAndDefaults(t *testing.T) {
	repo := newTestRepository(t)
	repo.now = func() time.Time { return time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC) }

	doc, err := repo.Create(context.Background(), CreateInput{Name: "  Untitled design  "})
	require.NoError(t, err)

	require.True(t, strings.HasPrefix(doc.ID, "design-"))
	require.NotEqual(t, "design-", doc.ID)
	require.Equal(t, "Untitled design", doc.Name)
	require.Equal(t, 1, doc.Revision)
	require.Equal(t, "2026-11-19T12:00:00.000Z", doc.UpdatedAt)
	require.JSONEq(t, `[]`, string(doc.Frames))
	require.JSONEq(t, `{}`, string(doc.Theme))
	require.JSONEq(t, `{}`, string(doc.Options))
	require.Nil(t, doc.Scenario)
	require.Nil(t, doc.Page)
}

func TestOptionsRoundTripOnCreateUpdateAndGet(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, CreateInput{
		Name:    "Chrome",
		Options: json.RawMessage(`{"radius":24,"display":"serif"}`),
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"radius":24,"display":"serif"}`, string(created.Options))

	got, err := repo.Get(ctx, created.ID)
	require.NoError(t, err)
	require.JSONEq(t, `{"radius":24,"display":"serif"}`, string(got.Options))

	revision := created.Revision
	updated, err := repo.Save(ctx, created.ID, UpdateInput{
		Name:     created.Name,
		Frames:   created.Frames,
		Theme:    created.Theme,
		Options:  json.RawMessage(`{"radius":4}`),
		Revision: &revision,
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"radius":4}`, string(updated.Options), "a save replaces the whole options object")

	reloaded, err := repo.Get(ctx, created.ID)
	require.NoError(t, err)
	require.JSONEq(t, `{"radius":4}`, string(reloaded.Options))
}

func TestOptionsAbsentOrNullMeansEmptyObject(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	absent, err := repo.Create(ctx, CreateInput{Name: "Absent"})
	require.NoError(t, err)
	require.JSONEq(t, `{}`, string(absent.Options))

	explicitNull, err := repo.Create(ctx, CreateInput{Name: "Null", Options: json.RawMessage(`null`)})
	require.NoError(t, err)
	require.JSONEq(t, `{}`, string(explicitNull.Options))

	// An old PUT body without options still saves and resets chrome to {}.
	revision := explicitNull.Revision
	saved, err := repo.Save(ctx, explicitNull.ID, UpdateInput{Name: "Null", Revision: &revision})
	require.NoError(t, err)
	require.JSONEq(t, `{}`, string(saved.Options))
}

func TestOptionsRejectsInvalidDocuments(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	oversize := `{"radius":8,"note":"` + strings.Repeat("x", maxOptionsBytes) + `"}`
	cases := map[string]string{
		"array":        `[]`,
		"string":       `"radius"`,
		"unknown key":  `{"shadow":true}`,
		"bad radius":   `{"radius":49}`,
		"low radius":   `{"radius":-1}`,
		"radius text":  `{"radius":"12"}`,
		"radius null":  `{"radius":null}`,
		"bad display":  `{"display":"mono"}`,
		"display null": `{"display":null}`,
		"oversized":    oversize,
	}
	for name, options := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := repo.Create(ctx, CreateInput{Name: "Bad", Options: json.RawMessage(options)})
			require.ErrorIs(t, err, ErrInvalid)
		})
	}

	// A valid edge radius is accepted.
	_, err := repo.Create(ctx, CreateInput{Name: "Edge", Options: json.RawMessage(`{"radius":0}`)})
	require.NoError(t, err)
	_, err = repo.Create(ctx, CreateInput{Name: "Edge", Options: json.RawMessage(`{"radius":48}`)})
	require.NoError(t, err)
}

func TestCreatePersistsFramesOpaquely(t *testing.T) {
	repo := newTestRepository(t)
	frames := json.RawMessage(`[
		{"id":"frame-1","name":"Desktop","device":"desktop","spec":{"root":{"kind":"row","children":[]}},"clientOnly":{"keep":"me"}}
	]`)

	doc, err := repo.Create(context.Background(), CreateInput{
		Name:   "Landing",
		Frames: frames,
		Theme:  json.RawMessage(`{"ink":"#111111","accent":"#ff0000"}`),
	})
	require.NoError(t, err)

	var restored []map[string]any
	require.NoError(t, json.Unmarshal(doc.Frames, &restored))
	require.Len(t, restored, 1)
	require.Equal(t, "frame-1", restored[0]["id"])
	require.Equal(t, map[string]any{"keep": "me"}, restored[0]["clientOnly"],
		"unknown frame fields must survive a round trip")

	require.JSONEq(t, `{"ink":"#111111","accent":"#ff0000"}`, string(doc.Theme))

	got, err := repo.Get(context.Background(), doc.ID)
	require.NoError(t, err)
	require.JSONEq(t, string(doc.Frames), string(got.Frames))
	require.JSONEq(t, string(doc.Theme), string(got.Theme))
}

func TestCreatePersistsBinding(t *testing.T) {
	repo := newTestRepository(t)
	scenario := "  agent-inbox  "
	page := "inbox"
	doc, err := repo.Create(context.Background(), CreateInput{
		Name:     "Inbox design",
		Scenario: &scenario,
		Page:     &page,
	})
	require.NoError(t, err)
	require.NotNil(t, doc.Scenario)
	require.Equal(t, "agent-inbox", *doc.Scenario)
	require.NotNil(t, doc.Page)
	require.Equal(t, "inbox", *doc.Page)

	blank := ""
	unbound, err := repo.Create(context.Background(), CreateInput{Name: "Unbound", Scenario: &blank, Page: &blank})
	require.NoError(t, err)
	require.Nil(t, unbound.Scenario, "empty text means unbound")
	require.Nil(t, unbound.Page)
}

func TestGetMissingReturnsNotFound(t *testing.T) {
	repo := newTestRepository(t)
	_, err := repo.Get(context.Background(), "design-missing")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestSaveBumpsRevisionAndTimestamp(t *testing.T) {
	repo := newTestRepository(t)
	repo.now = func() time.Time { return time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC) }
	created, err := repo.Create(context.Background(), CreateInput{Name: "First name", Frames: json.RawMessage(`[{"id":"a"}]`)})
	require.NoError(t, err)

	repo.now = func() time.Time { return time.Date(2026, 11, 19, 12, 5, 30, 0, time.UTC) }
	revision := created.Revision
	saved, err := repo.Save(context.Background(), created.ID, UpdateInput{
		Name:     "Second name",
		Frames:   json.RawMessage(`[{"id":"a"},{"id":"b"}]`),
		Theme:    json.RawMessage(`{"accent":"#00ff00"}`),
		Revision: &revision,
	})
	require.NoError(t, err)

	require.Equal(t, 2, saved.Revision, "save must bump the stored revision")
	require.Equal(t, "2026-11-19T12:05:30.000Z", saved.UpdatedAt)
	require.Equal(t, "Second name", saved.Name)
	require.Equal(t, created.ID, saved.ID)

	reloaded, err := repo.Get(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, saved, reloaded)
}

func TestSaveWithoutRevisionIsLastWriteWins(t *testing.T) {
	repo := newTestRepository(t)
	created, err := repo.Create(context.Background(), CreateInput{Name: "First"})
	require.NoError(t, err)

	first, err := repo.Save(context.Background(), created.ID, UpdateInput{Name: "A"})
	require.NoError(t, err)
	second, err := repo.Save(context.Background(), created.ID, UpdateInput{Name: "B"})
	require.NoError(t, err)
	require.Equal(t, 3, second.Revision)
	require.Equal(t, "B", second.Name)
	require.Equal(t, first.Revision+1, second.Revision)
}

func TestSaveRejectsStaleRevision(t *testing.T) {
	repo := newTestRepository(t)
	created, err := repo.Create(context.Background(), CreateInput{Name: "First"})
	require.NoError(t, err)

	stale := created.Revision
	_, err = repo.Save(context.Background(), created.ID, UpdateInput{Name: "A", Revision: &stale})
	require.NoError(t, err)

	_, err = repo.Save(context.Background(), created.ID, UpdateInput{Name: "B", Revision: &stale})
	require.ErrorIs(t, err, ErrConflict, "the stale base revision must not overwrite")

	unchanged, err := repo.Get(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, "A", unchanged.Name, "a conflicted save must write nothing")
	require.Equal(t, 2, unchanged.Revision)
}

func TestConcurrentSavesFromSameRevisionHaveOneWinner(t *testing.T) {
	repo := newTestRepository(t)
	created, err := repo.Create(context.Background(), CreateInput{Name: "Base"})
	require.NoError(t, err)

	base := created.Revision
	results := make([]error, 2)
	var wg sync.WaitGroup
	for i := range results {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			revision := base
			_, results[index] = repo.Save(context.Background(), created.ID, UpdateInput{
				Name:     "Writer",
				Revision: &revision,
			})
		}(i)
	}
	wg.Wait()

	successes, conflicts := 0, 0
	for _, result := range results {
		switch {
		case result == nil:
			successes++
		case errors.Is(result, ErrConflict):
			conflicts++
		default:
			require.NoError(t, result)
		}
	}
	require.Equal(t, 1, successes, "exactly one concurrent save may win")
	require.Equal(t, 1, conflicts, "the losing save must be a conflict")
}

func TestListOrdersNewestFirstAndCountsFrames(t *testing.T) {
	repo := newTestRepository(t)
	repo.now = func() time.Time { return time.Date(2026, 11, 19, 9, 0, 0, 0, time.UTC) }
	older, err := repo.Create(context.Background(), CreateInput{Name: "Older", Frames: json.RawMessage(`[{"id":"a"}]`)})
	require.NoError(t, err)

	repo.now = func() time.Time { return time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC) }
	newer, err := repo.Create(context.Background(), CreateInput{Name: "Newer", Frames: json.RawMessage(`[{"id":"a"},{"id":"b"},{"id":"c"}]`)})
	require.NoError(t, err)

	list, err := repo.List(context.Background(), ListFilter{})
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, newer.ID, list[0].ID)
	require.Equal(t, older.ID, list[1].ID)
	require.Equal(t, 3, list[0].FrameCount)
	require.Equal(t, 1, list[1].FrameCount)
}

func TestListFiltersByBinding(t *testing.T) {
	repo := newTestRepository(t)
	scenario := "agent-inbox"
	inbox := "inbox"
	drafts := "drafts"
	bound, err := repo.Create(context.Background(), CreateInput{Name: "Inbox", Scenario: &scenario, Page: &inbox})
	require.NoError(t, err)
	_, err = repo.Create(context.Background(), CreateInput{Name: "Drafts", Scenario: &scenario, Page: &drafts})
	require.NoError(t, err)
	_, err = repo.Create(context.Background(), CreateInput{Name: "Loose"})
	require.NoError(t, err)

	byScenarioAndPage, err := repo.List(context.Background(), ListFilter{Scenario: scenario, Page: inbox})
	require.NoError(t, err)
	require.Len(t, byScenarioAndPage, 1)
	require.Equal(t, bound.ID, byScenarioAndPage[0].ID)

	byScenario, err := repo.List(context.Background(), ListFilter{Scenario: scenario})
	require.NoError(t, err)
	require.Len(t, byScenario, 2)

	require.NotNil(t, byScenario[0].Scenario)
	require.Equal(t, scenario, *byScenario[0].Scenario)

	all, err := repo.List(context.Background(), ListFilter{})
	require.NoError(t, err)
	require.Len(t, all, 3)
}

func TestDeleteRemovesAndIsIdempotent(t *testing.T) {
	repo := newTestRepository(t)
	doc, err := repo.Create(context.Background(), CreateInput{Name: "Disposable"})
	require.NoError(t, err)

	require.NoError(t, repo.Delete(context.Background(), doc.ID))
	_, err = repo.Get(context.Background(), doc.ID)
	require.ErrorIs(t, err, ErrNotFound)

	require.NoError(t, repo.Delete(context.Background(), doc.ID), "deleting a missing design is success")
}

func TestCreateRejectsInvalidDocuments(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	_, err := repo.Create(ctx, CreateInput{Name: "   "})
	require.ErrorIs(t, err, ErrInvalid)

	_, err = repo.Create(ctx, CreateInput{Name: "Name", Frames: json.RawMessage(`{"not":"an array"}`)})
	require.ErrorIs(t, err, ErrInvalid)

	_, err = repo.Create(ctx, CreateInput{Name: "Name", Frames: json.RawMessage(`["frame"]`)})
	require.ErrorIs(t, err, ErrInvalid)

	_, err = repo.Create(ctx, CreateInput{Name: "Name", Theme: json.RawMessage(`[]`)})
	require.ErrorIs(t, err, ErrInvalid)
}
