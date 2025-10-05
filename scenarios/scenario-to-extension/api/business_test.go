package main

import (
	"path/filepath"
	"testing"
	"time"
)

// TestBusinessLogic tests core business logic and rules
func TestBusinessLogic(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configCleanup := setupTestConfig(env)
	defer configCleanup()

	t.Run("ExtensionGenerationBusinessRules", func(t *testing.T) {
		// Test that extension generation follows business rules

		t.Run("RequiredFieldsEnforcement", func(t *testing.T) {
			// Test that required fields are enforced
			testCases := []struct {
				name          string
				req           ExtensionGenerateRequest
				expectedError bool
			}{
				{
					name: "AllRequiredFieldsPresent",
					req: ExtensionGenerateRequest{
						ScenarioName: "valid-scenario",
						Config: ExtensionConfig{
							AppName:     "Valid App",
							Description: "Valid description",
							APIEndpoint: "http://localhost:3000",
						},
					},
					expectedError: false,
				},
				{
					name: "MissingScenarioName",
					req: ExtensionGenerateRequest{
						Config: ExtensionConfig{
							AppName:     "Valid App",
							Description: "Valid description",
							APIEndpoint: "http://localhost:3000",
						},
					},
					expectedError: true,
				},
				{
					name: "MissingAppName",
					req: ExtensionGenerateRequest{
						ScenarioName: "valid-scenario",
						Config: ExtensionConfig{
							Description: "Valid description",
							APIEndpoint: "http://localhost:3000",
						},
					},
					expectedError: true,
				},
				{
					name: "MissingAPIEndpoint",
					req: ExtensionGenerateRequest{
						ScenarioName: "valid-scenario",
						Config: ExtensionConfig{
							AppName:     "Valid App",
							Description: "Valid description",
						},
					},
					expectedError: true,
				},
			}

			for _, tc := range testCases {
				t.Run(tc.name, func(t *testing.T) {
					w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
						Method: "POST",
						Path:   "/api/v1/extension/generate",
						Body:   tc.req,
					})

					if err != nil {
						t.Fatalf("Failed to execute request: %v", err)
					}

					if tc.expectedError {
						if w.Code == 201 {
							t.Error("Expected error response, got success")
						}
					} else {
						if w.Code != 201 {
							t.Errorf("Expected success (201), got %d", w.Code)
						}
					}
				})
			}
		})

		t.Run("DefaultValueApplication", func(t *testing.T) {
			// Test that default values are correctly applied
			req := ExtensionGenerateRequest{
				ScenarioName: "defaults-test",
				Config: ExtensionConfig{
					AppName:     "Test App",
					Description: "Test description",
					APIEndpoint: "http://localhost:3000",
					// Intentionally omitting optional fields
				},
			}

			w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/extension/generate",
				Body:   req,
			})

			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			response := assertJSONResponse(t, w, 201, nil)
			buildID := response["build_id"].(string)

			buildsMux.RLock()
			build := builds[buildID]
			buildsMux.RUnlock()

			// Verify defaults were applied
			if build.Config.Version != "1.0.0" {
				t.Errorf("Expected default version 1.0.0, got %s", build.Config.Version)
			}

			if build.Config.License != "MIT" {
				t.Errorf("Expected default license MIT, got %s", build.Config.License)
			}

			if build.TemplateType != "full" {
				t.Errorf("Expected default template type 'full', got %s", build.TemplateType)
			}
		})

		t.Run("ExtensionPathGeneration", func(t *testing.T) {
			// Test that extension paths are correctly generated
			req := TestData.GenerateExtensionRequest(
				"path-test-scenario",
				"Path Test",
				"http://localhost:3000",
			)

			w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/extension/generate",
				Body:   req,
			})

			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			response := assertJSONResponse(t, w, 201, nil)
			buildID := response["build_id"].(string)

			buildsMux.RLock()
			build := builds[buildID]
			buildsMux.RUnlock()

			expectedPath := filepath.Join(config.OutputPath, "path-test-scenario", "platforms", "extension")
			if build.ExtensionPath != expectedPath {
				t.Errorf("Expected path %s, got %s", expectedPath, build.ExtensionPath)
			}
		})
	})

	t.Run("BuildLifecycleManagement", func(t *testing.T) {
		t.Run("BuildCreation", func(t *testing.T) {
			req := TestData.GenerateExtensionRequest(
				"lifecycle-test",
				"Lifecycle Test",
				"http://localhost:3000",
			)

			w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
				Method: "POST",
				Path:   "/api/v1/extension/generate",
				Body:   req,
			})

			if err != nil {
				t.Fatalf("Failed to execute request: %v", err)
			}

			response := assertJSONResponse(t, w, 201, nil)
			buildID := response["build_id"].(string)

			buildsMux.RLock()
			build, exists := builds[buildID]
			buildsMux.RUnlock()

			if !exists {
				t.Fatal("Expected build to exist in global builds map")
			}

			if build.Status != "building" {
				t.Errorf("Expected initial status 'building', got %s", build.Status)
			}

			if build.CreatedAt.IsZero() {
				t.Error("Expected CreatedAt to be set")
			}

			if build.CompletedAt != nil {
				t.Error("Expected CompletedAt to be nil for building status")
			}
		})

		t.Run("BuildCompletion", func(t *testing.T) {
			// Create a build
			build := &ExtensionBuild{
				BuildID:       "completion-test",
				ScenarioName:  "test",
				TemplateType:  "full",
				Status:        "building",
				ExtensionPath: "/tmp/test",
				BuildLog:      []string{},
				ErrorLog:      []string{},
				CreatedAt:     time.Now(),
			}

			buildsMux.Lock()
			builds[build.BuildID] = build
			buildsMux.Unlock()

			// Simulate build process
			generateExtension(build)

			// Wait a bit for async processing
			time.Sleep(200 * time.Millisecond)

			buildsMux.RLock()
			finalBuild := builds[build.BuildID]
			buildsMux.RUnlock()

			if finalBuild.Status == "building" {
				t.Error("Expected build to complete (status should not be 'building')")
			}

			if finalBuild.CompletedAt == nil {
				t.Error("Expected CompletedAt to be set after completion")
			}

			if len(finalBuild.BuildLog) == 0 {
				t.Error("Expected build log to have entries")
			}
		})

		t.Run("BuildFailureHandling", func(t *testing.T) {
			// Create a build with invalid output path to trigger failure
			build := &ExtensionBuild{
				BuildID:       "failure-test",
				ScenarioName:  "test",
				TemplateType:  "full",
				Status:        "building",
				ExtensionPath: "/root/invalid-permission-path/extension",
				BuildLog:      []string{},
				ErrorLog:      []string{},
				CreatedAt:     time.Now(),
			}

			buildsMux.Lock()
			builds[build.BuildID] = build
			buildsMux.Unlock()

			// Simulate build process (should fail)
			generateExtension(build)

			// Wait for async processing
			time.Sleep(200 * time.Millisecond)

			buildsMux.RLock()
			failedBuild := builds[build.BuildID]
			buildsMux.RUnlock()

			if failedBuild.Status != "failed" {
				t.Errorf("Expected status 'failed', got %s", failedBuild.Status)
			}

			if len(failedBuild.ErrorLog) == 0 {
				t.Error("Expected error log to have entries for failed build")
			}
		})
	})

	t.Run("TemplateTypeHandling", func(t *testing.T) {
		templateTypes := []string{"full", "content-script-only", "background-only", "popup-only"}

		for _, templateType := range templateTypes {
			t.Run(templateType, func(t *testing.T) {
				req := ExtensionGenerateRequest{
					ScenarioName: "template-type-test",
					TemplateType: templateType,
					Config: ExtensionConfig{
						AppName:     "Template Type Test",
						Description: "Testing template type handling",
						APIEndpoint: "http://localhost:3000",
					},
				}

				w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/extension/generate",
					Body:   req,
				})

				if err != nil {
					t.Fatalf("Failed to execute request: %v", err)
				}

				response := assertJSONResponse(t, w, 201, nil)
				buildID := response["build_id"].(string)

				buildsMux.RLock()
				build := builds[buildID]
				buildsMux.RUnlock()

				if build.TemplateType != templateType {
					t.Errorf("Expected template type %s, got %s", templateType, build.TemplateType)
				}
			})
		}
	})

	t.Run("CustomVariablesSupport", func(t *testing.T) {
		customVars := map[string]interface{}{
			"custom_feature_1": "enabled",
			"custom_feature_2": 42,
			"custom_feature_3": true,
		}

		req := ExtensionGenerateRequest{
			ScenarioName: "custom-vars-test",
			TemplateType: "full",
			Config: ExtensionConfig{
				AppName:         "Custom Vars Test",
				Description:     "Testing custom variables",
				APIEndpoint:     "http://localhost:3000",
				CustomVariables: customVars,
			},
		}

		w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/generate",
			Body:   req,
		})

		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, 201, nil)
		buildID := response["build_id"].(string)

		buildsMux.RLock()
		build := builds[buildID]
		buildsMux.RUnlock()

		if build.Config.CustomVariables == nil {
			t.Fatal("Expected custom variables to be preserved")
		}

		for key := range customVars {
			_, exists := build.Config.CustomVariables[key]
			if !exists {
				t.Errorf("Expected custom variable %s to exist", key)
			}
		}
	})
}

