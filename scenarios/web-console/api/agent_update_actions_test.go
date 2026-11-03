package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/identity"
	"github.com/vrooli/api-core/provenance"
	"github.com/vrooli/api-core/targetmodel"
	harnesslifecycle "github.com/vrooli/vrooli/packages/harness-lifecycle"
	relayv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay"
	relayconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/relay/relayv1connect"
	capsH "web-console/handlers/capabilities"
)

func operatorContext() context.Context {
	return identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorHuman, Subject: "op@example.com", Verified: true, ExpiresAt: time.Now().Add(time.Hour)})
}

// fakeHarnessLifecycle supplies a typed lifecycle unit-test seam. The browser
// journeys use the real shared lifecycle owner instead.
func fakeHarnessLifecycle(t *testing.T) (*Server, string, string) {
	t.Helper()
	dir := t.TempDir()
	log, version := filepath.Join(dir, "actions"), filepath.Join(dir, "version")
	if err := os.WriteFile(version, []byte("0.165.0"), 0o600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(dir, "codex")
	if err := os.WriteFile(binary, []byte(fmt.Sprintf("#!/bin/sh\necho \"codex-cli $(cat %q)\"\n", version)), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+":/usr/bin:/bin")
	inspect := func(context.Context, string, string) (harnesslifecycle.HarnessReport, error) {
		data, err := os.ReadFile(version)
		current := strings.TrimSpace(string(data))
		state, primary := "update_ready", "update"
		if current == "0.166.0" {
			state, primary = "up_to_date", ""
		}
		return harnesslifecycle.HarnessReport{Name: "codex", State: state, PrimaryAction: primary, Installed: current, Latest: "0.166.0", CommandPath: binary, OS: runtime.GOOS, Arch: runtime.GOARCH, CheckedAt: time.Now(), Source: harnesslifecycle.HarnessSource{Kind: "npm", ID: "@openai/codex"}}, err
	}
	srv := &Server{agentLifecycle: localHarnessLifecycle{inspect: inspect}}
	srv.agentLifecycle.manage = func(ctx context.Context, _, resource, action, actor, _ string) (harnesslifecycle.HarnessActionResult, error) {
		if err := ctx.Err(); err != nil {
			return harnesslifecycle.HarnessActionResult{}, err
		}
		_ = os.WriteFile(log, []byte(resource+" "+action+" "+actor), 0o600)
		success, message := true, "done"
		if _, err := os.Stat(version + ".fail"); err == nil {
			success, message = false, "refused"
		} else if err := os.WriteFile(version, []byte("0.166.0"), 0o600); err != nil {
			return harnesslifecycle.HarnessActionResult{}, err
		}
		report, err := inspect(ctx, "", resource)
		return harnesslifecycle.HarnessActionResult{Action: action, Success: success, Message: message, Report: report}, err
	}
	return srv, log, version
}

func TestUpdateActionRunsTheVerbInvalidatesTheCacheAndReturnsAFreshObservation(t *testing.T) {
	s, log, _ := fakeHarnessLifecycle(t)
	before, err := s.getAgentUpdate(context.Background(), "local", "codex")
	if err != nil || before.State != agentStateUpdateReady {
		t.Fatalf("before = %+v, %v", before, err)
	}
	result, err := s.runAgentUpdateAction(operatorContext(), "local", "codex", "update", "")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Success || result.Message != "done" || result.Update.State != agentStateUpToDate || result.Update.Installed != "0.166.0" {
		t.Fatalf("result = %+v", result)
	}
	argv, _ := os.ReadFile(log)
	if !strings.Contains(string(argv), "codex update op@example.com") {
		t.Fatalf("argv = %s", argv)
	}
}

func TestFailedActionKeepsTheVerbsReasonAndAFreshObservation(t *testing.T) {
	s, _, version := fakeHarnessLifecycle(t)
	if err := os.WriteFile(version+".fail", []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := s.runAgentUpdateAction(operatorContext(), "local", "codex", "update", "")
	if err != nil {
		t.Fatal(err)
	}
	if result.Success || result.Message != "refused" || result.Update.State != agentStateUpdateReady {
		t.Fatalf("result = %+v", result)
	}
}

func TestActionsMapToExactCommandsAndNothingFromTheRequestReachesArgv(t *testing.T) {
	want := map[string]string{
		"install":           "resource update|codex --action install --json",
		"update":            "resource update|codex --json",
		"use_managed_copy":  "resource update|codex --action use-managed-copy --json",
		"link_managed_copy": "resource update|codex --action link-managed-copy --json",
		"rollback":          "resource update|codex --action rollback --json",
	}
	for action, expected := range want {
		command, args, ok := agentUpdateActionArgs(action, "codex")
		if !ok || command+"|"+strings.Join(args, " ") != expected {
			t.Fatalf("%s = %q %v", action, command, args)
		}
	}
	if _, _, ok := agentUpdateActionArgs("--action=rollback; rm -rf /", "codex"); ok {
		t.Fatal("an unknown action must have no command")
	}
	if _, err := (&Server{}).runAgentUpdateAction(operatorContext(), "local", "codex", "unknown", ""); err == nil {
		t.Fatal("an unknown action must be refused")
	}
	if _, err := (&Server{}).runAgentUpdateAction(operatorContext(), "local", "not-an-agent", "update", ""); err == nil {
		t.Fatal("an unknown agent must be refused")
	}
}

func TestAgentsAreRefusedTheActionEvenWithAForwardedHumanCredential(t *testing.T) {
	s, log, _ := fakeHarnessLifecycle(t)
	human := identity.Principal{Kind: identity.ActorHuman, Subject: "op@example.com", Verified: true, ExpiresAt: time.Now().Add(time.Hour)}
	cases := map[string]context.Context{
		"anonymous": context.Background(),
		"agent":     identity.WithPrincipal(context.Background(), identity.Principal{Kind: identity.ActorAgent, Subject: "agent", Verified: true, ExpiresAt: time.Now().Add(time.Hour)}),
		"forwarded": provenance.NewContext(identity.WithPrincipal(context.Background(), human), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run-1"}),
	}
	for name, ctx := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := s.runAgentUpdateAction(ctx, "local", "codex", "update", "")
			if !errors.Is(err, capsH.ErrPermissionDenied) {
				t.Fatalf("err = %v", err)
			}
			if data, _ := os.ReadFile(log); len(data) != 0 {
				t.Fatalf("the verb ran: %s", data)
			}
		})
		r := httptest.NewRequest(http.MethodPost, "/rpc", nil).WithContext(ctx)
		r.RemoteAddr = "127.0.0.1:1000"
		recorder := httptest.NewRecorder()
		if guardAgentUpdateAction(recorder, r) || recorder.Code != http.StatusForbidden {
			t.Fatalf("%s passed the guard: %d", name, recorder.Code)
		}
	}
}

