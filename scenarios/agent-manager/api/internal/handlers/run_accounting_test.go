package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
)

type runAccountingServiceStub struct {
	accounting orchestration.RunAccounting
	err        error
}

func (s runAccountingServiceStub) RunAccounting(context.Context, uuid.UUID) (orchestration.RunAccounting, error) {
	return s.accounting, s.err
}

func TestGetRunAccountingEmitsUnknownUsageExplicitly(t *testing.T) {
	runID := uuid.New()
	h := &Handler{svc: orchestration.HandlerServices{RunAccountingService: runAccountingServiceStub{accounting: orchestration.RunAccounting{RunID: runID, Terminal: true}}}}
	req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/api/v1/runs/"+runID.String()+"/accounting", nil), map[string]string{"id": runID.String()})
	recorder := httptest.NewRecorder()

	h.GetRunAccounting(recorder, req)

	require.Equal(t, http.StatusOK, recorder.Code)
	for _, field := range []string{`"terminal":true`, `"tokens_known":false`, `"charge_measured":false`, `"wall_seconds":"0"`} {
		require.Truef(t, strings.Contains(recorder.Body.String(), field), "accounting JSON missing %s: %s", field, recorder.Body.String())
	}
}

func TestGetRunAccountingRejectsInvalidRunID(t *testing.T) {
	h := &Handler{svc: orchestration.HandlerServices{RunAccountingService: runAccountingServiceStub{}}}
	req := mux.SetURLVars(httptest.NewRequest(http.MethodGet, "/api/v1/runs/not-a-uuid/accounting", nil), map[string]string{"id": "not-a-uuid"})
	recorder := httptest.NewRecorder()

	h.GetRunAccounting(recorder, req)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestAgentManagerConnectGetRunAccountingProjectsUsageAndNotFound(t *testing.T) {
	runID := uuid.New()
	h := &AgentManagerConnectHandler{h: &Handler{svc: orchestration.HandlerServices{
		RunAccountingService: runAccountingServiceStub{accounting: orchestration.RunAccounting{RunID: runID, Terminal: true, Tokens: 1200, TokensKnown: true, ChargeMicroUSD: 4500, ChargeMeasured: true, WallSeconds: 91}},
	}}}

	response, err := h.GetRunAccounting(context.Background(), connect.NewRequest(&apipb.GetRunAccountingRequest{RunId: runID.String()}))
	require.NoError(t, err)
	require.Equal(t, runID.String(), response.Msg.GetRunId())
	require.True(t, response.Msg.GetTokensKnown())
	require.EqualValues(t, 4500, response.Msg.GetChargeMicroUsd())
	require.EqualValues(t, 91, response.Msg.GetWallSeconds())

	missing := uuid.New()
	h.h.svc.RunAccountingService = runAccountingServiceStub{err: domain.NewNotFoundError("Run", missing)}
	_, err = h.GetRunAccounting(context.Background(), connect.NewRequest(&apipb.GetRunAccountingRequest{RunId: missing.String()}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}
