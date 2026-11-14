/**
 * The Hub's side of the explainer bridge. The sealed page can only post
 * messages; everything it sends is untrusted input and is validated here
 * before anything in the Hub reacts to it. The page's half is `frameBridge.js`.
 */
export const BRIDGE_PROTOCOL = 1;

/** Frame heights the Hub accepts, in CSS pixels. */
export const MIN_FRAME_HEIGHT = 80;
export const MAX_FRAME_HEIGHT = 6000;
/** A pre-fill value larger than this (as JSON) is ignored. */
const MAX_PREFILL_BYTES = 16 * 1024;
const MAX_QUESTION_ID = 96;
const MAX_ENTRY_PATH = 512;
const ATTACHMENT_NAME = /^[A-Za-z0-9][A-Za-z0-9._-]{0,95}$/;

export type BridgeMessage =
  | { type: "height"; height: number }
  | { type: "open"; name: string; path: string }
  | { type: "prefill"; questionId: string; value: unknown };

/**
 * The message a frame posted, or null when it must be ignored: it did not
 * come from the explainer's own window, it is not in the protocol, or a field
 * is out of bounds.
 */
export function parseBridgeMessage(event: Pick<MessageEvent, "source" | "data">, frame: Window | null): BridgeMessage | null {
  if (!frame || event.source !== frame) return null;
  const data: unknown = event.data;
  if (!data || typeof data !== "object") return null;
  const message = data as Record<string, unknown>;
  if (message.hub !== BRIDGE_PROTOCOL) return null;
  switch (message.type) {
    case "height": {
      const height = message.height;
      if (typeof height !== "number" || !Number.isFinite(height)) return null;
      return { type: "height", height: Math.min(MAX_FRAME_HEIGHT, Math.max(MIN_FRAME_HEIGHT, Math.ceil(height))) };
    }
    case "open": {
      const { name, path } = message;
      if (typeof name !== "string" || !ATTACHMENT_NAME.test(name)) return null;
      if (path !== undefined && (typeof path !== "string" || path.length > MAX_ENTRY_PATH)) return null;
      return { type: "open", name, path: typeof path === "string" ? path : "" };
    }
    case "prefill": {
      const { questionId, value } = message;
      if (typeof questionId !== "string" || !questionId || questionId.length > MAX_QUESTION_ID) return null;
      let size = 0;
      try {
        size = JSON.stringify(value ?? null).length;
      } catch {
        return null;
      }
      return size <= MAX_PREFILL_BYTES ? { type: "prefill", questionId, value } : null;
    }
    default:
      return null;
  }
}
