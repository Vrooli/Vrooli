/**
 * useRecordingDebug Hook
 *
 * Fetches live recording debug info for a specific session.
 * Used for real-time debugging during active recording sessions.
 */

import { useState, useCallback } from 'react';
import { getConfig } from '@/config';
import { logger } from '@/utils/logger';

export interface BrowserScriptState {
  loaded: boolean;
  ready?: boolean;
  isActive?: boolean | null;
  inMainContext?: boolean;
  handlersCount?: number;
  version?: string | null;
  eventsDetected?: number;
  eventsCaptured?: number;
  eventsSent?: number;
  eventsSendFailed?: number;
  lastError?: string | null;
  serviceWorkerActive?: boolean;
  serviceWorkerUrl?: string | null;
}

export interface RecordingDebugResponse {
  session_id: string;
  timestamp: string;
  server: {
    is_recording: boolean;
    recording_id: string | null;
    has_event_handler: boolean;
    phase: string;
    current_url: string | null;
  };
  route_handler: {
    events_received: number;
    events_processed: number;
    events_dropped_no_handler: number;
    events_with_errors: number;
    last_event_at: string | null;
    last_event_type: string | null;
  } | null;
  injection: {
    attempted: number;
    successful: number;
    failed: number;
    skipped: number;
  } | null;
  browser_script: BrowserScriptState | null;
  diagnostics: {
    script_not_loaded: boolean;
    script_not_ready: boolean;
    script_not_in_main: boolean;
    script_inactive: boolean;
    no_handlers: boolean;
    no_event_handler: boolean;
    events_being_dropped: boolean;
    service_worker_blocking: boolean;
  };
}

interface UseRecordingDebugReturn {
  /** Fetch debug info for a session */
  fetchDebug: (sessionId: string) => Promise<RecordingDebugResponse>;
  /** Whether a debug request is in progress */
  isLoading: boolean;
  /** The last debug response */
  data: RecordingDebugResponse | null;
  /** Any error from the last request */
  error: Error | null;
  /** Reset the state */
  reset: () => void;
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

const isString = (value: unknown): value is string => typeof value === 'string';
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

const isServerState = (value: unknown): value is RecordingDebugResponse['server'] => {
  if (!isRecord(value)) return false;
  if (!isBoolean(value.is_recording)) return false;
  if (value.recording_id !== null && value.recording_id !== undefined && !isString(value.recording_id)) return false;
  if (!isBoolean(value.has_event_handler)) return false;
  if (!isString(value.phase)) return false;
  if (value.current_url !== null && value.current_url !== undefined && !isString(value.current_url)) return false;
  return true;
};

const isDiagnostics = (value: unknown): value is RecordingDebugResponse['diagnostics'] => {
  if (!isRecord(value)) return false;
  return (
    isBoolean(value.script_not_loaded) &&
    isBoolean(value.script_not_ready) &&
    isBoolean(value.script_not_in_main) &&
    isBoolean(value.script_inactive) &&
    isBoolean(value.no_handlers) &&
    isBoolean(value.no_event_handler) &&
    isBoolean(value.events_being_dropped) &&
    isBoolean(value.service_worker_blocking)
  );
};

const isRecordingDebugResponse = (value: unknown): value is RecordingDebugResponse => {
  if (!isRecord(value)) return false;
  if (!isString(value.session_id)) return false;
  if (!isString(value.timestamp)) return false;
  if (!isServerState(value.server)) return false;
  if (!isDiagnostics(value.diagnostics)) return false;
  if (value.route_handler !== null && value.route_handler !== undefined && !isRecord(value.route_handler)) return false;
  if (value.injection !== null && value.injection !== undefined && !isRecord(value.injection)) return false;
  if (value.browser_script !== null && value.browser_script !== undefined && !isRecord(value.browser_script)) return false;
  return true;
};

export function useRecordingDebug(): UseRecordingDebugReturn {
  const [isLoading, setIsLoading] = useState(false);
  const [data, setData] = useState<RecordingDebugResponse | null>(null);
  const [error, setError] = useState<Error | null>(null);

  const fetchDebug = useCallback(async (sessionId: string): Promise<RecordingDebugResponse> => {
    setIsLoading(true);
    setError(null);

    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/recordings/live/${sessionId}/debug`);

      const payload = await parseJson(response);

      if (!response.ok) {
        throw new Error(extractMessage(payload, `Failed to fetch debug info: ${response.statusText}`));
      }

      if (!isRecordingDebugResponse(payload)) {
        throw new Error('Invalid debug response');
      }

      setData(payload);
      return payload;
    } catch (err: unknown) {
      const error = err instanceof Error ? err : new Error(String(err));
      logger.error('Failed to fetch recording debug info', { component: 'useRecordingDebug' }, error);
      setError(error);
      throw error;
    } finally {
      setIsLoading(false);
    }
  }, []);

  const reset = useCallback(() => {
    setData(null);
    setError(null);
    setIsLoading(false);
  }, []);

  return {
    fetchDebug,
    isLoading,
    data,
    error,
    reset,
  };
}

export default useRecordingDebug;
