/**
 * Hook for managing health checks state in the report dialog
 */

import { useCallback, useMemo, useRef, useState } from 'react';
import type { App } from '@/types';
import { healthService } from '@/services/api';
import { logger } from '@/services/logger';
import type { ReportHealthCheckEntry } from './reportTypes';
import { formatOptionalTimestamp } from './reportFormatters';

interface UseReportHealthDataParams {
  app: App | null;
  appId?: string;
}

export interface ReportHealthDataState {
  includeHealthChecks: boolean;
  setIncludeHealthChecks: (value: boolean) => void;
  entries: ReportHealthCheckEntry[];
  expanded: boolean;
  toggleExpanded: () => void;
  loading: boolean;
  error: string | null;
  formattedCapturedAt: string | null;
  total: number | null;
  fetch: (options?: { force?: boolean }) => Promise<void>;
  reset: () => void;
}

/**
 * Hook for managing health checks data fetching and state
 */
export function useReportHealthData({
  app,
  appId,
}: UseReportHealthDataParams): ReportHealthDataState {
  const [entries, setEntries] = useState<ReportHealthCheckEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState(false);
  const [fetchedAt, setFetchedAt] = useState<number | null>(null);
  const [total, setTotal] = useState<number | null>(null);
  const [include, setInclude] = useState(true);

  const fetchedForRef = useRef<string | null>(null);

  const fetch = useCallback(async (options?: { force?: boolean }) => {
    if (loading && !options?.force) {
      return;
    }

    const targetAppId = (app?.id ?? appId ?? '').trim();
    if (!targetAppId) {
      setEntries([]);
      setTotal(null);
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
      const result = await healthService.checkPreviewHealth(targetAppId);
      const healthEntries = result.checks.map((entry, index): ReportHealthCheckEntry => {
        const rawId = typeof entry.id === 'string' ? entry.id.trim() : '';
        const id = rawId || `health-${index}`;
        const rawName = typeof entry.name === 'string' ? entry.name.trim() : '';
        const status: 'pass' | 'warn' | 'fail' = entry.status === 'pass'
          ? 'pass'
          : entry.status === 'warn'
            ? 'warn'
            : 'fail';
        const latency = typeof entry.latencyMs === 'number' && Number.isFinite(entry.latencyMs)
          ? Math.max(0, Math.round(entry.latencyMs))
          : null;

        return {
          id,
          name: rawName || id,
          status,
          endpoint: entry.endpoint ?? null,
          latencyMs: latency,
          message: entry.message ?? null,
          code: entry.code ?? null,
          response: entry.response ?? null,
        };
      });

      setEntries(healthEntries);
      setTotal(healthEntries.length);

      let capturedAtMs: number | null = null;
      if (result.capturedAt) {
        const parsed = Date.parse(result.capturedAt);
        if (!Number.isNaN(parsed)) {
          capturedAtMs = parsed;
        }
      }
      const nextFetchedAt = healthEntries.length > 0
        ? capturedAtMs ?? Date.now()
        : null;
      setFetchedAt(nextFetchedAt);
      fetchedForRef.current = normalizedIdentifier;

      const combinedErrors = result.errors.filter(message => typeof message === 'string' && message.trim().length > 0);
      const errorMessage = combinedErrors.length > 0 ? combinedErrors.join(' ') : null;

      if (healthEntries.length === 0) {
        setError(errorMessage ?? 'No health checks available.');
      } else {
        setError(errorMessage ?? null);
      }
    } catch (err) {
      logger.warn('Failed to gather preview health checks for issue report', err);
      setEntries([]);
      setTotal(null);
      setFetchedAt(null);
      fetchedForRef.current = null;
      const message = err instanceof Error && err.message
        ? err.message
        : 'Unable to gather health checks.';
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [app?.id, appId, loading]);

  const reset = useCallback(() => {
    setEntries([]);
    setLoading(false);
    setError(null);
    setExpanded(false);
    setFetchedAt(null);
    setTotal(null);
    setInclude(true);
    fetchedForRef.current = null;
  }, []);

  const formattedCapturedAt = useMemo(
    () => formatOptionalTimestamp(fetchedAt),
    [fetchedAt],
  );

  return {
    includeHealthChecks: include,
    setIncludeHealthChecks: setInclude,
    entries,
    expanded,
    toggleExpanded: () => setExpanded(prev => !prev),
    loading,
    error,
    formattedCapturedAt,
    total,
    fetch,
    reset,
  };
}
