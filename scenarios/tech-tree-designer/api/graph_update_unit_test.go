package main

import (
	"net/http"
	"testing"
)

func TestUpdateGraphEndpoint(t *testing.T) {
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
	stageAID := createTestStage(t, testDB, sectorID)

	stageBID := "00000000-0000-0000-0000-0000000000bb"
	examples := `["Bridge 1", "Bridge 2"]`
	if err := execWithBackoff(t, testDB, `
        INSERT INTO progression_stages (
            id, sector_id, stage_type, stage_order, name, description,
            progress_percentage, examples, position_x, position_y, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
        ON CONFLICT (id) DO UPDATE SET name = $5
    `, stageBID, sectorID, "integration", 3, "Integration Stage", "Bridge systems",
		62.0, examples, 340.0, 410.0); err != nil {
		t.Fatalf("Failed to create secondary stage: %v", err)
	}

	depID := createTestDependency(t, testDB, stageBID, stageAID)

	body := map[string]interface{}{
		"stages": []map[string]interface{}{
			{"id": stageAID, "position_x": 520.25, "position_y": 265.5},
			{"id": stageBID, "position_x": 640.0, "position_y": 330.75},
		},
		"dependencies": []map[string]interface{}{
			{
				"id":                    depID,
				"dependent_stage_id":    stageBID,
				"prerequisite_stage_id": stageAID,
				"dependency_type":       "requires",
				"dependency_strength":   0.88,
				"description":           "Updated baseline link",
			},
			{
				"dependent_stage_id":    stageAID,
				"prerequisite_stage_id": stageBID,
				"dependency_type":       "optional",
				"dependency_strength":   0.55,
				"description":           "Reverse enabling loop",
			},
		},
	}

    w := makeHTTPRequest(t, router, "PUT", "/api/v1/tech-tree/graph", body)
    if w.Code != http.StatusOK {
        t.Fatalf("expected 200 from graph update, got %d: %s", w.Code, w.Body.String())
    }
    response := assertJSONResponse(t, w, http.StatusOK)

	if message, ok := response["message"].(string); !ok || message != "Graph updated successfully" {
		t.Fatalf("Unexpected message: %v", response["message"])
	}

    if _, ok := response["dependencies"]; !ok {
        t.Fatal("Expected dependencies in response payload")
	}

	row := testDB.QueryRow(`SELECT position_x, position_y FROM progression_stages WHERE id = $1`, stageBID)
	var posX, posY float64
	if err := row.Scan(&posX, &posY); err != nil {
		t.Fatalf("Failed to verify stage position: %v", err)
	}

	if posX != 640.0 || posY != 330.75 {
		t.Fatalf("Stage coordinates not updated (got %f,%f)", posX, posY)
	}

	var reverseID string
	if err := testDB.QueryRow(
		`SELECT id FROM stage_dependencies WHERE dependent_stage_id = $1 AND prerequisite_stage_id = $2`,
		stageAID,
		stageBID,
	).Scan(&reverseID); err != nil {
		t.Fatalf("Expected reverse dependency to persist: %v", err)
	}

	if reverseID == "" {
		t.Fatal("Reverse dependency missing identifier")
	}
}
