package execute

import (
	"fmt"
	"io"
	"time"

	"test-genie/cli/internal/phases"
)

// StartProgress begins a progress ticker that displays execution status.
// Returns a stop function to call when execution completes.
func StartProgress(w io.Writer, phaseList []string, targets map[string]time.Duration) func() {
	if len(phaseList) == 0 {
		phaseList = []string{"structure", "unit", "integration"}
	}
	start := time.Now()
	ticker := time.NewTicker(2 * time.Second)
	done := make(chan struct{})
	go func() {
		idx := 0
		for {
			select {
			case <-done:
				ticker.Stop()
				return
			case <-ticker.C:
				phase := phaseList[idx%len(phaseList)]
				idx++
				elapsed := time.Since(start).Truncate(time.Second)
				target := ""
				if d, ok := targets[phases.NormalizeName(phase)]; ok && d > 0 {
					target = fmt.Sprintf(" target=%s", d.Truncate(time.Second))
				}
				fmt.Fprintf(w, "\rExecuting %-12s (t+%s%s)", phase, elapsed, target)
			}
		}
	}()
	return func() {
		close(done)
		fmt.Fprint(w, "\r")
	}
}
