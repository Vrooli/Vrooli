package middleware

import (
	"github.com/vrooli/api-core/identity"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNutritionAgentNeedsOperationCapabilityWithoutHumanApproval(t *testing.T) {
	for _, test := range []struct {
		method, scope string
		want          int
	}{{"GetPlan", "read", 204}, {"ApplyPlan", "read", 403}, {"ApplyPlan", "write", 204}, {"RecoverRestoreCheckpoint", "write", 403}, {"RecoverRestoreCheckpoint", "destructive", 204}, {"UnclassifiedOperation", "destructive", 403}} {
		req := httptest.NewRequest(http.MethodPost, "/vrooli.nutrition_planner.v1.planning.PlanningService/"+test.method, nil)
		req = req.WithContext(identity.WithPrincipal(req.Context(), identity.Principal{Kind: identity.ActorAgent, Subject: "run", Owner: "owner", Verified: true, Scopes: []string{"nutrition-planner:" + test.scope}}))
		rec := httptest.NewRecorder()
		AgentCapability(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })).ServeHTTP(rec, req)
		if rec.Code != test.want {
			t.Fatalf("%s with %s: status=%d want=%d", test.method, test.scope, rec.Code, test.want)
		}
	}
}
