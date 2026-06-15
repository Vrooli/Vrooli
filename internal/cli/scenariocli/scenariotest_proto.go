package scenariocli

import (
	"io"
	"time"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/lifecycle"
	cliv1 "github.com/vrooli/vrooli/packages/proto/gen/go/cli/v1"
)

func rfc3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

// TestPhaseResultProto maps the lifecycle test-phase outcome onto the
// vrooli.cli.v1.TestPhaseResult wire contract — the single producer-side
// translation every consumer decodes. A proto field rename breaks this mapping
// at compile time (the drift guard).
//
// status is "passed" when the phase ran clean and "failed" otherwise; runErr is
// the phase error (nil on success). The durable run id and per-phase detail are
// intentionally absent here: they live behind `test-genie runs …`, the
// server-owned run history.
func TestPhaseResultProto(result lifecycle.PhaseResult, runErr error) *cliv1.TestPhaseResult {
	status := "passed"
	exitCode := result.ExitCode
	if runErr != nil {
		status = "failed"
		if exitCode == 0 {
			exitCode = 1
		}
	}
	duration := ""
	if !result.StartedAt.IsZero() && !result.EndedAt.IsZero() {
		duration = result.EndedAt.Sub(result.StartedAt).Round(time.Second).String()
	}
	return &cliv1.TestPhaseResult{
		Scenario:  result.Scenario,
		Status:    status,
		ExitCode:  int32(exitCode),
		StartedAt: rfc3339(result.StartedAt),
		EndedAt:   rfc3339(result.EndedAt),
		Duration:  duration,
		LogFile:   result.LogFile,
	}
}

// WriteTestPhaseResultJSON emits the typed test-phase result as canonical CLI JSON.
func WriteTestPhaseResultJSON(w io.Writer, result lifecycle.PhaseResult, runErr error) error {
	return cliout.WriteProtoJSON(w, TestPhaseResultProto(result, runErr))
}
