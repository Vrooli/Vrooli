// Package agentscope answers what an agent session scope is holding.
//
// The agent_session_containment safeguard gives every coding-agent session a
// transient scope under vrooli-agents.slice. The scope is named after the
// agent that minted it, and it lives as long as any process remains in it —
// so a session that ran `vrooli scenario start` and then exited leaves an
// agent-named scope full of scenario servers. Two components have to tell
// those apart: the storm authority, which must never freeze one, and the
// containment safeguard, which reports how much of the ceiling they hold.
//
// The scope name is never the answer. Membership is the scope's own, read
// from the kernel; a caller must not find a session's processes by matching
// a command line, which any argument can impersonate.
//
// The agent vocabulary is duplicated once, in cli-core's launcher, on
// purpose: cli-core cannot import platform-go (244 modules replace it), so
// the launcher owns the names and this package owns the questions asked of
// the kernel with them.
package agentscope

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/vrooli/cli-core/agentcatalog/harness"
	platformgo "github.com/vrooli/platform-go"
)

const (
	// Slice is the user-manager slice every session scope lives under.
	Slice = "vrooli-agents.slice"
	// Prefix and ScopeSuffix bracket a session scope's unit name.
	Prefix      = "vrooli-agent-"
	ScopeSuffix = ".scope"
)

// Binaries are the coding-agent programs a live session runs, by the
// kernel's own name for the program. The launcher's codingAgentSpec mints
// scopes for exactly these.
var Binaries = declaredBinaries()

func declaredBinaries() map[string]bool {
	binaries := map[string]bool{}
	for _, descriptor := range harness.Builtin().All() {
		binaries[strings.ToLower(descriptor.Binary)] = true
	}
	return binaries
}

// Entry is one session scope and what it holds.
type Entry struct {
	// Name is the scope's unit name.
	Name string `json:"name"`
	// Tasks is the scope's current task count, or platformgo.Unknown.
	Tasks int64 `json:"tasks"`
	// Agentless is a scope whose coding agent has exited. Its tasks are
	// charged to the slice's ceiling for as long as its children run, and
	// nothing in the session lifecycle releases them.
	Agentless bool `json:"agentless"`
	// MembershipError leaves the scope unclassified. False Agentless alone
	// must not turn an unreadable member into a reported live agent.
	MembershipError string `json:"membership_error,omitempty"`
}

// Reader reads the kernel through injectable seams so the census is tested
// without a cgroup tree. NewReader binds the real ones.
type Reader struct {
	Children    func(platformgo.ScopeRef) ([]platformgo.ScopeRef, error)
	Occupancy   func(platformgo.ScopeRef) (platformgo.Occupancy, error)
	Processes   func(platformgo.ScopeRef) ([]int, error)
	ProcessName func(int) (string, error)
}

// NewReader binds the reader to this host.
func NewReader() Reader {
	return Reader{
		Children:    platformgo.ScopeChildren,
		Occupancy:   platformgo.ScopeOccupancy,
		Processes:   platformgo.ScopeProcesses,
		ProcessName: platformgo.ProcessName,
	}
}

// Ref points a reader at a cgroup path.
func Ref(cgroupPath string) platformgo.ScopeRef {
	return platformgo.ScopeRef{Kind: platformgo.ScopeKindCgroup, Path: cgroupPath}
}

// IsSessionScope reports whether a scope unit name is one the launcher mints.
func IsSessionScope(name string) bool {
	return strings.HasPrefix(name, Prefix) && strings.HasSuffix(name, ScopeSuffix)
}

// HoldsLiveAgent reports whether a coding-agent program is among the scope's
// members. An error means the question could not be asked: a caller that
// acts on the answer must refuse on error, and a caller that only reports
// may say the scope is undetermined.
func (r Reader) HoldsLiveAgent(ref platformgo.ScopeRef) (bool, error) {
	pids, err := r.Processes(ref)
	if err != nil {
		return false, err
	}
	var unread []error
	for _, pid := range pids {
		name, readErr := r.ProcessName(pid)
		if readErr != nil {
			// Only native absence means the member exited. Permissions,
			// unsupported readers and malformed proc data leave it unknown.
			if !errors.Is(readErr, os.ErrNotExist) {
				unread = append(unread, fmt.Errorf("scope member %d: %w", pid, readErr))
			}
			continue
		}
		if strings.TrimSpace(name) == "" {
			unread = append(unread, fmt.Errorf("scope member %d has an empty program name", pid))
			continue
		}
		if Binaries[strings.ToLower(strings.TrimSpace(name))] {
			return true, nil
		}
	}
	return false, errors.Join(unread...)
}

// Census reads every session scope a slice holds and asks each one the
// membership question. A scope that cannot be read is reported as holding an
// agent: the census must never invent an orphan it cannot see, because an
// invented orphan is a freeze candidate and a reaper target.
func (r Reader) Census(slice platformgo.ScopeRef) ([]Entry, error) {
	children, err := r.Children(slice)
	if err != nil {
		return nil, err
	}
	out := make([]Entry, 0, len(children))
	for _, child := range children {
		name := scopeUnitName(child)
		if !IsSessionScope(name) {
			continue
		}
		entry := Entry{Name: name, Tasks: platformgo.Unknown}
		if occupancy, occErr := r.Occupancy(child); occErr == nil {
			entry.Tasks = occupancy.Tasks
		}
		if live, liveErr := r.HoldsLiveAgent(child); liveErr == nil {
			entry.Agentless = !live
		} else {
			entry.MembershipError = liveErr.Error()
		}
		out = append(out, entry)
	}
	return out, nil
}

// Uncertain returns scopes whose member programs could not be classified.
func Uncertain(entries []Entry) []Entry {
	var out []Entry
	for _, entry := range entries {
		if entry.MembershipError != "" {
			out = append(out, entry)
		}
	}
	return out
}

// scopeUnitName recovers the unit name from a ref; ScopeChildren trims the
// suffix into Name, and the census matches on the full unit name.
func scopeUnitName(ref platformgo.ScopeRef) string {
	if strings.HasSuffix(ref.Name, ScopeSuffix) {
		return ref.Name
	}
	if ref.Name != "" {
		return ref.Name + ScopeSuffix
	}
	return ""
}

// AgentlessTasks is how much of a ceiling is held by sessions that ended.
func AgentlessTasks(entries []Entry) int64 {
	var total int64
	for _, entry := range entries {
		if entry.Agentless && entry.Tasks > 0 {
			total += entry.Tasks
		}
	}
	return total
}

// Agentless lists the scopes whose coding agent has exited.
func Agentless(entries []Entry) []Entry {
	out := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.Agentless {
			out = append(out, entry)
		}
	}
	return out
}
