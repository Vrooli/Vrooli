// Package system provides system-level health checks
// [REQ:SYSTEM-ZOMBIES-001] [REQ:HEAL-ACTION-001] [REQ:TEST-SEAM-001]
package system

import (
	"context"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"vrooli-autoheal/internal/checks"
	"vrooli-autoheal/internal/platform"
)

// ZombieCheck detects zombie (defunct) processes that indicate resource leaks.
type ZombieCheck struct {
	warningThreshold  int // count
	criticalThreshold int // count
	procReader        checks.ProcReader
	executor          checks.CommandExecutor
}

// ZombieCheckOption configures a ZombieCheck.
type ZombieCheckOption func(*ZombieCheck)

// WithZombieThresholds sets warning and critical thresholds (process counts).
func WithZombieThresholds(warning, critical int) ZombieCheckOption {
	return func(c *ZombieCheck) {
		c.warningThreshold = warning
		c.criticalThreshold = critical
	}
}

// WithZombieProcReader sets the proc reader (for testing).
// [REQ:TEST-SEAM-001]
func WithZombieProcReader(reader checks.ProcReader) ZombieCheckOption {
	return func(c *ZombieCheck) {
		c.procReader = reader
	}
}

// WithZombieExecutor sets the command executor (for testing).
// [REQ:TEST-SEAM-001]
func WithZombieExecutor(executor checks.CommandExecutor) ZombieCheckOption {
	return func(c *ZombieCheck) {
		c.executor = executor
	}
}

// NewZombieCheck creates a zombie process check.
// Default thresholds: warning at 5 zombies, critical at 20 zombies
func NewZombieCheck(opts ...ZombieCheckOption) *ZombieCheck {
	c := &ZombieCheck{
		warningThreshold:  5,
		criticalThreshold: 20,
		procReader:        checks.DefaultProcReader,
		executor:          checks.DefaultExecutor,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *ZombieCheck) ID() string    { return "system-zombies" }
func (c *ZombieCheck) Title() string { return "Zombie Processes" }
func (c *ZombieCheck) Description() string {
	return "Detects zombie (defunct) processes that indicate resource leaks"
}
func (c *ZombieCheck) Importance() string {
	return "Zombie processes indicate parent processes not reaping children, which can exhaust process table"
}
func (c *ZombieCheck) Category() checks.Category  { return checks.CategorySystem }
func (c *ZombieCheck) IntervalSeconds() int       { return 300 }
func (c *ZombieCheck) Platforms() []platform.Type { return []platform.Type{platform.Linux} }

func (c *ZombieCheck) Run(ctx context.Context) checks.Result {
	result := checks.Result{
		CheckID: c.ID(),
		Details: make(map[string]interface{}),
	}

	if runtime.GOOS == "windows" {
		result.Status = checks.StatusOK
		result.Message = "Zombie check not applicable on Windows"
		result.Details["platform"] = "windows"
		return result
	}

	zombies, zombieInfo, err := c.countZombies()
	if err != nil {
		result.Status = checks.StatusCritical
		result.Message = "Failed to count zombie processes"
		result.Details["error"] = err.Error()
		return result
	}

	result.Details["zombieCount"] = zombies
	result.Details["warningThreshold"] = c.warningThreshold
	result.Details["criticalThreshold"] = c.criticalThreshold

	if len(zombieInfo) > 0 {
		// Limit to first 10 zombies in details
		limit := 10
		if len(zombieInfo) < limit {
			limit = len(zombieInfo)
		}
		result.Details["zombies"] = zombieInfo[:limit]
	}

	// Calculate score
	score := 100
	if zombies > 0 {
		// Reduce score proportionally
		score = 100 - (zombies * 5) // -5 points per zombie
		if score < 0 {
			score = 0
		}
	}
	result.Metrics = &checks.HealthMetrics{
		Score: &score,
		SubChecks: []checks.SubCheck{
			{
				Name:   "zombie-count",
				Passed: zombies < c.criticalThreshold,
				Detail: fmt.Sprintf("%d zombie processes detected", zombies),
			},
		},
	}

	switch {
	case zombies >= c.criticalThreshold:
		result.Status = checks.StatusCritical
		result.Message = fmt.Sprintf("Critical: %d zombie processes detected", zombies)
	case zombies >= c.warningThreshold:
		result.Status = checks.StatusWarning
		result.Message = fmt.Sprintf("Warning: %d zombie processes detected", zombies)
	case zombies > 0:
		result.Status = checks.StatusOK
		result.Message = fmt.Sprintf("%d zombie processes (below threshold)", zombies)
	default:
		result.Status = checks.StatusOK
		result.Message = "No zombie processes detected"
	}

	return result
}

// zombieProcessInfo contains information about a zombie process
type zombieProcessInfo struct {
	PID  string `json:"pid"`
	PPID string `json:"ppid"`
	Name string `json:"name"`
}

// countZombies counts zombie processes using the injected ProcReader.
// [REQ:TEST-SEAM-001]
func (c *ZombieCheck) countZombies() (int, []zombieProcessInfo, error) {
	processes, err := c.procReader.ListProcesses()
	if err != nil {
		return 0, nil, err
	}

	var zombies []zombieProcessInfo
	count := 0

	for _, proc := range processes {
		if proc.State == "Z" {
			count++
			zombies = append(zombies, zombieProcessInfo{
				PID:  strconv.Itoa(proc.PID),
				PPID: strconv.Itoa(proc.PPid),
				Name: proc.Comm,
			})
		}
	}

	return count, zombies, nil
}

// RecoveryActions returns available recovery actions for zombie cleanup
// [REQ:HEAL-ACTION-001]
func (c *ZombieCheck) RecoveryActions(lastResult *checks.Result) []checks.RecoveryAction {
	hasZombies := false
	if lastResult != nil {
		if count, ok := lastResult.Details["zombieCount"].(int); ok {
			hasZombies = count > 0
		}
	}

	return []checks.RecoveryAction{
		{
			ID:          "reap",
			Name:        "Reap Zombies",
			Description: "Send SIGCHLD to parent processes to trigger zombie cleanup",
			Dangerous:   false,
			Available:   hasZombies,
		},
		{
			ID:          "list",
			Name:        "List Zombies",
			Description: "Show current zombie processes and their parents",
			Dangerous:   false,
			Available:   true,
		},
	}
}

// ExecuteAction runs the specified recovery action
// [REQ:HEAL-ACTION-001]
func (c *ZombieCheck) ExecuteAction(ctx context.Context, actionID string) checks.ActionResult {
	start := time.Now()
	result := checks.ActionResult{
		ActionID:  actionID,
		CheckID:   c.ID(),
		Timestamp: start,
	}

	switch actionID {
	case "reap":
		return c.executeReap(ctx, start)
	case "list":
		return c.executeList(ctx, start)
	default:
		result.Success = false
		result.Error = "unknown action: " + actionID
		result.Duration = time.Since(start)
		return result
	}
}

// executeReap sends SIGCHLD to parent processes of zombies to trigger reaping
func (c *ZombieCheck) executeReap(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "reap",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	// Get zombie parent PIDs
	parentPIDs, err := c.getZombieParentPIDs()
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}

	if len(parentPIDs) == 0 {
		result.Success = true
		result.Message = "No zombie parent processes to signal"
		result.Duration = time.Since(start)
		return result
	}

	// Send SIGCHLD to each parent (except init)
	var signaled []int
	var failed []int
	for _, ppid := range parentPIDs {
		if ppid <= 1 {
			continue // Never signal init
		}
		if err := syscall.Kill(ppid, syscall.SIGCHLD); err != nil {
			failed = append(failed, ppid)
		} else {
			signaled = append(signaled, ppid)
		}
	}

	// Wait briefly for parents to reap
	time.Sleep(2 * time.Second)

	// Check remaining zombies
	remainingCount, _, _ := c.countZombies()

	result.Duration = time.Since(start)
	result.Output = fmt.Sprintf("Signaled %d parent processes, %d failed\nRemaining zombies: %d",
		len(signaled), len(failed), remainingCount)

	if remainingCount < c.criticalThreshold {
		result.Success = true
		result.Message = "Zombie cleanup successful"
	} else {
		result.Success = false
		result.Message = "Zombie cleanup incomplete"
		result.Error = fmt.Sprintf("%d zombies still remain", remainingCount)
	}

	return result
}

