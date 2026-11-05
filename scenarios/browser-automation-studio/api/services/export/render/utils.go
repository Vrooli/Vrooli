package render

import (
	"fmt"
	"os"
	"strings"
)

// sanitizeFilename removes path separators and null bytes from a filename.
func sanitizeFilename(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "browser-automation-replay"
	}
	trimmed = strings.ReplaceAll(trimmed, string(os.PathSeparator), "-")
	trimmed = strings.ReplaceAll(trimmed, "\x00", "")
	return trimmed
}

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
