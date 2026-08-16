import { useEffect } from "react";
import { emitShortcutIntent } from "@vrooli/iframe-bridge";

/** Install one scoped Escape listener for dismissible UI surfaces. */
export function useEscapeKey(enabled: boolean, onEscape: () => void) {
  useEffect(() => {
    if (!enabled) return;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onEscape();
        emitShortcutIntent({
          action: "prose-studio.sidebar.close",
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
