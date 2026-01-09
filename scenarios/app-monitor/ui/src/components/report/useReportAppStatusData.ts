/**
 * Hook for managing app status state in the report dialog
 */

import { useCallback, useMemo, useRef, useState } from 'react';
import type { App } from '@/types';
import { appService, type ReportIssueAppStatusSeverity } from '@/services/api';
import { logger } from '@/services/logger';
import type { ReportAppStatusSnapshot } from './reportTypes';
import { formatOptionalTimestamp } from './reportFormatters';

interface UseReportAppStatusDataParams {
  app: App | null;
  appId?: string;
}

export interface ReportAppStatusDataState {
  includeAppStatus: boolean;
  setIncludeAppStatus: (value: boolean) => void;
  snapshot: ReportAppStatusSnapshot | null;
  expanded: boolean;
  toggleExpanded: () => void;
  loading: boolean;
  error: string | null;
  formattedCapturedAt: string | null;
  fetch: (options?: { force?: boolean }) => Promise<void>;
  reset: () => void;
}

/**
 * Hook for managing app status data fetching and state
 */
export function useReportAppStatusData({
  app,
  appId,
}: UseReportAppStatusDataParams): ReportAppStatusDataState {
  const [snapshot, setSnapshot] = useState<ReportAppStatusSnapshot | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState(false);
  const [fetchedAt, setFetchedAt] = useState<number | null>(null);
  const [include, setInclude] = useState(true);

  const fetchedForRef = useRef<string | null>(null);

  const fetch = useCallback(async (options?: { force?: boolean }) => {
    if (loading && !options?.force) {
      return;
    }

    const targetAppId = (app?.id ?? appId ?? '').trim();
    if (!targetAppId) {
      setSnapshot(null);
      setError('Preview app unavailable.');
      setFetchedAt(null);
      fetchedForRef.current = null;
      return;
    }

    const normalizedIdentifier = targetAppId.toLowerCase();
    if (!options?.force && fetchedForRef.current === normalizedIdentifier) {
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const result = await appService.getAppStatusSnapshot(targetAppId);
      if (!result) {
        setSnapshot(null);
        setError('Unable to retrieve scenario status.');
        setFetchedAt(null);
        fetchedForRef.current = null;
        return;
      }

      const runtimeValue = typeof result.runtime === 'string' ? result.runtime.trim() : '';
      const statusSnapshot: ReportAppStatusSnapshot = {
        appId: (result.appId ?? targetAppId).trim() || targetAppId,
        scenario: (result.scenario ?? targetAppId).trim() || targetAppId,
        statusLabel: (result.statusLabel ?? 'UNKNOWN').trim() || 'UNKNOWN',
        severity: (result.severity ?? 'warn') as ReportIssueAppStatusSeverity,
        runtime: runtimeValue || null,
        processCount: typeof result.processCount === 'number' ? result.processCount : null,
        details: Array.isArray(result.details)
          ? result.details.map(line => (typeof line === 'string' ? line : String(line))).filter(detail => detail.trim().length > 0)
          : [],
        capturedAt: result.capturedAt ?? null,
      };

      setSnapshot(statusSnapshot);
      const capturedAtMs = statusSnapshot.capturedAt ? Date.parse(statusSnapshot.capturedAt) : Number.NaN;
      const nextFetchedAt = Number.isNaN(capturedAtMs) ? Date.now() : capturedAtMs;
      setFetchedAt(nextFetchedAt);
      setError(null);
      fetchedForRef.current = normalizedIdentifier;
    } catch (err) {
      logger.warn('Failed to gather scenario status for issue report', err);
      setSnapshot(null);
      setError('Unable to gather scenario status.');
      setFetchedAt(null);
      fetchedForRef.current = null;
    } finally {
      setLoading(false);
    }
  }, [app?.id, appId, loading]);

  const reset = useCallback(() => {
    setSnapshot(null);
    setLoading(false);
    setError(null);
    setExpanded(false);
    setFetchedAt(null);
    setInclude(true);
    fetchedForRef.current = null;
  }, []);

  const formattedCapturedAt = useMemo(
    () => formatOptionalTimestamp(snapshot?.capturedAt)
      ?? formatOptionalTimestamp(fetchedAt),
    [fetchedAt, snapshot?.capturedAt],
  );

  return {
    includeAppStatus: include,
    setIncludeAppStatus: setInclude,
    snapshot,
    expanded,
    toggleExpanded: () => setExpanded(prev => !prev),
    loading,
    error,
    formattedCapturedAt,
    fetch,
    reset,
  };
}
