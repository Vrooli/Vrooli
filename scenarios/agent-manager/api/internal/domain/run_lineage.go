// Responsibility: render the short lineage and charter header a nested or
// chartered run's task prompt starts with.
package domain

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// RunLineage is what a run needs to know about its place in a run tree. It is
// fixed at creation from the admitted parent and the current depth setting.
type RunLineage struct {
	RunID       uuid.UUID
	TaskID      uuid.UUID
	ParentRunID *uuid.UUID
	Depth       int
	MaxDepth    int
	CharterPath string
}

// Header returns the lines Agent Manager prepends to the task prompt of a child
// run or a run with a charter, and "" for an unchartered root run. It names the
// run's own identity (a child needs it to start children of its own), the
// parent and depth, and where the brief lives after a context compaction. It
// never repeats the goal text: a native goal is re-injected by the harness, and
// the charter or task holds the rest.
func (l RunLineage) Header() string {
	charter := strings.TrimSpace(l.CharterPath)
	if l.ParentRunID == nil && charter == "" {
		return ""
	}
	lines := make([]string, 0, 3)
	if l.ParentRunID != nil {
		lines = append(lines, fmt.Sprintf("Agent Manager lineage: you are run %s, depth %d of max %d, a child of run %s.", l.RunID, l.Depth, l.MaxDepth, *l.ParentRunID))
	} else {
		lines = append(lines, fmt.Sprintf("Agent Manager lineage: you are run %s.", l.RunID))
	}
	if charter != "" {
		lines = append(lines, fmt.Sprintf("Charter: %s. Read it before substantive work, and re-read it at every unit start and after any context compaction.", charter))
	} else {
		lines = append(lines, fmt.Sprintf("Brief: no charter was named; after any context compaction, re-read your task with `agent-manager task get %s`.", l.TaskID))
	}
	switch {
	case l.MaxDepth <= 0:
		lines = append(lines, "Delegation: Agent Manager child runs are disabled; use your harness's native sub-agents.")
	case l.Depth >= l.MaxDepth:
		lines = append(lines, "Delegation: you are at the maximum depth; use your harness's native sub-agents, not Agent Manager child runs.")
	default:
		lines = append(lines, fmt.Sprintf("Delegation: you may start child runs with `agent-manager run create --parent-run-id %s`; if Agent Manager is unreachable or refuses, use your harness's native sub-agents with the same brief.", l.RunID))
	}
	return strings.Join(lines, "\n")
}

// WithLineageHeader prepends a non-empty lineage header to task instructions.
func WithLineageHeader(header, instructions string) string {
	if header == "" {
		return instructions
	}
	if strings.TrimSpace(instructions) == "" {
		return header
	}
	return header + "\n\n" + instructions
}
