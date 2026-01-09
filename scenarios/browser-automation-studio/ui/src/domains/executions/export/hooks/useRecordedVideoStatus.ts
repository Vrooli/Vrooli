/**
 * Hook for checking recorded video availability for an execution.
 *
 * This hook isolates the recorded video status concern, making it:
 * - Independently testable
 * - Reusable across components
 * - Clear in its single responsibility
 */

import { useCallback, useEffect, useState } from "react";
import {
  defaultExportApiClient,
  type ExportApiClient,
  type RecordedVideoStatus,
} from "../api/exportClient";

// =============================================================================
// Types
// =============================================================================

export interface UseRecordedVideoStatusOptions {
  /**
   * The execution ID to check for recorded videos.
   */
  executionId: string;

  /**
   * Optional API client for testing. Defaults to the real implementation.
   */
  apiClient?: ExportApiClient;
}

export interface UseRecordedVideoStatusResult {
  /**
   * Whether recorded video is available for this execution.
   */
  available: boolean;

  /**
   * Number of recorded video artifacts available.
   */
  count: number;

  /**
   * Whether the status check is currently loading.
   */
  loading: boolean;

  /**
   * Error message if the status check failed.
   */
  error: string | null;

  /**
   * Manually refresh the recorded video status.
   */
  refresh: () => void;
}

// =============================================================================
// Hook Implementation
// =============================================================================

export function useRecordedVideoStatus({
  executionId,
  apiClient = defaultExportApiClient,
}: UseRecordedVideoStatusOptions): UseRecordedVideoStatusResult {
  const [status, setStatus] = useState<RecordedVideoStatus>({
    available: false,
    count: 0,
    videos: [],
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchStatus = useCallback(
    async (signal?: AbortSignal) => {
      setLoading(true);
      setError(null);

      try {
        const result = await apiClient.fetchRecordedVideoStatus(executionId, signal);
        setStatus(result);
      } catch (err) {
        if (err instanceof DOMException && err.name === "AbortError") {
          return;
        }
        const message =
          err instanceof Error ? err.message : "Failed to check recorded video status";
        setError(message);
        setStatus({ available: false, count: 0, videos: [] });
      } finally {
        setLoading(false);
      }
    },
    [executionId, apiClient],
  );

  useEffect(() => {
    const abortController = new AbortController();
    void fetchStatus(abortController.signal);

    return () => {
      abortController.abort();
    };
  }, [fetchStatus]);

  const refresh = useCallback(() => {
    void fetchStatus();
  }, [fetchStatus]);

  return {
    available: status.available,
    count: status.count,
    loading,
    error,
    refresh,
  };
}
