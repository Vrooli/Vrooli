package search_test

import (
	"context"
	"math"
	"testing"

	apidb "github.com/vrooli/api-core/database"

	"knowledge-observatory/internal/dbtest"
	"knowledge-observatory/internal/search"
)

func newRepo(t *testing.T) *search.SQLite {
	t.Helper()
	return search.NewSQLite(dbtest.New(t, apidb.SchemaProviderFunc(search.Schema)))
}

func ptr(v float64) *float64 { return &v }

func TestHistoryRoundTripCoversEveryColumn(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	in := search.History{
		Query:          "how does autoheal detect rdp",
		Collection:     "vrooli_knowledge",
		ResultCount:    7,
		AvgScore:       ptr(0.61),
		ResponseTimeMS: 142,
		UserSession:    "session-abc",
	}
	id, err := repo.InsertHistory(ctx, in)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	rows, err := repo.RecentHistory(ctx, 10)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	got := rows[0]

	if got.ID != id {
		t.Errorf("id = %q, want %q", got.ID, id)
	}
	if got.Query != in.Query {
		t.Errorf("query = %q, want %q", got.Query, in.Query)
	}
	if got.Collection != in.Collection {
		t.Errorf("collection = %q, want %q", got.Collection, in.Collection)
	}
	if got.ResultCount != in.ResultCount {
		t.Errorf("result_count = %d, want %d", got.ResultCount, in.ResultCount)
	}
	if got.AvgScore == nil || *got.AvgScore != 0.61 {
		t.Errorf("avg_score = %v, want 0.61", got.AvgScore)
	}
	if got.ResponseTimeMS != in.ResponseTimeMS {
		t.Errorf("response_time_ms = %d, want %d", got.ResponseTimeMS, in.ResponseTimeMS)
	}
	if got.UserSession != in.UserSession {
		t.Errorf("user_session = %q, want %q", got.UserSession, in.UserSession)
	}
	if got.CreatedAt.IsZero() {
		t.Error("created_at was not defaulted")
	}
}

// TestEmptyOptionalsBecomeNull proves NULLIF(?, ”) survived the dialect change:
// an absent collection or session must read back as empty, not as the literal
// empty string masquerading as a value.
func TestEmptyOptionalsBecomeNull(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	if _, err := repo.InsertHistory(ctx, search.History{Query: "bare"}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	rows, err := repo.RecentHistory(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].Collection != "" || rows[0].UserSession != "" {
		t.Errorf("optional columns = %q/%q, want empty", rows[0].Collection, rows[0].UserSession)
	}
	if rows[0].AvgScore != nil {
		t.Errorf("avg_score = %v, want nil", rows[0].AvgScore)
	}
}

// TestNonFiniteScoreIsDropped guards the REAL column: NaN and Inf have no
// meaningful SQLite representation and must not be stored.
func TestNonFiniteScoreIsDropped(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	nan := math.NaN()
	if _, err := repo.InsertHistory(ctx, search.History{Query: "nan", AvgScore: &nan}); err != nil {
		t.Fatalf("insert: %v", err)
	}
	rows, err := repo.RecentHistory(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].AvgScore != nil {
		t.Errorf("avg_score = %v, want nil for a non-finite input", *rows[0].AvgScore)
	}
}

func TestEmptyQueryIsRejected(t *testing.T) {
	repo := newRepo(t)
	if _, err := repo.InsertHistory(context.Background(), search.History{Query: "   "}); err == nil {
		t.Fatal("expected an empty query to be rejected")
	}
}
