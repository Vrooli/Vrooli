// +build testing

package main

import (
	"net/http"
	"testing"
)

func TestHealthEndpoint(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("Health check returns healthy status", func(t *testing.T) {
		w := makeHTTPRequest(t, router, "GET", "/health", nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		if status, ok := response["status"].(string); !ok || status != "healthy" {
			t.Errorf("Expected status 'healthy', got '%v'", response["status"])
		}

		if service, ok := response["service"].(string); !ok || service != "tech-tree-designer" {
			t.Errorf("Expected service 'tech-tree-designer', got '%v'", response["service"])
		}
	})

	t.Run("Invalid HTTP methods on health endpoint", func(t *testing.T) {
		invalidMethods := []string{"POST", "PUT", "DELETE", "PATCH"}
		for _, method := range invalidMethods {
			w := makeHTTPRequest(t, router, method, "/health", nil)
			if w.Code == http.StatusOK {
				t.Errorf("Method %s should not be allowed on /health", method)
			}
		}
	})
}

func TestGetTechTree(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return // Skip if database not available
	}
	defer dbCleanup()

	router := setupTestRouter()

	t.Run("Success - Get active tech tree", func(t *testing.T) {
		// Setup: Create test tech tree
		treeID := createTestTechTree(t, testDB)

		w := makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree", nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		// Validate response fields
		if id, ok := response["id"].(string); !ok || id != treeID {
			t.Errorf("Expected tree ID '%s', got '%v'", treeID, response["id"])
		}

		if name, ok := response["name"].(string); !ok || name == "" {
			t.Error("Expected non-empty name field")
		}

		if _, ok := response["version"].(string); !ok {
			t.Error("Expected version field")
		}
	})

	t.Run("Error - No active tech tree", func(t *testing.T) {
		// Cleanup all tech trees
		testDB.Exec("DELETE FROM tech_trees")

		w := makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree", nil)
		if w.Code != http.StatusInternalServerError && w.Code != http.StatusNotFound {
			t.Errorf("Expected error status when no tech tree exists, got %d", w.Code)
		}
	})
}

func TestGetSectors(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	router := setupTestRouter()

	t.Run("Success - Get all sectors with stages", func(t *testing.T) {
		// Setup: Create test data
		treeID := createTestTechTree(t, testDB)
		sectorID := createTestSector(t, testDB, treeID)
		createTestStage(t, testDB, sectorID)

		w := makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree/sectors", nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		// Validate sectors array
		if sectors, ok := response["sectors"].([]interface{}); !ok {
			t.Error("Expected sectors array in response")
		} else if len(sectors) == 0 {
			t.Error("Expected at least one sector")
		} else {
			// Validate first sector
			sector := sectors[0].(map[string]interface{})
			if _, ok := sector["id"]; !ok {
				t.Error("Sector missing id field")
			}
			if _, ok := sector["name"]; !ok {
				t.Error("Sector missing name field")
			}
			if _, ok := sector["category"]; !ok {
				t.Error("Sector missing category field")
			}
			if _, ok := sector["progress_percentage"]; !ok {
				t.Error("Sector missing progress_percentage field")
			}
		}
	})

	t.Run("Success - Empty sectors list", func(t *testing.T) {
		// Cleanup sectors
		testDB.Exec("DELETE FROM sectors")

		w := makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree/sectors", nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		if sectors, ok := response["sectors"].([]interface{}); !ok {
			t.Error("Expected sectors array in response")
		} else if sectors != nil && len(sectors) != 0 {
			t.Errorf("Expected empty sectors array, got %d sectors", len(sectors))
		}
	})
}

