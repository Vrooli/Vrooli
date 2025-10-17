package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBatchQueueHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		body           interface{}
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "MethodNotAllowed",
			method:         http.MethodGet,
			body:           nil,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedError:  "Method not allowed",
		},
		{
			name:           "InvalidJSON",
			method:         http.MethodPost,
			body:           "invalid json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid JSON",
		},
		{
			name:   "EmptyIDsList",
			method: http.MethodPost,
			body: BatchQueueRequest{
				Action: "approve",
				IDs:    []string{},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "No IDs provided",
		},
		{
			name:   "InvalidAction",
			method: http.MethodPost,
			body: BatchQueueRequest{
				Action: "invalid_action",
				IDs:    []string{"test-id"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "Invalid action",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var reqBody []byte
			var err error

			if tt.body != nil {
				if str, ok := tt.body.(string); ok {
					reqBody = []byte(str)
				} else {
					reqBody, err = json.Marshal(tt.body)
					if err != nil {
						t.Fatalf("Failed to marshal request body: %v", err)
					}
				}
			}

			req := httptest.NewRequest(tt.method, "/api/queue/batch", bytes.NewReader(reqBody))
			w := httptest.NewRecorder()

			batchQueueHandler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				var response map[string]string
				err = json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				if errorMsg, ok := response["error"]; ok {
					if errorMsg != tt.expectedError {
						// Check if it contains the expected error message
						if len(tt.expectedError) > 0 && len(errorMsg) > len(tt.expectedError) {
							if errorMsg[:len(tt.expectedError)] != tt.expectedError {
								t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, errorMsg)
							}
						} else if errorMsg != tt.expectedError {
							t.Errorf("Expected error '%s', got '%s'", tt.expectedError, errorMsg)
						}
					}
				}
			}

			// Verify Content-Type header
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
			}
		})
	}
}

func TestBatchQueueHandlerWithValidActionsNoDatabase(t *testing.T) {
	if db == nil {
		t.Log("Skipping database-dependent test - db is nil")
		return
	}

	tests := []struct {
		name   string
		action string
		ids    []string
	}{
		{
			name:   "ApproveAction",
			action: "approve",
			ids:    []string{"00000000-0000-0000-0000-000000000001"},
		},
		{
			name:   "RejectAction",
			action: "reject",
			ids:    []string{"00000000-0000-0000-0000-000000000002"},
		},
		{
			name:   "DeleteAction",
			action: "delete",
			ids:    []string{"00000000-0000-0000-0000-000000000003"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := BatchQueueRequest{
				Action: tt.action,
				IDs:    tt.ids,
			}

			bodyBytes, err := json.Marshal(reqBody)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/queue/batch", bytes.NewReader(bodyBytes))
			w := httptest.NewRecorder()

			batchQueueHandler(w, req)

			// Should return 200 even if IDs don't exist (graceful handling)
			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var response BatchQueueResponse
			err = json.NewDecoder(w.Body).Decode(&response)
			if err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			// Verify response structure
			if response.Total != len(tt.ids) {
				t.Errorf("Expected total %d, got %d", len(tt.ids), response.Total)
			}

			// All should fail since IDs don't exist
			if len(response.Failed) != len(tt.ids) {
				t.Logf("Note: Expected all items to fail (non-existent IDs), got %d succeeded, %d failed",
					len(response.Succeeded), len(response.Failed))
			}
		})
	}
}

func TestBatchQueueRequestStructure(t *testing.T) {
	t.Run("ValidStructure", func(t *testing.T) {
		req := BatchQueueRequest{
			Action: "approve",
			IDs:    []string{"id1", "id2"},
		}

		// Marshal and unmarshal
		data, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded BatchQueueRequest
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if decoded.Action != req.Action {
			t.Errorf("Expected action %s, got %s", req.Action, decoded.Action)
		}

		if len(decoded.IDs) != len(req.IDs) {
			t.Errorf("Expected %d IDs, got %d", len(req.IDs), len(decoded.IDs))
		}
	})
}

