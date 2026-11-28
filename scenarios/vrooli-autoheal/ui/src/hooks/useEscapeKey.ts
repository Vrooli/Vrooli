import { useEffect } from "react";
import { emitShortcutIntent } from "@vrooli/iframe-bridge";

export function useEscapeKey(onEscape: () => void, enabled = true) {
  useEffect(() => {
    if (!enabled) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onEscape();
        emitShortcutIntent({
          action: "dialog.close",
          outcome: "handled",
          chord: "Escape",
          source: "keyboard",
        });
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [enabled, onEscape]);
}
