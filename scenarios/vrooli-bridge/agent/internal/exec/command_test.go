package exec_test

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	bridgeexec "vrooli-bridge/agent/internal/exec"

	"github.com/stretchr/testify/require"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
)

// [REQ:BRG-P0-004] End-to-end with the REAL os/exec command runner (no fake
// seam): a typed argv runs the binary directly, captures its output as LOG
// events, and reports the real exit code — with no shell anywhere in the path.
func TestRunner_RealExecNoShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a POSIX echo binary; the runner itself is cross-platform")
	}
	echo, err := exec.LookPath("echo")
	if err != nil {
		t.Skip("no echo binary on PATH")
	}

	rep := &fakeReporter{}
	// bin = the real echo binary; verb tokens become its args. argv =
	// [echo, "bridge-runner-ok"] — executed directly, never via `sh -c`.
	runner := bridgeexec.NewRunner(echo, "", rep,
		bridgeexec.WithClock(func() time.Time { return time.Unix(0, 0).UTC() }))

	require.NoError(t, runner.Execute(context.Background(), &channelv1.JobPush{
		RunId: "r", Verb: "bridge-runner-ok",
	}))

	var gotLog string
	var exitCode int32 = -1
	for _, ev := range rep.events {
		switch ev.Kind {
		case channelv1.RunEventKind_RUN_EVENT_KIND_LOG:
			gotLog += ev.LogChunk
		case channelv1.RunEventKind_RUN_EVENT_KIND_EXIT:
			exitCode = ev.ExitCode
		}
	}
	require.Equal(t, int32(0), exitCode)
	require.Contains(t, strings.TrimSpace(gotLog), "bridge-runner-ok")
}
