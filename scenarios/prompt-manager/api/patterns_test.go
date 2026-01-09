package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
)

// TestTestScenarioBuilder tests the scenario builder pattern
func TestTestScenarioBuilder(t *testing.T) {
	t.Run("BuildEmptyScenarios", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		scenarios := builder.Build()

		if scenarios == nil {
			t.Fatal("Expected non-nil scenarios")
		}

		if len(scenarios) != 0 {
			t.Errorf("Expected 0 scenarios, got %d", len(scenarios))
		}
	})

	t.Run("AddInvalidUUID", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddInvalidUUID("/api/v1/campaigns/{id}")
		scenarios := builder.Build()

		if len(scenarios) != 1 {
			t.Fatalf("Expected 1 scenario, got %d", len(scenarios))
		}

		scenario := scenarios[0]
		if scenario.Name != "InvalidUUID" {
			t.Errorf("Expected name 'InvalidUUID', got %s", scenario.Name)
		}

		if scenario.ExpectedStatus != 400 {
			t.Errorf("Expected status 400, got %d", scenario.ExpectedStatus)
		}

		// Test execute function
		req := scenario.Execute(t, nil)
		if req.Method != "GET" {
			t.Errorf("Expected method GET, got %s", req.Method)
		}
		if req.URLVars["id"] != "invalid-uuid-format" {
			t.Errorf("Expected invalid UUID, got %s", req.URLVars["id"])
		}
	})

	t.Run("AddNonExistentCampaign", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddNonExistentCampaign("/api/v1/campaigns/{id}")
		scenarios := builder.Build()

		if len(scenarios) != 1 {
			t.Fatalf("Expected 1 scenario, got %d", len(scenarios))
		}

		scenario := scenarios[0]
		if scenario.Name != "NonExistentCampaign" {
			t.Errorf("Expected name 'NonExistentCampaign', got %s", scenario.Name)
		}

		if scenario.ExpectedStatus != 404 {
			t.Errorf("Expected status 404, got %d", scenario.ExpectedStatus)
		}
	})

	t.Run("AddNonExistentPrompt", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddNonExistentPrompt("/api/v1/prompts/{id}")
		scenarios := builder.Build()

		if len(scenarios) != 1 {
			t.Fatalf("Expected 1 scenario, got %d", len(scenarios))
		}

		scenario := scenarios[0]
		if scenario.Name != "NonExistentPrompt" {
			t.Errorf("Expected name 'NonExistentPrompt', got %s", scenario.Name)
		}

		if scenario.ExpectedStatus != 404 {
			t.Errorf("Expected status 404, got %d", scenario.ExpectedStatus)
		}
	})

	t.Run("AddInvalidJSON", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddInvalidJSON("/api/v1/campaigns", "POST")
		scenarios := builder.Build()

		if len(scenarios) != 1 {
			t.Fatalf("Expected 1 scenario, got %d", len(scenarios))
		}

		scenario := scenarios[0]
		if scenario.Name != "InvalidJSON" {
			t.Errorf("Expected name 'InvalidJSON', got %s", scenario.Name)
		}

		if scenario.ExpectedStatus != 400 {
			t.Errorf("Expected status 400, got %d", scenario.ExpectedStatus)
		}

		// Test execute function
		req := scenario.Execute(t, nil)
		if req.Method != "POST" {
			t.Errorf("Expected method POST, got %s", req.Method)
		}
		if req.Body != `{"invalid": "json"` {
			t.Errorf("Expected malformed JSON, got %v", req.Body)
		}
	})

	t.Run("AddMissingRequiredField", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddMissingRequiredField("/api/v1/campaigns", "POST")
		scenarios := builder.Build()

		if len(scenarios) != 1 {
			t.Fatalf("Expected 1 scenario, got %d", len(scenarios))
		}

		scenario := scenarios[0]
		if scenario.Name != "MissingRequiredField" {
			t.Errorf("Expected name 'MissingRequiredField', got %s", scenario.Name)
		}
	})

	t.Run("AddEmptySearchQuery", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.AddEmptySearchQuery("/api/v1/search/prompts")
		scenarios := builder.Build()

		if len(scenarios) != 1 {
			t.Fatalf("Expected 1 scenario, got %d", len(scenarios))
		}

		scenario := scenarios[0]
		if scenario.Name != "EmptySearchQuery" {
			t.Errorf("Expected name 'EmptySearchQuery', got %s", scenario.Name)
		}

		// Test execute function
		req := scenario.Execute(t, nil)
		if !contains(req.Path, "?q=") {
			t.Errorf("Expected query parameter in path, got %s", req.Path)
		}
	})

	t.Run("ChainMultipleScenarios", func(t *testing.T) {
		builder := NewTestScenarioBuilder()
		builder.
			AddInvalidUUID("/api/v1/campaigns/{id}").
			AddNonExistentCampaign("/api/v1/campaigns/{id}").
			AddInvalidJSON("/api/v1/campaigns", "POST")

		scenarios := builder.Build()

		if len(scenarios) != 3 {
			t.Errorf("Expected 3 scenarios, got %d", len(scenarios))
		}
	})
}

