import { CommissionState, type CommissionSummary } from "../api/activity";
import { strings } from "../consts/strings";
import { toDate } from "./format";

export type Tone = "neutral" | "success" | "warning" | "danger" | "info";

/** How each derived commission state reads: a word plus a tone, never colour alone. */
export const STATE_META: Record<CommissionState, { label: (typeof strings.state)[keyof typeof strings.state]; tone: Tone }> = {
  [CommissionState.UNSPECIFIED]: { label: strings.state.unknown, tone: "neutral" },
  [CommissionState.LIVE]: { label: strings.state.live, tone: "success" },
  [CommissionState.PAUSED]: { label: strings.state.paused, tone: "info" },
  [CommissionState.LIMITED]: { label: strings.state.limited, tone: "warning" },
  [CommissionState.STOPPED]: { label: strings.state.stopped, tone: "neutral" },
  [CommissionState.PREPARED]: { label: strings.state.prepared, tone: "info" },
  [CommissionState.ENDED]: { label: strings.state.ended, tone: "neutral" },
  [CommissionState.UNKNOWN]: { label: strings.state.unknown, tone: "neutral" },
};

/**
 * The states with a supervisor window open, in the order Now shows them.
 * Only live is working: paused and limited are open but idle (rulings F4).
 */
export const WINDOW_OPEN: CommissionState[] = [CommissionState.LIVE, CommissionState.LIMITED, CommissionState.PAUSED];

export function windowOpen(state: CommissionState): boolean {
  return WINDOW_OPEN.includes(state);
}

/** A window still open after the allowance it was given has passed: worth saying out loud (rulings F4). */
export function pastAllowance(c: CommissionSummary, now: Date = new Date()): boolean {
  const end = toDate(c.allowanceEnd?.value);
  return windowOpen(c.state) && end !== null && end <= now;
}

/** Display order for grouped lists: what needs attention first. */
export const STATE_ORDER: CommissionState[] = [
  CommissionState.LIVE,
  CommissionState.LIMITED,
  CommissionState.PAUSED,
  CommissionState.PREPARED,
  CommissionState.STOPPED,
  CommissionState.UNKNOWN,
  CommissionState.ENDED,
];

/**
 * A packet that can be launched as it is: prepared, or unknown and the desk
 * found no session or goal for it, so the operator may say it never started
 * (rulings P1, V1).
 */
export function launchableAsIs(c: CommissionSummary): boolean {
  return c.state === CommissionState.PREPARED || (c.state === CommissionState.UNKNOWN && c.neverStarted?.assertable === true);
}
