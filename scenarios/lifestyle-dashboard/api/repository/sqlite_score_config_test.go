package repository

import (
	"context"
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"lifestyle-dashboard/domain"
)

// setupScoreConfigTestDB creates an in-memory SQLite database with schema.
func setupScoreConfigTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := domain.InitSchema(db); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}
	return db
}

// [REQ:LD-SCORE-CALC] Tests for configurable domain weights.
func TestScoreConfigRepository_GetWeights_Empty(t *testing.T) {
	db := setupScoreConfigTestDB(t)
	defer db.Close()

	repo := NewSQLiteScoreConfigRepository(db)
	weights, err := repo.GetWeights(context.Background())
	if err != nil {
		t.Fatalf("GetWeights failed: %v", err)
	}

	// With no domains, should return empty list
	if len(weights) != 0 {
		t.Errorf("Expected 0 weights for empty database, got %d", len(weights))
	}
}

// [REQ:LD-SCORE-CALC] Tests preset weight application.
func TestScoreConfigRepository_GetWeights_WithPresets(t *testing.T) {
	db := setupScoreConfigTestDB(t)
	defer db.Close()

	// Register a domain that has a preset (sleep = high)
	domainRepo := NewSQLiteDomainRepository(db)
	domainRepo.Upsert(context.Background(), &domain.Domain{
		Name:        "sleep",
		DisplayName: "Sleep Tracker",
	})

	repo := NewSQLiteScoreConfigRepository(db)
	weights, err := repo.GetWeights(context.Background())
	if err != nil {
		t.Fatalf("GetWeights failed: %v", err)
	}

	if len(weights) != 1 {
		t.Fatalf("Expected 1 weight, got %d", len(weights))
	}

	// Sleep should have preset weight of "high"
	if weights[0].Weight != "high" {
		t.Errorf("Expected 'sleep' domain to have preset weight 'high', got '%s'", weights[0].Weight)
	}
	if weights[0].Multiplier != 3.0 {
		t.Errorf("Expected multiplier 3.0 for 'high', got %f", weights[0].Multiplier)
	}
}

// [REQ:LD-SCORE-CALC] Tests default weight for unknown domains.
func TestScoreConfigRepository_GetWeights_DefaultMedium(t *testing.T) {
	db := setupScoreConfigTestDB(t)
	defer db.Close()

	// Register a domain without a preset
	domainRepo := NewSQLiteDomainRepository(db)
	domainRepo.Upsert(context.Background(), &domain.Domain{
		Name:        "custom-domain",
		DisplayName: "Custom Domain",
	})

	repo := NewSQLiteScoreConfigRepository(db)
	weights, err := repo.GetWeights(context.Background())
	if err != nil {
		t.Fatalf("GetWeights failed: %v", err)
	}

	if len(weights) != 1 {
		t.Fatalf("Expected 1 weight, got %d", len(weights))
	}

	// Unknown domains default to medium
	if weights[0].Weight != "medium" {
		t.Errorf("Expected default weight 'medium', got '%s'", weights[0].Weight)
	}
	if weights[0].Multiplier != 2.0 {
		t.Errorf("Expected multiplier 2.0 for 'medium', got %f", weights[0].Multiplier)
	}
}

// [REQ:LD-SCORE-CALC] Tests getting single domain weight.
func TestScoreConfigRepository_GetWeight(t *testing.T) {
	db := setupScoreConfigTestDB(t)
	defer db.Close()

	domainRepo := NewSQLiteDomainRepository(db)
	domainRepo.Upsert(context.Background(), &domain.Domain{
		Name:        "exercise",
		DisplayName: "Exercise Tracker",
	})

	repo := NewSQLiteScoreConfigRepository(db)
	weight, err := repo.GetWeight(context.Background(), "exercise")
	if err != nil {
		t.Fatalf("GetWeight failed: %v", err)
	}

	// Exercise has preset "high"
	if weight.Domain != "exercise" {
		t.Errorf("Expected domain 'exercise', got '%s'", weight.Domain)
	}
	if weight.Weight != "high" {
		t.Errorf("Expected weight 'high' from preset, got '%s'", weight.Weight)
	}
}

