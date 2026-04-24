/**
 * Initiative Feedback Types
 *
 * Wire-level shapes for the feedback-round surface. Mirrors the Go types in
 * `api/internal/feedback/types.go` and `api/internal/proposals/types.go`.
 *
 * The server emits snake_case JSON (proto-style tags), so these interfaces
 * use snake_case for fidelity with the raw payload. Services and components
 * read/write them directly — no normalization layer needed because there is
 * no proto generation for feedback types.
 */

// ---------------------------------------------------------------------------
// Round lifecycle
// ---------------------------------------------------------------------------

/** User-facing kind of feedback surface. `research` is disabled in UI today. */
export type FeedbackRoundType = "feedback" | "research" | "note";

/**
 * Where a round is in its lifecycle. Terminal states:
 * `applied` (mutations accepted), `rejected`, `dismissed`.
 */
export type FeedbackRoundStatus =
  | "submitting"
  | "agent_thinking"
  | "awaiting_user"
  | "applied"
  | "rejected"
  | "dismissed";

/**
 * Terminal decision shapes the user can pick. `revise` is not terminal —
 * it's handled via the continue endpoint — but the backend recognizes the
 * discriminator so we keep the type exhaustive.
 */
export type FeedbackDecisionKind =
  | "accept"
  | "partial_accept"
  | "reject"
  | "revise"
  | "dismiss";

export const FEEDBACK_TERMINAL_STATUSES: ReadonlySet<FeedbackRoundStatus> = new Set([
  "applied",
  "rejected",
  "dismissed",
]);

export function isFeedbackRoundTerminal(status: FeedbackRoundStatus): boolean {
  return FEEDBACK_TERMINAL_STATUSES.has(status);
}

// ---------------------------------------------------------------------------
// Proposal wire shapes (mirror `internal/proposals`)
// ---------------------------------------------------------------------------

/** Envelope form: flat list of mutations, or a target-state graph. */
export type ProposalForm = "mutation_list" | "full_graph";

/** The supported mutation ops. Keep in sync with `proposals.AllOps()`. */
export type ProposalOp =
  | "add_item"
  | "update_item"
  | "change_status"
  | "change_priority"
  | "add_edge"
  | "remove_edge"
  | "move_initiative"
  | "archive_item"
  | "interrupt_in_progress"
  | "split_item";

/** A prospective new item's authorable fields. */
export interface ProposalItemSpec {
  kind: string;
  name: string;
  title: string;
  description?: string;
  priority?: number;
  tags?: string[];
  depends_on?: string[];
  effort?: string;
  initiative?: string;
  acceptance_allow?: string[];
  acceptance_deny?: string[];
  note?: string;
}

/** Partial update payload for `update_item`. */
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

/** A single change in a proposal. Only fields relevant to `op` are populated. */
export interface ProposalMutation {
  id: string;
  op: ProposalOp;
  rationale?: string;
  target?: string;
  item?: ProposalItemSpec;
  patch?: ProposalItemPatch;
  status?: string;
  priority?: number;
  from?: string;
  to?: string;
  initiative?: string;
  into?: ProposalItemSpec[];
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

/** Agent-authored proposal envelope. Exactly one of `mutations` / `graph` is set. */
export interface Proposal {
  form: ProposalForm;
  mutations?: ProposalMutation[];
  graph?: ProposalGraph;
  rationale?: string;
}

/**
 * One revision of a proposal, attached to the round. Each agent turn that
 * produces a new proposal becomes a new revision — the user can compare them.
 */
export interface ProposalRevision {
  id: string;
  message_index: number;
  proposal: Proposal;
  rationale?: string;
  created_at: string;
  parse_warnings?: string[];
}

// ---------------------------------------------------------------------------
// Round contents
// ---------------------------------------------------------------------------

export interface FeedbackSubmission {
  text: string;
  attachment_ids?: string[];
  created_at: string;
}

export interface FeedbackMessage {
  role: "user" | "agent";
  content: string;
  attachment_ids?: string[];
  proposal_id?: string;
  run_id?: string;
  created_at: string;
}

export interface FeedbackDecision {
  kind: FeedbackDecisionKind;
  accepted_mutation_ids?: string[];
  rejected_mutation_ids?: string[];
  rationale?: string;
  decided_at: string;
  decided_by?: string;
}

export interface FeedbackRound {
  initiative_name: string;
  number: number;
  slug: string;
  type: FeedbackRoundType;
  status: FeedbackRoundStatus;
  submission: FeedbackSubmission;
  thread?: FeedbackMessage[];
  proposals?: ProposalRevision[];
  current_proposal_id?: string;
  decision?: FeedbackDecision;
  run_id?: string;
  /**
   * True when the most recent agent turn produced no extractable proposal.
   * Consumers show the "ask for a revision" CTA rather than an empty proposal
   * pane. Cleared on the next successful agent turn.
   */
  needs_revision?: boolean;
  /** Parser complaints from the most recent agent turn — surfaced alongside
   *  `needs_revision` so the user sees *why* the proposal didn't parse. */
  last_parse_warnings?: string[];
  created_at: string;
  updated_at: string;
}

// ---------------------------------------------------------------------------
// Apply result (decide response)
// ---------------------------------------------------------------------------

/**
 * Per-mutation result returned by the decide endpoint. Exactly one of
 * `applied`, `skipped`, or failure (`applied=false` with `error` set) is
 * true. `skipped` means the user deselected the mutation before apply —
 * renderers must distinguish it from failure so users don't confuse a
 * deliberate deselection with a broken apply.
 */
export interface ApplyOutcome {
  mutation_id: string;
  op: ProposalOp;
  target?: string;
  applied: boolean;
  skipped?: boolean;
  error?: string;
}

export interface ApplyResult {
  outcomes: ApplyOutcome[];
  applied: number;
  failed: number;
  skipped: number;
}

/** Response payload from `POST .../decide`. */
export interface FeedbackDecideResponse {
  round: FeedbackRound;
  apply_result?: ApplyResult;
}

// ---------------------------------------------------------------------------
// Lock status
// ---------------------------------------------------------------------------

export interface LockHolder {
  run_id: string;
  purpose: string;
  acquired_at?: string;
  acquired_by?: string;
  round_number?: number;
}

export interface LockStatusResponse {
  locked: boolean;
  holder?: LockHolder;
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Returns the current proposal attached to the round, if any. */
export function currentProposal(round: FeedbackRound): ProposalRevision | undefined {
  if (!round.current_proposal_id || !round.proposals) return undefined;
  return round.proposals.find((p) => p.id === round.current_proposal_id);
}

/** Returns every mutation in the current proposal, or empty if not a mutation_list. */
export function currentMutations(round: FeedbackRound): ProposalMutation[] {
  const rev = currentProposal(round);
  if (!rev || rev.proposal.form !== "mutation_list") return [];
  return rev.proposal.mutations ?? [];
}
