package runs

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"

	"github.com/vrooli/cli-core/cliutil"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
	"github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// fakeClient implements runs_v1connect.RunsServiceClient with canned responses.
type fakeClient struct {
	runs_v1connect.RunsServiceClient
	list      *runspb.ListRunsResponse
	compare   *runspb.CompareRunsResponse
	freshness *runspb.CheckFreshnessResponse
	// freshnessFn, when set, overrides the canned freshness response with a
	// per-scenario answer (used by the --changed fan-out tests).
	freshnessFn func(scenario string) (*runspb.CheckFreshnessResponse, error)
}

func (f *fakeClient) ListRuns(context.Context, *connect.Request[runspb.ListRunsRequest]) (*connect.Response[runspb.ListRunsResponse], error) {
	return connect.NewResponse(f.list), nil
}

func (f *fakeClient) CompareRuns(context.Context, *connect.Request[runspb.CompareRunsRequest]) (*connect.Response[runspb.CompareRunsResponse], error) {
	return connect.NewResponse(f.compare), nil
}

func withFakeClient(t *testing.T, fc runs_v1connect.RunsServiceClient) {
	t.Helper()
	prev := newClient
	newClient = func(*cliutil.APIClient) (runs_v1connect.RunsServiceClient, error) { return fc, nil }
	t.Cleanup(func() { newClient = prev })
}

func TestRunsListRequiresScenario(t *testing.T) {
	var buf bytes.Buffer
	err := runList(nil, []string{}, &buf)
	if err == nil || !strings.Contains(err.Error(), "scenario") {
		t.Fatalf("expected scenario-required error, got %v", err)
	}
}

func TestRunsListHumanOutput(t *testing.T) {
	withFakeClient(t, &fakeClient{list: &runspb.ListRunsResponse{Runs: []*runspb.RunInfo{
		{RunId: "r2", Status: "passed", StartedAt: "2026-05-26T15:00:00Z"},
		{RunId: "r1", Status: "failed", StartedAt: "2026-05-26T14:00:00Z", Pins: []*runspb.PinInfo{{PinnedBy: "gct"}}},
	}}})

	var buf bytes.Buffer
	if err := runList(nil, []string{"--scenario", "demo"}, &buf); err != nil {
		t.Fatalf("runList: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "r2") || !strings.Contains(out, "passed") {
		t.Errorf("missing run row: %q", out)
	}
	if !strings.Contains(out, "[pinned]") {
		t.Errorf("expected pinned marker: %q", out)
	}
}

func TestRunsCompareExitCodes(t *testing.T) {
	cases := []struct {
		verdict  string
		wantCode int
		wantErr  bool
	}{
		{"clean", 0, false},
		{"new-failure", 0, false},
		{"regression", exitRegression, true},
		{"not-comparable", exitNotComparable, true},
	}
	for _, tc := range cases {
		t.Run(tc.verdict, func(t *testing.T) {
			withFakeClient(t, &fakeClient{compare: &runspb.CompareRunsResponse{
				Verdict: tc.verdict,
				Phases:  []*runspb.PhaseDiff{{Phase: "playbooks", Verdict: tc.verdict, StatusA: "passed", StatusB: "failed"}},
			}})
			var buf bytes.Buffer
			err := runCompare(nil, []string{"--scenario", "demo", "a", "b"}, &buf)
			if tc.wantErr {
				var ee *exitErr
				if !errors.As(err, &ee) {
					t.Fatalf("expected exitErr, got %v", err)
				}
				if ee.ExitCode() != tc.wantCode {
					t.Errorf("exit code = %d, want %d", ee.ExitCode(), tc.wantCode)
				}
			} else if err != nil {
				t.Fatalf("expected nil error for %s, got %v", tc.verdict, err)
			}
			if !strings.Contains(buf.String(), "Verdict:") {
				t.Errorf("expected verdict line in output: %q", buf.String())
			}
		})
	}
}

func TestRunsCompareRequiresTwoIDs(t *testing.T) {
	withFakeClient(t, &fakeClient{})
	var buf bytes.Buffer
	if err := runCompare(nil, []string{"--scenario", "demo", "only-one"}, &buf); err == nil {
		t.Fatal("expected error for missing second runID")
	}
}

func (f *fakeClient) CheckFreshness(_ context.Context, req *connect.Request[runspb.CheckFreshnessRequest]) (*connect.Response[runspb.CheckFreshnessResponse], error) {
	if f.freshnessFn != nil {
		resp, err := f.freshnessFn(req.Msg.GetTarget())
		if err != nil {
			return nil, err
		}
		return connect.NewResponse(resp), nil
	}
	return connect.NewResponse(f.freshness), nil
}

func TestRunsFreshnessFreshExitsZero(t *testing.T) {
	withFakeClient(t, &fakeClient{freshness: &runspb.CheckFreshnessResponse{
		Target:     "demo",
		TreeDigest: "td:abc",
		Phases: []*runspb.PhaseFreshness{
			{Phase: "unit", Status: "fresh", LastRunId: "r1"},
			{Phase: "business", Status: "fresh", LastRunId: "r1"},
		},
	}})
	var buf bytes.Buffer
	if err := runFreshness(nil, []string{"--scenario", "demo"}, &buf); err != nil {
		t.Fatalf("all-fresh must exit clean, got %v", err)
	}
	if !strings.Contains(buf.String(), "All 2 phase(s) fresh") {
		t.Errorf("missing fresh summary: %q", buf.String())
	}
}

func TestRunsFreshnessStaleExitsOneWithCommand(t *testing.T) {
	withFakeClient(t, &fakeClient{freshness: &runspb.CheckFreshnessResponse{
		Target:     "demo",
		TreeDigest: "td:abc",
		Phases: []*runspb.PhaseFreshness{
			{Phase: "unit", Status: "stale", LastRunCompletedAt: "2026-06-09T00:00:00Z"},
			{Phase: "business", Status: "unknown"},
		},
		SuggestedCommand: "test-genie execute demo --preset quick",
	}})
	var buf bytes.Buffer
	err := runFreshness(nil, []string{"--scenario", "demo"}, &buf)
	var ee *exitErr
	if !errors.As(err, &ee) || ee.ExitCode() != exitRegression {
		t.Fatalf("stale must exit %d, got %v", exitRegression, err)
	}
	out := buf.String()
	if !strings.Contains(out, "test-genie execute demo --preset quick") {
		t.Errorf("missing suggested command: %q", out)
	}
	if !strings.Contains(out, "last passed 2026-06-09T00:00:00Z") {
		t.Errorf("missing last-passed context: %q", out)
	}
}

func TestRunsFreshnessPositionalScenario(t *testing.T) {
	withFakeClient(t, &fakeClient{freshness: &runspb.CheckFreshnessResponse{
		Target: "demo",
		Phases: []*runspb.PhaseFreshness{{Phase: "unit", Status: "fresh"}},
	}})
	var buf bytes.Buffer
	if err := runFreshness(nil, []string{"demo"}, &buf); err != nil {
		t.Fatalf("positional scenario must work, got %v", err)
	}
}
