// [REQ:LD-DIGEST-WEEKLY] Tests for weekly digest repository.
package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"lifestyle-dashboard/domain"
)

// setupDigestTestDB creates a test database with schema.
func setupDigestTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test db: %v", err)
	}
	db.SetMaxOpenConns(1)

	if err := domain.InitSchema(db); err != nil {
		t.Fatalf("Failed to init schema: %v", err)
	}

	return db
}

// TestGenerateWeeklyDigest_ReturnsDigest tests basic digest generation.
// [REQ:LD-DIGEST-WEEKLY]
func TestGenerateWeeklyDigest_ReturnsDigest(t *testing.T) {
	db := setupDigestTestDB(t)
	defer db.Close()

	repo := NewSQLiteDigestRepository(db)
	ctx := context.Background()

	// Use current week's Monday
	now := time.Now().UTC()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -(weekday - 1)).Format("2006-01-02")

	digest, err := repo.GenerateWeeklyDigest(ctx, monday)
	if err != nil {
		t.Fatalf("Failed to generate digest: %v", err)
	}

	if digest == nil {
		t.Fatal("Expected digest, got nil")
	}

	if digest.WeekStartDate != monday {
		t.Errorf("Expected week start %s, got %s", monday, digest.WeekStartDate)
	}

	if digest.GeneratedAt == "" {
		t.Error("Expected non-empty GeneratedAt")
	}
}

// TestGenerateWeeklyDigest_InvalidDate_UsesCurrent tests fallback for invalid dates.
// [REQ:LD-DIGEST-WEEKLY]
func TestGenerateWeeklyDigest_InvalidDate_UsesCurrent(t *testing.T) {
	db := setupDigestTestDB(t)
	defer db.Close()

	repo := NewSQLiteDigestRepository(db)
	ctx := context.Background()

	// Pass invalid date
	digest, err := repo.GenerateWeeklyDigest(ctx, "not-a-date")
	if err != nil {
		t.Fatalf("Failed to generate digest: %v", err)
	}

	if digest == nil {
		t.Fatal("Expected digest, got nil")
	}

	// Should have fallen back to a valid date
	if digest.WeekStartDate == "not-a-date" {
		t.Error("Expected date to be corrected from invalid value")
	}
}

// TestGetLatestDigest_ReturnsDigest tests latest digest retrieval.
// [REQ:LD-DIGEST-WEEKLY]
func TestGetLatestDigest_ReturnsDigest(t *testing.T) {
	db := setupDigestTestDB(t)
	defer db.Close()

	repo := NewSQLiteDigestRepository(db)
	ctx := context.Background()

	digest, err := repo.GetLatestDigest(ctx)
	if err != nil {
		t.Fatalf("Failed to get latest digest: %v", err)
	}

	if digest == nil {
		t.Fatal("Expected digest, got nil")
	}
}

// TestDigest_WithDomainActivity tests digest with actual domain data.
// [REQ:LD-DIGEST-WEEKLY]
func TestDigest_WithDomainActivity(t *testing.T) {
	db := setupDigestTestDB(t)
	defer db.Close()

	// Insert domain and events
	domainRepo := NewSQLiteDomainRepository(db)
	eventRepo := NewSQLiteEventRepository(db)
	ctx := context.Background()

	// Register domain
	d := &domain.Domain{
		Name:        "sleep",
		DisplayName: "Sleep Tracker",
		Status:      "active",
	}
	if err := domainRepo.Upsert(ctx, d); err != nil {
		t.Fatalf("Failed to create domain: %v", err)
	}

	// Create event for today
	e := &domain.Event{
		Domain:    "sleep",
		EventType: "bedtime",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Payload:   []byte(`{}`),
	}
	if err := eventRepo.Create(ctx, e); err != nil {
		t.Fatalf("Failed to create event: %v", err)
	}

	repo := NewSQLiteDigestRepository(db)

	now := time.Now().UTC()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -(weekday - 1)).Format("2006-01-02")

	digest, err := repo.GenerateWeeklyDigest(ctx, monday)
	if err != nil {
		t.Fatalf("Failed to generate digest: %v", err)
	}

	// Should have domain changes
	if len(digest.DomainChanges) == 0 {
		t.Error("Expected domain changes when domains exist")
	}

	// Check that sleep domain is included
	foundSleep := false
	for _, change := range digest.DomainChanges {
		if change.Domain == "sleep" {
			foundSleep = true
			if change.CurrentWeekEvents == 0 {
				t.Error("Expected non-zero events for sleep domain")
			}
			break
		}
	}
	if !foundSleep {
		t.Error("Expected sleep domain in changes")
	}
}

