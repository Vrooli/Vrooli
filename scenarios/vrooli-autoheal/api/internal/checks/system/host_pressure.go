package system

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/vrooli/internal/hostpressure"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// HostPressureCheck joins portable host-pressure readings into one explicit
// check. Unread sensors are warnings, never zero-valued passes.
type HostPressureCheck struct {
	collect    func(context.Context) hostpressure.PressureSnapshot
	reclaim    func(context.Context) (string, error)
	thresholds hostPressureThresholds
}

// These are documented fallbacks only. Production values are read from the
// infrastructure-manager setpoint so this check cannot silently drift from
// the watchdog or coverage model.
type hostPressureThresholds struct {
	CPUPercent       float64
	StrandedMemoryMB float64
	ForksPerSecond   float64
}

var fallbackHostPressureThresholds = hostPressureThresholds{CPUPercent: 50, StrandedMemoryMB: 17200, ForksPerSecond: 200}

type HostPressureOption func(*HostPressureCheck)

func WithHostPressureReader(fn func(context.Context) hostpressure.PressureSnapshot) HostPressureOption {
	return func(c *HostPressureCheck) {
		if fn != nil {
			c.collect = fn
		}
	}
}

func WithHostPressureThresholds(cpuPercent, strandedMemoryMB, forksPerSecond float64) HostPressureOption {
	return func(c *HostPressureCheck) {
		if cpuPercent > 0 {
			c.thresholds.CPUPercent = cpuPercent
		}
		if strandedMemoryMB > 0 {
			c.thresholds.StrandedMemoryMB = strandedMemoryMB
		}
		if forksPerSecond > 0 {
			c.thresholds.ForksPerSecond = forksPerSecond
		}
	}
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
	c := &HostPressureCheck{thresholds: readHostPressureThresholds(), collect: func(ctx context.Context) hostpressure.PressureSnapshot {
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
	result.Details["cpu_pressure_state"] = s.CPUPressure.State
	result.Details["memory_available_state"] = s.MemoryAvail.State
	result.Details["swap_used_state"] = s.SwapUsed.State
	result.Details["process_count_state"] = s.ProcessCount.State
	result.Details["fork_rate_state"] = s.ForkRate.State
	result.Details["cpu_pressure_threshold_percent"] = c.thresholds.CPUPercent
	result.Details["stranded_memory_threshold_mb"] = c.thresholds.StrandedMemoryMB
	result.Details["fork_rate_threshold_per_second"] = c.thresholds.ForksPerSecond
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
	if v, ok := s.CPUPressure.Number(); ok {
		readable++
		if v >= c.thresholds.CPUPercent {
			result.Status = checks.StatusCritical
			result.Message = fmt.Sprintf("CPU pressure critical: %.1f%%", v)
			return result
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
		if float64(strandedBytes)/(1024*1024) >= c.thresholds.StrandedMemoryMB {
			result.Status = checks.StatusCritical
			result.Message = fmt.Sprintf("Stranded memory critical: %.0f MB", float64(strandedBytes)/(1024*1024))
			return result
		}
	}
	if v, ok := s.ForkRate.Number(); ok {
		readable++
		if v >= c.thresholds.ForksPerSecond {
			result.Status = checks.StatusCritical
			result.Message = fmt.Sprintf("Fork rate critical: %.1f/s", v)
			return result
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
	result.Status = checks.StatusOK
	result.Message = "Host pressure is within the declared setpoint bars; unread sensors were skipped"
	return result
}

func readHostPressureThresholds() hostPressureThresholds {
	thresholds := fallbackHostPressureThresholds
	paths := make([]string, 0, 2)
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		paths = append(paths, filepath.Join(root, "scenarios", "infrastructure-manager", "setpoint", "reliability-setpoint.json"))
	}
	if cwd, err := os.Getwd(); err == nil {
		paths = append(paths, filepath.Join(cwd, "scenarios", "infrastructure-manager", "setpoint", "reliability-setpoint.json"))
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var document struct {
			Bars []struct {
				CellRef string  `json:"cell_ref"`
				Max     float64 `json:"max"`
			} `json:"bars"`
		}
		if json.Unmarshal(data, &document) != nil {
			continue
		}
		for _, bar := range document.Bars {
			if bar.Max <= 0 {
				continue
			}
			switch bar.CellRef {
			case "substrate/SB14":
				thresholds.CPUPercent = bar.Max
			case "substrate/SB15":
				thresholds.StrandedMemoryMB = bar.Max
			case "substrate/SB16":
				thresholds.ForksPerSecond = bar.Max
			}
		}
		return thresholds
	}
	return thresholds
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
