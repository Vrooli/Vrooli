import { useEffect } from "react";
import { emitShortcutIntent } from "@vrooli/iframe-bridge";

/**
 * Closes a locally owned surface on Escape while ensuring document-level
 * keyboard listeners are centralized in hooks rather than scattered through
 * components. Capture mode is available for nested dialogs that must consume
 * Escape before an ancestor acts on it.
 */
export function useDocumentEscape(
  enabled: boolean,
  onEscape: (event: KeyboardEvent) => void,
  capture = false,
) {
  useEffect(() => {
    if (!enabled) return;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onEscape(event);
        emitShortcutIntent({
          action: "dialog.close",
          outcome: "handled",
          chord: "Escape",
          source: "keyboard",
        });
      }
    };
    document.addEventListener("keydown", handleKeyDown, capture);
    return () => document.removeEventListener("keydown", handleKeyDown, capture);
  }, [enabled, onEscape, capture]);
}
