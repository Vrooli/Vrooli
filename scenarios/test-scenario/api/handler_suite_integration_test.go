package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestScenarioBuilderJSONValidator exercises the builder helpers against a realistic handler
func TestScenarioBuilderJSONValidator(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	validatorHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "missing content type"})
			return
		}

		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid json"})
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}

	suite := HandlerTestSuite{
		HandlerName: "JSONValidator",
		Handler:     validatorHandler,
		BaseURL:     "/validator",
	}

	patterns := NewTestScenarioBuilder().
		AddInvalidJSON("/validator", http.MethodPost).
		AddMissingContentType("/validator", http.MethodPost).
		AddCustom(ErrorTestPattern{
			Name:           "ValidPayload",
			Description:    "Ensures valid payload succeeds",
			ExpectedStatus: http.StatusOK,
			Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
				return &HTTPTestRequest{
					Method: http.MethodPost,
					Path:   "/validator",
					Body: map[string]string{
						"name": "tester",
					},
					Headers: map[string]string{
						"Content-Type": "application/json",
					},
				}
			},
			Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
				assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{"status": "ok"})
			},
		}).
		Build()

	suite.RunErrorTests(t, patterns)
}

// TestHandleConditionalWithHandlerSuite verifies handleConditional via HandlerTestSuite patterns
func TestHandleConditionalWithHandlerSuite(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	suite := HandlerTestSuite{
		HandlerName: "handleConditional",
		Handler:     handleConditional,
		BaseURL:     "/conditional",
	}

	patterns := NewTestScenarioBuilder().
		AddCustom(ErrorTestPattern{
			Name:           "CreateWithPost",
			Description:    "POST requests should create resources",
			ExpectedStatus: http.StatusCreated,
			Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
				return &HTTPTestRequest{Method: http.MethodPost, Path: "/conditional"}
			},
			Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
				assertJSONResponse(t, w, http.StatusCreated, map[string]interface{}{"status": "ok"})
			},
		}).
		AddCustom(ErrorTestPattern{
			Name:           "ReadWithGet",
			Description:    "GET requests return data",
			ExpectedStatus: http.StatusOK,
			Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
				return &HTTPTestRequest{Method: http.MethodGet, Path: "/conditional"}
			},
			Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
				assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{"status": "ok"})
			},
		}).
		AddCustom(ErrorTestPattern{
			Name:           "HeadDefaults",
			Description:    "HEAD falls back to default branch",
			ExpectedStatus: http.StatusOK,
			Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
				return &HTTPTestRequest{Method: http.MethodHead, Path: "/conditional"}
			},
			Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
				response := assertJSONResponse(t, w, http.StatusOK, nil)
				if response != nil && response["status"] != "ok" {
					t.Errorf("expected status ok, got %v", response["status"])
				}
			},
		}).
		Build()

	suite.RunErrorTests(t, patterns)
}

// TestHandleSwitchWithHandlerSuite ensures the switch-based handler covers all branches
func TestHandleSwitchWithHandlerSuite(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	suite := HandlerTestSuite{
		HandlerName: "handleSwitch",
		Handler:     handleSwitch,
		BaseURL:     "/switch",
	}

	patterns := NewTestScenarioBuilder().
		AddCustom(ErrorTestPattern{
			Name:           "PostBranch",
			ExpectedStatus: http.StatusCreated,
			Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
				return &HTTPTestRequest{Method: http.MethodPost, Path: "/switch"}
			},
			Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
				if w.Body.Len() != 0 {
					t.Errorf("expected empty body for POST, got %s", w.Body.String())
				}
			},
		}).
		AddCustom(ErrorTestPattern{
			Name:           "DeleteBranch",
			ExpectedStatus: http.StatusNoContent,
			Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
				return &HTTPTestRequest{Method: http.MethodDelete, Path: "/switch"}
			},
			Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
				if w.Body.Len() != 0 {
					t.Errorf("expected empty body for DELETE, got %s", w.Body.String())
				}
			},
		}).
		AddCustom(ErrorTestPattern{
			Name:           "FallbackBranch",
			ExpectedStatus: http.StatusOK,
			Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
				return &HTTPTestRequest{Method: http.MethodPut, Path: "/switch"}
			},
			Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
				if w.Code != http.StatusOK {
					t.Errorf("expected fallback status 200, got %d", w.Code)
				}
			},
		}).
		Build()

	suite.RunErrorTests(t, patterns)
}

// TestEdgeHandlersWithHandlerSuite uses HandlerTestSuite to exercise smaller edge handlers
func TestEdgeHandlersWithHandlerSuite(t *testing.T) {
	cleanup := setupTestLogger()
	defer cleanup()

	env := setupTestDirectory(t)
	defer env.Cleanup()

	cases := []struct {
		name     string
		handler  http.HandlerFunc
		patterns []ErrorTestPattern
	}{
		{
			name:    "handleWithVariable",
			handler: handleWithVariable,
			patterns: NewTestScenarioBuilder().
				AddCustom(ErrorTestPattern{
					Name:           "HappyPath",
					ExpectedStatus: http.StatusOK,
					Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
						return &HTTPTestRequest{Method: http.MethodGet, Path: "/vars"}
					},
					Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
						assertJSONResponse(t, w, http.StatusOK, map[string]interface{}{"ok": "true"})
					},
				}).
				Build(),
		},
		{
			name:    "handleWithFunction",
			handler: handleWithFunction,
			patterns: NewTestScenarioBuilder().
				AddCustom(ErrorTestPattern{
					Name:           "ReturnsNotFound",
					ExpectedStatus: http.StatusNotFound,
					Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
						return &HTTPTestRequest{Method: http.MethodGet, Path: "/fn"}
					},
					Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
						assertJSONResponse(t, w, http.StatusNotFound, map[string]interface{}{"error": "not found"})
					},
				}).
				Build(),
		},
		{
			name:    "handleCustomError",
			handler: handleCustomError,
			patterns: NewTestScenarioBuilder().
				AddCustom(ErrorTestPattern{
					Name:           "ReturnsCustomError",
					ExpectedStatus: http.StatusOK,
					Execute: func(t *testing.T, setupData interface{}) *HTTPTestRequest {
						return &HTTPTestRequest{Method: http.MethodGet, Path: "/custom"}
					},
					Validate: func(t *testing.T, w *httptest.ResponseRecorder, setupData interface{}) {
						response := assertJSONResponse(t, w, http.StatusOK, nil)
						if response == nil || response["error"] == nil {
							t.Error("expected error field in custom error response")
						}
					},
				}).
				Build(),
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			suite := HandlerTestSuite{
				HandlerName: tc.name,
				Handler:     tc.handler,
				BaseURL:     "/edge",
			}
			suite.RunErrorTests(t, tc.patterns)
		})
	}
}