// TestHandlerTestSuite tests the handler test suite pattern
func TestHandlerTestSuite(t *testing.T) {
	server := &APIServer{}
	router := mux.NewRouter()

	// Add a simple test handler
	router.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}).Methods("GET")

	t.Run("EmptySuite", func(t *testing.T) {
		suite := &HandlerTestSuite{
			Name:        "EmptyTestSuite",
			Description: "Suite with no tests",
			Tests:       []HandlerTest{},
		}

		// Should not panic
		suite.RunSuite(t, server, router)
	})

	t.Run("SuiteWithTests", func(t *testing.T) {
		setupCalled := false
		cleanupCalled := false

		suite := &HandlerTestSuite{
			Name: "BasicTestSuite",
			Setup: func(t *testing.T) interface{} {
				setupCalled = true
				return "setup_data"
			},
			Tests: []HandlerTest{
				{
					Name: "SimpleGET",
					Request: HTTPTestRequest{
						Method: "GET",
						Path:   "/test",
					},
					ExpectedStatus: 200,
					ValidateBody: func(t *testing.T, body string) {
						if body != "test response" {
							t.Errorf("Expected 'test response', got %s", body)
						}
					},
				},
			},
			Cleanup: func(setupData interface{}) {
				cleanupCalled = true
				if setupData != "setup_data" {
					t.Errorf("Expected 'setup_data', got %v", setupData)
				}
			},
		}

		suite.RunSuite(t, server, router)

		if !setupCalled {
			t.Error("Expected setup to be called")
		}
		if !cleanupCalled {
			t.Error("Expected cleanup to be called")
		}
	})
}

