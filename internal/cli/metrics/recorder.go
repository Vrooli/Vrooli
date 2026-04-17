package metrics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	// EnvDisable toggles recording off when set to a truthy-for-off value.
	EnvDisable = "VROOLI_METRICS"

	metricsDirName  = "metrics"
	timingsFileName = "timings.jsonl"
	readmeFileName  = "README.md"
)

const readmeContent = `# Vrooli CLI timings

This directory holds passive timing telemetry for the ` + "`vrooli`" + ` CLI,
one JSON object per line in ` + "`timings.jsonl`" + `.

Fields:

  ts                RFC3339 timestamp (UTC) when the command started.
  cmd               Top-level command (e.g. "scenario", "resource", "setup").
  sub               Redacted sub-args; flag values stripped, secrets masked.
  argc              Count of raw args before redaction.
  duration_ms       Elapsed milliseconds for the dispatched handler.
  exit              Process exit code (0 on success).
  error_class       Short category string, empty on success.
  cli_version       CLI version string at time of invocation.
  platform_version  Platform version string at time of invocation.
  host              os.Hostname().
  pid               Process ID.

Opting out:

  - Per invocation: pass --no-metrics on the command line.
  - Globally:       export VROOLI_METRICS=0 (also: false|no|off).

Caveat: commands that detach a subprocess (e.g. ` + "`scenario start`" + `)
record the parent-side duration, not the time for the child to become healthy.

Rotation: none. Truncate this file manually if it grows large.
`

// Recorder appends Events to a JSONL file. Safe for concurrent use.
// A nil *Recorder or one with disabled=true silently drops events.
type Recorder struct {
	path     string
	disabled bool
	onError  func(error)

	mu         sync.Mutex
	readmeDone bool
}

// New constructs a Recorder that writes to $home/.vrooli/metrics/timings.jsonl.
// If the VROOLI_METRICS env var is set to a disabling value, the recorder is a
// no-op and will never touch disk.
//
// onError, if non-nil, is invoked for each IO failure. It must not panic.
// Recording is always non-fatal; errors are never returned to the caller.
func New(home string, onError func(error)) *Recorder {
	r := &Recorder{
		path:     filepath.Join(home, ".vrooli", metricsDirName, timingsFileName),
		disabled: envDisabled(os.Getenv(EnvDisable)),
		onError:  onError,
	}
	return r
}

// Record appends e to the timings log. Safe to call on a nil Recorder.
func (r *Recorder) Record(e Event) {
	if r == nil || r.disabled {
		return
	}
	line, err := json.Marshal(e)
	if err != nil {
		r.report(err)
		return
	}
	line = append(line, '\n')

	r.mu.Lock()
	defer r.mu.Unlock()

	dir := filepath.Dir(r.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		r.report(err)
		return
	}
	if !r.readmeDone {
		r.ensureReadme(dir)
		r.readmeDone = true
	}
	f, err := os.OpenFile(r.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		r.report(err)
		return
	}
	defer f.Close()
	if _, err := f.Write(line); err != nil {
		r.report(err)
	}
}

// Disabled reports whether recording is suppressed at the process level.
func (r *Recorder) Disabled() bool {
	return r == nil || r.disabled
}

// Path returns the target timings file path (even when disabled).
func (r *Recorder) Path() string {
	if r == nil {
		return ""
	}
	return r.path
}

func (r *Recorder) ensureReadme(dir string) {
	readmePath := filepath.Join(dir, readmeFileName)
	if _, err := os.Stat(readmePath); err == nil {
		return
	}
	_ = os.WriteFile(readmePath, []byte(readmeContent), 0o644)
}

func (r *Recorder) report(err error) {
	if r.onError == nil || err == nil {
		return
	}
	r.onError(err)
}

func envDisabled(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "0", "false", "no", "off":
		return true
	}
	return false
}
