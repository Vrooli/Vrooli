package exec_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"vrooli-bridge/agent/internal/exec"

	"github.com/stretchr/testify/require"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/shared"
)

// fakeReporter collects the RunEvents the runner emits.
type fakeReporter struct {
	mu     sync.Mutex
	events []*sharedv1.RunEvent
}

func (f *fakeReporter) Report(_ context.Context, ev *sharedv1.RunEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	// Clone the minimal fields we assert on (the runner reuses the pointer
	// across emits only by constructing new events, but copy to be safe).
	f.events = append(f.events, &sharedv1.RunEvent{
		RunId: ev.RunId, Kind: ev.Kind, Sequence: ev.Sequence,
		LogChunk: ev.LogChunk, Status: ev.Status, ExitCode: ev.ExitCode, ArtifactRef: ev.ArtifactRef,
	})
	return nil
}

func (f *fakeReporter) kinds() []sharedv1.RunEventKind {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]sharedv1.RunEventKind, 0, len(f.events))
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

type fakeUploader struct {
	runID, name, mediaType string
	data                   []byte
}

func (f *fakeUploader) Upload(_ context.Context, runID, name, mediaType string, data []byte) (string, error) {
	f.runID, f.name, f.mediaType = runID, name, mediaType
	f.data = append([]byte(nil), data...)
	return "bridge://run/" + runID + "/" + name, nil
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

func TestBuildArgv_RoutesDeviceControlNamespaceToScenarioCLI(t *testing.T) {
	argv, err := exec.BuildArgv("vrooli", &channelv1.JobPush{
		RunId: "bridge-run-42", Verb: "device-control flow", Scenario: "device-control",
		Args: []string{"run", "--file", "/tmp/flow.json", "--device", "android-1"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"device-control", "flow", "run", "--file", "/tmp/flow.json", "--device", "android-1"}, argv)
}

func TestExecute_DeviceControlJobPreservesBridgeRunIdentity(t *testing.T) {
	cmd := &fakeCommand{exit: 0}
	rep := &fakeReporter{}
	runner := exec.NewRunner("vrooli", "/work", rep, exec.WithCommandRunner(cmd))

	require.NoError(t, runner.Execute(context.Background(), &channelv1.JobPush{
		RunId: "bridge-run-42", Verb: "device-control flow", Scenario: "device-control",
		Args: []string{"run", "--file", "/tmp/flow.json", "--device", "android-1"},
	}))
	require.Equal(t, []string{"device-control", "flow", "run", "--file", "/tmp/flow.json", "--device", "android-1"}, cmd.gotArgv)
	for _, event := range rep.events {
		require.Equal(t, "bridge-run-42", event.RunId)
	}
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
	require.Equal(t, sharedv1.RunEventKind_RUN_EVENT_KIND_STATUS, kinds[0], "first event is STATUS running")
	require.Contains(t, kinds, sharedv1.RunEventKind_RUN_EVENT_KIND_LOG)
	require.Equal(t, sharedv1.RunEventKind_RUN_EVENT_KIND_EXIT, kinds[len(kinds)-1], "last event is the terminal EXIT")

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
	require.Equal(t, sharedv1.RunEventKind_RUN_EVENT_KIND_EXIT, last.Kind)
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
	require.Equal(t, sharedv1.RunEventKind_RUN_EVENT_KIND_EXIT, kinds[len(kinds)-1])
	last := rep.events[len(rep.events)-1]
	require.NotEqual(t, int32(0), last.ExitCode, "a rejected job terminates non-zero")
	// The rejection reason is surfaced as a STATUS event.
	require.True(t, strings.HasPrefix(rep.events[0].Status, "rejected:"))
}

func TestExecute_AppendsTypedOutputFlagAndUploadsProducedArtifact(t *testing.T) {
	dir := t.TempDir()
	cmd := &fakeCommand{exit: 0}
	cmd.logs = nil
	// The command fake writes to the final argv token, exactly as the real
	// screenshot command writes to its typed --output path.
	cmdRun := func(_ context.Context, argv []string, workDir string, onLog func(string)) (int, error) {
		cmd.gotArgv, cmd.gotDir = argv, workDir
		if err := os.WriteFile(argv[len(argv)-1], []byte("png-bytes"), 0o600); err != nil {
			return 127, err
		}
		onLog("captured\n")
		return 0, nil
	}
	cmdRunner := commandFunc(cmdRun)
	uploader := &fakeUploader{}
	rep := &fakeReporter{}
	runner := exec.NewRunner("vrooli", "/work", rep,
		exec.WithCommandRunner(cmdRunner), exec.WithArtifactUploader(uploader), exec.WithArtifactDir(dir))

	err := runner.Execute(context.Background(), &channelv1.JobPush{
		RunId: "run-1", Verb: "scenario screenshot",
		Outputs: []*channelv1.ArtifactOutput{{Name: "screenshot.png", MediaType: "image/png", OutputFlag: "--output", MaxBytes: 1024}},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"vrooli", "scenario", "screenshot", "--output", filepath.Join(dir, "run-1-screenshot.png")}, cmd.gotArgv)
	require.Equal(t, "run-1", uploader.runID)
	require.Equal(t, "screenshot.png", uploader.name)
	require.Equal(t, "image/png", uploader.mediaType)
	require.Equal(t, []byte("png-bytes"), uploader.data)
	lastBeforeExit := rep.events[len(rep.events)-2]
	require.Equal(t, sharedv1.RunEventKind_RUN_EVENT_KIND_ARTIFACT_REF, lastBeforeExit.Kind)
	require.Equal(t, "bridge://run/run-1/screenshot.png", lastBeforeExit.ArtifactRef)
}

type commandFunc func(context.Context, []string, string, func(string)) (int, error)

func (f commandFunc) Run(ctx context.Context, argv []string, dir string, onLog func(string)) (int, error) {
	return f(ctx, argv, dir, onLog)
}
