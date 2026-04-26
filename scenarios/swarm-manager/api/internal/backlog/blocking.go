// Shared dependency-blocking evaluation for backlog items.
// This is the single source of truth for determining whether an item's
// dependencies block a workflow and whether the block is overridable.
package backlog

import (
	"fmt"
	"strings"

	"swarm-manager/internal/depgraph"
)

// backlogNode adapts BacklogItem to the depgraph.Node interface.
type backlogNode struct{ item BacklogItem }

func (n backlogNode) Key() string    { return string(n.item.Kind) + "/" + n.item.Name }
func (n backlogNode) Deps() []string { return n.item.DependsOn }
func (n backlogNode) Status() string { return string(n.item.Status) }

// BlockingReason represents a single blocking reason with forceability.
type BlockingReason struct {
	Message   string
	Forceable bool
}

// blockingDepStatuses are statuses that indicate a dependency is not yet
// planned/started — meaning the downstream item should not proceed yet.
// Once a dependency has progressed past the planning phase (ready, queued,
// in_progress, completed, failed, archived) it no longer blocks.
var blockingDepStatuses = map[BacklogStatus]bool{
	StatusBacklog:     true,
	StatusResearching: true,
}

// EvaluateDependencyBlocking checks an item's dependencies and returns
// structured blocking reasons. Dependencies in "backlog" or "researching"
// status produce forceable blocking reasons. Missing/deleted deps are
// presumed completed (fail-open).
func EvaluateDependencyBlocking(item BacklogItem, store Store) ([]BlockingReason, error) {
	if len(item.DependsOn) == 0 {
		return nil, nil
	}
	unmet, err := store.CheckDependencies(item.DependsOn)
	if err != nil {
		return nil, err
	}
	if len(unmet) == 0 {
		return nil, nil
	}
	return []BlockingReason{{
		Message:   fmt.Sprintf("unmet dependencies: %s", strings.Join(unmet, ", ")),
		Forceable: true,
	}}, nil
}

// HasNonForceableReasons returns true if any reason cannot be overridden
// with the force flag.
func HasNonForceableReasons(reasons []BlockingReason) bool {
	for _, r := range reasons {
		if !r.Forceable {
			return true
		}
	}
	return false
}

// AllForceable returns true if all reasons are forceable (or the slice is empty).
func AllForceable(reasons []BlockingReason) bool {
	return !HasNonForceableReasons(reasons)
}

// DedupeReasons removes duplicate and empty blocking reasons.
func DedupeReasons(reasons []BlockingReason) []BlockingReason {
	if len(reasons) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(reasons))
	result := make([]BlockingReason, 0, len(reasons))
	for _, r := range reasons {
		trimmed := strings.TrimSpace(r.Message)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, BlockingReason{Message: trimmed, Forceable: r.Forceable})
	}
	return result
}

// ListBlockingInfo is the per-item blocking summary for list views.
type ListBlockingInfo struct {
	Blocked         bool
	BlockingDepKeys []string
	AllForceable    bool
}

// ComputeListBlockingInfo evaluates dependency blocking for all items in a
// single pass. Returns a map keyed by "kind/name". Only items with
// dependencies that are actually blocked are included. Delegates to the
// generic depgraph package so backlog and initiatives share one implementation.
func ComputeListBlockingInfo(items []BacklogItem) map[string]ListBlockingInfo {
	nodes := make([]depgraph.Node, 0, len(items))
	for _, item := range items {
		if IsArchived(item) {
			continue
		}
		nodes = append(nodes, backlogNode{item: item})
	}
	blockingStatuses := make(map[string]bool, len(blockingDepStatuses))
	for s := range blockingDepStatuses {
		blockingStatuses[string(s)] = true
	}
	raw := depgraph.ComputeBlocking(nodes, blockingStatuses, true)

	result := make(map[string]ListBlockingInfo, len(raw))
	for k, info := range raw {
		result[k] = ListBlockingInfo{
			Blocked:         info.Blocked,
			BlockingDepKeys: info.BlockingKeys,
			AllForceable:    info.AllForceable,
		}
	}
	return result
}
