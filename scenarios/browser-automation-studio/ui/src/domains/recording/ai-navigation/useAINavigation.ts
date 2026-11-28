/**
 * useAINavigation Hook
 *
 * Manages AI-driven browser navigation sessions.
 * - Starts navigation via API
 * - Subscribes to WebSocket events for real-time updates
 * - Tracks step history for timeline display
 * - Handles abort functionality
 */

import { useCallback, type MutableRefObject } from 'react';
import type { GetNavigationStatusResponse } from '@/api/visionNavigation';
import { getAIRequestHeadersSync } from '@/utils/apiHeaders';
import { logger } from '@/utils/logger';
import { recordingApi, type ApiResult, type RequestOptions } from '../api';
import type {
  AINavigationState,
  AINavigationStep,
  VisionModelSpec,
} from './types';
import type { AINavigationRuntimeRefs } from './runtimeRefs';
import { VISION_MODELS } from './types';
import {
  mergeRecoveredSteps,
  parseRecoveryStatus,
  recoveredStatePatch,
  isRecord,
  statusStepToAINavigationStep,
} from './navigationEvents';
import { useAINavigationEvents } from './useAINavigationEvents';
import { useAINavigationRuntime } from './useAINavigationRuntime';

export class AINavigationError extends Error {
  code: string;
  details?: Record<string, string>;

  constructor(code: string, message: string, details?: Record<string, string>) {
    super(message);
    this.name = 'AINavigationError';
    this.code = code;
    this.details = details;
  }
}

const parseNavigationError = (rawError: string): AINavigationError => {
  let code = 'UNKNOWN_ERROR';
  let message = rawError;
  let details: Record<string, string> | undefined;

  try {
    const errorData: unknown = JSON.parse(rawError);
    if (!isRecord(errorData)) return new AINavigationError(code, message);
    if (typeof errorData.code === 'string') code = errorData.code;
    if (typeof errorData.message === 'string') message = errorData.message;
    if (isRecord(errorData.details) && Object.values(errorData.details).every((value) => typeof value === 'string')) {
      details = errorData.details as Record<string, string>;
    }
  } catch {
    // Non-JSON errors retain their raw message and the generic code.
  }
  return new AINavigationError(code, message, details);
};

const startAINavigationRequest = async (
  sessionId: string,
  prompt: string,
  model: string,
  maxSteps: number,
  signal: AbortSignal | undefined,
): Promise<string> => {
  const result = await recordingApi.startAINavigation(
    { sessionId, prompt, model, maxSteps },
    getAIRequestHeadersSync(),
    { signal },
  );
  if (!result.success) throw parseNavigationError(result.error);
  return result.data.navigationId;
};

const commitStartedNavigation = (
  navigationId: string,
  startAttempt: number,
  requestSignal: AbortSignal | undefined,
  refs: AINavigationRuntimeRefs,
  observeNavigationStatus: (navigationId: string, generation: number) => void,
): string | null => {
  if (requestSignal?.aborted || refs.startAttemptRef.current !== startAttempt) return null;
  refs.seenStepNumbersRef.current.clear();
  refs.navigationIdRef.current = navigationId;
  refs.startInFlightRef.current = false;
  const commandGeneration = refs.navigationCommandGenerationRef.current;
  refs.setState((prev) => ({ ...prev, navigationId }));
  refs.onStartedRef.current?.(navigationId, startAttempt);
  observeNavigationStatus(navigationId, commandGeneration);
  return navigationId;
};

const handleNavigationStartFailure = (
  error: unknown,
  startAttempt: number,
  requestSignal: AbortSignal | undefined,
  refs: AINavigationRuntimeRefs,
): never => {
  if (refs.startAttemptRef.current !== startAttempt) throw error;
  refs.startInFlightRef.current = false;
  if (requestSignal?.aborted) throw error;
  refs.navigationStatusRef.current = 'failed';
  const message = error instanceof Error ? error.message : 'Failed to start navigation';
  refs.setState((prev) => ({ ...prev, isNavigating: false, status: 'failed', error: message }));
  throw error;
};

