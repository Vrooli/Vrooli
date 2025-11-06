import { useEffect, useState, useCallback, useRef } from 'react';
import { logger } from '@/services/logger';

interface LighthouseScore {
  performance: number;
  accessibility: number;
  'best-practices': number;
  seo?: number;
}

interface LighthouseReport {
  id: string;
  timestamp: string;
  page_id: string;
  page_label: string;
  url: string;
  viewport: string;
  status: string;
  scores: LighthouseScore;
  failures: Array<{ category: string; score: string; threshold: string; level: string }>;
  warnings: Array<{ category: string; score: string; threshold: string; level: string }>;
  report_url: string;
}

interface TrendPoint {
  timestamp: string;
  score: number;
  page_id: string;
}

export interface LighthouseHistory {
  scenario: string;
  reports: LighthouseReport[];
  trend: {
    performance: TrendPoint[];
    accessibility: TrendPoint[];
    best_practices: TrendPoint[];
    seo: TrendPoint[];
  };
}

interface UseLighthouseHistoryOptions {
  enabled?: boolean; // Whether to fetch automatically
}

interface UseLighthouseHistoryResult {
  history: LighthouseHistory | null;
  loading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching Lighthouse history for an app.
 * Automatically fetches when enabled and caches the result.
 */
export function useLighthouseHistory(
  appId: string | null | undefined,
  options: UseLighthouseHistoryOptions = {}
): UseLighthouseHistoryResult {
  const { enabled = true } = options;

  const [history, setHistory] = useState<LighthouseHistory | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Track if we've attempted to fetch for this app already (successful or failed)
  const fetchedForAppRef = useRef<string | null>(null);

  const fetchHistory = useCallback(async () => {
    if (!appId) {
      setHistory(null);
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      logger.info(`Fetching Lighthouse history for ${appId}`);
      const response = await fetch(`/api/v1/scenarios/${appId}/lighthouse/history`);

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.error || 'Failed to load Lighthouse history');
      }

      const data = await response.json();
      setHistory(data);
      setError(null);
      fetchedForAppRef.current = appId;
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(message);
      logger.warn(`Failed to fetch Lighthouse history for ${appId}:`, err);
      // Mark as fetched even on error to prevent infinite retries
      fetchedForAppRef.current = appId;
    } finally {
      setLoading(false);
    }
  }, [appId]);

  useEffect(() => {
    // Clear history when appId changes
    if (appId !== fetchedForAppRef.current) {
      setHistory(null);
      setError(null);
      fetchedForAppRef.current = null;
    }

    // Don't fetch if disabled
    if (!enabled) {
      return;
    }

    // Check if we should fetch
    const shouldFetch =
      appId && // Has an app ID
      !loading && // Not currently loading
      fetchedForAppRef.current !== appId; // Haven't attempted to fetch for this app

    if (shouldFetch) {
      fetchHistory();
    }
  }, [appId, enabled, loading, fetchHistory]);

  return {
    history,
    loading,
    error,
    refetch: fetchHistory,
  };
}
