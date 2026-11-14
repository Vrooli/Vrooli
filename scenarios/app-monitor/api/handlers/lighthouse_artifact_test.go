package handlers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/gin-gonic/gin"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	runsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

func TestGetLighthouseReportResolvesCatalogIDThroughMetadata(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const transferPath = "/api/v1/scenarios/demo/runs/run-1/artifacts/artifact-1"
	var metadataCalls, transferCalls int
	var metadataRequest *runspb.GetRunArtifactRequest
	mux := http.NewServeMux()
	mux.Handle(runsconnect.RunsServiceListRunsProcedure, connect.NewUnaryHandler(
		runsconnect.RunsServiceListRunsProcedure,
		func(context.Context, *connect.Request[runspb.ListRunsRequest]) (*connect.Response[runspb.ListRunsResponse], error) {
			return connect.NewResponse(&runspb.ListRunsResponse{Runs: []*runspb.RunInfo{{RunId: "run-1"}}}), nil
		},
	))
	mux.Handle(runsconnect.RunsServiceListRunArtifactsProcedure, connect.NewUnaryHandler(
		runsconnect.RunsServiceListRunArtifactsProcedure,
		func(context.Context, *connect.Request[runspb.ListRunArtifactsRequest]) (*connect.Response[runspb.ListRunArtifactsResponse], error) {
			// The catalog intentionally contains no access path.
			return connect.NewResponse(&runspb.ListRunArtifactsResponse{Artifacts: []*runspb.ArtifactRef{{Id: "artifact-1", Label: "report-123.html", MediaType: "text/html"}}}), nil
		},
	))
	mux.Handle(runsconnect.RunsServiceGetRunArtifactProcedure, connect.NewUnaryHandler(
		runsconnect.RunsServiceGetRunArtifactProcedure,
		func(_ context.Context, req *connect.Request[runspb.GetRunArtifactRequest]) (*connect.Response[runspb.GetRunArtifactResponse], error) {
			metadataCalls++
			metadataRequest = req.Msg
			return connect.NewResponse(&runspb.GetRunArtifactResponse{Artifact: &runspb.ArtifactRef{
				Id: "artifact-1", AccessPath: transferPath, MediaType: "text/html",
			}}), nil
		},
	))
	mux.HandleFunc(transferPath, func(w http.ResponseWriter, _ *http.Request) {
		transferCalls++
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, "<html>lighthouse report</html>")
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	handler := &LighthouseHandler{runsClientFactory: func(context.Context) (runsconnect.RunsServiceClient, string, error) {
		return runsconnect.NewRunsServiceClient(server.Client(), server.URL), server.URL, nil
	}}
	router := gin.New()
	router.GET("/scenarios/:scenario/lighthouse/report/:reportId", handler.GetLighthouseReport)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/scenarios/demo/lighthouse/report/report-123", nil)
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if got := response.Header().Get("Content-Type"); got != "text/html" {
		t.Fatalf("Content-Type = %q, want text/html", got)
	}
	if got := response.Body.String(); got != "<html>lighthouse report</html>" {
		t.Fatalf("body = %q", got)
	}
	if metadataCalls != 1 {
		t.Fatalf("GetRunArtifact calls = %d, want 1", metadataCalls)
	}
	if metadataRequest.GetTarget() != "demo" || metadataRequest.GetRunId() != "run-1" || metadataRequest.GetArtifactId() != "artifact-1" {
		t.Fatalf("GetRunArtifact request = %+v", metadataRequest)
	}
	if transferCalls != 1 {
		t.Fatalf("artifact transfer calls = %d, want 1", transferCalls)
	}
}
