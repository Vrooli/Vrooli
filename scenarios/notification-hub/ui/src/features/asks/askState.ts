import type { Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { strings } from "../../consts/strings";
import { isAwaiting } from "./askView";

/** The statuses a decision list can be filtered to, in the order they are offered. */
export const STATE_FILTERS = ["all", "awaiting", "answered", "defaulted", "expired", "withdrawn", "dismissed"] as const;
export type StateFilter = typeof STATE_FILTERS[number];
export type AskStateKey = Exclude<StateFilter, "all"> | "unknownState";

export const STATE_LABEL = {
  all: strings.pages.history.all,
  awaiting: strings.pages.history.awaiting,
  answered: strings.pages.history.answered,
  defaulted: strings.pages.history.defaulted,
  expired: strings.pages.history.expired,
  withdrawn: strings.pages.history.withdrawn,
  dismissed: strings.pages.history.dismissed,
  unknownState: strings.pages.history.unknownState,
} as const satisfies Record<StateFilter | "unknownState", string>;

export function parseStateFilter(value: string | null): StateFilter {
  return STATE_FILTERS.find((state) => state === value) ?? "all";
}

/** The status key a decision is shown under. Pending and escalated both read as waiting. */
export function askStateKey(ask: Pick<Ask, "state">): AskStateKey {
  if (isAwaiting(ask)) return "awaiting";
  const settled = STATE_FILTERS.filter((state): state is Exclude<AskStateKey, "unknownState" | "awaiting"> => state !== "all" && state !== "awaiting");
  return settled.find((state) => state === ask.state) ?? "unknownState";
}

export type BadgeTone = "neutral" | "success" | "warning" | "danger" | "info";

export const STATE_TONE: Record<AskStateKey, BadgeTone> = {
  awaiting: "warning",
  answered: "success",
  defaulted: "info",
  expired: "neutral",
  withdrawn: "neutral",
  dismissed: "neutral",
  unknownState: "neutral",
};
