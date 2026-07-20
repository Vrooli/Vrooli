package proposals

import (
	"encoding/json"
	"fmt"
)

// Form discriminates between the two wire shapes an agent may emit. A
// mutation_list is a flat sequence of ops; a full_graph is a target-state
// description that the server diffs against current state to derive an
// equivalent mutation_list.
type Form string

const (
	FormMutationList Form = "mutation_list"
	FormFullGraph    Form = "full_graph"
)

// Op is the type tag on a single Mutation. Only the ops declared below are
// accepted by the apply layer; anything else is rejected with ErrUnknownOp
// so agents cannot silently introduce new semantics.
type Op string

const (
	// OpAddItem creates a new backlog item and attaches it to the initiative.
	OpAddItem Op = "add_item"

	// OpUpdateItem applies a partial patch to an existing item's metadata
	// (title, description, priority, tags, effort, acceptance globs, note,
	// depends_on). Status changes go through OpChangeStatus so terminal
	// writes get caught in one place.
	OpUpdateItem Op = "update_item"

	// OpChangeStatus transitions a non-terminal status. Terminal statuses
	// (completed/failed/needs_followup) are rejected — those require
	// review-decide. Lifecycle-controlled statuses (queued/in_progress/
	// in_review/review_pending) are also rejected.
	OpChangeStatus Op = "change_status"

	// OpChangePriority sets priority to the provided integer in [1,10].
	OpChangePriority Op = "change_priority"

	// OpAddEdge adds `From` depends_on `To`. Both endpoints must be
	// members of the initiative's current graph.
	OpAddEdge Op = "add_edge"

	// OpRemoveEdge removes `From` depends_on `To`.
	OpRemoveEdge Op = "remove_edge"

	// OpMoveInitiative transfers an item from its current initiative to
	// `Initiative` (the destination). Empty Initiative detaches the item.
	OpMoveInitiative Op = "move_initiative"

	// OpArchiveItem sets the item's ArchivedAt timestamp. Irreversible from
	// the proposal API — use the existing unarchive endpoint to restore.
	OpArchiveItem Op = "archive_item"

	// OpInterruptInProgress cancels the item's active execution (if any).
	// Must be emitted as a separate mutation so the user has an explicit
	// decision point before an agent races with live work.
	OpInterruptInProgress Op = "interrupt_in_progress"

	// OpSplitItem creates N new items from Into and archives the source
	// atomically — if any child creation or the source archive fails,
	// already-created children are archived to restore pre-split state.
	// Dependents of the source are NOT automatically retargeted;
	// callers that need to repoint edges must emit OpRemoveEdge /
	// OpAddEdge mutations in the same proposal.
	OpSplitItem Op = "split_item"

	// OpMergeItems collapses M source items (Sources) into a single new
	// item (Item). External edges to/from sources are auto-retargeted to
	// the merged item; intra-source edges are dropped. Sources are
	// archived after the merged item lands. Validate rejects strictly
	// when any source is in_progress — callers must emit
	// OpInterruptInProgress as a separate prior mutation if they want
	// to interrupt before merging.
	//
	// Wire shape: { sources: ["kind/a","kind/b",...], item: ItemSpec }.
	// Target is unused; the merged item's ref is Item.Ref().
	OpMergeItems Op = "merge_items"

	// OpRecreateItem archives a stale item and creates a lineage-preserving
	// replacement. The replacement is deliberately service-owned so direct
	// operator actions and approved proposals share the same rollback path.
	OpRecreateItem Op = "recreate_item"

	// OpResetArtifacts removes only the selected derived artifacts from an
	// item. It never deletes the canonical item specification.
	OpResetArtifacts Op = "reset_artifacts"

	// OpRecreateInitiative archives an initiative and creates a fresh active
	// successor while moving its member items to that successor.
	OpRecreateInitiative Op = "recreate_initiative"
)