func TestGetSector(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	router := setupTestRouter()

	t.Run("Success - Get specific sector", func(t *testing.T) {
		// Setup
		treeID := createTestTechTree(t, testDB)
		sectorID := createTestSector(t, testDB, treeID)
		stageID := createTestStage(t, testDB, sectorID)
		createTestScenarioMapping(t, testDB, stageID, "test-scenario")

		w := makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree/sectors/"+sectorID, nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		// Validate sector fields
		if id, ok := response["id"].(string); !ok || id != sectorID {
			t.Errorf("Expected sector ID '%s', got '%v'", sectorID, response["id"])
		}

		if name, ok := response["name"].(string); !ok || name == "" {
			t.Error("Expected non-empty sector name")
		}

		// Validate stages are loaded
		if stages, ok := response["stages"].([]interface{}); !ok {
			t.Error("Expected stages array")
		} else if len(stages) == 0 {
			t.Error("Expected at least one stage")
		}
	})

	t.Run("Error - Sector not found", func(t *testing.T) {
		nonExistentID := "00000000-0000-0000-0000-999999999999"
		w := makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree/sectors/"+nonExistentID, nil)
		assertErrorResponse(t, w, http.StatusNotFound, "not found")
	})

	t.Run("Error - Invalid sector ID", func(t *testing.T) {
		w := makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree/sectors/invalid-uuid", nil)
		if w.Code == http.StatusOK {
			t.Error("Expected error for invalid UUID")
		}
	})
}

func TestGetStage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	router := setupTestRouter()

	t.Run("Success - Get specific stage", func(t *testing.T) {
		// Setup
		treeID := createTestTechTree(t, testDB)
		sectorID := createTestSector(t, testDB, treeID)
		stageID := createTestStage(t, testDB, sectorID)

		w := makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree/stages/"+stageID, nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		// Validate stage fields
		if id, ok := response["id"].(string); !ok || id != stageID {
			t.Errorf("Expected stage ID '%s', got '%v'", stageID, response["id"])
		}

		if stageType, ok := response["stage_type"].(string); !ok || stageType == "" {
			t.Error("Expected non-empty stage_type")
		}

		if _, ok := response["stage_order"].(float64); !ok {
			t.Error("Expected stage_order field")
		}
	})

	t.Run("Error - Stage not found", func(t *testing.T) {
		nonExistentID := "00000000-0000-0000-0000-999999999999"
		w := makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree/stages/"+nonExistentID, nil)
		assertErrorResponse(t, w, http.StatusNotFound, "not found")
	})
}

func TestGetScenarioMappings(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	router := setupTestRouter()

	t.Run("Success - Get all scenario mappings", func(t *testing.T) {
		// Setup
		treeID := createTestTechTree(t, testDB)
		sectorID := createTestSector(t, testDB, treeID)
		stageID := createTestStage(t, testDB, sectorID)
		createTestScenarioMapping(t, testDB, stageID, "test-scenario-1")
		createTestScenarioMapping(t, testDB, stageID, "test-scenario-2")

		w := makeHTTPRequest(t, router, "GET", "/api/v1/progress/scenarios", nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		if mappings, ok := response["scenario_mappings"].([]interface{}); !ok {
			t.Error("Expected scenario_mappings array")
		} else if len(mappings) < 2 {
			t.Errorf("Expected at least 2 mappings, got %d", len(mappings))
		}
	})
}

func TestUpdateScenarioMapping(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	router := setupTestRouter()

	t.Run("Success - Create new scenario mapping", func(t *testing.T) {
		// Setup
		treeID := createTestTechTree(t, testDB)
		sectorID := createTestSector(t, testDB, treeID)
		stageID := createTestStage(t, testDB, sectorID)

		body := map[string]interface{}{
			"scenario_name":       "new-test-scenario",
			"stage_id":            stageID,
			"contribution_weight": 0.9,
			"completion_status":   "not_started",
			"priority":            2,
			"estimated_impact":    8.5,
			"notes":               "Test scenario mapping",
		}

		w := makeHTTPRequest(t, router, "POST", "/api/v1/progress/scenarios", body)
		response := assertJSONResponse(t, w, http.StatusOK)

		if message, ok := response["message"].(string); !ok || message == "" {
			t.Error("Expected success message")
		}
	})

	t.Run("Error - Invalid request body", func(t *testing.T) {
		w := makeHTTPRequest(t, router, "POST", "/api/v1/progress/scenarios", "invalid-json")
		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
		}
	})

	t.Run("Error - Missing required fields", func(t *testing.T) {
		body := map[string]interface{}{
			"scenario_name": "incomplete-scenario",
		}

		w := makeHTTPRequest(t, router, "POST", "/api/v1/progress/scenarios", body)
		if w.Code == http.StatusOK {
			t.Error("Expected error for missing required fields")
		}
	})
}