func TestGuardRefusesCrossOriginAndAdmitsTheOperator(t *testing.T) {
	request := func(origin string) *http.Request {
		r := httptest.NewRequest(http.MethodPost, "http://console.example/rpc", nil).WithContext(operatorContext())
		r.Host, r.RemoteAddr = "console.example", "10.0.0.8:1000"
		if origin != "" {
			r.Header.Set("Origin", origin)
		}
		return r
	}
	if recorder := httptest.NewRecorder(); guardAgentUpdateAction(recorder, request("https://evil.example")) || recorder.Code != http.StatusForbidden {
		t.Fatalf("cross-origin admitted: %d", recorder.Code)
	}
	if recorder := httptest.NewRecorder(); guardAgentUpdateAction(recorder, request("")) || recorder.Code != http.StatusForbidden {
		t.Fatalf("remote request without an Origin admitted: %d", recorder.Code)
	}
	if !guardAgentUpdateAction(httptest.NewRecorder(), request("http://console.example")) {
		t.Fatal("the operator's same-origin request must pass")
	}
}

func TestConcurrentActionsOnOneHarnessShareOrRefuseAndOtherTargetsAreIndependent(t *testing.T) {
	var actions agentUpdateActions
	release := make(chan struct{})
	var runs atomic.Int32
	run := func() (capsH.AgentUpdateActionResult, error) {
		runs.Add(1)
		<-release
		return capsH.AgentUpdateActionResult{Success: true, Message: "ran"}, nil
	}
	key := agentUpdateActionKey("local", "codex")
	first, err := actions.join(key, "update", run)
	if err != nil {
		t.Fatal(err)
	}
	second, err := actions.join(key, "update", run)
	if err != nil || second != first {
		t.Fatalf("a repeated request must rejoin: %v", err)
	}
	if _, err := actions.join(key, "rollback", run); !errors.Is(err, capsH.ErrFailedPrecondition) {
		t.Fatalf("a different action must be refused: %v", err)
	}
	other, err := actions.join(agentUpdateActionKey("bridge-node:one", "codex"), "update", run)
	if err != nil || other == first {
		t.Fatalf("another target is independent: %v", err)
	}
	if !actions.running(key) {
		t.Fatal("the action is running")
	}
	close(release)
	<-first.done
	<-other.done
	if runs.Load() != 2 || first.result.Message != "ran" {
		t.Fatalf("runs=%d result=%+v", runs.Load(), first.result)
	}
	if actions.running(key) {
		t.Fatal("a finished action must not block the next")
	}
}

