// Package system: stale supervised-binary check.
//
// A long-running Vrooli service keeps executing whatever binary it started
// with. When setup or a scenario rebuild replaces that file on disk, the
// process keeps the old inode — the kernel reports its executable as
// "<path> (deleted)" — and it goes on running code that no longer exists.
//
// Nothing owned this. The CLI has a stale-binary check that rebuilds and
// re-execs, but it only fires on invocation; a supervisor started days ago
// never re-reads it. The gap surfaced on 2026-08-19: after a rebuild shipped a
// new restart-gating signal into the runtime supervisor, the running supervisor
// still had the old code and the only way to pick it up was for an operator to
// run systemctl by hand. That is exactly the hand-operation setup exists to
// eliminate, so detection and recovery belong here instead.
//
// The recovery action is a plain "restart", which places it under the same
// pressure gate as every other restart: a stale binary is never urgent enough
// to restart a saturated host for.
package system

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// SupervisedUnit is a systemd user unit this check watches.
type SupervisedUnit struct {
	// Unit is the systemd user unit name. It is the only identity used: the
	// service manager knows which process belongs to it, so nothing here has to
	// guess from the process table.
	Unit string
}

// defaultSupervisedUnits are the long-lived services that keep Vrooli healthy.
// They are the ones whose staleness matters: a stale short-lived process fixes
// itself on its next invocation.
var defaultSupervisedUnits = []SupervisedUnit{
	{Unit: "vrooli-runtime-supervisor.service"},
	{Unit: "vrooli-autoheal.service"},
}

// ProcessExeResolver resolves a unit's running executable. It is the OS seam:
// systemd names the unit's main process, and Linux publishes that process's
// executable at /proc/<pid>/exe, where the "(deleted)" suffix the kernel
// appends is the whole signal.
//
// Resolution is by unit, never by command-line pattern. A `pgrep -f` search is
// tempting and wrong: it matches any process whose command line happens to
// contain the string — including the shell of whoever is currently grepping for
// it — so a collision would silently return an unrelated process's executable
// and report a stale service as healthy.
type ProcessExeResolver interface {
	// Resolve returns the executable path backing unit, whether that path has
	// been replaced, and whether the unit has a running main process.
	Resolve(ctx context.Context, unit string) (path string, deleted bool, found bool)
}

// StaleServiceBinaryCheck reports supervised services running a replaced binary.
type StaleServiceBinaryCheck struct {
	units    []SupervisedUnit
	resolver ProcessExeResolver
	restart  func(ctx context.Context, unit string) (string, error)
}

// StaleServiceBinaryOption configures the check.
type StaleServiceBinaryOption func(*StaleServiceBinaryCheck)

// WithSupervisedUnits overrides the watched unit list.
func WithSupervisedUnits(units []SupervisedUnit) StaleServiceBinaryOption {
	return func(c *StaleServiceBinaryCheck) { c.units = units }
}

// WithProcessExeResolver injects an executable resolver.
func WithProcessExeResolver(r ProcessExeResolver) StaleServiceBinaryOption {
	return func(c *StaleServiceBinaryCheck) { c.resolver = r }
}

// WithUnitRestarter injects the restart mechanism.
func WithUnitRestarter(fn func(ctx context.Context, unit string) (string, error)) StaleServiceBinaryOption {
	return func(c *StaleServiceBinaryCheck) { c.restart = fn }
}

