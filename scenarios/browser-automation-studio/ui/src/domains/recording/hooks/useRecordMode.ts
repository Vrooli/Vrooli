/**
 * useRecordMode Hook
 *
 * Manages recording state and API interactions for Record Mode.
 * Responsibilities are split into two layers:
 * - Transport: API calls for recording lifecycle (start/stop/generate/validate/replay)
 * - Editing: local action mutations and confidence bookkeeping
 *
 * Note: Timeline data (actions + page events) is now managed by useTimeline hook.
 * This hook focuses on recording lifecycle and action editing.
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
import type {
  RecordedAction,
  StopRecordingResponse,
  GenerateWorkflowResponse,
  SelectorValidation,
  SelectorSet,
  ReplayPreviewResponse,
} from '../types/types';
import type { WorkflowSettingsTyped } from '@/types/workflow';

interface UseRecordModeOptions {
  sessionId: string | null;
}

/** Minimal action data for inserting a new step */
export interface InsertActionData {
  actionType: RecordedAction['actionType'];
  payload?: Record<string, unknown>;
  selector?: string;
}

interface UseRecordModeReturn {
  isRecording: boolean;
  recordingId: string | null;
  actions: RecordedAction[];
  isLoading: boolean;
  error: string | null;
  startRecording: (sessionIdOverride?: string) => Promise<void>;
  stopRecording: () => Promise<void>;
  clearActions: () => void;
  deleteAction: (index: number) => void;
  insertAction: (data: InsertActionData) => void;
  updateSelector: (index: number, newSelector: string) => void;
  updatePayload: (index: number, payload: Record<string, unknown>) => void;
  generateWorkflow: (name: string, projectId?: string, actionsOverride?: RecordedAction[], settings?: WorkflowSettingsTyped) => Promise<GenerateWorkflowResponse>;
  validateSelector: (selector: string) => Promise<SelectorValidation>;
  replayPreview: (options?: { limit?: number; stopOnFailure?: boolean }, actionsOverride?: RecordedAction[]) => Promise<ReplayPreviewResponse>;
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
  startRecording: (sessionIdOverride?: string) => Promise<void>;
  stopRecording: () => Promise<void>;
  generateWorkflow: (name: string, projectId?: string, actionsOverride?: RecordedAction[], settings?: WorkflowSettingsTyped) => Promise<GenerateWorkflowResponse>;
  validateSelector: (selector: string) => Promise<SelectorValidation>;
  replayPreview: (options?: { limit?: number; stopOnFailure?: boolean }, actionsOverride?: RecordedAction[]) => Promise<ReplayPreviewResponse>;
}

function useRecordingTransport({
  sessionId,
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

  useEffect(() => {
    setActions([]);
    setRecordingId(null);
    setIsRecording(false);
    setError(null);
  }, [sessionId]);

  const startRecording = useCallback(async (sessionIdOverride?: string) => {
    const currentSessionId = sessionIdOverride ?? sessionIdRef.current;
    if (!currentSessionId || currentSessionId.trim() === '') {
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

      const data = await response.json().catch(() => ({}));

      // Handle 409 Conflict: Recording is already in progress for this session.
      // This happens when the page is refreshed - React state is lost but the
      // playwright-driver still has an active recording. We treat this as a
      // successful state sync rather than an error.
      if (response.status === 409 && data.error === 'RECORDING_IN_PROGRESS') {
        console.log('Recording already in progress, syncing state:', data.recording_id);
        setRecordingId(data.recording_id || null);
        setIsRecording(true);
        // Don't clear actions - they may be streaming in via WebSocket
        return;
      }

      if (!response.ok) {
        throw new Error(data.message || `Failed to start recording: ${response.statusText}`);
      }

      setRecordingId(data.recording_id);
      setIsRecording(true);
      setActions([]);
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
      setIsRecording(false);
      console.log('Recording stopped:', data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to stop recording');
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, [apiUrl]);

  const generateWorkflow = useCallback(
    async (name: string, projectId?: string, actionsOverride?: RecordedAction[], settings?: WorkflowSettingsTyped): Promise<GenerateWorkflowResponse> => {
      const currentSessionId = sessionIdRef.current;
      if (!currentSessionId) {
        throw new Error('No session ID provided');
      }

      const actionsToSend = actionsOverride ?? actions;
      if (actionsToSend.length === 0) {
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
              actions: actionsToSend,
              settings,
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
    async (options?: { limit?: number; stopOnFailure?: boolean }, actionsOverride?: RecordedAction[]): Promise<ReplayPreviewResponse> => {
      const currentSessionId = sessionIdRef.current;
      if (!currentSessionId) {
        throw new Error('No session ID provided');
      }

      const actionsToSend = actionsOverride ?? actions;
      if (actionsToSend.length === 0) {
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
              actions: actionsToSend,
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
    setActions,
    isLoading,
    isReplaying,
    error,
    startRecording,
    stopRecording,
    generateWorkflow,
    validateSelector,
    replayPreview,
  };
}

interface UseActionEditingReturn {
  clearActions: () => void;
  deleteAction: (index: number) => void;
  insertAction: (data: InsertActionData) => void;
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

  const insertAction = useCallback(
    (data: InsertActionData) => {
      setActions((prev) => {
        const newAction: RecordedAction = {
          id: `manual-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`,
          sessionId: '',
          sequenceNum: prev.length,
          timestamp: new Date().toISOString(),
          actionType: data.actionType,
          confidence: 1.0, // Manual actions have full confidence
          url: '',
          payload: data.payload as RecordedAction['payload'],
          selector: data.selector ? { primary: data.selector, candidates: [] } : undefined,
        };
        return [...prev, newAction];
      });
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
    insertAction,
    updateSelector,
    updatePayload,
    lowConfidenceCount,
    mediumConfidenceCount,
  };
}

export function useRecordMode({
  sessionId,
}: UseRecordModeOptions): UseRecordModeReturn {
  const apiUrl = getApiBase();

  const transport = useRecordingTransport({
    sessionId,
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
    insertAction: editing.insertAction,
    updateSelector: editing.updateSelector,
    updatePayload: editing.updatePayload,
    generateWorkflow: transport.generateWorkflow,
    validateSelector: transport.validateSelector,
    replayPreview: transport.replayPreview,
    isReplaying: transport.isReplaying,
    lowConfidenceCount: editing.lowConfidenceCount,
    mediumConfidenceCount: editing.mediumConfidenceCount,
  };
}
