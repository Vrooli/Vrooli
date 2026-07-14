/** Shared mutation-list proposal envelope used by sessions and operating modes. */
export type ProposalForm = "mutation_list" | "full_graph";
export type ProposalOp = "add_item" | "update_item" | "change_status" | "change_priority" | "add_edge" | "remove_edge" | "move_initiative" | "archive_item" | "interrupt_in_progress" | "split_item" | "merge_items";
export interface ProposalItemSpec { kind: string; name: string; title: string; description?: string; priority?: number; tags?: string[]; depends_on?: string[]; effort?: string; initiative?: string; acceptance_allow?: string[]; acceptance_deny?: string[]; note?: string; }
export interface ProposalItemPatch { title?: string; description?: string; priority?: number; tags?: string[]; depends_on?: string[]; effort?: string; acceptance_allow?: string[]; acceptance_deny?: string[]; note?: string; }
export interface ProposalMutation { id: string; op: ProposalOp; rationale?: string; target?: string; item?: ProposalItemSpec; patch?: ProposalItemPatch; status?: string; priority?: number; from?: string; to?: string; initiative?: string; into?: ProposalItemSpec[]; sources?: string[]; }
export interface ProposalGraphNode { id: string; kind: string; name: string; title: string; description?: string; priority?: number; effort?: string; tags?: string[]; }
export interface ProposalGraphEdge { from: string; to: string; kind?: string; }
export interface ProposalGraph { nodes: ProposalGraphNode[]; edges: ProposalGraphEdge[]; }
export interface Proposal { form: ProposalForm; mutations?: ProposalMutation[]; graph?: ProposalGraph; rationale?: string; }
