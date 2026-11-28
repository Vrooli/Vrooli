import { ListAsksRequest_Order, type Ask } from "@vrooli/proto-types/notification-hub/v1/conversations/conversations_pb";

import { strings } from "../../consts/strings";
import { URGENCY_LABEL } from "./askView";

/**
 * How a decision list is arranged: its sort, and the urgency and sender filters.
 * It lives in the address, so a reload, a shared link and Back all keep it, and
 * the server does the sorting and filtering so it spans every page.
 */
export const SORTS = ["attention", "due", "newest", "oldest"] as const;
export type Sort = typeof SORTS[number];

export const SORT_LABEL = {
  attention: strings.asks.sort.attention,
  due: strings.asks.sort.due,
  newest: strings.asks.sort.newest,
  oldest: strings.asks.sort.oldest,
} as const satisfies Record<Sort, string>;

const ORDER: Record<Sort, ListAsksRequest_Order> = {
  attention: ListAsksRequest_Order.ATTENTION,
  due: ListAsksRequest_Order.DUE,
  newest: ListAsksRequest_Order.NEWEST,
  oldest: ListAsksRequest_Order.OLDEST,
};

/** The most urgent first: the order the urgency chips and the attention groups read in. */
export const URGENCIES = ["critical", "high", "normal", "low"] as const;
export type Urgency = typeof URGENCIES[number];

const MAX_SENDER = 60;

export interface ListView {
  sort: Sort;
  urgencies: Urgency[];
  senders: string[];
}

/** Needs you opens on what needs the owner first; History on what happened last. */
export const defaultSort = (queue: boolean): Sort => (queue ? "attention" : "newest");

export function readListView(params: URLSearchParams, queue: boolean): ListView {
  const sort = SORTS.find((value) => value === params.get("sort")) ?? defaultSort(queue);
  const urgencies = URGENCIES.filter((value) => params.getAll("urgency").includes(value));
  const senders = [...new Set(params.getAll("sender").map((name) => name.trim()).filter((name) => name !== "" && name.length <= MAX_SENDER))];
  return { sort, urgencies, senders };
}

/** The view written onto `params` (a copy is returned); a sort that is the default is left out of the address. */
export function writeListView(params: URLSearchParams, view: ListView, queue: boolean): URLSearchParams {
  const next = new URLSearchParams(params);
  next.delete("sort");
  next.delete("urgency");
  next.delete("sender");
  if (view.sort !== defaultSort(queue)) next.set("sort", view.sort);
  view.urgencies.forEach((value) => next.append("urgency", value));
  view.senders.forEach((name) => next.append("sender", name));
  return next;
}

/** The one plain line that says how each sort orders the list; attention takes the closing window in hours as its count. */
export const SORT_HOW = {
  attention: strings.asks.sort.how.attention,
  due: strings.asks.sort.how.due,
  newest: strings.asks.sort.how.newest,
  oldest: strings.asks.sort.how.oldest,
} as const satisfies Record<Sort, string>;

/** The Hub's closing window in whole hours, or 0 when it has not said (no page read yet). */
export const closingHours = (seconds: number | undefined) => Math.round((seconds ?? 0) / 3600);

export const filterCount = (view: ListView) => view.urgencies.length + view.senders.length;

/** The fields of a `ListAsks` request that carry the view. Defaults and empty filters are left out. */
export function listViewRequest(view: ListView, queue: boolean) {
  return {
    // History's default order is the server's default, so it is not sent.
    ...(queue || view.sort !== "newest" ? { order: ORDER[view.sort] } : {}),
    ...(view.urgencies.length > 0 ? { urgencies: view.urgencies } : {}),
    ...(view.senders.length > 0 ? { senders: view.senders } : {}),
  };
}

/**
 * The group an ask sits in when the list is ordered by attention: escalated
 * asks first, then those closing soon (as the Hub measured it), then each urgency.
 */
export type AttentionTier = "escalated" | "closing" | Urgency;

export function attentionTier(ask: Pick<Ask, "state" | "urgency" | "closingSoon">): AttentionTier {
  if (ask.state === "escalated") return "escalated";
  if (ask.closingSoon) return "closing";
  return URGENCIES.find((value) => value === ask.urgency) ?? "normal";
}

/** The group headings; "closing" takes the window's length in hours as its count. */
export const TIER_LABEL = {
  escalated: strings.asks.row.escalated,
  closing: strings.asks.group.closing,
  critical: URGENCY_LABEL.critical,
  high: URGENCY_LABEL.high,
  normal: URGENCY_LABEL.normal,
  low: URGENCY_LABEL.low,
} as const satisfies Record<AttentionTier, string>;

/** Explicit producer family only: unclassified asks remain independent. Groups follow the first ask's list position. */
export function groupByFamily(asks: readonly Ask[]): { key: string; cause: string; asks: Ask[] }[] {
  const groups: { key: string; cause: string; asks: Ask[] }[] = [];
  const byFamily = new Map<string, { key: string; cause: string; asks: Ask[] }>();
  for (const ask of asks) {
    const family = ask.decisionFamily.trim();
    const cause = ask.decisionCause.trim();
    if (!family || !cause) {
      groups.push({ key: `ask:${ask.id}`, cause: "", asks: [ask] });
      continue;
    }
    const key = `${family}\u0000${cause}`;
    let group = byFamily.get(key);
    if (!group) {
      group = { key: family, cause, asks: [] };
      byFamily.set(key, group);
      groups.push(group);
    }
    group.asks.push(ask);
  }
  return groups;
}

/** Families move together under the highest-attention tier of their first listed ask. */
export function groupByAttentionAndFamily(asks: readonly Ask[], attention: boolean): { tier: AttentionTier; families: ReturnType<typeof groupByFamily> }[] {
  const groups: { tier: AttentionTier; families: ReturnType<typeof groupByFamily> }[] = [];
  for (const family of groupByFamily(asks)) {
    const tier = attention ? attentionTier(family.asks[0]!) : "normal";
    const last = groups.at(-1);
    if (last?.tier === tier) last.families.push(family);
    else groups.push({ tier, families: [family] });
  }
  return groups;
}
