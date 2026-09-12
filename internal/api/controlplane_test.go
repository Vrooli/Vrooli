package api

import (
	"context"
	"errors"
	"fmt"
	scenarioapp "github.com/vrooli/vrooli/internal/app/scenario"
	"github.com/vrooli/vrooli/internal/lifecycle"
	"github.com/vrooli/vrooli/internal/orchestrator"
	"github.com/vrooli/vrooli/internal/process"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
	cliv1connect "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1/cliv1connect"
)

func TestScenarioStatusMapsNewerRegistryToActionablePrecondition(t *testing.T) {
	cause := &scenarioruntime.SchemaCompatibilityError{DatabaseVersion: 8, BinaryVersion: 7}
	err := controlPlaneScenarioStatusError(cause)
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("error code = %s, want failed_precondition; error = %v", connect.CodeOf(err), err)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("error = %v, want wrapped schema compatibility cause", err)
	}
	for _, fragment := range []string{"schema_version 8", "supported 7", "vrooli develop"} {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("error = %q, want actionable fragment %q", err, fragment)
		}
	}
}

func TestScenarioControlPlaneServiceIsMounted(t *testing.T) {
	app := New(ResolveRepoRoot(), t.TempDir())
	server := httptest.NewServer(app.Router())
	defer server.Close()
	client := cliv1connect.NewScenarioControlPlaneServiceClient(server.Client(), server.URL)

	list, err := client.ListScenarios(context.Background(), connect.NewRequest(&cliv1.ListScenariosRequest{}))
	if err != nil {
		t.Fatalf("ListScenarios: %v", err)
	}
	if list.Msg.GetObservedAt() == nil || list.Msg.GetObservedAt().AsTime().IsZero() {
		t.Fatalf("ListScenarios observed_at = %v, want producer timestamp", list.Msg.GetObservedAt())
	}
	requests := []struct {
		name string
		call func() error
	}{
		{name: "GetScenarioStatus", call: func() error {
			_, err := client.GetScenarioStatus(context.Background(), connect.NewRequest(&cliv1.GetScenarioStatusRequest{}))
			return err
		}},
		{name: "GetScenarioLogs", call: func() error {
			_, err := client.GetScenarioLogs(context.Background(), connect.NewRequest(&cliv1.GetScenarioLogsRequest{}))
			return err
		}},
		{name: "StartScenario", call: func() error {
			_, err := client.StartScenario(context.Background(), connect.NewRequest(&cliv1.StartScenarioRequest{}))
			return err
		}},
		{name: "StopScenario", call: func() error {
			_, err := client.StopScenario(context.Background(), connect.NewRequest(&cliv1.StopScenarioRequest{}))
			return err
		}},
		{name: "RestartScenario", call: func() error {
			_, err := client.RestartScenario(context.Background(), connect.NewRequest(&cliv1.RestartScenarioRequest{}))
			return err
		}},
		{name: "SetupScenario", call: func() error {
			_, err := client.SetupScenario(context.Background(), connect.NewRequest(&cliv1.SetupScenarioRequest{}))
			return err
		}},
	}
	for _, request := range requests {
		t.Run(request.name, func(t *testing.T) {
			err := request.call()
			if connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Fatalf("error code = %s, want invalid_argument; error = %v", connect.CodeOf(err), err)
			}
		})
	}
}

// Exercise the wire boundary without starting host processes.
type ceilingScenarioOps struct {
	scenarioapp.ScenarioOperations
	observed      chan context.Context
	awaitDeadline bool
	options       chan lifecycle.StartOptions
}

func (f *ceilingScenarioOps) StartDetailed(_ string, opts lifecycle.StartOptions) (orchestrator.StartResult, error) {
	if f.options != nil {
		f.options <- opts
	}
	if f.observed != nil {
		f.observed <- opts.Context
	}
	if f.awaitDeadline {
		<-opts.Context.Done()
		return orchestrator.StartResult{}, opts.Context.Err()
	}
	return orchestrator.StartResult{}, errors.New("test operation ended")
}
func (f *ceilingScenarioOps) RestartDetailed(name string, opts lifecycle.StartOptions) (orchestrator.StartResult, error) {
	return f.StartDetailed(name, opts)
}
func TestScenarioLifecycleRPCCeiling(t *testing.T) {
	for _, action := range []string{"start", "restart"} {
		for _, seconds := range []int32{-1, 0, 30} {
			t.Run(fmt.Sprintf("%s/%d", action, seconds), func(t *testing.T) {
				ops := &ceilingScenarioOps{observed: make(chan context.Context, 1)}
				_, handler := cliv1connect.NewScenarioControlPlaneServiceHandler(&scenarioControlPlaneHandler{lifecycleOperations: ops})
				server := httptest.NewServer(handler)
				defer server.Close()
				client := cliv1connect.NewScenarioControlPlaneServiceClient(server.Client(), server.URL)
				before := time.Now()
				var err error
				if action == "start" {
					_, err = client.StartScenario(context.Background(), connect.NewRequest(&cliv1.StartScenarioRequest{Name: "test", TimeoutSeconds: seconds}))
				} else {
					_, err = client.RestartScenario(context.Background(), connect.NewRequest(&cliv1.RestartScenarioRequest{Name: "test", TimeoutSeconds: seconds}))
				}
				if seconds < 0 {
					if connect.CodeOf(err) != connect.CodeInvalidArgument {
						t.Fatalf("error = %v", err)
					}
					if len(ops.observed) != 0 {
						t.Fatal("invalid timeout started an operation")
					}
					return
				}
				if err == nil {
					t.Fatal("expected operation error")
				}
				select {
				case ctx := <-ops.observed:
					if seconds == 0 {
						if ctx != nil {
							t.Fatal("zero ceiling changed the default context")
						}
						return
					}
					deadline, ok := ctx.Deadline()
					if !ok || deadline.Before(before.Add(30*time.Second)) || deadline.After(time.Now().Add(30*time.Second)) {
						t.Fatalf("deadline = %v, present=%v", deadline, ok)
					}
					if ctx.Err() != context.Canceled {
						t.Fatalf("operation context not released: %v", ctx.Err())
					}
				default:
					t.Fatal("operation not called")
				}
			})
		}
	}
}

