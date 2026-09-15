package testing

import (
	"context"
	"testing"
	"time"

	"prompt-manager/internal/testsqlite"

	"github.com/vrooli/api-core/database"
)

func TestRepositorySavesHistoryForNonUUIDSkillID(t *testing.T) {
	db := testsqlite.Open(t)
	if err := database.EnsureSchemas(context.Background(), db.Primary(), database.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("apply testing schema: %v", err)
	}
	repo := NewRepository(db.Primary())

	vars := `{"topic":"sqlite"}`
	response := "Use embedded SQLite."
	elapsed := 42.5
	tokens := 17
	testedAt := time.Date(2026, 6, 17, 12, 0, 0, 123, time.UTC)
	result := &TestResult{
		SkillID:      "skill.core/storage-steer",
		Role:         "chat.small",
		InputVars:    &vars,
		Response:     &response,
		ResponseTime: &elapsed,
		TokenCount:   &tokens,
		TestedAt:     testedAt,
	}
	if err := repo.Save(result); err != nil {
		t.Fatalf("save test result: %v", err)
	}

	history, err := repo.GetHistory("skill.core/storage-steer", 10)
	if err != nil {
		t.Fatalf("get history: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history row, got %d: %+v", len(history), history)
	}
	got := history[0]
	if got.ID == "" || got.SkillID != result.SkillID || got.Role != result.Role {
		t.Fatalf("unexpected identity: %+v", got)
	}
	if got.InputVars == nil || *got.InputVars != vars {
		t.Fatalf("expected JSON text input variables, got %+v", got.InputVars)
	}
	if got.Response == nil || *got.Response != response || got.ResponseTime == nil || *got.ResponseTime != elapsed || got.TokenCount == nil || *got.TokenCount != tokens {
		t.Fatalf("unexpected response fields: %+v", got)
	}
	if !got.TestedAt.Equal(testedAt) {
		t.Fatalf("expected tested_at %s, got %s", testedAt, got.TestedAt)
	}
}
