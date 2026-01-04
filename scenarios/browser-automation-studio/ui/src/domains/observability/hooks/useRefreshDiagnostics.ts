/**
 * useRefreshDiagnostics Hook
 *
 * Triggers manual diagnostic runs against the playwright-driver.
 * Used for deep diagnostic scans that are too expensive to run continuously.
 */

import { useMutation, useQueryClient } from '@tanstack/react-query';
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

export default useRefreshDiagnostics;