// TestExtensionTestingBusinessLogic tests extension testing business rules
func TestExtensionTestingBusinessLogic(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configCleanup := setupTestConfig(env)
	defer configCleanup()

	t.Run("TestSiteValidation", func(t *testing.T) {
		testCases := []struct {
			name      string
			sites     []string
			expectErr bool
		}{
			{
				name:      "ValidSites",
				sites:     []string{"https://example.com", "https://google.com"},
				expectErr: false,
			},
			{
				name:      "EmptySites",
				sites:     []string{},
				expectErr: false, // Should use default
			},
			{
				name:      "NilSites",
				sites:     nil,
				expectErr: false, // Should use default
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := ExtensionTestRequest{
					ExtensionPath: "/tmp/test-extension",
					TestSites:     tc.sites,
					Screenshot:    true,
				}

				w, err := executeRequest(testExtensionHandler, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/extension/test",
					Body:   req,
				})

				if err != nil {
					t.Fatalf("Failed to execute request: %v", err)
				}

				if tc.expectErr {
					if w.Code == 200 {
						t.Error("Expected error response, got success")
					}
				} else {
					assertJSONResponse(t, w, 200, nil)
				}
			})
		}
	})

	t.Run("TestResultAggregation", func(t *testing.T) {
		// Test that results are properly aggregated
		req := TestData.GenerateTestRequest("/tmp/test", []string{
			"https://site1.com",
			"https://site2.com",
			"https://site3.com",
		})

		result := testExtension(&req)

		if result == nil {
			t.Fatal("Expected test result")
		}

		if len(result.TestResults) != 3 {
			t.Errorf("Expected 3 test results, got %d", len(result.TestResults))
		}

		// Verify summary calculations
		expectedTotal := 3
		if result.Summary.TotalTests != expectedTotal {
			t.Errorf("Expected total_tests=%d, got %d", expectedTotal, result.Summary.TotalTests)
		}

		if result.Summary.Passed+result.Summary.Failed != expectedTotal {
			t.Error("Passed + Failed should equal TotalTests")
		}

		expectedSuccessRate := float64(result.Summary.Passed) / float64(result.Summary.TotalTests) * 100
		if result.Summary.SuccessRate != expectedSuccessRate {
			t.Errorf("Expected success_rate=%.2f, got %.2f", expectedSuccessRate, result.Summary.SuccessRate)
		}
	})
}

