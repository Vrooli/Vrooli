package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"

	measurepb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures"
	measureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures/measures_v1connect"
)

type modeServer struct {
	measureconnect.UnimplementedMeasuresServiceHandler
	started  *measurepb.StartUsageModeRequest
	ended    *measurepb.EndUsageModeRequest
	extended *measurepb.ExtendUsageModeRequest
	rebased  *measurepb.RebaseUsageModeRunsRequest
	operator []bool
	denyEnd  bool
}

func (s *modeServer) StartUsageMode(_ context.Context, req *connect.Request[measurepb.StartUsageModeRequest]) (*connect.Response[measurepb.StartUsageModeResponse], error) {
	s.started = req.Msg
	return connect.NewResponse(&measurepb.StartUsageModeResponse{
		Mode: &measurepb.UsageMode{
			Id: "um-20261030-a1b2c3", StartedBy: "operator@example.com", UntilKind: "window_reset", UntilProvider: "openai", UntilWindowMinutes: 10080, ExpiresAt: "2026-11-04T06:57:00Z",
			Scope:    &measurepb.UsageModeScope{Runner: req.Msg.GetRunner()},
			Settings: []*measurepb.UsageModeSettings{{Runner: "codex", ModelMap: map[string]string{"gpt-6.1-terra": "gpt-6-luna"}, SessionModel: "gpt-6-luna", Effort: req.Msg.GetEffort()}},
		},
		ReplacedIds: []string{"um-old"},
	}), nil
}

func (s *modeServer) EndUsageMode(_ context.Context, req *connect.Request[measurepb.EndUsageModeRequest]) (*connect.Response[measurepb.EndUsageModeResponse], error) {
	if s.denyEnd {
		return nil, connect.NewError(connect.CodePermissionDenied, errString("usage modes can be started or ended only by a verified operator, not an agent"))
	}
	s.ended = req.Msg
	return connect.NewResponse(&measurepb.EndUsageModeResponse{EndedIds: []string{"um-20261030-a1b2c3"}, NotFoundIds: []string{"um-missing"}}), nil
}

func (s *modeServer) ExtendUsageMode(_ context.Context, req *connect.Request[measurepb.ExtendUsageModeRequest]) (*connect.Response[measurepb.ExtendUsageModeResponse], error) {
	s.extended = req.Msg
	return connect.NewResponse(&measurepb.ExtendUsageModeResponse{Mode: &measurepb.UsageMode{Id: req.Msg.GetId(), UntilKind: "time", ExpiresAt: "2026-10-31T00:00:00Z", Scope: &measurepb.UsageModeScope{}}}), nil
}

func (s *modeServer) RebaseUsageModeRuns(_ context.Context, req *connect.Request[measurepb.RebaseUsageModeRunsRequest]) (*connect.Response[measurepb.RebaseUsageModeRunsResponse], error) {
	s.rebased = req.Msg
	return connect.NewResponse(&measurepb.RebaseUsageModeRunsResponse{DryRun: req.Msg.GetDryRun(), Results: []*measurepb.UsageModeRebaseResult{{RunId: "run-1", Status: "switched_at_next_resume", FromModel: "terra", ToModel: "luna", EstimatedRereadTokens: 1200}}}), nil
}

