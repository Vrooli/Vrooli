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
import toast from 'react-hot-toast';
import { recordingApi } from '../api';
import type { RecordedAction, SelectorSet } from '../types/types';
import type {
  GenerateWorkflowResponse,
  SelectorValidation,
  ReplayPreviewResponse,
} from '../api/schemas';
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

type UseRecordingTransportOptions = UseRecordModeOptions;

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
}: UseRecordingTransportOptions): UseRecordingTransportReturn {
  const [isRecording, setIsRecording] = useState(false);
  const [recordingId, setRecordingId] = useState<string | null>(null);
  const [actions, setActions] = useState<RecordedAction[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isReplaying, setIsReplaying] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const sessionIdRef = useRef<string | null>(sessionId ?? null);
  sessionIdRef.current = sessionId ?? null;

  // AbortController for request cancellation
  const abortControllerRef = useRef<AbortController | null>(null);

  // Clean up abort controller on unmount
  useEffect(() => {
    return () => {
      abortControllerRef.current?.abort();
    };
  }, []);

  // Reset state when session changes
  useEffect(() => {
    // Abort any pending requests when session changes
    abortControllerRef.current?.abort();
    abortControllerRef.current = new AbortController();

    setActions([]);
    setRecordingId(null);
    setIsRecording(false);
    setError(null);
  }, [sessionId]);

  const startRecording = useCallback(async (sessionIdOverride?: string) => {
    const currentSessionId = sessionIdOverride ?? sessionIdRef.current;
    if (!currentSessionId?.trim()) {
      setError('No session ID provided');
      return;
    }

    setIsLoading(true);
    setError(null);

    const result = await recordingApi.startRecording(currentSessionId, {
      signal: abortControllerRef.current?.signal,
    });

    setIsLoading(false);

    if (!result.success) {
      setError(result.error);
      return;
    }

    // Handle 409 case - recording was already in progress
    setRecordingId(result.data.recording_id);
    setIsRecording(true);

    // Only clear actions for new recordings, not for 409 reconnections
    // (check if we were already recording - if so, don't clear)
    if (!isRecording) {
      setActions([]);
      toast.success('Recording started', { duration: 2000 });
    } else {
      toast.success('Reconnected to existing session', { duration: 2000 });
    }
  }, [isRecording]);

  const stopRecording = useCallback(async () => {
    const currentSessionId = sessionIdRef.current;
    if (!currentSessionId) {
      setError('No session ID provided');
      return;
    }

    setIsLoading(true);
    setError(null);

    const result = await recordingApi.stopRecording(currentSessionId, {
      signal: abortControllerRef.current?.signal,
    });

    setIsLoading(false);

    if (!result.success) {
      setError(result.error);
      return;
    }

    setIsRecording(false);
    console.log('Recording stopped:', result.data);
  }, []);

  const generateWorkflow = useCallback(
    async (name: string, projectId?: string, actionsOverride?: RecordedAction[], settings?: WorkflowSettingsTyped): Promise<GenerateWorkflowResponse> => {
      const currentSessionId = sessionIdRef.current;
      if (!currentSessionId) {
        const error = 'No session ID provided';
        setError(error);
        throw new Error(error);
      }

      const actionsToSend = actionsOverride ?? actions;
      if (actionsToSend.length === 0) {
        const error = 'No actions to generate workflow from';
        setError(error);
        throw new Error(error);
      }

      setIsLoading(true);
      setError(null);

      const result = await recordingApi.generateWorkflow(
        currentSessionId,
        { name, projectId, actions: actionsToSend, settings },
        { signal: abortControllerRef.current?.signal }
      );

      setIsLoading(false);

      if (!result.success) {
        setError(result.error);
        throw new Error(result.error);
      }

      return result.data;
    },
    [actions]
  );

  const validateSelector = useCallback(
    async (selector: string): Promise<SelectorValidation> => {
      const currentSessionId = sessionIdRef.current;
      if (!currentSessionId) {
        throw new Error('No session ID provided');
      }

      const result = await recordingApi.validateSelector(currentSessionId, selector, {
        signal: abortControllerRef.current?.signal,
      });

      if (!result.success) {
        throw new Error(result.error);
      }

      return result.data;
    },
    []
  );

  const replayPreview = useCallback(
    async (options?: { limit?: number; stopOnFailure?: boolean }, actionsOverride?: RecordedAction[]): Promise<ReplayPreviewResponse> => {
      const currentSessionId = sessionIdRef.current;
      if (!currentSessionId) {
        const error = 'No session ID provided';
        setError(error);
        throw new Error(error);
      }

      const actionsToSend = actionsOverride ?? actions;
      if (actionsToSend.length === 0) {
        const error = 'No actions to replay';
        setError(error);
        throw new Error(error);
      }

      setIsReplaying(true);
      setError(null);

      const result = await recordingApi.replayPreview(
        currentSessionId,
        {
          actions: actionsToSend,
          limit: options?.limit,
          stopOnFailure: options?.stopOnFailure,
        },
        { signal: abortControllerRef.current?.signal }
      );

      setIsReplaying(false);

      if (!result.success) {
        setError(result.error);
        throw new Error(result.error);
      }

      return result.data;
    },
    [actions]
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
  const transport = useRecordingTransport({
    sessionId,
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
