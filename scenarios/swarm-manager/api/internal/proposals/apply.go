package proposals

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"swarm-manager/internal/backlog"
)

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

// Applier executes accepted mutations against the underlying services.
//
// Correct use: normalize → validate → filter to accepted IDs → Apply. The
// Applier re-validates defensively, so a malformed mutation surfaces as an
// Outcome error rather than a partial mutation.
type Applier struct {
	store        BacklogStore
	assigner     InitiativeAssigner
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
	Canceller    ExecutionCanceller
	Invalidator  GraphInvalidator
	Events       EventEmitter
	Clock        func() time.Time // optional; defaults to time.Now
	DefaultOwner string           // falls back to "proposal" if empty
}

// NewApplier constructs an Applier. Store and Assigner are required; the
// others may be nil for tests or degraded modes.
func NewApplier(cfg Config) (*Applier, error) {
	if cfg.Store == nil {
		return nil, errors.New("proposals: Store is required")
	}
	if cfg.Assigner == nil {
		return nil, errors.New("proposals: Assigner is required")
	}
	clk := cfg.Clock
	if clk == nil {
		clk = time.Now
	}
	owner := cfg.DefaultOwner
	if owner == "" {
		owner = "proposal"
	}
	return &Applier{
		store:        cfg.Store,
		assigner:     cfg.Assigner,
		cancel:       cfg.Canceller,
		invalidator:  cfg.Invalidator,
		events:       cfg.Events,
		clock:        clk,
		defaultOwner: owner,
	}, nil
}

// Outcome reports the result of applying a single mutation.
type Outcome struct {
	MutationID string `json:"mutation_id"`
	Op         Op     `json:"op"`
	Target     string `json:"target,omitempty"`
	Applied    bool   `json:"applied"`
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
				outcome.Error = ""
				result.Outcomes = append(result.Outcomes, outcome)
				continue
			}
		}
		if err := a.applyOne(ctx, m, source); err != nil {
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

func (a *Applier) applyOne(ctx context.Context, m Mutation, source Source) error {
	switch m.Op {
	case OpAddItem:
		return a.applyAddItem(*m.Item, source)
	case OpUpdateItem:
		return a.applyUpdateItem(m.Target, m.Patch)
	case OpChangeStatus:
		return a.applyChangeStatus(m.Target, m.Status)
	case OpChangePriority:
		return a.applyChangePriority(m.Target, *m.Priority)
	case OpAddEdge:
		return a.applyAddEdge(m.From, m.To)
	case OpRemoveEdge:
		return a.applyRemoveEdge(m.From, m.To)
	case OpMoveInitiative:
		return a.applyMoveInitiative(m.Target, source.InitiativeName, m.Initiative)
	case OpArchiveItem:
		return a.applyArchive(m.Target)
	case OpInterruptInProgress:
		return a.applyInterrupt(ctx, m.Target)
	case OpSplitItem:
		return a.applySplit(m.Target, m.Into, source)
	}
	return fmt.Errorf("%w: %s", ErrUnknownOp, m.Op)
}

func (a *Applier) applyAddItem(spec ItemSpec, source Source) error {
	kind, err := backlog.ParseBacklogKind(spec.Kind)
	if err != nil {
		return err
	}
	if _, loadErr := a.store.LoadItem(kind, spec.Name); loadErr == nil {
		return fmt.Errorf("%w: %s", ErrDuplicateItem, spec.Ref())
	}

	if len(spec.DependsOn) > 0 {
		if err := a.store.ValidateDependencies(spec.DependsOn); err != nil {
			return fmt.Errorf("depends_on: %w", err)
		}
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
		item.Priority = 5
	}

	if err := os.MkdirAll(a.store.ItemDir(kind, spec.Name), 0o755); err != nil {
		return fmt.Errorf("create item dir: %w", err)
	}
	if err := a.store.SaveItem(item); err != nil {
		return fmt.Errorf("save item: %w", err)
	}
	if err := a.assigner.RememberItem(source.InitiativeName, string(item.Kind)+"/"+item.Name); err != nil {
		return fmt.Errorf("attach to initiative: %w", err)
	}
	return nil
}

func (a *Applier) applyUpdateItem(ref string, patch *ItemPatch) error {
	kind, name, err := splitRef(ref)
	if err != nil {
		return err
	}
	item, err := a.store.LoadItem(kind, name)
	if err != nil {
		return err
	}
	if patch.Title != nil {
		item.Title = strings.TrimSpace(*patch.Title)
	}
	if patch.Description != nil {
		item.Description = *patch.Description
	}
	if patch.Priority != nil {
		item.Priority = *patch.Priority
	}
	if patch.Tags != nil {
		item.Tags = append([]string(nil), *patch.Tags...)
	}
	if patch.DependsOn != nil {
		deps := append([]string(nil), *patch.DependsOn...)
		if len(deps) > 0 {
			if err := a.store.ValidateDependencies(deps); err != nil {
				return fmt.Errorf("depends_on: %w", err)
			}
		}
		item.DependsOn = deps
	}
	if patch.Effort != nil {
		item.Effort = strings.ToUpper(strings.TrimSpace(*patch.Effort))
	}
	if patch.AcceptanceAllow != nil {
		item.AcceptanceAllow = append([]string(nil), *patch.AcceptanceAllow...)
	}
	if patch.AcceptanceDeny != nil {
		item.AcceptanceDeny = append([]string(nil), *patch.AcceptanceDeny...)
	}
	if patch.Note != nil {
		item.Note = strings.TrimSpace(*patch.Note)
	}
	item.Updated = a.clock().UTC().Format(time.RFC3339)
	return a.store.SaveItem(item)
}

func (a *Applier) applyChangeStatus(ref, status string) error {
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

func (a *Applier) applyChangePriority(ref string, priority int) error {
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

func (a *Applier) applyAddEdge(from, to string) error {
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

func (a *Applier) applyRemoveEdge(from, to string) error {
	kind, name, err := splitRef(from)
	if err != nil {
		return err
	}
	item, err := a.store.LoadItem(kind, name)
	if err != nil {
		return err
	}
	filtered := item.DependsOn[:0]
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
	item.DependsOn = append([]string(nil), filtered...)
	item.Updated = a.clock().UTC().Format(time.RFC3339)
	return a.store.SaveItem(item)
}

func (a *Applier) applyMoveInitiative(ref, currentInit, destination string) error {
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
		return fmt.Errorf("attach to %s: %w", dest, err)
	}
	item, loadErr := a.store.LoadItem(kind, name)
	if loadErr == nil {
		item.Updated = a.clock().UTC().Format(time.RFC3339)
		_ = a.store.SaveItem(item)
	}
	return nil
}

func (a *Applier) applyArchive(ref string) error {
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

func (a *Applier) applySplit(ref string, into []ItemSpec, source Source) error {
	kind, name, err := splitRef(ref)
	if err != nil {
		return err
	}
	_, err = a.store.LoadItem(kind, name)
	if err != nil {
		return err
	}

	created := make([]string, 0, len(into))
	for _, spec := range into {
		if err := a.applyAddItem(spec, source); err != nil {
			// Best-effort rollback: archive any items already created so the
			// split is all-or-nothing from the user's perspective.
			for _, r := range created {
				_ = a.applyArchive(r)
			}
			return fmt.Errorf("create split child %s: %w", spec.Ref(), err)
		}
		created = append(created, spec.Ref())
	}

	if err := a.applyArchive(ref); err != nil {
		// Leave children created; the source archive failure is surfaced.
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
