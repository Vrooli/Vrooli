/**
 * useStorePolling — Shared hook for interval-based store polling.
 *
 * Starts a `setInterval` when `enabled` is true and cleans up when
 * `enabled` flips to false or the component unmounts.
 */

import { useEffect, useRef } from "react";

export interface UseStorePollingOptions {
  /** Whether polling is currently active. */
  enabled: boolean;
  /** Polling interval in milliseconds. */
  intervalMs: number;
  /** Function called on each tick. May return a promise (it will be voided). */
  pollFn: () => void | Promise<void>;
  /** If true, `pollFn` is invoked immediately when polling starts. */
  immediate?: boolean;
}

export function useStorePolling({
  enabled,
  intervalMs,
  pollFn,
  immediate = false,
}: UseStorePollingOptions): void {
  // Keep pollFn in a ref so the interval always calls the latest version
  // without needing to restart the timer on every render.
  const pollFnRef = useRef(pollFn);
  pollFnRef.current = pollFn;

  useEffect(() => {
    if (!enabled) return;

    if (immediate) {
      void pollFnRef.current();
    }

    const id = window.setInterval(() => {
      void pollFnRef.current();
    }, intervalMs);

    return () => window.clearInterval(id);
  }, [enabled, intervalMs, immediate]);
}