// TestPerformanceTestPattern tests the performance testing pattern
func TestPerformanceTestPattern(t *testing.T) {
	server := &APIServer{}
	router := mux.NewRouter()

	// Add a test handler
	router.HandleFunc("/fast", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("GET")

	t.Run("SuccessfulPerformanceTest", func(t *testing.T) {
		pattern := PerformanceTestPattern{
			Name:        "FastEndpoint",
			Description: "Test fast endpoint",
			MaxDuration: 100 * time.Millisecond,
			Iterations:  5,
			Execute: func(t *testing.T, iteration int, setupData interface{}) HTTPTestRequest {
				return HTTPTestRequest{
					Method: "GET",
					Path:   "/fast",
				}
			},
		}

		RunPerformanceTest(t, server, router, pattern)
	})

	t.Run("PerformanceTestWithSetupCleanup", func(t *testing.T) {
		setupCalled := false
		cleanupCalled := false
		validateCalled := false

		pattern := PerformanceTestPattern{
			Name:        "WithSetup",
			Description: "Test with setup and cleanup",
			MaxDuration: 100 * time.Millisecond,
			Iterations:  3,
			Setup: func(t *testing.T) interface{} {
				setupCalled = true
				return map[string]int{"counter": 0}
			},
			Execute: func(t *testing.T, iteration int, setupData interface{}) HTTPTestRequest {
				data := setupData.(map[string]int)
				data["counter"]++
				return HTTPTestRequest{
					Method: "GET",
					Path:   "/fast",
				}
			},
			Validate: func(t *testing.T, avgDuration time.Duration, setupData interface{}) {
				validateCalled = true
				data := setupData.(map[string]int)
				if data["counter"] != 3 {
					t.Errorf("Expected 3 iterations, got %d", data["counter"])
				}
			},
			Cleanup: func(setupData interface{}) {
				cleanupCalled = true
			},
		}

		RunPerformanceTest(t, server, router, pattern)

		if !setupCalled {
			t.Error("Expected setup to be called")
		}
		if !cleanupCalled {
			t.Error("Expected cleanup to be called")
		}
		if !validateCalled {
			t.Error("Expected validate to be called")
		}
	})
}

// TestEdgeCasePattern tests the edge case testing pattern
func TestEdgeCasePattern(t *testing.T) {
	server := &APIServer{}
	router := mux.NewRouter()

	t.Run("EmptyEdgeCases", func(t *testing.T) {
		patterns := []EdgeCasePattern{}

		// Should not panic
		RunEdgeCaseTests(t, server, router, patterns)
	})

	t.Run("SingleEdgeCase", func(t *testing.T) {
		testCalled := false

		patterns := []EdgeCasePattern{
			{
				Name:        "TestCase1",
				Description: "First edge case",
				Test: func(t *testing.T, server *APIServer, router *mux.Router) {
					testCalled = true
				},
			},
		}

		RunEdgeCaseTests(t, server, router, patterns)

		if !testCalled {
			t.Error("Expected test to be called")
		}
	})

	t.Run("MultipleEdgeCases", func(t *testing.T) {
		callCount := 0

		patterns := []EdgeCasePattern{
			{
				Name: "TestCase1",
				Test: func(t *testing.T, server *APIServer, router *mux.Router) {
					callCount++
				},
			},
			{
				Name: "TestCase2",
				Test: func(t *testing.T, server *APIServer, router *mux.Router) {
					callCount++
				},
			},
			{
				Name: "TestCase3",
				Test: func(t *testing.T, server *APIServer, router *mux.Router) {
					callCount++
				},
			},
		}

		RunEdgeCaseTests(t, server, router, patterns)

		if callCount != 3 {
			t.Errorf("Expected 3 test calls, got %d", callCount)
		}
	})
}

// TestErrorTestPattern tests the error test pattern structure
func TestErrorTestPattern(t *testing.T) {
	t.Run("CreatePattern", func(t *testing.T) {
		pattern := ErrorTestPattern{
			Name:           "TestError",
			Description:    "Test error description",
			ExpectedStatus: 400,
		}

		if pattern.Name != "TestError" {
			t.Errorf("Expected name 'TestError', got %s", pattern.Name)
		}

		if pattern.ExpectedStatus != 400 {
			t.Errorf("Expected status 400, got %d", pattern.ExpectedStatus)
		}
	})

	t.Run("PatternWithCallbacks", func(t *testing.T) {
		setupCalled := false
		executeCalled := false
		validateCalled := false
		cleanupCalled := false

		pattern := ErrorTestPattern{
			Name:           "CallbackTest",
			ExpectedStatus: 200,
			Setup: func(t *testing.T) interface{} {
				setupCalled = true
				return "test_data"
			},
			Execute: func(t *testing.T, setupData interface{}) HTTPTestRequest {
				executeCalled = true
				if setupData != "test_data" {
					t.Errorf("Expected 'test_data', got %v", setupData)
				}
				return HTTPTestRequest{
					Method: "GET",
					Path:   "/test",
				}
			},
			Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
				validateCalled = true
			},
			Cleanup: func(setupData interface{}) {
				cleanupCalled = true
			},
		}

		// Simulate pattern execution
		if pattern.Setup != nil {
			data := pattern.Setup(t)
			req := pattern.Execute(t, data)
			if req.Method != "GET" {
				t.Error("Execute should return HTTPTestRequest")
			}
			if pattern.Validate != nil {
				pattern.Validate(t, nil, data)
			}
			if pattern.Cleanup != nil {
				pattern.Cleanup(data)
			}
		}

		if !setupCalled {
			t.Error("Expected setup to be called")
		}
		if !executeCalled {
			t.Error("Expected execute to be called")
		}
		if !validateCalled {
			t.Error("Expected validate to be called")
		}
		if !cleanupCalled {
			t.Error("Expected cleanup to be called")
		}
	})
}
