package proposals

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"swarm-manager/internal/backlog"
)

// CurrentState is the read-only snapshot of the initiative that a proposal
// will be validated and applied against. Constructed from a materialized
// graph.json (graph.MaterializedGraph) plus the set of known initiative
// names for move_initiative validation.
//
// Kept narrow so callers can build one from any source — tests use a fixture
// literal; the HTTP layer hydrates from the graph Materializer.
type CurrentState struct {
	// InitiativeName is the initiative the proposal targets.
	InitiativeName string

	// Nodes indexes members of the initiative by ref ("kind/name").
	Nodes map[string]GraphNode

	// Edges lists dependency edges within the initiative. Order-insensitive.
	Edges []GraphEdge

	// KnownInitiatives is the set of initiative names that exist on disk.
	// Used by OpMoveInitiative to reject phantom destinations.
	KnownInitiatives map[string]struct{}

	// InProgressRefs is the subset of Nodes whose items are currently
	// StatusInProgress — gates OpInterruptInProgress to realistic targets.
	InProgressRefs map[string]struct{}
}

// HasNode reports whether a ref is a member of the initiative's current graph.
func (s *CurrentState) HasNode(ref string) bool {
	_, ok := s.Nodes[ref]
	return ok
}

// HasEdge reports whether an edge exists in the initiative's current graph.
func (s *CurrentState) HasEdge(from, to string) bool {
	for _, e := range s.Edges {
		if e.From == from && e.To == to {
			return true
		}
	}
	return false
}

