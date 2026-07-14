package fleet

import (
	"context"
	"sort"
	"strings"

	"vrooli-bridge/internal/clock"
)

// Service is the application-layer surface the fleet handler depends on. It owns
// the RollFleet fan-out: enumerate the eligible nodes, classify each, dispatch a
// privileged provisioning op per eligible node (delegating to the Provisioner
// seam), and record a durable Rollout with the per-node ledger.
type Service interface {
	// Roll pins the fleet (or the named subset) to a target revision. It
	// classifies every node, dispatches a provisioning op to each eligible one,
	// and records a durable rollout. On a dry-run it classifies and short-
	// circuits before creating any rollout/op.
	Roll(ctx context.Context, in RollInput) (RollDecision, error)

	// GetRollout returns a rollout and its per-node ledger.
	GetRollout(ctx context.Context, id string) (Rollout, []NodeResult, error)

	// ListRollouts returns rollouts newest-first, narrowed by filter.
	ListRollouts(ctx context.Context, filter ListFilter) ([]Rollout, error)
}

type service struct {
	repo        Repository
	nodes       NodeLister
	presence    Presence
	provisioner Provisioner
	clock       clock.Clock
	revResolver RevisionResolver
}

// Option customises the service.
type Option func(*service)

// WithRevisionResolver wires the control-plane revision resolver. When set, the
// roll target is defaulted (empty/"@cp" → the control plane's commit),
// metacharacter-validated, and preflighted ONCE before dispatch, and the resolved
// commit is what every node is pinned to (and what the rollout records); when
// unset, the target is required non-empty and used verbatim (legacy behaviour).
func WithRevisionResolver(r RevisionResolver) Option {
	return func(s *service) { s.revResolver = r }
}

// NewService constructs the production Service.
func NewService(repo Repository, nodes NodeLister, presence Presence, provisioner Provisioner, clk clock.Clock, opts ...Option) Service {
	s := &service{repo: repo, nodes: nodes, presence: presence, provisioner: provisioner, clock: clk}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Compile-time guarantee.
var _ Service = (*service)(nil)

func (s *service) Roll(ctx context.Context, in RollInput) (RollDecision, error) {
	target := trimRevision(in.TargetRevision)
	if s.revResolver != nil {
		// Resolve ONCE so the whole roll pins to a single exact commit (empty/"@cp"
		// → the control plane's commit), and preflight it here so an unpushed or
		// invalid target fails the roll before any node is dispatched.
		resolved, err := s.revResolver.Resolve(ctx, target)
		if err != nil {
			return RollDecision{}, err
		}
		target = resolved
	} else if target == "" {
		return RollDecision{}, ErrInvalidRoll{Field: "target_revision", Reason: "required"}
	}

	// Enumerate the candidate nodes: the explicit subset (resolved against the
	// registry) or every registered node.
	candidates, err := s.resolveCandidates(ctx, in.NodeIDs)
	if err != nil {
		return RollDecision{}, err
	}

	results := make([]NodeResult, 0, len(candidates))
	for _, node := range candidates {
		results = append(results, s.classifyAndDispatch(ctx, node, in, target))
	}

	status := aggregateStatus(results)

	// Dry-run: report what WOULD happen without persisting a rollout or pushing.
	if in.DryRun {
		return RollDecision{DryRun: true, Status: status, Results: results}, nil
	}

	rollout := Rollout{
		TargetRevision: target,
		Status:         status,
		TotalNodes:     len(results),
		Dispatched:     countDisposition(results, DispositionDispatched),
		Skipped:        countSkipped(results),
		Failed:         countDisposition(results, DispositionFailed),
	}
	persisted, err := s.repo.Create(ctx, rollout, results)
	if err != nil {
		return RollDecision{}, err
	}

	return RollDecision{RolloutID: persisted.ID, Status: status, Results: results}, nil
}

// classifyAndDispatch decides one node's disposition and, when eligible (and not
// a dry-run), dispatches its provisioning op. Classification order: revoked →
// offline → needs-update → dispatch.
func (s *service) classifyAndDispatch(ctx context.Context, node NodeRef, in RollInput, target string) NodeResult {
	switch {
	case node.Revoked:
		return NodeResult{NodeID: node.ID, Disposition: DispositionSkippedRevoked, Detail: "node revoked"}
	case !s.presence.IsOnline(node.ID):
		return NodeResult{NodeID: node.ID, Disposition: DispositionSkippedOffline, Detail: "node offline"}
	case !s.presence.Dispatchable(node.ID):
		return NodeResult{NodeID: node.ID, Disposition: DispositionSkippedNeedsUpdate, Detail: "agent protocol needs update"}
	}

	// On a dry-run the node is eligible but we dispatch nothing.
	if in.DryRun {
		return NodeResult{NodeID: node.ID, Disposition: DispositionDispatched, Detail: "would dispatch"}
	}

	opID, err := s.provisioner.Provision(ctx, ProvisionRequest{
		Actor:          in.Actor,
		NodeID:         node.ID,
		TargetRevision: target,
		TimeoutSeconds: in.TimeoutSeconds,
	})
	if err != nil {
		return NodeResult{NodeID: node.ID, Disposition: DispositionFailed, Detail: err.Error()}
	}
	return NodeResult{NodeID: node.ID, Disposition: DispositionDispatched, OpID: opID, Detail: "dispatched"}
}

// resolveCandidates returns the nodes a roll targets: the explicit subset
// (filtered to registered nodes, deduped, stable-sorted) or every registered
// node. An unknown id in the subset is dropped silently (it is not in the
// fleet); a roll over an empty/unknown subset yields no candidates.
func (s *service) resolveCandidates(ctx context.Context, subset []string) ([]NodeRef, error) {
	all, err := s.nodes.ListNodes(ctx)
	if err != nil {
		return nil, err
	}
	if len(subset) == 0 {
		sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
		return all, nil
	}

	byID := make(map[string]NodeRef, len(all))
	for _, n := range all {
		byID[n.ID] = n
	}
	want := make(map[string]struct{}, len(subset))
	for _, id := range subset {
		if t := strings.TrimSpace(id); t != "" {
			want[t] = struct{}{}
		}
	}
	out := make([]NodeRef, 0, len(want))
	for id := range want {
		if n, ok := byID[id]; ok {
			out = append(out, n)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *service) GetRollout(ctx context.Context, id string) (Rollout, []NodeResult, error) {
	rollout, err := s.repo.Get(ctx, id)
	if err != nil {
		return Rollout{}, nil, err
	}
	results, err := s.repo.Results(ctx, id)
	if err != nil {
		return Rollout{}, nil, err
	}
	return rollout, results, nil
}

func (s *service) ListRollouts(ctx context.Context, filter ListFilter) ([]Rollout, error) {
	return s.repo.List(ctx, filter)
}

func countDisposition(results []NodeResult, d NodeDisposition) int {
	n := 0
	for _, r := range results {
		if r.Disposition == d {
			n++
		}
	}
	return n
}

// countSkipped counts the three skip dispositions (offline / needs-update /
// revoked).
func countSkipped(results []NodeResult) int {
	n := 0
	for _, r := range results {
		switch r.Disposition {
		case DispositionSkippedOffline, DispositionSkippedNeedsUpdate, DispositionSkippedRevoked:
			n++
		}
	}
	return n
}
