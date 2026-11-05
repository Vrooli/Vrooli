import { ApplyRunState, ApplyStepState, type ApplyStep } from "@vrooli/proto-types/vrooli-onboarding/v1/apply/apply_pb";

/**
 * What a configuration apply run means to the person who pressed Re-apply.
 *
 * The run comes back as a status enum and a list of steps whose failures are
 * raw installer output — JSON log lines, several per step. Rendering that is
 * noise; rendering only the enum ("partially applied") hides which scenarios
 * failed and why. This reduces a run to counts, an outcome, and one readable
 * reason per failed step.
 */
export type ApplyOutcome =
  | "running"
  | "applied"
  | "already-satisfied"
  | "partial"
  | "incomplete"
  | "failed"
  | "cancelled"
  | "indeterminate";

export interface FailedApplyStep {
  id: string;
  name: string;
  reason: string;
  remediation?: string;
}

export interface ApplyRunSummary {
  outcome: ApplyOutcome;
  active: boolean;
  total: number;
  finished: number;
  failed: FailedApplyStep[];
}

const FAILED_STEP_STATES = new Set<ApplyStepState>([
  ApplyStepState.FAILED,
  ApplyStepState.BLOCKED,
  ApplyStepState.TIMED_OUT,
  ApplyStepState.NEEDS_ELEVATION,
]);

const FINISHED_STEP_STATES = new Set<ApplyStepState>([
  ...FAILED_STEP_STATES,
  ApplyStepState.APPLIED,
  ApplyStepState.ALREADY_SATISFIED,
  ApplyStepState.NOT_APPLICABLE,
  ApplyStepState.SKIPPED_SELF,
]);

export function isApplyRunActive(state: ApplyRunState): boolean {
  return state === ApplyRunState.PENDING || state === ApplyRunState.APPLYING || state === ApplyRunState.UNSPECIFIED;
}

function outcomeFor(state: ApplyRunState): ApplyOutcome {
  switch (state) {
    case ApplyRunState.APPLIED:
      return "applied";
    case ApplyRunState.ALREADY_SATISFIED:
      return "already-satisfied";
    case ApplyRunState.PARTIALLY_APPLIED:
      return "partial";
    case ApplyRunState.CONFIGURATION_INCOMPLETE:
      return "incomplete";
    case ApplyRunState.FAILED:
      return "failed";
    case ApplyRunState.CANCELLED:
      return "cancelled";
    case ApplyRunState.INDETERMINATE:
      return "indeterminate";
    default:
      return "running";
  }
}

const REASON_LIMIT = 240;

/**
 * The one sentence worth showing from a step's raw failure output.
 *
 * Installer output ends with the lifecycle's structured error, whose
 * "message" is the actual cause ("start postgres: … not supported on this
 * platform"); everything before it is progress chatter. Plain-text output
 * falls back to its last non-empty line.
 */
export function stepFailureReason(raw: string): string {
  const text = raw.trim();
  if (!text) return "";
  const messages = [...text.matchAll(/"message":"((?:[^"\\]|\\.)*)"/g)];
  let reason = "";
  const last = messages.at(-1)?.[1];
  if (last !== undefined) {
    try {
      reason = JSON.parse(`"${last}"`) as string;
    } catch {
      reason = last;
    }
  } else {
    reason = text.split("\n").map((line) => line.trim()).filter(Boolean).at(-1) ?? "";
  }
  reason = reason.replace(/\s+/g, " ").trim();
  return reason.length > REASON_LIMIT ? `${reason.slice(0, REASON_LIMIT - 1)}…` : reason;
}

export function summarizeApplyRun(state: ApplyRunState, steps: readonly ApplyStep[] = []): ApplyRunSummary {
  const failed = steps
    .filter((step) => FAILED_STEP_STATES.has(step.state))
    .map((step) => ({
      id: step.id,
      name: step.name || step.id,
      reason: stepFailureReason(step.error) || step.errorCode || "the step failed without a reason",
      remediation: step.remediation || undefined,
    }));
  return {
    outcome: outcomeFor(state),
    active: isApplyRunActive(state),
    total: steps.length,
    finished: steps.filter((step) => FINISHED_STEP_STATES.has(step.state)).length,
    failed,
  };
}