// [REQ:LD-SCORE-CALC] Tests weight not found error.
func TestScoreConfigRepository_GetWeight_NotFound(t *testing.T) {
	db := setupScoreConfigTestDB(t)
	defer db.Close()

	repo := NewSQLiteScoreConfigRepository(db)
	_, err := repo.GetWeight(context.Background(), "nonexistent")

	if !IsNotFound(err) {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

// [REQ:LD-SCORE-CALC] Tests setting domain weight.
func TestScoreConfigRepository_SetWeight(t *testing.T) {
	db := setupScoreConfigTestDB(t)
	defer db.Close()

	domainRepo := NewSQLiteDomainRepository(db)
	domainRepo.Upsert(context.Background(), &domain.Domain{
		Name:        "sleep",
		DisplayName: "Sleep Tracker",
	})

	repo := NewSQLiteScoreConfigRepository(db)

	// Set to low (overriding the preset)
	err := repo.SetWeight(context.Background(), "sleep", "low")
	if err != nil {
		t.Fatalf("SetWeight failed: %v", err)
	}

	// Verify the override
	weight, err := repo.GetWeight(context.Background(), "sleep")
	if err != nil {
		t.Fatalf("GetWeight failed: %v", err)
	}

	if weight.Weight != "low" {
		t.Errorf("Expected weight 'low', got '%s'", weight.Weight)
	}
	if weight.Multiplier != 1.0 {
		t.Errorf("Expected multiplier 1.0, got %f", weight.Multiplier)
	}
}

// [REQ:LD-SCORE-CALC] Tests setting weight to none (disabled).
func TestScoreConfigRepository_SetWeight_None(t *testing.T) {
	db := setupScoreConfigTestDB(t)
	defer db.Close()

	domainRepo := NewSQLiteDomainRepository(db)
	domainRepo.Upsert(context.Background(), &domain.Domain{
		Name:        "test-domain",
		DisplayName: "Test Domain",
	})

	repo := NewSQLiteScoreConfigRepository(db)
	err := repo.SetWeight(context.Background(), "test-domain", "none")
	if err != nil {
		t.Fatalf("SetWeight failed: %v", err)
	}

	weight, err := repo.GetWeight(context.Background(), "test-domain")
	if err != nil {
		t.Fatalf("GetWeight failed: %v", err)
	}

	if weight.Multiplier != 0.0 {
		t.Errorf("Expected multiplier 0.0 for 'none', got %f", weight.Multiplier)
	}
}

// [REQ:LD-SCORE-CALC] Tests setting weight for nonexistent domain.
func TestScoreConfigRepository_SetWeight_NotFound(t *testing.T) {
	db := setupScoreConfigTestDB(t)
	defer db.Close()

	repo := NewSQLiteScoreConfigRepository(db)
	err := repo.SetWeight(context.Background(), "nonexistent", "high")

	if !IsNotFound(err) {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

// [REQ:LD-SCORE-CALC] Tests setting invalid weight value.
func TestScoreConfigRepository_SetWeight_InvalidWeight(t *testing.T) {
	db := setupScoreConfigTestDB(t)
	defer db.Close()

	domainRepo := NewSQLiteDomainRepository(db)
	domainRepo.Upsert(context.Background(), &domain.Domain{
		Name:        "test-domain",
		DisplayName: "Test Domain",
	})

	repo := NewSQLiteScoreConfigRepository(db)
	err := repo.SetWeight(context.Background(), "test-domain", "invalid")

	if !IsNotFound(err) {
		t.Errorf("Expected ErrNotFound for invalid weight, got %v", err)
	}
}

// [REQ:LD-SCORE-CALC] Tests weight upsert (update existing).
func TestScoreConfigRepository_SetWeight_Upsert(t *testing.T) {
	db := setupScoreConfigTestDB(t)
	defer db.Close()

	domainRepo := NewSQLiteDomainRepository(db)
	domainRepo.Upsert(context.Background(), &domain.Domain{
		Name:        "test-domain",
		DisplayName: "Test Domain",
	})

	repo := NewSQLiteScoreConfigRepository(db)

	// Set initial weight
	err := repo.SetWeight(context.Background(), "test-domain", "low")
	if err != nil {
		t.Fatalf("SetWeight (first) failed: %v", err)
	}

	// Update to different weight
	err = repo.SetWeight(context.Background(), "test-domain", "high")
	if err != nil {
		t.Fatalf("SetWeight (update) failed: %v", err)
	}

	weight, err := repo.GetWeight(context.Background(), "test-domain")
	if err != nil {
		t.Fatalf("GetWeight failed: %v", err)
	}

	if weight.Weight != "high" {
		t.Errorf("Expected updated weight 'high', got '%s'", weight.Weight)
	}
}
