/**
 * useRefreshDiagnostics Hook
 *
 * Triggers manual diagnostic runs against the playwright-driver.
 * Used for deep diagnostic scans that are too expensive to run continuously.
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback } from 'react';
import { getConfig } from '@/config';
import { logger } from '@/utils/logger';
import type { DiagnosticRunRequest, DiagnosticRunResponse } from '../types';

const OBSERVABILITY_QUERY_KEY = 'observability';

interface UseRefreshDiagnosticsOptions {
  /**
   * Callback when diagnostics complete successfully
   */
  onSuccess?: (result: DiagnosticRunResponse) => void;

  /**
   * Callback when diagnostics fail
   */
  onError?: (error: Error) => void;
}

interface UseRefreshDiagnosticsReturn {
  /** Run diagnostics with the given options */
  runDiagnostics: (request: DiagnosticRunRequest) => Promise<DiagnosticRunResponse>;

  /** Whether diagnostics are currently running */
  isRunning: boolean;

  /** The result of the last diagnostic run */
  result: DiagnosticRunResponse | undefined;

  /** Any error from the last diagnostic run */
  error: Error | null;

  /** Reset the mutation state */
  reset: () => void;
}

async function runDiagnosticsRequest(request: DiagnosticRunRequest): Promise<DiagnosticRunResponse> {
  const config = await getConfig();

  const response = await fetch(`${config.API_URL}/observability/diagnostics/run`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(request),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.message || `Failed to run diagnostics: ${response.statusText}`);
  }

  return response.json();
}

async function refreshCache(): Promise<void> {
  const config = await getConfig();

  const response = await fetch(`${config.API_URL}/observability/refresh`, {
    method: 'POST',
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.message || `Failed to refresh cache: ${response.statusText}`);
  }
}

export function useRefreshDiagnostics(options: UseRefreshDiagnosticsOptions = {}): UseRefreshDiagnosticsReturn {
  const { onSuccess, onError } = options;
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: runDiagnosticsRequest,
    onSuccess: (result) => {
      // Invalidate observability queries to pick up new data
      queryClient.invalidateQueries({ queryKey: [OBSERVABILITY_QUERY_KEY] });
      onSuccess?.(result);
    },
    onError: (error: Error) => {
      logger.error('Diagnostics run failed', { component: 'useRefreshDiagnostics', action: 'runDiagnostics' }, error);
      onError?.(error);
    },
  });

  const runDiagnostics = useCallback(async (request: DiagnosticRunRequest): Promise<DiagnosticRunResponse> => {
    return mutation.mutateAsync(request);
  }, [mutation]);

  return {
    runDiagnostics,
    isRunning: mutation.isPending,
    result: mutation.data,
    error: mutation.error,
    reset: mutation.reset,
  };
}

/**
 * Simple hook to just refresh the observability cache
 */
export function useRefreshCache() {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: refreshCache,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [OBSERVABILITY_QUERY_KEY] });
    },
    onError: (error: Error) => {
      logger.error('Cache refresh failed', { component: 'useRefreshCache', action: 'refresh' }, error);
    },
  });

  return {
    refresh: mutation.mutate,
    isRefreshing: mutation.isPending,
    error: mutation.error,
  };
}

/**
 * Response from the cleanup run endpoint
 */
export interface CleanupRunResponse {
  success: boolean;
  cleaned_up: number;
  remaining_sessions: number;
  started_at: string;
  completed_at: string;
  duration_ms: number;
}

async function runCleanupRequest(): Promise<CleanupRunResponse> {
  const config = await getConfig();

  const response = await fetch(`${config.API_URL}/observability/cleanup/run`, {
    method: 'POST',
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.message || `Failed to run cleanup: ${response.statusText}`);
  }

  return response.json();
}

/**
 * Hook to trigger manual cleanup of idle sessions
 */
export function useRunCleanup() {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: runCleanupRequest,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [OBSERVABILITY_QUERY_KEY] });
    },
    onError: (error: Error) => {
      logger.error('Cleanup run failed', { component: 'useRunCleanup', action: 'runCleanup' }, error);
    },
  });

  return {
    runCleanup: mutation.mutateAsync,
    isRunning: mutation.isPending,
    result: mutation.data,
    error: mutation.error,
    reset: mutation.reset,
  };
}

/**
 * Session info from the observability endpoint
 */
export interface SessionInfo {
  id: string;
  phase: string;
  created_at: string;
  last_used_at: string;
  idle_time_ms: number;
  is_idle: boolean;
  is_recording: boolean;
  instruction_count: number;
  workflow_id?: string;
  current_url?: string;
  page_count: number;
}

export interface SessionListResponse {
  sessions: SessionInfo[];
  summary: {
    total: number;
    active: number;
    idle: number;
    active_recordings: number;
    capacity: number;
  };
  timestamp: string;
}

async function fetchSessionList(): Promise<SessionListResponse> {
  const config = await getConfig();

  const response = await fetch(`${config.API_URL}/observability/sessions`);

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw new Error(errorData.message || `Failed to fetch sessions: ${response.statusText}`);
  }

  return response.json();
}

/**
 * Hook to fetch the list of active browser sessions
 */
export function useSessionList(options: { enabled?: boolean } = {}) {
  const { enabled = true } = options;

  const query = useQuery({
    queryKey: ['observability-sessions'],
    queryFn: fetchSessionList,
    enabled,
    staleTime: 5000,
    gcTime: 30000,
    retry: 1,
    refetchOnWindowFocus: false,
  });

  return {
    data: query.data,
    isLoading: query.isLoading,
    isFetching: query.isFetching,
    error: query.error,
    refetch: query.refetch,
  };
}

export default useRefreshDiagnostics;
