// This file reads a run and its descendants as one tree for nested-run
// supervision.
package orchestration

import (
	"context"
	"slices"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/stepback"

	"github.com/google/uuid"
)

// maxRunTreeNodes bounds one tree read. Nesting is capped by
// runExecution.maxChildDepth, so only very wide fan-out reaches it.
const maxRunTreeNodes = 500

// RunTreeNode is one run in a run tree with its usage and orphan marker.
type RunTreeNode struct {
	Run *domain.Run
	// Weighted non-cache tokens, as run report weights them. Known is false
	// when usage is not final or the model has no known tier weight.
	WeightedTokens      float64
	WeightedTokensKnown bool
	// ElapsedSeconds runs from start (or creation) to end, or to the read.
	ElapsedSeconds int64
	// Orphaned marks a live run whose parent is terminal. Agent Manager never
	// stops it: a scheduler crash must not take its segment owners down.
	Orphaned bool
	// StepBack lists the transcript step-back triggers observed on this run;
	// short-workers counts its descendants in this tree.
	StepBack []stepback.Trigger
	Children []*RunTreeNode
}

// RunTree is one bounded read of a run and its descendants.
type RunTree struct {
	Root          *RunTreeNode
	NodeCount     int
	Truncated     bool
	MaxChildDepth int
}

// GetRunTree reads id and its descendants breadth first, oldest child first,
// up to maxRunTreeNodes. Usage comes from the same batch accounting the effort
// board reads; when it is unavailable the tree is still returned with every
// weight marked unknown. Step-back triggers come from each run's stored
// transcript facts; when those are unavailable the tree has none.
func (o *Orchestrator) GetRunTree(ctx context.Context, id uuid.UUID) (*RunTree, error) {
	reader := repositoryChildReader{o: o}
	tree, nodes, ids, err := o.collectRunTree(ctx, id)
	if err != nil {
		return nil, err
	}

	// Listing rows omit the resolved configuration (runner); the batch read
	// returns full snapshots together with their accounting.
	full, accounts, err := o.RunAccountingBatch(ctx, ids)
	if err != nil {
		obs.Component("run-tree").Warn("run tree accounting unavailable; weights reported unknown", obs.KeyRunID, id.String(), obs.KeyError, err.Error())
	}
	for runID, node := range nodes {
		if run := full[runID]; run != nil {
			node.Run = run
		}
		if account, ok := accounts[runID]; ok && account.WeightedPresent {
			node.WeightedTokens = account.WeightedTokens
			node.WeightedTokensKnown = account.WeightedKnown
		}
	}
	facts := o.transcriptFacts(ctx, ids)
	for runID, node := range nodes {
		node.StepBack = stepback.Assess(stepback.Input{Transcript: facts[runID], Descendants: descendantsOf(node)})
	}
	parentStatus := func(node *RunTreeNode) (domain.RunStatus, bool) {
		if node.Run.ParentRunID == nil {
			return "", false
		}
		if parent := nodes[*node.Run.ParentRunID]; parent != nil {
			return parent.Run.Status, true
		}
		parent, err := reader.GetRun(ctx, *node.Run.ParentRunID)
		if err != nil {
			return "", false
		}
		return parent.Status, true
	}
	for _, node := range nodes {
		if !node.Run.Status.IsLive() {
			continue
		}
		if status, ok := parentStatus(node); ok && status.IsTerminal() {
			node.Orphaned = true
		}
	}
	return tree, nil
}

