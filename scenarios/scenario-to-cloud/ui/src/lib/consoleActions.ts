import { ApiError, ErrorCodes } from "./apiErrors";
import { TERMINAL_OPERATION_STATES } from "../types/console";
import type {
  ActionState,
  AuthzMatrix,
  AuthzRoute,
  NextAction,
  OperationStanding,
  PlanOutcome,
  TargetRef,
} from "../types/console";

/**
 * Pure derivations from API state to what the console renders. Nothing in
 * this file decides whether an action is permitted from the deployment
 * status: availability is read from plan.outcome, standing.next_action and
 * typed refusals, and the authz matrix only labels which scope a route needs.
 */

export function targetKey(target: TargetRef | undefined | null): string {
  if (!target) return "unbound";
  if (target.machine_id) return `machine:${target.machine_id}`;
  if (target.locator?.host) return `host:${target.locator.host}`;
  return "unbound";
}

export function shortDigest(digest: string | undefined | null, length = 12): string {
  if (!digest) return "";
  const hex = digest.includes(":") ? digest.slice(digest.indexOf(":") + 1) : digest;
  return hex.length > length ? hex.slice(0, length) : hex;
}

/** Human age of an RFC3339 timestamp relative to now; empty when unparseable. */
export function formatAge(timestamp: string | undefined | null, now: number = Date.now()): string {
  if (!timestamp) return "";
  const t = new Date(timestamp).getTime();
  if (Number.isNaN(t)) return "";
  const seconds = Math.max(0, Math.round((now - t) / 1000));
  if (seconds < 60) return `${seconds}s ago`;
  const minutes = Math.round(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.round(minutes / 60);
  if (hours < 48) return `${hours}h ago`;
  return `${Math.round(hours / 24)}d ago`;
}

export function isTerminalState(state: string | undefined): boolean {
  return (TERMINAL_OPERATION_STATES as readonly string[]).includes(state ?? "");
}

function routeMatches(route: AuthzRoute, method: string, path: string): boolean {
  if (route.method !== method) return false;
  if (route.prefix) return path.startsWith(route.path);
  const pattern = route.path.replace(/\{[^}]+\}/g, "[^/]+");
  return new RegExp(`^${pattern}$`).test(path);
}

/** The scope the authz matrix declares for a route, when the matrix is loaded. */
export function requiredScopeFor(matrix: AuthzMatrix | null | undefined, method: string, path: string): string | undefined {
  if (!matrix) return undefined;
  const route = matrix.routes.find((r) => routeMatches(r, method, path));
  return route?.scope;
}

export type DenialView = {
  code: string;
  title: string;
  explanation: string;
  missingScope?: string;
  target?: string;
  nextAction?: NextAction | null;
  retryable: boolean;
};

/**
 * describeDenial turns a typed authority refusal into the specific
 * permission that is missing. It reads details.required_scope first and
 * falls back to the matrix scope of the refused route.
 */
export function describeDenial(
  error: unknown,
  options: { matrix?: AuthzMatrix | null; method?: string; path?: string; target?: string } = {},
): DenialView | null {
  if (!(error instanceof ApiError)) return null;
  const details = (error.typed.details ?? {}) as Record<string, unknown>;
  const fromMatrix = options.method && options.path ? requiredScopeFor(options.matrix, options.method, options.path) : undefined;
  switch (error.code) {
    case ErrorCodes.unauthenticated:
      return {
        code: error.code,
        title: "Sign in required",
        explanation: "No operator identity was presented for this request.",
        missingScope: fromMatrix,
        nextAction: error.nextAction,
        retryable: error.retryable,
      };
    case ErrorCodes.forbiddenScope: {
      const scope = typeof details.required_scope === "string" ? details.required_scope : fromMatrix;
      return {
        code: error.code,
        title: "Missing permission",
        explanation: scope
          ? `This action needs the scope ${scope}, which your identity does not hold.`
          : "Your identity does not hold the scope this route needs.",
        missingScope: scope,
        nextAction: error.nextAction,
        retryable: error.retryable,
      };
    }
    case ErrorCodes.forbiddenTarget:
      return {
        code: error.code,
        title: "Target not granted",
        explanation: options.target
          ? `Your identity is not granted for target ${options.target}.`
          : "Your identity is not granted for this deployment's target.",
        missingScope: fromMatrix,
        target: options.target,
        nextAction: error.nextAction,
        retryable: error.retryable,
      };
    case "forbidden_origin":
      return {
        code: error.code,
        title: "Origin refused",
        explanation: "The request origin is not allowed to perform effects on this API.",
        nextAction: error.nextAction,
        retryable: error.retryable,
      };
    default:
      return null;
  }
}

