/**
 * Readiness computation for backlog items.
 *
 * Computes a 5-dimension readiness indicator from MaturityItemSummary data.
 * Replaces the old 3-phase maturity model (clarify/suggest/enhance).
 *
 * DOC: docs/concepts/ARCHITECTURE.md#workshop-refinement
 */

import type { BacklogKind, MaturityItemSummary, ReadinessDimension } from "../types/domain";

export const READINESS_DIMENSIONS: ReadinessDimension[] = [
  "problem_clarity",
  "scope_defined",
  "approach_solid",
  "testable",
  "risk_awareness",
];

export const DIMENSION_LABELS: Record<ReadinessDimension, string> = {
  problem_clarity: "Problem Clarity",
  scope_defined: "Scope",
  approach_solid: "Approach",
  testable: "Testability",
  risk_awareness: "Risk Awareness",
};

/** Research-specific dimension labels — reinterprets the same dimensions for research items. */
const RESEARCH_DIMENSION_LABELS: Record<ReadinessDimension, string> = {
  problem_clarity: "Question Clarity",
  scope_defined: "Scope",
  approach_solid: "Methodology",
  testable: "Verifiable",
  risk_awareness: "Risk Awareness",
};

/**
 * Returns the appropriate dimension label for a given kind.
 * Research items get research-specific labels; all others use the defaults.
 * Extensible: add new kind overrides by adding a new labels record.
 */
export function getDimensionLabel(dim: ReadinessDimension, kind?: BacklogKind): string {
  if (kind === "research") {
    return RESEARCH_DIMENSION_LABELS[dim];
  }
  return DIMENSION_LABELS[dim];
}

export const DIMENSION_SHORT_LABELS: Record<ReadinessDimension, string> = {
  problem_clarity: "P",
  scope_defined: "S",
  approach_solid: "A",
  testable: "T",
  risk_awareness: "R",
};

export const SCORE_COLORS: Record<number, string> = {
  0: "slate",
  1: "rose",
  2: "amber",
  3: "emerald",
};

export interface ReadinessIndicatorData {
  rawScores: Record<ReadinessDimension, number>;
  effectiveScores: Record<ReadinessDimension, number>;
  roundsCompleted: number;
  ready: boolean;
  pendingItems: number;
  pendingSynthesis: boolean;
  hasPlan: boolean;
  nextNudge: string | null;
}

export function buildReadinessData(summary: MaturityItemSummary): ReadinessIndicatorData {
  return {
    rawScores: summary.raw_scores,
    effectiveScores: summary.effective_scores,
    roundsCompleted: summary.rounds_completed,
    ready: summary.ready,
    pendingItems: summary.pending_items,
    pendingSynthesis: summary.pending_synthesis ?? false,
    hasPlan: summary.has_plan,
    nextNudge: computeNextNudge({
      rawScores: summary.raw_scores,
      effectiveScores: summary.effective_scores,
      roundsCompleted: summary.rounds_completed,
      ready: summary.ready,
      pendingItems: summary.pending_items,
      pendingSynthesis: summary.pending_synthesis ?? false,
      hasPlan: summary.has_plan,
      nextNudge: null,
    }),
  };
}

export function computeNextNudge(data: ReadinessIndicatorData): string | null {
  if (data.roundsCompleted === 0) {
    return "Run Workshop to start refining this item";
  }

  if (data.pendingItems > 0) {
    return `Respond to ${data.pendingItems} pending item${data.pendingItems === 1 ? "" : "s"} from the latest workshop round`;
  }

  if (data.pendingSynthesis) {
    if (data.ready) {
      return "Finalize the latest workshop answers into the deliverable";
    }
    return "Run another Workshop round to incorporate the latest answers";
  }

  if (data.ready) {
    return "Ready for execution — review the plan and queue when satisfied";
  }

  // Find the weakest dimension
  const weakDims = READINESS_DIMENSIONS.filter(
    (dim) => data.effectiveScores[dim] < 3,
  );
  if (weakDims.length > 0) {
    const labels = weakDims.map((d) => DIMENSION_LABELS[d]).join(", ");
    return `Run another Workshop round to strengthen: ${labels}`;
  }

  return null;
}
