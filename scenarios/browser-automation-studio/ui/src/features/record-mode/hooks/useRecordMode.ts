/**
 * useRecordMode Hook
 *
 * Manages recording state and API interactions for Record Mode.
 * Responsibilities are split into two layers:
 * - Transport: API/WebSocket/polling + recording lifecycle
 * - Editing: local action mutations and confidence bookkeeping
 */

import {
  useState,
  useCallback,
  useRef,
  useEffect,
  useMemo,
  type Dispatch,
  type SetStateAction,
} from 'react';
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
  sessionId: string | null;
  pollInterval?: number;
  onActionsReceived?: (actions: RecordedAction[]) => void;
  useWebSocketUpdates?: boolean;
}

interface UseRecordModeReturn {
  isRecording: boolean;
  recordingId: string | null;
  actions: RecordedAction[];
  isLoading: boolean;
  error: string | null;
  startRecording: () => Promise<void>;
  stopRecording: () => Promise<void>;
  clearActions: () => void;
  deleteAction: (index: number) => void;
  updateSelector: (index: number, newSelector: string) => void;
  updatePayload: (index: number, payload: Record<string, unknown>) => void;
  generateWorkflow: (name: string, projectId?: string) => Promise<GenerateWorkflowResponse>;
  validateSelector: (selector: string) => Promise<SelectorValidation>;
  refreshActions: () => Promise<void>;
  replayPreview: (options?: { limit?: number; stopOnFailure?: boolean }) => Promise<ReplayPreviewResponse>;
  isReplaying: boolean;
  lowConfidenceCount: number;
  mediumConfidenceCount: number;
}

const CONFIDENCE = {
  HIGH: 0.8,
  MEDIUM: 0.5,
};

type ActionSetter = Dispatch<SetStateAction<RecordedAction[]>>;

interface UseRecordingTransportOptions extends UseRecordModeOptions {
  apiUrl: string;
}

interface UseRecordingTransportReturn {
  isRecording: boolean;
  recordingId: string | null;
  actions: RecordedAction[];
  setActions: ActionSetter;
  isLoading: boolean;
  isReplaying: boolean;
  error: string | null;
  startRecording: () => Promise<void>;
  stopRecording: () => Promise<void>;
  generateWorkflow: (name: string, projectId?: string) => Promise<GenerateWorkflowResponse>;
  validateSelector: (selector: string) => Promise<SelectorValidation>;
  refreshActions: () => Promise<void>;
  replayPreview: (options?: { limit?: number; stopOnFailure?: boolean }) => Promise<ReplayPreviewResponse>;
}

function useRecordingTransport({
  sessionId,
  pollInterval,
  onActionsReceived,
  useWebSocketUpdates,
  apiUrl,
}: UseRecordingTransportOptions): UseRecordingTransportReturn {
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

  const { isConnected, lastMessage, send } = useWebSocket();

  const setActionsWithSync = useCallback<ActionSetter>(
    (next) => {
      setActions((prev) => {
        const resolved = typeof next === 'function' ? (next as (p: RecordedAction[]) => RecordedAction[])(prev) : next;
        lastActionCountRef.current = resolved.length;
        return resolved;
      });
    },
    []
  );

  useEffect(() => {
    setActionsWithSync([]);
    setRecordingId(null);
    setIsRecording(false);
    setError(null);
  }, [sessionId, setActionsWithSync]);

  const refreshActions = useCallback(async () => {
    const currentSessionId = sessionIdRef.current;
    if (!currentSessionId) return;

    try {
      const previousCount = lastActionCountRef.current;
      const response = await fetch(`${apiUrl}/recordings/live/${currentSessionId}/actions`);
      if (!response.ok) {
        throw new Error(`Failed to fetch actions: ${response.statusText}`);
      }

      const data: GetActionsResponse = await response.json();
      const actionsFromApi = Array.isArray(data.actions) ? data.actions : [];
      setActionsWithSync(actionsFromApi);

      if (actionsFromApi.length > previousCount && onActionsReceived) {
        const newActions = actionsFromApi.slice(previousCount);
        onActionsReceived(newActions);
      }
    } catch (err) {
      console.error('Failed to refresh actions:', err);
    }
  }, [apiUrl, onActionsReceived, setActionsWithSync]);

  useEffect(() => {
    const currentSessionId = sessionIdRef.current;
    if (isRecording && useWebSocketUpdates && isConnected && currentSessionId && !wsSubscribedRef.current) {
      send({ type: 'subscribe_recording', session_id: currentSessionId });
      wsSubscribedRef.current = true;
      console.log('[useRecordMode] Subscribed to recording WebSocket updates');
    }

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

  useEffect(() => {
    if (!lastMessage) return;

    const msg = lastMessage as { type: string; action?: unknown; session_id?: string };
    if (msg.type === 'recording_action' && msg.action) {
      const action = msg.action as RecordedAction;

      setActionsWithSync((prev) => {
        const exists = prev.some((a) => a.id === action.id);
        if (exists) return prev;

        const newActions = [...prev, action];
        if (onActionsReceived) {
          onActionsReceived([action]);
        }
        return newActions;
      });

      console.log('[useRecordMode] Received action via WebSocket:', action.actionType);
    }
  }, [lastMessage, onActionsReceived, setActionsWithSync]);

  useEffect(() => {
    const shouldPoll = isRecording && (pollInterval ?? 0) > 0;
    const intervalMs = pollInterval && pollInterval > 0 ? pollInterval : 2000;

    if (shouldPoll) {
      pollTimerRef.current = setInterval(() => {
        void refreshActions();
      }, intervalMs);
      console.log('[useRecordMode] Started action polling');
    }

    return () => {
      if (pollTimerRef.current) {
        clearInterval(pollTimerRef.current);
        pollTimerRef.current = null;
      }
    };
  }, [isRecording, pollInterval, refreshActions]);

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
      setActionsWithSync([]);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start recording');
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, [apiUrl, setActionsWithSync]);

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

      await refreshActions();
      setIsRecording(false);
      console.log('Recording stopped:', data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to stop recording');
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, [apiUrl, refreshActions]);

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
        const response = await fetch(
          `${apiUrl}/recordings/live/${currentSessionId}/generate-workflow`,
          {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
              name,
              project_id: projectId,
              actions,
            }),
          }
        );

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new Error(
            errorData.message || `Failed to generate workflow: ${response.statusText}`
          );
        }

        return response.json();
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

        return response.json();
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
    setActions: setActionsWithSync,
    isLoading,
    isReplaying,
    error,
    startRecording,
    stopRecording,
    generateWorkflow,
    validateSelector,
    refreshActions,
    replayPreview,
  };
}

