// Package asyncpoll provides the shared timing loop for asynchronous CLI
// operations. Callers retain ownership of fetching records and deciding which
// domain statuses are terminal.
package asyncpoll

import (
	"fmt"
	"time"

	"github.com/vrooli/api-core/schedule"
)

// UntilTerminal polls immediately, then waits between nonterminal results.
// The shared API Core clock keeps deadline and wait behavior controllable in
// tests without a parallel clock interface in this CLI.
func UntilTerminal[T any](
	clock schedule.Clock,
	interval, timeout time.Duration,
	fetch func() (value T, terminal bool, status string, err error),
	operation, id string,
) (T, error) {
	var zero T
	deadline := clock.Now().Add(timeout)
	for {
		value, terminal, status, err := fetch()
		if err != nil {
			return zero, err
		}
		if terminal {
			return value, nil
		}
		if clock.Now().After(deadline) {
			return zero, fmt.Errorf("%s %q did not reach a terminal state within %s (still %s)", operation, id, timeout, status)
		}
		clock.Sleep(interval)
	}
}
