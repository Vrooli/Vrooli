package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	checksproto "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/checks"
	checksconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-autoheal/v1/checks/checks_v1connect"
)

type runCheckServer struct {
	checksconnect.UnimplementedChecksServiceHandler
	calls int
	id    string
}

func (s *runCheckServer) RunCheck(_ context.Context, r *connect.Request[checksproto.RunCheckRequest]) (*connect.Response[checksproto.RunCheckResponse], error) {
	s.calls++
	s.id = r.Msg.GetCheckId()
	return connect.NewResponse(&checksproto.RunCheckResponse{Result: &checksproto.CheckResult{CheckId: s.id, Status: checksproto.CheckStatus_CHECK_STATUS_OK, Message: "fresh observation"}}), nil
}

func TestCLICheckRunUsesTypedRequestInHumanAndJSONModes(t *testing.T) {
	for _, jsonMode := range []bool{false, true} {
		s := &runCheckServer{}
		path, handler := checksconnect.NewChecksServiceHandler(s)
		mux := http.NewServeMux()
		mux.Handle(path, handler)
		mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"status":"healthy","readiness":true}`))
		})
		ts := httptest.NewServer(mux)
		app, err := NewApp()
		if err != nil {
			t.Fatal(err)
		}
		app.core.APIOverride = ts.URL
		app.core.CLI.SetStaleChecker(nil)
		args := []string{"check", "run", "system-boot-recovery-readiness"}
		if jsonMode {
			args = append(args, "--json")
		}
		out := captureStdout(t, func() {
			if err := app.Run(args); err != nil {
				t.Fatal(err)
			}
		})
		ts.Close()
		if s.calls != 1 || s.id != "system-boot-recovery-readiness" || !strings.Contains(out, "fresh observation") {
			t.Fatalf("calls=%d id=%s output=%s", s.calls, s.id, out)
		}
	}
}
