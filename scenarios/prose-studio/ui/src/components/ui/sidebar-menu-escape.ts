import { useEffect } from "react";

export function useSidebarMenuEscape(enabled: boolean, onEscape: () => void) {
  useEffect(() => {
    if (!enabled) {
      return;
    }
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onEscape();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [enabled, onEscape]);
}
