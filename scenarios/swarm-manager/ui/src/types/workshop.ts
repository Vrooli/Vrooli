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