// refPattern mirrors the sanitizeName rules used by batch-create: lowercase
// alphanumerics and hyphens in the name segment.
var refPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*/[a-z0-9][a-z0-9-]*$`)

// validateRef checks the "kind/name" shape and that kind is a known backlog
// kind. Callers still need to check membership via CurrentState.HasNode.
func validateRef(ref string) error {
	if !refPattern.MatchString(ref) {
		return fmt.Errorf("invalid ref %q: expected kind/name with lowercase letters, digits, hyphens", ref)
	}
	kind := strings.SplitN(ref, "/", 2)[0]
	if _, err := backlog.ParseBacklogKind(kind); err != nil {
		return fmt.Errorf("invalid ref %q: %w", ref, err)
	}
	return nil
}

// Validate checks a mutation_list proposal for schema and semantic errors
// against the given CurrentState. Proposals in FormFullGraph must be passed
// through Normalize first.
//
// Returns a joined error describing *all* problems found — not just the
// first — so the agent can revise in a single turn. Any error satisfies
// errors.Is(err, ErrInvalidProposal).
func Validate(p Proposal, state CurrentState) error {
	if p.Form != FormMutationList {
		return fmt.Errorf("%w: Validate requires form=%s (got %s); call Normalize first", ErrInvalidProposal, FormMutationList, p.Form)
	}

	var problems []error
	seenIDs := make(map[string]int, len(p.Mutations))
	newItems := make(map[string]int, len(p.Mutations)) // ref -> mutation index

	for i, m := range p.Mutations {
		prefix := fmt.Sprintf("mutations[%d]", i)
		if strings.TrimSpace(m.ID) == "" {
			problems = append(problems, fmt.Errorf("%s: id is required", prefix))
		} else if prev, dup := seenIDs[m.ID]; dup {
			problems = append(problems, fmt.Errorf("%s: duplicate id %q (first used by mutations[%d])", prefix, m.ID, prev))
		} else {
			seenIDs[m.ID] = i
		}

		if m.Op == "" {
			problems = append(problems, fmt.Errorf("%s: op is required", prefix))
			continue
		}
		if !isKnownOp(m.Op) {
			problems = append(problems, fmt.Errorf("%s: %w %q", prefix, ErrUnknownOp, m.Op))
			continue
		}

		if err := validateMutation(m, i, state, newItems); err != nil {
			problems = append(problems, fmt.Errorf("%s: %w", prefix, err))
		}
	}

	if len(problems) == 0 {
		return nil
	}
	return errors.Join(append([]error{ErrInvalidProposal}, problems...)...)
}

func isKnownOp(op Op) bool {
	for _, known := range AllOps() {
		if op == known {
			return true
		}
	}
	return false
}

// validateMutation returns a non-nil error with an actionable message if the
// mutation is malformed. newItems is mutated to track add_item/split_item
// outputs so later mutations can reference them.
func validateMutation(m Mutation, idx int, state CurrentState, newItems map[string]int) error {
	switch m.Op {
	case OpAddItem:
		return validateAddItem(m, idx, state, newItems)
	case OpUpdateItem:
		return validateUpdateItem(m, state, newItems)
	case OpChangeStatus:
		return validateChangeStatus(m, state, newItems)
	case OpChangePriority:
		return validateChangePriority(m, state, newItems)
	case OpAddEdge:
		return validateEdge(m, state, newItems, true)
	case OpRemoveEdge:
		return validateEdge(m, state, newItems, false)
	case OpMoveInitiative:
		return validateMoveInitiative(m, state, newItems)
	case OpArchiveItem:
		return validateTargetExists(m, state, newItems)
	case OpInterruptInProgress:
		return validateInterrupt(m, state)
	case OpSplitItem:
		return validateSplitItem(m, idx, state, newItems)
	}
	return fmt.Errorf("%w: %s", ErrUnknownOp, m.Op)
}

func validateAddItem(m Mutation, idx int, state CurrentState, newItems map[string]int) error {
	if m.Item == nil {
		return fmt.Errorf("op %s requires item spec", m.Op)
	}
	if err := validateItemSpec(*m.Item); err != nil {
		return err
	}
	ref := m.Item.Ref()
	if _, collides := state.Nodes[ref]; collides {
		return fmt.Errorf("%w: %s already exists in initiative", ErrDuplicateItem, ref)
	}
	if prev, dup := newItems[ref]; dup {
		return fmt.Errorf("%w: %s already staged by mutations[%d]", ErrDuplicateItem, ref, prev)
	}
	newItems[ref] = idx
	return nil
}

func validateUpdateItem(m Mutation, state CurrentState, newItems map[string]int) error {
	if err := validateRef(m.Target); err != nil {
		return err
	}
	if !state.HasNode(m.Target) && !hasStagedNewItem(m.Target, newItems) {
		return fmt.Errorf("%w: %s", ErrTargetNotFound, m.Target)
	}
	if m.Patch == nil {
		return fmt.Errorf("op %s requires patch", m.Op)
	}
	if patchIsEmpty(m.Patch) {
		return fmt.Errorf("op %s patch must set at least one field", m.Op)
	}
	if m.Patch.Priority != nil {
		if *m.Patch.Priority < MinItemPriority || *m.Patch.Priority > MaxItemPriority {
			return fmt.Errorf("patch.priority must be between %d and %d", MinItemPriority, MaxItemPriority)
		}
	}
	if m.Patch.Effort != nil {
		if err := validateEffortValue(*m.Patch.Effort); err != nil {
			return err
		}
	}
	return nil
}

func validateChangeStatus(m Mutation, state CurrentState, newItems map[string]int) error {
	if err := validateRef(m.Target); err != nil {
		return err
	}
	if !state.HasNode(m.Target) && !hasStagedNewItem(m.Target, newItems) {
		return fmt.Errorf("%w: %s", ErrTargetNotFound, m.Target)
	}
	status := backlog.BacklogStatus(strings.ToLower(strings.TrimSpace(m.Status)))
	if status == "" {
		return fmt.Errorf("op %s requires status", m.Op)
	}
	if backlog.IsTerminalStatus(status) {
		return fmt.Errorf("%w: %s is terminal", ErrTerminalStatusWrite, status)
	}
	// Explicitly forbid lifecycle-controlled states so agents can't drive
	// them via proposals; queue/in_progress/in_review/review_pending are
	// owned by the execution + review systems respectively.
	switch status {
	case backlog.StatusQueued, backlog.StatusInProgress, backlog.StatusInReview, backlog.StatusReviewPending:
		return fmt.Errorf("status %s is controlled by the execution/review system and cannot be set via proposals", status)
	}
	// What remains: backlog, researching, ready — the user-settable set.
	switch status {
	case backlog.StatusBacklog, backlog.StatusResearching, backlog.StatusReady:
		return nil
	}
	return fmt.Errorf("status %s is not a valid proposal target", status)
}

func validateChangePriority(m Mutation, state CurrentState, newItems map[string]int) error {
	if err := validateRef(m.Target); err != nil {
		return err
	}
	if !state.HasNode(m.Target) && !hasStagedNewItem(m.Target, newItems) {
		return fmt.Errorf("%w: %s", ErrTargetNotFound, m.Target)
	}
	if m.Priority == nil {
		return fmt.Errorf("op %s requires priority", m.Op)
	}
	if *m.Priority < MinItemPriority || *m.Priority > MaxItemPriority {
		return fmt.Errorf("priority must be between %d and %d", MinItemPriority, MaxItemPriority)
	}
	return nil
}

func validateEdge(m Mutation, state CurrentState, newItems map[string]int, adding bool) error {
	if err := validateRef(m.From); err != nil {
		return fmt.Errorf("from: %w", err)
	}
	if err := validateRef(m.To); err != nil {
		return fmt.Errorf("to: %w", err)
	}
	if m.From == m.To {
		return fmt.Errorf("edge must have distinct endpoints (got %s -> %s)", m.From, m.To)
	}
	if !state.HasNode(m.From) && !hasStagedNewItem(m.From, newItems) {
		return fmt.Errorf("%w: from=%s", ErrTargetNotFound, m.From)
	}
	if !state.HasNode(m.To) && !hasStagedNewItem(m.To, newItems) {
		return fmt.Errorf("%w: to=%s", ErrTargetNotFound, m.To)
	}
	if adding && state.HasEdge(m.From, m.To) {
		return fmt.Errorf("edge already exists: %s -> %s", m.From, m.To)
	}
	if !adding && !state.HasEdge(m.From, m.To) {
		return fmt.Errorf("edge does not exist: %s -> %s", m.From, m.To)
	}
	return nil
}

func validateMoveInitiative(m Mutation, state CurrentState, newItems map[string]int) error {
	if err := validateRef(m.Target); err != nil {
		return err
	}
	if !state.HasNode(m.Target) && !hasStagedNewItem(m.Target, newItems) {
		return fmt.Errorf("%w: %s", ErrTargetNotFound, m.Target)
	}
	dest := strings.TrimSpace(m.Initiative)
	if dest == "" {
		return nil // detach: allowed
	}
	if dest == state.InitiativeName {
		return fmt.Errorf("move_initiative destination %q is the current initiative", dest)
	}
	if state.KnownInitiatives != nil {
		if _, ok := state.KnownInitiatives[dest]; !ok {
			return fmt.Errorf("move_initiative destination %q is not a known initiative", dest)
		}
	}
	return nil
}

func validateTargetExists(m Mutation, state CurrentState, newItems map[string]int) error {
	if err := validateRef(m.Target); err != nil {
		return err
	}
	if !state.HasNode(m.Target) && !hasStagedNewItem(m.Target, newItems) {
		return fmt.Errorf("%w: %s", ErrTargetNotFound, m.Target)
	}
	return nil
}

func validateInterrupt(m Mutation, state CurrentState) error {
	if err := validateRef(m.Target); err != nil {
		return err
	}
	if !state.HasNode(m.Target) {
		return fmt.Errorf("%w: %s", ErrTargetNotFound, m.Target)
	}
	if state.InProgressRefs != nil {
		if _, inProgress := state.InProgressRefs[m.Target]; !inProgress {
			return fmt.Errorf("interrupt_in_progress requires %s to be in_progress", m.Target)
		}
	}
	return nil
}

func validateSplitItem(m Mutation, idx int, state CurrentState, newItems map[string]int) error {
	if err := validateRef(m.Target); err != nil {
		return err
	}
	if !state.HasNode(m.Target) {
		return fmt.Errorf("%w: %s", ErrTargetNotFound, m.Target)
	}
	if len(m.Into) < 2 {
		return fmt.Errorf("op %s requires into[] with at least 2 new items", m.Op)
	}
	for j, spec := range m.Into {
		if err := validateItemSpec(spec); err != nil {
			return fmt.Errorf("into[%d]: %w", j, err)
		}
		ref := spec.Ref()
		if _, collides := state.Nodes[ref]; collides {
			return fmt.Errorf("into[%d]: %w: %s", j, ErrDuplicateItem, ref)
		}
		if prev, dup := newItems[ref]; dup {
			return fmt.Errorf("into[%d]: %w: %s already staged by mutations[%d]", j, ErrDuplicateItem, ref, prev)
		}
		newItems[ref] = idx
	}
	return nil
}

func validateItemSpec(spec ItemSpec) error {
	if strings.TrimSpace(spec.Kind) == "" {
		return fmt.Errorf("item.kind is required")
	}
	if _, err := backlog.ParseBacklogKind(spec.Kind); err != nil {
		return fmt.Errorf("item.kind: %w", err)
	}
	if strings.TrimSpace(spec.Name) == "" {
		return fmt.Errorf("item.name is required")
	}
	if !refPattern.MatchString(spec.Kind + "/" + spec.Name) {
		return fmt.Errorf("item.name %q must be lowercase alphanumeric and hyphens", spec.Name)
	}
	if strings.TrimSpace(spec.Title) == "" {
		return fmt.Errorf("item.title is required")
	}
	if spec.Priority != 0 && (spec.Priority < MinItemPriority || spec.Priority > MaxItemPriority) {
		return fmt.Errorf("item.priority must be between %d and %d", MinItemPriority, MaxItemPriority)
	}
	if spec.Effort != "" {
		if err := validateEffortValue(spec.Effort); err != nil {
			return err
		}
	}
	return nil
}

func validateEffortValue(raw string) error {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "", "XS", "S", "M", "L", "XL":
		return nil
	}
	return fmt.Errorf("effort must be XS, S, M, L, or XL")
}

func patchIsEmpty(p *ItemPatch) bool {
	return p.Title == nil && p.Description == nil && p.Priority == nil &&
		p.Tags == nil && p.DependsOn == nil && p.Effort == nil &&
		p.AcceptanceAllow == nil && p.AcceptanceDeny == nil && p.Note == nil
}

func hasStagedNewItem(ref string, newItems map[string]int) bool {
	_, ok := newItems[ref]
	return ok
}
