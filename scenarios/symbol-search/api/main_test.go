// +build testing

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, _, cleanupAPI := setupTestRouter(t)
	defer cleanupAPI()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w := makeHTTPRequest(router, req)
		response := assertJSONResponse(t, w, http.StatusOK, "status", "database")

		if status, ok := response["status"].(string); !ok || status != "healthy" {
			t.Errorf("Expected healthy status, got: %v", response["status"])
		}
	})

	t.Run("DatabaseConnection", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/health",
		}

		w := makeHTTPRequest(router, req)
		response := assertJSONResponse(t, w, http.StatusOK, "database", "characters_loaded")

		if db, ok := response["database"].(string); !ok || db != "connected" {
			t.Errorf("Expected database connected, got: %v", response["database"])
		}
	})
}

// TestSearchCharacters tests the character search endpoint
func TestSearchCharacters(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, api, cleanupAPI := setupTestRouter(t)
	defer cleanupAPI()

	t.Run("Success_BasicSearch", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/search",
			QueryParams: map[string]string{
				"q": "LATIN",
			},
		}

		w := makeHTTPRequest(router, req)
		response := assertSearchResponse(t, w, http.StatusOK)

		if response.Characters == nil {
			t.Error("Expected characters array, got nil")
		}

		if response.Total < 0 {
			t.Errorf("Expected non-negative total, got: %d", response.Total)
		}

		if response.QueryTimeMs <= 0 {
			t.Errorf("Expected positive query time, got: %f", response.QueryTimeMs)
		}

		if response.FiltersApplied == nil {
			t.Error("Expected filters_applied map, got nil")
		}
	})

	t.Run("Success_WithCategory", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/search",
			QueryParams: map[string]string{
				"q":        "LETTER",
				"category": "Lu",
			},
		}

		w := makeHTTPRequest(router, req)
		response := assertSearchResponse(t, w, http.StatusOK)

		// Verify category filter was applied
		if response.FiltersApplied["category"] != "Lu" {
			t.Errorf("Expected category filter 'Lu', got: %v", response.FiltersApplied["category"])
		}

		// Verify all returned characters match category
		for _, char := range response.Characters {
			if char.Category != "Lu" {
				t.Errorf("Expected category 'Lu' for all results, got: %s for %s",
					char.Category, char.Name)
			}
		}
	})

	t.Run("Success_WithBlock", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/search",
			QueryParams: map[string]string{
				"block": "Basic Latin",
			},
		}

		w := makeHTTPRequest(router, req)
		response := assertSearchResponse(t, w, http.StatusOK)

		// Verify block filter was applied
		if response.FiltersApplied["block"] != "Basic Latin" {
			t.Errorf("Expected block filter 'Basic Latin', got: %v", response.FiltersApplied["block"])
		}

		// Verify all returned characters match block
		for _, char := range response.Characters {
			if char.Block != "Basic Latin" {
				t.Errorf("Expected block 'Basic Latin' for all results, got: %s for %s",
					char.Block, char.Name)
			}
		}
	})

	t.Run("Success_WithPagination", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/search",
			QueryParams: map[string]string{
				"q":      "LATIN",
				"limit":  "10",
				"offset": "5",
			},
		}

		w := makeHTTPRequest(router, req)
		response := assertSearchResponse(t, w, http.StatusOK)

		if len(response.Characters) > 10 {
			t.Errorf("Expected at most 10 characters, got: %d", len(response.Characters))
		}

		// Verify pagination params in filters
		if limit, ok := response.FiltersApplied["limit"].(int); !ok || limit != 10 {
			t.Errorf("Expected limit 10, got: %v", response.FiltersApplied["limit"])
		}

		if offset, ok := response.FiltersApplied["offset"].(int); !ok || offset != 5 {
			t.Errorf("Expected offset 5, got: %v", response.FiltersApplied["offset"])
		}
	})

	t.Run("Success_EmptyResults", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/search",
			QueryParams: map[string]string{
				"q": "NONEXISTENT_CHARACTER_XYZABC123",
			},
		}

		w := makeHTTPRequest(router, req)
		response := assertSearchResponse(t, w, http.StatusOK)

		if response.Total != 0 {
			t.Errorf("Expected 0 results, got: %d", response.Total)
		}

		if len(response.Characters) != 0 {
			t.Errorf("Expected empty characters array, got: %d items", len(response.Characters))
		}
	})

	t.Run("EdgeCase_DefaultPagination", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/search",
			QueryParams: map[string]string{
				"q": "LATIN",
			},
		}

		w := makeHTTPRequest(router, req)
		response := assertSearchResponse(t, w, http.StatusOK)

		// Should default to limit=100, offset=0
		if limit, ok := response.FiltersApplied["limit"].(int); !ok || limit != 100 {
			t.Errorf("Expected default limit 100, got: %v", response.FiltersApplied["limit"])
		}

		if offset, ok := response.FiltersApplied["offset"].(int); !ok || offset != 0 {
			t.Errorf("Expected default offset 0, got: %v", response.FiltersApplied["offset"])
		}
	})

	// Run edge case tests
	edgeCases := CommonEdgeCases()
	for _, edgeCase := range edgeCases {
		t.Run("EdgeCase_"+edgeCase.Name, func(t *testing.T) {
			edgeCase.Test(t, router, api)
		})
	}
}