func (s *modeServer) ListUsageModes(context.Context, *connect.Request[measurepb.ListUsageModesRequest]) (*connect.Response[measurepb.ListUsageModesResponse], error) {
	return connect.NewResponse(&measurepb.ListUsageModesResponse{
		GeneratedAt: "2026-10-30T18:00:00Z", ModeFile: "/home/op/.vrooli/state/vrooli/usage-mode/mode.json",
		Modes: []*measurepb.UsageModeStatus{{
			Mode:        &measurepb.UsageMode{Id: "um-1", Label: "Lower tier for codex", StartedBy: "operator@example.com", StartedAt: "2026-10-30T17:00:00Z", ExpiresAt: "2026-10-30T23:00:00Z", UntilKind: "time", Scope: &measurepb.UsageModeScope{Runner: "codex"}},
			SecondsLeft: 5 * 3600,
			Burn:        []*measurepb.UsageModeBurn{{Provider: "openai", Window: "weekly", ObservedIncreasePoints: 4.2, AttributedPoints: 3.9, UnattributedPoints: 0.3}},
		}},
		Reach:      []*measurepb.UsageModeReachRow{{Provider: "openai", Window: "weekly", ResetsAt: "2026-11-04T06:57:00Z", ObservedIncreasePoints: 10, OperatorPoints: 6, ProfilePoints: 3, UnattributedPoints: 1, OperatorShare: 0.6, ProfileShare: 0.3, UnattributedShare: 0.1}},
		ReachNotes: []string{"Desktop apps and harnesses started outside the shim cannot be reached"},
	}), nil
}

type errString string

func (e errString) Error() string { return string(e) }

func useModeServer(t *testing.T, server *modeServer) {
	t.Helper()
	path, handler := measureconnect.NewMeasuresServiceHandler(server)
	mux := httptest.NewServer(handlerMux(path, handler))
	t.Cleanup(func() { modeClientOverride = nil; mux.Close() })
	modeClientOverride = func(operator bool) measureconnect.MeasuresServiceClient {
		server.operator = append(server.operator, operator)
		return measureconnect.NewMeasuresServiceClient(mux.Client(), mux.URL)
	}
}

func TestUsageModeStartSendsEveryFlagAsOperator(t *testing.T) {
	server := &modeServer{}
	useModeServer(t, server)
	out := captureStdout(t, func() error {
		return (&App{}).usageMode([]string{"start", "--preset", "one-tier-down", "--runner", "codex", "--role", "code.default", "--profile", "delivery",
			"--model", "gpt-6-luna", "--effort", "low", "--context-limit", "150000", "--until", "weekly-reset", "--label", "stretch"})
	})
	want := &measurepb.StartUsageModeRequest{Preset: "one-tier-down", Runner: "codex", Role: "code.default", Profile: "delivery", Model: "gpt-6-luna", Effort: "low", ContextLimitTokens: 150000, Until: "weekly-reset", Label: "stretch"}
	if server.started == nil || server.started.String() != want.String() {
		t.Fatalf("request = %v, want %v", server.started, want)
	}
	if len(server.operator) != 1 || !server.operator[0] {
		t.Fatalf("start must use operator authority: %v", server.operator)
	}
	for _, text := range []string{"um-20261030-a1b2c3", "operator@example.com", "gpt-6.1-terra -> gpt-6-luna", "Replaced", "um-old", "next launch only", "desktop apps are unreachable"} {
		if !strings.Contains(out, text) {
			t.Errorf("start output lacks %q:\n%s", text, out)
		}
	}
}

func TestUsageModeStartJSONIsTheWireShape(t *testing.T) {
	useModeServer(t, &modeServer{})
	out := captureStdout(t, func() error {
		return (&App{}).usageMode([]string{"start", "--preset", "one-tier-down", "--until", "off", "--json"})
	})
	var decoded struct {
		Mode struct {
			ID        string `json:"id"`
			StartedBy string `json:"started_by"`
		} `json:"mode"`
		ReplacedIDs []string `json:"replaced_ids"`
	}
	if err := json.Unmarshal([]byte(out), &decoded); err != nil || decoded.Mode.ID != "um-20261030-a1b2c3" || decoded.Mode.StartedBy != "operator@example.com" || len(decoded.ReplacedIDs) != 1 {
		t.Fatalf("json output %q: %+v err=%v", out, decoded, err)
	}
}

