import { useEffect } from "react";
import { emitShortcutIntent } from "@vrooli/iframe-bridge";

/** Registers the canonical escaped-dialog shortcut and relays it to an iframe host. */
export function useEscapeKey(onEscape: () => void, enabled = true): void {
  useEffect(() => {
    if (!enabled) return;
    const handler = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      onEscape();
      emitShortcutIntent({ action: "dialog.close", outcome: "handled", chord: "Escape", source: "keyboard" });
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [enabled, onEscape]);
}