// TestGetCharacterDetail tests the character detail endpoint
func TestGetCharacterDetail(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, api, cleanupAPI := setupTestRouter(t)
	defer cleanupAPI()

	// Seed a test character for reliable testing
	seedTestCharacter(t, api.db, "U+1F600", 128512, "GRINNING FACE")
	defer cleanupTestCharacter(t, api.db, "U+1F600")

	t.Run("Success_UnicodeFormat", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/character/U+1F600",
		}

		w := makeHTTPRequest(router, req)
		response := assertCharacterResponse(t, w, http.StatusOK)

		if response.Character.Codepoint != "U+1F600" {
			t.Errorf("Expected codepoint U+1F600, got: %s", response.Character.Codepoint)
		}

		if response.Character.Name != "GRINNING FACE" {
			t.Errorf("Expected name 'GRINNING FACE', got: %s", response.Character.Name)
		}

		if response.Character.Decimal != 128512 {
			t.Errorf("Expected decimal 128512, got: %d", response.Character.Decimal)
		}
	})

	t.Run("Success_DecimalFormat", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/character/128512",
		}

		w := makeHTTPRequest(router, req)
		response := assertCharacterResponse(t, w, http.StatusOK)

		if response.Character.Decimal != 128512 {
			t.Errorf("Expected decimal 128512, got: %d", response.Character.Decimal)
		}
	})

	t.Run("Success_WithRelatedCharacters", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/character/U+1F600",
		}

		w := makeHTTPRequest(router, req)
		response := assertCharacterResponse(t, w, http.StatusOK)

		// Related characters should be from the same block
		for _, related := range response.RelatedCharacters {
			if related.Block != response.Character.Block {
				t.Errorf("Related character %s has different block: %s vs %s",
					related.Name, related.Block, response.Character.Block)
			}

			// Should not include the character itself
			if related.Codepoint == response.Character.Codepoint {
				t.Errorf("Related characters should not include the character itself")
			}
		}

		// Should have at most 5 related characters
		if len(response.RelatedCharacters) > 5 {
			t.Errorf("Expected at most 5 related characters, got: %d", len(response.RelatedCharacters))
		}
	})

	t.Run("Success_WithUsageExamples", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/character/128512",
		}

		w := makeHTTPRequest(router, req)
		response := assertCharacterResponse(t, w, http.StatusOK)

		if len(response.UsageExamples) == 0 {
			t.Error("Expected usage examples, got none")
		}

		// Should contain HTML and Unicode examples at minimum
		hasHTML := false
		hasUnicode := false
		for _, example := range response.UsageExamples {
			if len(example) > 0 {
				if example[:4] == "HTML" {
					hasHTML = true
				}
				if example[:7] == "Unicode" {
					hasUnicode = true
				}
			}
		}

		if !hasHTML {
			t.Error("Expected HTML usage example")
		}
		if !hasUnicode {
			t.Error("Expected Unicode usage example")
		}
	})

	t.Run("Error_NotFound", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/character/U+999999",
		}

		w := makeHTTPRequest(router, req)
		assertErrorResponse(t, w, http.StatusNotFound)
	})

	t.Run("Error_InvalidFormat", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/character/INVALID",
		}

		w := makeHTTPRequest(router, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	// Test error scenarios using builder pattern
	scenarios := NewTestScenarioBuilder().
		AddInvalidCodepoint("/api/character/invalid-format").
		AddNonExistentCharacter("/api/character/U+FFFFFF").
		Build()

	suite := &HandlerTestSuite{
		Name:      "GetCharacterDetail",
		Router:    router,
		API:       api,
		Cleanup:   cleanupAPI,
		Scenarios: scenarios,
	}

	suite.RunErrorTests(t)
}