func TestUpdateScenarioStatus(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	router := setupTestRouter()

	t.Run("Success - Update scenario status", func(t *testing.T) {
		// Setup
		treeID := createTestTechTree(t, testDB)
		sectorID := createTestSector(t, testDB, treeID)
		stageID := createTestStage(t, testDB, sectorID)
		createTestScenarioMapping(t, testDB, stageID, "update-test-scenario")

		body := map[string]interface{}{
			"completion_status": "completed",
			"notes":             "Scenario completed successfully",
			"estimated_impact":  9.0,
		}

		w := makeHTTPRequest(t, router, "PUT", "/api/v1/progress/scenarios/update-test-scenario", body)
		response := assertJSONResponse(t, w, http.StatusOK)

		if scenario, ok := response["scenario"].(string); !ok || scenario != "update-test-scenario" {
			t.Errorf("Expected scenario name 'update-test-scenario', got '%v'", response["scenario"])
		}

		if status, ok := response["status"].(string); !ok || status != "completed" {
			t.Errorf("Expected status 'completed', got '%v'", response["status"])
		}
	})

	t.Run("Error - Invalid request body", func(t *testing.T) {
		w := makeHTTPRequest(t, router, "PUT", "/api/v1/progress/scenarios/test", "invalid")
		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})
}

func TestAnalyzeStrategicPath(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("Success - Generate strategic analysis", func(t *testing.T) {
		body := map[string]interface{}{
			"current_resources": 5,
			"time_horizon":      12,
			"priority_sectors":  []string{"software", "manufacturing"},
		}

		w := makeHTTPRequest(t, router, "POST", "/api/v1/tech-tree/analyze", body)
		response := assertJSONResponse(t, w, http.StatusOK)

		// Validate recommendations
		if recs, ok := response["recommendations"].([]interface{}); !ok {
			t.Error("Expected recommendations array")
		} else if len(recs) == 0 {
			t.Error("Expected at least one recommendation")
		} else {
			// Validate recommendation structure
			rec := recs[0].(map[string]interface{})
			if _, ok := rec["scenario"]; !ok {
				t.Error("Recommendation missing scenario field")
			}
			if _, ok := rec["priority_score"]; !ok {
				t.Error("Recommendation missing priority_score field")
			}
			if _, ok := rec["impact_multiplier"]; !ok {
				t.Error("Recommendation missing impact_multiplier field")
			}
			if _, ok := rec["reasoning"]; !ok {
				t.Error("Recommendation missing reasoning field")
			}
		}

		// Validate projected timeline
		if timeline, ok := response["projected_timeline"].(map[string]interface{}); !ok {
			t.Error("Expected projected_timeline object")
		} else {
			if milestones, ok := timeline["milestones"].([]interface{}); !ok {
				t.Error("Expected milestones array in timeline")
			} else if len(milestones) == 0 {
				t.Error("Expected at least one milestone")
			}
		}

		// Validate bottleneck analysis
		if bottlenecks, ok := response["bottleneck_analysis"].([]interface{}); !ok {
			t.Error("Expected bottleneck_analysis array")
		} else if len(bottlenecks) == 0 {
			t.Error("Expected at least one bottleneck")
		}

		// Validate cross-sector impacts
		if impacts, ok := response["cross_sector_impacts"].([]interface{}); !ok {
			t.Error("Expected cross_sector_impacts array")
		} else if len(impacts) == 0 {
			t.Error("Expected at least one cross-sector impact")
		}
	})

	t.Run("Error - Invalid request body", func(t *testing.T) {
		w := makeHTTPRequest(t, router, "POST", "/api/v1/tech-tree/analyze", "invalid")
		assertErrorResponse(t, w, http.StatusBadRequest, "")
	})

	t.Run("Edge case - Empty priority sectors", func(t *testing.T) {
		body := map[string]interface{}{
			"current_resources": 3,
			"time_horizon":      6,
			"priority_sectors":  []string{},
		}

		w := makeHTTPRequest(t, router, "POST", "/api/v1/tech-tree/analyze", body)
		response := assertJSONResponse(t, w, http.StatusOK)

		// Should still return recommendations
		if _, ok := response["recommendations"]; !ok {
			t.Error("Expected recommendations even with empty priority sectors")
		}
	})

	t.Run("Edge case - Zero resources", func(t *testing.T) {
		body := map[string]interface{}{
			"current_resources": 0,
			"time_horizon":      12,
			"priority_sectors":  []string{"software"},
		}

		w := makeHTTPRequest(t, router, "POST", "/api/v1/tech-tree/analyze", body)
		// Should still work, just with different recommendations
		assertJSONResponse(t, w, http.StatusOK)
	})
}

