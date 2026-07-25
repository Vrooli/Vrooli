package proposals

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"strings"
	"time"

	"swarm-manager/internal/backlog"
)

// Priority bounds shared across Validate and Apply — proposals reject
// anything outside [MinItemPriority, MaxItemPriority] and Apply clamps
// unset priorities to DefaultItemPriority (mid-band). Keeping the three
// constants adjacent makes the intent obvious: new fields that reuse
// this scale should anchor on these constants rather than 1/5/10
// literals.
const (
	MinItemPriority     = 1
	MaxItemPriority     = 10
	DefaultItemPriority = 5
)

// DefaultApplierOwner is the fallback Source.DecidedBy / attribution
// owner recorded when a proposal is applied without an explicit owner.
const DefaultApplierOwner = "proposal"

// BacklogStore is the minimal backlog persistence surface the Applier uses.
// Satisfied by backlog.FileStore (via the exported Store interface).
type BacklogStore interface {
	LoadItem(kind backlog.BacklogKind, name string) (backlog.BacklogItem, error)
	SaveItem(item backlog.BacklogItem) error
	ItemDir(kind backlog.BacklogKind, name string) string
	SetItemMilestone(kind backlog.BacklogKind, name, milestone string) (string, error)
	ClearItemMilestone(kind backlog.BacklogKind, name, expected string) (string, bool, error)
	ValidateDependencies(dependsOn []string) error
}

// MilestoneAssigner is the milestone-side cascade surface the Applier uses.
// Satisfied by milestones.Service; mirrors backlog.MilestoneAssigner but is
// redefined here to keep proposals a leaf package.
type MilestoneAssigner interface {
	RememberItem(milestoneName, ref string) error
	ForgetItem(milestoneName, ref string) error
}

// ExecutionCanceller cancels the item's active execution. Satisfied by
// execution.Service via a thin adapter (see adapter_test.go in this package
// and wiring in main.go).
type ExecutionCanceller interface {
	CancelForBacklog(ctx context.Context, kind, name string) error
}

// GraphInvalidator notifies the graph projection that topology changed so
// graph.json (and any in-memory lenses) get rebuilt. Optional — a nil
// invalidator is a no-op.
type GraphInvalidator interface {
	ScheduleAll()
}

// EventEmitter records each applied mutation for audit. Optional — the
// Applier works fine with a nil emitter.
type EventEmitter interface {
	EmitProposalMutationApplied(source Source, mutation Mutation)
}

// ItemCreator is the chokepoint the Applier uses for OpAddItem. Satisfied
// by *backlog.Service. Required: there is no fallback inline create path,
// because the whole point of routing through Service is uniform side
// effects (backlog.created event with attribution, graph invalidation
// policy, optional workshop).
type ItemCreator interface {
	Create(item backlog.BacklogItem, cc backlog.CreationContext) error
}

// ItemLifecycle owns destructive item transitions so proposal approval and
// direct operator actions share guards and compensating rollback.
type ItemLifecycle interface {
	RecreateItem(ctx context.Context, kind backlog.BacklogKind, name string) (backlog.BacklogItem, error)
	ResetArtifacts(ctx context.Context, kind backlog.BacklogKind, name string, scopes []backlog.ResetArtifactScope) (backlog.ResetArtifactsResult, error)
}

// MilestoneLifecycle owns lineage-preserving milestone recreation.
type MilestoneLifecycle interface {
	RecreateMilestone(ctx context.Context, name string) error
}

// Applier executes accepted mutations against the underlying services.
//
// Correct use: normalize → validate → filter to accepted IDs → Apply. The
// Applier re-validates defensively, so a malformed mutation surfaces as an
// Outcome error rather than a partial mutation.
type Applier struct {
	store              BacklogStore
	assigner           MilestoneAssigner
	creator            ItemCreator
	itemLifecycle      ItemLifecycle
	milestoneLifecycle MilestoneLifecycle
	cancel             ExecutionCanceller
	invalidator        GraphInvalidator
	events             EventEmitter
	clock              func() time.Time
	defaultOwner       string
}

