/**
 * Hook for managing network requests state in the report dialog
 */

import { useCallback, useMemo, useReducer, useRef } from 'react';
import type {
  BridgeNetworkEvent,
  BridgeNetworkStreamState,
} from '@vrooli/iframe-bridge';
import type { App } from '@/types';
import { logger } from '@/services/logger';
import type { ReportNetworkEntry } from './reportTypes';
import { isTruncated } from './reportTypes';
import { REPORT_NETWORK_MAX_EVENTS } from './reportConstants';
import { formatOptionalTimestamp, toNetworkEntry } from './reportFormatters';
import {
  reportNetworkReducer,
  initialReportNetworkState,
} from './reportNetworkReducer';

interface UseReportNetworkDataParams {
  app: App | null;
  appId?: string;
  bridgeSupported: boolean;
  bridgeCaps: string[];
  networkState: BridgeNetworkStreamState | null;
  configureNetwork: ((config: { enable?: boolean; streaming?: boolean; bufferSize?: number }) => boolean) | null;
  getRecentNetworkEvents: () => BridgeNetworkEvent[];
  requestNetworkBatch: (options?: { since?: number; afterSeq?: number; limit?: number }) => Promise<BridgeNetworkEvent[]>;
}

export interface ReportNetworkDataState {
  includeNetworkRequests: boolean;
  setIncludeNetworkRequests: (value: boolean) => void;
  events: ReportNetworkEntry[];
  expanded: boolean;
  toggleExpanded: () => void;
  loading: boolean;
  error: string | null;
  truncated: boolean;
  formattedCapturedAt: string | null;
  total: number | null;
  fetch: (options?: { force?: boolean }) => Promise<void>;
  reset: () => void;
}

/**
 * Hook for managing network requests data fetching and state from the bridge
 */
export function useReportNetworkData({
  app,
  appId,
  bridgeSupported,
  bridgeCaps,
  networkState,
  configureNetwork,
  getRecentNetworkEvents,
  requestNetworkBatch,
}: UseReportNetworkDataParams): ReportNetworkDataState {
  const [state, dispatch] = useReducer(reportNetworkReducer, initialReportNetworkState);
  const fetchedForRef = useRef<string | null>(null);

  const resolveIdentifier = useCallback(() => {
    const candidates = [app?.scenario_name, app?.id, appId]
      .map(value => (typeof value === 'string' ? value.trim() : ''))
      .filter(Boolean);

    return candidates.length > 0 ? candidates[0] : null;
  }, [app, appId]);

  const fetch = useCallback(async (options?: { force?: boolean }) => {
    const identifier = resolveIdentifier();
    if (!identifier) {
      dispatch({ type: 'FETCH_ERROR', payload: 'Unable to determine which network events to include.' });
      fetchedForRef.current = null;
      return;
    }

    const normalizedIdentifier = identifier.toLowerCase();
    if (!options?.force && fetchedForRef.current === normalizedIdentifier) {
      return;
    }

    // The iframe bridge is the only network-capture source. When it does not
    // advertise network support, degrade to a clear, informative empty state.
    if (!bridgeSupported || !bridgeCaps.includes('network')) {
      dispatch({
        type: 'FETCH_ERROR',
        payload: "Network requests unavailable — the preview's iframe bridge did not advertise network support.",
      });
      fetchedForRef.current = null;
      return;
    }

    dispatch({ type: 'FETCH_START' });

    try {
      if (networkState && configureNetwork && networkState.enabled === false) {
        configureNetwork({ enable: true });
      }

      let events: BridgeNetworkEvent[] = [];
      try {
        events = await requestNetworkBatch({ limit: REPORT_NETWORK_MAX_EVENTS });
      } catch (error) {
        logger.warn('Network request snapshot failed; using buffered events', error);
        events = getRecentNetworkEvents();
      }

      if (!Array.isArray(events)) {
        events = [];
      }

      const limited = events.slice(-REPORT_NETWORK_MAX_EVENTS);
      const entries = limited.map(toNetworkEntry);

      dispatch({
        type: 'FETCH_SUCCESS',
        payload: {
          events: entries,
          total: events.length,
          fetchedAt: Date.now(),
        },
      });
      fetchedForRef.current = normalizedIdentifier;
    } catch (error) {
      logger.warn('Failed to load network events for issue report', error);
      dispatch({ type: 'FETCH_ERROR', payload: 'Unable to load network requests from the preview iframe.' });
      fetchedForRef.current = null;
    }
  }, [
    resolveIdentifier,
    bridgeSupported,
    bridgeCaps,
    networkState,
    configureNetwork,
    getRecentNetworkEvents,
    requestNetworkBatch,
  ]);

  const reset = useCallback(() => {
    dispatch({ type: 'RESET' });
    fetchedForRef.current = null;
  }, []);

  const formattedCapturedAt = useMemo(
    () => formatOptionalTimestamp(state.fetchedAt),
    [state.fetchedAt],
  );

  const truncated = useMemo(
    () => isTruncated(state.total, state.events.length),
    [state.total, state.events.length],
  );

  return {
    includeNetworkRequests: state.include,
    setIncludeNetworkRequests: (value: boolean) => dispatch({ type: 'SET_INCLUDE', payload: value }),
    events: state.events,
    expanded: state.expanded,
    toggleExpanded: () => dispatch({ type: 'SET_EXPANDED', payload: !state.expanded }),
    loading: state.loading,
    error: state.error,
    truncated,
    formattedCapturedAt,
    total: state.total,
    fetch,
    reset,
  };
}
