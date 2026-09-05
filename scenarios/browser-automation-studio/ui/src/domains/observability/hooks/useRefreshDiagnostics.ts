/**
 * useRefreshDiagnostics Hook
 *
 * Triggers manual diagnostic runs against the playwright-driver. Used for
 * deep diagnostic scans that are too expensive to run continuously.
 *
 * Transport: Connect-RPC via `observability` client in src/api/observability.ts.
 */

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useCallback } from 'react';
import { observability } from '@/api/observability';
import { logger } from '@/utils/logger';
import type { DiagnosticRunRequest, DiagnosticRunResponse } from '../types';

const OBSERVABILITY_QUERY_KEY = 'observability';

interface UseRefreshDiagnosticsOptions {
  onSuccess?: (result: DiagnosticRunResponse) => void;
  onError?: (error: Error) => void;
}

interface UseRefreshDiagnosticsReturn {
  runDiagnostics: (request: DiagnosticRunRequest) => Promise<DiagnosticRunResponse>;
  isRunning: boolean;
  result: DiagnosticRunResponse | undefined;
  error: Error | null;
  reset: () => void;
}

async function runDiagnosticsRequest(request: DiagnosticRunRequest): Promise<DiagnosticRunResponse> {
  return observability.runDiagnostics<DiagnosticRunResponse>(request);
}

async function refreshCache(): Promise<void> {
  await observability.refresh();
}

export function useRefreshDiagnostics(options: UseRefreshDiagnosticsOptions = {}): UseRefreshDiagnosticsReturn {
  const { onSuccess, onError } = options;
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: runDiagnosticsRequest,
    onSuccess: (result) => {
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
 * Simple hook to just refresh the observability cache.
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

export interface CleanupRunResponse {
  success: boolean;
  cleaned_up: number;
  remaining_sessions: number;
  started_at: string;
  completed_at: string;
  duration_ms: number;
}

async function runCleanupRequest(): Promise<CleanupRunResponse> {
  return observability.runCleanup<CleanupRunResponse>();
}

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
  return observability.listSessions<SessionListResponse>();
}

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
