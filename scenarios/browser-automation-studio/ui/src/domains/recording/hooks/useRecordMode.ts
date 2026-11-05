/**
 * useRecordMode Hook
 *
 * Manages recording state and API interactions for Record Mode.
 * Responsibilities are split into two layers:
 * - Transport: API calls for recording lifecycle (start/stop/generate/validate/replay)
 * TimelineEntry is the recording action owner; legacy API payloads are supplied by the workspace.
 *
 * Note: Timeline data (actions + page events) is managed by useWorkspaceTimeline.
 * This hook focuses on recording lifecycle and boundary API calls.
 */

import {
  useState,
  useCallback,
  useRef,
  useEffect,
} from 'react';
import toast from 'react-hot-toast';
import { recordingApi } from '../api';
import type { RecordedAction } from '../types/types';
import type {
  GenerateWorkflowResponse,
  SelectorValidation,
  ReplayPreviewResponse,
} from '../api/schemas';
import type { WorkflowSettingsTyped } from '@/types/workflow';

interface UseRecordModeOptions {
  sessionId: string | null;
}

interface UseRecordModeReturn {
  isRecording: boolean;
  recordingId: string | null;
  isLoading: boolean;
  error: string | null;
  startRecording: (sessionIdOverride?: string) => Promise<void>;
  stopRecording: () => Promise<void>;
  generateWorkflow: (name: string, projectId?: string, actionsOverride?: RecordedAction[], settings?: WorkflowSettingsTyped) => Promise<GenerateWorkflowResponse>;
  validateSelector: (selector: string) => Promise<SelectorValidation>;
  replayPreview: (options?: { limit?: number; stopOnFailure?: boolean }, actionsOverride?: RecordedAction[]) => Promise<ReplayPreviewResponse>;
  isReplaying: boolean;
}

type UseRecordingTransportOptions = UseRecordModeOptions;

interface UseRecordingTransportReturn {
  isRecording: boolean;
  recordingId: string | null;
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
  const [isLoading, setIsLoading] = useState(false);
  const [isReplaying, setIsReplaying] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const sessionIdRef = useRef<string | null>(sessionId ?? null);
  sessionIdRef.current = sessionId ?? null;

  // AbortController for request cancellation
  const abortControllerRef = useRef<AbortController | null>(null);
  const mountedRef = useRef(false);
  const ownerRef = useRef({ sessionId, generation: 0, adoptingSessionId: null as string | null });
  if (ownerRef.current.sessionId !== sessionId) {
    const adopting = ownerRef.current.sessionId === null && ownerRef.current.adoptingSessionId === sessionId;
    ownerRef.current = { sessionId, generation: ownerRef.current.generation + (adopting ? 0 : 1), adoptingSessionId: null };
  }
  const controllerGenerationRef = useRef(-1);
  const captureOwner = useCallback(() => {
    const generation = ownerRef.current.generation;
    const signal = abortControllerRef.current?.signal;
    return { signal, isCurrent: () => mountedRef.current && generation === ownerRef.current.generation && !signal?.aborted };
  }, []);

