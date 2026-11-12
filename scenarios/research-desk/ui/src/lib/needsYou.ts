import type { TFunction } from "i18next";

import { NeedsYouKind, type CourierStatus, type ListNeedsYouResponse, type NeedsYouItem } from "../api/operator";
import { strings } from "../consts/strings";

/**
 * One row's identity: its commission plus its ID, never the bare ID, which
 * two commissions may share. Global ATTENTION.md items have no
 * commission and use "global"; an older list has no ID and uses its file.
 */
export function itemKey(item: Pick<NeedsYouItem, "commissionId" | "id" | "sources">): string {
  return `${item.commissionId || "global"}/${item.id || item.sources[0] || ""}`;
}

/** The URL fragment that opens an item on the Needs you page. */
export function itemAnchor(item: Pick<NeedsYouItem, "commissionId" | "id" | "sources">): string {
  return encodeURIComponent(itemKey(item));
}

/** The item key a URL fragment names, or null. */
export function keyFromHash(hash: string): string | null {
  if (!hash || hash === "#") return null;
  try {
    return decodeURIComponent(hash.replace(/^#/, ""));
  } catch {
    return null;
  }
}

/** Whether any switch sends needs-you items to Notification Hub. */
export function deliveryOn(courier: CourierStatus): boolean {
  return courier.globalOn || courier.commissions.some((c) => c.on);
}

/** An older-format STATE.md list: shown in the feed, never sent, never answerable here. */
export function isOlderList(item: Pick<NeedsYouItem, "kind">): boolean {
  return item.kind === NeedsYouKind.OLDER_LIST;
}

/** The rows the feed lists open, in the API's order: unanswered items, then older lists. Counts come from the API. */
export function waiting(items: NeedsYouItem[]): NeedsYouItem[] {
  return items.filter((item) => !item.answer);
}

/** Everything waiting on the operator, as the API counts it: items still waiting plus older lists. */
export function waitingCount(feed: Pick<ListNeedsYouResponse, "itemCount" | "olderListCount">): number {
  return feed.itemCount + feed.olderListCount;
}

/** The API's counts in words: "9 need you", "7 items and 4 older lists need you" or "2 older lists need you". */
export function needsYouCount(t: TFunction, { itemCount: items, olderListCount: older }: Pick<ListNeedsYouResponse, "itemCount" | "olderListCount">): string {
  if (older === 0) return t(strings.now.needsYou, { count: items });
  if (items === 0) return t(strings.now.olderListsNeedYou, { count: older });
  return t(strings.now.needsYouWithOlder, { items: t(strings.needsYou.itemCount, { count: items }), lists: t(strings.needsYou.olderCount, { count: older }) });
}