// Closing the tab or losing the tunnel must not kill an install midway.
func TestActionOutlivesItsRequest(t *testing.T) {
	s, log, version := fakeHarnessLifecycle(t)
	ctx, cancel := context.WithCancel(operatorContext())
	cancel()
	if _, err := s.runAgentUpdateAction(ctx, "local", "codex", "update", ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for s.agentUpdateActions.running(agentUpdateActionKey("local", "codex")) && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if got, _ := os.ReadFile(version); !strings.Contains(string(got), "0.166.0") {
		data, _ := os.ReadFile(log)
		t.Fatalf("version = %s, argv = %s", got, data)
	}
}

type actionRelayFake struct {
	agentUpdateRelayFake
	requests []*relayv1.RelayCallRequest
	data     func(*relayv1.RelayCallRequest) ([]byte, relayv1.RelayCallOutcome, int32)
}

func (f *actionRelayFake) Call(_ context.Context, req *connect.Request[relayv1.RelayCallRequest]) (*connect.Response[relayv1.RelayCallResponse], error) {
	f.requests = append(f.requests, req.Msg)
	data, outcome, code := f.data(req.Msg)
	return connect.NewResponse(&relayv1.RelayCallResponse{Outcome: outcome, Data: data, ExitCode: code, CorrelationId: "corr-1"}), nil
}

func TestRemoteActionRelaysTypedArgvAndKeepsTheOperationID(t *testing.T) {
	fake := &actionRelayFake{}
	fake.data = func(*relayv1.RelayCallRequest) ([]byte, relayv1.RelayCallOutcome, int32) {
		return []byte(`{"success":false,"harness_update":{"action":"rollback","success":false,"message":"no previous version is installed","report":{}}}`), relayv1.RelayCallOutcome_RELAY_CALL_OUTCOME_FAILED, 1
	}
	_, handler := relayconnect.NewRelayServiceHandler(fake)
	server := httptest.NewServer(handler)
	defer server.Close()
	target := targetConnection{Target: targetmodel.Target{NodeID: "node-one", Label: "Node"}, BaseURL: server.URL, OwnerToken: "Bearer test"}
	outcome, err := (&Server{}).invokeAgentUpdateCommand(context.Background(), target, "bridge-node:one", "resource update", []string{"codex", "--action", "rollback", "--json"}, "codex", "rollback", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Success || outcome.Message != "no previous version is installed" || outcome.OperationID != "corr-1" {
		t.Fatalf("outcome = %+v", outcome)
	}
	request := fake.requests[0]
	if request.GetNodeId() != "node-one" || request.GetCommand() != "resource update" || strings.Join(request.GetArgs(), " ") != "codex --action rollback --json" || request.GetTimeoutSeconds() != int64(harnessActionWindow.Seconds()) {
		t.Fatalf("request = %+v", request)
	}
}

func TestHarnessActionBudgetContainsItsParts(t *testing.T) {
	if harnessActionWindow+harnessActionConfirmationWindow > harnessActionBudget {
		t.Fatalf("window %s + confirmation %s exceeds the budget %s", harnessActionWindow, harnessActionConfirmationWindow, harnessActionBudget)
	}
	if harnessActionResponseDeadline <= harnessActionBudget {
		t.Fatalf("the response deadline %s must outlast the budget %s", harnessActionResponseDeadline, harnessActionBudget)
	}
	// The verb bounds itself at six minutes so a repair can still restore what
	// it moved; the console must not kill it first.
	if harnessActionWindow <= 6*time.Minute {
		t.Fatalf("window %s must exceed the verb's own six-minute bound", harnessActionWindow)
	}
}

func TestRefreshCatalogIsAControlPlaneVerbOnThisMachineOnly(t *testing.T) {
	command, args, ok := agentUpdateActionArgs(agentActionRefreshCatalog, "claude-code")
	if !ok || command != "resource update" || !slices.Equal(args, []string{"claude-code", "--action", "refresh-catalog", "--json"}) {
		t.Fatalf("command %q %v %v", command, args, ok)
	}
}