// TestGetCategories tests the categories list endpoint
func TestGetCategories(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, _, cleanupAPI := setupTestRouter(t)
	defer cleanupAPI()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/categories",
		}

		w := makeHTTPRequest(router, req)
		response := assertJSONResponse(t, w, http.StatusOK, "categories")

		categories, ok := response["categories"].([]interface{})
		if !ok {
			t.Fatalf("Expected categories array, got: %T", response["categories"])
		}

		if len(categories) == 0 {
			t.Error("Expected at least one category")
		}

		// Verify each category has required fields
		for i, cat := range categories {
			catMap, ok := cat.(map[string]interface{})
			if !ok {
				t.Errorf("Category %d is not a map", i)
				continue
			}

			requiredFields := []string{"code", "name", "description", "character_count"}
			for _, field := range requiredFields {
				if _, exists := catMap[field]; !exists {
					t.Errorf("Category %d missing required field: %s", i, field)
				}
			}
		}
	})

	t.Run("Success_WithCharacterCounts", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/categories",
		}

		w := makeHTTPRequest(router, req)
		response := assertJSONResponse(t, w, http.StatusOK, "categories")

		categories, ok := response["categories"].([]interface{})
		if !ok {
			t.Fatalf("Expected categories array, got: %T", response["categories"])
		}

		// At least some categories should have character counts
		hasNonZeroCount := false
		for _, cat := range categories {
			catMap := cat.(map[string]interface{})
			if count, ok := catMap["character_count"].(float64); ok && count > 0 {
				hasNonZeroCount = true
				break
			}
		}

		if !hasNonZeroCount {
			t.Error("Expected at least one category with non-zero character count")
		}
	})
}

// TestGetBlocks tests the blocks list endpoint
func TestGetBlocks(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, _, cleanupAPI := setupTestRouter(t)
	defer cleanupAPI()

	t.Run("Success", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/blocks",
		}

		w := makeHTTPRequest(router, req)
		response := assertJSONResponse(t, w, http.StatusOK, "blocks")

		blocks, ok := response["blocks"].([]interface{})
		if !ok {
			t.Fatalf("Expected blocks array, got: %T", response["blocks"])
		}

		if len(blocks) == 0 {
			t.Error("Expected at least one block")
		}

		// Verify each block has required fields
		for i, blk := range blocks {
			blkMap, ok := blk.(map[string]interface{})
			if !ok {
				t.Errorf("Block %d is not a map", i)
				continue
			}

			requiredFields := []string{"id", "name", "start_codepoint", "end_codepoint", "description", "character_count"}
			for _, field := range requiredFields {
				if _, exists := blkMap[field]; !exists {
					t.Errorf("Block %d missing required field: %s", i, field)
				}
			}
		}
	})

	t.Run("Success_OrderedByCodepoint", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/blocks",
		}

		w := makeHTTPRequest(router, req)
		response := assertJSONResponse(t, w, http.StatusOK, "blocks")

		blocks, ok := response["blocks"].([]interface{})
		if !ok || len(blocks) < 2 {
			t.Skip("Not enough blocks to verify ordering")
		}

		// Verify blocks are ordered by start_codepoint
		var prevStart float64 = -1
		for i, blk := range blocks {
			blkMap := blk.(map[string]interface{})
			start, ok := blkMap["start_codepoint"].(float64)
			if !ok {
				t.Errorf("Block %d has invalid start_codepoint", i)
				continue
			}

			if prevStart >= 0 && start < prevStart {
				t.Errorf("Blocks not ordered: block %d start (%f) < previous start (%f)",
					i, start, prevStart)
			}
			prevStart = start
		}
	})
}