func TestGetStrategicMilestones(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	router := setupTestRouter()

	t.Run("Success - Get milestones list", func(t *testing.T) {
		w := makeHTTPRequest(t, router, "GET", "/api/v1/milestones", nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		if _, ok := response["milestones"]; !ok {
			t.Error("Expected milestones field in response")
		}
	})
}

func TestGetRecommendations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("Success - Get recommendations with default parameters", func(t *testing.T) {
		w := makeHTTPRequest(t, router, "GET", "/api/v1/recommendations", nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		if recs, ok := response["recommendations"].([]interface{}); !ok {
			t.Error("Expected recommendations array")
		} else if len(recs) == 0 {
			t.Error("Expected at least one recommendation")
		}
	})

	t.Run("Success - Get recommendations with custom resources", func(t *testing.T) {
		w := makeHTTPRequest(t, router, "GET", "/api/v1/recommendations?resources=8", nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		if _, ok := response["recommendations"]; !ok {
			t.Error("Expected recommendations field")
		}
	})

	t.Run("Edge case - Invalid resources parameter", func(t *testing.T) {
		w := makeHTTPRequest(t, router, "GET", "/api/v1/recommendations?resources=invalid", nil)
		// Should use default value
		assertJSONResponse(t, w, http.StatusOK)
	})
}

func TestGetDependencies(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	router := setupTestRouter()

	t.Run("Success - Get dependencies list", func(t *testing.T) {
		w := makeHTTPRequest(t, router, "GET", "/api/v1/dependencies", nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		if _, ok := response["dependencies"]; !ok {
			t.Error("Expected dependencies field in response")
		}
	})
}

func TestGetCrossSectorConnections(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	router := setupTestRouter()

	t.Run("Success - Get connections list", func(t *testing.T) {
		w := makeHTTPRequest(t, router, "GET", "/api/v1/connections", nil)
		response := assertJSONResponse(t, w, http.StatusOK)

		if _, ok := response["connections"]; !ok {
			t.Error("Expected connections field in response")
		}
	})
}

// Integration test for complete workflow
func TestCompleteWorkflow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	router := setupTestRouter()

	t.Run("Complete tech tree workflow", func(t *testing.T) {
		// 1. Get tech tree
		w := makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree", nil)
		if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
			t.Errorf("Unexpected status for tech tree: %d", w.Code)
		}

		// 2. Get sectors
		w = makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree/sectors", nil)
		assertJSONResponse(t, w, http.StatusOK)

		// 3. Get recommendations
		w = makeHTTPRequest(t, router, "GET", "/api/v1/recommendations", nil)
		assertJSONResponse(t, w, http.StatusOK)

		// 4. Analyze strategic path
		body := map[string]interface{}{
			"current_resources": 5,
			"time_horizon":      12,
			"priority_sectors":  []string{"software"},
		}
		w = makeHTTPRequest(t, router, "POST", "/api/v1/tech-tree/analyze", body)
		assertJSONResponse(t, w, http.StatusOK)
	})
}
