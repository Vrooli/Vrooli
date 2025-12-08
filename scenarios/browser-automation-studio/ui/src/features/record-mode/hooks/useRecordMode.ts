/**
 * useRecordMode Hook
 *
 * Manages recording state and API interactions for Record Mode.
 * Supports:
 * - Starting/stopping recording
 * - Real-time WebSocket updates for actions (<100ms latency)
 * - Fallback polling for resilience
 * - Editing actions (selector and payload)
 * - Generating workflows
 * - Validating selectors
 */

import { useState, useCallback, useRef, useEffect } from 'react';
import { getApiBase } from '@/config';
import { useWebSocket } from '@/contexts/WebSocketContext';
import type {
  RecordedAction,
  StartRecordingResponse,
  StopRecordingResponse,
  GetActionsResponse,
  GenerateWorkflowResponse,
  SelectorValidation,
  SelectorSet,
  ReplayPreviewResponse,
} from '../types';

interface UseRecordModeOptions {
  /** Session ID for the browser session */
  sessionId: string | null;
  /** Polling interval for actions in ms (0 to disable). Used as fallback when WebSocket is unavailable. */
  pollInterval?: number;
  /** Callback when new actions are received */
  onActionsReceived?: (actions: RecordedAction[]) => void;
  /** Whether to use WebSocket for real-time updates (default: true) */
  useWebSocketUpdates?: boolean;
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
  /** Update an action's selector */
  updateSelector: (index: number, newSelector: string) => void;
  /** Update an action's payload */
  updatePayload: (index: number, payload: Record<string, unknown>) => void;
  /** Generate workflow from current actions */
  generateWorkflow: (name: string, projectId?: string) => Promise<GenerateWorkflowResponse>;
  /** Validate a selector */
  validateSelector: (selector: string) => Promise<SelectorValidation>;
  /** Refresh actions from server */
  refreshActions: () => Promise<void>;
  /** Replay recorded actions for preview testing */
  replayPreview: (options?: { limit?: number; stopOnFailure?: boolean }) => Promise<ReplayPreviewResponse>;
  /** Whether replay is currently in progress */
  isReplaying: boolean;
  /** Count of actions with low confidence */
  lowConfidenceCount: number;
  /** Count of actions with medium confidence */
  mediumConfidenceCount: number;
}

/** Confidence thresholds */
const CONFIDENCE = {
  HIGH: 0.8,
  MEDIUM: 0.5,
};

