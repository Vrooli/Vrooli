package main

import (
	"testing"
)

// TestListPresets tests the preset listing functionality
func TestListPresets(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	server := setupTestServer()

	t.Run("ListPresetsSuccess", func(t *testing.T) {
		resp, body, err := makeHTTPRequest(server.app, "GET", "/api/v1/presets", nil, nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		result := assertJSONResponse(t, resp, body, 200)

		presets, ok := result["presets"].([]interface{})
		if !ok {
			t.Fatal("Expected presets array in response")
		}

		if len(presets) == 0 {
			t.Error("Expected at least one preset")
		}

		count, ok := result["count"].(float64)
		if !ok {
			t.Error("Expected count field in response")
		}

		if int(count) != len(presets) {
			t.Errorf("Count mismatch: count=%d, presets length=%d", int(count), len(presets))
		}

		// Verify preset structure
		for i, preset := range presets {
			presetMap, ok := preset.(map[string]interface{})
			if !ok {
				t.Errorf("Preset %d is not a map", i)
				continue
			}

			requiredFields := []string{"name", "description", "target_use", "operations"}
			for _, field := range requiredFields {
				if _, exists := presetMap[field]; !exists {
					t.Errorf("Preset %d missing required field: %s", i, field)
				}
			}
		}
	})

	t.Run("VerifyBuiltinPresets", func(t *testing.T) {
		resp, body, err := makeHTTPRequest(server.app, "GET", "/api/v1/presets", nil, nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		result := assertJSONResponse(t, resp, body, 200)

		presets, ok := result["presets"].([]interface{})
		if !ok {
			t.Fatal("Expected presets array")
		}

		// Check that we have expected presets
		expectedPresets := []string{"web-optimized", "email-safe", "aggressive", "high-quality", "social-media"}
		foundPresets := make(map[string]bool)

		for _, preset := range presets {
			presetMap := preset.(map[string]interface{})
			name := presetMap["name"].(string)
			foundPresets[name] = true
		}

		for _, expected := range expectedPresets {
			if !foundPresets[expected] {
				t.Errorf("Missing expected preset: %s", expected)
			}
		}
	})
}

// TestGetPreset tests retrieving individual presets
func TestGetPreset(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	server := setupTestServer()

	t.Run("GetWebOptimizedPreset", func(t *testing.T) {
		resp, body, err := makeHTTPRequest(server.app, "GET", "/api/v1/presets/web-optimized", nil, nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		result := assertJSONResponse(t, resp, body, 200)

		if name, ok := result["name"].(string); !ok || name != "web-optimized" {
			t.Errorf("Expected preset name 'web-optimized', got: %v", result["name"])
		}

		if _, ok := result["description"]; !ok {
			t.Error("Preset missing description")
		}

		if _, ok := result["target_use"]; !ok {
			t.Error("Preset missing target_use")
		}

		operations, ok := result["operations"].([]interface{})
		if !ok {
			t.Fatal("Expected operations array")
		}

		if len(operations) == 0 {
			t.Error("Expected at least one operation in web-optimized preset")
		}
	})

	t.Run("GetEmailSafePreset", func(t *testing.T) {
		resp, body, err := makeHTTPRequest(server.app, "GET", "/api/v1/presets/email-safe", nil, nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		result := assertJSONResponse(t, resp, body, 200)

		if name, ok := result["name"].(string); !ok || name != "email-safe" {
			t.Errorf("Expected preset name 'email-safe', got: %v", result["name"])
		}
	})

	t.Run("GetAggressivePreset", func(t *testing.T) {
		resp, body, err := makeHTTPRequest(server.app, "GET", "/api/v1/presets/aggressive", nil, nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		result := assertJSONResponse(t, resp, body, 200)

		if name, ok := result["name"].(string); !ok || name != "aggressive" {
			t.Errorf("Expected preset name 'aggressive', got: %v", result["name"])
		}
	})

	t.Run("GetHighQualityPreset", func(t *testing.T) {
		resp, body, err := makeHTTPRequest(server.app, "GET", "/api/v1/presets/high-quality", nil, nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		result := assertJSONResponse(t, resp, body, 200)

		if name, ok := result["name"].(string); !ok || name != "high-quality" {
			t.Errorf("Expected preset name 'high-quality', got: %v", result["name"])
		}
	})

	t.Run("GetSocialMediaPreset", func(t *testing.T) {
		resp, body, err := makeHTTPRequest(server.app, "GET", "/api/v1/presets/social-media", nil, nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		result := assertJSONResponse(t, resp, body, 200)

		if name, ok := result["name"].(string); !ok || name != "social-media" {
			t.Errorf("Expected preset name 'social-media', got: %v", result["name"])
		}
	})

	t.Run("GetNonExistentPreset", func(t *testing.T) {
		resp, body, err := makeHTTPRequest(server.app, "GET", "/api/v1/presets/nonexistent-preset", nil, nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		result := assertJSONResponse(t, resp, body, 404)

		if _, ok := result["error"]; !ok {
			t.Error("Expected error field in response")
		}

		if _, ok := result["available_presets"]; !ok {
			t.Error("Expected available_presets field in error response")
		}
	})
}

// TestApplyPreset tests applying presets
func TestApplyPreset(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	server := setupTestServer()

	t.Run("ApplyWebOptimizedPreset", func(t *testing.T) {
		resp, body, err := makeHTTPRequest(server.app, "POST", "/api/v1/image/preset/web-optimized", nil, nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		result := assertJSONResponse(t, resp, body, 200)

		if _, ok := result["preset"]; !ok {
			t.Error("Response missing preset field")
		}

		if _, ok := result["operations"]; !ok {
			t.Error("Response missing operations field")
		}

		if batchReady, ok := result["batch_ready"].(bool); !ok || !batchReady {
			t.Error("Expected batch_ready to be true")
		}

		if _, ok := result["message"]; !ok {
			t.Error("Response missing message field")
		}
	})

	t.Run("ApplyEmailSafePreset", func(t *testing.T) {
		resp, body, err := makeHTTPRequest(server.app, "POST", "/api/v1/image/preset/email-safe", nil, nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		result := assertJSONResponse(t, resp, body, 200)

		preset, ok := result["preset"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected preset object")
		}

		if name, ok := preset["name"].(string); !ok || name != "email-safe" {
			t.Errorf("Expected preset name 'email-safe', got: %v", preset["name"])
		}
	})

	t.Run("ApplyNonExistentPreset", func(t *testing.T) {
		resp, body, err := makeHTTPRequest(server.app, "POST", "/api/v1/image/preset/invalid-preset", nil, nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		result := assertJSONResponse(t, resp, body, 404)

		if _, ok := result["error"]; !ok {
			t.Error("Expected error field in response")
		}

		if _, ok := result["available_presets"]; !ok {
			t.Error("Expected available_presets in error response")
		}
	})
}

// TestPresetOperations tests the structure of preset operations
func TestPresetOperations(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	t.Run("WebOptimizedOperations", func(t *testing.T) {
		preset, ok := GetPreset("web-optimized")
		if !ok {
			t.Fatal("Failed to get web-optimized preset")
		}

		if len(preset.Operations) == 0 {
			t.Error("Web-optimized preset should have operations")
		}

		// Verify operation types
		hasResize := false
		hasCompress := false
		hasMetadata := false

		for _, op := range preset.Operations {
			switch op.Type {
			case "resize":
				hasResize = true
				// Verify resize options
				if _, ok := op.Options["max_width"]; !ok {
					t.Error("Resize operation missing max_width")
				}
				if _, ok := op.Options["max_height"]; !ok {
					t.Error("Resize operation missing max_height")
				}
			case "compress":
				hasCompress = true
				// Verify compress options
				if _, ok := op.Options["quality"]; !ok {
					t.Error("Compress operation missing quality")
				}
			case "metadata":
				hasMetadata = true
			}
		}

		if !hasResize {
			t.Error("Web-optimized should have resize operation")
		}
		if !hasCompress {
			t.Error("Web-optimized should have compress operation")
		}
		if !hasMetadata {
			t.Error("Web-optimized should have metadata operation")
		}
	})

	t.Run("HighQualityOperations", func(t *testing.T) {
		preset, ok := GetPreset("high-quality")
		if !ok {
			t.Fatal("Failed to get high-quality preset")
		}

		if len(preset.Operations) == 0 {
			t.Error("High-quality preset should have operations")
		}

		// High-quality should focus on compression with high quality
		for _, op := range preset.Operations {
			if op.Type == "compress" {
				if quality, ok := op.Options["quality"].(int); ok {
					if quality < 90 {
						t.Errorf("High-quality preset should have quality >= 90, got: %d", quality)
					}
				}
			}
		}
	})

	t.Run("AggressiveOperations", func(t *testing.T) {
		preset, ok := GetPreset("aggressive")
		if !ok {
			t.Fatal("Failed to get aggressive preset")
		}

		// Aggressive should have lower quality for maximum compression
		for _, op := range preset.Operations {
			if op.Type == "compress" {
				if quality, ok := op.Options["quality"].(int); ok {
					if quality > 70 {
						t.Logf("Note: Aggressive preset has quality %d (expected lower for max compression)", quality)
					}
				}
			}
		}
	})
}

// TestPresetsFunction tests the preset utility functions
func TestPresetsFunction(t *testing.T) {
	t.Run("ListPresetsFunction", func(t *testing.T) {
		presets := ListPresets()

		if len(presets) == 0 {
			t.Error("Expected at least one preset")
		}

		// Verify each preset has required fields
		for i, preset := range presets {
			if preset.Name == "" {
				t.Errorf("Preset %d has empty name", i)
			}

			if preset.Description == "" {
				t.Errorf("Preset %s has empty description", preset.Name)
			}

			if preset.TargetUse == "" {
				t.Errorf("Preset %s has empty target_use", preset.Name)
			}

			if len(preset.Operations) == 0 {
				t.Errorf("Preset %s has no operations", preset.Name)
			}
		}
	})

	t.Run("GetPresetFunction", func(t *testing.T) {
		// Test existing preset
		preset, ok := GetPreset("web-optimized")
		if !ok {
			t.Error("Failed to get existing preset")
		}

		if preset.Name != "web-optimized" {
			t.Errorf("Expected name 'web-optimized', got: %s", preset.Name)
		}

		// Test non-existent preset
		_, ok = GetPreset("nonexistent")
		if ok {
			t.Error("GetPreset should return false for non-existent preset")
		}
	})

	t.Run("AllPresetsAccessible", func(t *testing.T) {
		expectedPresets := []string{
			"web-optimized",
			"email-safe",
			"aggressive",
			"high-quality",
			"social-media",
		}

		for _, name := range expectedPresets {
			preset, ok := GetPreset(name)
			if !ok {
				t.Errorf("Failed to get preset: %s", name)
				continue
			}

			if preset.Name != name {
				t.Errorf("Preset name mismatch: expected %s, got %s", name, preset.Name)
			}
		}
	})

	t.Run("PresetOperationStructure", func(t *testing.T) {
		preset, ok := GetPreset("web-optimized")
		if !ok {
			t.Fatal("Failed to get preset")
		}

		for i, op := range preset.Operations {
			if op.Type == "" {
				t.Errorf("Operation %d has empty type", i)
			}

			// Options can be empty, but if present should be a map
			if op.Options != nil && len(op.Options) == 0 {
				t.Logf("Operation %d (%s) has empty options map", i, op.Type)
			}
		}
	})
}

// TestPresetEdgeCases tests edge cases in preset handling
func TestPresetEdgeCases(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	server := setupTestServer()

	t.Run("GetPresetWithSpecialCharacters", func(t *testing.T) {
		resp, body, err := makeHTTPRequest(server.app, "GET", "/api/v1/presets/web%20optimized", nil, nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should return 404 as spaces are not valid in preset names
		if resp.StatusCode != 404 {
			t.Logf("Expected 404 for preset with spaces, got: %d", resp.StatusCode)
		}

		_ = body
	})

	t.Run("ApplyPresetEmptyName", func(t *testing.T) {
		resp, body, err := makeHTTPRequest(server.app, "POST", "/api/v1/image/preset/", nil, nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Should either redirect or return 404
		if resp.StatusCode != 404 && resp.StatusCode != 301 && resp.StatusCode != 308 {
			t.Logf("Unexpected status for empty preset name: %d. Body: %s", resp.StatusCode, string(body))
		}
	})

	t.Run("CaseInsensitivePresetName", func(t *testing.T) {
		// Try uppercase preset name
		resp, body, err := makeHTTPRequest(server.app, "GET", "/api/v1/presets/WEB-OPTIMIZED", nil, nil)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}

		// Preset names are case-sensitive by default
		if resp.StatusCode == 200 {
			t.Log("Note: Preset names appear to be case-insensitive")
		} else if resp.StatusCode == 404 {
			// Expected - preset names are case-sensitive
			_ = body
		}
	})

	t.Run("ListPresetsConsistency", func(t *testing.T) {
		// Call list multiple times and verify consistency
		resp1, body1, err := makeHTTPRequest(server.app, "GET", "/api/v1/presets", nil, nil)
		if err != nil {
			t.Fatalf("Failed to make first request: %v", err)
		}

		resp2, body2, err := makeHTTPRequest(server.app, "GET", "/api/v1/presets", nil, nil)
		if err != nil {
			t.Fatalf("Failed to make second request: %v", err)
		}

		result1 := assertJSONResponse(t, resp1, body1, 200)
		result2 := assertJSONResponse(t, resp2, body2, 200)

		count1 := result1["count"].(float64)
		count2 := result2["count"].(float64)

		if count1 != count2 {
			t.Error("Preset count should be consistent across calls")
		}
	})
}
