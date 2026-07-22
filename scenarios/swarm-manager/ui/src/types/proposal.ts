/** Shared mutation-list proposal envelope used by sessions and operating modes. */
export type ProposalForm = "mutation_list" | "full_graph";
export type ProposalOp = "add_item" | "update_item" | "change_status" | "change_priority" | "add_edge" | "remove_edge" | "move_milestone" | "archive_item" | "interrupt_in_progress" | "split_item" | "merge_items" | "recreate_item" | "reset_artifacts" | "recreate_milestone";
export type ResetArtifactScope = "workshop" | "clarifications" | "review" | "handoff_executions" | "plan_unbind";
export interface ProposalItemSpec { kind: string; name: string; title: string; description?: string; priority?: number; tags?: string[]; depends_on?: string[]; effort?: string; milestone?: string; acceptance_allow?: string[]; acceptance_deny?: string[]; note?: string; spawned_from?: string; }
export interface ProposalItemPatch { title?: string; description?: string; priority?: number; tags?: string[]; depends_on?: string[]; effort?: string; acceptance_allow?: string[]; acceptance_deny?: string[]; note?: string; }
export interface ProposalMutation { id: string; op: ProposalOp; rationale?: string; target?: string; item?: ProposalItemSpec; patch?: ProposalItemPatch; status?: string; priority?: number; from?: string; to?: string; milestone?: string; into?: ProposalItemSpec[]; sources?: string[]; reset_scope?: ResetArtifactScope[]; }
export interface ProposalGraphNode { id: string; kind: string; name: string; title: string; description?: string; priority?: number; effort?: string; tags?: string[]; }
export interface ProposalGraphEdge { from: string; to: string; kind?: string; }
export interface ProposalGraph { nodes: ProposalGraphNode[]; edges: ProposalGraphEdge[]; }
export interface Proposal { form: ProposalForm; mutations?: ProposalMutation[]; graph?: ProposalGraph; rationale?: string; }
