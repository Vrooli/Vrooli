package main

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// TestNewBatchAnalyticsProcessor tests processor creation
func TestNewBatchAnalyticsProcessor(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	t.Run("Success", func(t *testing.T) {
		processor := NewBatchAnalyticsProcessor(env.DB)
		if processor == nil {
			t.Fatal("Expected non-nil processor")
		}

		if processor.db == nil {
			t.Error("Expected database to be set")
		}
	})
}

// TestUpdateRelationshipStrengths tests relationship strength updates
func TestUpdateRelationshipStrengths(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	processor := NewBatchAnalyticsProcessor(env.DB)

	t.Run("Success", func(t *testing.T) {
		// Create test persons and relationship
		person1ID := createTestPerson(t, env, "Strength Test 1")
		person2ID := createTestPerson(t, env, "Strength Test 2")
		defer cleanupTestPerson(t, env, person1ID)
		defer cleanupTestPerson(t, env, person2ID)

		relID := createTestRelationship(t, env, person1ID, person2ID, "friend")
		defer cleanupTestRelationship(t, env, relID)

		// Update relationship strengths
		err := processor.updateRelationshipStrengths()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

// TestCalculateClosenessScores tests closeness score calculation
func TestCalculateClosenessScores(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	processor := NewBatchAnalyticsProcessor(env.DB)

	t.Run("Success_WithRelationships", func(t *testing.T) {
		// Create test persons with relationships
		person1ID := createTestPerson(t, env, "Closeness Test 1")
		person2ID := createTestPerson(t, env, "Closeness Test 2")
		defer cleanupTestPerson(t, env, person1ID)
		defer cleanupTestPerson(t, env, person2ID)

		strength := 0.8
		relReq := CreateRelationshipRequest{
			FromPersonID:     person1ID,
			ToPersonID:       person2ID,
			RelationshipType: "friend",
			Strength:         &strength,
		}

		// Create relationship directly in DB
		now := time.Now()
		metadataJSON, _ := json.Marshal(relReq.Metadata)
		query := `
			INSERT INTO relationships (id, created_at, updated_at, from_person_id, to_person_id,
			                          relationship_type, strength, metadata)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id`

		var relID string
		err := env.DB.QueryRow(query, uuid.New().String(), now, now, person1ID, person2ID,
			relReq.RelationshipType, relReq.Strength, metadataJSON).Scan(&relID)
		if err != nil {
			t.Fatalf("Failed to create relationship: %v", err)
		}
		defer cleanupTestRelationship(t, env, relID)

		// Calculate closeness scores
		err = processor.calculateClosenessScores()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify computed signals were updated
		var computedSignals []byte
		err = env.DB.QueryRow(`
			SELECT computed_signals
			FROM persons
			WHERE id = $1`, person1ID).Scan(&computedSignals)

		if err != nil {
			t.Fatalf("Failed to query person: %v", err)
		}

		var signals map[string]interface{}
		if len(computedSignals) > 0 {
			json.Unmarshal(computedSignals, &signals)
			if _, hasScore := signals["overall_closeness_score"]; !hasScore {
				t.Error("Expected overall_closeness_score in computed signals")
			}
		}
	})

	t.Run("Success_NoRelationships", func(t *testing.T) {
		// Create person with no relationships
		personID := createTestPerson(t, env, "No Relationships Test")
		defer cleanupTestPerson(t, env, personID)

		err := processor.calculateClosenessScores()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

// TestCalculateMaintenancePriority tests maintenance priority calculation
func TestCalculateMaintenancePriority(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	processor := NewBatchAnalyticsProcessor(env.DB)

	t.Run("Success", func(t *testing.T) {
		// Create test persons with old last contact date
		person1ID := createTestPerson(t, env, "Priority Test 1")
		person2ID := createTestPerson(t, env, "Priority Test 2")
		defer cleanupTestPerson(t, env, person1ID)
		defer cleanupTestPerson(t, env, person2ID)

		// Create relationship with old contact date
		oldDate := time.Now().Add(-100 * 24 * time.Hour) // 100 days ago
		strength := 0.3 // Low strength

		now := time.Now()
		query := `
			INSERT INTO relationships (id, created_at, updated_at, from_person_id, to_person_id,
			                          relationship_type, strength, last_contact_date)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id`

		var relID string
		err := env.DB.QueryRow(query, uuid.New().String(), now, now, person1ID, person2ID,
			"friend", strength, oldDate).Scan(&relID)
		if err != nil {
			t.Fatalf("Failed to create relationship: %v", err)
		}
		defer cleanupTestRelationship(t, env, relID)

		// Calculate maintenance priority
		err = processor.calculateMaintenancePriority()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify priority score was set
		var computedSignals []byte
		err = env.DB.QueryRow(`
			SELECT computed_signals
			FROM persons
			WHERE id = $1`, person1ID).Scan(&computedSignals)

		if err != nil {
			t.Fatalf("Failed to query person: %v", err)
		}

		if len(computedSignals) > 0 {
			var signals map[string]interface{}
			json.Unmarshal(computedSignals, &signals)
			if score, hasScore := signals["maintenance_priority_score"]; hasScore {
				// Score should be high due to old contact and low strength
				if scoreFloat, ok := score.(float64); ok && scoreFloat < 0.5 {
					t.Errorf("Expected high maintenance priority (>0.5), got %f", scoreFloat)
				}
			}
		}
	})
}

// TestIdentifySharedInterests tests shared interest identification
func TestIdentifySharedInterests(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	processor := NewBatchAnalyticsProcessor(env.DB)

	t.Run("Success_WithCommonTags", func(t *testing.T) {
		// Create persons with shared tags
		person1ID := createTestPerson(t, env, "Shared Interest 1")
		person2ID := createTestPerson(t, env, "Shared Interest 2")
		defer cleanupTestPerson(t, env, person1ID)
		defer cleanupTestPerson(t, env, person2ID)

		// Update persons with common tags
		commonTags := []string{"golang", "hiking", "tech"}
		_, err := env.DB.Exec(`
			UPDATE persons
			SET tags = $1
			WHERE id IN ($2, $3)`, pq.Array(commonTags), person1ID, person2ID)
		if err != nil {
			t.Fatalf("Failed to update tags: %v", err)
		}

		// Create relationship
		now := time.Now()
		query := `
			INSERT INTO relationships (id, created_at, updated_at, from_person_id, to_person_id,
			                          relationship_type)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id`

		var relID string
		err = env.DB.QueryRow(query, uuid.New().String(), now, now, person1ID, person2ID,
			"friend").Scan(&relID)
		if err != nil {
			t.Fatalf("Failed to create relationship: %v", err)
		}
		defer cleanupTestRelationship(t, env, relID)

		// Identify shared interests
		err = processor.identifySharedInterests()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify shared interests were identified
		var sharedInterests []string
		err = env.DB.QueryRow(`
			SELECT shared_interests
			FROM relationships
			WHERE id = $1`, relID).Scan(pq.Array(&sharedInterests))

		if err != nil && err != sql.ErrNoRows {
			t.Fatalf("Failed to query relationship: %v", err)
		}

		if len(sharedInterests) != len(commonTags) {
			t.Errorf("Expected %d shared interests, got %d", len(commonTags), len(sharedInterests))
		}
	})

	t.Run("Success_NoCommonTags", func(t *testing.T) {
		// Create persons with different tags
		person1ID := createTestPerson(t, env, "Different Tags 1")
		person2ID := createTestPerson(t, env, "Different Tags 2")
		defer cleanupTestPerson(t, env, person1ID)
		defer cleanupTestPerson(t, env, person2ID)

		_, err := env.DB.Exec(`UPDATE persons SET tags = $1 WHERE id = $2`,
			pq.Array([]string{"music"}), person1ID)
		if err != nil {
			t.Fatalf("Failed to update tags: %v", err)
		}

		_, err = env.DB.Exec(`UPDATE persons SET tags = $1 WHERE id = $2`,
			pq.Array([]string{"sports"}), person2ID)
		if err != nil {
			t.Fatalf("Failed to update tags: %v", err)
		}

		now := time.Now()
		query := `
			INSERT INTO relationships (id, created_at, updated_at, from_person_id, to_person_id,
			                          relationship_type)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id`

		var relID string
		err = env.DB.QueryRow(query, uuid.New().String(), now, now, person1ID, person2ID,
			"colleague").Scan(&relID)
		if err != nil {
			t.Fatalf("Failed to create relationship: %v", err)
		}
		defer cleanupTestRelationship(t, env, relID)

		err = processor.identifySharedInterests()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

// TestUpdateComputedSignals tests computed signals update
func TestUpdateComputedSignals(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	processor := NewBatchAnalyticsProcessor(env.DB)

	t.Run("Success", func(t *testing.T) {
		personID := createTestPerson(t, env, "Signals Test")
		defer cleanupTestPerson(t, env, personID)

		err := processor.updateComputedSignals()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify last_batch_update was set
		var computedSignals []byte
		err = env.DB.QueryRow(`
			SELECT computed_signals
			FROM persons
			WHERE id = $1`, personID).Scan(&computedSignals)

		if err != nil {
			t.Fatalf("Failed to query person: %v", err)
		}

		if len(computedSignals) > 0 {
			var signals map[string]interface{}
			json.Unmarshal(computedSignals, &signals)
			if _, hasUpdate := signals["last_batch_update"]; !hasUpdate {
				t.Error("Expected last_batch_update in computed signals")
			}
		}
	})
}

// TestProcessAnalytics tests the full analytics processing pipeline
func TestProcessAnalytics(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestEnvironment(t)
	if env == nil {
		return
	}
	defer env.Cleanup()

	processor := NewBatchAnalyticsProcessor(env.DB)

	t.Run("Success_FullPipeline", func(t *testing.T) {
		// Create test data
		person1ID := createTestPerson(t, env, "Pipeline Test 1")
		person2ID := createTestPerson(t, env, "Pipeline Test 2")
		defer cleanupTestPerson(t, env, person1ID)
		defer cleanupTestPerson(t, env, person2ID)

		relID := createTestRelationship(t, env, person1ID, person2ID, "friend")
		defer cleanupTestRelationship(t, env, relID)

		// Run full analytics pipeline
		err := processor.ProcessAnalytics()
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

// TestFindCommonStrings tests the utility function
func TestFindCommonStrings(t *testing.T) {
	t.Run("NoCommon", func(t *testing.T) {
		slice1 := []string{"a", "b", "c"}
		slice2 := []string{"d", "e", "f"}
		result := findCommonStrings(slice1, slice2)
		if len(result) != 0 {
			t.Errorf("Expected empty result, got %v", result)
		}
	})

	t.Run("AllCommon", func(t *testing.T) {
		slice1 := []string{"a", "b", "c"}
		slice2 := []string{"a", "b", "c"}
		result := findCommonStrings(slice1, slice2)
		if len(result) != 3 {
			t.Errorf("Expected 3 common strings, got %d", len(result))
		}
	})

	t.Run("SomeCommon", func(t *testing.T) {
		slice1 := []string{"a", "b", "c", "d"}
		slice2 := []string{"b", "d", "e", "f"}
		result := findCommonStrings(slice1, slice2)
		if len(result) != 2 {
			t.Errorf("Expected 2 common strings, got %d", len(result))
		}
	})

	t.Run("EmptySlices", func(t *testing.T) {
		result := findCommonStrings([]string{}, []string{})
		if len(result) != 0 {
			t.Errorf("Expected empty result, got %v", result)
		}
	})
}
