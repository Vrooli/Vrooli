import { useEffect } from "react";
import { emitShortcutIntent } from "@vrooli/iframe-bridge";

export function useDismissOnEscape(enabled: boolean, onDismiss: () => void): void {
  useEffect(() => {
    if (!enabled) return undefined;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onDismiss();
        emitShortcutIntent({ action: "notification-hub.sidebar.close", outcome: "handled", chord: "Escape", source: "keyboard" });
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [enabled, onDismiss]);
}
