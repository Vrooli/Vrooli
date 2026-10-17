import { useEffect, useRef } from 'react';

export interface PollingBackoffConfig {
  enabled: boolean;
  maxIntervalMs: number;
}

/**
 * Runs `callback` on a fixed interval while `enabled` is true.
 * Automatically clears the interval on unmount or when deps change.
 * Does NOT invoke `callback` immediately — callers should trigger
 * the initial fetch separately (e.g. in a useEffect).
 *
 * When `backoff` is provided and enabled, failures double the interval
 * (up to `maxIntervalMs`) and successes reset to `intervalMs`.
 */
export function usePolling(
  callback: () => void | Promise<void>,
  intervalMs: number,
  enabled = true,
  backoff?: PollingBackoffConfig
): void {
  const savedCallback = useRef(callback);
  savedCallback.current = callback;
  const currentIntervalRef = useRef(intervalMs);
  const backoffRef = useRef(backoff);
  backoffRef.current = backoff;

  // Reset interval when base interval changes
  useEffect(() => {
    currentIntervalRef.current = intervalMs;
  }, [intervalMs]);

  useEffect(() => {
    if (!enabled || intervalMs <= 0) return;

    let timeoutId: ReturnType<typeof setTimeout>;

    const tick = async () => {
      try {
        await savedCallback.current();
        // Success: reset to base interval
        if (backoffRef.current?.enabled) {
          currentIntervalRef.current = intervalMs;
        }
      } catch {
        // Failure: double interval up to max
        if (backoffRef.current?.enabled) {
          currentIntervalRef.current = Math.min(
            currentIntervalRef.current * 2,
            backoffRef.current.maxIntervalMs
          );
        }
      }
      timeoutId = setTimeout(() => { void tick(); }, currentIntervalRef.current);
    };

    timeoutId = setTimeout(() => { void tick(); }, currentIntervalRef.current);

    return () => { clearTimeout(timeoutId); };
  }, [intervalMs, enabled]);
}
