import { useEffect, useState, useCallback, useRef } from 'react';
import { appService } from '@/services/api';
import type { CompletenessScore } from '@/types';
import { logger } from '@/services/logger';

interface UseAppCompletenessOptions {
  enabled?: boolean; // Whether to fetch automatically
  refetchOnOpen?: boolean; // Refetch when enabled changes to true
}

interface UseAppCompletenessResult {
  completeness: CompletenessScore | null;
  loading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching completeness score for an app.
 * Automatically fetches when enabled and caches the result.
 */
export function useAppCompleteness(
  appId: string | null | undefined,
  options: UseAppCompletenessOptions = {}
): UseAppCompletenessResult {
  const { enabled = true, refetchOnOpen = true } = options;

  const [completeness, setCompleteness] = useState<CompletenessScore | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Track if we've fetched for this app already
  const fetchedForAppRef = useRef<string | null>(null);
  const previousEnabledRef = useRef(enabled);

  const fetchCompleteness = useCallback(async () => {
    if (!appId) {
      setCompleteness(null);
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      logger.info(`Fetching completeness score for ${appId}`);
      const result = await appService.getAppCompleteness(appId);

      if (result) {
        setCompleteness(result);
        setError(null);
        fetchedForAppRef.current = appId;
      } else {
        setError('Failed to fetch completeness score');
        logger.warn(`No completeness data returned for ${appId}`);
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(message);
      logger.error(`Error fetching completeness for ${appId}`, err);
    } finally {
      setLoading(false);
    }
  }, [appId]);

  useEffect(() => {
    // Clear completeness when appId changes
    if (appId !== fetchedForAppRef.current) {
      setCompleteness(null);
      setError(null);
      fetchedForAppRef.current = null;
    }

    // Don't fetch if disabled
    if (!enabled) {
      previousEnabledRef.current = enabled;
      return;
    }

    // Check if we should fetch
    const shouldFetch =
      appId && // Has an app ID
      !completeness && // No existing completeness data
      !loading && // Not currently loading
      (fetchedForAppRef.current !== appId || // Haven't fetched for this app
        (refetchOnOpen && !previousEnabledRef.current && enabled)); // Or refetch on open

    if (shouldFetch) {
      fetchCompleteness();
    }

    previousEnabledRef.current = enabled;
  }, [appId, enabled, completeness, loading, fetchCompleteness, refetchOnOpen]);

  return {
    completeness,
    loading,
    error,
    refetch: fetchCompleteness,
  };
}
