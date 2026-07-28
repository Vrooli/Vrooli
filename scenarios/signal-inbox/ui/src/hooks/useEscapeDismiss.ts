import { useEffect } from "react";

/** Keeps a dismissible surface's Escape listener scoped to a reusable hook. */
export function useEscapeDismiss(enabled: boolean, onDismiss: () => void) {
  useEffect(() => {
    if (!enabled) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onDismiss();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [enabled, onDismiss]);
}