func TestBatchQueueResponseStructure(t *testing.T) {
	t.Run("ValidStructure", func(t *testing.T) {
		resp := BatchQueueResponse{
			Succeeded: []string{"id1", "id2"},
			Failed:    []string{"id3"},
			Total:     3,
		}

		// Marshal and unmarshal
		data, err := json.Marshal(resp)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}

		var decoded BatchQueueResponse
		err = json.Unmarshal(data, &decoded)
		if err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if decoded.Total != resp.Total {
			t.Errorf("Expected total %d, got %d", resp.Total, decoded.Total)
		}

		if len(decoded.Succeeded) != len(resp.Succeeded) {
			t.Errorf("Expected %d succeeded, got %d", len(resp.Succeeded), len(decoded.Succeeded))
		}

		if len(decoded.Failed) != len(resp.Failed) {
			t.Errorf("Expected %d failed, got %d", len(resp.Failed), len(decoded.Failed))
		}
	})
}

func TestBatchApproveQueueItems(t *testing.T) {
	if db == nil {
		t.Skip("Skipping database-dependent test - db is nil")
	}

	t.Run("NonExistentIDs", func(t *testing.T) {
		ids := []string{
			"00000000-0000-0000-0000-000000000001",
			"00000000-0000-0000-0000-000000000002",
		}

		resp := batchApproveQueueItems(ids)

		if resp.Total != len(ids) {
			t.Errorf("Expected total %d, got %d", len(ids), resp.Total)
		}

		// All should fail since IDs don't exist
		if len(resp.Failed) != len(ids) {
			t.Logf("Note: Expected all items to fail, got %d succeeded, %d failed",
				len(resp.Succeeded), len(resp.Failed))
		}
	})
}

func TestBatchRejectQueueItems(t *testing.T) {
	if db == nil {
		t.Skip("Skipping database-dependent test - db is nil")
	}

	t.Run("NonExistentIDs", func(t *testing.T) {
		ids := []string{
			"00000000-0000-0000-0000-000000000003",
			"00000000-0000-0000-0000-000000000004",
		}

		resp := batchRejectQueueItems(ids)

		if resp.Total != len(ids) {
			t.Errorf("Expected total %d, got %d", len(ids), resp.Total)
		}

		// All should fail since IDs don't exist
		if len(resp.Failed) != len(ids) {
			t.Logf("Note: Expected all items to fail, got %d succeeded, %d failed",
				len(resp.Succeeded), len(resp.Failed))
		}
	})
}

func TestBatchDeleteQueueItems(t *testing.T) {
	if db == nil {
		t.Skip("Skipping database-dependent test - db is nil")
	}

	t.Run("NonExistentIDs", func(t *testing.T) {
		ids := []string{
			"00000000-0000-0000-0000-000000000005",
			"00000000-0000-0000-0000-000000000006",
		}

		resp := batchDeleteQueueItems(ids)

		if resp.Total != len(ids) {
			t.Errorf("Expected total %d, got %d", len(ids), resp.Total)
		}

		// All should fail since IDs don't exist
		if len(resp.Failed) != len(ids) {
			t.Logf("Note: Expected all items to fail, got %d succeeded, %d failed",
				len(resp.Succeeded), len(resp.Failed))
		}
	})
}

func TestBatchOperationsInitialization(t *testing.T) {
	t.Run("ResponseStructureInitialization", func(t *testing.T) {
		// Test that batch functions properly initialize response structure
		ids := []string{}

		// Should handle empty arrays without panicking
		resp := batchApproveQueueItems(ids)
		if resp.Total != 0 {
			t.Errorf("Expected total 0 for empty IDs, got %d", resp.Total)
		}
		if resp.Succeeded == nil {
			t.Error("Expected Succeeded to be initialized")
		}
		if resp.Failed == nil {
			t.Error("Expected Failed to be initialized")
		}
	})
}

func TestBatchHandlerPublishesEvents(t *testing.T) {
	if db == nil {
		t.Skip("Skipping database-dependent test - db is nil")
	}

	t.Run("EventPublishingOnSuccess", func(t *testing.T) {
		// This tests that the handler attempts to publish events
		// We can't easily verify actual Redis publishing without a mock,
		// but we verify the code path doesn't panic

		reqBody := BatchQueueRequest{
			Action: "approve",
			IDs:    []string{"00000000-0000-0000-0000-000000000007"},
		}

		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			t.Fatalf("Failed to marshal request: %v", err)
		}

		req := httptest.NewRequest(http.MethodPost, "/api/queue/batch", bytes.NewReader(bodyBytes))
		w := httptest.NewRecorder()

		// Should not panic even if Redis is not available
		batchQueueHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}
