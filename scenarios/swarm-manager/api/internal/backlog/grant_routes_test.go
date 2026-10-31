package backlog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/identity"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

func TestInteractiveGrantCannotReachUnreviewedMutations(t *testing.T) {
	for _, tc := range []struct {
		method, path, template string
		want                   int
	}{
		{http.MethodPatch, "/api/v1/backlog/execute/item/archive-item", "/api/v1/backlog/{kind}/{name}/archive-item", 204},
		{http.MethodDelete, "/api/v1/backlog/execute/item", "/api/v1/backlog/{kind}/{name}", 403},
		{http.MethodPost, apiconnect.GoalServiceCloseOutGoalProcedure, apiconnect.GoalServiceCloseOutGoalProcedure, 403},
		{http.MethodPost, apiconnect.BacklogServiceDecideAttemptProcedure, apiconnect.BacklogServiceDecideAttemptProcedure, 403},
		{http.MethodPost, apiconnect.GoalServiceUpdateGoalProcedure, apiconnect.GoalServiceUpdateGoalProcedure, 204},
		{http.MethodPost, apiconnect.BacklogServiceArchiveTerminalItemsProcedure, apiconnect.BacklogServiceArchiveTerminalItemsProcedure, 204},
		{http.MethodPost, "/future.Service/Mutate", "/future.Service/Mutate", 403},
	} {
		t.Run(tc.path, func(t *testing.T) {
			router := mux.NewRouter()
			router.Use(InteractiveGrantMiddleware)
			router.HandleFunc(tc.template, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) }).Methods(tc.method)
			ctx := identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorAgent, Subject: "run", Verified: true, Grant: &identity.OperationGrant{ID: "grant"}})
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(tc.method, tc.path, nil).WithContext(ctx))
			if w.Code != tc.want {
				t.Fatalf("got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestBookkeepingBoundaryCannotBeEvadedByDroppingAgentIdentity(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/queue", nil)
	response := httptest.NewRecorder()
	InteractiveGrantMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) })).ServeHTTP(response, req)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous caller reached retained operator mutation: %d", response.Code)
	}
}
