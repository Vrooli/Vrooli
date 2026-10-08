package scenario

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"vrooli-bridge/internal/auth"
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

// A missing procedure on a reachable target is version skew. The failure must
// say which revisions differ and how to update the target, in both the prose
// and machine-readable metadata, so callers stop recommending retry/re-apply.
func TestIncompatibleTargetFailureCarriesVersionSkewAndUpdatePath(t *testing.T) {
	failure := classifyScenarioProxyFailure(errors.New("target scenario: scenario API returned HTTP 404"), "vrooli-onboarding", "/OperatorInputsService/ListOperatorInputs").
		withVersionFacts(VersionFacts{TargetRevision: "8ec19147c85f+dirty", ControlPlaneRevision: "1911dc4aea6", UpdatePath: "reonboard", UpdateCommand: "vrooli-bridge onboard connect --host minimouse.local --user matt --source working-tree"})

	require.Contains(t, failure.message, "target runs revision 8ec19147c85f+dirty and the control plane runs 1911dc4aea6")
	require.Contains(t, failure.message, "`vrooli-bridge onboard connect --host minimouse.local")

	rec := httptest.NewRecorder()
	writeScenarioProxyFailure(rec, httptest.NewRequest(http.MethodPost, "/", nil), failure)
	require.Equal(t, "8ec19147c85f+dirty", rec.Header().Get("X-Vrooli-Target-Revision"))
	require.Equal(t, "1911dc4aea6", rec.Header().Get("X-Vrooli-Control-Plane-Revision"))
	require.Equal(t, "reonboard", rec.Header().Get("X-Vrooli-Update-Path"))
	require.Contains(t, rec.Header().Get("X-Vrooli-Update-Command"), "--source working-tree")
}

// A node on the control plane's revision whose scenario still 404s is running
// a stale process. The failure must say so and name the restart, not tell the
// operator to re-ship a tree the node already has.
func TestIncompatibleTargetOnSameRevisionNamesTheRestart(t *testing.T) {
	failure := classifyScenarioProxyFailure(errors.New("target scenario: scenario API returned HTTP 404"), "vrooli-onboarding", "/OperatorInputsService/ListOperatorInputs").
		withVersionFacts(VersionFacts{TargetRevision: "8585f56f4765+dirty", ControlPlaneRevision: "8585f56f4765", UpdatePath: "restart", UpdateCommand: `vrooli-bridge relay call --node-id n1 --scenario vrooli-onboarding --command "scenario restart"`})

	require.Contains(t, failure.message, "already has revision 8585f56f4765+dirty but its running scenario has not restarted onto it")
	require.Contains(t, failure.message, "scenario restart")
	require.NotContains(t, failure.message, "control plane runs")
}

// Transient failures are not version skew; their wording must not change.
func TestTransientFailureIgnoresVersionFacts(t *testing.T) {
	failure := classifyScenarioProxyFailure(errors.New("target scenario: scenario API returned HTTP 503"), "vrooli-onboarding", "/ReadinessService/GetReadiness")
	enriched := failure.withVersionFacts(VersionFacts{TargetRevision: "a", ControlPlaneRevision: "b", UpdateCommand: "x"})
	require.Equal(t, failure.message, enriched.message)
	require.Empty(t, enriched.version.UpdateCommand)
}

func TestShortRevisionKeepsWorkingTreeMarker(t *testing.T) {
	require.Equal(t, "8ec19147c85f+dirty", shortRevision("8ec19147c85f1cc8fdd57fcbf07ca047776f8179+dirty"))
	require.Equal(t, "1911dc4aea6", shortRevision("1911dc4aea6"))
	require.Equal(t, "", shortRevision(""))
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

type desktopResponseService struct{}

func (desktopResponseService) Call(context.Context, internal.Request) (internal.Response, error) {
	return internal.Response{Body: []byte(`{"state":6}`)}, nil
}

func TestDesktopScenarioProxyPreservesConnectJSONForBrowser(t *testing.T) {
	h := NewHandler(Deps{Service: desktopResponseService{}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/targets/node-1/scenarios/device-control/vrooli.device_control.v1.desktop.DesktopSessionService/GetReadiness", nil)
	req = req.WithContext(auth.WithIdentity(req.Context(), auth.Identity{OwnerID: "owner-1"}))
	req = mux.SetURLVars(req, map[string]string{
		"node": "node-1", "scenario": "device-control", "procedure": "vrooli.device_control.v1.desktop.DesktopSessionService/GetReadiness",
	})
	rec := httptest.NewRecorder()

	h.Call(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	require.JSONEq(t, `{"state":6}`, rec.Body.String())
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