// AllOps returns the canonical list of supported ops. Used for validation
// and for rendering "supported ops" to the agent prompt.
func AllOps() []Op {
	return []Op{
		OpAddItem,
		OpUpdateItem,
		OpChangeStatus,
		OpChangePriority,
		OpAddEdge,
		OpRemoveEdge,
		OpMoveInitiative,
		OpArchiveItem,
		OpInterruptInProgress,
		OpSplitItem,
		OpMergeItems,
		OpRecreateItem,
		OpResetArtifacts,
		OpRecreateInitiative,
	}
}

// Proposal is the envelope an agent submits. Exactly one of Mutations /
// Graph is populated, keyed by Form.
type Proposal struct {
	Form      Form       `json:"form"`
	Mutations []Mutation `json:"mutations,omitempty"`
	Graph     *Graph     `json:"graph,omitempty"`
	Rationale string     `json:"rationale,omitempty"`
}

// Mutation is a single change. Only the fields relevant to Op are populated;
// Validate enforces which fields are required per op.
type Mutation struct {
	ID        string `json:"id"`
	Op        Op     `json:"op"`
	Rationale string `json:"rationale,omitempty"`

	// Common: "kind/name" reference on the target item for ops that mutate
	// or reference an existing item. Required by all ops except OpAddItem.
	Target string `json:"target,omitempty"`

	// OpAddItem + OpSplitItem.Into payload: full item spec.
	Item *ItemSpec `json:"item,omitempty"`

	// OpUpdateItem payload.
	Patch *ItemPatch `json:"patch,omitempty"`

	// OpChangeStatus payload.
	Status string `json:"status,omitempty"`

	// OpChangePriority payload (pointer so "0" is distinguishable from
	// unset when agents round-trip JSON without omitempty awareness).
	Priority *int `json:"priority,omitempty"`

	// OpAddEdge / OpRemoveEdge payload: dependency edge endpoints. Target
	// mirrors `From` for apply convenience; callers may populate either.
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`

	// OpMoveInitiative destination initiative name. Empty detaches.
	Initiative string `json:"initiative,omitempty"`

	// OpSplitItem payload: the new items to create.
	Into []ItemSpec `json:"into,omitempty"`

	// OpMergeItems payload: the source items ("kind/name" refs) to be
	// collapsed into a single merged item. The merged item itself is
	// described by Item (reusing the OpAddItem field).
	Sources []string `json:"sources,omitempty"`

	// OpResetArtifacts payload: the independently-removable artifact scopes.
	ResetScope []ResetArtifactScope `json:"reset_scope,omitempty"`
}

// ResetArtifactScope identifies a derived-artifact group that may be removed
// without deleting the backlog item's canonical specification.
type ResetArtifactScope string

const (
	ResetScopeWorkshop          ResetArtifactScope = "workshop"
	ResetScopeClarifications    ResetArtifactScope = "clarifications"
	ResetScopeReview            ResetArtifactScope = "review"
	ResetScopeHandoffExecutions ResetArtifactScope = "handoff_executions"
	ResetScopePlanUnbind        ResetArtifactScope = "plan_unbind"
)

func AllResetArtifactScopes() []ResetArtifactScope {
	return []ResetArtifactScope{
		ResetScopeWorkshop,
		ResetScopeClarifications,
		ResetScopeReview,
		ResetScopeHandoffExecutions,
		ResetScopePlanUnbind,
	}
}

// ItemSpec describes a new item to create. Fields mirror BacklogItem with
// the input-safe subset: lifecycle fields (Status, Created, Updated) are
// set by the apply layer.
type ItemSpec struct {
	Kind            string   `json:"kind"`
	Name            string   `json:"name"`
	Title           string   `json:"title"`
	Description     string   `json:"description,omitempty"`
	Priority        int      `json:"priority,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	DependsOn       []string `json:"depends_on,omitempty"`
	Effort          string   `json:"effort,omitempty"`
	Initiative      string   `json:"initiative,omitempty"`
	AcceptanceAllow []string `json:"acceptance_allow,omitempty"`
	AcceptanceDeny  []string `json:"acceptance_deny,omitempty"`
	Note            string   `json:"note,omitempty"`
	SpawnedFrom     string   `json:"spawned_from,omitempty"`
}

