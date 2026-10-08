/**
 * useOperationsPolling — drives the Operations Center's 4-second tick.
 *
 * Mirrors `useAgentSessionPolling` in shape, but the operations endpoint
 * is always relevant when the page is mounted (the user explicitly
 * navigated to it), so we don't gate polling on activity-presence the
 * way the session list does. The first fetch fires immediately so the
 * page never shows a flash of empty state when its data is fresh.
 *
 * Pause controls (`enabled`) let the page suspend polling when the tab
 * is hidden or when filters are mid-edit and a refresh would clobber
 * an in-flight selection. Today only `enabled` is wired.
 */

import { useCallback, useEffect, useRef } from "react";
import { useOperationsStore } from "../stores/operations-store";
import { useStorePolling } from "./useStorePolling";

export const OPERATIONS_POLL_INTERVAL_MS = 4_000;

export interface UseOperationsPollingOptions {
  enabled?: boolean;
  intervalMs?: number;
}

export function useOperationsPolling(
  options: UseOperationsPollingOptions = {},
): void {
  const { enabled = true, intervalMs = OPERATIONS_POLL_INTERVAL_MS } = options;
  const refresh = useOperationsStore((state) => state.refresh);
  const filters = useOperationsStore((state) => state.filters);

  // Re-fetch whenever the filters object identity changes. setFilters
  // produces a new object reference each call, so this is safe even
  // though the filter values may be deeply equal across renders.
  const previousFiltersRef = useRef(filters);

  useEffect(() => {
    if (previousFiltersRef.current !== filters && enabled) {
      previousFiltersRef.current = filters;
      void refresh({ force: true });
    } else {
      previousFiltersRef.current = filters;
    }
  }, [filters, enabled, refresh]);

  const tick = useCallback(async () => {
    await refresh();
  }, [refresh]);

  useStorePolling({
    enabled,
    intervalMs,
    pollFn: tick,
    immediate: true,
  });
}
