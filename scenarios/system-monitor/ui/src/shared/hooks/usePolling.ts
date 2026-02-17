import { useEffect, useRef } from 'react';

/**
 * Runs `callback` on a fixed interval while `enabled` is true.
 * Automatically clears the interval on unmount or when deps change.
 * Does NOT invoke `callback` immediately — callers should trigger
 * the initial fetch separately (e.g. in a useEffect).
 */
export function usePolling(
  callback: () => void | Promise<void>,
  intervalMs: number,
  enabled = true
): void {
  const savedCallback = useRef(callback);
  savedCallback.current = callback;

  useEffect(() => {
    if (!enabled || intervalMs <= 0) return;

    const id = setInterval(() => {
      void savedCallback.current();
    }, intervalMs);

    return () => clearInterval(id);
  }, [intervalMs, enabled]);
}
