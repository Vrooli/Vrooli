import { useEffect } from "react";
import { emitShortcutIntent } from "@vrooli/iframe-bridge";

// Centralizes document-level dismissal behavior so embedded UI components do
// not each install their own global keyboard listener.
export function useEscapeDismiss(active: boolean, onDismiss: () => void) {
  useEffect(() => {
    if (!active) return;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onDismiss();
        emitShortcutIntent({
          action: "keyboard.escape",
          outcome: "handled",
          chord: "Escape",
        });
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [active, onDismiss]);
}