export function useRecordMode({
  sessionId,
  pollInterval = 5000, // Reduced frequency - WebSocket is primary, polling is fallback
  onActionsReceived,
  useWebSocketUpdates = true,
}: UseRecordModeOptions): UseRecordModeReturn {
  const [isRecording, setIsRecording] = useState(false);
  const [recordingId, setRecordingId] = useState<string | null>(null);
  const [actions, setActions] = useState<RecordedAction[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isReplaying, setIsReplaying] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const sessionIdRef = useRef<string | null>(sessionId ?? null);
  sessionIdRef.current = sessionId ?? null;
  const pollTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const lastActionCountRef = useRef(0);
  const wsSubscribedRef = useRef(false);

  const apiUrl = getApiBase();
  const { isConnected, lastMessage, send } = useWebSocket();

  useEffect(() => {
    setActions([]);
    setRecordingId(null);
    setIsRecording(false);
    setError(null);
    lastActionCountRef.current = 0;
  }, [sessionId]);

  // Calculate confidence counts
  const lowConfidenceCount = actions.filter(
    (a) => a.selector && a.confidence < CONFIDENCE.MEDIUM
  ).length;
  const mediumConfidenceCount = actions.filter(
    (a) => a.selector && a.confidence >= CONFIDENCE.MEDIUM && a.confidence < CONFIDENCE.HIGH
  ).length;

  // Fetch actions from server
  const refreshActions = useCallback(async () => {
    const currentSessionId = sessionIdRef.current;
    if (!currentSessionId) return;

    try {
      const response = await fetch(`${apiUrl}/recordings/live/${currentSessionId}/actions`);
      if (!response.ok) {
        throw new Error(`Failed to fetch actions: ${response.statusText}`);
      }

      const data: GetActionsResponse = await response.json();
      const actionsFromApi = Array.isArray(data.actions) ? data.actions : [];
      setActions(actionsFromApi);

      // Notify if new actions received
      if (actionsFromApi.length > lastActionCountRef.current && onActionsReceived) {
        const newActions = actionsFromApi.slice(lastActionCountRef.current);
        onActionsReceived(newActions);
      }
      lastActionCountRef.current = actionsFromApi.length;
    } catch (err) {
      console.error('Failed to refresh actions:', err);
    }
  }, [apiUrl, onActionsReceived]);

  // Subscribe to WebSocket updates when recording starts
  useEffect(() => {
    const currentSessionId = sessionIdRef.current;
    if (isRecording && useWebSocketUpdates && isConnected && currentSessionId && !wsSubscribedRef.current) {
      send({ type: 'subscribe_recording', session_id: currentSessionId });
      wsSubscribedRef.current = true;
      console.log('[useRecordMode] Subscribed to recording WebSocket updates');
    }

    // Unsubscribe when recording stops
    if (!isRecording && wsSubscribedRef.current) {
      send({ type: 'unsubscribe_recording' });
      wsSubscribedRef.current = false;
      console.log('[useRecordMode] Unsubscribed from recording WebSocket updates');
    }

    return () => {
      if (wsSubscribedRef.current) {
        send({ type: 'unsubscribe_recording' });
        wsSubscribedRef.current = false;
      }
    };
  }, [isRecording, useWebSocketUpdates, isConnected, sessionId, send]);

  // Handle incoming WebSocket messages
  useEffect(() => {
    if (!lastMessage) return;

    // Type guard for recording action messages
    const msg = lastMessage as { type: string; action?: unknown; session_id?: string };
    if (msg.type === 'recording_action' && msg.action) {
      const action = msg.action as RecordedAction;

      // Deduplicate - check if action already exists by ID
      setActions((prev) => {
        const exists = prev.some((a) => a.id === action.id);
        if (exists) return prev;

        const newActions = [...prev, action];

        // Notify callback
        if (onActionsReceived) {
          onActionsReceived([action]);
        }

        lastActionCountRef.current = newActions.length;
        return newActions;
      });

      console.log('[useRecordMode] Received action via WebSocket:', action.actionType);
    }
  }, [lastMessage, onActionsReceived]);

  // Fallback polling when WebSocket is not available or as periodic sync
  // Uses longer interval since WebSocket provides real-time updates
  useEffect(() => {
    const shouldPoll = isRecording && pollInterval > 0 && (!useWebSocketUpdates || !isConnected);

    if (shouldPoll) {
      pollTimerRef.current = setInterval(() => {
        void refreshActions();
      }, pollInterval);
      console.log('[useRecordMode] Started fallback polling');
    }

    return () => {
      if (pollTimerRef.current) {
        clearInterval(pollTimerRef.current);
        pollTimerRef.current = null;
      }
    };
  }, [isRecording, pollInterval, refreshActions, useWebSocketUpdates, isConnected]);

  const startRecording = useCallback(async () => {
    const currentSessionId = sessionIdRef.current;
    if (!currentSessionId) {
      setError('No session ID provided');
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`${apiUrl}/recordings/live/start`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ session_id: currentSessionId }),
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
  }, [apiUrl]);

  const stopRecording = useCallback(async () => {
    const currentSessionId = sessionIdRef.current;
    if (!currentSessionId) {
      setError('No session ID provided');
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const response = await fetch(`${apiUrl}/recordings/live/${currentSessionId}/stop`, {
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
  }, [apiUrl, refreshActions]);

  const clearActions = useCallback(() => {
    setActions([]);
    lastActionCountRef.current = 0;
  }, []);

  const deleteAction = useCallback((index: number) => {
    setActions((prev) => prev.filter((_, i) => i !== index));
  }, []);

  const updateSelector = useCallback((index: number, newSelector: string) => {
    setActions((prev) =>
      prev.map((action, i) => {
        if (i !== index) return action;

        // Update the primary selector and recalculate confidence
        const updatedSelector: SelectorSet = action.selector
          ? {
              ...action.selector,
              primary: newSelector,
            }
          : {
              primary: newSelector,
              candidates: [],
            };

        // Find confidence from candidates or default to medium
        const matchingCandidate = action.selector?.candidates.find(
          (c) => c.value === newSelector
        );
        const newConfidence = matchingCandidate?.confidence ?? 0.7;

        return {
          ...action,
          selector: updatedSelector,
          confidence: newConfidence,
        };
      })
    );
  }, []);

  const updatePayload = useCallback(
    (index: number, payload: Record<string, unknown>) => {
      setActions((prev) =>
        prev.map((action, i) => {
          if (i !== index) return action;
          return {
            ...action,
            payload: {
              ...action.payload,
              ...payload,
            } as RecordedAction['payload'],
          };
        })
      );
    },
    []
  );

  const generateWorkflow = useCallback(
    async (name: string, projectId?: string): Promise<GenerateWorkflowResponse> => {
      const currentSessionId = sessionIdRef.current;
      if (!currentSessionId) {
        throw new Error('No session ID provided');
      }

      if (actions.length === 0) {
        throw new Error('No actions to generate workflow from');
      }

      setIsLoading(true);
      setError(null);

      try {
        // Send local actions (with edits) to the API
        const response = await fetch(
          `${apiUrl}/recordings/live/${currentSessionId}/generate-workflow`,
          {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              name,
              project_id: projectId,
              // Include edited actions in the request
              actions: actions,
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
    [apiUrl, actions]
  );

  const validateSelector = useCallback(
    async (selector: string): Promise<SelectorValidation> => {
      const currentSessionId = sessionIdRef.current;
      if (!currentSessionId) {
        throw new Error('No session ID provided');
      }

      const response = await fetch(
        `${apiUrl}/recordings/live/${currentSessionId}/validate-selector`,
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
    [apiUrl]
  );

  const replayPreview = useCallback(
    async (options?: { limit?: number; stopOnFailure?: boolean }): Promise<ReplayPreviewResponse> => {
      const currentSessionId = sessionIdRef.current;
      if (!currentSessionId) {
        throw new Error('No session ID provided');
      }

      if (actions.length === 0) {
        throw new Error('No actions to replay');
      }

      setIsReplaying(true);
      setError(null);

      try {
        const response = await fetch(
          `${apiUrl}/recordings/live/${currentSessionId}/replay-preview`,
          {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              actions,
              limit: options?.limit,
              stop_on_failure: options?.stopOnFailure ?? true,
            }),
          }
        );

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new Error(
            errorData.message || `Failed to replay recording: ${response.statusText}`
          );
        }

        const data: ReplayPreviewResponse = await response.json();
        return data;
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to replay recording';
        setError(message);
        throw err;
      } finally {
        setIsReplaying(false);
      }
    },
    [apiUrl, actions]
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
    updateSelector,
    updatePayload,
    generateWorkflow,
    validateSelector,
    refreshActions,
    replayPreview,
    isReplaying,
    lowConfidenceCount,
    mediumConfidenceCount,
  };
}