// Config bundles Applier dependencies.
type Config struct {
	Store              BacklogStore
	Assigner           MilestoneAssigner
	Creator            ItemCreator
	ItemLifecycle      ItemLifecycle
	MilestoneLifecycle MilestoneLifecycle
	Canceller          ExecutionCanceller
	Invalidator        GraphInvalidator
	Events             EventEmitter
	Clock              func() time.Time // optional; defaults to time.Now
	DefaultOwner       string           // falls back to "proposal" if empty
}

// NewApplier constructs an Applier. Store, Assigner, and Creator are
// required; the others may be nil for tests or degraded modes.
func NewApplier(cfg Config) (*Applier, error) {
	if cfg.Store == nil {
		return nil, errors.New("proposals: Store is required")
	}
	if cfg.Assigner == nil {
		return nil, errors.New("proposals: Assigner is required")
	}
	if cfg.Creator == nil {
		return nil, errors.New("proposals: Creator is required (route OpAddItem through backlog.Service)")
	}
	clk := cfg.Clock
	if clk == nil {
		clk = time.Now
	}
	owner := cfg.DefaultOwner
	if owner == "" {
		owner = DefaultApplierOwner
	}
	return &Applier{
		store:              cfg.Store,
		assigner:           cfg.Assigner,
		creator:            cfg.Creator,
		itemLifecycle:      cfg.ItemLifecycle,
		milestoneLifecycle: cfg.MilestoneLifecycle,
		cancel:             cfg.Canceller,
		invalidator:        cfg.Invalidator,
		events:             cfg.Events,
		clock:              clk,
		defaultOwner:       owner,
	}, nil
}

// Outcome reports the result of applying a single mutation.
//
// Exactly one of Applied, Skipped, or an Error-bearing failure is true.
// Skipped means the user did not select this mutation (expected partial
// accept) and is distinct from a failure — renderers should surface the
// two differently so users don't confuse "chose not to apply" with
// "tried to apply and something broke."
type Outcome struct {
	MutationID string `json:"mutation_id"`
	Op         Op     `json:"op"`
	Target     string `json:"target,omitempty"`
	Applied    bool   `json:"applied"`
	Skipped    bool   `json:"skipped,omitempty"`
	Error      string `json:"error,omitempty"`
}

// ApplyResult bundles per-mutation outcomes and the overall summary. A
// partial apply (some succeed, some fail) still returns a nil error; the
// caller inspects Outcomes for per-mutation results. A non-nil error is
// reserved for pre-flight failures (invalid proposal, no store) before any
// mutation runs.
type ApplyResult struct {
	Outcomes []Outcome `json:"outcomes"`
	Applied  int       `json:"applied"`
	Failed   int       `json:"failed"`
	Skipped  int       `json:"skipped"`
}

