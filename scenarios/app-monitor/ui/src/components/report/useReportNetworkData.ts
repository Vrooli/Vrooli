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
import { appService } from '@/services/api';
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
  activePreviewUrl: string | null;
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
  fromFallback: boolean;
}

/**
 * Hook for managing network requests data fetching and state from the bridge
 */
export function useReportNetworkData({
  app,
  appId,
  activePreviewUrl,
  bridgeSupported,
  bridgeCaps,
  networkState,
  configureNetwork,
  getRecentNetworkEvents,
  requestNetworkBatch,
}: UseReportNetworkDataParams): ReportNetworkDataState {
  const [state, dispatch] = useReducer(reportNetworkReducer, initialReportNetworkState);
  const fetchedForRef = useRef<string | null>(null);
  const fromFallbackRef = useRef<boolean>(false);

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

    // Try fallback if bridge doesn't support network
    if (!bridgeSupported || !bridgeCaps.includes('network')) {
      if (activePreviewUrl) {
        logger.info('Bridge does not support network capture, attempting browserless fallback');
        dispatch({ type: 'FETCH_START' });

        try {
          const fallbackData = await appService.getFallbackDiagnostics(identifier, activePreviewUrl);

          if (fallbackData && fallbackData.networkRequests) {
            const entries: ReportNetworkEntry[] = fallbackData.networkRequests.map(req => {
              const ts = new Date(req.timestamp).getTime();
              const timestamp = new Date(req.timestamp).toLocaleTimeString();
              const method = req.method.toUpperCase();
              const statusLabel = req.status ? `HTTP ${req.status}` : req.failed ? 'Failed' : 'Pending';
              const durationLabel = req.duration ? `${req.duration}ms` : null;
              const display = `${timestamp} [${method}] ${req.url} → ${statusLabel}${durationLabel ? ` (${durationLabel})` : ''}`.trim();

              return {
                display,
                payload: {
                  ts,
                  kind: 'fetch' as const,
                  method: req.method,
                  url: req.url,
                  status: req.status,
                  ok: req.ok,
                  durationMs: req.duration,
                  error: req.error,
                  requestId: req.requestId,
                },
                timestamp,
                method: req.method,
                statusLabel: req.status ? `${req.status}` : req.failed ? 'Failed' : 'Pending',
                durationLabel,
                errorText: req.error || null,
              };
            });

            dispatch({
              type: 'FETCH_SUCCESS',
              payload: {
                events: entries,
                total: fallbackData.networkRequests.length,
                fetchedAt: new Date(fallbackData.capturedAt).getTime(),
              },
            });

            fromFallbackRef.current = true;
            fetchedForRef.current = normalizedIdentifier;
            logger.info(`Successfully retrieved ${entries.length} network requests via browserless fallback`);
            return;
          }
        } catch (error) {
          logger.warn('Browserless fallback failed', error);
        }
      }

      dispatch({ type: 'FETCH_ERROR', payload: 'Network capture is not available for this preview.' });
      fetchedForRef.current = null;
      fromFallbackRef.current = false;
      return;
    }

    fromFallbackRef.current = false;
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
    fromFallbackRef.current = false;
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
    fromFallback: fromFallbackRef.current,
  };
}
