import type { ExecutionRecord, Finalization, ReviewResult } from "../types";

/**
 * Synthetic finalization used when an execution is in "validating" status
 * but the real finalization data hasn't arrived yet.
 */
const VALIDATING_PLACEHOLDER_FINALIZATION: Finalization = {
  eligible: true,
  status: "running",
  phase: "scope_detection",
  scopeSource: "none",
  warnings: [],
  affectedScenarios: [],
  aggregateClassification: "not_assessable",
  scenarios: [],
};

/**
 * Returns an execution suitable for PostRunStatusBadge rendering:
 * - If finalization data exists, returns execution as-is.
 * - If status is "validating" but no finalization yet, returns execution
 *   with a synthetic placeholder finalization.
 * - Otherwise returns null (no post-run badge should be shown).
 */
export function resolvePostRunExecution(execution: ExecutionRecord): ExecutionRecord | null {
  if (execution.finalization) {
    return execution;
  }
  if (execution.status === "validating") {
    return { ...execution, finalization: VALIDATING_PLACEHOLDER_FINALIZATION };
  }
  return null;
}

export function getExecutionReviewResults(execution: ExecutionRecord): ReviewResult[] {
  return execution.finalization?.scenarios.flatMap((scenario) => scenario.review.result ? [scenario.review.result] : []) ?? [];
}

export function hasActionableFinalizationIssues(execution: ExecutionRecord): boolean {
  const finalization = execution.finalization;
  if (!finalization) {
    return false;
  }
  if (finalization.status === "failed") {
    return true;
  }
  return finalization.aggregateClassification === "needs_work" || finalization.aggregateClassification === "not_assessable";
}

export function canRunPostRunChecks(execution: ExecutionRecord): boolean {
  if (execution.status === "validating") {
    return false;
  }
  return execution.status === "completed" || execution.status === "failed" || execution.status === "needs_fixup";
}

export function buildFinalizationContext(finalization?: Finalization): string {
  if (!finalization) {
    return "";
  }
  const lines: string[] = [];
  if (finalization.aggregateSummary) {
    lines.push(finalization.aggregateSummary);
  }
  for (const warning of finalization.warnings) {
    const scope = warning.scenarioName ? `${warning.scenarioName}: ` : "";
    lines.push(`- warning [${warning.code}] ${scope}${warning.message}`);
  }
  for (const scenario of finalization.scenarios) {
    if (scenario.restart.status !== "completed" && scenario.restart.lastError) {
      lines.push(`- ${scenario.scenarioName} restart: ${scenario.restart.lastError}`);
    }
    if (scenario.health.status !== "completed" && scenario.health.details) {
      lines.push(`- ${scenario.scenarioName} health: ${scenario.health.details}`);
    }
    if (scenario.review.skipReason) {
      lines.push(`- ${scenario.scenarioName} review: ${scenario.review.skipReason}`);
    }
    if (scenario.review.result) {
      if (scenario.review.result.summary) {
        lines.push(`- ${scenario.scenarioName} review summary: ${scenario.review.result.summary}`);
      }
      for (const dimension of scenario.review.result.dimensions) {
        if (dimension.status === "green" || dimension.status === "skipped") {
          continue;
        }
        lines.push(`- ${scenario.scenarioName} ${dimension.name} (${dimension.status})${dimension.details ? `: ${dimension.details}` : ""}`);
      }
    }
  }
  return lines.join("\n");
}
