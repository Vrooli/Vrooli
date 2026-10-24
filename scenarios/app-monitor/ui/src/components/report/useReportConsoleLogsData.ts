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
}

/**
 * Hook for managing console logs data fetching and state from the bridge
 */
export function useReportConsoleLogsData({
  app,
  appId,
  bridgeSupported,
  bridgeCaps,
  logState,
  configureLogs,
  getRecentLogs,
  requestLogBatch,
}: UseReportConsoleLogsDataParams): ReportConsoleLogsDataState {
  const [state, dispatch] = useReducer(reportConsoleLogsReducer, initialReportConsoleLogsState);
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
      dispatch({ type: 'FETCH_ERROR', payload: 'Unable to determine which console logs to include.' });
      fetchedForRef.current = null;
      return;
    }

    const normalizedIdentifier = identifier.toLowerCase();
    if (!options?.force && fetchedForRef.current === normalizedIdentifier) {
      return;
    }

    // The iframe bridge is the only console-log source. When it does not
    // advertise log support, degrade to a clear, informative empty state.
    if (!bridgeSupported || !bridgeCaps.includes('logs')) {
      dispatch({
        type: 'FETCH_ERROR',
        payload: "Console logs unavailable — the preview's iframe bridge did not advertise log support.",
      });
      fetchedForRef.current = null;
      return;
    }

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
  };
}
