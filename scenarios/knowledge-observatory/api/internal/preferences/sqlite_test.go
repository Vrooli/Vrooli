package preferences_test

import (
	"context"
	"testing"

	apidb "github.com/vrooli/api-core/database"

	"knowledge-observatory/internal/dbtest"
	"knowledge-observatory/internal/preferences"
)

func newRepo(t *testing.T) *preferences.SQLite {
	t.Helper()
	return preferences.NewSQLite(dbtest.New(t, apidb.SchemaProviderFunc(preferences.Schema)))
}

func TestPreferenceRoundTripCoversEveryColumn(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	in := preferences.Preference{
		UserID:            "matt",
		DefaultCollection: "vrooli_knowledge",
		SavedQueries:      `["rdp","keyring"]`,
		DashboardLayout:   `{"columns":3}`,
		AlertPreferences:  `{"email":false}`,
	}
	if err := repo.Upsert(ctx, in); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, ok, err := repo.Get(ctx, "matt")
	if err != nil || !ok {
		t.Fatalf("get: ok=%v err=%v", ok, err)
	}
	if got.ID == "" {
		t.Error("id was not generated")
	}
	if got.UserID != in.UserID || got.DefaultCollection != in.DefaultCollection {
		t.Errorf("identity = %q/%q", got.UserID, got.DefaultCollection)
	}
	// The three JSON documents were JSONB on Postgres and are TEXT here; they
	// must survive byte for byte.
	if got.SavedQueries != in.SavedQueries {
		t.Errorf("saved_queries = %q, want %q", got.SavedQueries, in.SavedQueries)
	}
	if got.DashboardLayout != in.DashboardLayout {
		t.Errorf("dashboard_layout = %q, want %q", got.DashboardLayout, in.DashboardLayout)
	}
	if got.AlertPreferences != in.AlertPreferences {
		t.Errorf("alert_preferences = %q, want %q", got.AlertPreferences, in.AlertPreferences)
	}
	if got.CreatedAt.IsZero() || got.UpdatedAt.IsZero() {
		t.Error("timestamps were not defaulted")
	}
}

// TestUpsertPreservesUnsuppliedDocuments checks the COALESCE arms: saving only
// a layout must not wipe saved queries.
func TestUpsertPreservesUnsuppliedDocuments(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	if err := repo.Upsert(ctx, preferences.Preference{
		UserID: "matt", SavedQueries: `["a"]`, DashboardLayout: `{"columns":1}`,
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Upsert(ctx, preferences.Preference{
		UserID: "matt", DashboardLayout: `{"columns":9}`,
	}); err != nil {
		t.Fatal(err)
	}

	got, _, err := repo.Get(ctx, "matt")
	if err != nil {
		t.Fatal(err)
	}
	if got.DashboardLayout != `{"columns":9}` {
		t.Errorf("dashboard_layout = %q, want the update", got.DashboardLayout)
	}
	if got.SavedQueries != `["a"]` {
		t.Errorf("saved_queries = %q, want it preserved", got.SavedQueries)
	}
}

func TestMissingUserIDIsRejected(t *testing.T) {
	repo := newRepo(t)
	if err := repo.Upsert(context.Background(), preferences.Preference{}); err == nil {
		t.Fatal("expected a missing user_id to be rejected")
	}
}