func TestScenarioLifecycleRPCCeilingExpires(t *testing.T) {
	for _, action := range []string{"start", "restart"} {
		t.Run(action, func(t *testing.T) {
			t.Parallel()
			ops := &ceilingScenarioOps{observed: make(chan context.Context, 1), awaitDeadline: true}
			_, handler := cliv1connect.NewScenarioControlPlaneServiceHandler(&scenarioControlPlaneHandler{lifecycleOperations: ops})
			server := httptest.NewServer(handler)
			defer server.Close()
			client := cliv1connect.NewScenarioControlPlaneServiceClient(server.Client(), server.URL)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			var err error
			if action == "start" {
				_, err = client.StartScenario(ctx, connect.NewRequest(&cliv1.StartScenarioRequest{Name: "test", TimeoutSeconds: 1}))
			} else {
				_, err = client.RestartScenario(ctx, connect.NewRequest(&cliv1.RestartScenarioRequest{Name: "test", TimeoutSeconds: 1}))
			}
			if connect.CodeOf(err) != connect.CodeDeadlineExceeded {
				t.Fatalf("error = %v", err)
			}
			if ctx.Err() != nil {
				t.Fatal("client ceiling expired before server operation")
			}
			operationCtx := <-ops.observed
			if operationCtx.Err() != context.DeadlineExceeded {
				t.Fatalf("operation error = %v", operationCtx.Err())
			}
		})
	}
}

func TestLogSnapshotBounds(t *testing.T) {
	app := &App{}
	path := filepath.Join(t.TempDir(), "log")
	for _, tc := range []struct {
		name, content, lines, want string
		failure                    error
	}{
		{name: "empty", lines: "1"},
		{name: "crlf", content: "one\r\ntwo\r\n", lines: "1", want: "two"},
		{name: "unterminated", content: "one\ntwo", lines: "1", want: "two"},
		{name: "blank line", content: "one\n\n", lines: "1", want: ""},
		{name: "oversized line", content: strings.Repeat("x", maxLogSnapshotBytes+2), lines: "1", failure: errLogSnapshotTooLarge},
		{name: "small suffix", content: strings.Repeat("x", maxLogSnapshotBytes+2) + "\nlast\n", lines: "1", want: "last"},
		{name: "incomplete requested prefix", content: strings.Repeat("x", maxLogSnapshotBytes+2) + "\nlast\n", lines: "2", failure: errLogSnapshotTooLarge},
		{name: "negative", lines: "-1", failure: errLogSnapshotLines},
		{name: "too many", lines: "10001", failure: errLogSnapshotLines},
		{name: "invalid", lines: "many", failure: errLogSnapshotLines},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := os.WriteFile(path, []byte(tc.content), 0600); err != nil {
				t.Fatal(err)
			}
			got, err := app.readTail(path, tc.lines)
			if !errors.Is(err, tc.failure) {
				t.Fatalf("error=%v, want %v", err, tc.failure)
			}
			if got != tc.want {
				t.Fatalf("output bytes=%d, want %q", len(got), tc.want)
			}
		})
	}
}

func TestLogSnapshotReadsLargeFileSuffix(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.log")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	// Sparse file makes whole-file allocation a regression without costly IO.
	if err := f.Truncate(1 << 30); err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteAt([]byte("\nlast\n"), (1<<30)-6); err != nil {
		t.Fatal(err)
	}
	got, err := (&App{}).readTail(path, "1")
	if err != nil || got != "last" {
		t.Fatalf("tail=%q, err=%v", got, err)
	}
}

func TestLogRPCRejectsInvalidLimitBeforeLookup(t *testing.T) {
	h := &scenarioControlPlaneHandler{}
	for _, limit := range []int32{-1, 10001} {
		_, err := h.GetScenarioLogs(context.Background(), connect.NewRequest(&cliv1.GetScenarioLogsRequest{Name: "test", TailLines: limit}))
		if connect.CodeOf(err) != connect.CodeInvalidArgument {
			t.Fatalf("limit=%d err=%v", limit, err)
		}
	}
}