// TestDataIntegrity tests data consistency and integrity
func TestDataIntegrity(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configCleanup := setupTestConfig(env)
	defer configCleanup()

	t.Run("ThreadSafeBuildAccess", func(t *testing.T) {
		// Test thread-safe access to builds map
		const goroutines = 10
		const iterations = 100
		done := make(chan bool, goroutines)

		for g := 0; g < goroutines; g++ {
			go func(id int) {
				for i := 0; i < iterations; i++ {
					// Write
					buildsMux.Lock()
					builds[string(rune(id*iterations+i))] = &ExtensionBuild{
						BuildID: string(rune(id*iterations + i)),
						Status:  "building",
					}
					buildsMux.Unlock()

					// Read
					buildsMux.RLock()
					_ = len(builds)
					buildsMux.RUnlock()
				}
				done <- true
			}(g)
		}

		for g := 0; g < goroutines; g++ {
			<-done
		}

		// No race conditions should occur
		t.Log("Thread-safe build access completed successfully")
	})

	t.Run("BuildDataConsistency", func(t *testing.T) {
		// Create a build
		req := TestData.GenerateExtensionRequest(
			"consistency-test",
			"Consistency Test",
			"http://localhost:3000",
		)

		w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/generate",
			Body:   req,
		})

		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, 201, nil)
		buildID := response["build_id"].(string)

		// Retrieve build multiple times
		for i := 0; i < 10; i++ {
			w2, err := executeRequest(getExtensionStatusHandler, HTTPTestRequest{
				Method:  "GET",
				Path:    "/api/v1/extension/status/" + buildID,
				URLVars: map[string]string{"build_id": buildID},
			})

			if err != nil {
				t.Fatalf("Failed to get status: %v", err)
			}

			statusResponse := assertJSONResponse(t, w2, 200, nil)

			// Verify consistent data
			if statusResponse["build_id"] != buildID {
				t.Errorf("Inconsistent build_id in response %d", i)
			}

			if statusResponse["scenario_name"] != "consistency-test" {
				t.Errorf("Inconsistent scenario_name in response %d", i)
			}
		}
	})
}

