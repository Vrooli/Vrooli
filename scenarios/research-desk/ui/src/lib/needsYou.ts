import type { CourierStatus, NeedsYouItem } from "../api/operator";

/**
 * One item's identity: its commission plus its ID, never the bare ID, which
 * two commissions may share (rulings V2). Global ATTENTION.md items have no
 * commission and use "global".
 */
export function itemKey(item: Pick<NeedsYouItem, "commissionId" | "id">): string {
  return `${item.commissionId || "global"}/${item.id}`;
}

/** The URL fragment that opens an item on the Needs you page. */
export function itemAnchor(item: Pick<NeedsYouItem, "commissionId" | "id">): string {
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
