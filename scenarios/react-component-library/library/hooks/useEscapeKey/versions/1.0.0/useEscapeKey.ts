import { useEffect } from "react";

/**
 * Invokes `onEscape` when the Escape key is pressed while `active` is true.
 * Keeping the listener in a shared hook prevents overlay components from
 * competing with host-frame spatial navigation through ad-hoc listeners.
 */
export function useEscapeKey(active: boolean, onEscape: () => void): void {
  useEffect(() => {
    if (!active) return;
    if (typeof window === "undefined") return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onEscape();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [active, onEscape]);
}
