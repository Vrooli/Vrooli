package render

import (
	"fmt"
	"strings"
)

// defaultFilename generates a default filename for a replay export based on the execution ID.
func defaultFilename(spec *ReplayMovieSpec, extension string) string {
	stem := "browser-automation-replay"
	if spec != nil {
		execID := strings.TrimSpace(spec.GetExecution().GetExecutionId())
		if len(execID) >= 8 {
			stem = fmt.Sprintf("browser-automation-replay-%s", execID[:8])
		}
	}
	return fmt.Sprintf("%s.%s", stem, extension)
}