// ============================================================================
// Hook Interface
// ============================================================================

interface UseAINavigationOptions {
  sessionId: string | null;
  /** Callback when the server accepts a navigation and assigns its identity. */
  onStarted?: (navigationId: string, startAttempt?: number) => void;
  /** Callback when a step is received */
  onStep?: (step: AINavigationStep, navigationId?: string) => void;
  /** Callback when navigation completes */
  onComplete?: (status: string, summary?: string, navigationId?: string) => void;
}

interface UseAINavigationReturn {
  /** Current navigation state */
  state: AINavigationState;
  /** Start AI navigation with a prompt. Returns the navigationId on success, null on failure. */
  startNavigation: (prompt: string, model: string, maxSteps?: number) => Promise<string | null>;
  /** Abort the current navigation */
  abortNavigation: () => Promise<void>;
  /** Resume navigation after human intervention */
  resumeNavigation: () => Promise<boolean>;
  /** Reset the navigation state */
  reset: () => void;
  /** Available vision models */
  availableModels: VisionModelSpec[];
  /** Whether navigation is in progress */
  isNavigating: boolean;
  /** Whether navigation is awaiting human intervention */
  isAwaitingHuman: boolean;
}

const applyObservedNavigationStatus = (
  data: GetNavigationStatusResponse,
  status: AINavigationState['status'],
  navigationId: string,
  refs: AINavigationRuntimeRefs,
): void => {
  const recoveredSteps = data.steps.map(statusStepToAINavigationStep);
  const recoveredData = data.extractedData
    ? { ...data.extractedData } as Record<string, unknown>
    : null;
  const recoveredStepsForConsumers = recoveredSteps.filter((step) => {
    if (refs.seenStepNumbersRef.current.has(step.stepNumber)) return false;
    refs.seenStepNumbersRef.current.add(step.stepNumber);
    return true;
  });
  recoveredStepsForConsumers.forEach((step) => refs.onStepRef.current?.(step, navigationId));

  const isAwaitingHuman = status === 'awaiting_human';
  const isTerminal = data.terminal && !isAwaitingHuman;
  if (isAwaitingHuman || isTerminal || refs.navigationStatusRef.current !== 'aborting') {
    refs.handoffCommandInFlightRef.current = false;
  }
  refs.navigationStatusRef.current = status;
  refs.setState((prev) => ({
    ...prev,
    ...(isTerminal ? { navigationId: null } : {}),
    isNavigating: !isAwaitingHuman && !isTerminal,
    ...recoveredStatePatch(data, status, mergeRecoveredSteps(recoveredSteps, prev.steps), recoveredData),
    error: isAwaitingHuman
      ? data.error || data.verificationError || null
      : data.error || null,
    ...(!isAwaitingHuman ? { humanIntervention: null } : {}),
  }));
  if (isTerminal) {
    refs.navigationIdRef.current = null;
    refs.onCompleteRef.current?.(status, data.summary || undefined, navigationId);
  }
};

const recoverableNavigationStatuses = new Set<AINavigationState['status']>([
  'navigating',
  'awaiting_human',
]);

const isRecoverableNavigationStatus = (
  status: AINavigationState['status'],
  terminal: boolean,
): boolean => terminal || recoverableNavigationStatuses.has(status);

type NavigationObservation =
  | { kind: 'aborted' }
  | { kind: 'unavailable'; message: string }
  | { kind: 'received'; data: GetNavigationStatusResponse };

