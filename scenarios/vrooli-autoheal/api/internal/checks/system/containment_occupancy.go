// Package system provides system-level health checks
// [REQ:SYSTEM-CONTAINMENT-OCCUPANCY-001] [REQ:TEST-SEAM-001]
package system

import (
	"context"
	"fmt"
	"sort"
	"strings"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/agentscope"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/platform"
)

// ContainmentOccupancyCheckID is this check's id.
const ContainmentOccupancyCheckID = "system-containment-occupancy"

// ContainmentOccupancyCheck reports how full the containment ceilings are.
//
// The polite-host plan gave agent sessions a task and memory ceiling and gave
// nobody a reading of how full it was. Every other sensor on this host watches
// the host — CPU pressure, fork rate, swap, memory stalls — and the failure
// this ceiling introduced happens inside the cgroup, where none of them look.
// On 2026-09-04 the agent slice reached 4,059 of its 4,096 tasks with every
// host-level bar green; the kernel then refused the fork of every new session,
// and the first thing the operator saw was an unrelated component reporting
// its own database corrupt.
//
// A ceiling is a safety property only while there is room under it. This check
// is the other side of the ratio.
type ContainmentOccupancyCheck struct {
	slices            []string
	warningThreshold  int
	criticalThreshold int
	reader            OccupancyReader
}

// OccupancyReader is the kernel read, injected so the check is tested without
// a cgroup tree.
// [REQ:TEST-SEAM-001]
type OccupancyReader interface {
	// SliceCgroup resolves a slice unit to its cgroup path.
	SliceCgroup(slice string) (string, error)
	// Occupancy reads how full one cgroup's ceilings are.
	Occupancy(cgroupPath string) (platformgo.Occupancy, error)
	// Census lists the session scopes a slice holds.
	Census(cgroupPath string) ([]agentscope.Entry, error)
}

type hostOccupancyReader struct{}

func (hostOccupancyReader) SliceCgroup(slice string) (string, error) {
	return platformgo.SliceCgroup(slice)
}

func (hostOccupancyReader) Occupancy(cgroupPath string) (platformgo.Occupancy, error) {
	return platformgo.ScopeOccupancy(agentscope.Ref(cgroupPath))
}

func (hostOccupancyReader) Census(cgroupPath string) ([]agentscope.Entry, error) {
	return agentscope.NewReader().Census(agentscope.Ref(cgroupPath))
}

// DefaultOccupancyReader reads this host.
var DefaultOccupancyReader OccupancyReader = hostOccupancyReader{}

// ContainmentOccupancyOption configures the check.
type ContainmentOccupancyOption func(*ContainmentOccupancyCheck)

// WithOccupancySlices sets the slices to read.
func WithOccupancySlices(slices []string) ContainmentOccupancyOption {
	return func(c *ContainmentOccupancyCheck) { c.slices = slices }
}

// WithOccupancyThresholds sets the warning and critical fullness percentages.
func WithOccupancyThresholds(warning, critical int) ContainmentOccupancyOption {
	return func(c *ContainmentOccupancyCheck) {
		c.warningThreshold = warning
		c.criticalThreshold = critical
	}
}

// WithOccupancyReader sets the kernel read.
// [REQ:TEST-SEAM-001]
func WithOccupancyReader(reader OccupancyReader) ContainmentOccupancyOption {
	return func(c *ContainmentOccupancyCheck) { c.reader = reader }
}

