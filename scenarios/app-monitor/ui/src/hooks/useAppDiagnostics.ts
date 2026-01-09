import { useEffect, useState, useCallback, useRef } from 'react';
import { appService } from '@/services/api';
import type { CompleteDiagnostics } from '@/types';
import { logger } from '@/services/logger';

interface UseAppDiagnosticsOptions {
  enabled?: boolean; // Whether to fetch automatically
  fast?: boolean; // Use fast mode (health + status only)
  refetchOnOpen?: boolean; // Refetch when enabled changes to true
}

interface UseAppDiagnosticsResult {
  diagnostics: CompleteDiagnostics | null;
  loading: boolean;
  error: string | null;
  refetch: () => Promise<void>;
}

/**
 * Hook for fetching complete diagnostics for an app.
 * Automatically fetches when enabled and caches the result.
 */
export function useAppDiagnostics(
  appId: string | null | undefined,
  options: UseAppDiagnosticsOptions = {}
): UseAppDiagnosticsResult {
  const { enabled = true, fast = false, refetchOnOpen = true } = options;

  const [diagnostics, setDiagnostics] = useState<CompleteDiagnostics | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Track if we've fetched for this app already
  const fetchedForAppRef = useRef<string | null>(null);
  const previousEnabledRef = useRef(enabled);

  const fetchDiagnostics = useCallback(async () => {
    if (!appId) {
      setDiagnostics(null);
      setError(null);
      return;
    }

    setLoading(true);
    setError(null);

    try {
      logger.info(`Fetching ${fast ? 'fast' : 'complete'} diagnostics for ${appId}`);
      const result = await appService.getCompleteDiagnostics(appId, fast);

      if (result) {
        setDiagnostics(result);
        setError(null);
        fetchedForAppRef.current = appId;
      } else {
        setError('Failed to fetch diagnostics');
        logger.warn(`No diagnostics returned for ${appId}`);
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Unknown error occurred';
      setError(message);
      logger.error(`Error fetching diagnostics for ${appId}`, err);
    } finally {
      setLoading(false);
    }
  }, [appId, fast]);

  useEffect(() => {
    // Clear diagnostics when appId changes
    if (appId !== fetchedForAppRef.current) {
      setDiagnostics(null);
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
      !diagnostics && // No existing diagnostics
      !loading && // Not currently loading
      (fetchedForAppRef.current !== appId || // Haven't fetched for this app
        (refetchOnOpen && !previousEnabledRef.current && enabled)); // Or refetch on open

    if (shouldFetch) {
      fetchDiagnostics();
    }

    previousEnabledRef.current = enabled;
  }, [appId, enabled, diagnostics, loading, fetchDiagnostics, refetchOnOpen]);

  return {
    diagnostics,
    loading,
    error,
    refetch: fetchDiagnostics,
  };
}
