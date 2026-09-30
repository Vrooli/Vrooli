package agentscope

import (
	"errors"
	"testing"

	platformgo "github.com/vrooli/platform-go"
)

// fakeKernel is a slice and the scopes it holds, answered without a cgroup.
type fakeKernel struct {
	children  []platformgo.ScopeRef
	tasks     map[string]int64
	members   map[string][]int
	names     map[int]string
	procsErr  map[string]error
	childErr  error
	occErrFor map[string]bool
}

func (f *fakeKernel) reader() Reader {
	return Reader{
		Children: func(platformgo.ScopeRef) ([]platformgo.ScopeRef, error) {
			return f.children, f.childErr
		},
		Occupancy: func(ref platformgo.ScopeRef) (platformgo.Occupancy, error) {
			if f.occErrFor[ref.Name] {
				return platformgo.Occupancy{}, errors.New("no such cgroup")
			}
			return platformgo.Occupancy{Tasks: f.tasks[ref.Name], TasksMax: 16384}, nil
		},
		Processes: func(ref platformgo.ScopeRef) ([]int, error) {
			if err := f.procsErr[ref.Name]; err != nil {
				return nil, err
			}
			return f.members[ref.Name], nil
		},
		ProcessName: func(pid int) (string, error) {
			name, ok := f.names[pid]
			if !ok {
				return "", errors.New("no such process")
			}
			return name, nil
		},
	}
}

func scope(name string) platformgo.ScopeRef {
	return platformgo.ScopeRef{Name: name, Kind: platformgo.ScopeKindCgroup, Path: "/user.slice/vrooli-agents.slice/" + name + ScopeSuffix}
}

// The 2026-09-04 shape: a scope named after codex whose codex exited, still
// holding the scenario servers that session started.
func TestScopeNamedForAnAgentWithNoAgentIsAgentless(t *testing.T) {
	kernel := &fakeKernel{
		children: []platformgo.ScopeRef{scope("vrooli-agent-codex-eb0c")},
		tasks:    map[string]int64{"vrooli-agent-codex-eb0c": 787},
		members:  map[string][]int{"vrooli-agent-codex-eb0c": {11, 12}},
		names:    map[int]string{11: "test-genie-api", 12: "node"},
	}
	entries, err := kernel.reader().Census(Ref("/user.slice/vrooli-agents.slice"))
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	if len(entries) != 1 || !entries[0].Agentless {
		t.Fatalf("an agent-named scope with no agent in it is agentless: %+v", entries)
	}
	if AgentlessTasks(entries) != 787 {
		t.Fatalf("agentless tasks: got %d, want 787", AgentlessTasks(entries))
	}
}

func TestScopeWithACodingAgentIsLive(t *testing.T) {
	kernel := &fakeKernel{
		children: []platformgo.ScopeRef{scope("vrooli-agent-codex-live")},
		tasks:    map[string]int64{"vrooli-agent-codex-live": 38},
		members:  map[string][]int{"vrooli-agent-codex-live": {21, 22}},
		names:    map[int]string{21: "node", 22: "codex"},
	}
	entries, err := kernel.reader().Census(Ref("/user.slice/vrooli-agents.slice"))
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	if entries[0].Agentless {
		t.Fatal("a scope still running codex is a live session")
	}
	if AgentlessTasks(entries) != 0 {
		t.Fatalf("a live session holds no agentless tasks: %d", AgentlessTasks(entries))
	}
}

// An unreadable scope must not be invented as an orphan: an invented orphan
// is a freeze candidate and a reaper target.
func TestUnreadableScopeIsNotReportedAgentless(t *testing.T) {
	kernel := &fakeKernel{
		children: []platformgo.ScopeRef{scope("vrooli-agent-codex-sealed")},
		tasks:    map[string]int64{"vrooli-agent-codex-sealed": 9},
		procsErr: map[string]error{"vrooli-agent-codex-sealed": errors.New("permission denied")},
	}
	entries, err := kernel.reader().Census(Ref("/user.slice/vrooli-agents.slice"))
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	if len(entries) != 1 || entries[0].Agentless {
		t.Fatalf("an unreadable scope must not be reported agentless: %+v", entries)
	}
}

// The census considers session scopes only; the slice also holds tmux-spawn
// scopes and anything else the user manager put there.
func TestCensusIgnoresScopesTheLauncherDidNotMint(t *testing.T) {
	kernel := &fakeKernel{
		children: []platformgo.ScopeRef{scope("tmux-spawn-abc"), scope("vrooli-agent-grok-1")},
		tasks:    map[string]int64{"tmux-spawn-abc": 1, "vrooli-agent-grok-1": 3},
		members:  map[string][]int{"vrooli-agent-grok-1": {31}},
		names:    map[int]string{31: "grok"},
	}
	entries, err := kernel.reader().Census(Ref("/user.slice/vrooli-agents.slice"))
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	if len(entries) != 1 || entries[0].Name != "vrooli-agent-grok-1"+ScopeSuffix {
		t.Fatalf("census must consider only session scopes: %+v", entries)
	}
}

// A member whose name cannot be read is not evidence either way; another
// member still answers the question.
func TestAnExitedMemberDoesNotHideALiveAgent(t *testing.T) {
	kernel := &fakeKernel{
		children: []platformgo.ScopeRef{scope("vrooli-agent-claude-code-1")},
		tasks:    map[string]int64{"vrooli-agent-claude-code-1": 8},
		members:  map[string][]int{"vrooli-agent-claude-code-1": {41, 42}},
		names:    map[int]string{42: "claude"},
	}
	live, err := kernel.reader().HoldsLiveAgent(scope("vrooli-agent-claude-code-1"))
	if err != nil {
		t.Fatalf("holds live agent: %v", err)
	}
	if !live {
		t.Fatal("a pid that exited between the listing and the read must not mask a live agent")
	}
}

// Occupancy failing leaves the task count unknown rather than zero: zero is
// the reading that would make a full scope look empty.
func TestUnreadableOccupancyLeavesTasksUnknown(t *testing.T) {
	kernel := &fakeKernel{
		children:  []platformgo.ScopeRef{scope("vrooli-agent-codex-gone")},
		occErrFor: map[string]bool{"vrooli-agent-codex-gone": true},
		members:   map[string][]int{"vrooli-agent-codex-gone": {51}},
		names:     map[int]string{51: "codex"},
	}
	entries, err := kernel.reader().Census(Ref("/user.slice/vrooli-agents.slice"))
	if err != nil {
		t.Fatalf("census: %v", err)
	}
	if entries[0].Tasks != platformgo.Unknown {
		t.Fatalf("tasks: got %d, want Unknown", entries[0].Tasks)
	}
	if AgentlessTasks(entries) != 0 {
		t.Fatalf("an unknown task count contributes nothing to the total: %d", AgentlessTasks(entries))
	}
}
