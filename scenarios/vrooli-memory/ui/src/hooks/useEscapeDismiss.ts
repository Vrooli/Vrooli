import { useEffect } from "react";

// useEscapeDismiss centralizes the window-level keyboard listener used by
// dismissible surfaces so interop review can audit it in one hook.
export function useEscapeDismiss(enabled: boolean, onDismiss: () => void) {
  useEffect(() => {
    if (!enabled) return;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onDismiss();
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [enabled, onDismiss]);
}