// ItemPatch is the set of fields an OpUpdateItem may change. Each pointer is
// "set or leave alone"; explicit clears use empty values (not nil).
type ItemPatch struct {
	Title           *string   `json:"title,omitempty"`
	Description     *string   `json:"description,omitempty"`
	Priority        *int      `json:"priority,omitempty"`
	Tags            *[]string `json:"tags,omitempty"`
	DependsOn       *[]string `json:"depends_on,omitempty"`
	Effort          *string   `json:"effort,omitempty"`
	AcceptanceAllow *[]string `json:"acceptance_allow,omitempty"`
	AcceptanceDeny  *[]string `json:"acceptance_deny,omitempty"`
	Note            *string   `json:"note,omitempty"`
}

// Graph is the full-graph target state. Only the nodes/edges relevant to
// the initiative are represented — cross-initiative relationships are out
// of scope for proposals.
type Graph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// GraphNode describes a target item state. For existing items, Status is
// ignored (status changes require OpChangeStatus / review-decide); for new
// items, Status is also ignored — all new items enter as StatusBacklog.
type GraphNode struct {
	ID          string   `json:"id"` // canonical "kind/name"
	Kind        string   `json:"kind"`
	Name        string   `json:"name"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Priority    int      `json:"priority,omitempty"`
	Effort      string   `json:"effort,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// GraphEdge describes a dependency edge. Kind defaults to "depends_on" when
// omitted; no other kinds are supported today.
type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind,omitempty"`
}

// Source carries attribution metadata for every apply. Feedback rounds
// (W4) populate FeedbackRoundID + InitiativeName + RoundNumber/RoundSlug;
// review rounds (W5) populate ReviewRoundID; operating-mode proposals
// also populate Mode, Phase, and RunID. At least InitiativeName is
// required. Entrypoint identifies the originating surface so downstream
// telemetry (event log, agentactivity) can group by code path
// ("session.proposal", "initiative.review", etc.).
type Source struct {
	InitiativeName  string
	Mode            string
	Phase           string
	FeedbackRoundID string
	RoundNumber     int
	RoundSlug       string
	RunID           string
	// SessionID attributes proposal applications emitted from an agent session.
	// It is independent of legacy feedback-round provenance during cutover.
	SessionID        string
	Entrypoint       string
	ReviewRoundID    string
	DecidedBy        string
	DecidedAtRFC3339 string
}

// Ref is a convenience "kind/name" that parses an ItemSpec.
func (s ItemSpec) Ref() string {
	return s.Kind + "/" + s.Name
}

// String gives a stable debug representation of a mutation.
func (m Mutation) String() string {
	return fmt.Sprintf("mutation{id=%s op=%s target=%s}", m.ID, m.Op, m.Target)
}

// UnmarshalJSON enforces form-specific payload presence at decode time so
// bogus envelopes (e.g., form=full_graph without graph) are rejected before
// validation. This is cheap insurance against agents producing half-shaped
// JSON that would otherwise only fail much later.
func (p *Proposal) UnmarshalJSON(data []byte) error {
	type alias Proposal
	var raw alias
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*p = Proposal(raw)
	switch p.Form {
	case FormMutationList:
		if p.Graph != nil {
			return fmt.Errorf("proposal form=%s must not include graph", FormMutationList)
		}
	case FormFullGraph:
		if p.Graph == nil {
			return fmt.Errorf("proposal form=%s must include graph", FormFullGraph)
		}
		if len(p.Mutations) > 0 {
			return fmt.Errorf("proposal form=%s must not include mutations", FormFullGraph)
		}
	case "":
		return fmt.Errorf("proposal form is required (got empty); expected %q or %q", FormMutationList, FormFullGraph)
	default:
		return fmt.Errorf("proposal form %q is not recognized; expected %q or %q", p.Form, FormMutationList, FormFullGraph)
	}
	return nil
}