  // Clean up abort controller on unmount
  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      ownerRef.current = { ...ownerRef.current, generation: ownerRef.current.generation + 1, adoptingSessionId: null };
      abortControllerRef.current?.abort();
    };
  }, []);

  // Reset only when the session lifetime changes. Adopting the explicitly
  // created session keeps its recording, whether start completed before or after.
  useEffect(() => {
    if (controllerGenerationRef.current !== ownerRef.current.generation) {
      abortControllerRef.current?.abort();
      abortControllerRef.current = new AbortController();
      controllerGenerationRef.current = ownerRef.current.generation;
      setIsLoading(false);
      setIsReplaying(false);
      setRecordingId(null);
      setIsRecording(false);
      setError(null);
    }
  }, [sessionId]);

  const startRecording = useCallback(async (sessionIdOverride?: string) => {
    if (!mountedRef.current) return;
    const currentSessionId = sessionIdOverride ?? sessionIdRef.current;
    if (!currentSessionId?.trim()) {
      setError('No session ID provided');
      return;
    }

    setIsLoading(true);
    setError(null);
    if (sessionIdOverride && sessionIdRef.current === null) ownerRef.current.adoptingSessionId = currentSessionId;
    const owner = captureOwner();

    const result = await recordingApi.startRecording(currentSessionId, {
      signal: owner.signal,
    });

    if (!owner.isCurrent()) return;

    setIsLoading(false);

    if (!result.success) {
      setError(result.error);
      return;
    }

    // Handle 409 case - recording was already in progress
    setRecordingId(result.data.recording_id);
    setIsRecording(true);

    if (!isRecording) {
      toast.success('Recording started', { duration: 2000 });
    } else {
      toast.success('Reconnected to existing session', { duration: 2000 });
    }
  }, [isRecording, captureOwner]);

  const stopRecording = useCallback(async () => {
    if (!mountedRef.current) return;
    const currentSessionId = sessionIdRef.current;
    if (!currentSessionId) {
      setError('No session ID provided');
      return;
    }

    setIsLoading(true);
    setError(null);
    const owner = captureOwner();

    const result = await recordingApi.stopRecording(currentSessionId, {
      signal: owner.signal,
    });

    if (!owner.isCurrent()) return;

    setIsLoading(false);

    if (!result.success) {
      setError(result.error);
      return;
    }

    setIsRecording(false);
    console.log('Recording stopped:', result.data);
  }, [captureOwner]);

  const generateWorkflow = useCallback(
    async (name: string, projectId?: string, actionsOverride?: RecordedAction[], settings?: WorkflowSettingsTyped): Promise<GenerateWorkflowResponse> => {
      if (!mountedRef.current) throw new DOMException('Recording session retired', 'AbortError');
      const currentSessionId = sessionIdRef.current;
      if (!currentSessionId) {
        const error = 'No session ID provided';
        setError(error);
        throw new Error(error);
      }

      const actionsToSend = actionsOverride ?? [];
      if (actionsToSend.length === 0) {
        const error = 'No actions to generate workflow from';
        setError(error);
        throw new Error(error);
      }

      setIsLoading(true);
      setError(null);
      const owner = captureOwner();

      const result = await recordingApi.generateWorkflow(
        currentSessionId,
        { name, projectId, actions: actionsToSend, settings },
        { signal: owner.signal }
      );

      if (!owner.isCurrent()) throw new DOMException('Recording session retired', 'AbortError');

      setIsLoading(false);

      if (!result.success) {
        setError(result.error);
        throw new Error(result.error);
      }

      return result.data;
    },
    [captureOwner]
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
      if (!mountedRef.current) throw new DOMException('Recording session retired', 'AbortError');
      const currentSessionId = sessionIdRef.current;
      if (!currentSessionId) {
        const error = 'No session ID provided';
        setError(error);
        throw new Error(error);
      }

      const actionsToSend = actionsOverride ?? [];
      if (actionsToSend.length === 0) {
        const error = 'No actions to replay';
        setError(error);
        throw new Error(error);
      }

      setIsReplaying(true);
      setError(null);
      const owner = captureOwner();

      const result = await recordingApi.replayPreview(
        currentSessionId,
        {
          actions: actionsToSend,
          limit: options?.limit,
          stopOnFailure: options?.stopOnFailure,
        },
        { signal: owner.signal }
      );

      if (!owner.isCurrent()) throw new DOMException('Recording session retired', 'AbortError');

      setIsReplaying(false);

      if (!result.success) {
        setError(result.error);
        throw new Error(result.error);
      }

      return result.data;
    },
    [captureOwner]
  );

  return {
    isRecording,
    recordingId,
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

export function useRecordMode({
  sessionId,
}: UseRecordModeOptions): UseRecordModeReturn {
  const transport = useRecordingTransport({
    sessionId,
  });

  return {
    isRecording: transport.isRecording,
    recordingId: transport.recordingId,
    isLoading: transport.isLoading,
    error: transport.error,
    startRecording: transport.startRecording,
    stopRecording: transport.stopRecording,
    generateWorkflow: transport.generateWorkflow,
    validateSelector: transport.validateSelector,
    replayPreview: transport.replayPreview,
    isReplaying: transport.isReplaying,
  };
}
