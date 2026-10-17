import type { Dispatch, MutableRefObject, SetStateAction } from 'react';
import type { AINavigationState, AINavigationStep } from './types';

/** Client projection refs for server-owned navigation identity and state. */
export interface AINavigationRuntimeRefs {
  navigationIdRef: MutableRefObject<string | null>;
  navigationStatusRef: MutableRefObject<AINavigationState['status']>;
  handoffCommandInFlightRef: MutableRefObject<boolean>;
  seenStepNumbersRef: MutableRefObject<Set<number>>;
  onStepRef: MutableRefObject<((step: AINavigationStep, navigationId?: string) => void) | undefined>;
  onCompleteRef: MutableRefObject<((status: string, summary?: string, navigationId?: string) => void) | undefined>;
  setState: Dispatch<SetStateAction<AINavigationState>>;
  isCurrentCommand: (navigationId: string, generation: number) => boolean;
  updateCurrentCommandState: (
    navigationId: string,
    generation: number,
    update: (previous: AINavigationState) => AINavigationState,
  ) => void;
  navigationCommandGenerationRef: MutableRefObject<number>;
  cancelStatusObservation: () => void;
  startInFlightRef: MutableRefObject<boolean>;
  startAttemptRef: MutableRefObject<number>;
  abortControllerRef: MutableRefObject<AbortController | null>;
  onStartedRef: MutableRefObject<((navigationId: string, startAttempt?: number) => void) | undefined>;
}
