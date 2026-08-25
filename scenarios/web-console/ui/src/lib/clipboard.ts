/** Clipboard capability boundary. Every browser clipboard operation belongs here. */
export type ClipboardFailureReason = "unsupported" | "denied" | "failed";
export type ClipboardWriteResult = { ok: true } | { ok: false; reason: ClipboardFailureReason };
export type ClipboardReadResult = { ok: true; text: string } | { ok: false; reason: ClipboardFailureReason };

export function isClipboardSupported(): boolean {
  return (typeof navigator !== "undefined" && !!navigator.clipboard?.writeText)
    || (typeof document !== "undefined" && typeof document.execCommand === "function");
}

export async function writeText(text: string): Promise<ClipboardWriteResult> {
  if (typeof navigator !== "undefined" && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(text);
      return { ok: true };
    } catch {
      // Fall through to the legacy gesture-bound copy path.
    }
  }
  if (typeof document === "undefined") return { ok: false, reason: "unsupported" };
  const textarea = document.createElement("textarea");
  textarea.value = text;
  textarea.setAttribute("readonly", "");
  textarea.style.position = "fixed";
  textarea.style.opacity = "0";
  document.body.appendChild(textarea);
  textarea.select();
  try {
    return document.execCommand("copy")
      ? { ok: true }
      : { ok: false, reason: "denied" };
  } catch {
    return { ok: false, reason: "failed" };
  } finally {
    textarea.remove();
  }
}

export async function readText(): Promise<ClipboardReadResult> {
  if (typeof navigator !== "undefined" && navigator.clipboard?.readText) {
    try {
      return { ok: true, text: await navigator.clipboard.readText() };
    } catch {
      return { ok: false, reason: "denied" };
    }
  }
  return { ok: false, reason: "unsupported" };
}
