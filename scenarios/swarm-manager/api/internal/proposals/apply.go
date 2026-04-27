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
	SetItemInitiative(kind backlog.BacklogKind, name, initiative string) (string, error)
	ClearItemInitiative(kind backlog.BacklogKind, name, expected string) (string, bool, error)
	ValidateDependencies(dependsOn []string) error
}

// InitiativeAssigner is the initiative-side cascade surface the Applier uses.
// Satisfied by initiatives.Service; mirrors backlog.InitiativeAssigner but is
// redefined here to keep proposals a leaf package.
type InitiativeAssigner interface {
	RememberItem(initiativeName, ref string) error
	ForgetItem(initiativeName, ref string) error
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

// Applier executes accepted mutations against the underlying services.
//
// Correct use: normalize → validate → filter to accepted IDs → Apply. The
// Applier re-validates defensively, so a malformed mutation surfaces as an
// Outcome error rather than a partial mutation.
type Applier struct {
	store        BacklogStore
	assigner     InitiativeAssigner
	creator      ItemCreator
	cancel       ExecutionCanceller
	invalidator  GraphInvalidator
	events       EventEmitter
	clock        func() time.Time
	defaultOwner string
}

// Config bundles Applier dependencies.
type Config struct {
	Store        BacklogStore
	Assigner     InitiativeAssigner
	Creator      ItemCreator
	Canceller    ExecutionCanceller
	Invalidator  GraphInvalidator
	Events       EventEmitter
	Clock        func() time.Time // optional; defaults to time.Now
	DefaultOwner string           // falls back to "proposal" if empty
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
		store:        cfg.Store,
		assigner:     cfg.Assigner,
		creator:      cfg.Creator,
		cancel:       cfg.Canceller,
		invalidator:  cfg.Invalidator,
		events:       cfg.Events,
		clock:        clk,
		defaultOwner: owner,
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
	if strings.TrimSpace(source.InitiativeName) == "" {
		return nil, errors.New("apply requires source.InitiativeName")
	}
	if source.InitiativeName != current.InitiativeName {
		return nil, fmt.Errorf("source initiative %q does not match current state %q", source.InitiativeName, current.InitiativeName)
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
		if err := a.applyOneSafe(ctx, m, source); err != nil {
			outcome.Applied = false
			outcome.Error = err.Error()
			result.Failed++
			slog.Warn("proposals: mutation failed",
				"initiative", source.InitiativeName,
				"mutation", m.ID,
				"op", m.Op,
				"err", err,
			)
		} else {
			outcome.Applied = true
			result.Applied++
			slog.Info("proposals: mutation applied",
				"initiative", source.InitiativeName,
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

func applyTarget(m Mutation) string {
	switch m.Op {
	case OpAddItem:
		if m.Item != nil {
			return m.Item.Ref()
		}
		return ""
	case OpAddEdge, OpRemoveEdge:
		return m.From + "->" + m.To
	}
	return m.Target
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
func (a *Applier) applyOneSafe(ctx context.Context, m Mutation, source Source) (err error) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("proposals: mutation panicked",
				"initiative", source.InitiativeName,
				"mutation", m.ID,
				"op", m.Op,
				"panic", r,
				"stack", string(debug.Stack()),
			)
			err = fmt.Errorf("mutation panicked: %v", r)
		}
	}()
	return a.applyOne(ctx, m, source)
}

func (a *Applier) applyOne(ctx context.Context, m Mutation, source Source) error {
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
	case OpMoveInitiative:
		return a.applyMoveInitiative(ctx, m.Target, source.InitiativeName, m.Initiative)
	case OpArchiveItem:
		return a.applyArchive(ctx, m.Target)
	case OpInterruptInProgress:
		return a.applyInterrupt(ctx, m.Target)
	case OpSplitItem:
		return a.applySplit(ctx, m.Target, m.Into, source)
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

func (a *Applier) applyAddItem(ctx context.Context, spec ItemSpec, source Source) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	kind, err := backlog.ParseBacklogKind(spec.Kind)
	if err != nil {
		return err
	}

	now := a.clock().UTC().Format(time.RFC3339)
	item := backlog.BacklogItem{
		Name:            spec.Name,
		Title:           strings.TrimSpace(spec.Title),
		Description:     spec.Description,
		Kind:            kind,
		Status:          backlog.StatusBacklog,
		Priority:        spec.Priority,
		Tags:            append([]string(nil), spec.Tags...),
		DependsOn:       append([]string(nil), spec.DependsOn...),
		Effort:          strings.ToUpper(strings.TrimSpace(spec.Effort)),
		Initiative:      source.InitiativeName,
		AcceptanceAllow: append([]string(nil), spec.AcceptanceAllow...),
		AcceptanceDeny:  append([]string(nil), spec.AcceptanceDeny...),
		Note:            spec.Note,
		Created:         now,
		Updated:         now,
	}
	if item.Priority == 0 {
		item.Priority = DefaultItemPriority
	}

	cc := backlog.CreationContext{
		Source:          backlog.SourceProposal,
		DecidedBy:       source.DecidedBy,
		FeedbackRoundID: source.FeedbackRoundID,
		ReviewRoundID:   source.ReviewRoundID,
		RoundNumber:     source.RoundNumber,
		RoundSlug:       source.RoundSlug,
		Entrypoint:      source.Entrypoint,
		// Apply re-validates depends_on against ValidateDependencies via
		// CurrentState; the per-item cycle check inside Service is
		// redundant here (and would require a full LoadAll on a path
		// that already paid the cost in Validate). Skip it.
		SkipCycleCheck: true,
		// The Applier batches graph invalidation across the whole
		// mutation set in Apply (see ScheduleAll at the bottom of
		// the loop) so individual add_item ops should not fire one
		// re-projection each.
		SkipGraphInvalidation: true,
	}
	if err := a.creator.Create(item, cc); err != nil {
		if errors.Is(err, backlog.ErrItemExists) {
			return fmt.Errorf("%w: %s", ErrDuplicateItem, spec.Ref())
		}
		return err
	}
	return nil
}

func (a *Applier) applyUpdateItem(ctx context.Context, ref string, patch *ItemPatch) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	kind, name, err := splitRef(ref)
	if err != nil {
		return err
	}
	item, err := a.store.LoadItem(kind, name)
	if err != nil {
		return err
	}
	// Dependency cycles are a business-level concern the applier enforces
	// before delegating to the shared patch helper — ApplyItemPatch is
	// intentionally semantics-free about validity.
	if patch.DependsOn != nil && len(*patch.DependsOn) > 0 {
		if err := a.store.ValidateDependencies(*patch.DependsOn); err != nil {
			return fmt.Errorf("depends_on: %w", err)
		}
	}
	backlog.ApplyItemPatch(&item, toBacklogPatch(patch))
	item.Updated = a.clock().UTC().Format(time.RFC3339)
	return a.store.SaveItem(item)
}

// toBacklogPatch converts proposals' ItemPatch (the wire shape an agent
// produces) into backlog.ItemPatch (the shared mutation primitive). Ops
// that touch status/initiative/spawned_from are excluded here — those
// have dedicated mutations so misuse surfaces as an unknown op instead
// of a silent field smuggle.
func toBacklogPatch(p *ItemPatch) backlog.ItemPatch {
	if p == nil {
		return backlog.ItemPatch{}
	}
	out := backlog.ItemPatch{
		Title:       p.Title,
		Description: p.Description,
		Priority:    p.Priority,
		Effort:      p.Effort,
		Note:        p.Note,
	}
	if p.Tags != nil {
		cp := append([]string(nil), *p.Tags...)
		out.Tags = &cp
	}
	if p.DependsOn != nil {
		cp := append([]string(nil), *p.DependsOn...)
		out.DependsOn = &cp
	}
	if p.AcceptanceAllow != nil {
		cp := append([]string(nil), *p.AcceptanceAllow...)
		out.AcceptanceAllow = &cp
	}
	if p.AcceptanceDeny != nil {
		cp := append([]string(nil), *p.AcceptanceDeny...)
		out.AcceptanceDeny = &cp
	}
	return out
}

func (a *Applier) applyChangeStatus(ctx context.Context, ref, status string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	kind, name, err := splitRef(ref)
	if err != nil {
		return err
	}
	item, err := a.store.LoadItem(kind, name)
	if err != nil {
		return err
	}
	newStatus := backlog.BacklogStatus(strings.ToLower(strings.TrimSpace(status)))
	if backlog.IsTerminalStatus(newStatus) {
		return fmt.Errorf("%w: %s", ErrTerminalStatusWrite, newStatus)
	}
	item.Status = newStatus
	item.Updated = a.clock().UTC().Format(time.RFC3339)
	return a.store.SaveItem(item)
}

func (a *Applier) applyChangePriority(ctx context.Context, ref string, priority int) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	kind, name, err := splitRef(ref)
	if err != nil {
		return err
	}
	item, err := a.store.LoadItem(kind, name)
	if err != nil {
		return err
	}
	item.Priority = priority
	item.Updated = a.clock().UTC().Format(time.RFC3339)
	return a.store.SaveItem(item)
}

func (a *Applier) applyAddEdge(ctx context.Context, from, to string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	kind, name, err := splitRef(from)
	if err != nil {
		return err
	}
	item, err := a.store.LoadItem(kind, name)
	if err != nil {
		return err
	}
	for _, dep := range item.DependsOn {
		if dep == to {
			return fmt.Errorf("edge already exists: %s -> %s", from, to)
		}
	}
	deps := append(append([]string(nil), item.DependsOn...), to)
	if err := a.store.ValidateDependencies(deps); err != nil {
		return fmt.Errorf("depends_on: %w", err)
	}
	item.DependsOn = deps
	item.Updated = a.clock().UTC().Format(time.RFC3339)
	return a.store.SaveItem(item)
}

func (a *Applier) applyRemoveEdge(ctx context.Context, from, to string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	kind, name, err := splitRef(from)
	if err != nil {
		return err
	}
	item, err := a.store.LoadItem(kind, name)
	if err != nil {
		return err
	}
	filtered := make([]string, 0, len(item.DependsOn))
	removed := false
	for _, dep := range item.DependsOn {
		if dep == to {
			removed = true
			continue
		}
		filtered = append(filtered, dep)
	}
	if !removed {
		return fmt.Errorf("edge does not exist: %s -> %s", from, to)
	}
	item.DependsOn = filtered
	item.Updated = a.clock().UTC().Format(time.RFC3339)
	return a.store.SaveItem(item)
}

func (a *Applier) applyMoveInitiative(ctx context.Context, ref, currentInit, destination string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	kind, name, err := splitRef(ref)
	if err != nil {
		return err
	}
	dest := strings.TrimSpace(destination)

	if currentInit != "" {
		if err := a.assigner.ForgetItem(currentInit, ref); err != nil {
			return fmt.Errorf("detach from %s: %w", currentInit, err)
		}
	}

	if dest == "" {
		if _, _, err := a.store.ClearItemInitiative(kind, name, currentInit); err != nil {
			return fmt.Errorf("clear initiative field: %w", err)
		}
		item, loadErr := a.store.LoadItem(kind, name)
		if loadErr == nil {
			item.Updated = a.clock().UTC().Format(time.RFC3339)
			_ = a.store.SaveItem(item)
		}
		return nil
	}

	if _, err := a.store.SetItemInitiative(kind, name, dest); err != nil {
		// Roll back detach so the item isn't orphaned.
		if currentInit != "" {
			_ = a.assigner.RememberItem(currentInit, ref)
		}
		return fmt.Errorf("set initiative field: %w", err)
	}
	if err := a.assigner.RememberItem(dest, ref); err != nil {
		// Full rollback: the item now claims `dest` on disk but neither
		// initiative lists it. Restore the item field and re-attach to
		// the original initiative so the user sees pre-mutation state.
		if _, _, clearErr := a.store.ClearItemInitiative(kind, name, dest); clearErr != nil {
			return fmt.Errorf("attach to %s: %w; rollback clear failed: %v", dest, err, clearErr)
		}
		if currentInit != "" {
			if _, restoreErr := a.store.SetItemInitiative(kind, name, currentInit); restoreErr != nil {
				return fmt.Errorf("attach to %s: %w; rollback restore failed: %v", dest, err, restoreErr)
			}
			if remErr := a.assigner.RememberItem(currentInit, ref); remErr != nil {
				return fmt.Errorf("attach to %s: %w; rollback remember failed: %v", dest, err, remErr)
			}
		}
		return fmt.Errorf("attach to %s: %w", dest, err)
	}
	item, loadErr := a.store.LoadItem(kind, name)
	if loadErr == nil {
		item.Updated = a.clock().UTC().Format(time.RFC3339)
		_ = a.store.SaveItem(item)
	}
	return nil
}

func (a *Applier) applyArchive(ctx context.Context, ref string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	kind, name, err := splitRef(ref)
	if err != nil {
		return err
	}
	item, err := a.store.LoadItem(kind, name)
	if err != nil {
		return err
	}
	if item.ArchivedAt != nil && *item.ArchivedAt != "" {
		return nil
	}
	ts := a.clock().UTC().Format(time.RFC3339)
	item.ArchivedAt = &ts
	item.Updated = ts
	return a.store.SaveItem(item)
}

func (a *Applier) applyInterrupt(ctx context.Context, ref string) error {
	if a.cancel == nil {
		return errors.New("interrupt_in_progress requires an ExecutionCanceller; none wired")
	}
	kind, name, err := splitRef(ref)
	if err != nil {
		return err
	}
	return a.cancel.CancelForBacklog(ctx, string(kind), name)
}

func (a *Applier) applySplit(ctx context.Context, ref string, into []ItemSpec, source Source) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	kind, name, err := splitRef(ref)
	if err != nil {
		return err
	}
	_, err = a.store.LoadItem(kind, name)
	if err != nil {
		return err
	}

	created := make([]string, 0, len(into))
	rollback := func(reason error) {
		for _, r := range created {
			if rbErr := a.applyArchive(ctx, r); rbErr != nil {
				slog.Warn("proposals: split rollback failed",
					"source", ref,
					"child", r,
					"reason", reason,
					"err", rbErr,
				)
			}
		}
	}
	for _, spec := range into {
		if err := a.applyAddItem(ctx, spec, source); err != nil {
			rollback(err)
			return fmt.Errorf("create split child %s: %w", spec.Ref(), err)
		}
		created = append(created, spec.Ref())
	}

	// True atomicity: if archiving the source fails after all children
	// land, roll back the children too so the user sees pre-split state
	// instead of a half-split orphan graph. Dependents of the source
	// stay pointed at the original ref and must be retargeted by
	// subsequent OpAddEdge / OpRemoveEdge mutations — split does not
	// retarget dependents implicitly (see types.go OpSplitItem).
	if err := a.applyArchive(ctx, ref); err != nil {
		rollback(err)
		return fmt.Errorf("archive source item: %w", err)
	}
	return nil
}

func splitRef(ref string) (backlog.BacklogKind, string, error) {
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid ref %q", ref)
	}
	kind, err := backlog.ParseBacklogKind(parts[0])
	if err != nil {
		return "", "", err
	}
	return kind, parts[1], nil
}
