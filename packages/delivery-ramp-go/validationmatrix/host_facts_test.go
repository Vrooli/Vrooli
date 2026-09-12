package validationmatrix

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	dispatchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/shared"
)

// recordingRuns captures how many times a run detail was fetched so cache
// behaviour is observable.
type recordingRuns struct {
	events   []*sharedv1.RunEvent
	status   runsv1.RunStatus
	getCalls *int
}

func (r recordingRuns) WaitRun(_ context.Context, request *connect.Request[runsv1.WaitRunRequest]) (*connect.Response[runsv1.WaitRunResponse], error) {
	return connect.NewResponse(&runsv1.WaitRunResponse{Run: &runsv1.Run{Id: request.Msg.Id, Status: r.status}}), nil
}

func (r recordingRuns) GetRun(_ context.Context, request *connect.Request[runsv1.GetRunRequest]) (*connect.Response[runsv1.GetRunResponse], error) {
	if r.getCalls != nil {
		*r.getCalls++
	}
	return connect.NewResponse(&runsv1.GetRunResponse{
		Run:    &runsv1.Run{Id: request.Msg.Id, Status: r.status},
		Events: r.events,
	}), nil
}

func logEvents(chunks ...string) []*sharedv1.RunEvent {
	events := make([]*sharedv1.RunEvent, 0, len(chunks)+1)
	events = append(events, &sharedv1.RunEvent{Kind: sharedv1.RunEventKind_RUN_EVENT_KIND_STATUS, Status: "running"})
	for _, chunk := range chunks {
		events = append(events, &sharedv1.RunEvent{Kind: sharedv1.RunEventKind_RUN_EVENT_KIND_LOG, LogChunk: chunk})
	}
	return events
}

const darwinProbeJSON = `{"os":"darwin","arch":"amd64","runtime_tools":{"xcodebuild":{"present":true,"path":"/usr/bin/xcodebuild","version":"26.1"},"simctl":{"present":true,"version":"iOS 26.1"}},"probe_statuses":{"apple_toolchain":"ok"}}`

func TestRunLogTextConcatenatesOnlyLogEvents(t *testing.T) {
	// Bridge splits command output across many log chunks and interleaves
	// status events; only the log payload is the command's output.
	text := RunLogText(logEvents(`{"os":`, `"darwin"}`))
	if text != `{"os":"darwin"}` {
		t.Fatalf("log text = %q", text)
	}
}

func TestParseHostFactsReadsToolchainFromDispatchedOutput(t *testing.T) {
	facts, err := ParseHostFacts(darwinProbeJSON)
	if err != nil {
		t.Fatal(err)
	}
	if facts.OS != "darwin" || facts.Arch != "amd64" {
		t.Fatalf("facts = %#v", facts)
	}
	xcode, probed := facts.Tool("xcodebuild")
	if !probed || !xcode.Present || xcode.Version != "26.1" {
		t.Fatalf("xcodebuild = %#v probed=%v", xcode, probed)
	}
	if !facts.HasTool("simctl") {
		t.Fatal("simctl must be present")
	}
}

func TestParseHostFactsToleratesSurroundingOutput(t *testing.T) {
	// A dispatched command may print human preamble around the document.
	facts, err := ParseHostFacts("Collecting host facts...\n" + darwinProbeJSON + "\ndone\n")
	if err != nil {
		t.Fatal(err)
	}
	if facts.OS != "darwin" {
		t.Fatalf("facts = %#v", facts)
	}
}

