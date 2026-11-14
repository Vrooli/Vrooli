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

/** The group an ask sits in when the list is ordered by attention: escalated asks first, then each urgency. */
export type AttentionTier = "escalated" | Urgency;

export function attentionTier(ask: Pick<Ask, "state" | "urgency">): AttentionTier {
  if (ask.state === "escalated") return "escalated";
  return URGENCIES.find((value) => value === ask.urgency) ?? "normal";
}

export const TIER_LABEL = {
  escalated: strings.asks.row.escalated,
  critical: URGENCY_LABEL.critical,
  high: URGENCY_LABEL.high,
  normal: URGENCY_LABEL.normal,
  low: URGENCY_LABEL.low,
} as const satisfies Record<AttentionTier, string>;

/** Consecutive asks of one tier, which is how an attention-ordered list reads. */
export function groupByTier(asks: readonly Ask[]): { tier: AttentionTier; asks: Ask[] }[] {
  const groups: { tier: AttentionTier; asks: Ask[] }[] = [];
  for (const ask of asks) {
    const tier = attentionTier(ask);
    const last = groups.at(-1);
    if (last?.tier === tier) last.asks.push(ask);
    else groups.push({ tier, asks: [ask] });
  }
  return groups;
}
