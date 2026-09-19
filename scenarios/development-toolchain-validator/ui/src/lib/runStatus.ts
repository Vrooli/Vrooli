import { strings } from "../consts/strings";
import { ValidationRunStatus } from "../api/validationRun";
import { Verdict } from "../api/validationRecord";

/**
 * Maps a ValidationRun operational status to its translation key path.
 * Returns a `strings.runs.status.*` leaf for `t()`; unknown/unspecified
 * statuses fall back to the "unknown" copy rather than rendering a raw
 * enum number.
 */
export function runStatusLabelKey(status: ValidationRunStatus) {
  switch (status) {
    case ValidationRunStatus.QUEUED:
      return strings.runs.status.queued;
    case ValidationRunStatus.RUNNING:
      return strings.runs.status.running;
    case ValidationRunStatus.EVALUATING:
      return strings.runs.status.evaluating;
    case ValidationRunStatus.TERMINAL:
      return strings.runs.status.terminal;
    case ValidationRunStatus.UNSPECIFIED:
    default:
      return strings.runs.status.unknown;
  }
}

/**
 * Maps a terminal Verdict to its translation key path. A run that has
 * not reached a terminal verdict (VERDICT_UNSPECIFIED) renders the
 * "pending" copy.
 */
export function runVerdictLabelKey(verdict: Verdict) {
  switch (verdict) {
    case Verdict.PASS:
      return strings.runs.verdict.pass;
    case Verdict.UNEXPECTED_MUTATION:
      return strings.runs.verdict.unexpected;
    case Verdict.RUN_FAILURE:
      return strings.runs.verdict.runFailure;
    case Verdict.TOOL_FAILURE:
      return strings.runs.verdict.toolFailure;
    case Verdict.UNSPECIFIED:
    default:
      return strings.runs.verdict.pending;
  }
}
