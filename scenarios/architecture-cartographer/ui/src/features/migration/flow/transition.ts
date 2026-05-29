/**
 * Migration finding-lifecycle state machine — UI runtime authority.
 *
 * Re-targeted from the conflict-resolution flow (`features/conflicts/flow`):
 * the *states* are the same seven `TrackedFindingStatus` values, but the
 * *events* are the ones the migration TRACKER actually drives. Conflicts had
 * an operator manually walking assign→resolve→validate→commit; the migration
 * tracker instead ingests findings (all `detected`), the agent marks them
 * `resolve`/`apply`, and a re-audit reconciles by stable id — absent findings
 * `validate`, reappeared terminal findings `regress` back to `detected`.
 *
 * Mirrors the API-side reconciliation in
 * `api/internal/migration/service.go` (Resolve / Reaudit). The exported
 * `transition()` is replayed against `flow.json` in `flow.test.ts`; any drift
 * fails CI. Per-finding action buttons derive from `legalActionsFor(state)`
 * so the UI never surfaces an action the tracker can't perform on that state.
 *
 * `validate`, `regress`, and `close` are migration-level operations (a whole
 * re-audit, or closing the migration), NOT per-finding buttons — they appear
 * in the matrix so the lifecycle is fully described and testable, but
 * `legalActionsFor` returns only the per-finding actions.
 */

import { TrackedFindingStatus } from "@vrooli/proto-types/architecture-cartographer/v1/migration/migration_pb";

export const MIGRATION_STATES = [
  "detected",
  "assigned",
  "split",
  "resolved",
  "validated",
  "committed",
  "force_resolved",
] as const;

export type MigrationFindingState = (typeof MIGRATION_STATES)[number];

export const MIGRATION_EVENTS = [
  "resolve",
  "apply",
  "validate",
  "regress",
  "close",
] as const;

export type MigrationEvent = (typeof MIGRATION_EVENTS)[number];

/** Per-finding actions the UI surfaces as buttons. */
export type MigrationFindingAction = Extract<MigrationEvent, "resolve" | "apply">;

/** Initial state every ingested finding enters. */
export const INITIAL_MIGRATION_STATE: MigrationFindingState = "detected";

/**
 * States that still need agent work — the tracker's `IsOpen`. The worklist
 * (`NextMigrationStep`) only returns findings in these states.
 */
export const OPEN_MIGRATION_STATES: readonly MigrationFindingState[] = [
  "detected",
  "assigned",
  "split",
];

/**
 * Defensive terminal sinks. The migration domain never *produces* committed
 * or force_resolved (it has no commit/force path — those enum values are
 * mirrored from the conflict lifecycle), so they act as absorbing states:
 * no event moves a finding back out. Keeping them sinks mirrors the
 * conflict flow's "committed is terminal" invariant.
 */
export const TERMINAL_MIGRATION_STATES: readonly MigrationFindingState[] = [
  "committed",
  "force_resolved",
];

/**
 * Map proto TrackedFindingStatus → UI state. Unspecified normalizes to
 * "detected" so a malformed server response never leaves the UI without a
 * legal state. This is the only normalization seam.
 */
export function statusToState(status: TrackedFindingStatus): MigrationFindingState {
  switch (status) {
    case TrackedFindingStatus.DETECTED:
      return "detected";
    case TrackedFindingStatus.ASSIGNED:
      return "assigned";
    case TrackedFindingStatus.SPLIT:
      return "split";
    case TrackedFindingStatus.RESOLVED:
      return "resolved";
    case TrackedFindingStatus.VALIDATED:
      return "validated";
    case TrackedFindingStatus.COMMITTED:
      return "committed";
    case TrackedFindingStatus.FORCE_RESOLVED:
      return "force_resolved";
    case TrackedFindingStatus.UNSPECIFIED:
      return "detected";
    default:
      return "detected";
  }
}

/** Whether a finding still needs work (the inverse of terminal). */
export function isOpenState(state: MigrationFindingState): boolean {
  return OPEN_MIGRATION_STATES.includes(state);
}

/**
 * Pure transition function mirroring the tracker's reconciliation. Disallowed
 * events are state-preserving.
 */
export function transition(state: MigrationFindingState, event: MigrationEvent): MigrationFindingState {
  if (TERMINAL_MIGRATION_STATES.includes(state)) return state;

  // close is a migration-level op; it never changes a finding's status.
  if (event === "close") return state;

  switch (state) {
    case "detected":
    case "assigned":
    case "split": {
      switch (event) {
        case "resolve":
        case "apply":
          return "resolved";
        case "validate":
          return "validated";
        case "regress":
          return "detected";
        default:
          return state;
      }
    }
    case "resolved": {
      switch (event) {
        case "validate":
          return "validated";
        case "regress":
          return "detected";
        default:
          return state;
      }
    }
    case "validated": {
      // A validated finding that reappears in a re-audit regresses.
      return event === "regress" ? "detected" : state;
    }
    default:
      return state;
  }
}

/**
 * The per-finding actions legal from the given state. Only `resolve`/`apply`
 * are per-finding buttons; `validate`/`regress`/`close` are migration-level.
 * Open findings can be resolved or applied; closed findings expose nothing.
 */
export function legalActionsFor(state: MigrationFindingState): readonly MigrationFindingAction[] {
  return isOpenState(state) ? (["resolve", "apply"] as const) : [];
}
