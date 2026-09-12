package handlers

import (
	"context"
	"net/http/httptest"
	"testing"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/runreport"
	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/connectx"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api"
	"github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/api/apiconnect"
)

func TestAgentManagerConnectListRunsUsesBoundedDefault(t *testing.T) {
	handler, _ := setupTestHandler(t)
	router := mux.NewRouter()
	path, connectHandler := apiconnect.NewAgentManagerServiceHandler(NewAgentManagerConnectHandler(handler))
	connectx.RegisterServices(router, connectx.ServiceMount{Path: path, Handler: connectHandler})

	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := apiconnect.NewAgentManagerServiceClient(server.Client(), server.URL)

	response, err := client.ListRuns(context.Background(), connect.NewRequest(&apipb.ListRunsRequest{}))
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotNil(t, response.Msg)
}

type runReportServiceStub struct {
	report *runreport.RunReport
	err    error
}

func (s runReportServiceStub) BuildRunReport(context.Context, uuid.UUID) (*runreport.RunReport, error) {
	return s.report, s.err
}

func TestAgentManagerConnectGetRunReportProjectsBoundedReportAndNotFound(t *testing.T) {
	runID := uuid.New()
	h := &AgentManagerConnectHandler{h: &Handler{svc: orchestration.HandlerServices{
		RunReportService: runReportServiceStub{report: &runreport.RunReport{RunID: runID, Status: "complete", Tokens: 7}},
	}}}

	response, err := h.GetRunReport(context.Background(), connect.NewRequest(&apipb.GetRunReportRequest{RunId: runID.String()}))
	require.NoError(t, err)
	require.Equal(t, runID.String(), response.Msg.GetRunId())
	require.Equal(t, "complete", response.Msg.GetStatus())
	require.EqualValues(t, 7, response.Msg.GetTokens())

	missing := uuid.New()
	h.h.svc.RunReportService = runReportServiceStub{err: domain.NewNotFoundError("Run", missing)}
	_, err = h.GetRunReport(context.Background(), connect.NewRequest(&apipb.GetRunReportRequest{RunId: missing.String()}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}
