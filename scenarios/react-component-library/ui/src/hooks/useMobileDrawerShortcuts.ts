import { useEffect } from "react";
import { emitShortcutIntent } from "@vrooli/iframe-bridge";

interface UseMobileDrawerShortcutsInput {
  open: boolean;
  onClose: () => void;
}

export function useMobileDrawerShortcuts({ open, onClose }: UseMobileDrawerShortcutsInput) {
  useEffect(() => {
    if (!open) return;
    const handler = (e: KeyboardEvent) => {
      if (e.key !== "Escape") return;
      emitShortcutIntent({
        action: "react-component-library.mobile-drawer.close",
        chord: "Escape",
        source: "keyboard",
        outcome: "handled",
      });
      onClose();
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [open, onClose]);
}
