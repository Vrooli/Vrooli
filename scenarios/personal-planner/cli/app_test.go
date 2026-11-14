package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
	focusv1 "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/focus"
	focusconnect "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/focus/focus_v1connect"
	goalsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/goals"
	goalsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/goals/goals_v1connect"
	workv1 "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/work"
	workconnect "github.com/vrooli/vrooli/packages/proto/gen/go/personal-planner/v1/work/work_v1connect"
)

// TestNewAppConstructs is the smoke gate: NewApp() must succeed against
// the cli-core wiring declared in app.go. This catches the most common
// regression class — a misconfigured StandardScenarioOptions or a missing
// dependency from cli-core — before any tests touch real commands.
func TestNewAppConstructs(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if app == nil || app.core == nil || app.core.CLI == nil {
		t.Fatal("NewApp() returned an incomplete app")
	}
}

// TestRunVersion exercises a non-API command path through cli-core.
// --version must succeed and must NOT trigger the NeedsAPI preflight
// (which would try to reach the configured API base and fail in CI).
func TestRunVersion(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if err := app.Run([]string{"--version"}); err != nil {
		t.Fatalf("app.Run(--version) error: %v", err)
	}
}

// TestRunHelp exercises cli-core's help renderer through the scenario
// app's wiring. Help is rendered to stdout by cli-core; we only verify
// Run returns without error. The presence of each registered command in
// the actual help surface is covered by cli-core's own tests.
func TestRunHelp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if err := app.Run([]string{"--help"}); err != nil {
		t.Fatalf("app.Run(--help) error: %v", err)
	}
}

// TestMetadata pins the values app.go declares — appName must match the
// scenario id (post-substitution), appVersion must be non-empty.
// Catches accidental edits that decouple the binary identity from the
// scenario it belongs to.
func TestMetadata(t *testing.T) {
	if strings.TrimSpace(appName) == "" {
		t.Fatal("appName must not be empty")
	}
	if strings.TrimSpace(appVersion) == "" {
		t.Fatal("appVersion must not be empty")
	}
}

func TestGeneratedConnectClientsRouteTestModeHeaderForGoalsWorkAndFocus(t *testing.T) {
	for _, testMode := range []struct {
		name   string
		enable bool
	}{
		{name: "enabled", enable: true},
		{name: "default off"},
	} {
		t.Run(testMode.name, func(t *testing.T) {
			if testMode.enable {
				t.Setenv("PERSONAL_PLANNER_TEST_MODE", "1")
			} else {
				t.Setenv("PERSONAL_PLANNER_TEST_MODE", "")
			}
			var mu sync.Mutex
			received := map[string]http.Header{}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				received[r.URL.Path] = r.Header.Clone()
				mu.Unlock()
				w.Header().Set("Content-Type", "application/proto")
			}))
			defer server.Close()

			t.Setenv("PERSONAL_PLANNER_API_BASE", server.URL)
			app, err := NewApp()
			if err != nil {
				t.Fatalf("NewApp() error: %v", err)
			}
			if got := strings.TrimRight(app.core.APIRootBase(), "/"); got != server.URL {
				t.Fatalf("API root %q does not resolve to test server %q", got, server.URL)
			}
			unrelatedClient := cliutil.NewHTTPClient(cliutil.HTTPClientOptions{})
			unrelatedRequest := httptest.NewRequest(http.MethodPost, server.URL, nil)
			unrelatedClient.ApplyRequestHeaders(unrelatedRequest)
			if got := unrelatedRequest.Header.Get("X-Vrooli-Test-Mode"); got != "" {
				t.Fatalf("test-mode header leaked to unrelated HTTP client: %q", got)
			}
			client, base := cliapp.NewConnectHTTPClient(app.core)
			identity := "synthetic-preserved-identity"
			call := func(path string, invoke func() error) {
				t.Helper()
				if err := invoke(); err != nil {
					t.Fatalf("Connect call %s: %v", path, err)
				}
				mu.Lock()
				var headers http.Header
				var gotPath string
				for requestPath, requestHeaders := range received {
					if strings.HasSuffix(requestPath, path) {
						headers = requestHeaders
						gotPath = requestPath
						break
					}
				}
				mu.Unlock()
				if headers == nil {
					t.Fatalf("server did not receive %s; received paths: %v", path, received)
				}
				wantTestMode := ""
				if testMode.enable {
					wantTestMode = "1"
				}
				if got := headers.Get("X-Vrooli-Test-Mode"); got != wantTestMode {
					t.Errorf("%s test-mode header = %q, want %q", gotPath, got, wantTestMode)
				}
				if got := headers.Get(cliutil.HeaderAgentIdentityToken); got != identity {
					t.Errorf("%s identity header = %q, want preserved request value", gotPath, got)
				}
			}
			goals := goalsconnect.NewGoalsServiceClient(client, base)
			goalsReq := connect.NewRequest(&goalsv1.ListGoalsRequest{})
			goalsReq.Header().Set(cliutil.HeaderAgentIdentityToken, identity)
			call(goalsconnect.GoalsServiceListGoalsProcedure, func() error {
				_, err := goals.ListGoals(context.Background(), goalsReq)
				return err
			})

			work := workconnect.NewWorkServiceClient(client, base)
			workReq := connect.NewRequest(&workv1.ListWorkItemsRequest{})
			workReq.Header().Set(cliutil.HeaderAgentIdentityToken, identity)
			call(workconnect.WorkServiceListWorkItemsProcedure, func() error {
				_, err := work.ListWorkItems(context.Background(), workReq)
				return err
			})

			focus := focusconnect.NewFocusServiceClient(client, base)
			focusReq := connect.NewRequest(&focusv1.ListActualsRequest{LocalDate: "2026-10-19"})
			focusReq.Header().Set(cliutil.HeaderAgentIdentityToken, identity)
			call(focusconnect.FocusServiceListActualsProcedure, func() error {
				_, err := focus.ListActuals(context.Background(), focusReq)
				return err
			})
		})
	}
}
