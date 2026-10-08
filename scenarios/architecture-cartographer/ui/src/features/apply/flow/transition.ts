/**
 * Per-domain apply state machine — UI runtime authority.
 *
 * Mirrors the proto-side ApplyStatus enum and FLOWS.md::Per-domain apply.
 * `transition()` is replayed against `flow.json` in `flow.test.ts`; any
 * drift fails CI. Action gating in components derives the legal event list
 * from `legalEventsFor(state)` so disallowed transitions never render a
 * button.
 */
import { ApplyStatus } from "@vrooli/proto-types/architecture-cartographer/v1/apply/apply_pb";

export const APPLY_STATES = [
  "baseline_captured",
  "plan_generated",
  "dry_run_ok",
  "applied",
  "committed",
  "refused_build_break",
  "force_committed",
] as const;

export type ApplyState = (typeof APPLY_STATES)[number];

export const APPLY_EVENTS = [
  "plan",
  "dry_run",
  "apply",
  "commit",
  "refuse",
  "force_commit",
  "rebaseline",
] as const;

export type ApplyEvent = (typeof APPLY_EVENTS)[number];

export const INITIAL_APPLY_STATE: ApplyState = "baseline_captured";

export const TERMINAL_APPLY_STATES: readonly ApplyState[] = [
  "committed",
  "force_committed",
];

/**
 * Normalize a proto ApplyStatus into a UI-side state. Unspecified or
 * unknown values fall back to baseline_captured so a malformed server
 * response never leaves the UI without a legal state.
 */
export function statusToState(status: ApplyStatus): ApplyState {
  switch (status) {
    case ApplyStatus.PLANNED:
      return "plan_generated";
    case ApplyStatus.RUNNING:
      return "applied";
    case ApplyStatus.BUILD_GREEN:
      return "applied";
    case ApplyStatus.BUILD_RED:
      return "refused_build_break";
    case ApplyStatus.REVERTED:
      return "refused_build_break";
    case ApplyStatus.COMMITTED:
      return "committed";
    case ApplyStatus.UNSPECIFIED:
      return "baseline_captured";
    default:
      return "baseline_captured";
  }
}

export function transition(state: ApplyState, event: ApplyEvent): ApplyState {
  if (TERMINAL_APPLY_STATES.includes(state)) return state;

  switch (state) {
    case "baseline_captured":
      if (event === "plan" || event === "dry_run") return "plan_generated";
      return state;
    case "plan_generated":
      if (event === "dry_run") return "dry_run_ok";
      if (event === "apply") return "applied";
      return state;
    case "dry_run_ok":
      if (event === "plan") return "plan_generated";
      if (event === "apply") return "applied";
      return state;
    case "applied":
      if (event === "commit") return "committed";
      if (event === "refuse") return "refused_build_break";
      return state;
    case "refused_build_break":
      if (event === "force_commit") return "force_committed";
      if (event === "rebaseline") return "baseline_captured";
      if (event === "plan") return "plan_generated";
      return state;
    default:
      return state;
  }
}

export function legalEventsFor(state: ApplyState): readonly ApplyEvent[] {
  return APPLY_EVENTS.filter((event) => transition(state, event) !== state);
}
