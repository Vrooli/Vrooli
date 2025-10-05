// +build testing

package main

import (
	"net/http"
	"testing"
)

// TestIntegration_TechTreeDataFlow tests the complete data flow through tech tree endpoints
func TestIntegration_TechTreeDataFlow(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	router := setupTestRouter()

	t.Run("Complete tech tree data retrieval flow", func(t *testing.T) {
		// Setup comprehensive test data
		treeID := createTestTechTree(t, testDB)
		sector1ID := createTestSector(t, testDB, treeID)

		// Create second sector
		sector2ID := "00000000-0000-0000-0000-000000000008"
		testDB.Exec(`
			INSERT INTO sectors (id, tree_id, name, category, description, progress_percentage,
				position_x, position_y, color, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), NOW())
		`, sector2ID, treeID, "Manufacturing", "manufacturing", "Manufacturing systems",
			30.0, 300.0, 200.0, "#e74c3c")

		stage1ID := createTestStage(t, testDB, sector1ID)

		// Create second stage
		stage2ID := "00000000-0000-0000-0000-000000000009"
		examples := `["Example 3", "Example 4"]`
		testDB.Exec(`
			INSERT INTO progression_stages (id, sector_id, stage_type, stage_order, name, description,
				progress_percentage, examples, position_x, position_y, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		`, stage2ID, sector2ID, "intermediate", 2, "Intermediate Stage", "Intermediate level",
			50.0, examples, 350.0, 250.0)

		createTestScenarioMapping(t, testDB, stage1ID, "graph-studio")
		createTestScenarioMapping(t, testDB, stage2ID, "research-assistant")
		createTestMilestone(t, testDB, treeID)
		createTestDependency(t, testDB, stage2ID, stage1ID)
		createTestSectorConnection(t, testDB, sector1ID, sector2ID)

		// 1. Get tech tree
		t.Run("Retrieve tech tree", func(t *testing.T) {
			w := makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree", nil)
			response := assertJSONResponse(t, w, http.StatusOK)
			assertFieldExists(t, response, "id")
			assertFieldExists(t, response, "name")
			assertFieldExists(t, response, "version")
		})

		// 2. Get all sectors with stages
		t.Run("Retrieve all sectors", func(t *testing.T) {
			w := makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree/sectors", nil)
			response := assertJSONResponse(t, w, http.StatusOK)
			sectors := assertArrayLength(t, response, "sectors", 2)

			// Validate sector structure
			if len(sectors) > 0 {
				sector := sectors[0].(map[string]interface{})
				assertFieldExists(t, sector, "id")
				assertFieldExists(t, sector, "name")
				assertFieldExists(t, sector, "category")
				assertFieldExists(t, sector, "progress_percentage")
				assertFieldExists(t, sector, "stages")
			}
		})

		// 3. Get specific sector with stages and mappings
		t.Run("Retrieve specific sector", func(t *testing.T) {
			w := makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree/sectors/"+sector1ID, nil)
			response := assertJSONResponse(t, w, http.StatusOK)
			assertFieldExists(t, response, "stages")

			stages := response["stages"].([]interface{})
			if len(stages) > 0 {
				stage := stages[0].(map[string]interface{})
				assertFieldExists(t, stage, "id")
				assertFieldExists(t, stage, "scenario_mappings")
			}
		})

		// 4. Get specific stage
		t.Run("Retrieve specific stage", func(t *testing.T) {
			w := makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree/stages/"+stage1ID, nil)
			response := assertJSONResponse(t, w, http.StatusOK)
			assertFieldExists(t, response, "id")
			assertFieldExists(t, response, "stage_type")
			assertFieldExists(t, response, "progress_percentage")
		})

		// 5. Get dependencies
		t.Run("Retrieve dependencies", func(t *testing.T) {
			w := makeHTTPRequest(t, router, "GET", "/api/v1/dependencies", nil)
			response := assertJSONResponse(t, w, http.StatusOK)
			deps := assertArrayLength(t, response, "dependencies", 1)

			if len(deps) > 0 {
				dep := deps[0].(map[string]interface{})
				assertFieldExists(t, dep, "dependency")
				assertFieldExists(t, dep, "dependent_name")
				assertFieldExists(t, dep, "prerequisite_name")
			}
		})

		// 6. Get cross-sector connections
		t.Run("Retrieve connections", func(t *testing.T) {
			w := makeHTTPRequest(t, router, "GET", "/api/v1/connections", nil)
			response := assertJSONResponse(t, w, http.StatusOK)
			conns := assertArrayLength(t, response, "connections", 1)

			if len(conns) > 0 {
				conn := conns[0].(map[string]interface{})
				assertFieldExists(t, conn, "connection")
				assertFieldExists(t, conn, "source_name")
				assertFieldExists(t, conn, "target_name")
			}
		})
	})
}

