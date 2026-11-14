import { useCallback, useLayoutEffect, useRef } from "react";

/**
 * Returns a stable function that runs `fn` at most once per `waitMs`, on the
 * trailing edge, so it sees the latest state. A run still pending at unmount
 * happens then (while the DOM is still attached), so the last change is never
 * lost to a quick navigation or reload.
 */
export function useTrailingThrottle(fn: () => void, waitMs: number): () => void {
  const fnRef = useRef(fn);
  fnRef.current = fn;
  const timerRef = useRef<number | null>(null);

  useLayoutEffect(() => () => {
    if (timerRef.current == null) return;
    window.clearTimeout(timerRef.current);
    timerRef.current = null;
    fnRef.current();
  }, []);

  return useCallback(() => {
    if (timerRef.current != null) return;
    timerRef.current = window.setTimeout(() => {
      timerRef.current = null;
      fnRef.current();
    }, waitMs);
  }, [waitMs]);
}
