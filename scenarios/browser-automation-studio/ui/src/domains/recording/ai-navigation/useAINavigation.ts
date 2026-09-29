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
import { recordingApi } from '../api';
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
  statusStepToAINavigationStep,
} from './navigationEvents';
import { useAINavigationEvents } from './useAINavigationEvents';
import { useAINavigationCommands } from './useAINavigationCommands';
import { useAINavigationRuntime } from './useAINavigationRuntime';

export { AINavigationError } from './useAINavigationCommands';

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

  const { startNavigation, abortNavigation, resumeNavigation } = useAINavigationCommands({
    sessionId,
    refs,
    observeNavigationStatus,
  });

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
