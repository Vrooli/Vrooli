package templates

import (
	"context"
	"encoding/json"
	"strings"
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

func intPtr(value int) *int { return &value }

func TestCreateRoundTripsEveryAuthoredField(t *testing.T) {
	repo := newTestRepository(t)
	repo.now = func() time.Time { return time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC) }

	created, err := repo.Create(context.Background(), Input{
		ID:       "aperture-study-abc",
		Name:     "  Aperture study  ",
		Tagline:  "Analytics workspace",
		Category: "analytics",
		Palette:  json.RawMessage(`{"bg":"#0c0e12","accent":"#e0a45e"}`),
		Display:  "serif",
		Radius:   intPtr(16),
		Spec:     json.RawMessage(`{"id":"aperture.page","type":"page","props":{"pad":40},"children":[]}`),
		SlotHint: "Primary action in the workspace header",
		Idea:     "One number leads.",
	})
	require.NoError(t, err)

	require.Equal(t, "aperture-study-abc", created.ID)
	require.Equal(t, "Aperture study", created.Name)
	require.Equal(t, "Analytics workspace", created.Tagline)
	require.Equal(t, "analytics", created.Category)
	require.Equal(t, "serif", created.Display)
	require.Equal(t, 16, created.Radius)
	require.JSONEq(t, `{"bg":"#0c0e12","accent":"#e0a45e"}`, string(created.Palette))
	require.JSONEq(t, `{"id":"aperture.page","type":"page","props":{"pad":40},"children":[]}`, string(created.Spec))
	require.Equal(t, "Primary action in the workspace header", created.SlotHint)
	require.Equal(t, "One number leads.", created.Idea)
	require.Equal(t, "2026-11-19T12:00:00.000Z", created.CreatedAt)
	require.Equal(t, created.CreatedAt, created.UpdatedAt)

	// The copy fields travel with the spec document; a reload must return them
	// byte-identical rather than dropping them at the column boundary.
	reloaded, err := repo.Get(context.Background(), created.ID)
	require.NoError(t, err)
	require.Equal(t, created, reloaded)
}

func TestCreateAppliesDocumentedDefaults(t *testing.T) {
	repo := newTestRepository(t)
	created, err := repo.Create(context.Background(), Input{ID: "untitled-abc", Name: "Blank"})
	require.NoError(t, err)

	require.Equal(t, "", created.Tagline)
	require.Equal(t, "", created.Category)
	require.Equal(t, "sans", created.Display)
	require.Equal(t, 14, created.Radius)
	require.JSONEq(t, `{}`, string(created.Palette))
	require.JSONEq(t, `{}`, string(created.Spec))
	require.Equal(t, "", created.SlotHint)
	require.Equal(t, "", created.Idea)
}

func TestCreateRejectsDuplicateID(t *testing.T) {
	repo := newTestRepository(t)
	_, err := repo.Create(context.Background(), Input{ID: "same-id", Name: "First"})
	require.NoError(t, err)

	_, err = repo.Create(context.Background(), Input{ID: "same-id", Name: "Second"})
	require.ErrorIs(t, err, ErrConflict)
}

func TestUpdateReplacesAndBumpsTimestamp(t *testing.T) {
	repo := newTestRepository(t)
	repo.now = func() time.Time { return time.Date(2026, 11, 19, 9, 0, 0, 0, time.UTC) }
	created, err := repo.Create(context.Background(), Input{
		ID:       "shelf-one",
		Name:     "First",
		Spec:     json.RawMessage(`{"id":"one","type":"page"}`),
		SlotHint: "Before",
	})
	require.NoError(t, err)

	repo.now = func() time.Time { return time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC) }
	updated, err := repo.Update(context.Background(), "shelf-one", Input{
		Name:    "Second",
		Display: "serif",
		Radius:  intPtr(0),
		Spec:    json.RawMessage(`{"id":"two","type":"page"}`),
	})
	require.NoError(t, err)

	require.Equal(t, "shelf-one", updated.ID)
	require.Equal(t, "Second", updated.Name)
	require.Equal(t, "serif", updated.Display)
	require.Equal(t, 0, updated.Radius)
	require.JSONEq(t, `{"id":"two","type":"page"}`, string(updated.Spec))
	require.Empty(t, updated.SlotHint, "replace semantics clear omitted copy")
	require.Equal(t, "2026-11-19T10:00:00.000Z", updated.UpdatedAt)
	require.Equal(t, created.CreatedAt, updated.CreatedAt, "createdAt is server-owned and stable")
}

