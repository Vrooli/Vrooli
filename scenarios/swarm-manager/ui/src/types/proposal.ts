/**
 * Shared mutation-list proposal envelope used by sessions and operating modes.
 *
 * This file is the UI's mirror of `api/internal/proposals/types.go`. Every op in
 * the server's `AllOps()` must appear in {@link ProposalOp}, and every payload
 * field the server may populate must be representable here — a field missing
 * from these types cannot be rendered, which is how operators ended up
 * approving mutations whose contents were never displayed.
 *
 * `api/internal/proposals/uicontract_test.go` fails the build when the server
 * gains an op that this union does not carry.
 */
export type ProposalForm = "mutation_list" | "full_graph";

/** Ops that act on a backlog item. Mirrors the item half of Go's AllOps(). */
export type ProposalItemOp =
  | "add_item"
  | "update_item"
  | "change_status"
  | "change_priority"
  | "add_edge"
  | "remove_edge"
  | "move_milestone"
  | "archive_item"
  | "interrupt_in_progress"
  | "split_item"
  | "merge_items"
  | "recreate_item"
  | "reset_artifacts"
  | "recreate_milestone";

/** Ops valid only when the proposal target is a goal. Mirrors Go's GoalOps(). */
export type ProposalGoalOp =
  | "create_goal"
  | "create_milestone"
  | "update_milestone"
  | "archive_milestone"
  | "assign_milestone_items"
  | "unassign_milestone_items"
  | "add_goal_target"
  | "remove_goal_target";

export type ProposalOp = ProposalItemOp | ProposalGoalOp;

export type ResetArtifactScope = "review" | "handoff_executions" | "plan_unbind";

/** Declares whether Swarm may apply directly or must batch into reconciliation. */
export type ProposalApplyMode = "direct" | "reconciliation" | "attention";

export interface ProposalItemSpec {
  kind: string;
  name: string;
  title: string;
  description?: string;
  priority?: number;
  tags?: string[];
  depends_on?: string[];
  effort?: string;
  milestone?: string;
  acceptance_allow?: string[];
  acceptance_deny?: string[];
  note?: string;
  spawned_from?: string;
}

/**
 * The fields an `update_item` may change. Each key is "set or leave alone":
 * an absent key means untouched, while a present key holding an empty value is
 * an explicit clear. Renderers must distinguish the two — they are different
 * operator intents that look identical if only the presence of text is shown.
 */
export interface ProposalItemPatch {
  title?: string;
  description?: string;
  priority?: number;
  tags?: string[];
  depends_on?: string[];
  effort?: string;
  acceptance_allow?: string[];
  acceptance_deny?: string[];
  note?: string;
}

export interface ProposalGoalMilestone {
  items?: string[];
  name: string;
  title: string;
  description?: string;
  acceptance_criteria?: string[];
  depends_on?: string[];
  spawned_from?: string;
}

export interface ProposalGoalSpec {
  name: string;
  title: string;
  description?: string;
  priority?: number;
  targets?: string[];
  milestones?: ProposalGoalMilestone[];
  spawned_from?: string;
}

export interface ProposalMutation {
  id: string;
  op: ProposalOp;
  rationale?: string;
  /** "kind/name" of an existing item. Absent for add_item and goal creation. */
  target?: string;
  item?: ProposalItemSpec;
  patch?: ProposalItemPatch;
  status?: string;
  priority?: number;
  from?: string;
  to?: string;
  milestone?: string;
  into?: ProposalItemSpec[];
  sources?: string[];
  reset_scope?: ResetArtifactScope[];
  goal_milestone?: ProposalGoalMilestone;
  goal?: ProposalGoalSpec;
  milestone_name?: string;
  items?: string[];
  targets?: string[];
  detach_open?: boolean;
}

export interface ProposalGraphNode {
  id: string;
  kind: string;
  name: string;
  title: string;
  description?: string;
  priority?: number;
  effort?: string;
  tags?: string[];
}

export interface ProposalGraphEdge {
  from: string;
  to: string;
  kind?: string;
}

export interface ProposalGraph {
  nodes: ProposalGraphNode[];
  edges: ProposalGraphEdge[];
}

export interface Proposal {
  form: ProposalForm;
  mutations?: ProposalMutation[];
  graph?: ProposalGraph;
  rationale?: string;
  base_version?: string;
  conflict_key?: string;
  dependencies?: string[];
  apply_mode?: ProposalApplyMode;
  requires_authorization?: boolean;
}

/**
 * Every op the server accepts, in AllOps() order. Kept as a runtime value (not
 * only a type) so tests can assert exhaustive renderer coverage.
 */
export const PROPOSAL_OPS: readonly ProposalOp[] = [
  "add_item",
  "update_item",
  "change_status",
  "change_priority",
  "add_edge",
  "remove_edge",
  "move_milestone",
  "archive_item",
  "interrupt_in_progress",
  "split_item",
  "merge_items",
  "recreate_item",
  "reset_artifacts",
  "recreate_milestone",
  "create_goal",
  "create_milestone",
  "update_milestone",
  "archive_milestone",
  "assign_milestone_items",
  "unassign_milestone_items",
  "add_goal_target",
  "remove_goal_target",
] as const;
