package hostpressure

import (
	"runtime"
	"sort"
	"time"
)

// attributionTop bounds how many parents each ranking names.
const attributionTop = 5

// attributionOS is the only platform whose snapshot carries a process tree.
const attributionOS = "linux"

// Parent is one process ranked by the children it owns.
type Parent struct {
	PID      int64  `json:"pid"`
	Name     string `json:"name"`
	Children int    `json:"children"`
	// Delta is the change in child count against the previous snapshot;
	// zero when no previous snapshot was available.
	Delta int `json:"delta"`
	// Scope is the parent's own cgroup path (unified hierarchy) when it can
	// be read; an agent session scope contains "vrooli-agents.slice".
	Scope string `json:"scope,omitempty"`
}

// AttributionReading names the parents behind a host-wide fork rate. It is
// unread, with a reason, wherever the process list is not collected, so an
// absent culprit never reads as "no culprit".
type AttributionReading struct {
	State      State         `json:"state"`
	Provenance string        `json:"provenance,omitempty"`
	Reason     string        `json:"reason,omitempty"`
	ByChildren []Parent      `json:"by_children,omitempty"`
	ByDelta    []Parent      `json:"by_delta,omitempty"`
	Elapsed    time.Duration `json:"elapsed_ns,omitempty"`
}

// TopParent is the leading parent by live child count.
func (a AttributionReading) TopParent() (Parent, bool) {
	if a.State != Read || len(a.ByChildren) == 0 {
		return Parent{}, false
	}
	return a.ByChildren[0], true
}

// Attribution folds the snapshot's process list into the parents with the
// most live children and, when a previous snapshot is given, the parents
// whose child count grew the most. It walks nothing itself: the list the
// snapshot already collected is the only input.
func Attribution(snapshot PressureSnapshot, previous *PressureSnapshot) AttributionReading {
	return attributionFor(runtime.GOOS, snapshot, previous)
}

func attributionFor(goos string, snapshot PressureSnapshot, previous *PressureSnapshot) AttributionReading {
	if goos != attributionOS {
		return AttributionReading{State: Unread, Provenance: "attribution", Reason: "parent attribution needs the process list, which is unread on " + goos}
	}
	if len(snapshot.Processes) == 0 {
		return AttributionReading{State: Unread, Provenance: "attribution", Reason: "the snapshot carries no process list"}
	}
	start := time.Now()
	children := countChildren(snapshot.Processes)
	names := make(map[int64]string, len(snapshot.Processes))
	for _, p := range snapshot.Processes {
		names[p.PID] = p.Name
	}
	var before map[int64]int
	if previous != nil && len(previous.Processes) > 0 {
		before = countChildren(previous.Processes)
	}
	parents := make([]Parent, 0, len(children))
	for pid, count := range children {
		parent := Parent{PID: pid, Name: names[pid], Children: count}
		if before != nil {
			parent.Delta = count - before[pid]
		}
		parents = append(parents, parent)
	}
	byChildren := rank(parents, func(a, b Parent) bool {
		if a.Children != b.Children {
			return a.Children > b.Children
		}
		return a.PID < b.PID
	})
	out := AttributionReading{State: Read, Provenance: "snapshot process list (PPid)", ByChildren: byChildren}
	if before != nil {
		out.ByDelta = rank(parents, func(a, b Parent) bool {
			if a.Delta != b.Delta {
				return a.Delta > b.Delta
			}
			return a.PID < b.PID
		})
	}
	for _, list := range [][]Parent{out.ByChildren, out.ByDelta} {
		for i := range list {
			list[i].Scope = parentScope(list[i].PID, list[i].Name)
		}
	}
	out.Elapsed = time.Since(start)
	return out
}

// kthreaddPID owns every kernel thread; its "children" are never forks a
// user process made, so it is not a candidate parent.
const kthreaddPID = 2

func countChildren(list []Process) map[int64]int {
	counts := make(map[int64]int, len(list)/4+1)
	for _, p := range list {
		if p.PPID > 0 && p.PPID != kthreaddPID {
			counts[p.PPID]++
		}
	}
	return counts
}

func rank(parents []Parent, less func(a, b Parent) bool) []Parent {
	sorted := append([]Parent(nil), parents...)
	sort.Slice(sorted, func(i, j int) bool { return less(sorted[i], sorted[j]) })
	if len(sorted) > attributionTop {
		sorted = sorted[:attributionTop]
	}
	return sorted
}