func TestUpdateMissingReturnsNotFound(t *testing.T) {
	repo := newTestRepository(t)
	_, err := repo.Update(context.Background(), "missing-id", Input{Name: "Nope"})
	require.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteRemovesAndIsIdempotent(t *testing.T) {
	repo := newTestRepository(t)
	_, err := repo.Create(context.Background(), Input{ID: "disposable", Name: "Disposable"})
	require.NoError(t, err)

	require.NoError(t, repo.Delete(context.Background(), "disposable"))
	_, err = repo.Get(context.Background(), "disposable")
	require.ErrorIs(t, err, ErrNotFound)

	require.NoError(t, repo.Delete(context.Background(), "disposable"), "deleting a missing template is success")
}

func TestListOrdersNewestUpdatedFirst(t *testing.T) {
	repo := newTestRepository(t)
	repo.now = func() time.Time { return time.Date(2026, 11, 19, 9, 0, 0, 0, time.UTC) }
	_, err := repo.Create(context.Background(), Input{ID: "older", Name: "Older"})
	require.NoError(t, err)

	repo.now = func() time.Time { return time.Date(2026, 11, 19, 10, 0, 0, 0, time.UTC) }
	_, err = repo.Create(context.Background(), Input{ID: "newer", Name: "Newer"})
	require.NoError(t, err)

	list, err := repo.List(context.Background())
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, "newer", list[0].ID)
	require.Equal(t, "older", list[1].ID)

	// Replacing the older row moves it to the front: updated_at, not insertion
	// order, is what the shelf reads.
	repo.now = func() time.Time { return time.Date(2026, 11, 19, 11, 0, 0, 0, time.UTC) }
	_, err = repo.Update(context.Background(), "older", Input{Name: "Older again"})
	require.NoError(t, err)

	list, err = repo.List(context.Background())
	require.NoError(t, err)
	require.Equal(t, "older", list[0].ID)
	require.Equal(t, "newer", list[1].ID)
}

func TestCreateRejectsInvalidFields(t *testing.T) {
	repo := newTestRepository(t)
	ctx := context.Background()

	tooBigSpec := json.RawMessage(`{"data":"` + strings.Repeat("a", MaxSpecBytes) + `"}`)
	tooBigPalette := json.RawMessage(`{"data":"` + strings.Repeat("a", MaxPaletteBytes) + `"}`)

	cases := map[string]struct {
		input Input
		field string
	}{
		"missing id":         {Input{Name: "x"}, "id is required"},
		"bad id":             {Input{ID: "Not A Slug", Name: "x"}, "id must match"},
		"missing name":       {Input{ID: "ok-id", Name: "   "}, "name is required"},
		"long name":          {Input{ID: "ok-id", Name: strings.Repeat("n", MaxNameLength+1)}, "name must be at most"},
		"long tagline":       {Input{ID: "ok-id", Name: "x", Tagline: strings.Repeat("t", MaxTaglineLength+1)}, "tagline must be at most"},
		"long category":      {Input{ID: "ok-id", Name: "x", Category: strings.Repeat("c", MaxCategoryLength+1)}, "category must be at most"},
		"bad display":        {Input{ID: "ok-id", Name: "x", Display: "mono"}, "display must be"},
		"negative radius":    {Input{ID: "ok-id", Name: "x", Radius: intPtr(-1)}, "radius must be between"},
		"radius too large":   {Input{ID: "ok-id", Name: "x", Radius: intPtr(MaxRadius + 1)}, "radius must be between"},
		"palette not object": {Input{ID: "ok-id", Name: "x", Palette: json.RawMessage(`[]`)}, "palette must be a JSON object"},
		"palette invalid":    {Input{ID: "ok-id", Name: "x", Palette: json.RawMessage(`{"a":`)}, "palette must be a JSON object"},
		"palette too big":    {Input{ID: "ok-id", Name: "x", Palette: tooBigPalette}, "palette must be at most"},
		"spec not object":    {Input{ID: "ok-id", Name: "x", Spec: json.RawMessage(`"page"`)}, "spec must be a JSON object"},
		"spec too big":       {Input{ID: "ok-id", Name: "x", Spec: tooBigSpec}, "spec must be at most"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := repo.Create(ctx, tc.input)
			require.ErrorIs(t, err, ErrInvalid)
			require.Contains(t, err.Error(), tc.field)
		})
	}
}

func TestUpdateValidatesThePathID(t *testing.T) {
	repo := newTestRepository(t)
	_, err := repo.Update(context.Background(), "Bad ID", Input{Name: "x"})
	require.ErrorIs(t, err, ErrInvalid)
	require.Contains(t, err.Error(), "id must match")
}

func TestJSONObjectsAreStoredCompactly(t *testing.T) {
	repo := newTestRepository(t)
	created, err := repo.Create(context.Background(), Input{
		ID:      "compact",
		Name:    "Compact",
		Palette: json.RawMessage("{\n  \"bg\": \"#000\"\n}"),
		Spec:    json.RawMessage("{\n  \"id\": \"a\",\n  \"type\": \"page\"\n}"),
	})
	require.NoError(t, err)
	require.Equal(t, `{"bg":"#000"}`, string(created.Palette))
	require.Equal(t, `{"id":"a","type":"page"}`, string(created.Spec))
}

func TestDecodeSpecDocumentReadsBareLegacySpec(t *testing.T) {
	spec, slotHint, idea := decodeSpecDocument(`{"id":"legacy","type":"page","props":{}}`)
	require.JSONEq(t, `{"id":"legacy","type":"page","props":{}}`, string(spec))
	require.Empty(t, slotHint)
	require.Empty(t, idea)
}
