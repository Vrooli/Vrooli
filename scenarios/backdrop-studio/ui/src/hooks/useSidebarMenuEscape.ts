import { useEffect } from "react";
import { emitShortcutIntent } from "@vrooli/iframe-bridge";

export function useSidebarMenuEscape(enabled: boolean, onEscape: () => void) {
  useEffect(() => {
    if (!enabled) return;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        emitShortcutIntent({ action: "backdrop-studio.sidebar.close", outcome: "handled", chord: "Escape", source: "keyboard" });
        onEscape();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [enabled, onEscape]);
}
