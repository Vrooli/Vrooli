/**
 * Named, shared predicates that narrow a starter card's backing set to the
 * *actionable* subset the card's wording promises (e.g. "a failed or stale run").
 *
 * The same predicate is consumed by two call sites so the card's count badge and
 * the picker's list can never disagree:
 *   - `useStarterContextCounts` (the badge) — via {@link countForStarterCard}
 *   - `SessionContextPicker` (the list)     — via {@link executionIsFailedOrStale}
 *
 * If a card has no filter key, its count is simply the full type list length.
 */
import type { AgentSessionContextType, ExecutionRecord } from "../../../types";
import type { SessionContextOption } from "./session-context-refs";

export type StarterContextFilterKey = "execution_failed_or_stale";

/** Context type each filter key narrows. Lets the picker apply a filter to the right tab. */
export const STARTER_FILTER_TARGET_TYPE: Record<StarterContextFilterKey, AgentSessionContextType> = {
  execution_failed_or_stale: "execution",
};

/** Terminal execution states that count as "failed" for recovery purposes. */
const FAILED_STATUSES = new Set(["failed", "canceled"]);

/** Statuses considered terminal (a healthy completion or a clean stop) — never "stale". */
const TERMINAL_STATUSES = new Set(["completed", "failed", "canceled"]);

/**
 * A non-terminal execution that hasn't advanced in this long is treated as
 * stuck/stale and surfaced for recovery.
 */
export const EXECUTION_STALE_MS = 30 * 60 * 1000;

/**
 * "Failed or stale run that may need recovery": a failed/canceled execution, or a
 * non-terminal execution whose last update is older than {@link EXECUTION_STALE_MS}.
 * `now` is injectable for deterministic tests.
 */
export function executionIsFailedOrStale(execution: ExecutionRecord, now: number = Date.now()): boolean {
  if (FAILED_STATUSES.has(execution.status)) return true;
  if (TERMINAL_STATUSES.has(execution.status)) return false;
  const lastTouch = execution.updatedAt || execution.startedAt || execution.createdAt;
  if (!lastTouch) return false;
  const age = now - new Date(lastTouch).getTime();
  return Number.isFinite(age) && age >= EXECUTION_STALE_MS;
}

/**
 * Count for a single starter card. When the card declares a filter key, the count
 * narrows to the actionable subset; otherwise it is the full type list length.
 * Pure — the picker mirrors this exact logic so badge === picker contents.
 */
export function countForStarterCard(params: {
  optionsByType: Record<AgentSessionContextType, SessionContextOption[]>;
  executions: ExecutionRecord[];
  type: AgentSessionContextType;
  filterKey?: StarterContextFilterKey;
  now?: number;
}): number {
  const { optionsByType, executions, type, filterKey, now } = params;
  if (filterKey === "execution_failed_or_stale") {
    return executions.filter((execution) => executionIsFailedOrStale(execution, now)).length;
  }
  return optionsByType[type]?.length ?? 0;
}