// TestDigest_ScoreTrend tests score trend calculation.
// [REQ:LD-DIGEST-WEEKLY]
func TestDigest_ScoreTrend(t *testing.T) {
	db := setupDigestTestDB(t)
	defer db.Close()

	repo := NewSQLiteDigestRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -(weekday - 1)).Format("2006-01-02")

	digest, err := repo.GenerateWeeklyDigest(ctx, monday)
	if err != nil {
		t.Fatalf("Failed to generate digest: %v", err)
	}

	// Score trend direction must be valid
	validDirections := map[string]bool{"up": true, "down": true, "stable": true}
	if !validDirections[digest.ScoreTrend.Direction] {
		t.Errorf("Invalid direction '%s'", digest.ScoreTrend.Direction)
	}

	// Message should be populated
	if digest.ScoreTrend.Message == "" {
		t.Error("Expected non-empty score trend message")
	}
}

// TestDigest_Highlights tests highlight generation.
// [REQ:LD-DIGEST-WEEKLY]
func TestDigest_Highlights(t *testing.T) {
	db := setupDigestTestDB(t)
	defer db.Close()

	repo := NewSQLiteDigestRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -(weekday - 1)).Format("2006-01-02")

	digest, err := repo.GenerateWeeklyDigest(ctx, monday)
	if err != nil {
		t.Fatalf("Failed to generate digest: %v", err)
	}

	// Should always have at least one highlight
	if len(digest.Highlights) == 0 {
		t.Error("Expected at least one highlight")
	}
}

// TestDigest_NextWeekFocus tests focus recommendations.
// [REQ:LD-DIGEST-WEEKLY]
func TestDigest_NextWeekFocus(t *testing.T) {
	db := setupDigestTestDB(t)
	defer db.Close()

	repo := NewSQLiteDigestRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -(weekday - 1)).Format("2006-01-02")

	digest, err := repo.GenerateWeeklyDigest(ctx, monday)
	if err != nil {
		t.Fatalf("Failed to generate digest: %v", err)
	}

	// Should always have at least one focus item
	if len(digest.NextWeekFocus) == 0 {
		t.Error("Expected at least one next week focus")
	}
}

// TestDigest_Summary tests summary generation.
// [REQ:LD-DIGEST-WEEKLY]
func TestDigest_Summary(t *testing.T) {
	db := setupDigestTestDB(t)
	defer db.Close()

	repo := NewSQLiteDigestRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -(weekday - 1)).Format("2006-01-02")

	digest, err := repo.GenerateWeeklyDigest(ctx, monday)
	if err != nil {
		t.Fatalf("Failed to generate digest: %v", err)
	}

	// Summary should always be populated
	if digest.Summary == "" {
		t.Error("Expected non-empty summary")
	}
}

// TestGetLastMonday tests the helper function.
func TestGetLastMonday(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected time.Weekday
	}{
		{
			name:     "from Monday",
			input:    time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC), // Monday
			expected: time.Monday,
		},
		{
			name:     "from Wednesday",
			input:    time.Date(2026, 3, 11, 12, 0, 0, 0, time.UTC), // Wednesday
			expected: time.Monday,
		},
		{
			name:     "from Sunday",
			input:    time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC), // Sunday
			expected: time.Monday,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getLastMonday(tt.input)
			if result.Weekday() != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result.Weekday())
			}
		})
	}
}
