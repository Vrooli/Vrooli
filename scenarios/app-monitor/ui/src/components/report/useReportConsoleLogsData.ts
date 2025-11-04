/**
 * Hook for managing console logs state in the report dialog
 */

import { useCallback, useMemo, useReducer, useRef } from 'react';
import type {
  BridgeLogEvent,
  BridgeLogLevel,
  BridgeLogStreamState,
} from '@vrooli/iframe-bridge';
import type { App } from '@/types';
import { logger } from '@/services/logger';
import { appService } from '@/services/api';
import type { ReportConsoleEntry } from './reportTypes';
import { isTruncated } from './reportTypes';
import { REPORT_CONSOLE_LOGS_MAX_LINES } from './reportConstants';
import { formatOptionalTimestamp, toConsoleEntry } from './reportFormatters';
import {
  reportConsoleLogsReducer,
  initialReportConsoleLogsState,
} from './reportConsoleLogsReducer';

interface UseReportConsoleLogsDataParams {
  app: App | null;
  appId?: string;
  activePreviewUrl: string | null;
  bridgeSupported: boolean;
  bridgeCaps: string[];
  logState: BridgeLogStreamState | null;
  configureLogs: ((config: { enable?: boolean; streaming?: boolean; levels?: BridgeLogLevel[]; bufferSize?: number }) => boolean) | null;
  getRecentLogs: () => BridgeLogEvent[];
  requestLogBatch: (options?: { since?: number; afterSeq?: number; limit?: number }) => Promise<BridgeLogEvent[]>;
}

export interface ReportConsoleLogsDataState {
  includeConsoleLogs: boolean;
  setIncludeConsoleLogs: (value: boolean) => void;
  entries: ReportConsoleEntry[];
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
  pageStatus: import('@/services/api').BrowserlessFallbackPageStatus | null;
}

/**
 * Hook for managing console logs data fetching and state from the bridge
 */
export function useReportConsoleLogsData({
  app,
  appId,
  activePreviewUrl,
  bridgeSupported,
  bridgeCaps,
  logState,
  configureLogs,
  getRecentLogs,
  requestLogBatch,
}: UseReportConsoleLogsDataParams): ReportConsoleLogsDataState {
  const [state, dispatch] = useReducer(reportConsoleLogsReducer, initialReportConsoleLogsState);
  const fetchedForRef = useRef<string | null>(null);
  const fromFallbackRef = useRef<boolean>(false);
  const pageStatusRef = useRef<import('@/services/api').BrowserlessFallbackPageStatus | null>(null);

  const resolveIdentifier = useCallback(() => {
    const candidates = [app?.scenario_name, app?.id, appId]
      .map(value => (typeof value === 'string' ? value.trim() : ''))
      .filter(Boolean);

    return candidates.length > 0 ? candidates[0] : null;
  }, [app, appId]);

  const fetch = useCallback(async (options?: { force?: boolean }) => {
    const identifier = resolveIdentifier();
    if (!identifier) {
      dispatch({ type: 'FETCH_ERROR', payload: 'Unable to determine which console logs to include.' });
      fetchedForRef.current = null;
      return;
    }

    const normalizedIdentifier = identifier.toLowerCase();
    if (!options?.force && fetchedForRef.current === normalizedIdentifier) {
      return;
    }

    // Try fallback if bridge doesn't support logs
    if (!bridgeSupported || !bridgeCaps.includes('logs')) {
      if (activePreviewUrl) {
        logger.info('Bridge does not support console logs, attempting browserless fallback');
        dispatch({ type: 'FETCH_START' });

        try {
          const fallbackData = await appService.getFallbackDiagnostics(identifier, activePreviewUrl);

          if (fallbackData && fallbackData.consoleLogs) {
            const entries: ReportConsoleEntry[] = fallbackData.consoleLogs.map(log => {
              const ts = new Date(log.timestamp).getTime();
              const timestamp = new Date(log.timestamp).toLocaleTimeString();
              const level = log.level.toUpperCase();
              const display = `${timestamp} [Console/${level}] ${log.message}`.trim();

              return {
                display,
                payload: {
                  ts,
                  level: log.level,
                  source: 'console' as const,
                  text: log.message,
                },
                timestamp,
                source: 'Console',
                severity: log.level === 'error' ? ('error' as const) : log.level === 'warn' ? ('warn' as const) : ('info' as const),
                body: log.message,
              };
            });

            dispatch({
              type: 'FETCH_SUCCESS',
              payload: {
                entries,
                total: fallbackData.consoleLogs.length,
                fetchedAt: new Date(fallbackData.capturedAt).getTime(),
              },
            });

            fromFallbackRef.current = true;
            pageStatusRef.current = fallbackData.pageStatus;
            fetchedForRef.current = normalizedIdentifier;
            logger.info(`Successfully retrieved ${entries.length} console logs via browserless fallback`);
            return;
          }
        } catch (error) {
          logger.warn('Browserless fallback failed', error);
        }
      }

      dispatch({ type: 'FETCH_ERROR', payload: 'Console log capture is not available for this preview.' });
      fetchedForRef.current = null;
      fromFallbackRef.current = false;
      return;
    }

    fromFallbackRef.current = false;

    dispatch({ type: 'FETCH_START' });

    try {
      if (logState && configureLogs && logState.enabled === false) {
        configureLogs({ enable: true });
      }

      let events: BridgeLogEvent[] = [];
      try {
        events = await requestLogBatch({ limit: REPORT_CONSOLE_LOGS_MAX_LINES });
      } catch (error) {
        logger.warn('Console log snapshot failed; using buffered events', error);
        events = getRecentLogs();
      }

      if (!Array.isArray(events)) {
        events = [];
      }

      const limited = events.slice(-REPORT_CONSOLE_LOGS_MAX_LINES);
      const entries = limited.map(toConsoleEntry);

      dispatch({
        type: 'FETCH_SUCCESS',
        payload: {
          entries,
          total: events.length,
          fetchedAt: Date.now(),
        },
      });
      fetchedForRef.current = normalizedIdentifier;
    } catch (error) {
      logger.warn('Failed to load console logs for issue report', error);
      dispatch({ type: 'FETCH_ERROR', payload: 'Unable to load console logs from the preview iframe.' });
      fetchedForRef.current = null;
    }
  }, [
    resolveIdentifier,
    bridgeSupported,
    bridgeCaps,
    logState,
    configureLogs,
    getRecentLogs,
    requestLogBatch,
  ]);

  const reset = useCallback(() => {
    dispatch({ type: 'RESET' });
    fetchedForRef.current = null;
    fromFallbackRef.current = false;
    pageStatusRef.current = null;
  }, []);

  const formattedCapturedAt = useMemo(
    () => formatOptionalTimestamp(state.fetchedAt),
    [state.fetchedAt],
  );

  const truncated = useMemo(
    () => isTruncated(state.total, state.entries.length),
    [state.total, state.entries.length],
  );

  return {
    includeConsoleLogs: state.include,
    setIncludeConsoleLogs: (value: boolean) => dispatch({ type: 'SET_INCLUDE', payload: value }),
    entries: state.entries,
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
    pageStatus: pageStatusRef.current,
  };
}
