/**
 * Conflict-resolution state machine — UI runtime authority.
 *
 * Mirrors the API-side flow at `api/internal/conflicts/flow/flow.json` and
 * the FLOWS.md state list. The exported `transition()` is replayed against
 * `flow.json` in `flow.test.ts`; any drift fails CI.
 *
 * Action gating in `useConflictActions` derives the legal event list from
 * `legalEventsFor(state)` so disallowed transitions never render a button.
 */

import { ResolutionStatus } from "@vrooli/proto-types/architecture-cartographer/v1/conflicts/conflicts_pb";

export const CONFLICT_STATES = [
  "detected",
  "assigned",
  "split",
  "resolved",
  "validated",
  "committed",
  "force_resolved",
] as const;

export type ConflictState = (typeof CONFLICT_STATES)[number];

export const CONFLICT_EVENTS = [
  "assign",
  "split",
  "resolve",
  "force_resolve",
  "validate",
  "commit",
  "reopen",
] as const;

export type ConflictEvent = (typeof CONFLICT_EVENTS)[number];

/** Initial state every conflict enters when a detector emits it. */
export const INITIAL_CONFLICT_STATE: ConflictState = "detected";

/** Terminal state — no event moves a conflict back out of here. */
export const TERMINAL_CONFLICT_STATES: readonly ConflictState[] = ["committed"];

/**
 * Map proto ResolutionStatus → UI state. Unspecified is normalized to
 * "detected" so a malformed server response never leaves the UI without a
 * legal state. This is the only normalization seam: every other path
 * funnels through it.
 */
export function statusToState(status: ResolutionStatus): ConflictState {
  switch (status) {
    case ResolutionStatus.DETECTED:
      return "detected";
    case ResolutionStatus.ASSIGNED:
      return "assigned";
    case ResolutionStatus.SPLIT:
      return "split";
    case ResolutionStatus.RESOLVED:
      return "resolved";
    case ResolutionStatus.VALIDATED:
      return "validated";
    case ResolutionStatus.COMMITTED:
      return "committed";
    case ResolutionStatus.FORCE_RESOLVED:
      return "force_resolved";
    case ResolutionStatus.UNSPECIFIED:
      return "detected";
    default:
      return "detected";
  }
}

/** Pure transition function. Disallowed events are state-preserving. */
export function transition(state: ConflictState, event: ConflictEvent): ConflictState {
  if (TERMINAL_CONFLICT_STATES.includes(state)) return state;

  if (event === "reopen") {
    return state === INITIAL_CONFLICT_STATE ? state : "detected";
  }

  switch (state) {
    case "detected": {
      switch (event) {
        case "assign":
          return "assigned";
        case "split":
          return "split";
        case "resolve":
          return "resolved";
        case "force_resolve":
          return "force_resolved";
        default:
          return state;
      }
    }
    case "assigned": {
      switch (event) {
        case "split":
          return "split";
        case "resolve":
          return "resolved";
        case "force_resolve":
          return "force_resolved";
        default:
          return state;
      }
    }
    case "split": {
      switch (event) {
        case "resolve":
          return "resolved";
        case "force_resolve":
          return "force_resolved";
        default:
          return state;
      }
    }
    case "resolved": {
      return event === "validate" ? "validated" : state;
    }
    case "validated": {
      return event === "commit" ? "committed" : state;
    }
    case "force_resolved":
      return state;
    case "committed":
      return state;
    default:
      return state;
  }
}

/**
 * The set of events that actually move a conflict to a *different* state from
 * the given state. Used by `useConflictActions` to gate action buttons so the
 * UI never surfaces a disallowed transition.
 */
export function legalEventsFor(state: ConflictState): readonly ConflictEvent[] {
  return CONFLICT_EVENTS.filter((event) => transition(state, event) !== state);
}
