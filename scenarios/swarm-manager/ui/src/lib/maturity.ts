/**
 * Maturity computation for idea backlog items.
 *
 * Computes a 3-phase maturity indicator (Clarify → Suggest → Enhance)
 * from either a MaturityItemSummary (API response) or locally parsed data.
 */

export type MaturityPhase = "clarify" | "suggest" | "enhance";
export type PhaseState = "empty" | "in-progress" | "complete";

export interface MaturityIndicatorData {
  phases: Record<MaturityPhase, PhaseState>;
  enhanceRound: number;
  needsResynthesis: number;
  nextNudge: string | null;
}

export interface MaturityInput {
  clarifyCount: number;
  suggestCount: number;
  enhanceCount: number;
  questionsTotal: number;
  questionsAnswered: number;
  suggestionsTotal: number;
  suggestionsDecided: number;
  questionsNewOrUpdated: number;
  suggestionsNewOrUpdated: number;
  hasEnhanceSummary: boolean;
}

export function computeMaturity(input: MaturityInput): MaturityIndicatorData {
  const { questionsTotal, questionsAnswered, suggestionsTotal, suggestionsDecided } = input;
  const totalNewOrUpdated = input.questionsNewOrUpdated + input.suggestionsNewOrUpdated;

  // Clarify phase
  let clarify: PhaseState = "empty";
  if (questionsTotal > 0) {
    clarify = questionsAnswered >= questionsTotal ? "complete" : "in-progress";
  }

  // Suggest phase
  let suggest: PhaseState = "empty";
  if (suggestionsTotal > 0) {
    suggest = suggestionsDecided >= suggestionsTotal ? "complete" : "in-progress";
  }

  // Enhance phase
  let enhance: PhaseState = "empty";
  if (input.hasEnhanceSummary) {
    enhance = totalNewOrUpdated > 0 ? "in-progress" : "complete";
  }

  // Next nudge
  let nextNudge: string | null = null;
  if (clarify === "empty" && suggest === "empty" && enhance === "empty") {
    nextNudge = "Run Clarify to start refining this idea";
  } else if (clarify === "in-progress") {
    const remaining = questionsTotal - questionsAnswered;
    nextNudge = `Answer ${remaining} remaining question${remaining === 1 ? "" : "s"}`;
  } else if (suggest === "in-progress") {
    const remaining = suggestionsTotal - suggestionsDecided;
    nextNudge = `Decide on ${remaining} pending suggestion${remaining === 1 ? "" : "s"}`;
  } else if (enhance === "empty" && (clarify === "complete" || suggest === "complete")) {
    nextNudge = "Run Enhance to synthesize into a refined plan";
  } else if (enhance === "in-progress") {
    nextNudge = `Run Enhance to incorporate ${totalNewOrUpdated} change${totalNewOrUpdated === 1 ? "" : "s"}`;
  } else if (clarify === "complete" && suggest === "empty" && enhance === "complete") {
    nextNudge = "Run Suggest to get improvement ideas, or queue if ready";
  }

  return {
    phases: { clarify, suggest, enhance },
    enhanceRound: input.enhanceCount,
    needsResynthesis: totalNewOrUpdated,
    nextNudge,
  };
}

/**
 * Build a MaturityInput from locally parsed data on the details page,
 * avoiding an extra API call.
 */
export function buildMaturityInputFromLocal(opts: {
  clarifyRaw: Record<string, unknown> | null;
  questionsCount: number;
  questionsAnsweredCount: number;
  questionsNewOrUpdated: number;
  suggestionsRaw: Record<string, unknown> | null;
  suggestionsCount: number;
  suggestionsDecidedCount: number;
  suggestionsNewOrUpdated: number;
  hasEnhanceSummary: boolean;
}): MaturityInput {
  const clarifyCount = typeof opts.clarifyRaw?.clarifyCount === "number" ? opts.clarifyRaw.clarifyCount : 0;
  const suggestCount = typeof opts.suggestionsRaw?.suggestCount === "number" ? opts.suggestionsRaw.suggestCount : 0;
  const enhanceCountQ = typeof opts.clarifyRaw?.enhanceCount === "number" ? opts.clarifyRaw.enhanceCount : 0;
  const enhanceCountS = typeof opts.suggestionsRaw?.enhanceCount === "number" ? opts.suggestionsRaw.enhanceCount : 0;

  return {
    clarifyCount,
    suggestCount,
    enhanceCount: Math.max(enhanceCountQ, enhanceCountS),
    questionsTotal: opts.questionsCount,
    questionsAnswered: opts.questionsAnsweredCount,
    suggestionsTotal: opts.suggestionsCount,
    suggestionsDecided: opts.suggestionsDecidedCount,
    questionsNewOrUpdated: opts.questionsNewOrUpdated,
    suggestionsNewOrUpdated: opts.suggestionsNewOrUpdated,
    hasEnhanceSummary: opts.hasEnhanceSummary,
  };
}
