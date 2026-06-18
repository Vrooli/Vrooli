package exec_test

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"vrooli-bridge/agent/internal/exec"

	"github.com/stretchr/testify/require"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
)

// fakeReporter collects the RunEvents the runner emits.
type fakeReporter struct {
	mu     sync.Mutex
	events []*channelv1.RunEvent
}

func (f *fakeReporter) Report(_ context.Context, ev *channelv1.RunEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	// Clone the minimal fields we assert on (the runner reuses the pointer
	// across emits only by constructing new events, but copy to be safe).
	f.events = append(f.events, &channelv1.RunEvent{
		RunId: ev.RunId, Kind: ev.Kind, Sequence: ev.Sequence,
		LogChunk: ev.LogChunk, Status: ev.Status, ExitCode: ev.ExitCode, ArtifactRef: ev.ArtifactRef,
	})
	return nil
}

func (f *fakeReporter) kinds() []channelv1.RunEventKind {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]channelv1.RunEventKind, 0, len(f.events))
	for _, e := range f.events {
		out = append(out, e.Kind)
	}
	return out
}

// fakeCommand records the argv it was asked to run and returns a canned exit.
type fakeCommand struct {
	gotArgv []string
	gotDir  string
	exit    int
	logs    []string
}

func (f *fakeCommand) Run(_ context.Context, argv []string, dir string, onLog func(string)) (int, error) {
	f.gotArgv = argv
	f.gotDir = dir
	for _, l := range f.logs {
		onLog(l)
	}
	return f.exit, nil
}

// [REQ:BRG-P0-004] BuildArgv constructs a typed argv — [bin] + verb tokens +
// scenario + args — and is the no-shell-path proof: it returns a []string the
// runner hands to os/exec, never a shell string.
func TestBuildArgv_ConstructsTypedArgv(t *testing.T) {
	argv, err := exec.BuildArgv("vrooli", &channelv1.JobPush{
		Verb: "scenario test", Scenario: "web-search", Args: []string{"--json"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"vrooli", "scenario", "test", "web-search", "--json"}, argv)
}

// [REQ:BRG-P0-004] A shell metacharacter in any typed field is rejected before
// execution — a smuggled shell construct can never reach the runner.
func TestBuildArgv_RejectsShellMetacharacters(t *testing.T) {
	cases := []*channelv1.JobPush{
		{Verb: "scenario test", Args: []string{"web-search; rm -rf /"}},
		{Verb: "scenario test", Scenario: "$(whoami)"},
		{Verb: "scenario test", Args: []string{"a && b"}},
		{Verb: "scenario `id`"},
		{Verb: "scenario test", Args: []string{"x | y"}},
	}
	for _, job := range cases {
		_, err := exec.BuildArgv("vrooli", job)
		require.Error(t, err, "job %+v must be rejected", job)
		require.Contains(t, err.Error(), "unsafe token")
	}
}

// [REQ:BRG-P0-004] An empty verb is rejected.
func TestBuildArgv_RejectsEmptyVerb(t *testing.T) {
	_, err := exec.BuildArgv("vrooli", &channelv1.JobPush{Verb: "   "})
	require.Error(t, err)
}

// [REQ:BRG-P0-004] Execute runs the typed argv via the command seam (never a
// shell), streaming STATUS running → LOG → terminal EXIT with the real code.
func TestExecute_StreamsLifecycle(t *testing.T) {
	cmd := &fakeCommand{exit: 0, logs: []string{"RUN web-search\n", "PASS\n"}}
	rep := &fakeReporter{}
	runner := exec.NewRunner("vrooli", "/work", rep,
		exec.WithCommandRunner(cmd),
		exec.WithClock(func() time.Time { return time.Unix(0, 0).UTC() }),
	)

	err := runner.Execute(context.Background(), &channelv1.JobPush{
		RunId: "run-1", Verb: "scenario test", Scenario: "web-search",
	})
	require.NoError(t, err)

	require.Equal(t, []string{"vrooli", "scenario", "test", "web-search"}, cmd.gotArgv,
		"the runner execs a typed argv, never a shell string")
	require.Equal(t, "/work", cmd.gotDir)

	kinds := rep.kinds()
	require.Equal(t, channelv1.RunEventKind_RUN_EVENT_KIND_STATUS, kinds[0], "first event is STATUS running")
	require.Contains(t, kinds, channelv1.RunEventKind_RUN_EVENT_KIND_LOG)
	require.Equal(t, channelv1.RunEventKind_RUN_EVENT_KIND_EXIT, kinds[len(kinds)-1], "last event is the terminal EXIT")

	// Sequences are monotonic per run starting at 1.
	require.Equal(t, uint64(1), rep.events[0].Sequence)
}

// [REQ:BRG-P0-004] A non-zero exit is reported via the EXIT event (the run is
// FAILED), not as a Go error from Execute.
func TestExecute_NonZeroExitReported(t *testing.T) {
	cmd := &fakeCommand{exit: 2}
	rep := &fakeReporter{}
	runner := exec.NewRunner("vrooli", "", rep, exec.WithCommandRunner(cmd))
	require.NoError(t, runner.Execute(context.Background(), &channelv1.JobPush{RunId: "r", Verb: "scenario test"}))

	last := rep.events[len(rep.events)-1]
	require.Equal(t, channelv1.RunEventKind_RUN_EVENT_KIND_EXIT, last.Kind)
	require.Equal(t, int32(2), last.ExitCode)
}

// [REQ:BRG-P0-004] A job rejected by BuildArgv (unsafe token) is never executed:
// the command seam is not invoked, and a rejection STATUS + non-zero EXIT are
// reported so the run terminates without running anything.
func TestExecute_RejectedJobNeverRuns(t *testing.T) {
	cmd := &fakeCommand{exit: 0}
	rep := &fakeReporter{}
	runner := exec.NewRunner("vrooli", "", rep, exec.WithCommandRunner(cmd))

	require.NoError(t, runner.Execute(context.Background(), &channelv1.JobPush{
		RunId: "r", Verb: "scenario test", Args: []string{"web-search; cat /etc/passwd"},
	}))

	require.Nil(t, cmd.gotArgv, "the command seam is never invoked for a rejected job")
	kinds := rep.kinds()
	require.Equal(t, channelv1.RunEventKind_RUN_EVENT_KIND_EXIT, kinds[len(kinds)-1])
	last := rep.events[len(rep.events)-1]
	require.NotEqual(t, int32(0), last.ExitCode, "a rejected job terminates non-zero")
	// The rejection reason is surfaced as a STATUS event.
	require.True(t, strings.HasPrefix(rep.events[0].Status, "rejected:"))
}