// collectRunTree reads id and its descendants breadth first, oldest child
// first, up to maxRunTreeNodes, with each node's elapsed time.
func (o *Orchestrator) collectRunTree(ctx context.Context, id uuid.UUID) (*RunTree, map[uuid.UUID]*RunTreeNode, []uuid.UUID, error) {
	reader := repositoryChildReader{o: o}
	rootRun, err := reader.GetRun(ctx, id)
	if err != nil {
		return nil, nil, nil, err
	}
	tree := &RunTree{Root: &RunTreeNode{Run: rootRun}, NodeCount: 1, MaxChildDepth: o.maxChildDepth()}
	nodes := map[uuid.UUID]*RunTreeNode{rootRun.ID: tree.Root}
	ids := []uuid.UUID{rootRun.ID}
	for queue := []*RunTreeNode{tree.Root}; len(queue) > 0 && !tree.Truncated; queue = queue[1:] {
		children, err := reader.ListRuns(ctx, RunListOptions{ListOptions: ListOptions{Limit: childReadLimit}, ParentRunID: &queue[0].Run.ID})
		if err != nil {
			return nil, nil, nil, err
		}
		// Listings are newest first; a tree reads in creation order.
		slices.Reverse(children)
		for _, child := range children {
			if child == nil || nodes[child.ID] != nil {
				continue
			}
			if tree.NodeCount >= maxRunTreeNodes {
				tree.Truncated = true
				break
			}
			node := &RunTreeNode{Run: child}
			queue[0].Children = append(queue[0].Children, node)
			nodes[child.ID] = node
			ids = append(ids, child.ID)
			queue = append(queue, node)
			tree.NodeCount++
		}
	}
	now := o.now()
	for _, node := range nodes {
		node.ElapsedSeconds = runElapsedSeconds(node.Run, now)
	}
	return tree, nodes, ids, nil
}

// ReadRunStepBack assesses the transcript step-back triggers of each run,
// counting its descendants (up to maxRunTreeNodes) for short-workers. A run
// that cannot be read is absent from the result.
func (o *Orchestrator) ReadRunStepBack(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]stepback.Trigger, error) {
	roots := make(map[uuid.UUID]*RunTreeNode, len(ids))
	for _, id := range ids {
		tree, _, _, err := o.collectRunTree(ctx, id)
		if err != nil {
			continue
		}
		roots[id] = tree.Root
	}
	facts := o.transcriptFacts(ctx, ids)
	result := make(map[uuid.UUID][]stepback.Trigger, len(roots))
	for id, root := range roots {
		result[id] = stepback.Assess(stepback.Input{Transcript: facts[id], Descendants: descendantsOf(root)})
	}
	return result, nil
}

// transcriptFacts reads the runs' transcript facts, or none when the event
// store cannot provide them.
func (o *Orchestrator) transcriptFacts(ctx context.Context, ids []uuid.UUID) map[uuid.UUID]stepback.Transcript {
	reader, ok := o.events.(event.TranscriptFactsReader)
	if !ok {
		return nil
	}
	facts, err := reader.TranscriptFacts(ctx, ids)
	if err != nil {
		obs.Component("run-tree").Warn("transcript facts unavailable; step-back triggers omitted", obs.KeyError, err.Error())
		return nil
	}
	return facts
}

// descendantsOf lists every run below node with its liveness and elapsed time.
func descendantsOf(node *RunTreeNode) []stepback.Descendant {
	var out []stepback.Descendant
	for stack := append([]*RunTreeNode(nil), node.Children...); len(stack) > 0; {
		next := stack[len(stack)-1]
		stack = append(stack[:len(stack)-1], next.Children...)
		out = append(out, stepback.Descendant{Live: next.Run.Status.IsLive(), Duration: time.Duration(next.ElapsedSeconds) * time.Second})
	}
	return out
}

func runElapsedSeconds(run *domain.Run, now time.Time) int64 {
	start := run.CreatedAt
	if run.StartedAt != nil {
		start = *run.StartedAt
	}
	// A parked or continued run keeps its previous turn's end; it is live.
	end := now
	if run.EndedAt != nil && !run.Status.IsLive() {
		end = *run.EndedAt
	}
	if start.IsZero() || end.Before(start) {
		return 0
	}
	return int64(end.Sub(start).Seconds())
}
