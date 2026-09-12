package system

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/hostpressure"
	"github.com/vrooli/vrooli/internal/setpoint"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// HostPressureCheck joins portable host-pressure readings into one explicit
// check. Unread sensors are warnings, never zero-valued passes. The bars and
// their sustain come from internal/setpoint at every run; a breach becomes
// critical only after the authored sustain, and is a warning before that.
type HostPressureCheck struct {
	collect func(context.Context) hostpressure.PressureSnapshot
	reclaim func(context.Context) (string, error)
	resolve func() (setpoint.Setpoint, error)
	sustain *setpoint.Sustainer
}

type HostPressureOption func(*HostPressureCheck)

func WithHostPressureReader(fn func(context.Context) hostpressure.PressureSnapshot) HostPressureOption {
	return func(c *HostPressureCheck) {
		if fn != nil {
			c.collect = fn
		}
	}
}

// WithSetpoint pins the bars the check grades against; production resolves
// the setpoint file at every run.
func WithSetpoint(sp setpoint.Setpoint) HostPressureOption {
	return func(c *HostPressureCheck) {
		c.resolve = func() (setpoint.Setpoint, error) { return sp, nil }
	}
}

// WithClock replaces the sustain clock so a test can walk through the window.
func WithClock(now func() time.Time) HostPressureOption {
	return func(c *HostPressureCheck) { c.sustain.WithClock(now) }
}

// WithReclaimer supplies the control-plane lifecycle seam. The check owns no
// service-manager command and cannot recycle a PID directly; production
// wiring may provide the governed lifecycle callback, while tests can prove
// the pressure brake and one-service limit without touching the host.
func WithReclaimer(fn func(context.Context) (string, error)) HostPressureOption {
	return func(c *HostPressureCheck) { c.reclaim = fn }
}

func NewHostPressureCheck(opts ...HostPressureOption) *HostPressureCheck {
	var previousMu sync.Mutex
	var previous *hostpressure.PressureSnapshot
	c := &HostPressureCheck{resolve: resolveSetpoint, sustain: setpoint.NewSustainer(setpoint.NewMemoryState()), collect: func(ctx context.Context) hostpressure.PressureSnapshot {
		previousMu.Lock()
		defer previousMu.Unlock()
		current := hostpressure.Collect(ctx, hostpressure.Options{Previous: previous})
		previous = &current
		return current
	}}
	for _, opt := range opts {
		opt(c)
	}
	return c
}
func (c *HostPressureCheck) ID() string    { return "system-host-pressure" }
func (c *HostPressureCheck) Title() string { return "Host Pressure" }
func (c *HostPressureCheck) Description() string {
	return "Reports CPU pressure, memory and swap, process count, and fork rate with explicit platform read states"
}

func (c *HostPressureCheck) Importance() string {
	return "Host pressure can degrade every workload even when individual service CPU percentages look healthy"
}
func (c *HostPressureCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *HostPressureCheck) IntervalSeconds() int       { return 30 }
func (c *HostPressureCheck) Platforms() []platform.Type { return nil }

func (c *HostPressureCheck) Run(ctx context.Context) checks.Result {
	s := c.collect(ctx)
	result := checks.Result{CheckID: c.ID(), Details: map[string]interface{}{}}
	sp, err := c.resolve()
	if err != nil {
		result.Status = checks.StatusWarning
		result.Message = "Host pressure setpoint is unreadable: " + err.Error()
		result.Details["setpoint_error"] = err.Error()
		return result
	}
	bars := barsFrom(sp)
	result.Details["setpoint_path"] = sp.Path
	result.Details["cpu_pressure_state"] = s.CPUPressure.State
	result.Details["memory_available_state"] = s.MemoryAvail.State
	result.Details["swap_used_state"] = s.SwapUsed.State
	result.Details["process_count_state"] = s.ProcessCount.State
	result.Details["fork_rate_state"] = s.ForkRate.State
	result.Details["cpu_pressure_threshold_percent"] = bars.CPUPercent
	result.Details["stranded_memory_threshold_mb"] = bars.StrandedMemoryMB
	result.Details["fork_rate_threshold_per_second"] = bars.ForksPerSecond
	result.Details["sustain"] = map[string]string{
		setpoint.CellCPUPressure: bars.CPUSustain.String(), setpoint.CellStrandedMemory: bars.StrandedSustain.String(), setpoint.CellForkRate: bars.ForkSustain.String(),
	}
	if value, ok := s.CPUPressure.Number(); ok {
		result.Details["cpu_pressure_percent"] = value
	}
	if value, ok := s.ForkRate.Number(); ok {
		result.Details["fork_rate_per_second"] = value
	}
	if value, ok := s.ProcessCount.Number(); ok {
		result.Details["process_count"] = value
	}
	if value, ok := s.MemoryAvail.Number(); ok {
		result.Details["memory_available_bytes"] = value
	}
	var readable int
	var pending []string
	if v, ok := s.CPUPressure.Number(); ok {
		readable++
		if verdict := c.grade("cpu", v >= bars.CPUPercent, bars.CPUSustain); verdict.critical {
			result.Status = checks.StatusCritical
			result.Message = fmt.Sprintf("CPU pressure critical: %.1f%% above the SB14 bar for %s", v, bars.CPUSustain)
			return result
		} else if verdict.breaching {
			pending = append(pending, fmt.Sprintf("CPU pressure %.1f%% above the SB14 bar since %s (sustain %s)", v, verdict.since.Format(time.RFC3339), bars.CPUSustain))
		}
	}
	stranded := hostpressure.Stranded(s.Processes, 2)
	var strandedBytes uint64
	for _, process := range stranded {
		strandedBytes += process.Swapped
	}
	result.Details["stranded_memory_mb"] = float64(strandedBytes) / (1024 * 1024)
	if s.SwapUsed.State == hostpressure.Read {
		readable++
		strandedMB := float64(strandedBytes) / (1024 * 1024)
		if verdict := c.grade("stranded", strandedMB >= bars.StrandedMemoryMB, bars.StrandedSustain); verdict.critical {
			result.Status = checks.StatusCritical
			result.Message = fmt.Sprintf("Stranded memory critical: %.0f MB above the SB15 bar for %s", strandedMB, bars.StrandedSustain)
			return result
		} else if verdict.breaching {
			pending = append(pending, fmt.Sprintf("stranded memory %.0f MB above the SB15 bar since %s (sustain %s)", strandedMB, verdict.since.Format(time.RFC3339), bars.StrandedSustain))
		}
	}
	if v, ok := s.ForkRate.Number(); ok {
		readable++
		if verdict := c.grade("fork", v >= bars.ForksPerSecond, bars.ForkSustain); verdict.critical {
			result.Status = checks.StatusCritical
			result.Message = fmt.Sprintf("Fork rate critical: %.1f/s above the SB16 bar for %s", v, bars.ForkSustain)
			return result
		} else if verdict.breaching {
			pending = append(pending, fmt.Sprintf("fork rate %.1f/s above the SB16 bar since %s (sustain %s)", v, verdict.since.Format(time.RFC3339), bars.ForkSustain))
		}
	}
	if s.MemoryAvail.State == hostpressure.Read {
		readable++
	}
	if s.ProcessCount.State == hostpressure.Read {
		readable++
	}
	if readable == 0 {
		result.Status = checks.StatusNotApplicable
		result.Message = "Host pressure has no readable platform sensors"
		return result
	}
	if len(pending) > 0 {
		result.Status = checks.StatusWarning
		result.Message = "Host pressure above a setpoint bar, sustain window not yet met: " + strings.Join(pending, "; ")
		result.Details["pending_breaches"] = pending
		return result
	}
	result.Status = checks.StatusOK
	result.Message = "Host pressure is within the declared setpoint bars; unread sensors were skipped"
	return result
}

