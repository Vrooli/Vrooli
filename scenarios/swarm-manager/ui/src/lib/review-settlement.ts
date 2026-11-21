/**
 * Rolls per-criterion evidence up into one settlement verdict.
 *
 * The review card previously showed an amber "this item has no typed criteria"
 * warning whenever the `criteria` prop was empty — which, in the decision
 * stream, was always, because the call site never passed it. The warning was
 * therefore a fact about the code rather than about the work. With criteria
 * actually supplied, the useful warning is the one below: how many criteria
 * are still unsettled at the moment you are about to accept.
 */
export type SettlementState = "settled" | "refuted" | "pending" | "unsettled";

export interface CriterionEvidenceLike {
  criterion_id?: string;
  settlement?: "settled" | "refuted" | "unavailable" | "pending";
}

export interface CriterionLike {
  id: string;
  gherkin?: string;
}

export interface CriterionSettlement<C extends CriterionLike> {
  criterion: C;
  evidenceCount: number;
  state: SettlementState;
}

/**
 * A criterion is settled only when it has evidence and none of it refutes or
 * is still pending. Absence of evidence is `unsettled`, never `settled` — the
 * optimistic reading is how an item ships with an unproven acceptance claim.
 */
export function settlementFor(evidence: readonly CriterionEvidenceLike[]): SettlementState {
  if (evidence.length === 0) return "unsettled";
  if (evidence.some((item) => item.settlement === "refuted")) return "refuted";
  if (evidence.every((item) => item.settlement === "settled")) return "settled";
  return "pending";
}

export function summarizeCriteria<C extends CriterionLike>(
  criteria: readonly C[],
  evidence: readonly CriterionEvidenceLike[],
): { rows: CriterionSettlement<C>[]; unsettled: number } {
  const rows = criteria.map((criterion) => {
    const related = evidence.filter((item) => item.criterion_id === criterion.id);
    return { criterion, evidenceCount: related.length, state: settlementFor(related) };
  });
  return { rows, unsettled: rows.filter((row) => row.state !== "settled").length };
}
