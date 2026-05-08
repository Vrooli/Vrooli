// Package system: recent Machine Check Exception (MCE) check.
//
// MCEs are CPU-reported hardware errors. Uncorrected MCEs are the closest
// thing to a smoking gun for unexplained resets. We surface them via
// rasdaemon's `ras-mc-ctl --summary --since="1 hour ago"` rather than reading
// the SQLite database directly — keeps autoheal CGo-free.
package system

import (
	"context"
	"fmt"
	"regexp"
	"runtime"
	"strconv"
	"time"
	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// MCERecentCheck reports MCE activity from the last hour.
type MCERecentCheck struct {
	executor       checks.CommandExecutor
	correctedWarn  int
	rasdaemonProbe func(ctx context.Context, exec checks.CommandExecutor) bool
}

// MCERecentCheckOption configures an MCERecentCheck.
type MCERecentCheckOption func(*MCERecentCheck)

// WithMCEExecutor injects a CommandExecutor (for testing).
func WithMCEExecutor(e checks.CommandExecutor) MCERecentCheckOption {
	return func(c *MCERecentCheck) { c.executor = e }
}

// WithMCECorrectedWarnThreshold sets the corrected-error count above which
// the check transitions to WARNING.
func WithMCECorrectedWarnThreshold(n int) MCERecentCheckOption {
	return func(c *MCERecentCheck) { c.correctedWarn = n }
}

// NewMCERecentCheck builds the check.
func NewMCERecentCheck(opts ...MCERecentCheckOption) *MCERecentCheck {
	c := &MCERecentCheck{
		executor:       checks.DefaultExecutor,
		correctedWarn:  5,
		rasdaemonProbe: probeRasMCCtl,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *MCERecentCheck) ID() string    { return "system-mce-recent" }
func (c *MCERecentCheck) Title() string { return "Recent Machine Check Exceptions" }
func (c *MCERecentCheck) Description() string {
	return "Surfaces hardware errors reported via rasdaemon's MCE summary"
}

func (c *MCERecentCheck) Importance() string {
	return "Uncorrected MCEs strongly correlate with unexplained resets and impending hardware failure"
}
func (c *MCERecentCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *MCERecentCheck) IntervalSeconds() int       { return 300 }
func (c *MCERecentCheck) Platforms() []platform.Type { return []platform.Type{platform.Linux} }

func probeRasMCCtl(ctx context.Context, exec checks.CommandExecutor) bool {
	_, err := exec.CombinedOutput(ctx, "ras-mc-ctl", "--help")
	return err == nil
}

var (
	correctedRE   = regexp.MustCompile(`(?m)^\s*([0-9]+)\s+(?i:Corrected\s+errors?)`)
	uncorrectedRE = regexp.MustCompile(`(?m)^\s*([0-9]+)\s+(?i:Uncorrected\s+errors?)`)
)

func (c *MCERecentCheck) Run(ctx context.Context) checks.Result {
	r := checks.Result{CheckID: c.ID(), Details: map[string]interface{}{}}
	if runtime.GOOS != "linux" {
		r.Status = checks.StatusOK
		r.Message = "MCE reporting is Linux-only"
		return r
	}

	if !c.rasdaemonProbe(ctx, c.executor) {
		r.Status = checks.StatusWarning
		r.Message = "rasdaemon not installed; MCE reporting unavailable"
		r.Details["rasdaemonAvailable"] = false
		return r
	}
	r.Details["rasdaemonAvailable"] = true

	out, err := c.executor.CombinedOutput(ctx, "ras-mc-ctl", "--summary", "--since=1 hour ago")
	if err != nil {
		r.Status = checks.StatusWarning
		r.Message = "ras-mc-ctl --summary failed: " + err.Error()
		r.Details["error"] = err.Error()
		r.Details["output"] = string(out)
		return r
	}
	corrected := extractCount(correctedRE, out)
	uncorrected := extractCount(uncorrectedRE, out)
	r.Details["correctedErrors"] = corrected
	r.Details["uncorrectedErrors"] = uncorrected

	switch {
	case uncorrected > 0:
		r.Status = checks.StatusCritical
		r.Message = fmt.Sprintf("%d uncorrected MCE(s) in last hour — hardware failure suspected", uncorrected)
	case corrected > c.correctedWarn:
		r.Status = checks.StatusWarning
		r.Message = fmt.Sprintf("%d corrected MCE(s) in last hour — early sign of hardware degradation", corrected)
	default:
		r.Status = checks.StatusOK
		r.Message = "No recent MCEs"
	}
	r.Timestamp = time.Now()
	return r
}

func extractCount(re *regexp.Regexp, out []byte) int {
	m := re.FindSubmatch(out)
	if len(m) < 2 {
		return 0
	}
	n, _ := strconv.Atoi(string(m[1]))
	return n
}

// RecoveryActions returns diagnostic-only actions; nothing destructive.
// Clearing rasdaemon's DB or resetting EDAC counters destroys forensic data.
func (c *MCERecentCheck) RecoveryActions(*checks.Result) []checks.RecoveryAction {
	return []checks.RecoveryAction{
		{ID: "show-summary", Name: "Show MCE summary", Description: "Run ras-mc-ctl --summary", Available: true},
		{ID: "show-recent-errors", Name: "Show recent error detail", Description: "Run ras-mc-ctl --errors --since='1 hour ago'", Available: true},
		{ID: "enable-rasdaemon-service", Name: "Enable rasdaemon", Description: "systemctl enable --now rasdaemon", Available: true},
		{ID: "dump-edac-counters", Name: "Dump EDAC counters", Description: "Cat /sys/devices/system/edac/mc/mc*/ce_count", Available: true},
	}
}

func (c *MCERecentCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	res := checks.ActionResult{ActionID: actionID, CheckID: c.ID(), Timestamp: start}
	var out []byte
	var err error
	switch actionID {
	case "show-summary":
		out, err = c.executor.CombinedOutput(ctx, "ras-mc-ctl", "--summary")
	case "show-recent-errors":
		out, err = c.executor.CombinedOutput(ctx, "ras-mc-ctl", "--errors", "--since=1 hour ago")
	case "enable-rasdaemon-service":
		out, err = c.executor.CombinedOutput(ctx, "sudo", "systemctl", "enable", "--now", "rasdaemon")
	case "dump-edac-counters":
		out, err = c.executor.CombinedOutput(ctx, "sh", "-c", "cat /sys/devices/system/edac/mc/mc*/ce_count 2>/dev/null || echo 'no MCs registered'")
	default:
		res.Success = false
		res.Error = "unknown action: " + actionID
		res.Duration = time.Since(start)
		return res
	}
	res.Output = string(out)
	res.Duration = time.Since(start)
	if err != nil {
		res.Success = false
		res.Error = err.Error()
		res.Message = "Action failed"
		return res
	}
	res.Success = true
	res.Message = "Action completed"
	return res
}
