import { useCallback, useEffect, useRef, useState } from 'react';
import { usePolling } from '../../../shared/hooks/usePolling';
import { extractErrorMessage } from '../../../shared/api/apiFetch';
import { fetchForensicsSummary } from '../api';
import type { ForensicsSummary } from '../types';

export interface UseForensicsSummaryResult {
  summary: ForensicsSummary | null;
  isLoading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
}

/**
 * Polls /api/v1/forensics/summary every 60 seconds.
 * Uses a single in-flight AbortController to prevent overlapping fetches.
 */
export function useForensicsSummary(intervalMs = 60_000): UseForensicsSummaryResult {
  const [summary, setSummary] = useState<ForensicsSummary | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const refresh = useCallback(async () => {
    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;
    setIsLoading(true);
    try {
      const data = await fetchForensicsSummary(controller.signal);
      if (controller.signal.aborted) return;
      setSummary(data);
      setError(null);
    } catch (err) {
      if (err instanceof DOMException && err.name === 'AbortError') return;
      setError(extractErrorMessage(err, 'Failed to load forensics summary'));
    } finally {
      if (!controller.signal.aborted) setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
    return () => abortRef.current?.abort();
  }, [refresh]);

  usePolling(refresh, intervalMs, true);

  return { summary, isLoading, error, refresh };
}