const readNavigationObservation = async (
  navigationId: string,
  controller: AbortController,
): Promise<NavigationObservation> => {
  try {
    const result = await recordingApi.getAINavigationStatus(navigationId, 300_000, {
      signal: controller.signal,
    });
    if (!result.success) {
      return { kind: 'unavailable', message: result.error || 'Failed to observe navigation status' };
    }
    return { kind: 'received', data: result.data };
  } catch (error) {
    if (controller.signal.aborted) return { kind: 'aborted' };
    return {
      kind: 'unavailable',
      message: error instanceof Error ? error.message : 'Failed to observe navigation status',
    };
  }
};

const observeNavigationStatusSnapshot = async (
  navigationId: string,
  expectedCommandGeneration: number,
  refs: AINavigationRuntimeRefs,
  statusObservationControllerRef: MutableRefObject<AbortController | null>,
): Promise<void> => {
  const markObservationUnavailable = (message: string): void => {
    if (!refs.isCurrentCommand(navigationId, expectedCommandGeneration)) return;
    refs.handoffCommandInFlightRef.current = false;
    // The server operation may still be running. A transport failure is not
    // an authoritative navigation failure; retain its identity so the user
    // can stop it and a later WebSocket/recovery response can settle it.
    refs.navigationStatusRef.current = 'observation_unavailable';
    refs.updateCurrentCommandState(navigationId, expectedCommandGeneration, (prev) => ({
      ...prev,
      isNavigating: true,
      status: 'observation_unavailable',
      error: message,
    }));
  };

  statusObservationControllerRef.current?.abort();
  const observationController = new AbortController();
  statusObservationControllerRef.current = observationController;

  const observation = await readNavigationObservation(navigationId, observationController);
  if (statusObservationControllerRef.current === observationController) {
    statusObservationControllerRef.current = null;
  }
  if (observation.kind === 'aborted') return;
  if (!refs.isCurrentCommand(navigationId, expectedCommandGeneration)) return;
  if (observation.kind === 'unavailable') {
    markObservationUnavailable(observation.message);
    return;
  }

  const status = parseRecoveryStatus(observation.data.status);
  if (!status) {
    markObservationUnavailable('Received invalid navigation status');
    return;
  }
  if (!isRecoverableNavigationStatus(status, observation.data.terminal)) return;
  applyObservedNavigationStatus(observation.data, status, navigationId, refs);
};

// ============================================================================
// Hook Implementation
// ============================================================================

