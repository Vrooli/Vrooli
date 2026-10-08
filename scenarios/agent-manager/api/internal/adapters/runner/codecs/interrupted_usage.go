package codecs

import (
	"strings"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// InterruptedUsageRequest describes one invocation whose stream ended without
// carrying any usage: the process stopped mid-turn (a park, a stop, a crash)
// before the harness printed its turn receipt. The harness's own session log is
// then the only record of what the turn consumed.
type InterruptedUsageRequest struct {
	RunID     uuid.UUID
	SessionID string
	// Env is the invocation's launch environment ("KEY=VALUE"). It locates the
	// run-scoped session home, for example CODEX_HOME.
	Env []string
	// StartedAt is when the invocation launched. A session-log turn that began
	// earlier belongs to an earlier invocation and is never counted again.
	StartedAt time.Time
	EndedAt   time.Time
	Model     string
	Billing   domain.BillingSnapshot
}

// InterruptedUsageRecoverer is implemented by codecs whose harness records
// model-call usage in a session log that outlives the process. The core runner
// calls it once, after the process has exited, only for an invocation whose
// stream produced no usage event, so recovered usage never duplicates streamed
// usage. It returns the events to persist (none when the log holds nothing for
// this invocation) and a one-line note for the run timeline.
//
// A returned usage event is a ReconciliationAuthority receipt only when the log
// proves the turn's usage is complete; otherwise it is the usage of the model
// calls that completed before the stop, which is a lower bound.
type InterruptedUsageRecoverer interface {
	RecoverInterruptedUsage(req InterruptedUsageRequest) (events []*domain.RunEvent, note string)
}

// launchEnvValue returns the last value of key in an os.Environ-shaped list.
func launchEnvValue(env []string, key string) string {
	value := ""
	for _, entry := range env {
		if name, v, ok := strings.Cut(entry, "="); ok && name == key {
			value = v
		}
	}
	return strings.TrimSpace(value)
}
