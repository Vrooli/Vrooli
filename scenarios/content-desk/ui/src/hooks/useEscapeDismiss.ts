import { useEffect } from "react";

/**
 * Registers the common Escape-to-dismiss interaction for an open overlay.
 * Keeping the document listener in hooks/ makes shared components declarative
 * and gives UI-health one audited keyboard-listener boundary.
 */
export function useEscapeDismiss(open: boolean, onDismiss: () => void) {
  useEffect(() => {
    if (!open) return;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onDismiss();
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [open, onDismiss]);
}
