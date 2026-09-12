import { useEffect } from "react";
import { emitShortcutIntent } from "@vrooli/iframe-bridge";

/** Shared, audited Escape-to-dismiss boundary for overlay components. */
export function useEscapeDismiss(open: boolean, onDismiss: () => void) {
  useEffect(() => {
    if (!open) return;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      onDismiss();
      // The bridge no-ops in standalone mode. In an embedded console it makes
      // the handled dismissal visible to the host without taking ownership of
      // any other shortcut.
      emitShortcutIntent({ action: "channel-manager.dialog.close", outcome: "handled", chord: "Escape", source: "keyboard" });
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [open, onDismiss]);
}
