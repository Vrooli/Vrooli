package selection

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vrooli/api-core/identity"
	setupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/setup/v1"
	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-onboarding/v1/selection"
	"github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/authz"
	internalselection "github.com/vrooli/vrooli/scenarios/vrooli-onboarding/internal/selection"
)

func TestRESTHandoffUsesCanonicalSelectionErrorStatus(t *testing.T) {
	tests := []struct {
		name       string
		serviceErr error
		wantStatus int
	}{
		{name: "unsupported schema", serviceErr: errors.New("unsupported setup selection schema version \"old\""), wantStatus: http.StatusBadRequest},
		{name: "stale target revision", serviceErr: errors.New("stale or outside the authorized target revision"), wantStatus: http.StatusPreconditionFailed},
		{name: "internal failure", serviceErr: errors.New("storage unavailable"), wantStatus: http.StatusInternalServerError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := internalselection.Service{
				CreateHandoff: func(context.Context, *selectionv1.CreateHandoffRequest) (*selectionv1.CreateHandoffResponse, error) {
					return nil, tt.serviceErr
				},
			}
			request := httptest.NewRequest(http.MethodPost, "/api/v2/handoff", strings.NewReader(`{"target":"local","node_id":"node-1","node_kind":"agent"}`))
			request = request.WithContext(identity.WithPrincipal(request.Context(), identity.Principal{
				Kind: identity.ActorHuman, Subject: "operator-1", Verified: true,
				Scopes: []string{authz.MutationCapability},
			}))
			response := httptest.NewRecorder()

			RESTHandoffHandler(service).ServeHTTP(response, request)

			if response.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, tt.wantStatus, response.Body.String())
			}
		})
	}
}

func TestRESTHandoffPreservesBridgeWireProjection(t *testing.T) {
	called := false
	service := internalselection.Service{
		CreateHandoff: func(_ context.Context, request *selectionv1.CreateHandoffRequest) (*selectionv1.CreateHandoffResponse, error) {
			called = true
			if request.GetTarget() != "local" || request.GetMachineId() != "machine-1" || request.GetNodeId() != "node-1" || request.GetNodeKind() != "agent" {
				t.Fatalf("handoff request = %+v", request)
			}
			return &selectionv1.CreateHandoffResponse{Selection: &setupv1.Selection{
				SchemaVersion: "v1", Target: "local", Scenarios: []string{"alpha"}, Apply: true,
			}}, nil
		},
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v2/handoff", strings.NewReader(`{"target":"local","machine_id":"machine-1","node_id":"node-1","node_kind":"agent"}`))
	request = request.WithContext(identity.WithPrincipal(request.Context(), identity.Principal{
		Kind: identity.ActorHuman, Subject: "operator-1", Verified: true,
		Scopes: []string{authz.MutationCapability},
	}))
	response := httptest.NewRecorder()

	RESTHandoffHandler(service).ServeHTTP(response, request)

	if !called || response.Code != http.StatusOK {
		t.Fatalf("called = %t, status = %d, body=%s", called, response.Code, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["schema_version"] != "v1" || body["target"] != "local" || body["apply"] != true {
		t.Fatalf("REST projection = %v", body)
	}
	if _, exists := body["schemaVersion"]; exists {
		t.Fatalf("Bridge wire format changed to protobuf JSON casing: %v", body)
	}
	if got, ok := body["scenarios"].([]any); !ok || len(got) != 1 || got[0] != "alpha" {
		t.Fatalf("scenarios = %v", body["scenarios"])
	}
}