func TestParseHostFactsRejectsUnusableOutput(t *testing.T) {
	tests := []struct{ name, input string }{
		{name: "no json", input: "command not found"},
		{name: "empty", input: ""},
		{name: "malformed", input: "{not json}"},
		{name: "no operating system", input: `{"arch":"amd64"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseHostFacts(tt.input); err == nil {
				t.Fatalf("expected an error for %q", tt.input)
			}
		})
	}
}

func TestHostFactsDistinguishProbedAbsentFromNeverProbed(t *testing.T) {
	facts := HostFacts{OS: "darwin", RuntimeTools: map[string]HostTool{"xcodebuild": {Present: false}}}

	if _, probed := facts.Tool("xcodebuild"); !probed {
		t.Fatal("a recorded absent tool must report as probed")
	}
	if _, probed := facts.Tool("simctl"); probed {
		t.Fatal("an unrecorded tool must not report as probed")
	}
	if facts.HasTool("xcodebuild") {
		t.Fatal("a probed-absent tool must not report present")
	}
}

func TestDispatchHostProberReadsFactsFromRunEvents(t *testing.T) {
	dispatcher := &fakeDispatcher{}
	prober := newDispatchHostProber(dispatcher, recordingRuns{
		events: logEvents(darwinProbeJSON), status: runsv1.RunStatus_RUN_STATUS_PASSED,
	})

	facts, err := prober.ProbeHost(context.Background(), "mac-1")
	if err != nil {
		t.Fatal(err)
	}
	if facts.OS != "darwin" || !facts.HasTool("xcodebuild") {
		t.Fatalf("facts = %#v", facts)
	}
	// The probe must stay inside the node's allowlist and stay read-only.
	if dispatcher.request.Verb != hostProbeVerb {
		t.Fatalf("probe verb = %q, want %q", dispatcher.request.Verb, hostProbeVerb)
	}
	if strings.Join(dispatcher.request.Args, " ") != strings.Join(hostProbeArgs, " ") {
		t.Fatalf("probe args = %v", dispatcher.request.Args)
	}
	if dispatcher.request.NodeId != "mac-1" {
		t.Fatalf("probe node = %q", dispatcher.request.NodeId)
	}
}

func TestDispatchHostProberCachesWithinTTL(t *testing.T) {
	// Listing targets must not dispatch a job per node per call.
	calls := 0
	prober := newDispatchHostProber(&fakeDispatcher{}, recordingRuns{
		events: logEvents(darwinProbeJSON), status: runsv1.RunStatus_RUN_STATUS_PASSED, getCalls: &calls,
	})
	now := time.Now()
	prober.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if _, err := prober.ProbeHost(context.Background(), "mac-1"); err != nil {
			t.Fatal(err)
		}
	}
	if calls != 1 {
		t.Fatalf("probe dispatched %d times within the TTL, want 1", calls)
	}

	now = now.Add(HostFactsTTL + time.Second)
	if _, err := prober.ProbeHost(context.Background(), "mac-1"); err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("probe was not refreshed after the TTL: calls = %d", calls)
	}
}

func TestDispatchHostProberRejectsNonPassingRun(t *testing.T) {
	// A failed probe must surface as an error, never as empty facts that would
	// read as "this node has no toolchain".
	prober := newDispatchHostProber(&fakeDispatcher{}, recordingRuns{
		events: logEvents(darwinProbeJSON), status: runsv1.RunStatus_RUN_STATUS_FAILED,
	})
	if _, err := prober.ProbeHost(context.Background(), "mac-1"); err == nil {
		t.Fatal("a failed probe run must return an error")
	}
}

func TestDispatchHostProberRequiresConfiguredTransport(t *testing.T) {
	var prober *dispatchHostProber
	if _, err := prober.ProbeHost(context.Background(), "mac-1"); err == nil {
		t.Fatal("an unconfigured prober must return an error")
	}
}

func TestDispatchHostProberSurfacesDispatchRefusal(t *testing.T) {
	prober := newDispatchHostProber(refusingDispatcher{}, recordingRuns{status: runsv1.RunStatus_RUN_STATUS_PASSED})
	if _, err := prober.ProbeHost(context.Background(), "mac-1"); err == nil {
		t.Fatal("a refused dispatch must return an error")
	}
}

type refusingDispatcher struct{}

func (refusingDispatcher) DispatchJob(context.Context, *connect.Request[dispatchv1.DispatchJobRequest]) (*connect.Response[dispatchv1.DispatchJobResponse], error) {
	return nil, errors.New("verb not allowlisted")
}