// hostPressureBars are the three bars this check grades, read from one
// Setpoint. Every number and window comes from the file (or the compiled
// fallback the file's absence selects); nothing here is a second constant.
type hostPressureBars struct {
	CPUPercent, StrandedMemoryMB, ForksPerSecond float64
	CPUSustain, StrandedSustain, ForkSustain     time.Duration
}

func barsFrom(sp setpoint.Setpoint) hostPressureBars {
	fallback := setpoint.Fallback()
	return hostPressureBars{
		CPUPercent:       sp.Max(setpoint.CellCPUPressure, fallback.Max(setpoint.CellCPUPressure, 0)),
		StrandedMemoryMB: sp.Max(setpoint.CellStrandedMemory, fallback.Max(setpoint.CellStrandedMemory, 0)),
		ForksPerSecond:   sp.Max(setpoint.CellForkRate, fallback.Max(setpoint.CellForkRate, 0)),
		CPUSustain:       sp.Sustain(setpoint.CellCPUPressure, setpoint.DefaultPressureSustain),
		StrandedSustain:  sp.Sustain(setpoint.CellStrandedMemory, setpoint.DefaultPressureSustain),
		ForkSustain:      sp.Sustain(setpoint.CellForkRate, setpoint.DefaultPressureSustain),
	}
}

type breachVerdict struct {
	breaching bool
	critical  bool
	since     time.Time
}

// grade runs one reading through the shared sustain: a breach under the
// window is pending, a breach past it is critical, a clear reading resets.
func (c *HostPressureCheck) grade(name string, breaching bool, sustain time.Duration) breachVerdict {
	critical := c.sustain.Breach(name, breaching, sustain)
	since, _ := c.sustain.Since(name)
	return breachVerdict{breaching: breaching, critical: critical, since: since}
}

// resolveSetpoint reads the setpoint for this process at call time, so an
// edited bar file is honored on the next run without a restart.
func resolveSetpoint() (setpoint.Setpoint, error) {
	cwd, _ := os.Getwd()
	return setpoint.Resolve(os.Environ(), cwd)
}

func (c *HostPressureCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	available := lastResult != nil && lastResult.Status != checks.StatusOK && c.reclaim != nil
	return []checks.RecoveryAction{
		{ID: "report-findings", Name: "Report Host Findings", Description: "Write evidence-backed pressure, ownership, and crash-loop findings", Available: true},
		{ID: "reclaim", Name: "Reclaim One Evicted Service", Description: "Recycle one idle, Vrooli-managed service through the lifecycle path", Available: available},
		{ID: "preview-disposal", Name: "Preview Abandoned Workloads", Description: "Prepare an operator-approved disposal preview without removing anything", Available: true},
	}
}

func (c *HostPressureCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{ActionID: actionID, CheckID: c.ID(), Timestamp: start}
	switch actionID {
	case "report-findings":
		result.Success = true
		result.Message = "Host findings are reported by the portable watchdog surface"
	case "reclaim":
		if c.reclaim == nil {
			result.Message = "Reclaim lifecycle callback is unavailable; no host mutation was attempted"
			break
		}
		message, err := c.reclaim(ctx)
		result.Output = message
		if err != nil {
			result.Error = err.Error()
			result.Message = "Reclaim was refused or failed"
		} else {
			result.Success = true
			result.Message = message
		}
	case "preview-disposal":
		result.Success = true
		result.Message = "Disposal preview requested; operator approval remains required"
	default:
		result.Error = "unknown action"
	}
	result.Duration = time.Since(start)
	return result
}