// TestGetBulkRange tests the bulk range endpoint
func TestGetBulkRange(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	router, api, cleanupAPI := setupTestRouter(t)
	defer cleanupAPI()

	t.Run("Success_SingleRange", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/bulk/range",
			Body: BulkRangeRequest{
				Ranges: []struct {
					Start  string `json:"start"`
					End    string `json:"end"`
					Format string `json:"format,omitempty"`
				}{
					{Start: "U+0020", End: "U+007E"}, // Basic Latin printable chars
				},
			},
		}

		w := makeHTTPRequest(router, req)

		var response BulkRangeResponse
		assertJSONResponse(t, w, http.StatusOK, "characters", "total_characters", "ranges_processed")

		// Additional parsing to check values
		w = makeHTTPRequest(router, req)
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse bulk range response: %v", err)
		}

		if response.RangesProcessed != 1 {
			t.Errorf("Expected 1 range processed, got: %d", response.RangesProcessed)
		}

		if response.TotalCharacters == 0 {
			t.Error("Expected characters in range U+0020 to U+007E")
		}

		// Verify all characters are within range
		for _, char := range response.Characters {
			if char.Decimal < 32 || char.Decimal > 126 {
				t.Errorf("Character %s (decimal %d) outside requested range",
					char.Name, char.Decimal)
			}
		}
	})

	t.Run("Success_MultipleRanges", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/bulk/range",
			Body: BulkRangeRequest{
				Ranges: []struct {
					Start  string `json:"start"`
					End    string `json:"end"`
					Format string `json:"format,omitempty"`
				}{
					{Start: "U+0041", End: "U+005A"}, // A-Z
					{Start: "U+0061", End: "U+007A"}, // a-z
				},
			},
		}

		w := makeHTTPRequest(router, req)

		var response BulkRangeResponse
		assertJSONResponse(t, w, http.StatusOK)

		w = makeHTTPRequest(router, req)
		json.Unmarshal(w.Body.Bytes(), &response)

		if response.RangesProcessed != 2 {
			t.Errorf("Expected 2 ranges processed, got: %d", response.RangesProcessed)
		}

		// Should have 26 + 26 = 52 characters
		expectedCount := 52
		if response.TotalCharacters != expectedCount {
			t.Errorf("Expected %d characters, got: %d", expectedCount, response.TotalCharacters)
		}
	})

	t.Run("Success_DecimalFormat", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/bulk/range",
			Body: BulkRangeRequest{
				Ranges: []struct {
					Start  string `json:"start"`
					End    string `json:"end"`
					Format string `json:"format,omitempty"`
				}{
					{Start: "65", End: "90"}, // A-Z in decimal
				},
			},
		}

		w := makeHTTPRequest(router, req)
		assertJSONResponse(t, w, http.StatusOK)
	})

	t.Run("Error_EmptyRanges", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/bulk/range",
			Body: BulkRangeRequest{
				Ranges: []struct {
					Start  string `json:"start"`
					End    string `json:"end"`
					Format string `json:"format,omitempty"`
				}{},
			},
		}

		w := makeHTTPRequest(router, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error_TooManyRanges", func(t *testing.T) {
		ranges := make([]struct {
			Start  string `json:"start"`
			End    string `json:"end"`
			Format string `json:"format,omitempty"`
		}, 11) // More than limit of 10

		for i := 0; i < 11; i++ {
			ranges[i] = struct {
				Start  string `json:"start"`
				End    string `json:"end"`
				Format string `json:"format,omitempty"`
			}{
				Start: fmt.Sprintf("U+%04X", i*10),
				End:   fmt.Sprintf("U+%04X", i*10+5),
			}
		}

		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/bulk/range",
			Body:   BulkRangeRequest{Ranges: ranges},
		}

		w := makeHTTPRequest(router, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	t.Run("Error_InvalidRange", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "POST",
			Path:   "/api/bulk/range",
			Body: BulkRangeRequest{
				Ranges: []struct {
					Start  string `json:"start"`
					End    string `json:"end"`
					Format string `json:"format,omitempty"`
				}{
					{Start: "U+007F", End: "U+0020"}, // start > end
				},
			},
		}

		w := makeHTTPRequest(router, req)
		assertErrorResponse(t, w, http.StatusBadRequest)
	})

	// Run comprehensive error tests
	scenarios := NewTestScenarioBuilder().
		AddInvalidRange("/api/bulk/range").
		AddTooManyRanges("/api/bulk/range").
		AddEmptyRanges("/api/bulk/range").
		AddInvalidJSON("/api/bulk/range").
		Build()

	suite := &HandlerTestSuite{
		Name:      "GetBulkRange",
		Router:    router,
		API:       api,
		Cleanup:   cleanupAPI,
		Scenarios: scenarios,
	}

	suite.RunErrorTests(t)
}

// TestHelperFunctions tests utility functions
func TestParseCodepoint(t *testing.T) {
	t.Run("Success_UnicodeFormat", func(t *testing.T) {
		result, err := parseCodepoint("U+1F600")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if result != 128512 {
			t.Errorf("Expected 128512, got: %d", result)
		}
	})

	t.Run("Success_DecimalFormat", func(t *testing.T) {
		result, err := parseCodepoint("65")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if result != 65 {
			t.Errorf("Expected 65, got: %d", result)
		}
	})

	t.Run("Success_LowercaseUnicode", func(t *testing.T) {
		result, err := parseCodepoint("u+41")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if result != 65 {
			t.Errorf("Expected 65, got: %d", result)
		}
	})

	t.Run("Error_InvalidFormat", func(t *testing.T) {
		_, err := parseCodepoint("INVALID")
		if err == nil {
			t.Error("Expected error for invalid format")
		}
	})
}

