/**
 * Workshop domain types.
 */

/**
 * Workshop item types within a round.
 */
export type WorkshopItemType = "decision" | "info";

/**
 * The 5 universal readiness dimensions.
 */
export type ReadinessDimension =
  | "problem_clarity"
  | "scope_defined"
  | "approach_solid"
  | "testable"
  | "risk_awareness";

/**
 * A lettered choice within a decision item.
 */
export interface DecisionOption {
  key: string;
  label: string;
  rationale: string;
  recommended?: boolean;
}

/**
 * A single item within a workshop round — either a decision point or informational.
 */
export interface WorkshopItem {
  id: string;
  type: WorkshopItemType;
  topic?: string;
  text?: string;
  context?: string;
  options?: DecisionOption[];
  selected?: string | null;
  freeform?: string | null;
  notes?: string | null;
  context_note?: string;
  clarification_id?: string;
}

/**
 * A single message within a clarification thread.
 */
export interface ClarificationMessage {
  role: "user" | "assistant";
  content: string;
  created_at: string;
  attachment_ids?: string[];
}

/**
 * Impact assessment from a clarification response.
 */
export interface ClarificationImpact {
  level: "none" | "decision" | "round";
  reasoning: string;
  context_note: string;
  suggested_update?: string;
}

/**
 * A multi-turn clarification thread attached to a workshop decision.
 */
export interface ClarificationThread {
  id: string;
  round_number: number;
  item_id: string;
  run_id: string;
  messages: ClarificationMessage[];
  latest_impact?: ClarificationImpact;
  status: "active" | "resolved" | "dismissed";
  created_at: string;
  updated_at: string;
}

/**
 * A single workshop round stored on disk.
 */
export interface WorkshopRound {
  round: number;
  generated_at: string;
  mode?: "workshop" | "finalize";
  pending_synthesis?: boolean;
  readiness: Record<ReadinessDimension, number>;
  items: WorkshopItem[];
  plan_updates?: string;
}
