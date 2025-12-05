/**
 * useRecordMode Hook
 *
 * Manages recording state and API interactions for Record Mode.
 */

import { useState, useCallback, useRef, useEffect } from 'react';
import { getApiBaseUrl } from '@/lib/api';
import type {
  RecordedAction,
  StartRecordingResponse,
  StopRecordingResponse,
  GetActionsResponse,
  GenerateWorkflowResponse,
  SelectorValidation,
} from '../types';

interface UseRecordModeOptions {
  /** Session ID for the browser session */
  sessionId: string;
  /** Polling interval for actions in ms (0 to disable) */
  pollInterval?: number;
  /** Callback when new actions are received */
  onActionsReceived?: (actions: RecordedAction[]) => void;
}

interface UseRecordModeReturn {
  /** Whether recording is currently active */
  isRecording: boolean;
  /** Current recording ID */
  recordingId: string | null;
  /** List of recorded actions */
  actions: RecordedAction[];
  /** Loading state */
  isLoading: boolean;
  /** Error message */
  error: string | null;
  /** Start recording */
  startRecording: () => Promise<void>;
  /** Stop recording */
  stopRecording: () => Promise<void>;
  /** Clear actions list */
  clearActions: () => void;
  /** Delete an action by index */
  deleteAction: (index: number) => void;
  /** Generate workflow from current actions */
  generateWorkflow: (name: string, projectId?: string) => Promise<GenerateWorkflowResponse>;
  /** Validate a selector */
  validateSelector: (selector: string) => Promise<SelectorValidation>;
  /** Refresh actions from server */
  refreshActions: () => Promise<void>;
}

export function useRecordMode({
  sessionId,
  pollInterval = 1000,
  onActionsReceived,
}: UseRecordModeOptions): UseRecordModeReturn {
  const [isRecording, setIsRecording] = useState(false);
  const [recordingId, setRecordingId] = useState<string | null>(null);
  const [actions, setActions] = useState<RecordedAction[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const pollTimerRef = useRef<NodeJS.Timeout | null>(null);
  const lastActionCountRef = useRef(0);

  const apiUrl = getApiBaseUrl();

  // Fetch actions from server
  const refreshActions = useCallback(async () => {
    if (!sessionId) return;

    try {
      const response = await fetch(`${apiUrl}/api/v1/recordings/live/${sessionId}/actions`);
      if (!response.ok) {
        throw new Error(`Failed to fetch actions: ${response.statusText}`);
      }

      const data: GetActionsResponse = await response.json();
      setActions(data.actions);

      // Notify if new actions received
      if (data.actions.length > lastActionCountRef.current && onActionsReceived) {
        const newActions = data.actions.slice(lastActionCountRef.current);
        onActionsReceived(newActions);
      }
      lastActionCountRef.current = data.actions.length;
    } catch (err) {
      console.error('Failed to refresh actions:', err);
    }
  }, [sessionId, apiUrl, onActionsReceived]);

  // Start polling when recording is active
  useEffect(() => {
    if (isRecording && pollInterval > 0) {
      pollTimerRef.current = setInterval(() => {
        void refreshActions();
      }, pollInterval);
    }

    return () => {
      if (pollTimerRef.current) {
        clearInterval(pollTimerRef.current);
        pollTimerRef.current = null;
      }
    };
  }, [isRecording, pollInterval, refreshActions]);

  const startRecording = useCallback(async () => {
    if (!sessionId) {
      setError('No session ID provided');
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`${apiUrl}/api/v1/recordings/live/start`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ session_id: sessionId }),
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || `Failed to start recording: ${response.statusText}`);
      }

      const data: StartRecordingResponse = await response.json();
      setRecordingId(data.recording_id);
      setIsRecording(true);
      setActions([]);
      lastActionCountRef.current = 0;
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start recording');
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, [sessionId, apiUrl]);

  const stopRecording = useCallback(async () => {
    if (!sessionId) {
      setError('No session ID provided');
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`${apiUrl}/api/v1/recordings/live/${sessionId}/stop`, {
        method: 'POST',
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || `Failed to stop recording: ${response.statusText}`);
      }

      const data: StopRecordingResponse = await response.json();

      // Final refresh to get all actions
      await refreshActions();

      setIsRecording(false);
      // Keep recordingId for reference
      console.log('Recording stopped:', data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to stop recording');
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, [sessionId, apiUrl, refreshActions]);

  const clearActions = useCallback(() => {
    setActions([]);
    lastActionCountRef.current = 0;
  }, []);

  const deleteAction = useCallback((index: number) => {
    setActions((prev) => prev.filter((_, i) => i !== index));
  }, []);

  const generateWorkflow = useCallback(
    async (name: string, projectId?: string): Promise<GenerateWorkflowResponse> => {
      if (!sessionId) {
        throw new Error('No session ID provided');
      }

      setIsLoading(true);
      setError(null);

      try {
        const response = await fetch(
          `${apiUrl}/api/v1/recordings/live/${sessionId}/generate-workflow`,
          {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              name,
              project_id: projectId,
            }),
          }
        );

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new Error(
            errorData.message || `Failed to generate workflow: ${response.statusText}`
          );
        }

        const data: GenerateWorkflowResponse = await response.json();
        return data;
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to generate workflow';
        setError(message);
        throw err;
      } finally {
        setIsLoading(false);
      }
    },
    [sessionId, apiUrl]
  );

  const validateSelector = useCallback(
    async (selector: string): Promise<SelectorValidation> => {
      if (!sessionId) {
        throw new Error('No session ID provided');
      }

      const response = await fetch(
        `${apiUrl}/api/v1/recordings/live/${sessionId}/validate-selector`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ selector }),
        }
      );

      if (!response.ok) {
        throw new Error(`Failed to validate selector: ${response.statusText}`);
      }

      return response.json();
    },
    [sessionId, apiUrl]
  );

  return {
    isRecording,
    recordingId,
    actions,
    isLoading,
    error,
    startRecording,
    stopRecording,
    clearActions,
    deleteAction,
    generateWorkflow,
    validateSelector,
    refreshActions,
  };
}
