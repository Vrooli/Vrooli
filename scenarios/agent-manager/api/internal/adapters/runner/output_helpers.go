// Package runner — shared output / classification helpers used by the
// generic [core.Runner] and any direct callers in tests.
//
// These were previously private to claude_code.go but became seam-level
// utilities once the per-runner duplication collapsed into core/. They
// live in the parent runner package so both core/ and tests can reach
// them without import cycles.
package runner

import (
	"fmt"
	"strings"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// EmitStderrAsWarnOnSuccess publishes captured stderr as a warn-level
// run-log event when the process exited cleanly but produced diagnostic
// output. Without this, launch-time diagnostics (e.g. bwrap warnings or
// the chdir failure that reproduced the swarm-manager 134ms-no-output
// regression) are silently dropped on the success path because stderr
// is otherwise only consulted when the wait error is non-nil.
//
// The body is truncated at 4 KB to keep run-event payload sizes bounded;
// operators reading the raw process logs on disk get the unabridged output.
func EmitStderrAsWarnOnSuccess(runID uuid.UUID, sink EventSink, stderr string) {
	trimmed := strings.TrimSpace(stderr)
	if trimmed == "" || sink == nil {
		return
	}
	_ = sink.Emit(domain.NewLogEvent(
		runID,
		"warn",
		fmt.Sprintf("Runner stderr (process exited cleanly):\n%s", TruncateForLog(trimmed, 4096)),
	))
}

// TruncateForLog returns s capped at max bytes, suffixing with a marker
// when truncation occurred. Callers use this for run-event payloads that
// must not balloon the event store.
func TruncateForLog(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n…[truncated]"
}