// executeList returns a list of current zombie processes
func (c *ZombieCheck) executeList(ctx context.Context, start time.Time) checks.ActionResult {
	result := checks.ActionResult{
		ActionID:  "list",
		CheckID:   c.ID(),
		Timestamp: start,
	}

	// Use ps to get zombie process info via executor
	output, err := c.executor.Output(ctx, "ps", "aux")
	if err != nil {
		result.Success = false
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}

	// Filter for zombie processes (state Z)
	var zombieLines []string
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		zombieLines = append(zombieLines, lines[0]) // Header
	}
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) >= 8 && strings.Contains(fields[7], "Z") {
			zombieLines = append(zombieLines, line)
		}
	}

	result.Duration = time.Since(start)
	result.Success = true
	result.Message = fmt.Sprintf("Found %d zombie processes", len(zombieLines)-1)
	result.Output = strings.Join(zombieLines, "\n")
	return result
}

// getZombieParentPIDs returns unique parent PIDs of all zombie processes
func (c *ZombieCheck) getZombieParentPIDs() ([]int, error) {
	_, zombieInfo, err := c.countZombies()
	if err != nil {
		return nil, err
	}

	// Collect unique parent PIDs
	parentSet := make(map[int]bool)
	for _, z := range zombieInfo {
		if pid, err := strconv.Atoi(z.PPID); err == nil && pid > 1 {
			parentSet[pid] = true
		}
	}

	var parents []int
	for pid := range parentSet {
		parents = append(parents, pid)
	}
	return parents, nil
}

// Ensure ZombieCheck implements HealableCheck
var _ checks.HealableCheck = (*ZombieCheck)(nil)
