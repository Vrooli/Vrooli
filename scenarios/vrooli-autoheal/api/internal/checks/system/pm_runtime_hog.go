// Package system: pm_runtime hog check.
//
// Linux's runtime PM subsystem logs warnings when a device's autosuspend
// callback hogs the CPU. A flood of these is a known leading indicator of
// USB / NVMe / GPU misbehavior on Ryzen 7000 + X870E platforms — the same
// boxes that are most prone to unexplained hard resets.
package system

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/journal"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// PMRuntimeHogCheck reports kernel "hogged CPU" warnings from runtime PM.
type PMRuntimeHogCheck struct {
	reader        *journal.Reader
	executor      checks.CommandExecutor
	warnPerHour   int
	criticalPerHr int
	now           func() time.Time
}

// PMRuntimeHogCheckOption configures a PMRuntimeHogCheck.
type PMRuntimeHogCheckOption func(*PMRuntimeHogCheck)

// WithPMRuntimeReader injects the journal reader (for testing).
func WithPMRuntimeReader(r *journal.Reader) PMRuntimeHogCheckOption {
	return func(c *PMRuntimeHogCheck) { c.reader = r }
}

// WithPMRuntimeExecutor injects the command executor (for recovery actions).
func WithPMRuntimeExecutor(e checks.CommandExecutor) PMRuntimeHogCheckOption {
	return func(c *PMRuntimeHogCheck) { c.executor = e }
}

// WithPMRuntimeThresholds sets warn/critical thresholds (events per hour).
func WithPMRuntimeThresholds(warn, critical int) PMRuntimeHogCheckOption {
	return func(c *PMRuntimeHogCheck) {
		c.warnPerHour = warn
		c.criticalPerHr = critical
	}
}

// NewPMRuntimeHogCheck builds the check.
func NewPMRuntimeHogCheck(opts ...PMRuntimeHogCheckOption) *PMRuntimeHogCheck {
	c := &PMRuntimeHogCheck{
		executor:      checks.DefaultExecutor,
		warnPerHour:   50,
		criticalPerHr: 200,
		now:           time.Now,
	}
	c.reader = journal.NewReader(c.executor)
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *PMRuntimeHogCheck) ID() string    { return "system-pm-runtime-hog" }
func (c *PMRuntimeHogCheck) Title() string { return "Runtime PM CPU Hogs" }
func (c *PMRuntimeHogCheck) Description() string {
	return "Counts \"pm_runtime_work hogged CPU\" kernel warnings in the last hour"
}

func (c *PMRuntimeHogCheck) Importance() string {
	return "Bursts of pm_runtime hog warnings precede unexplained hard resets on consumer Ryzen platforms"
}
func (c *PMRuntimeHogCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *PMRuntimeHogCheck) IntervalSeconds() int       { return 300 }
func (c *PMRuntimeHogCheck) Platforms() []platform.Type { return []platform.Type{platform.Linux} }

func (c *PMRuntimeHogCheck) Run(ctx context.Context) checks.Result {
	r := checks.Result{CheckID: c.ID(), Details: map[string]interface{}{}}
	if runtime.GOOS != "linux" {
		r.Status = checks.StatusOK
		r.Message = "pm_runtime check is Linux-only"
		return r
	}

	if !c.reader.Available(ctx) {
		r.Status = checks.StatusWarning
		r.Message = "journalctl unavailable; cannot count pm_runtime warnings"
		r.Details["journalAvailable"] = false
		return r
	}
	r.Details["journalAvailable"] = true

	entries, err := c.reader.QueryLogs(ctx, journal.QueryOpts{
		Kernel: true,
		Grep:   "pm_runtime_work .* hogged CPU",
		Since:  "1 hour ago",
		Tail:   1000,
	})
	if err != nil {
		r.Status = checks.StatusWarning
		r.Message = "Failed to query journal: " + err.Error()
		r.Details["error"] = err.Error()
		return r
	}
	count := len(entries)
	r.Details["count"] = count
	r.Details["warnThreshold"] = c.warnPerHour
	r.Details["criticalThreshold"] = c.criticalPerHr

	switch {
	case count >= c.criticalPerHr:
		r.Status = checks.StatusCritical
		r.Message = fmt.Sprintf("%d pm_runtime hog warnings in last hour — investigate misbehaving driver", count)
	case count >= c.warnPerHour:
		r.Status = checks.StatusWarning
		r.Message = fmt.Sprintf("%d pm_runtime hog warnings in last hour", count)
	default:
		r.Status = checks.StatusOK
		r.Message = fmt.Sprintf("pm_runtime hog rate normal (%d/hr)", count)
	}
	r.Timestamp = c.now()
	return r
}

// RecoveryActions: diagnostic only.
func (c *PMRuntimeHogCheck) RecoveryActions(*checks.Result) []checks.RecoveryAction {
	return []checks.RecoveryAction{
		{ID: "show-recent-warnings", Name: "Show recent warnings", Description: "Tail kernel pm_runtime warnings from last hour", Available: true},
		{ID: "show-suspect-devices", Name: "Show suspect devices", Description: "List autosuspend-enabled devices via /sys/bus/*/devices/*/power/runtime_status", Available: true},
	}
}

func (c *PMRuntimeHogCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	res := checks.ActionResult{ActionID: actionID, CheckID: c.ID(), Timestamp: start}
	switch actionID {
	case "show-recent-warnings":
		entries, err := c.reader.QueryLogs(ctx, journal.QueryOpts{
			Kernel: true, Grep: "pm_runtime_work .* hogged CPU",
			Since: "1 hour ago", Tail: 200,
		})
		res.Duration = time.Since(start)
		if err != nil {
			res.Success = false
			res.Error = err.Error()
			res.Message = "Failed to query journal"
			return res
		}
		res.Output = journal.RenderText(entries)
		res.Success = true
		res.Message = fmt.Sprintf("Retrieved %d entries", len(entries))
		return res
	case "show-suspect-devices":
		out, err := c.executor.CombinedOutput(ctx, "sh", "-c",
			`for f in /sys/bus/*/devices/*/power/runtime_status; do printf "%s: %s\n" "$f" "$(cat "$f")"; done`)
		res.Duration = time.Since(start)
		res.Output = string(out)
		if err != nil {
			res.Success = false
			res.Error = err.Error()
			res.Message = "Failed to enumerate runtime PM status"
			return res
		}
		res.Success = true
		res.Message = "Enumerated runtime PM status"
		return res
	default:
		res.Duration = time.Since(start)
		res.Success = false
		res.Error = "unknown action: " + actionID
		return res
	}
}