// TestIntegration_ProgressTracking tests scenario progress tracking workflow
func TestIntegration_ProgressTracking(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	router := setupTestRouter()

	t.Run("Complete progress tracking workflow", func(t *testing.T) {
		// Setup
		treeID := createTestTechTree(t, testDB)
		sectorID := createTestSector(t, testDB, treeID)
		stageID := createTestStage(t, testDB, sectorID)

		// 1. Create new scenario mapping
		t.Run("Create scenario mapping", func(t *testing.T) {
			body := map[string]interface{}{
				"scenario_name":       "integration-test-scenario",
				"stage_id":            stageID,
				"contribution_weight": 0.95,
				"completion_status":   "not_started",
				"priority":            1,
				"estimated_impact":    9.5,
				"notes":               "Integration test scenario",
			}

			w := makeHTTPRequest(t, router, "POST", "/api/v1/progress/scenarios", body)
			response := assertJSONResponse(t, w, http.StatusOK)
			assertFieldExists(t, response, "message")
			assertFieldExists(t, response, "mapping")
		})

		// 2. Retrieve all scenario mappings
		t.Run("Get all scenario mappings", func(t *testing.T) {
			w := makeHTTPRequest(t, router, "GET", "/api/v1/progress/scenarios", nil)
			response := assertJSONResponse(t, w, http.StatusOK)
			mappings := assertArrayLength(t, response, "scenario_mappings", 1)

			if len(mappings) > 0 {
				mapping := mappings[0].(map[string]interface{})
				assertFieldExists(t, mapping, "mapping")
				assertFieldExists(t, mapping, "stage_name")
				assertFieldExists(t, mapping, "sector_name")
			}
		})

		// 3. Update scenario status
		t.Run("Update scenario status", func(t *testing.T) {
			body := map[string]interface{}{
				"completion_status": "in_progress",
				"notes":             "Work started on integration test",
				"estimated_impact":  9.8,
			}

			w := makeHTTPRequest(t, router, "PUT", "/api/v1/progress/scenarios/integration-test-scenario", body)
			response := assertJSONResponse(t, w, http.StatusOK)

			if scenario, ok := response["scenario"].(string); !ok || scenario != "integration-test-scenario" {
				t.Errorf("Expected scenario 'integration-test-scenario', got '%v'", response["scenario"])
			}
		})

		// 4. Verify updated status
		t.Run("Verify status update", func(t *testing.T) {
			w := makeHTTPRequest(t, router, "GET", "/api/v1/progress/scenarios", nil)
			response := assertJSONResponse(t, w, http.StatusOK)
			mappings := response["scenario_mappings"].([]interface{})

			// Find our mapping
			found := false
			for _, m := range mappings {
				mapping := m.(map[string]interface{})["mapping"].(map[string]interface{})
				if mapping["scenario_name"].(string) == "integration-test-scenario" {
					found = true
					if status, ok := mapping["completion_status"].(string); !ok || status != "in_progress" {
						t.Errorf("Expected status 'in_progress', got '%v'", mapping["completion_status"])
					}
					break
				}
			}

			if !found {
				t.Error("Could not find updated scenario mapping")
			}
		})
	})
}

// TestIntegration_StrategicAnalysis tests strategic analysis and recommendations workflow
func TestIntegration_StrategicAnalysis(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	router := setupTestRouter()

	t.Run("Complete strategic analysis workflow", func(t *testing.T) {
		// Setup
		treeID := createTestTechTree(t, testDB)
		createTestMilestone(t, testDB, treeID)

		// 1. Get default recommendations
		t.Run("Get default recommendations", func(t *testing.T) {
			w := makeHTTPRequest(t, router, "GET", "/api/v1/recommendations", nil)
			response := assertJSONResponse(t, w, http.StatusOK)
			recs := assertArrayLength(t, response, "recommendations", 1)

			// Validate recommendation structure
			if len(recs) > 0 {
				rec := recs[0].(map[string]interface{})
				assertFieldExists(t, rec, "scenario")
				assertFieldExists(t, rec, "priority_score")
				assertFieldExists(t, rec, "impact_multiplier")
				assertFieldExists(t, rec, "reasoning")

				// Validate score values
				if score, ok := rec["priority_score"].(float64); !ok || score < 0 || score > 10 {
					t.Errorf("Expected priority_score between 0 and 10, got %v", rec["priority_score"])
				}
			}
		})

		// 2. Run strategic analysis
		t.Run("Analyze strategic path", func(t *testing.T) {
			body := map[string]interface{}{
				"current_resources": 7,
				"time_horizon":      24,
				"priority_sectors":  []string{"software", "manufacturing", "healthcare"},
			}

			w := makeHTTPRequest(t, router, "POST", "/api/v1/tech-tree/analyze", body)
			response := assertJSONResponse(t, w, http.StatusOK)

			// Validate all analysis components
			assertFieldExists(t, response, "recommendations")
			assertFieldExists(t, response, "projected_timeline")
			assertFieldExists(t, response, "bottleneck_analysis")
			assertFieldExists(t, response, "cross_sector_impacts")

			// Validate projected timeline structure
			timeline := response["projected_timeline"].(map[string]interface{})
			milestones := assertArrayLength(t, timeline, "milestones", 1)

			if len(milestones) > 0 {
				milestone := milestones[0].(map[string]interface{})
				assertFieldExists(t, milestone, "name")
				assertFieldExists(t, milestone, "estimated_completion")
				assertFieldExists(t, milestone, "confidence")

				// Validate confidence level
				if conf, ok := milestone["confidence"].(float64); !ok || conf < 0 || conf > 1 {
					t.Errorf("Expected confidence between 0 and 1, got %v", milestone["confidence"])
				}
			}

			// Validate cross-sector impacts
			impacts := assertArrayLength(t, response, "cross_sector_impacts", 1)
			if len(impacts) > 0 {
				impact := impacts[0].(map[string]interface{})
				assertFieldExists(t, impact, "source_sector")
				assertFieldExists(t, impact, "target_sector")
				assertFieldExists(t, impact, "impact_score")
				assertFieldExists(t, impact, "description")
			}
		})

		// 3. Get strategic milestones
		t.Run("Get strategic milestones", func(t *testing.T) {
			w := makeHTTPRequest(t, router, "GET", "/api/v1/milestones", nil)
			response := assertJSONResponse(t, w, http.StatusOK)
			milestones := assertArrayLength(t, response, "milestones", 1)

			if len(milestones) > 0 {
				milestone := milestones[0].(map[string]interface{})
				assertFieldExists(t, milestone, "id")
				assertFieldExists(t, milestone, "name")
				assertFieldExists(t, milestone, "milestone_type")
				assertFieldExists(t, milestone, "completion_percentage")
				assertFieldExists(t, milestone, "business_value_estimate")
			}
		})

		// 4. Test analysis with edge cases
		t.Run("Analyze with minimal resources", func(t *testing.T) {
			body := map[string]interface{}{
				"current_resources": 1,
				"time_horizon":      3,
				"priority_sectors":  []string{},
			}

			w := makeHTTPRequest(t, router, "POST", "/api/v1/tech-tree/analyze", body)
			response := assertJSONResponse(t, w, http.StatusOK)

			// Should still provide recommendations
			assertArrayLength(t, response, "recommendations", 0)
			assertFieldExists(t, response, "projected_timeline")
		})
	})
}