export function useAINavigation({
  sessionId,
  onStarted,
  onStep,
  onComplete,
}: UseAINavigationOptions): UseAINavigationReturn {
  const runtime = useAINavigationRuntime({ sessionId, onStarted, onStep, onComplete });
  const { state, refs, statusObservationControllerRef } = runtime;

  useAINavigationEvents(sessionId, refs);

  // WebSocket events are the low-latency path. The server-owned status wait is
  // the recovery path when a completion event is missed during reconnect.
  const observeNavigationStatus = useCallback((
    navigationId: string,
    expectedCommandGeneration: number,
  ): void => {
    observeNavigationStatusSnapshot(navigationId, expectedCommandGeneration, refs, statusObservationControllerRef);
  }, [refs, statusObservationControllerRef]);

  const observeCurrentCommand = useCallback((
    navigationId: string,
    generation: number,
    status?: AINavigationState['status'],
  ): void => {
    if (!refs.isCurrentCommand(navigationId, generation)) return;
    if (status) refs.navigationStatusRef.current = status;
    observeNavigationStatus(navigationId, generation);
  }, [observeNavigationStatus, refs]);

  const runHandoffRequest = useCallback(async (
    navigationId: string,
    operation: 'abort' | 'resume',
    request: (id: string, options: RequestOptions) => Promise<ApiResult<void>>,
  ): Promise<number | null> => {
    const commandGeneration = refs.navigationCommandGenerationRef.current + 1;
    refs.navigationCommandGenerationRef.current = commandGeneration;
    refs.handoffCommandInFlightRef.current = true;
    const result = await request(navigationId, { signal: refs.abortControllerRef.current?.signal });
    if (result.success) return commandGeneration;
    if (refs.navigationCommandGenerationRef.current === commandGeneration) {
      refs.handoffCommandInFlightRef.current = false;
    }
    if (operation === 'abort') {
      logger.error('Abort failed', { component: 'useAINavigation' }, new Error(result.error));
    }
    refs.updateCurrentCommandState(navigationId, commandGeneration, (prev) => ({ ...prev, error: result.error }));
    return null;
  }, [refs]);

  const startNavigation = useCallback(async (
    prompt: string,
    model: string,
    maxSteps = 20,
  ): Promise<string | null> => {
    if (!sessionId) {
      refs.setState((prev) => ({ ...prev, error: 'No session available' }));
      return null;
    }
    if (refs.startInFlightRef.current || refs.navigationIdRef.current) {
      refs.setState((prev) => ({ ...prev, error: 'Navigation already in progress' }));
      return null;
    }
    refs.startInFlightRef.current = true;
    refs.navigationStatusRef.current = 'navigating';
    const startAttempt = refs.startAttemptRef.current + 1;
    refs.startAttemptRef.current = startAttempt;
    const requestSignal = refs.abortControllerRef.current?.signal;
    refs.setState((prev) => ({
      ...prev,
      isNavigating: true,
      prompt,
      model,
      steps: [],
      status: 'navigating',
      totalTokens: 0,
      error: null,
    }));
    try {
      const navigationId = await startAINavigationRequest(sessionId, prompt, model, maxSteps, requestSignal);
      return commitStartedNavigation(navigationId, startAttempt, requestSignal, refs, observeNavigationStatus);
    } catch (error) {
      return handleNavigationStartFailure(error, startAttempt, requestSignal, refs);
    }
  }, [observeNavigationStatus, refs, sessionId]);

  const abortNavigation = useCallback(async () => {
    const navigationId = refs.navigationIdRef.current;
    if (!navigationId) {
      logger.warn('Cannot abort: no navigationId in ref', { component: 'useAINavigation' });
      return;
    }
    if (refs.handoffCommandInFlightRef.current) return;
    logger.info('Aborting navigation', { component: 'useAINavigation', navigationId });
    const commandGeneration = await runHandoffRequest(navigationId, 'abort', recordingApi.abortAINavigation.bind(recordingApi));
    if (commandGeneration === null) return;
    logger.info('Abort request sent, waiting for completion', { component: 'useAINavigation' });
    refs.updateCurrentCommandState(navigationId, commandGeneration, (prev) => ({
      ...prev,
      isNavigating: true,
      status: 'aborting',
      humanIntervention: null,
      error: null,
    }));
    observeCurrentCommand(navigationId, commandGeneration, 'aborting');
  }, [observeCurrentCommand, refs, runHandoffRequest]);

  const resumeNavigation = useCallback(async () => {
    const navigationId = refs.navigationIdRef.current;
    if (!navigationId) return false;
    if (refs.navigationStatusRef.current !== 'awaiting_human') {
      refs.setState((prev) => ({ ...prev, error: 'Navigation is not awaiting human intervention' }));
      return false;
    }
    if (refs.handoffCommandInFlightRef.current) return false;
    const commandGeneration = await runHandoffRequest(navigationId, 'resume', recordingApi.resumeAINavigation.bind(recordingApi));
    if (commandGeneration === null) return false;
    refs.updateCurrentCommandState(navigationId, commandGeneration, (prev) => ({ ...prev, error: null }));
    observeCurrentCommand(navigationId, commandGeneration);
    return refs.navigationIdRef.current === navigationId;
  }, [observeCurrentCommand, refs, runHandoffRequest]);

  return {
    state,
    startNavigation,
    abortNavigation,
    resumeNavigation,
    reset: runtime.resetNavigation,
    availableModels: VISION_MODELS,
    isNavigating: state.isNavigating,
    isAwaitingHuman: state.status === 'awaiting_human',
  };
}
