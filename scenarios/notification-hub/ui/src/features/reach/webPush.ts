/** Small pure helpers for Web Push enrollment and channel addresses. */

/** Convert a URL-safe base64 VAPID key to the bytes `pushManager.subscribe` wants. */
export function base64UrlToBytes(value: string): Uint8Array {
  const normalized = value.replace(/-/g, "+").replace(/_/g, "/").padEnd(Math.ceil(value.length / 4) * 4, "=");
  const decoded = atob(normalized);
  return Uint8Array.from(decoded, (character) => character.charCodeAt(0));
}

/**
 * A UUID for a new channel address. The caller keeps it across retries, so a
 * create whose answer was lost is recognised by the Hub rather than repeated.
 * Browsers outside a secure context have no `crypto.randomUUID`.
 */
export function newStableAddressId(): string {
  const cryptoApi = (globalThis as { crypto?: Crypto }).crypto;
  if (typeof cryptoApi?.randomUUID === "function") return cryptoApi.randomUUID();
  if (typeof cryptoApi?.getRandomValues !== "function") throw new Error("Secure random IDs are unavailable in this browser");
  const bytes = cryptoApi.getRandomValues(new Uint8Array(16));
  bytes[6] = ((bytes[6] ?? 0) & 0x0f) | 0x40;
  bytes[8] = ((bytes[8] ?? 0) & 0x3f) | 0x80;
  const hex = [...bytes].map((byte) => byte.toString(16).padStart(2, "0")).join("");
  return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(12, 16)}-${hex.slice(16, 20)}-${hex.slice(20)}`;
}

/** Browser push registers a device named for the origin it was enrolled from. */
export function isThisBrowser(device: { name: string; channels: readonly string[] }): boolean {
  return device.name === window.location.origin && device.channels.includes("web_push");
}

/** The content labels a channel can be approved for, least to most sensitive. */
export const CONTENT_LABELS = ["public", "private", "sensitive", "critical"] as const;
export type ContentLabel = (typeof CONTENT_LABELS)[number];

export function isContentLabel(label: string): label is ContentLabel {
  return (CONTENT_LABELS as readonly string[]).includes(label);
}

/** The same approvals regardless of order. */
export function sameLabels(a: readonly string[], b: readonly string[]): boolean {
  return a.length === b.length && [...a].sort().join("\u0000") === [...b].sort().join("\u0000");
}
