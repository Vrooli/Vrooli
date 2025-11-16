package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAPIEndpointCoverage(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	testDB, dbCleanup := setupTestDB(t)
	if testDB == nil {
		return
	}
	defer dbCleanup()

	router := setupTestRouter()

	treeID := createTestTechTree(t, testDB)
	sectorID := createTestSector(t, testDB, treeID)
	stageID := createTestStage(t, testDB, sectorID)

	// Ensure scenario mappings and milestones for downstream endpoints
	createTestScenarioMapping(t, testDB, stageID, "api-endpoint-smoke")
	createTestMilestone(t, testDB, treeID)

	// Insert dependency data for coverage
	stageBID := "00000000-0000-0000-0000-0000000000cc"
	examples := `["Example Alpha", "Example Beta"]`
	if err := execWithBackoff(t, testDB, `
        INSERT INTO progression_stages (
            id, sector_id, stage_type, stage_order, name, description,
            progress_percentage, examples, position_x, position_y, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
        ON CONFLICT (id) DO UPDATE SET name = $5
    `, stageBID, sectorID, "analytics", 2, "Analytics Stage", "Data intelligence",
		48.0, examples, 420.0, 390.0); err != nil {
		t.Fatalf("Failed to create analytics stage: %v", err)
	}

	createTestDependency(t, testDB, stageBID, stageID)
	createTestSectorConnection(t, testDB, sectorID, sectorID)

	// graph update payload to exercise persistence path
	graphBody := map[string]interface{}{
		"stages": []map[string]interface{}{
			{"id": stageID, "position_x": 505.0, "position_y": 210.0},
			{"id": stageBID, "position_x": 585.0, "position_y": 320.0},
		},
		"dependencies": []map[string]interface{}{
			{
				"dependent_stage_id":    stageBID,
				"prerequisite_stage_id": stageID,
				"dependency_type":       "requires",
				"dependency_strength":   0.9,
				"description":           "Analytics depends on foundation",
			},
		},
	}
	if w := makeHTTPRequest(t, router, "PUT", "/api/v1/tech-tree/graph", graphBody); w.Code != http.StatusOK {
		t.Fatalf("graph update failed: %d %s", w.Code, w.Body.String())
	}

	// Core tech tree endpoints
	if w := makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree", nil); w.Code != http.StatusOK {
		t.Fatalf("get tech tree failed: %d %s", w.Code, w.Body.String())
	}
	sectorsReq := makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree/sectors", nil)
	if sectorsReq.Code != http.StatusOK {
		t.Fatalf("get sectors failed: %d %s", sectorsReq.Code, sectorsReq.Body.String())
	}
	sectorsResp := assertJSONResponse(t, sectorsReq, http.StatusOK)

	// Extract sector ID from response to reuse in subsequent calls
	firstSector := sectorsResp["sectors"].([]interface{})[0].(map[string]interface{})
	sectorUUID := firstSector["id"].(string)

	if w := makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree/sectors/"+sectorUUID, nil); w.Code != http.StatusOK {
		t.Fatalf("get sector failed: %d %s", w.Code, w.Body.String())
	}
	if w := makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree/stages/"+stageID, nil); w.Code != http.StatusOK {
		t.Fatalf("get stage failed: %d %s", w.Code, w.Body.String())
	}

	// Dependency and connection routes
	if w := makeHTTPRequest(t, router, "GET", "/api/v1/dependencies", nil); w.Code != http.StatusOK {
		t.Fatalf("get dependencies failed: %d %s", w.Code, w.Body.String())
	}
	if w := makeHTTPRequest(t, router, "GET", "/api/v1/connections", nil); w.Code != http.StatusOK {
		t.Fatalf("get connections failed: %d %s", w.Code, w.Body.String())
	}

	// Scenario progress routes
	mappingBody := map[string]interface{}{
		"scenario_name":       "api-endpoint-smoke",
		"stage_id":            stageID,
		"contribution_weight": 0.9,
		"completion_status":   "in_progress",
		"priority":            1,
		"estimated_impact":    9.2,
		"notes":               "API endpoint coverage",
	}
	if w := makeHTTPRequest(t, router, "POST", "/api/v1/progress/scenarios", mappingBody); w.Code != http.StatusOK {
		t.Fatalf("create scenario mapping failed: %d %s", w.Code, w.Body.String())
	}
	if w := makeHTTPRequest(t, router, "GET", "/api/v1/progress/scenarios", nil); w.Code != http.StatusOK {
		t.Fatalf("get scenario mappings failed: %d %s", w.Code, w.Body.String())
	}

	statusBody := map[string]interface{}{
		"completion_status": "completed",
		"notes":             "Scenario delivered",
		"estimated_impact":  9.5,
	}
	if w := makeHTTPRequest(t, router, "PUT", "/api/v1/progress/scenarios/api-endpoint-smoke", statusBody); w.Code != http.StatusOK {
		t.Fatalf("update scenario status failed: %d %s", w.Code, w.Body.String())
	}

	// CRUD for sectors
	sectorCreate := map[string]interface{}{
		"name":        "Experimental Energy",
		"category":    "science",
		"description": "Edge experiments",
		"color":       "#f472b6",
		"position_x":  1280,
		"position_y":  540,
	}
	sectorResp := makeHTTPRequest(t, router, "POST", "/api/v1/tech-tree/sectors", sectorCreate)
	if sectorResp.Code != http.StatusCreated {
		t.Fatalf("create sector failed: %d %s", sectorResp.Code, sectorResp.Body.String())
	}
	sectorPayload := assertJSONResponse(t, sectorResp, http.StatusCreated)
	createdSector := sectorPayload["sector"].(map[string]interface{})
	newSectorID := createdSector["id"].(string)

	sectorUpdate := map[string]interface{}{
		"description": "Edge experiments – updated",
		"position_x":  1300,
	}
	if w := makeHTTPRequest(t, router, "PATCH", "/api/v1/tech-tree/sectors/"+newSectorID, sectorUpdate); w.Code != http.StatusOK {
		t.Fatalf("update sector failed: %d %s", w.Code, w.Body.String())
	}

	// Stage creation tied to new sector
	stageCreate := map[string]interface{}{
		"sector_id":   newSectorID,
		"stage_type":  "integration",
		"stage_order": 1,
		"name":        "Energy Coordination Mesh",
		"description": "Coordinates experimental energy nodes",
		"examples":    []string{"fusion-control", "grid-orchestrator"},
	}
	stageResp := makeHTTPRequest(t, router, "POST", "/api/v1/tech-tree/stages", stageCreate)
	if stageResp.Code != http.StatusCreated {
		t.Fatalf("create stage failed: %d %s", stageResp.Code, stageResp.Body.String())
	}
	stagePayload := assertJSONResponse(t, stageResp, http.StatusCreated)
	createdStage := stagePayload["stage"].(map[string]interface{})
	newStageID := createdStage["id"].(string)

	stageUpdate := map[string]interface{}{
		"progress_percentage": 45.0,
		"stage_type":          "digital_twin",
	}
	if w := makeHTTPRequest(t, router, "PATCH", "/api/v1/tech-tree/stages/"+newStageID, stageUpdate); w.Code != http.StatusOK {
		t.Fatalf("update stage failed: %d %s", w.Code, w.Body.String())
	}

	if w := makeHTTPRequest(t, router, "DELETE", "/api/v1/tech-tree/stages/"+newStageID, nil); w.Code != http.StatusNoContent {
		t.Fatalf("delete stage failed: %d %s", w.Code, w.Body.String())
	}
	if w := makeHTTPRequest(t, router, "DELETE", "/api/v1/tech-tree/sectors/"+newSectorID, nil); w.Code != http.StatusNoContent {
		t.Fatalf("delete sector failed: %d %s", w.Code, w.Body.String())
	}

	// AI ideas + DOT export
	ideasBody := map[string]interface{}{
		"sector_id": sectorID,
		"count":     2,
		"prompt":    "Focus on interoperability",
	}
	if w := makeHTTPRequest(t, router, "POST", "/api/v1/tech-tree/ai/stage-ideas", ideasBody); w.Code != http.StatusOK {
		t.Fatalf("ai stage ideas failed: %d %s", w.Code, w.Body.String())
	}
	if w := makeHTTPRequest(t, router, "GET", "/api/v1/tech-tree/graph/dot", nil); w.Code != http.StatusOK {
		t.Fatalf("graph export failed: %d %s", w.Code, w.Body.String())
	}

	// Strategic analysis endpoints
	analyzeBody := map[string]interface{}{
		"current_resources": 6,
		"time_horizon":      12,
		"priority_sectors":  []string{"software", "manufacturing"},
	}
	if w := makeHTTPRequest(t, router, "POST", "/api/v1/tech-tree/analyze", analyzeBody); w.Code != http.StatusOK {
		t.Fatalf("analyze endpoint failed: %d %s", w.Code, w.Body.String())
	}
	if w := makeHTTPRequest(t, router, "GET", "/api/v1/milestones", nil); w.Code != http.StatusOK {
		t.Fatalf("get milestones failed: %d %s", w.Code, w.Body.String())
	}

	milestoneCreate := map[string]interface{}{
		"name":                   "API Created Milestone",
		"description":            "End-to-end milestone coverage",
		"milestone_type":         "sector_complete",
		"completion_percentage":  35.0,
		"confidence_level":       0.7,
		"business_value_estimate": 123456,
		"estimated_completion_date": "2030-01-01",
		"target_sector_ids":      []string{sectorID},
		"target_stage_ids":       []string{stageID},
	}
	createMilestoneResp := makeHTTPRequest(t, router, "POST", "/api/v1/milestones?tree_id="+treeID, milestoneCreate)
	if createMilestoneResp.Code != http.StatusCreated {
		t.Fatalf("create milestone failed: %d %s", createMilestoneResp.Code, createMilestoneResp.Body.String())
	}
	createPayload := assertJSONResponse(t, createMilestoneResp, http.StatusCreated)
	createdMilestone := createPayload["milestone"].(map[string]interface{})
	newMilestoneID := createdMilestone["id"].(string)

	milestoneUpdate := map[string]interface{}{
		"name":                   "API Updated Milestone",
		"description":            "Updated via API",
		"milestone_type":         "cross_sector_integration",
		"completion_percentage":  55.0,
		"confidence_level":       0.65,
		"business_value_estimate": 789000,
		"estimated_completion_date": "2031-06-15",
		"target_sector_ids":      []string{sectorID},
		"target_stage_ids":       []string{stageID},
	}
	if w := makeHTTPRequest(t, router, "PATCH", "/api/v1/milestones/"+newMilestoneID+"?tree_id="+treeID, milestoneUpdate); w.Code != http.StatusOK {
		t.Fatalf("update milestone failed: %d %s", w.Code, w.Body.String())
	}
	if w := makeHTTPRequest(t, router, "DELETE", "/api/v1/milestones/"+newMilestoneID+"?tree_id="+treeID, nil); w.Code != http.StatusNoContent {
		t.Fatalf("delete milestone failed: %d %s", w.Code, w.Body.String())
	}
	if w := makeHTTPRequest(t, router, "GET", "/api/v1/recommendations", nil); w.Code != http.StatusOK {
		t.Fatalf("get recommendations failed: %d %s", w.Code, w.Body.String())
	}

	// Direct handler invocations to ensure coverage
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/tech-tree", nil)
	getTechTree(ctx)
	if w.Code != http.StatusOK {
		t.Fatalf("direct getTechTree failed: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/tech-tree/sectors", nil)
	getSectors(ctx)
	if w.Code != http.StatusOK {
		t.Fatalf("direct getSectors failed: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: sectorUUID}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/tech-tree/sectors/"+sectorUUID, nil)
	getSector(ctx)
	if w.Code != http.StatusOK {
		t.Fatalf("direct getSector failed: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(w)
	ctx.Params = gin.Params{{Key: "id", Value: stageID}}
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/tech-tree/stages/"+stageID, nil)
	getStage(ctx)
	if w.Code != http.StatusOK {
		t.Fatalf("direct getStage failed: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/dependencies", nil)
	getDependencies(ctx)
	if w.Code != http.StatusOK {
		t.Fatalf("direct getDependencies failed: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/connections", nil)
	getCrossSectorConnections(ctx)
	if w.Code != http.StatusOK {
		t.Fatalf("direct getCrossSectorConnections failed: %d %s", w.Code, w.Body.String())
	}

	defaultTree, errTree := fetchDefaultTechTree(context.Background())
	if errTree != nil {
		t.Fatalf("unable to resolve default tech tree: %v", errTree)
	}

	if sectorsList, err := fetchSectorsWithStages(context.Background(), defaultTree.ID); err != nil {
		t.Fatalf("fetchSectorsWithStages error: %v", err)
	} else if len(sectorsList) == 0 {
		t.Fatal("expected seeded sectors from fetchSectorsWithStages")
	}

	if depsList, err := fetchDependencies(context.Background(), defaultTree.ID); err != nil {
		t.Fatalf("fetchDependencies error: %v", err)
	} else if len(depsList) == 0 {
		t.Fatal("expected seeded dependencies from fetchDependencies")
	}

	health := buildHealthResponse(context.Background())
	if _, ok := health["status"].(string); !ok {
		t.Fatal("expected health response to include status")
	}

	if version := getAppVersion(); version == "" {
		t.Fatal("expected getAppVersion to return non-empty value")
	}
}