// NewStaleServiceBinaryCheck builds the check with production defaults.
func NewStaleServiceBinaryCheck(opts ...StaleServiceBinaryOption) *StaleServiceBinaryCheck {
	c := &StaleServiceBinaryCheck{
		units:    defaultSupervisedUnits,
		resolver: linuxProcResolver{},
		restart:  restartUserUnit,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *StaleServiceBinaryCheck) ID() string    { return "system-stale-service-binary" }
func (c *StaleServiceBinaryCheck) Title() string { return "Supervised Service Binary Freshness" }
func (c *StaleServiceBinaryCheck) Description() string {
	return "Detects long-running Vrooli services still executing a binary that has been replaced on disk"
}

func (c *StaleServiceBinaryCheck) Importance() string {
	return "A supervisor running deleted code silently ignores every fix shipped since it started, including fixes to its own safety gates"
}
func (c *StaleServiceBinaryCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *StaleServiceBinaryCheck) IntervalSeconds() int       { return 300 }
func (c *StaleServiceBinaryCheck) Platforms() []platform.Type { return []platform.Type{platform.Linux} }

func (c *StaleServiceBinaryCheck) Run(ctx context.Context) (r checks.Result) {
	r = checks.Result{CheckID: c.ID(), Details: map[string]interface{}{}}
	defer func() {
		if r.Timestamp.IsZero() {
			r.Timestamp = time.Now()
		}
	}()
	if checkOS != "linux" {
		r.Status = checks.StatusNotApplicable
		r.Message = "supervised binary freshness is read from /proc, which is Linux-only"
		r.Details["platform"] = checkOS
		return r
	}

	var stale []string
	checked := 0
	for _, unit := range c.units {
		path, deleted, found := c.resolver.Resolve(ctx, unit.Unit)
		if !found {
			// A unit that is not running is a liveness problem, which the
			// emergency watchdog and the unit checks already own. Silence here
			// keeps one condition from being reported by two checks.
			continue
		}
		checked++
		if deleted {
			stale = append(stale, unit.Unit)
			r.Details["staleExe_"+unit.Unit] = path
		}
	}

	r.Details["unitsChecked"] = checked
	r.Details["staleUnits"] = stale
	// The evidence dimension the incident fingerprint keys on: one unit going
	// stale repeatedly is one incident, not a new one every five minutes.
	r.Details["findingKey"] = strings.Join(stale, ",")

	if len(stale) == 0 {
		r.Status = checks.StatusOK
		r.Message = "Supervised services are running their installed binaries"
		return r
	}

	r.Status = checks.StatusWarning
	r.Message = fmt.Sprintf("Supervised service running a replaced binary: %s — restart to pick up the installed version", strings.Join(stale, ", "))
	r.Details["recommendations"] = []string{
		"restart the affected unit so it executes the installed binary",
	}
	return r
}

// RecoveryActions offers the restart. It is not dangerous: these units are
// designed to be restarted, and the pressure gate decides when it is safe.
func (c *StaleServiceBinaryCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	available := false
	if lastResult != nil {
		available = len(staleUnitsFromDetails(lastResult.Details)) > 0
	}
	return []checks.RecoveryAction{{
		ID:          "restart",
		Name:        "Restart Stale Services",
		Description: "Restart supervised units whose binary was replaced, so they execute the installed version",
		Dangerous:   false,
		Available:   available,
	}}
}

func (c *StaleServiceBinaryCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{ActionID: actionID, CheckID: c.ID(), Timestamp: start}
	if actionID != "restart" {
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}

	// Re-resolve rather than trusting the last result: the condition may have
	// cleared, and restarting a healthy supervisor for no reason is exactly the
	// churn this check is meant to remove.
	var restarted, failed []string
	var output strings.Builder
	for _, unit := range c.units {
		_, deleted, found := c.resolver.Resolve(ctx, unit.Unit)
		if !found || !deleted {
			continue
		}
		out, err := c.restart(ctx, unit.Unit)
		output.WriteString(out)
		if err != nil {
			failed = append(failed, unit.Unit)
			continue
		}
		restarted = append(restarted, unit.Unit)
	}

	result.Output = output.String()
	result.Duration = time.Since(start)
	switch {
	case len(restarted) == 0 && len(failed) == 0:
		result.Success = true
		result.Message = "No supervised service was still running a replaced binary"
	case len(failed) > 0:
		result.Success = false
		result.Error = "failed to restart: " + strings.Join(failed, ", ")
		result.Message = result.Error
	default:
		result.Success = true
		result.Message = "Restarted " + strings.Join(restarted, ", ")
	}
	return result
}

// staleUnitsFromDetails reads the stale-unit list tolerantly.
//
// A Result reaches RecoveryActions either straight from Run (where the value is
// a []string) or after a round trip through JSON storage or the API (where the
// same value is a []interface{} of strings). Asserting only the first shape
// silently offers no recovery action for exactly the persisted results the
// auto-heal path actually feeds in.
func staleUnitsFromDetails(details map[string]interface{}) []string {
	if details == nil {
		return nil
	}
	switch value := details["staleUnits"].(type) {
	case []string:
		return value
	case []interface{}:
		out := make([]string, 0, len(value))
		for _, item := range value {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

var (
	_ checks.Check         = (*StaleServiceBinaryCheck)(nil)
	_ checks.HealableCheck = (*StaleServiceBinaryCheck)(nil)
)