// Apply runs the selected mutations sequentially. If `acceptedIDs` is nil,
// all mutations are applied; otherwise only those whose IDs are in the set.
//
// The proposal must already be in FormMutationList. Re-validates against
// `current` before applying.
func (a *Applier) Apply(ctx context.Context, proposal Proposal, current CurrentState, acceptedIDs []string, source Source) (*ApplyResult, error) {
	if proposal.Form != FormMutationList {
		return nil, fmt.Errorf("apply requires form=%s (got %s); call Normalize first", FormMutationList, proposal.Form)
	}
	if err := Validate(proposal, current); err != nil {
		return nil, err
	}
	// A subset is its own proposal. Validating only the full list would let an
	// operator accept a mutation whose premise they rejected — an edge whose
	// endpoint was never created, a patch to a split's output. Those survive
	// the full-list check because the rejected mutation supplied the premise.
	if acceptedIDs != nil {
		if err := Validate(Subset(proposal, acceptedIDs), current); err != nil {
			return nil, fmt.Errorf("accepted subset is not applicable on its own: %w", err)
		}
	}
	if strings.TrimSpace(source.MilestoneName) == "" {
		return nil, errors.New("apply requires source.MilestoneName")
	}
	if source.MilestoneName != current.MilestoneName {
		return nil, fmt.Errorf("source milestone %q does not match current state %q", source.MilestoneName, current.MilestoneName)
	}

	accepted := acceptSet(acceptedIDs)
	result := &ApplyResult{Outcomes: make([]Outcome, 0, len(proposal.Mutations))}

	for _, m := range proposal.Mutations {
		outcome := Outcome{MutationID: m.ID, Op: m.Op, Target: applyTarget(m)}
		if accepted != nil {
			if _, ok := accepted[m.ID]; !ok {
				result.Skipped++
				outcome.Applied = false
				outcome.Skipped = true
				outcome.Error = ""
				result.Outcomes = append(result.Outcomes, outcome)
				continue
			}
		}
		// Per-mutation cancellation check. Callers that abort mid-batch
		// (shutdown, request timeout) see the remaining mutations
		// surface as ctx.Err() rather than silently completing slow I/O.
		if err := ctx.Err(); err != nil {
			outcome.Applied = false
			outcome.Error = err.Error()
			result.Failed++
			result.Outcomes = append(result.Outcomes, outcome)
			continue
		}
		if err := a.applyOneSafe(ctx, m, current, source); err != nil {
			outcome.Applied = false
			outcome.Error = err.Error()
			result.Failed++
			slog.Warn("proposals: mutation failed",
				"milestone", source.MilestoneName,
				"mutation", m.ID,
				"op", m.Op,
				"err", err,
			)
		} else {
			outcome.Applied = true
			result.Applied++
			slog.Info("proposals: mutation applied",
				"milestone", source.MilestoneName,
				"feedback_round", source.FeedbackRoundID,
				"mutation", m.ID,
				"op", m.Op,
				"target", outcome.Target,
				"entrypoint", source.Entrypoint,
			)
			if a.events != nil {
				a.events.EmitProposalMutationApplied(source, m)
			}
		}
		result.Outcomes = append(result.Outcomes, outcome)
	}

	if result.Applied > 0 && a.invalidator != nil {
		a.invalidator.ScheduleAll()
	}
	return result, nil
}

// StateBuilder loads CurrentState for an milestone. The single argument is
// the milestone name and matches the signatures used by feedback rounds and
// operating-mode reconciliation, so the same builder closure can drive both.
type StateBuilder func(milestoneName string) (CurrentState, error)

// ApplyFlow is the canonical recipe for turning an agent-supplied proposal
// into applied mutations: build state for the source milestone, Normalize
// the proposal against that state, then Apply.
//
// Every surface that applies proposals (feedback rounds, operating-mode
// reconciliation, future agent surfaces) MUST call ApplyFlow rather than
// re-implementing the recipe. The Applier's contract assumes pre-normalized
// input; skipping Normalize means agent-produced whitespace/casing quirks
// (e.g. "  ready  ", "EXECUTE/Foo") fall through Validate and surface as
// per-mutation errors instead of being canonicalized.
//
// Errors returned here are pre-flight: state-build, normalize, or
// Apply-level rejection (invalid form, missing milestone). A successful
// call returns a non-nil result whose Outcomes capture per-mutation
// success/failure; callers inspect those rather than expecting a non-nil
// error on partial failure.
func (a *Applier) ApplyFlow(ctx context.Context, proposal Proposal, stateBuilder StateBuilder, acceptedIDs []string, source Source) (*ApplyResult, error) {
	if stateBuilder == nil {
		return nil, errors.New("proposals: ApplyFlow requires a StateBuilder")
	}
	if strings.TrimSpace(source.MilestoneName) == "" {
		return nil, errors.New("proposals: ApplyFlow requires source.MilestoneName")
	}
	state, err := stateBuilder(source.MilestoneName)
	if err != nil {
		return nil, fmt.Errorf("build proposal state: %w", err)
	}
	normalized, err := Normalize(proposal, state)
	if err != nil {
		return nil, fmt.Errorf("normalize proposal: %w", err)
	}
	return a.Apply(ctx, normalized, state, acceptedIDs, source)
}

