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

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

const isString = (value: unknown): value is string => typeof value === 'string';
const isNumber = (value: unknown): value is number => typeof value === 'number';
const isBoolean = (value: unknown): value is boolean => typeof value === 'boolean';

const parseJson = async (response: Response): Promise<unknown> => {
  try {
    return await response.json();
  } catch {
    return null;
  }
};

const extractMessage = (payload: unknown, fallback: string): string => {
  if (isRecord(payload) && typeof payload.message === 'string') {
    return payload.message;
  }
  return fallback;
};

const isDiagnosticRunResponse = (value: unknown): value is DiagnosticRunResponse => {
  if (!isRecord(value)) return false;
  if (!isString(value.started_at)) return false;
  if (!isString(value.completed_at)) return false;
  if (!isNumber(value.duration_ms)) return false;
  if (!isRecord(value.results)) return false;
  return true;
};

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

  const payload = await parseJson(response);

  if (!response.ok) {
    throw new Error(extractMessage(payload, `Failed to run diagnostics: ${response.statusText}`));
  }

  if (!isDiagnosticRunResponse(payload)) {
    throw new Error('Invalid diagnostics response');
  }

  return payload;
}

async function refreshCache(): Promise<void> {
  const config = await getConfig();

  const response = await fetch(`${config.API_URL}/observability/refresh`, {
    method: 'POST',
  });

  if (!response.ok) {
    const payload = await parseJson(response);
    throw new Error(extractMessage(payload, `Failed to refresh cache: ${response.statusText}`));
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

const isCleanupRunResponse = (value: unknown): value is CleanupRunResponse => {
  if (!isRecord(value)) return false;
  if (!isBoolean(value.success)) return false;
  if (!isNumber(value.cleaned_up)) return false;
  if (!isNumber(value.remaining_sessions)) return false;
  if (!isString(value.started_at)) return false;
  if (!isString(value.completed_at)) return false;
  if (!isNumber(value.duration_ms)) return false;
  return true;
};

async function runCleanupRequest(): Promise<CleanupRunResponse> {
  const config = await getConfig();

  const response = await fetch(`${config.API_URL}/observability/cleanup/run`, {
    method: 'POST',
  });

  const payload = await parseJson(response);

  if (!response.ok) {
    throw new Error(extractMessage(payload, `Failed to run cleanup: ${response.statusText}`));
  }

  if (!isCleanupRunResponse(payload)) {
    throw new Error('Invalid cleanup response');
  }

  return payload;
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

const isSessionInfo = (value: unknown): value is SessionInfo => {
  if (!isRecord(value)) return false;
  if (!isString(value.id)) return false;
  if (!isString(value.phase)) return false;
  if (!isString(value.created_at)) return false;
  if (!isString(value.last_used_at)) return false;
  if (!isNumber(value.idle_time_ms)) return false;
  if (!isBoolean(value.is_idle)) return false;
  if (!isBoolean(value.is_recording)) return false;
  if (!isNumber(value.instruction_count)) return false;
  if (!isNumber(value.page_count)) return false;
  if (value.workflow_id !== undefined && !isString(value.workflow_id)) return false;
  if (value.current_url !== undefined && value.current_url !== null && !isString(value.current_url)) return false;
  return true;
};

const isSessionListResponse = (value: unknown): value is SessionListResponse => {
  if (!isRecord(value)) return false;
  if (!Array.isArray(value.sessions) || !value.sessions.every(isSessionInfo)) return false;
  if (!isRecord(value.summary)) return false;
  if (!isNumber(value.summary.total)) return false;
  if (!isNumber(value.summary.active)) return false;
  if (!isNumber(value.summary.idle)) return false;
  if (!isNumber(value.summary.active_recordings)) return false;
  if (!isNumber(value.summary.capacity)) return false;
  if (!isString(value.timestamp)) return false;
  return true;
};

async function fetchSessionList(): Promise<SessionListResponse> {
  const config = await getConfig();

  const response = await fetch(`${config.API_URL}/observability/sessions`);

  const payload = await parseJson(response);

  if (!response.ok) {
    throw new Error(extractMessage(payload, `Failed to fetch sessions: ${response.statusText}`));
  }

  if (!isSessionListResponse(payload)) {
    throw new Error('Invalid sessions response');
  }

  return payload;
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