/** Which browser-executable action a next_action kind maps to. */
export type NextActionKind = "wait" | "resume" | "reconcile" | "recovery" | "operation" | "resume_handoff" | "replan" | "sign_in" | "other";

export function classifyNextAction(next: NextAction | undefined | null): NextActionKind {
  switch (next?.kind) {
    case "wait":
    case "resume":
    case "reconcile":
    case "recovery":
    case "operation":
    case "resume_handoff":
    case "replan":
    case "sign_in":
      return next.kind;
    default:
      return "other";
  }
}

/** Default operator-facing label for a next_action that carries none. */
export function nextActionLabel(next: NextAction | undefined | null): string {
  if (next?.label) return next.label;
  switch (classifyNextAction(next)) {
    case "wait":
      return "Keep waiting for the operation";
    case "resume":
      return "Resume the operation";
    case "reconcile":
      return "Reconcile the operation owner";
    case "recovery":
      return "Open recovery";
    case "operation":
      return "Inspect the operation";
    case "resume_handoff":
      return "Finish setup in onboarding";
    case "replan":
      return "Review the plan again";
    case "sign_in":
      return "Sign in and retry";
    default:
      return next?.reference ? `See ${next.reference}` : "No next action provided";
  }
}

/** The plan review's apply action, derived only from the compiled outcome. */
export function applyActionFor(outcome: PlanOutcome | string | undefined, requiredScope?: string): ActionState {
  switch (outcome) {
    case "apply":
      return { id: "apply", label: "Apply plan", available: true, requiredScope };
    case "no_op":
      return {
        id: "apply",
        label: "Apply plan",
        available: false,
        reason: { code: "no_op", message: "Nothing would change: the target already runs the desired release and configuration." },
        requiredScope,
      };
    case "needs_input":
      return {
        id: "apply",
        label: "Apply plan",
        available: false,
        reason: { code: "needs_input", message: "The plan needs operator input before it can be applied." },
        requiredScope,
      };
    default:
      return {
        id: "apply",
        label: "Apply plan",
        available: false,
        reason: { code: "plan_unavailable", message: "No reviewed plan is available." },
        requiredScope,
      };
  }
}

/** The cancel action, derived from the standing (terminal or already requested). */
export function cancelActionFor(standing: OperationStanding | null | undefined, requiredScope?: string): ActionState {
  if (!standing) {
    return { id: "cancel", label: "Cancel operation", available: false, reason: { code: "no_operation", message: "No operation to cancel." }, requiredScope };
  }
  if (standing.terminal || isTerminalState(standing.state)) {
    return { id: "cancel", label: "Cancel operation", available: false, reason: { code: "terminal", message: `The operation is already ${standing.state}.` }, requiredScope };
  }
  if (standing.cancel_requested || standing.state === "cancel_requested") {
    return { id: "cancel", label: "Cancel operation", available: false, reason: { code: "cancel_requested", message: "Cancellation is recorded and will be honoured at the next cancel point." }, requiredScope };
  }
  return { id: "cancel", label: "Cancel operation", available: true, requiredScope };
}

/** Desired vs observed release comparison for the release panel. */
export type ReleaseComparison = "match" | "mismatch" | "unknown";

export function compareRelease(desired: string | undefined, observed: string | undefined): ReleaseComparison {
  if (!desired || !observed) return "unknown";
  return desired === observed ? "match" : "mismatch";
}
