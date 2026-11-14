package main

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	clitest "github.com/vrooli/cli-core/cliapptest"
	apipb "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api"
	apiconnect "github.com/vrooli/vrooli/packages/proto/gen/go/swarm-manager/v1/api/apiconnect"
)

type terminalArchiveService struct {
	apiconnect.UnimplementedBacklogServiceHandler
	requests []*apipb.ArchiveTerminalItemsRequest
}

func (s *terminalArchiveService) ArchiveTerminalItems(_ context.Context, req *connect.Request[apipb.ArchiveTerminalItemsRequest]) (*connect.Response[apipb.ArchiveTerminalItemsResponse], error) {
	s.requests = append(s.requests, req.Msg)
	return connect.NewResponse(&apipb.ArchiveTerminalItemsResponse{Executed: req.Msg.Execute}), nil
}

func TestTerminalArchiveCLIDefaultsToPreviewAndRequiresExplicitExecution(t *testing.T) {
	svc := &terminalArchiveService{}
	_, handler := apiconnect.NewBacklogServiceHandler(svc)
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" || r.URL.Path == "/api/v1/health" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"healthy"}`))
			return
		}
		handler.ServeHTTP(w, r)
	}))
	app := newAppT(t)
	items := `[{"kind":"execute","name":"done","expected_status":"completed","expected_updated":"reviewed"}]`
	for _, flags := range [][]string{{"--items", items, "--json"}, {"--items", items, "--execute", "--json"}} {
		clitest.CaptureStdout(t, func() error { return app.Run(append([]string{"backlog", "archive-terminal"}, flags...)) })
	}
	if len(svc.requests) != 2 || svc.requests[0].Execute || !svc.requests[1].Execute || svc.requests[1].Items[0].ExpectedUpdated != "reviewed" {
		t.Fatalf("preview/execution contract = %+v", svc.requests)
	}
	if err := app.Run([]string{"backlog", "archive-terminal", "--items", "broken", "--execute"}); err == nil {
		t.Fatal("invalid selection accepted")
	}
	if len(svc.requests) != 2 {
		t.Fatal("invalid selection reached domain transport")
	}
}
