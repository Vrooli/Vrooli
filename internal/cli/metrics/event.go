package metrics

import "time"

// Event is a single CLI invocation timing record. One Event per line in the
// JSONL log. Field tags control wire format; renaming requires updating the
// analysis queries documented in internal/cli/metrics/README.md.
type Event struct {
	StartedAt       time.Time `json:"ts"`
	Command         string    `json:"cmd"`
	Args            []string  `json:"sub,omitempty"`
	Argc            int       `json:"argc"`
	DurationMs      int64     `json:"duration_ms"`
	ExitCode        int       `json:"exit"`
	ErrorClass      string    `json:"error_class,omitempty"`
	CLIVersion      string    `json:"cli_version,omitempty"`
	PlatformVersion string    `json:"platform_version,omitempty"`
	Hostname        string    `json:"host,omitempty"`
	PID             int       `json:"pid,omitempty"`
}