// TestIntegration_CORSAndSecurity tests CORS headers and basic security
func TestIntegration_CORSAndSecurity(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router := setupTestRouter()

	t.Run("CORS preflight request", func(t *testing.T) {
		w := makeHTTPRequest(t, router, "OPTIONS", "/api/v1/tech-tree", nil)

		if w.Code != http.StatusNoContent {
			t.Errorf("Expected status 204 for OPTIONS, got %d", w.Code)
		}

		// Check CORS headers
		if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
			t.Errorf("Expected CORS origin '*', got '%s'", origin)
		}

		if methods := w.Header().Get("Access-Control-Allow-Methods"); methods == "" {
			t.Error("Expected Access-Control-Allow-Methods header")
		}
	})

	t.Run("CORS headers on regular request", func(t *testing.T) {
		w := makeHTTPRequest(t, router, "GET", "/health", nil)

		if origin := w.Header().Get("Access-Control-Allow-Origin"); origin != "*" {
			t.Errorf("Expected CORS origin '*', got '%s'", origin)
		}
	})
}

// TestIntegration_ErrorHandling tests comprehensive error handling across endpoints
func TestIntegration_ErrorHandling(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	router := setupTestRouter()

	t.Run("Invalid UUID handling across endpoints", func(t *testing.T) {
		invalidID := "not-a-uuid"

		endpoints := []struct {
			path   string
			method string
		}{
			{"/api/v1/tech-tree/sectors/" + invalidID, "GET"},
			{"/api/v1/tech-tree/stages/" + invalidID, "GET"},
		}

		for _, ep := range endpoints {
			t.Run(ep.method+" "+ep.path, func(t *testing.T) {
				w := makeHTTPRequest(t, router, ep.method, ep.path, nil)
				if w.Code == http.StatusOK {
					t.Errorf("Expected error for invalid UUID on %s %s", ep.method, ep.path)
				}
			})
		}
	})

	t.Run("Non-existent resource handling", func(t *testing.T) {
		nonExistentID := "00000000-0000-0000-0000-999999999999"

		endpoints := []struct {
			path   string
			method string
		}{
			{"/api/v1/tech-tree/sectors/" + nonExistentID, "GET"},
			{"/api/v1/tech-tree/stages/" + nonExistentID, "GET"},
		}

		for _, ep := range endpoints {
			t.Run(ep.method+" "+ep.path, func(t *testing.T) {
				w := makeHTTPRequest(t, router, ep.method, ep.path, nil)
				assertErrorResponse(t, w, http.StatusNotFound, "not found")
			})
		}
	})

	t.Run("Malformed JSON handling", func(t *testing.T) {
		endpoints := []string{
			"/api/v1/progress/scenarios",
			"/api/v1/tech-tree/analyze",
		}

		for _, path := range endpoints {
			t.Run("POST "+path, func(t *testing.T) {
				w := makeHTTPRequest(t, router, "POST", path, "invalid-json-{")
				assertErrorResponse(t, w, http.StatusBadRequest, "")
			})
		}
	})
}