// NewContainmentOccupancyCheck creates the check.
// Default slices: the agent slice and the services slice.
// Default thresholds: warning at 70%, critical at 85%, matching the
// agent_session_containment safeguard's typed config.
func NewContainmentOccupancyCheck(opts ...ContainmentOccupancyOption) *ContainmentOccupancyCheck {
	c := &ContainmentOccupancyCheck{
		slices:            []string{agentscope.Slice, platformgo.ServicesSlice},
		warningThreshold:  70,
		criticalThreshold: 85,
		reader:            DefaultOccupancyReader,
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.reader == nil {
		c.reader = DefaultOccupancyReader
	}
	return c
}

func (c *ContainmentOccupancyCheck) ID() string    { return ContainmentOccupancyCheckID }
func (c *ContainmentOccupancyCheck) Title() string { return "Containment Ceiling Occupancy" }

func (c *ContainmentOccupancyCheck) Description() string {
	return "Reports how full the agent and service containment ceilings are, and which session scopes hold them"
}

func (c *ContainmentOccupancyCheck) Importance() string {
	return "At its task ceiling the kernel refuses the fork of every new agent session with no warning of its own, and the failure surfaces as an unrelated component's startup error"
}

func (c *ContainmentOccupancyCheck) Category() checks.Category { return checks.CategorySystem }
func (c *ContainmentOccupancyCheck) IntervalSeconds() int      { return 60 }
func (c *ContainmentOccupancyCheck) Platforms() []platform.Type {
	return []platform.Type{platform.Linux}
}

func (c *ContainmentOccupancyCheck) Run(context.Context) checks.Result {
	result := checks.Result{CheckID: c.ID(), Details: make(map[string]interface{})}
	var subChecks []checks.SubCheck
	worst := checks.StatusOK
	sliceDetails := make([]map[string]interface{}, 0, len(c.slices))
	messages := make([]string, 0, len(c.slices))

	for _, slice := range c.slices {
		reading, status, detail, message := c.readSlice(slice)
		worst = checks.WorstStatus(worst, status)
		subChecks = append(subChecks, checks.SubCheck{Name: slice, Passed: status != checks.StatusCritical, Detail: detail})
		sliceDetails = append(sliceDetails, reading)
		if message != "" {
			messages = append(messages, message)
		}
	}

	result.Status = worst
	result.Metrics = &checks.HealthMetrics{SubChecks: subChecks}
	result.Details["slices"] = sliceDetails
	if len(messages) == 0 {
		result.Message = "Containment ceilings have room"
		return result
	}
	result.Message = strings.Join(messages, "; ")
	return result
}

// readSlice reads one slice and decides what it means.
func (c *ContainmentOccupancyCheck) readSlice(slice string) (map[string]interface{}, checks.Status, string, string) {
	detail := map[string]interface{}{"slice": slice}
	cgroupPath, err := c.reader.SliceCgroup(slice)
	if err != nil {
		// A slice this host does not have is not a failure: a host with no
		// agent sessions has no agent slice.
		detail["state"] = "absent"
		return detail, checks.StatusOK, "not present on this host", ""
	}
	detail["cgroup"] = cgroupPath
	occupancy, err := c.reader.Occupancy(cgroupPath)
	if err != nil {
		// A ceiling that cannot be read is undetermined, never room to spare.
		detail["state"] = "undetermined"
		return detail, checks.StatusWarning, "occupancy could not be read: " + err.Error(),
			fmt.Sprintf("%s occupancy is undetermined", slice)
	}
	detail["tasks"], detail["tasks_max"] = occupancy.Tasks, occupancy.TasksMax
	detail["memory_bytes"], detail["memory_max_bytes"] = occupancy.MemoryBytes, occupancy.MemoryMaxBytes
	detail["memory_high_events"] = occupancy.MemoryHighEvents

	status, detailText, message := c.judge(slice, occupancy)

	// Whatever the fullness, name what is holding it. A scope whose coding
	// agent has exited holds its share of the ceiling for as long as its
	// children run, and nothing in the session lifecycle releases it.
	if entries, censusErr := c.reader.Census(cgroupPath); censusErr == nil {
		orphans := agentscope.Agentless(entries)
		detail["scopes"], detail["agentless_scopes"] = len(entries), len(orphans)
		detail["agentless_tasks"] = agentscope.AgentlessTasks(entries)
		if len(orphans) > 0 {
			detail["agentless"] = agentlessNames(orphans)
			held := agentscope.AgentlessTasks(entries)
			note := fmt.Sprintf("%d of %d session scopes hold %d tasks with no coding agent left in them", len(orphans), len(entries), held)
			detailText = strings.TrimSpace(detailText + "; " + note)
			if status == checks.StatusOK {
				message = fmt.Sprintf("%s: %s", slice, note)
				status = checks.StatusWarning
			}
		}
	}
	return detail, status, detailText, message
}

// judge turns a reading into a status. A full ceiling is not a misconfigured
// ceiling, so this never asks anyone to re-apply the safeguard.
func (c *ContainmentOccupancyCheck) judge(slice string, occupancy platformgo.Occupancy) (checks.Status, string, string) {
	tasks := occupancy.TaskSaturation()
	memory := occupancy.MemorySaturation()
	warn, critical := float64(c.warningThreshold)/100, float64(c.criticalThreshold)/100

	switch {
	case tasks >= critical:
		return checks.StatusCritical,
			fmt.Sprintf("%d of %d tasks (%.0f%%)", occupancy.Tasks, occupancy.TasksMax, tasks*100),
			fmt.Sprintf("%s is at %.0f%% of its task ceiling; above it the kernel refuses every new session's first fork", slice, tasks*100)
	case memory >= critical:
		return checks.StatusCritical,
			fmt.Sprintf("%d of %d memory bytes (%.0f%%)", occupancy.MemoryBytes, occupancy.MemoryMaxBytes, memory*100),
			fmt.Sprintf("%s is at %.0f%% of its memory ceiling", slice, memory*100)
	case tasks >= warn:
		return checks.StatusWarning,
			fmt.Sprintf("%d of %d tasks (%.0f%%)", occupancy.Tasks, occupancy.TasksMax, tasks*100),
			fmt.Sprintf("%s is at %.0f%% of its task ceiling", slice, tasks*100)
	case memory >= warn:
		return checks.StatusWarning,
			fmt.Sprintf("%d of %d memory bytes (%.0f%%)", occupancy.MemoryBytes, occupancy.MemoryMaxBytes, memory*100),
			fmt.Sprintf("%s is at %.0f%% of its memory ceiling", slice, memory*100)
	}
	return checks.StatusOK, fmt.Sprintf("%d of %d tasks", occupancy.Tasks, occupancy.TasksMax), ""
}

// agentlessNames lists orphaned scopes worst first, so the operator reads the
// one worth acting on before the ones that hardly matter.
func agentlessNames(entries []agentscope.Entry) []string {
	sorted := append([]agentscope.Entry(nil), entries...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Tasks > sorted[j].Tasks })
	names := make([]string, 0, len(sorted))
	for _, entry := range sorted {
		names = append(names, fmt.Sprintf("%s (%d tasks)", entry.Name, entry.Tasks))
	}
	return names
}