func applyTarget(m Mutation) string {
	switch m.Op {
	case OpAddItem:
		if m.Item != nil {
			return m.Item.Ref()
		}
		return ""
	case OpAddEdge, OpRemoveEdge:
		return m.From + "->" + m.To
	case OpMergeItems:
		if m.Item != nil {
			return m.Item.Ref()
		}
		return ""
	case OpRecreateMilestone:
		return m.Target
	}
	return m.Target
}

// Subset returns the proposal restricted to the accepted mutation IDs,
// preserving order and envelope metadata. A nil id list returns the proposal
// unchanged, matching Apply's "nil accepts everything" contract.
func Subset(p Proposal, acceptedIDs []string) Proposal {
	if acceptedIDs == nil {
		return p
	}
	accepted := acceptSet(acceptedIDs)
	out := p
	out.Mutations = make([]Mutation, 0, len(p.Mutations))
	for _, m := range p.Mutations {
		if _, ok := accepted[m.ID]; ok {
			out.Mutations = append(out.Mutations, m)
		}
	}
	return out
}

func acceptSet(ids []string) map[string]struct{} {
	if ids == nil {
		return nil
	}
	out := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return out
}

// applyOneSafe wraps applyOne in a recover so a panic mid-batch (e.g. a
// nil-pointer dereference in an event-log emit) becomes a per-mutation
// failure recorded in the Outcome instead of unwinding the whole HTTP
// request and stranding partially-written items on disk. The stack trace
// is logged so the underlying defect remains diagnosable. Without this,
// a single faulty downstream surface causes the Apply button to return
// 500 with prior mutations already persisted but no apply_result.
func (a *Applier) applyOneSafe(ctx context.Context, m Mutation, current CurrentState, source Source) (err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("proposals: mutation panicked",
				"milestone", source.MilestoneName,
				"mutation", m.ID,
				"op", m.Op,
				"panic", r,
				"stack", string(debug.Stack()),
			)
			err = fmt.Errorf("mutation panicked: %v", r)
		}
	}()
	return a.applyOne(ctx, m, current, source)
}

func (a *Applier) applyOne(ctx context.Context, m Mutation, current CurrentState, source Source) error {
	switch m.Op {
	case OpAddItem:
		return a.applyAddItem(ctx, *m.Item, source)
	case OpUpdateItem:
		return a.applyUpdateItem(ctx, m.Target, m.Patch)
	case OpChangeStatus:
		return a.applyChangeStatus(ctx, m.Target, m.Status)
	case OpChangePriority:
		return a.applyChangePriority(ctx, m.Target, *m.Priority)
	case OpAddEdge:
		return a.applyAddEdge(ctx, m.From, m.To)
	case OpRemoveEdge:
		return a.applyRemoveEdge(ctx, m.From, m.To)
	case OpMoveMilestone:
		return a.applyMoveMilestone(ctx, m.Target, source.MilestoneName, m.Milestone)
	case OpArchiveItem:
		return a.applyArchive(ctx, m.Target)
	case OpInterruptInProgress:
		return a.applyInterrupt(ctx, m.Target)
	case OpSplitItem:
		return a.applySplit(ctx, m.Target, m.Into, source)
	case OpMergeItems:
		if m.Item == nil {
			return fmt.Errorf("merge_items: missing merged item spec")
		}
		return a.applyMergeItems(ctx, m.Sources, *m.Item, current, source)
	case OpRecreateItem:
		return a.applyRecreateItem(ctx, m.Target)
	case OpResetArtifacts:
		return a.applyResetArtifacts(ctx, m.Target, m.ResetScope)
	case OpRecreateMilestone:
		return a.applyRecreateMilestone(ctx, m.Target)
	default:
		// Defence against a new op added to types.go without a
		// handler here: every recognized op in AllOps must map to a
		// case above. Unknown ops are already caught by Validate, so
		// reaching the default means the op is known but unhandled —
		// a programmer error surfaced loudly rather than silently
		// swallowed as a skipped mutation.
		return fmt.Errorf("%w: %s", ErrUnknownOp, m.Op)
	}
}
