/**
 * Hook for checking recorded video availability for an execution.
 *
 * This hook isolates the recorded video status concern, making it:
 * - Independently testable
 * - Reusable across components
 * - Clear in its single responsibility
 */

import { useCallback, useEffect, useRef, useState } from "react";
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
  url: string | null;

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

const EMPTY_STATUS: RecordedVideoStatus = { available: false, count: 0, videos: [] };
interface RecordedVideoSnapshot {
  executionId: string;
  status: RecordedVideoStatus;
  loading: boolean;
  error: string | null;
}

// =============================================================================
// Hook Implementation
// =============================================================================

export function useRecordedVideoStatus({
  executionId,
  apiClient = defaultExportApiClient,
}: UseRecordedVideoStatusOptions): UseRecordedVideoStatusResult {
  const [snapshot, setSnapshot] = useState<RecordedVideoSnapshot>({ executionId, status: EMPTY_STATUS, loading: false, error: null });
  const currentExecution = useRef(executionId);
  currentExecution.current = executionId;
  const requestRef = useRef<{ generation: number; controller: AbortController | null }>({ generation: 0, controller: null });

  const cancelRequest = useCallback(() => {
    requestRef.current.controller?.abort();
    requestRef.current = { generation: requestRef.current.generation + 1, controller: null };
  }, []);

  const fetchStatus = useCallback(async () => {
    // A retained callback from an old execution cannot retire its successor.
    if (currentExecution.current !== executionId) return;
    cancelRequest();
    const controller = new AbortController();
    const generation = requestRef.current.generation;
    requestRef.current.controller = controller;
    const isCurrent = () => requestRef.current.generation === generation
      && currentExecution.current === executionId && !controller.signal.aborted;
    if (!executionId.trim()) {
      setSnapshot({ executionId, status: EMPTY_STATUS, loading: false, error: null });
      return;
    }
    setSnapshot(previous => ({ executionId, status: previous.executionId === executionId ? previous.status : EMPTY_STATUS, loading: true, error: null }));
    try {
      const status = await apiClient.fetchRecordedVideoStatus(executionId, controller.signal);
      if (isCurrent()) setSnapshot({ executionId, status, loading: false, error: null });
    } catch (error) {
      if (!isCurrent() || (error instanceof DOMException && error.name === "AbortError")) return;
      const message = error instanceof Error ? error.message : "Failed to check recorded video status";
      setSnapshot({ executionId, status: EMPTY_STATUS, loading: false, error: message });
    } finally {
      // Old transports may resolve even after abort, or return an abort failure.
      // Their finalizers must not clear a successor's loading state.
      if (isCurrent()) setSnapshot(previous => ({ ...previous, loading: false }));
    }
  }, [executionId, apiClient, cancelRequest]);

  useEffect(() => {
    void fetchStatus();
    return cancelRequest;
  }, [fetchStatus, cancelRequest]);

  const refresh = useCallback(() => { void fetchStatus(); }, [fetchStatus]);
  const current = snapshot.executionId === executionId;
  const status = current ? snapshot.status : EMPTY_STATUS;
  return {
    available: status.available,
    url: status.videos.find(video => video.url)?.url ?? null,
    count: status.count,
    loading: current ? snapshot.loading : Boolean(executionId.trim()),
    error: current ? snapshot.error : null,
    refresh,
  };
}
