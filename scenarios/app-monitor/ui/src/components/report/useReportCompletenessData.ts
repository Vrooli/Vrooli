/**
 * Hook for managing completeness score state in the report dialog
 */

import { useCallback, useMemo, useRef, useState } from 'react';
import type { App, CompletenessScore } from '@/types';
import { appService } from '@/services/api';
import { logger } from '@/services/logger';
import { formatOptionalTimestamp } from './reportFormatters';

interface UseReportCompletenessDataParams {
  app: App | null;
  appId?: string;
}

export interface ReportCompletenessDataState {
  includeCompleteness: boolean;
  setIncludeCompleteness: (value: boolean) => void;
  data: CompletenessScore | null;
  expanded: boolean;
  toggleExpanded: () => void;
  loading: boolean;
  error: string | null;
  formattedCapturedAt: string | null;
  fetch: (options?: { force?: boolean }) => Promise<void>;
  reset: () => void;
}

/**
 * Hook for managing completeness score data fetching and state
 */
export function useReportCompletenessData({
  app,
  appId,
}: UseReportCompletenessDataParams): ReportCompletenessDataState {
  const [data, setData] = useState<CompletenessScore | null>(null);
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
      setData(null);
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
      const result = await appService.getAppCompleteness(targetAppId);

      if (result) {
        setData(result);
        setFetchedAt(Date.now());
        fetchedForRef.current = normalizedIdentifier;
        setError(null);
      } else {
        setData(null);
        setFetchedAt(null);
        fetchedForRef.current = null;
        setError('No completeness data available.');
      }
    } catch (err) {
      logger.warn('Failed to gather completeness score for issue report', err);
      setData(null);
      setFetchedAt(null);
      fetchedForRef.current = null;
      const message = err instanceof Error && err.message
        ? err.message
        : 'Unable to gather completeness score.';
      setError(message);
    } finally {
      setLoading(false);
    }
  }, [app?.id, appId, loading]);

  const reset = useCallback(() => {
    setData(null);
    setLoading(false);
    setError(null);
    setExpanded(false);
    setFetchedAt(null);
    setInclude(true);
    fetchedForRef.current = null;
  }, []);

  const formattedCapturedAt = useMemo(
    () => formatOptionalTimestamp(fetchedAt),
    [fetchedAt],
  );

  return {
    includeCompleteness: include,
    setIncludeCompleteness: setInclude,
    data,
    expanded,
    toggleExpanded: () => setExpanded(prev => !prev),
    loading,
    error,
    formattedCapturedAt,
    fetch,
    reset,
  };
}