// TestBusinessConstraints tests business constraints and validation
func TestBusinessConstraints(t *testing.T) {
	loggerCleanup := setupTestLogger()
	defer loggerCleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	configCleanup := setupTestConfig(env)
	defer configCleanup()

	t.Run("ExtensionNameConstraints", func(t *testing.T) {
		// Test various extension names
		testCases := []struct {
			name      string
			appName   string
			expectErr bool
		}{
			{"ValidName", "My Extension", false},
			{"ValidNameWithNumbers", "Extension123", false},
			{"ValidNameWithHyphens", "My-Extension", false},
			{"ValidNameWithUnderscores", "My_Extension", false},
			{"EmptyName", "", true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := ExtensionGenerateRequest{
					ScenarioName: "name-test",
					Config: ExtensionConfig{
						AppName:     tc.appName,
						Description: "Test",
						APIEndpoint: "http://localhost:3000",
					},
				}

				w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
					Method: "POST",
					Path:   "/api/v1/extension/generate",
					Body:   req,
				})

				if err != nil {
					t.Fatalf("Failed to execute request: %v", err)
				}

				if tc.expectErr {
					if w.Code == 201 {
						t.Error("Expected error for invalid name, got success")
					}
				} else {
					if w.Code != 201 {
						t.Errorf("Expected success for valid name, got %d", w.Code)
					}
				}
			})
		}
	})

	t.Run("PermissionsValidation", func(t *testing.T) {
		// Test that permissions are properly stored
		permissions := []string{"storage", "activeTab", "tabs", "webRequest"}
		hostPermissions := []string{"https://*.example.com/*", "https://google.com/*"}

		req := ExtensionGenerateRequest{
			ScenarioName: "permissions-test",
			Config: ExtensionConfig{
				AppName:         "Permissions Test",
				Description:     "Test",
				APIEndpoint:     "http://localhost:3000",
				Permissions:     permissions,
				HostPermissions: hostPermissions,
			},
		}

		w, err := executeRequest(generateExtensionHandler, HTTPTestRequest{
			Method: "POST",
			Path:   "/api/v1/extension/generate",
			Body:   req,
		})

		if err != nil {
			t.Fatalf("Failed to execute request: %v", err)
		}

		response := assertJSONResponse(t, w, 201, nil)
		buildID := response["build_id"].(string)

		buildsMux.RLock()
		build := builds[buildID]
		buildsMux.RUnlock()

		if len(build.Config.Permissions) != len(permissions) {
			t.Errorf("Expected %d permissions, got %d", len(permissions), len(build.Config.Permissions))
		}

		if len(build.Config.HostPermissions) != len(hostPermissions) {
			t.Errorf("Expected %d host permissions, got %d", len(hostPermissions), len(build.Config.HostPermissions))
		}
	})
}
