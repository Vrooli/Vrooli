import { useEffect } from "react";

/**
 * Invokes `onEscape` when the Escape key is pressed while `active` is true.
 * Centralizing the listener here keeps overlay components free of raw
 * `addEventListener` calls (which fight host-frame spatial navigation).
 */
export function useEscapeKey(active: boolean, onEscape: () => void): void {
  useEffect(() => {
    if (!active) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onEscape();
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [active, onEscape]);
}