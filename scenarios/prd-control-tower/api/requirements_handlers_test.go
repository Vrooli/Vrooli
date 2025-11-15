package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

func TestHandleGetRequirements(t *testing.T) {
	tests := []struct {
		name         string
		entityType   string
		entityName   string
		wantStatus   int
		wantMinItems int
	}{
		{
			name:         "get requirements for existing scenario",
			entityType:   "scenario",
			entityName:   "prd-control-tower",
			wantStatus:   http.StatusOK,
			wantMinItems: 0, // May have requirements
		},
		{
			name:         "get requirements for non-existent scenario",
			entityType:   "scenario",
			entityName:   "non-existent-scenario",
			wantStatus:   http.StatusOK,
			wantMinItems: 0, // Empty is valid
		},
		{
			name:       "invalid entity type",
			entityType: "invalid",
			entityName: "test",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/"+tt.entityType+"/"+tt.entityName+"/requirements", nil)
			req = mux.SetURLVars(req, map[string]string{
				"type": tt.entityType,
				"name": tt.entityName,
			})
			w := httptest.NewRecorder()

			handleGetRequirements(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var resp RequirementsResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if resp.EntityType != tt.entityType {
					t.Errorf("expected entity_type %s, got %s", tt.entityType, resp.EntityType)
				}

				if resp.EntityName != tt.entityName {
					t.Errorf("expected entity_name %s, got %s", tt.entityName, resp.EntityName)
				}

				if len(resp.Groups) < tt.wantMinItems {
					t.Errorf("expected at least %d groups, got %d", tt.wantMinItems, len(resp.Groups))
				}
			}
		})
	}
}

func TestHandleGetOperationalTargets(t *testing.T) {
	tests := []struct {
		name         string
		entityType   string
		entityName   string
		wantStatus   int
		checkTargets bool
	}{
		{
			name:         "get targets for existing scenario",
			entityType:   "scenario",
			entityName:   "prd-control-tower",
			wantStatus:   http.StatusOK,
			checkTargets: true,
		},
		{
			name:         "get targets for non-existent scenario",
			entityType:   "scenario",
			entityName:   "non-existent-scenario",
			wantStatus:   http.StatusInternalServerError, // Doesn't exist so error is expected
			checkTargets: false,
		},
		{
			name:       "invalid entity type",
			entityType: "invalid",
			entityName: "test",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/"+tt.entityType+"/"+tt.entityName+"/targets", nil)
			req = mux.SetURLVars(req, map[string]string{
				"type": tt.entityType,
				"name": tt.entityName,
			})
			w := httptest.NewRecorder()

			handleGetOperationalTargets(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("expected status %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
				return // Skip further checks if status doesn't match
			}

			if tt.wantStatus == http.StatusOK {
				var resp OperationalTargetsResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if resp.EntityType != tt.entityType {
					t.Errorf("expected entity_type %s, got %s", tt.entityType, resp.EntityType)
				}

				if resp.EntityName != tt.entityName {
					t.Errorf("expected entity_name %s, got %s", tt.entityName, resp.EntityName)
				}

				// Targets can be empty, that's valid
				t.Logf("Found %d targets for %s/%s", len(resp.Targets), tt.entityType, tt.entityName)
			}
		})
	}
}
