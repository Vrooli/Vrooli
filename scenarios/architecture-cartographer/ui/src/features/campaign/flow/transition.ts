/**
 * Campaign item-lifecycle state machine — UI runtime authority.
 *
 * Re-targeted from the conflict-resolution flow (`features/conflicts/flow`):
 * the *states* are the same seven `CampaignItemStatus` values, but the
 * *events* are the ones the campaign TRACKER actually drives. Conflicts had
 * an operator manually walking assign→resolve→validate→commit; the campaign
 * tracker instead ingests items (all `detected`), the agent marks them
 * `resolve`/`apply`, and a re-audit reconciles by stable id — absent items
 * `validate`, reappeared terminal items `regress` back to `detected`.
 *
 * Mirrors the API-side reconciliation in
 * `api/internal/campaign/service.go` (ResolveItem / ReauditCampaign). The exported
 * `transition()` is replayed against `flow.json` in `flow.test.ts`; any drift
 * fails CI. Per-item action buttons derive from `legalActionsFor(state)`
 * so the UI never surfaces an action the tracker can't perform on that state.
 *
 * `validate`, `regress`, and `close` are campaign-level operations (a whole
 * re-audit, or closing the campaign), NOT per-item buttons — they appear
 * in the matrix so the lifecycle is fully described and testable, but
 * `legalActionsFor` returns only the per-item actions.
 */

import { CampaignItemStatus } from "@vrooli/proto-types/architecture-cartographer/v1/campaign/campaign_pb";

export const CAMPAIGN_STATES = [
  "detected",
  "assigned",
  "split",
  "resolved",
  "validated",
  "committed",
  "force_resolved",
] as const;

export type CampaignItemState = (typeof CAMPAIGN_STATES)[number];

export const CAMPAIGN_EVENTS = [
  "resolve",
  "apply",
  "validate",
  "regress",
  "close",
] as const;

export type CampaignEvent = (typeof CAMPAIGN_EVENTS)[number];

/** Per-item actions the UI surfaces as buttons. */
export type CampaignItemAction = Extract<CampaignEvent, "resolve" | "apply">;

/** Initial state every ingested item enters. */
export const INITIAL_CAMPAIGN_STATE: CampaignItemState = "detected";

/**
 * States that still need agent work — the tracker's `IsOpen`. The worklist
 * (`NextCampaignStep`) only returns items in these states.
 */
export const OPEN_CAMPAIGN_STATES: readonly CampaignItemState[] = [
  "detected",
  "assigned",
  "split",
];

/**
 * Defensive terminal sinks. The campaign domain never *produces* committed
 * or force_resolved (it has no commit/force path — those enum values are
 * mirrored from the conflict lifecycle), so they act as absorbing states:
 * no event moves an item back out. Keeping them sinks mirrors the
 * conflict flow's "committed is terminal" invariant.
 */
export const TERMINAL_CAMPAIGN_STATES: readonly CampaignItemState[] = [
  "committed",
  "force_resolved",
];

/**
 * Map proto CampaignItemStatus → UI state. Unspecified normalizes to
 * "detected" so a malformed server response never leaves the UI without a
 * legal state. This is the only normalization seam.
 */
export function statusToState(status: CampaignItemStatus): CampaignItemState {
  switch (status) {
    case CampaignItemStatus.DETECTED:
      return "detected";
    case CampaignItemStatus.ASSIGNED:
      return "assigned";
    case CampaignItemStatus.SPLIT:
      return "split";
    case CampaignItemStatus.RESOLVED:
      return "resolved";
    case CampaignItemStatus.VALIDATED:
      return "validated";
    case CampaignItemStatus.COMMITTED:
      return "committed";
    case CampaignItemStatus.FORCE_RESOLVED:
      return "force_resolved";
    case CampaignItemStatus.UNSPECIFIED:
      return "detected";
    default:
      return "detected";
  }
}

/** Whether an item still needs work (the inverse of terminal). */
export function isOpenState(state: CampaignItemState): boolean {
  return OPEN_CAMPAIGN_STATES.includes(state);
}

/**
 * Pure transition function mirroring the tracker's reconciliation. Disallowed
 * events are state-preserving.
 */
export function transition(state: CampaignItemState, event: CampaignEvent): CampaignItemState {
  if (TERMINAL_CAMPAIGN_STATES.includes(state)) return state;

  // close is a campaign-level op; it never changes an item's status.
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
      // A validated item that reappears in a re-audit regresses.
      return event === "regress" ? "detected" : state;
    }
    default:
      return state;
  }
}

/**
 * The per-item actions legal from the given state. Only `resolve`/`apply`
 * are per-item buttons; `validate`/`regress`/`close` are campaign-level.
 * Open items can be resolved or applied; closed items expose nothing.
 */
export function legalActionsFor(state: CampaignItemState): readonly CampaignItemAction[] {
  return isOpenState(state) ? (["resolve", "apply"] as const) : [];
}