func TestLifecycleRPCForwardsCLIOptions(t *testing.T) {
	for _, action := range []string{"start", "restart"} {
		for _, enabled := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", action, enabled), func(t *testing.T) {
				ops := &ceilingScenarioOps{options: make(chan lifecycle.StartOptions, 1)}
				_, handler := cliv1connect.NewScenarioControlPlaneServiceHandler(&scenarioControlPlaneHandler{lifecycleOperations: ops})
				server := httptest.NewServer(handler)
				defer server.Close()
				client := cliv1connect.NewScenarioControlPlaneServiceClient(server.Client(), server.URL)
				var err error
				if action == "start" {
					_, err = client.StartScenario(context.Background(), connect.NewRequest(&cliv1.StartScenarioRequest{Name: "fixture", Path: "/tmp/fixture", BestEffort: enabled, CleanStale: enabled, Force: enabled, AcceptCredentialLoss: enabled, DemandManaged: enabled}))
				} else {
					_, err = client.RestartScenario(context.Background(), connect.NewRequest(&cliv1.RestartScenarioRequest{Name: "fixture", Path: "/tmp/fixture", BestEffort: enabled, CleanStale: enabled, Force: enabled, AcceptCredentialLoss: enabled, DemandManaged: enabled}))
				}
				if err == nil {
					t.Fatal("expected injected operation error")
				}
				select {
				case opts := <-ops.options:
					if opts.CustomPath != "/tmp/fixture" || opts.BestEffort != enabled || opts.CleanStale != enabled || opts.ForceSetup != enabled || opts.AcceptCredentialLoss != enabled || opts.DemandManaged != enabled {
						t.Fatalf("options not preserved: %+v", opts)
					}
				default:
					t.Fatal("operation not called")
				}
			})
		}
	}
}

type setupPathRunner struct {
	scenarioapp.PhaseRunner
	path string
}

func (r *setupPathRunner) RunPhaseDetailed(name, phase string, opts lifecycle.PhaseOptions) (lifecycle.PhaseResult, error) {
	if name != "fixture" || phase != "setup" {
		return lifecycle.PhaseResult{}, fmt.Errorf("unexpected target %s/%s", name, phase)
	}
	r.path = opts.CustomPath
	return lifecycle.PhaseResult{}, errors.New("test setup ended")
}
func TestSetupRPCForwardsPath(t *testing.T) {
	runner := &setupPathRunner{}
	h := &scenarioControlPlaneHandler{phaseRunner: runner}
	_, err := h.SetupScenario(context.Background(), connect.NewRequest(&cliv1.SetupScenarioRequest{Name: "fixture", Path: "/tmp/fixture"}))
	if err == nil || runner.path != "/tmp/fixture" {
		t.Fatalf("path=%q err=%v", runner.path, err)
	}
}

func TestScenarioLogSourceSelectionAndAggregateBound(t *testing.T) {
	home := t.TempDir()
	dir, err := process.ScenarioLogsDir(home, "fixture")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"vrooli.develop.fixture.api.log", "vrooli.develop.fixture.ui.log", "vrooli.develop.fixture.api.log.bak"} {
		if err = os.WriteFile(filepath.Join(dir, name), []byte(name+"\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	paths, err := process.ScenarioLogPaths(home, "fixture", "api", true, false)
	if err != nil || len(paths) != 1 || !strings.HasSuffix(paths[0], "api.log.bak") {
		t.Fatalf("backup paths=%v err=%v", paths, err)
	}
	paths, err = process.ScenarioLogPaths(home, "fixture", "", false, true)
	if err != nil || len(paths) != 2 {
		t.Fatalf("runtime paths=%v err=%v", paths, err)
	}
	out, err := (&App{}).readLogSnapshots(paths, 1)
	if err != nil || !strings.Contains(out, "api.log <==") || !strings.Contains(out, "ui.log <==") {
		t.Fatalf("snapshot=%q err=%v", out, err)
	}
	for _, p := range paths {
		if err = os.WriteFile(p, []byte(strings.Repeat("x", maxLogSnapshotBytes/2)), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if out, err = (&App{}).readLogSnapshots(paths, 1); out != "" || !errors.Is(err, errLogSnapshotTooLarge) {
		t.Fatalf("oversized aggregate bytes=%d err=%v", len(out), err)
	}
	if _, err = process.ScenarioLogPaths(home, "fixture", "../api", false, false); !errors.Is(err, process.ErrInvalidLogSelector) {
		t.Fatalf("unsafe step error=%v", err)
	}
	for i := 0; i < 129; i++ {
		if err = os.WriteFile(filepath.Join(dir, fmt.Sprintf("extra-%d.log", i)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = process.ScenarioLogPaths(home, "fixture", "", false, true); !errors.Is(err, process.ErrLogSelectionLimit) {
		t.Fatalf("file limit error=%v", err)
	}
}
