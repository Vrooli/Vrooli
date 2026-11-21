import { useCallback, useEffect, useRef, useState } from 'react';
import type { AINavigationState } from './types';
import type { AINavigationRuntimeRefs } from './runtimeRefs';

export const initialAINavigationState: AINavigationState = {
  isNavigating: false,
  navigationId: null,
  prompt: '',
  model: 'local_first',
  steps: [],
  status: 'idle',
  totalTokens: 0,
  totalDurationMs: 0,
  finalUrl: '',
  verifiedSuccess: false,
  extractedData: null,
  verificationError: null,
  error: null,
  humanIntervention: null,
};

interface UseAINavigationRuntimeOptions {
  sessionId: string | null;
  onStarted?: (navigationId: string, startAttempt?: number) => void;
  onStep?: AINavigationRuntimeRefs['onStepRef']['current'];
  onComplete?: AINavigationRuntimeRefs['onCompleteRef']['current'];
}

/** Owns navigation identity, cancellation and the shared command/event refs. */
export function useAINavigationRuntime({
  sessionId,
  onStarted,
  onStep,
  onComplete,
}: UseAINavigationRuntimeOptions) {
  const [state, setState] = useState<AINavigationState>(initialAINavigationState);
  const navigationIdRef = useRef<string | null>(null);
  const navigationStatusRef = useRef<AINavigationState['status']>(initialAINavigationState.status);
  const startInFlightRef = useRef(false);
  const startAttemptRef = useRef(0);
  const navigationCommandGenerationRef = useRef(0);
  const handoffCommandInFlightRef = useRef(false);
  const seenStepNumbersRef = useRef(new Set<number>());
  const onStepRef = useRef(onStep);
  const onCompleteRef = useRef(onComplete);
  const onStartedRef = useRef(onStarted);
  const abortControllerRef = useRef<AbortController | null>(null);
  const statusObservationControllerRef = useRef<AbortController | null>(null);

  onStepRef.current = onStep;
  onCompleteRef.current = onComplete;
  onStartedRef.current = onStarted;

  const isCurrentCommand = useCallback((navigationId: string, generation: number): boolean => (
    navigationIdRef.current === navigationId
    && navigationCommandGenerationRef.current === generation
  ), []);

  const updateCurrentCommandState = useCallback((
    navigationId: string,
    generation: number,
    update: (previous: AINavigationState) => AINavigationState,
  ): void => {
    setState((previous) => (
      isCurrentCommand(navigationId, generation) ? update(previous) : previous
    ));
  }, [isCurrentCommand]);

  const cancelStatusObservation = useCallback(() => {
    const controller = statusObservationControllerRef.current;
    statusObservationControllerRef.current = null;
    controller?.abort();
  }, []);

  const resetNavigation = useCallback(() => {
    cancelStatusObservation();
    abortControllerRef.current?.abort();
    abortControllerRef.current = new AbortController();
    startAttemptRef.current += 1;
    startInFlightRef.current = false;
    navigationCommandGenerationRef.current += 1;
    handoffCommandInFlightRef.current = false;
    navigationIdRef.current = null;
    navigationStatusRef.current = initialAINavigationState.status;
    seenStepNumbersRef.current.clear();
    setState(initialAINavigationState);
  }, [cancelStatusObservation]);

  useEffect(() => {
    resetNavigation();

    return () => {
      cancelStatusObservation();
      abortControllerRef.current?.abort();
    };
  }, [cancelStatusObservation, resetNavigation, sessionId]);

  const refs = useRef<AINavigationRuntimeRefs>({
    navigationIdRef,
    navigationStatusRef,
    handoffCommandInFlightRef,
    seenStepNumbersRef,
    onStepRef,
    onCompleteRef,
    navigationCommandGenerationRef,
    isCurrentCommand,
    updateCurrentCommandState,
    cancelStatusObservation,
    setState,
    startInFlightRef,
    startAttemptRef,
    abortControllerRef,
    onStartedRef,
  }).current;

  return {
    state,
    refs,
    statusObservationControllerRef,
    resetNavigation,
  };
}
