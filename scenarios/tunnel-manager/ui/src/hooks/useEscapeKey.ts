import { useEffect, useRef } from "react";
import { emitShortcutIntent } from "@vrooli/iframe-bridge";

/**
 * Attach one document-level Escape listener for an active overlay.
 *
 * Keeping the listener in a hook gives every modal the same interop behavior:
 * it only listens while active, always calls the latest callback, and cleans
 * up before an iframe or proxied shell can retain a stale handler.
 */
export function useEscapeKey(enabled: boolean, onEscape: () => void): void {
  const callbackRef = useRef(onEscape);

  useEffect(() => {
    callbackRef.current = onEscape;
  }, [onEscape]);

  useEffect(() => {
    if (!enabled) return undefined;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        callbackRef.current();
        emitShortcutIntent({ action: "dialog.close", outcome: "handled", chord: "Escape", source: "keyboard" });
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [enabled]);
}
