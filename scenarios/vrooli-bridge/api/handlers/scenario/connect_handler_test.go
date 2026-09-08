package scenario

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	internal "vrooli-bridge/internal/scenario"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

type scenarioTestService struct{ called bool }

func (s *scenarioTestService) Call(context.Context, internal.Request) (internal.Response, error) {
	s.called = true
	return internal.Response{}, nil
}

func TestScenarioProxyRequiresOwnerBeforeAdmission(t *testing.T) {
	svc := &scenarioTestService{}
	h := NewHandler(Deps{Service: svc})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/targets/node-1/scenarios/vrooli-onboarding/not-a-procedure", nil)
	req = mux.SetURLVars(req, map[string]string{
		"node": "node-1", "scenario": "vrooli-onboarding", "procedure": "not-a-procedure",
	})
	rec := httptest.NewRecorder()

	h.Call(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.False(t, svc.called)
}
