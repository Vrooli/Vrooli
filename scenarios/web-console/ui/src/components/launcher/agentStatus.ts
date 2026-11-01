/**
 * A harness has one state and at most one primary action. The server decides
 * both from what the control plane observed; this narrows the wire vocabulary
 * and supplies the state when no server answer exists, which is never reported
 * as an update.
 */
import type { AgentCard } from "./agentGrid";
import type { AgentUpdate } from "./useAgentUpdates";

export type HarnessState =
  | "missing" | "shadowed" | "unlinked" | "needs_setup" | "update_ready" | "up_to_date" | "check_failed"
  /** No build for this machine. Local to the UI: the probe knows, and there is nothing to do. */
  | "not_applicable"
  /** The check has not answered yet. */
  | "checking";

/** What the detail page's main button does. `retry` re-reads; the others are control-plane actions. */
export type HarnessAction = "install" | "use_managed_copy" | "link_managed_copy" | "update" | "retry";

const SERVER_STATES = new Set<string>(["missing", "shadowed", "unlinked", "needs_setup", "update_ready", "up_to_date", "check_failed"]);
const SERVER_ACTIONS = new Set<string>(["install", "use_managed_copy", "link_managed_copy", "update", "retry"]);
const ACTIONS_BY_STATE: Partial<Record<HarnessState, HarnessAction>> = {
  missing: "install", shadowed: "use_managed_copy", unlinked: "link_managed_copy", update_ready: "update", check_failed: "retry",
};

export function harnessState(card: AgentCard, update: AgentUpdate | null | undefined, loading: boolean): HarnessState {
  if (card.state === "not-applicable") return "not_applicable";
  if (update?.state && SERVER_STATES.has(update.state)) return update.state as HarnessState;
  if (card.state === "missing" || card.state === "installing") return "missing";
  if (update === undefined) return loading ? "checking" : "check_failed";
  // An older peer sends no state. "behind" alone once put a false arrow on every
  // card, so without an explicit state only "up to date" is believed.
  return update?.status === "up-to-date" ? "up_to_date" : "check_failed";
}

/**
 * The server's word decides. An empty action on a known state is deliberate: a
 * system package's shadow, or an action already running. The table only speaks
 * when there is no server answer at all.
 */
export function harnessAction(state: HarnessState, update: AgentUpdate | null | undefined): HarnessAction | null {
  if (update?.state) return update.primaryAction && SERVER_ACTIONS.has(update.primaryAction) ? update.primaryAction as HarnessAction : null;
  return ACTIONS_BY_STATE[state] ?? null;
}

/** Something the operator can act on. A failed check is not: it earns no dot and no count. */
export function harnessNeedsAction(state: HarnessState): boolean {
  return state === "missing" || state === "shadowed" || state === "unlinked" || state === "needs_setup" || state === "update_ready";
}

export type PillTone = "ok" | "up" | "fix" | "muted";

export const PILL_TONE: Record<HarnessState, PillTone> = {
  up_to_date: "ok",
  update_ready: "up",
  missing: "fix",
  shadowed: "fix",
  unlinked: "fix",
  needs_setup: "fix",
  check_failed: "muted",
  not_applicable: "muted",
  checking: "muted",
};
