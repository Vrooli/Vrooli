package maintenance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/api-core/identity"
)

type ownerVerifier struct{ principal identity.Principal }

func testInterlock(t *testing.T) *ScenarioInterlock {
	t.Helper()
	l, err := NewScenarioInterlock(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func emptyInventory(context.Context) (Inventory, error) {
	count := 0
	return Inventory{Remaining: &count, Work: []WorkRef{}, Executors: []ExecutorEvidence{}, Unknown: []string{}}, nil
}

func (v ownerVerifier) Verify(context.Context, string) (identity.Principal, error) {
	return v.principal, nil
}

func TestMaintenanceHTTPRequiresVerifiedOwnerAuthority(t *testing.T) {
	for _, tc := range []struct {
		name      string
		principal identity.Principal
		token     string
		want      int
	}{
		{"missing", identity.Principal{}, "", http.StatusUnauthorized},
		{"agent", identity.Principal{Verified: true, Kind: identity.ActorAgent, Subject: "run", Scopes: []string{"*"}}, "agent", http.StatusForbidden},
		{"unscoped", identity.Principal{Verified: true, Kind: identity.ActorHuman, Subject: "owner"}, "owner", http.StatusForbidden},
		{"unverified", identity.Principal{Kind: identity.ActorHuman, Subject: "owner", Scopes: []string{"*"}}, "owner", http.StatusForbidden},
		{"owner", identity.Principal{Verified: true, Kind: identity.ActorHuman, Subject: "owner", Scopes: []string{"agent-manager:write"}}, "owner", http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gate, _ := openGate(t, filepath.Join(t.TempDir(), "gate.db"))
			handler, err := NewHandler(gate, emptyInventory, ownerVerifier{tc.principal}, testInterlock(t))
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPost, "/api/v1/maintenance/admission/enter", strings.NewReader(`{"reason":"rollout"}`))
			if tc.token != "" {
				request.Header.Set("Authorization", "Bearer "+tc.token)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != tc.want {
				t.Fatalf("http=%d %s", response.Code, response.Body.String())
			}
			standing, err := gate.Status(t.Context())
			if err != nil || standing.Closed != (tc.want == http.StatusOK) {
				t.Fatalf("unauthorized mutation: %+v %v", standing, err)
			}
		})
	}
}

func TestMaintenanceHTTPRefusesUnwiredDrain(t *testing.T) {
	gate, _ := openGate(t, filepath.Join(t.TempDir(), "gate.db"))
	if _, err := NewHandler(gate, nil, ownerVerifier{}, testInterlock(t)); err == nil {
		t.Fatal("missing owner drain accepted")
	}
}

func TestMaintenanceHTTPTimeoutAndStaleResumePreserveFence(t *testing.T) {
	gate, _ := openGate(t, filepath.Join(t.TempDir(), "gate.db"))
	state, err := gate.Enter(t.Context(), "owner", "maintenance")
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewHandler(gate, func(ctx context.Context) (Inventory, error) { return Inventory{}, context.DeadlineExceeded }, ownerVerifier{identity.Principal{Verified: true, Kind: identity.ActorHuman, Subject: "owner", Scopes: []string{"*"}}}, testInterlock(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		operation, body string
		want            int
	}{
		{"wait", `{"timeoutSeconds":1}`, http.StatusRequestTimeout},
		{"resume", `{"revision":0}`, http.StatusConflict},
		{"enter", `{"reason":"rollout","force":true}`, http.StatusBadRequest},
	} {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/maintenance/admission/"+tc.operation, strings.NewReader(tc.body))
		request.Header.Set("Authorization", "Bearer owner")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != tc.want {
			t.Fatalf("%s=%d %s", tc.operation, response.Code, response.Body.String())
		}
		standing, err := gate.Status(t.Context())
		if err != nil || !standing.Closed || standing.Revision != state.Revision {
			t.Fatalf("fence changed: %+v %v", standing, err)
		}
	}
}
