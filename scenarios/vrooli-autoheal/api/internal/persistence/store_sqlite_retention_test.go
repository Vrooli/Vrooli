package persistence

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

func TestSaveResultSQLiteBoundsOversizedDetails(t *testing.T) {
	db := openSQLiteTestDB(t)
	store := NewStore(db)
	if err := store.SaveResult(context.Background(), checks.Result{
		CheckID: "oversized", Status: checks.StatusCritical, Timestamp: time.Now(),
		Details: map[string]interface{}{"payload": strings.Repeat("x", maxHealthResultDetailsBytes*2)},
	}); err != nil {
		t.Fatalf("SaveResult: %v", err)
	}
	results, err := store.GetRecentResults(context.Background(), "oversized", 1)
	if err != nil {
		t.Fatalf("GetRecentResults: %v", err)
	}
	if len(results) != 1 || results[0].Details["truncated"] != true {
		t.Fatalf("bounded details = %#v", results)
	}
}