interface UseActionEditingReturn {
  clearActions: () => void;
  deleteAction: (index: number) => void;
  updateSelector: (index: number, newSelector: string) => void;
  updatePayload: (index: number, payload: Record<string, unknown>) => void;
  lowConfidenceCount: number;
  mediumConfidenceCount: number;
}

function useActionEditing(actions: RecordedAction[], setActions: ActionSetter): UseActionEditingReturn {
  const updateSelector = useCallback(
    (index: number, newSelector: string) => {
      setActions((prev) =>
        prev.map((action, i) => {
          if (i !== index) return action;

          const updatedSelector: SelectorSet = action.selector
            ? { ...action.selector, primary: newSelector }
            : { primary: newSelector, candidates: [] };

          const matchingCandidate = action.selector?.candidates.find((c) => c.value === newSelector);
          const newConfidence = matchingCandidate?.confidence ?? 0.7;

          return { ...action, selector: updatedSelector, confidence: newConfidence };
        })
      );
    },
    [setActions]
  );

  const updatePayload = useCallback(
    (index: number, payload: Record<string, unknown>) => {
      setActions((prev) =>
        prev.map((action, i) => {
          if (i !== index) return action;
          return {
            ...action,
            payload: { ...action.payload, ...payload } as RecordedAction['payload'],
          };
        })
      );
    },
    [setActions]
  );

  const clearActions = useCallback(() => {
    setActions([]);
  }, [setActions]);

  const deleteAction = useCallback(
    (index: number) => {
      setActions((prev) => prev.filter((_, i) => i !== index));
    },
    [setActions]
  );

  const { lowConfidenceCount, mediumConfidenceCount } = useMemo(() => {
    const low = actions.filter((a) => a.selector && a.confidence < CONFIDENCE.MEDIUM).length;
    const medium = actions.filter(
      (a) => a.selector && a.confidence >= CONFIDENCE.MEDIUM && a.confidence < CONFIDENCE.HIGH
    ).length;
    return { lowConfidenceCount: low, mediumConfidenceCount: medium };
  }, [actions]);

  return {
    clearActions,
    deleteAction,
    updateSelector,
    updatePayload,
    lowConfidenceCount,
    mediumConfidenceCount,
  };
}

export function useRecordMode({
  sessionId,
  pollInterval = 5000,
  onActionsReceived,
  useWebSocketUpdates = true,
}: UseRecordModeOptions): UseRecordModeReturn {
  const apiUrl = getApiBase();

  const transport = useRecordingTransport({
    sessionId,
    pollInterval,
    onActionsReceived,
    useWebSocketUpdates,
    apiUrl,
  });

  const editing = useActionEditing(transport.actions, transport.setActions);

  return {
    isRecording: transport.isRecording,
    recordingId: transport.recordingId,
    actions: transport.actions,
    isLoading: transport.isLoading,
    error: transport.error,
    startRecording: transport.startRecording,
    stopRecording: transport.stopRecording,
    clearActions: editing.clearActions,
    deleteAction: editing.deleteAction,
    updateSelector: editing.updateSelector,
    updatePayload: editing.updatePayload,
    generateWorkflow: transport.generateWorkflow,
    validateSelector: transport.validateSelector,
    refreshActions: transport.refreshActions,
    replayPreview: transport.replayPreview,
    isReplaying: transport.isReplaying,
    lowConfidenceCount: editing.lowConfidenceCount,
    mediumConfidenceCount: editing.mediumConfidenceCount,
  };
}
