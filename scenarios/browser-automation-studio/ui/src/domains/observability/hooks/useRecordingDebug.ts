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

      if (!isRecord(payload)) {
        throw new Error('Invalid debug response');
      }

      const result = payload as RecordingDebugResponse;
      setData(result);
      return result;
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
