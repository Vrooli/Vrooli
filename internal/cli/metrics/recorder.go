package metrics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
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

Rotation: the active file rotates at the configured size or age boundary;
rotated files are retained up to the configured backup count. Rotation is
owner-local and does not delete source, state, or other runtime-home data.
`

// RotationPolicy bounds the passive timings log. The policy is injectable so
// size and age boundaries can be tested without waiting or relying on the host
// clock.
type RotationPolicy struct {
	MaxBytes   int64
	MaxAge     time.Duration
	MaxBackups int
	Now        func() time.Time
}

const (
	defaultMaxTimingBytes   = 64 * 1024 * 1024
	defaultMaxTimingAge     = 30 * 24 * time.Hour
	defaultMaxTimingBackups = 3
)

func defaultRotationPolicy() RotationPolicy {
	return RotationPolicy{
		MaxBytes:   defaultMaxTimingBytes,
		MaxAge:     defaultMaxTimingAge,
		MaxBackups: defaultMaxTimingBackups,
		Now:        func() time.Time { return time.Now().UTC() },
	}
}

// Recorder appends Events to a JSONL file. Safe for concurrent use.
// A nil *Recorder or one with disabled=true silently drops events.
type Recorder struct {
	path     string
	disabled bool
	onError  func(error)

	mu         sync.Mutex
	readmeDone bool
	rotation   RotationPolicy
}

// New constructs a Recorder that writes to $home/.vrooli/metrics/timings.jsonl.
// If the VROOLI_METRICS env var is set to a disabling value, the recorder is a
// no-op and will never touch disk.
//
// onError, if non-nil, is invoked for each IO failure. It must not panic.
// Recording is always non-fatal; errors are never returned to the caller.
func New(home string, onError func(error)) *Recorder {
	return NewWithPolicy(home, onError, defaultRotationPolicy())
}

// NewWithPolicy constructs a Recorder with an explicit bounded-log policy.
// Zero policy fields use safe defaults; negative values are clamped.
func NewWithPolicy(home string, onError func(error), policy RotationPolicy) *Recorder {
	defaults := defaultRotationPolicy()
	if policy.MaxBytes == 0 {
		policy.MaxBytes = defaults.MaxBytes
	}
	if policy.MaxAge == 0 {
		policy.MaxAge = defaults.MaxAge
	}
	if policy.MaxBackups == 0 {
		policy.MaxBackups = defaults.MaxBackups
	}
	if policy.MaxBytes < 0 {
		policy.MaxBytes = 0
	}
	if policy.MaxAge < 0 {
		policy.MaxAge = 0
	}
	if policy.MaxBackups < 0 {
		policy.MaxBackups = 0
	}
	if policy.Now == nil {
		policy.Now = defaults.Now
	}
	disabled := envDisabled(os.Getenv(EnvDisable))
	path := ""
	if root, err := repocontract.VrooliUserRoot(home); err == nil {
		path = filepath.Join(root, metricsDirName, timingsFileName)
	} else {
		// Without a resolvable runtime home there is nowhere to record; stay a no-op.
		disabled = true
	}
	r := &Recorder{
		path:     path,
		disabled: disabled,
		onError:  onError,
		rotation: policy,
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
	if _, err := config.EnsureOwnedDir(dir); err != nil {
		r.report(err)
		return
	}
	if !r.readmeDone {
		r.ensureReadme(dir)
		r.readmeDone = true
	}
	if err := r.rotateIfNeeded(len(line)); err != nil {
		r.report(err)
	}
	f, err := os.OpenFile(r.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, tuning.PermFile)
	if err != nil {
		r.report(err)
		return
	}
	defer f.Close()
	_ = config.ChownToInvokingUser(r.path)
	if _, err := f.Write(line); err != nil {
		r.report(err)
	}
}

func (r *Recorder) rotateIfNeeded(nextBytes int) error {
	info, err := os.Stat(r.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	now := r.rotation.Now()
	bySize := r.rotation.MaxBytes > 0 && info.Size()+int64(nextBytes) > r.rotation.MaxBytes
	byAge := r.rotation.MaxAge > 0 && !info.ModTime().IsZero() && now.Sub(info.ModTime()) >= r.rotation.MaxAge
	if !bySize && !byAge {
		return nil
	}
	stamp := now.UTC().Format("20060102-150405")
	rotated := r.path + "." + stamp
	for suffix := 1; ; suffix++ {
		if _, statErr := os.Stat(rotated); os.IsNotExist(statErr) {
			break
		}
		rotated = fmt.Sprintf("%s.%s-%d", r.path, stamp, suffix)
	}
	if err := os.Rename(r.path, rotated); err != nil {
		return fmt.Errorf("rotate timings: %w", err)
	}
	r.cleanupRotated(now)
	return nil
}

func (r *Recorder) cleanupRotated(now time.Time) {
	matches, err := filepath.Glob(r.path + ".*")
	if err != nil {
		return
	}
	type backup struct {
		path string
		mod  time.Time
	}
	backups := make([]backup, 0, len(matches))
	for _, path := range matches {
		info, statErr := os.Stat(path)
		if statErr != nil || !info.Mode().IsRegular() {
			continue
		}
		if r.rotation.MaxAge > 0 && now.Sub(info.ModTime()) >= r.rotation.MaxAge {
			_ = os.Remove(path)
			continue
		}
		backups = append(backups, backup{path: path, mod: info.ModTime()})
	}
	sort.Slice(backups, func(i, j int) bool { return backups[i].mod.After(backups[j].mod) })
	for i := r.rotation.MaxBackups; i < len(backups); i++ {
		_ = os.Remove(backups[i].path)
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
	_ = config.WriteOwnedFile(readmePath, []byte(readmeContent), tuning.PermFile)
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