func TestParseCodepointRange(t *testing.T) {
	t.Run("Success_ValidRange", func(t *testing.T) {
		start, end, err := parseCodepointRange("U+0041", "U+005A")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if start != 65 || end != 90 {
			t.Errorf("Expected 65-90, got: %d-%d", start, end)
		}
	})

	t.Run("Success_DecimalRange", func(t *testing.T) {
		start, end, err := parseCodepointRange("65", "90")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		if start != 65 || end != 90 {
			t.Errorf("Expected 65-90, got: %d-%d", start, end)
		}
	})

	t.Run("Error_StartGreaterThanEnd", func(t *testing.T) {
		_, _, err := parseCodepointRange("U+007F", "U+0020")
		if err == nil {
			t.Error("Expected error when start > end")
		}
	})

	t.Run("Error_InvalidStart", func(t *testing.T) {
		_, _, err := parseCodepointRange("INVALID", "U+007F")
		if err == nil {
			t.Error("Expected error for invalid start")
		}
	})

	t.Run("Error_InvalidEnd", func(t *testing.T) {
		_, _, err := parseCodepointRange("U+0020", "INVALID")
		if err == nil {
			t.Error("Expected error for invalid end")
		}
	})
}

func TestGenerateUsageExamples(t *testing.T) {
	t.Run("Success_BasicCharacter", func(t *testing.T) {
		char := Character{
			Decimal:   65,
			Codepoint: "U+0041",
			Name:      "LATIN CAPITAL LETTER A",
		}

		examples := generateUsageExamples(char)

		if len(examples) < 2 {
			t.Errorf("Expected at least 2 examples, got: %d", len(examples))
		}

		// Should include HTML and Unicode examples
		hasHTML := false
		hasUnicode := false
		for _, ex := range examples {
			if len(ex) >= 4 && ex[:4] == "HTML" {
				hasHTML = true
			}
			if len(ex) >= 7 && ex[:7] == "Unicode" {
				hasUnicode = true
			}
		}

		if !hasHTML {
			t.Error("Missing HTML example")
		}
		if !hasUnicode {
			t.Error("Missing Unicode example")
		}
	})

	t.Run("Success_WithOptionalFields", func(t *testing.T) {
		htmlEntity := "&nbsp;"
		cssContent := "\\00A0"

		char := Character{
			Decimal:    160,
			Codepoint:  "U+00A0",
			Name:       "NO-BREAK SPACE",
			HTMLEntity: &htmlEntity,
			CSSContent: &cssContent,
		}

		examples := generateUsageExamples(char)

		if len(examples) < 4 {
			t.Errorf("Expected at least 4 examples with optional fields, got: %d", len(examples))
		}
	})
}

// TestPerformance tests basic performance requirements
func TestPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	cleanup := setupTestLogger()
	defer cleanup()

	router, _, cleanupAPI := setupTestRouter(t)
	defer cleanupAPI()

	t.Run("SearchResponseTime", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/search",
			QueryParams: map[string]string{
				"q": "LATIN",
			},
		}

		// Warmup
		for i := 0; i < 5; i++ {
			makeHTTPRequest(router, req)
		}

		// Measure
		start := time.Now()
		iterations := 100
		for i := 0; i < iterations; i++ {
			w := makeHTTPRequest(router, req)
			if w.Code != http.StatusOK {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}
		elapsed := time.Since(start)

		avgTime := elapsed / time.Duration(iterations)
		t.Logf("Average search time: %v", avgTime)

		// Target: < 50ms for 95% of queries (we're testing average here)
		if avgTime > 50*time.Millisecond {
			t.Errorf("Average search time %v exceeds 50ms target", avgTime)
		}
	})

	t.Run("CharacterDetailResponseTime", func(t *testing.T) {
		req := HTTPTestRequest{
			Method: "GET",
			Path:   "/api/character/U+0041",
		}

		// Warmup
		for i := 0; i < 5; i++ {
			makeHTTPRequest(router, req)
		}

		// Measure
		start := time.Now()
		iterations := 100
		for i := 0; i < iterations; i++ {
			w := makeHTTPRequest(router, req)
			if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
				t.Errorf("Request %d failed with status %d", i, w.Code)
			}
		}
		elapsed := time.Since(start)

		avgTime := elapsed / time.Duration(iterations)
		t.Logf("Average character detail time: %v", avgTime)

		// Target: < 25ms
		if avgTime > 25*time.Millisecond {
			t.Errorf("Average character detail time %v exceeds 25ms target", avgTime)
		}
	})
}
