/**
 * Hook for managing app logs state in the report dialog
 */

import { useCallback, useMemo, useReducer, useRef } from 'react';
import type { App } from '@/types';
import { appService } from '@/services/api';
import { logger } from '@/services/logger';
import type { ReportAppLogStream } from './reportTypes';
import { isTruncated } from './reportTypes';
import { REPORT_APP_LOGS_MAX_LINES } from './reportConstants';
import { formatOptionalTimestamp } from './reportFormatters';
import {
  reportLogsReducer,
  initialReportLogsState,
} from './reportLogsReducer';

interface UseReportLogsDataParams {
  app: App | null;
  appId?: string;
}

export interface ReportLogsDataState {
  includeAppLogs: boolean;
  setIncludeAppLogs: (value: boolean) => void;
  streams: ReportAppLogStream[];
  selections: Record<string, boolean>;
  toggleStream: (key: string, checked: boolean) => void;
  expanded: boolean;
  toggleExpanded: () => void;
  loading: boolean;
  error: string | null;
  truncated: boolean;
  formattedCapturedAt: string | null;
  logs: string[];
  total: number | null;
  fetch: (options?: { force?: boolean }) => Promise<void>;
  reset: () => void;
  selectedCount: number;
  totalStreamCount: number;
  buildPayload: () => {
    logs?: string[];
    logsTotal?: number;
    logsCapturedAt?: string;
  };
}

/**
 * Hook for managing app logs data fetching and state
 */
export function useReportLogsData({
  app,
  appId,
}: UseReportLogsDataParams): ReportLogsDataState {
  const [state, dispatch] = useReducer(reportLogsReducer, initialReportLogsState);
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
      dispatch({ type: 'FETCH_ERROR', payload: 'Unable to determine which logs to include.' });
      fetchedForRef.current = null;
      return;
    }

    const normalizedIdentifier = identifier.toLowerCase();
    if (!options?.force && fetchedForRef.current === normalizedIdentifier) {
      return;
    }

    dispatch({ type: 'FETCH_START' });

    try {
      const result = await appService.getAppLogs(identifier, 'both');
      const rawLogs = Array.isArray(result.logs) ? result.logs : [];
      const trimmedLogs = rawLogs.slice(-REPORT_APP_LOGS_MAX_LINES);

      const streams = Array.isArray(result.streams) ? result.streams : [];
      const normalizedStreams: ReportAppLogStream[] = streams.map((stream) => {
        const safeLines = Array.isArray(stream.lines) ? stream.lines : [];
        const trimmedLines = safeLines.slice(-REPORT_APP_LOGS_MAX_LINES);
        let label = stream.label || stream.key;
        if (!label) {
          label = stream.type === 'lifecycle' ? 'Lifecycle' : stream.step || 'Background';
        }
        if (stream.type === 'lifecycle') {
          label = 'Lifecycle';
        }
        return {
          key: stream.key,
          label,
          type: stream.type,
          lines: trimmedLines,
          total: safeLines.length,
          command: stream.command,
        };
      });

      const selections = normalizedStreams.reduce<Record<string, boolean>>((acc, stream) => {
        const prior = state.selections[stream.key];
        acc[stream.key] = prior !== undefined ? prior : true;
        return acc;
      }, {});

      dispatch({
        type: 'FETCH_SUCCESS',
        payload: {
          logs: trimmedLogs,
          logsTotal: rawLogs.length,
          streams: normalizedStreams,
          selections,
          fetchedAt: Date.now(),
        },
      });
      fetchedForRef.current = normalizedIdentifier;
    } catch (error) {
      logger.error('Failed to load logs for issue report', error);
      dispatch({ type: 'FETCH_ERROR', payload: 'Unable to load logs. Try again.' });
      fetchedForRef.current = null;
    }
  }, [resolveIdentifier, state.selections]);

  const reset = useCallback(() => {
    dispatch({ type: 'RESET' });
    fetchedForRef.current = null;
  }, []);

  const handleToggleStream = useCallback((key: string, checked: boolean) => {
    dispatch({ type: 'TOGGLE_STREAM', payload: { key, checked } });
  }, []);

  const formattedCapturedAt = useMemo(
    () => formatOptionalTimestamp(state.fetchedAt),
    [state.fetchedAt],
  );

  const truncated = useMemo(
    () => isTruncated(state.logsTotal, state.logs.length),
    [state.logsTotal, state.logs.length],
  );

  const selectedStreams = useMemo(
    () => state.streams.filter(stream => state.selections[stream.key] !== false),
    [state.selections, state.streams],
  );

  const selectedCount = selectedStreams.length;
  const totalStreamCount = state.streams.length;

  const buildPayload = useCallback(() => {
    const result: {
      logs?: string[];
      logsTotal?: number;
      logsCapturedAt?: string;
    } = {};

    if (!state.include) {
      return result;
    }

    if (selectedStreams.length > 0) {
      const combinedLogs: string[] = [];

      selectedStreams.forEach((stream) => {
        if (combinedLogs.length >= REPORT_APP_LOGS_MAX_LINES) {
          return;
        }

        const contextLabel = stream.type === 'background'
          ? `Background: ${stream.label}`
          : 'Lifecycle';
        combinedLogs.push(`--- ${contextLabel} ---`);

        if (stream.type === 'background' && stream.command && combinedLogs.length < REPORT_APP_LOGS_MAX_LINES) {
          combinedLogs.push(`# ${stream.command}`);
        }

        const remaining = REPORT_APP_LOGS_MAX_LINES - combinedLogs.length;
        if (remaining > 0) {
          const tail = stream.lines.slice(-remaining);
          combinedLogs.push(...tail);
        }
      });

      result.logs = combinedLogs;
      result.logsTotal = selectedStreams.reduce((total, stream) => total + stream.total, 0);
      if (state.fetchedAt) {
        result.logsCapturedAt = new Date(state.fetchedAt).toISOString();
      }
    } else if (state.logs.length > 0) {
      result.logs = state.logs;
      result.logsTotal = typeof state.logsTotal === 'number' ? state.logsTotal : state.logs.length;
      if (state.fetchedAt) {
        result.logsCapturedAt = new Date(state.fetchedAt).toISOString();
      }
    }

    return result;
  }, [state.include, state.logs, state.logsTotal, state.fetchedAt, selectedStreams]);

  return {
    includeAppLogs: state.include,
    setIncludeAppLogs: (value: boolean) => dispatch({ type: 'SET_INCLUDE', payload: value }),
    streams: state.streams,
    selections: state.selections,
    toggleStream: handleToggleStream,
    expanded: state.expanded,
    toggleExpanded: () => dispatch({ type: 'SET_EXPANDED', payload: !state.expanded }),
    loading: state.loading,
    error: state.error,
    truncated,
    formattedCapturedAt,
    logs: state.logs,
    total: state.logsTotal,
    fetch,
    reset,
    selectedCount,
    totalStreamCount,
    buildPayload,
  };
}
