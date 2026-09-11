package scenario

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	internal "vrooli-bridge/internal/scenario"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

func TestClassifyScenarioProxyFailureDistinguishesMissingRoute(t *testing.T) {
	failure := classifyScenarioProxyFailure(errors.New("target scenario: scenario API returned HTTP 404"), "vrooli-onboarding", "/OperatorInputsService/ListOperatorInputs")

	require.Equal(t, connect.CodeFailedPrecondition, failure.code)
	require.Equal(t, "target_incompatible", failure.classification)
	require.False(t, failure.retry)
	require.Equal(t, http.StatusNotFound, failure.upstreamStatus)
	require.Contains(t, failure.message, "missing the required API procedure")
}

func TestClassifyScenarioProxyFailureKeepsTransientFailureRetryable(t *testing.T) {
	failure := classifyScenarioProxyFailure(errors.New("target scenario: scenario API returned HTTP 503"), "vrooli-onboarding", "/ReadinessService/GetReadiness")

	require.Equal(t, connect.CodeUnavailable, failure.code)
	require.Equal(t, "target_service_failure", failure.classification)
	require.True(t, failure.retry)
	require.Equal(t, http.StatusServiceUnavailable, failure.upstreamStatus)
}

func TestWriteScenarioProxyFailureUsesConnectErrorContract(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/targets/node-1/scenarios/vrooli-onboarding/procedure", nil)
	failure := scenarioProxyFailure{
		code:           connect.CodeFailedPrecondition,
		classification: "target_incompatible",
		message:        "target onboarding API is incompatible",
		upstreamStatus: http.StatusNotFound,
	}

	writeScenarioProxyFailure(rec, req, failure)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	require.Contains(t, rec.Body.String(), "target onboarding API is incompatible")
}

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