func TestUsageModeStatusIsReadableAndSaysWhatIsUnreachable(t *testing.T) {
	server := &modeServer{}
	useModeServer(t, server)
	out := captureStdout(t, func() error { return (&App{}).usageMode([]string{"status"}) })
	for _, text := range []string{"um-1", "5h0m0s left", "estimate, not a saving", "+4.2 points", "operator sessions   60%", "profile runs", "unattributed", "change on next shim launch only", "cannot be reached"} {
		if !strings.Contains(out, text) {
			t.Errorf("status output lacks %q:\n%s", text, out)
		}
	}
	if len(server.operator) != 1 || server.operator[0] {
		t.Fatalf("status is a read and must not request operator authority: %v", server.operator)
	}
}

func TestUsageModeEndNeedsAnIDOrAll(t *testing.T) {
	server := &modeServer{}
	useModeServer(t, server)
	if err := (&App{}).usageMode([]string{"end"}); err == nil || !strings.Contains(err.Error(), "end <id> | --all") {
		t.Fatalf("end without target err = %v", err)
	}
	out := captureStdout(t, func() error { return (&App{}).usageMode([]string{"end", "um-20261030-a1b2c3", "um-missing"}) })
	if server.ended == nil || len(server.ended.GetIds()) != 2 || server.ended.GetAll() || !strings.Contains(out, "Not active: um-missing") {
		t.Fatalf("end by id: request=%v out=%s", server.ended, out)
	}
	captureStdout(t, func() error { return (&App{}).usageMode([]string{"end", "--all"}) })
	if !server.ended.GetAll() {
		t.Fatalf("--all not sent: %v", server.ended)
	}
}

func TestUsageModeExtendSendsIDAndTargetAsOperator(t *testing.T) {
	server := &modeServer{}
	useModeServer(t, server)
	if err := (&App{}).usageMode([]string{"extend", "um-1"}); err == nil {
		t.Fatal("extend without --until was accepted")
	}
	out := captureStdout(t, func() error { return (&App{}).usageMode([]string{"extend", "um-1", "--until", "2h"}) })
	if server.extended.GetId() != "um-1" || server.extended.GetTo() != "2h" || !strings.Contains(out, "Extended usage mode um-1") {
		t.Fatalf("extend: request=%v out=%s", server.extended, out)
	}
	if len(server.operator) != 1 || !server.operator[0] {
		t.Fatalf("extend needs operator authority: %v", server.operator)
	}
}

func TestUsageModeRebaseIsADryRunUntilConfirmed(t *testing.T) {
	server := &modeServer{}
	useModeServer(t, server)
	out := captureStdout(t, func() error { return (&App{}).usageMode([]string{"rebase", "um-1"}) })
	if server.rebased.GetModeId() != "um-1" || !server.rebased.GetDryRun() || !strings.Contains(out, "Dry run") || !strings.Contains(out, "terra -> luna") {
		t.Fatalf("dry run: request=%v out=%s", server.rebased, out)
	}
	captureStdout(t, func() error { return (&App{}).usageMode([]string{"rebase", "um-1", "--confirm"}) })
	if server.rebased.GetDryRun() {
		t.Fatalf("--confirm still sent dry_run: %v", server.rebased)
	}
	if err := (&App{}).usageMode([]string{"rebase"}); err == nil {
		t.Fatal("rebase without an id was accepted")
	}
}

func TestUsageModeShowsTheServersRefusalOfAnAgent(t *testing.T) {
	useModeServer(t, &modeServer{denyEnd: true})
	err := (&App{}).usageMode([]string{"end", "--all"})
	if err == nil || !strings.Contains(err.Error(), "verified operator, not an agent") {
		t.Fatalf("err = %v", err)
	}
}

func TestUsageModeRejectsUnknownSubcommandsAndPositionals(t *testing.T) {
	useModeServer(t, &modeServer{})
	if err := (&App{}).usageMode([]string{"pause"}); err == nil || !strings.Contains(err.Error(), "start, status, end, extend, rebase or ask") {
		t.Fatalf("unknown subcommand err = %v", err)
	}
	if err := (&App{}).usageMode([]string{"start", "extra", "--until", "1h"}); err == nil {
		t.Fatal("start accepted a positional argument")
	}
}

func handlerMux(path string, handler http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}
