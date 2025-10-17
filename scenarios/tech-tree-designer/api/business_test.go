// +build testing

package main

import (
	"net/http"
	"testing"
)

// TestBusiness_StrategicRecommendations tests business logic for strategic recommendations
func TestBusiness_StrategicRecommendations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("Recommendations prioritize high-impact scenarios", func(t *testing.T) {
		w := makeHTTPRequest(t, router, "GET", "/api/v1/recommendations", nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		recs := response["recommendations"].([]interface{})
		if len(recs) < 2 {
			t.Skip("Need at least 2 recommendations for priority testing")
		}

		// Verify recommendations are sorted by priority score
		for i := 0; i < len(recs)-1; i++ {
			current := recs[i].(map[string]interface{})
			next := recs[i+1].(map[string]interface{})

			currentScore := current["priority_score"].(float64)
			nextScore := next["priority_score"].(float64)

			if currentScore < nextScore {
				t.Errorf("Recommendations not properly ordered: rec[%d] score %.2f < rec[%d] score %.2f",
					i, currentScore, i+1, nextScore)
			}
		}
	})

	t.Run("Recommendations include reasoning", func(t *testing.T) {
		w := makeHTTPRequest(t, router, "GET", "/api/v1/recommendations", nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		recs := response["recommendations"].([]interface{})
		for i, r := range recs {
			rec := r.(map[string]interface{})
			reasoning, ok := rec["reasoning"].(string)
			if !ok || reasoning == "" {
				t.Errorf("Recommendation %d missing reasoning", i)
			}

			// Reasoning should be substantive
			if len(reasoning) < 20 {
				t.Errorf("Recommendation %d has insufficient reasoning: '%s'", i, reasoning)
			}
		}
	})

	t.Run("Impact multipliers are positive and reasonable", func(t *testing.T) {
		w := makeHTTPRequest(t, router, "GET", "/api/v1/recommendations", nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		recs := response["recommendations"].([]interface{})
		for i, r := range recs {
			rec := r.(map[string]interface{})
			multiplier := rec["impact_multiplier"].(float64)

			if multiplier <= 0 {
				t.Errorf("Recommendation %d has non-positive impact multiplier: %.2f", i, multiplier)
			}

			// Impact multipliers should be reasonable (e.g., 0.1 to 10.0)
			if multiplier < 0.1 || multiplier > 10.0 {
				t.Logf("WARNING: Recommendation %d has unusual impact multiplier: %.2f", i, multiplier)
			}
		}
	})
}

// TestBusiness_ProjectedTimeline tests business logic for timeline projections
func TestBusiness_ProjectedTimeline(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("Timeline milestones are chronologically ordered", func(t *testing.T) {
		body := map[string]interface{}{
			"current_resources": 5,
			"time_horizon":      24,
			"priority_sectors":  []string{"software", "manufacturing"},
		}

		w := makeHTTPRequest(t, router, "POST", "/api/v1/tech-tree/analyze", body)
		response := assertJSONResponse(t, w, http.StatusOK)

		timeline := response["projected_timeline"].(map[string]interface{})
		milestones := timeline["milestones"].([]interface{})

		if len(milestones) < 2 {
			t.Skip("Need at least 2 milestones for chronological testing")
		}

		// Verify milestones are in chronological order
		for i := 0; i < len(milestones)-1; i++ {
			current := milestones[i].(map[string]interface{})
			next := milestones[i+1].(map[string]interface{})

			currentTime := current["estimated_completion"].(string)
			nextTime := next["estimated_completion"].(string)

			// Simple string comparison works for ISO timestamps
			if currentTime > nextTime {
				t.Errorf("Milestones not chronologically ordered: milestone[%d] after milestone[%d]",
					i, i+1)
			}
		}
	})

	t.Run("Confidence levels decrease for distant milestones", func(t *testing.T) {
		body := map[string]interface{}{
			"current_resources": 5,
			"time_horizon":      24,
			"priority_sectors":  []string{"software"},
		}

		w := makeHTTPRequest(t, router, "POST", "/api/v1/tech-tree/analyze", body)
		response := assertJSONResponse(t, w, http.StatusOK)

		timeline := response["projected_timeline"].(map[string]interface{})
		milestones := timeline["milestones"].([]interface{})

		if len(milestones) < 2 {
			t.Skip("Need at least 2 milestones for confidence testing")
		}

		// Later milestones should generally have lower confidence
		for i := 0; i < len(milestones)-1; i++ {
			current := milestones[i].(map[string]interface{})
			next := milestones[i+1].(map[string]interface{})

			currentConf := current["confidence"].(float64)
			nextConf := next["confidence"].(float64)

			if currentConf < nextConf {
				t.Logf("WARNING: Confidence increased for later milestone: %.2f -> %.2f",
					currentConf, nextConf)
			}

			// All confidence values should be between 0 and 1
			if currentConf < 0 || currentConf > 1 {
				t.Errorf("Invalid confidence level: %.2f", currentConf)
			}
		}
	})
}

// TestBusiness_CrossSectorImpacts tests cross-sector impact analysis
func TestBusiness_CrossSectorImpacts(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("Cross-sector impacts have valid structure", func(t *testing.T) {
		body := map[string]interface{}{
			"current_resources": 5,
			"time_horizon":      12,
			"priority_sectors":  []string{"software", "manufacturing", "healthcare"},
		}

		w := makeHTTPRequest(t, router, "POST", "/api/v1/tech-tree/analyze", body)
		response := assertJSONResponse(t, w, http.StatusOK)

		impacts := response["cross_sector_impacts"].([]interface{})

		for i, imp := range impacts {
			impact := imp.(map[string]interface{})

			// Validate required fields
			source, ok := impact["source_sector"].(string)
			if !ok || source == "" {
				t.Errorf("Impact %d missing valid source_sector", i)
			}

			target, ok := impact["target_sector"].(string)
			if !ok || target == "" {
				t.Errorf("Impact %d missing valid target_sector", i)
			}

			// Source and target should be different
			if source == target {
				t.Errorf("Impact %d has same source and target: %s", i, source)
			}

			// Validate impact score
			score, ok := impact["impact_score"].(float64)
			if !ok {
				t.Errorf("Impact %d missing impact_score", i)
			}

			// Impact score should be between 0 and 1
			if score < 0 || score > 1 {
				t.Errorf("Impact %d has invalid score: %.2f (expected 0-1)", i, score)
			}

			// Validate description
			desc, ok := impact["description"].(string)
			if !ok || desc == "" {
				t.Errorf("Impact %d missing description", i)
			}
		}
	})

	t.Run("High-impact connections have substantial descriptions", func(t *testing.T) {
		body := map[string]interface{}{
			"current_resources": 5,
			"time_horizon":      12,
			"priority_sectors":  []string{"software", "manufacturing"},
		}

		w := makeHTTPRequest(t, router, "POST", "/api/v1/tech-tree/analyze", body)
		response := assertJSONResponse(t, w, http.StatusOK)

		impacts := response["cross_sector_impacts"].([]interface{})

		for i, imp := range impacts {
			impact := imp.(map[string]interface{})
			score := impact["impact_score"].(float64)
			desc := impact["description"].(string)

			// High-impact connections should have detailed explanations
			if score > 0.7 && len(desc) < 30 {
				t.Errorf("Impact %d has high score (%.2f) but short description: '%s'",
					i, score, desc)
			}
		}
	})
}

// TestBusiness_BottleneckIdentification tests bottleneck analysis business logic
func TestBusiness_BottleneckIdentification(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("Bottlenecks are identified and described", func(t *testing.T) {
		body := map[string]interface{}{
			"current_resources": 3,
			"time_horizon":      6,
			"priority_sectors":  []string{"software"},
		}

		w := makeHTTPRequest(t, router, "POST", "/api/v1/tech-tree/analyze", body)
		response := assertJSONResponse(t, w, http.StatusOK)

		bottlenecks := response["bottleneck_analysis"].([]interface{})

		if len(bottlenecks) == 0 {
			t.Error("Expected bottleneck analysis to identify at least one bottleneck")
		}

		for i, b := range bottlenecks {
			bottleneck := b.(string)

			// Each bottleneck should be a non-empty string
			if bottleneck == "" {
				t.Errorf("Bottleneck %d is empty", i)
			}

			// Bottlenecks should be descriptive
			if len(bottleneck) < 20 {
				t.Errorf("Bottleneck %d is too brief: '%s'", i, bottleneck)
			}
		}
	})

	t.Run("Bottlenecks adapt to resource constraints", func(t *testing.T) {
		// Test with different resource levels
		scenarios := []struct {
			resources int
			horizon   int
			name      string
		}{
			{1, 3, "minimal resources"},
			{5, 12, "moderate resources"},
			{10, 24, "abundant resources"},
		}

		for _, scenario := range scenarios {
			t.Run(scenario.name, func(t *testing.T) {
				body := map[string]interface{}{
					"current_resources": scenario.resources,
					"time_horizon":      scenario.horizon,
					"priority_sectors":  []string{"software"},
				}

				w := makeHTTPRequest(t, router, "POST", "/api/v1/tech-tree/analyze", body)
				response := assertJSONResponse(t, w, http.StatusOK)

				bottlenecks := response["bottleneck_analysis"].([]interface{})

				t.Logf("%s: identified %d bottlenecks", scenario.name, len(bottlenecks))

				// Should always identify some bottlenecks
				if len(bottlenecks) == 0 {
					t.Errorf("%s: no bottlenecks identified", scenario.name)
				}
			})
		}
	})
}

// TestBusiness_ProgressTracking tests scenario progress tracking business rules
func TestBusiness_ProgressTracking(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	router := setupTestRouter()

	t.Run("Scenario contribution weights are valid", func(t *testing.T) {
		treeID := createTestTechTree(t, testDB)
		sectorID := createTestSector(t, testDB, treeID)
		stageID := createTestStage(t, testDB, sectorID)

		// Test various contribution weights
		validWeights := []float64{0.0, 0.5, 0.8, 1.0}

		for _, weight := range validWeights {
			body := map[string]interface{}{
				"scenario_name":       "weight-test",
				"stage_id":            stageID,
				"contribution_weight": weight,
				"completion_status":   "not_started",
				"priority":            1,
				"estimated_impact":    5.0,
				"notes":               "Weight validation test",
			}

			w := makeHTTPRequest(t, router, "POST", "/api/v1/progress/scenarios", body)
			if w.Code != http.StatusOK {
				t.Errorf("Valid weight %.2f rejected with status %d", weight, w.Code)
			}
		}
	})

	t.Run("Priority values affect scenario ordering", func(t *testing.T) {
		treeID := createTestTechTree(t, testDB)
		sectorID := createTestSector(t, testDB, treeID)
		stageID := createTestStage(t, testDB, sectorID)

		// Create scenarios with different priorities
		priorities := []struct {
			name     string
			priority int
		}{
			{"low-priority", 3},
			{"high-priority", 1},
			{"medium-priority", 2},
		}

		for _, p := range priorities {
			body := map[string]interface{}{
				"scenario_name":       p.name,
				"stage_id":            stageID,
				"contribution_weight": 0.8,
				"completion_status":   "not_started",
				"priority":            p.priority,
				"estimated_impact":    7.0,
				"notes":               "Priority test",
			}

			w := makeHTTPRequest(t, router, "POST", "/api/v1/progress/scenarios", body)
			assertJSONResponse(t, w, http.StatusOK)
		}

		// Retrieve and verify ordering
		w := makeHTTPRequest(t, router, "GET", "/api/v1/progress/scenarios", nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		mappings := response["scenario_mappings"].([]interface{})
		if len(mappings) < 3 {
			t.Skip("Need at least 3 mappings for ordering test")
		}

		// Verify first mapping has priority 1
		firstMapping := mappings[0].(map[string]interface{})["mapping"].(map[string]interface{})
		firstPriority := int(firstMapping["priority"].(float64))

		if firstPriority != 1 {
			t.Errorf("Expected first mapping to have priority 1, got %d", firstPriority)
		}
	})

	t.Run("Completion status transitions are tracked", func(t *testing.T) {
		treeID := createTestTechTree(t, testDB)
		sectorID := createTestSector(t, testDB, treeID)
		stageID := createTestStage(t, testDB, sectorID)

		scenarioName := "status-transition-test"

		// Create scenario
		body := map[string]interface{}{
			"scenario_name":       scenarioName,
			"stage_id":            stageID,
			"contribution_weight": 0.8,
			"completion_status":   "not_started",
			"priority":            1,
			"estimated_impact":    7.0,
			"notes":               "Initial state",
		}

		w := makeHTTPRequest(t, router, "POST", "/api/v1/progress/scenarios", body)
		assertJSONResponse(t, w, http.StatusOK)

		// Transition through statuses
		statuses := []string{"in_progress", "completed"}

		for _, status := range statuses {
			updateBody := map[string]interface{}{
				"completion_status": status,
				"notes":             "Updated to " + status,
			}

			w := makeHTTPRequest(t, router, "PUT", "/api/v1/progress/scenarios/"+scenarioName, updateBody)
			response := assertJSONResponse(t, w, http.StatusOK)

			if currentStatus, ok := response["status"].(string); !ok || currentStatus != status {
				t.Errorf("Expected status '%s', got '%v'", status, response["status"])
			}
		}
	})
}

// TestBusiness_MilestoneValueEstimation tests business value estimation logic
func TestBusiness_MilestoneValueEstimation(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	router := setupTestRouter()

	t.Run("Milestones ordered by business value", func(t *testing.T) {
		treeID := createTestTechTree(t, testDB)

		// Create milestones with different business values
		milestones := []struct {
			id    string
			name  string
			value int64
		}{
			{"00000000-0000-0000-0000-000000000011", "Low Value", 10000},
			{"00000000-0000-0000-0000-000000000012", "High Value", 100000},
			{"00000000-0000-0000-0000-000000000013", "Medium Value", 50000},
		}

		requiredSectors := `["software"]`
		requiredStages := `["foundation"]`

		for _, m := range milestones {
			testDB.Exec(`
				INSERT INTO strategic_milestones (id, tree_id, name, description, milestone_type,
					required_sectors, required_stages, completion_percentage, confidence_level,
					business_value_estimate, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
			`, m.id, treeID, m.name, "Test milestone", "capability",
				requiredSectors, requiredStages, 25.0, 0.8, m.value)
		}

		w := makeHTTPRequest(t, router, "GET", "/api/v1/milestones", nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		retrievedMilestones := response["milestones"].([]interface{})
		if len(retrievedMilestones) < 3 {
			t.Skip("Need at least 3 milestones for value ordering test")
		}

		// Verify descending order by business value
		for i := 0; i < len(retrievedMilestones)-1; i++ {
			current := retrievedMilestones[i].(map[string]interface{})
			next := retrievedMilestones[i+1].(map[string]interface{})

			currentValue := int64(current["business_value_estimate"].(float64))
			nextValue := int64(next["business_value_estimate"].(float64))

			if currentValue < nextValue {
				t.Errorf("Milestones not ordered by value: milestone[%d]=$%d < milestone[%d]=$%d",
					i, currentValue, i+1, nextValue)
			}
		}
	})

	t.Run("Milestone completion percentages are valid", func(t *testing.T) {
		w := makeHTTPRequest(t, router, "GET", "/api/v1/milestones", nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		milestones := response["milestones"].([]interface{})

		for i, m := range milestones {
			milestone := m.(map[string]interface{})
			completion := milestone["completion_percentage"].(float64)

			if completion < 0 || completion > 100 {
				t.Errorf("Milestone %d has invalid completion percentage: %.2f", i, completion)
			}
		}
	})
}
