package system

import (
	"context"
	"errors"
	"strings"
	"testing"

	platformgo "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/agentscope"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

type fakeOccupancy struct {
	paths     map[string]string
	readings  map[string]platformgo.Occupancy
	entries   map[string][]agentscope.Entry
	readErr   map[string]error
	censusErr map[string]error
}

func (f fakeOccupancy) SliceCgroup(slice string) (string, error) {
	path, ok := f.paths[slice]
	if !ok {
		return "", errors.New("no cgroup for " + slice)
	}
	return path, nil
}

func (f fakeOccupancy) Occupancy(path string) (platformgo.Occupancy, error) {
	if err := f.readErr[path]; err != nil {
		return platformgo.Occupancy{}, err
	}
	return f.readings[path], nil
}

func (f fakeOccupancy) Census(path string) ([]agentscope.Entry, error) {
	if err := f.censusErr[path]; err != nil {
		return nil, err
	}
	return f.entries[path], nil
}

func occupancyCheck(reader OccupancyReader) *ContainmentOccupancyCheck {
	return NewContainmentOccupancyCheck(WithOccupancySlices([]string{"vrooli-agents.slice"}), WithOccupancyReader(reader))
}

// The 2026-09-04 reading. Above the ceiling the kernel refuses the fork of
// every new session, so this must be critical before it gets there.
func TestOccupancyIsCriticalNearTheTaskCeiling(t *testing.T) {
	reader := fakeOccupancy{
		paths:    map[string]string{"vrooli-agents.slice": "/user.slice/vrooli-agents.slice"},
		readings: map[string]platformgo.Occupancy{"/user.slice/vrooli-agents.slice": {Tasks: 4059, TasksMax: 4096, MemoryBytes: 1, MemoryMaxBytes: 100}},
	}
	result := occupancyCheck(reader).Run(context.Background())
	if result.Status != checks.StatusCritical {
		t.Fatalf("status = %v, want critical: %s", result.Status, result.Message)
	}
	if !strings.Contains(result.Message, "refuses every new session's first fork") {
		t.Fatalf("the message must say what happens at the ceiling: %q", result.Message)
	}
}

func TestOccupancyIsOKWithRoomToSpare(t *testing.T) {
	reader := fakeOccupancy{
		paths:    map[string]string{"vrooli-agents.slice": "/user.slice/vrooli-agents.slice"},
		readings: map[string]platformgo.Occupancy{"/user.slice/vrooli-agents.slice": {Tasks: 3079, TasksMax: 16384, MemoryBytes: 10, MemoryMaxBytes: 100}},
	}
	result := occupancyCheck(reader).Run(context.Background())
	if result.Status != checks.StatusOK {
		t.Fatalf("status = %v, want ok: %s", result.Status, result.Message)
	}
}

// A ceiling that cannot be read is undetermined, never room to spare. This is
// the shape that let a saturated slice read green for two days.
func TestUnreadableCeilingIsNotHealthy(t *testing.T) {
	reader := fakeOccupancy{
		paths:   map[string]string{"vrooli-agents.slice": "/user.slice/vrooli-agents.slice"},
		readErr: map[string]error{"/user.slice/vrooli-agents.slice": errors.New("permission denied")},
	}
	result := occupancyCheck(reader).Run(context.Background())
	if result.Status == checks.StatusOK {
		t.Fatalf("an unreadable ceiling must not read as healthy: %s", result.Message)
	}
	if !strings.Contains(result.Message, "undetermined") {
		t.Fatalf("message = %q, want it to say undetermined", result.Message)
	}
}

// An unlimited ceiling has no saturation; the check must not invent one.
func TestUnlimitedCeilingIsNotSaturated(t *testing.T) {
	reader := fakeOccupancy{
		paths:    map[string]string{"vrooli-agents.slice": "/user.slice/vrooli-agents.slice"},
		readings: map[string]platformgo.Occupancy{"/user.slice/vrooli-agents.slice": {Tasks: 900000, TasksMax: platformgo.Unlimited, MemoryBytes: 5, MemoryMaxBytes: platformgo.Unlimited}},
	}
	result := occupancyCheck(reader).Run(context.Background())
	if result.Status != checks.StatusOK {
		t.Fatalf("an unlimited ceiling cannot be full: %v %s", result.Status, result.Message)
	}
}

// Sessions that ended and left long-running children hold their share of the
// ceiling for as long as those children run, and nothing releases it. The
// check names them even when there is room, because that is the leak.
func TestOccupancyNamesAgentlessScopesEvenWithRoom(t *testing.T) {
	path := "/user.slice/vrooli-agents.slice"
	reader := fakeOccupancy{
		paths:    map[string]string{"vrooli-agents.slice": path},
		readings: map[string]platformgo.Occupancy{path: {Tasks: 100, TasksMax: 16384, MemoryBytes: 1, MemoryMaxBytes: 100}},
		entries: map[string][]agentscope.Entry{path: {
			{Name: "vrooli-agent-codex-live.scope", Tasks: 38},
			{Name: "vrooli-agent-codex-eb0c.scope", Tasks: 787, Agentless: true},
			{Name: "vrooli-agent-codex-small.scope", Tasks: 14, Agentless: true},
		}},
	}
	result := occupancyCheck(reader).Run(context.Background())
	if result.Status != checks.StatusWarning {
		t.Fatalf("status = %v, want warning: %s", result.Status, result.Message)
	}
	if !strings.Contains(result.Message, "2 of 3 session scopes hold 801 tasks") {
		t.Fatalf("message = %q", result.Message)
	}
	slices, _ := result.Details["slices"].([]map[string]interface{})
	if len(slices) != 1 {
		t.Fatalf("details = %+v", result.Details)
	}
	names, _ := slices[0]["agentless"].([]string)
	if len(names) != 2 || !strings.HasPrefix(names[0], "vrooli-agent-codex-eb0c.scope") {
		t.Fatalf("orphans must be listed worst first: %v", names)
	}
}

// A host with no agent slice is not a failing host.
func TestAbsentSliceIsNotAFailure(t *testing.T) {
	result := occupancyCheck(fakeOccupancy{}).Run(context.Background())
	if result.Status != checks.StatusOK {
		t.Fatalf("a host with no agent slice must not fail: %v %s", result.Status, result.Message)
	}
}

func TestOccupancyRunsEveryMinute(t *testing.T) {
	if got := NewContainmentOccupancyCheck().IntervalSeconds(); got != 60 {
		t.Fatalf("interval = %d seconds, want 60: the ceiling filled in minutes", got)
	}
}
