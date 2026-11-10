package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	coreidentity "github.com/vrooli/api-core/identity"
)

func taskRequestAsAgentWithoutGrant(req *http.Request) *http.Request {
	principal := coreidentity.Principal{
		Kind: coreidentity.ActorAgent, Subject: "verified-run", Owner: "owner", Verified: true,
		Scopes: []string{"agent-manager:read", "agent-manager:write"}, ExpiresAt: time.Now().Add(time.Hour),
	}
	return req.WithContext(coreidentity.WithPrincipal(req.Context(), principal))
}

// An agent refused a task operation must be told the exact command that asks
// the operator to approve that operation on that task, in the 403 body
// (`request_authority`) and in the message that is printed.
func TestTaskCancelAndDeleteRefusalCarryTheExactRequestForThatTask(t *testing.T) {
	t.Setenv("AGENT_MANAGER_AUTHORIZATION_URL", "https://agent-manager.example.test")
	_, router := setupTestHandler(t)
	for _, tc := range []struct{ name, method, path, message string }{
		{"cancel", http.MethodPost, "/cancel", "task cancellation requires exact agent.manage authority"},
		{"delete", http.MethodDelete, "", "task deletion requires agent.manage authority"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			id := uuid.NewString()
			req := httptest.NewRequest(tc.method, "/api/v1/tasks/"+id+tc.path, nil)
			rr := httptest.NewRecorder()
			router.ServeHTTP(rr, taskRequestAsAgentWithoutGrant(req))
			if rr.Code != http.StatusForbidden {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
			var body map[string]string
			if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			want := `vrooli agent authorize "$VROOLI_AGENT_SESSION" --policy agent-management --target agent-manager:task/` + id + ` --approval-base https://agent-manager.example.test --reason "<why>"`
			if body["request_authority"] != want {
				t.Fatalf("request_authority = %q, want %q", body["request_authority"], want)
			}
			if !strings.HasPrefix(body["error"], tc.message) || !strings.Contains(body["error"], want) {
				t.Fatalf("error text does not lead with the refusal and carry the command: %q", body["error"])
			}
			if body["operation"] != "agent.manage" || body["target"] != "agent-manager:task/"+id || body["policy"] != "agent-management" {
				t.Fatalf("refusal detail = %v", body)
			}
		})
	}
}

// A human without the capability has no agent request to run, so the body keeps
// its plain shape.
func TestTaskCancelRefusalForANonAgentOffersNoAgentRequest(t *testing.T) {
	_, router := setupTestHandler(t)
	human := coreidentity.Principal{Kind: coreidentity.ActorHuman, Subject: "viewer", Verified: true, Scopes: []string{"agent-manager:read"}, ExpiresAt: time.Now().Add(time.Hour)}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+uuid.NewString()+"/cancel", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req.WithContext(coreidentity.WithPrincipal(req.Context(), human)))
	if rr.Code != http.StatusForbidden || strings.Contains(rr.Body.String(), "request_authority") {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
}
