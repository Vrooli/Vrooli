/**
 * Owns WebSocket event admission and state projection for AI navigation.
 *
 * The unified navigation hook owns command admission and server-status
 * recovery. This hook owns only the low-latency event path
 * and its session/generation fences.
 */

import { useWebSocketMessage } from '@/contexts/WebSocketContext';
import { logger } from '@/utils/logger';
import type {
  AINavigationAwaitingHumanEvent,
  AINavigationCompleteEvent,
  AINavigationResumedEvent,
  AINavigationState,
  AINavigationStep,
  AINavigationStepEvent,
} from './types';
import type { AINavigationRuntimeRefs } from './runtimeRefs';
import { isRecord, parseNavigationEvent } from './navigationEvents';

const isCurrentEvent = (
  sessionId: string | null,
  refs: AINavigationRuntimeRefs,
  navigationId: string,
  eventSessionId: string,
): boolean => navigationId === refs.navigationIdRef.current && eventSessionId === sessionId;

const admitHandoffEvent = (
  refs: AINavigationRuntimeRefs,
  status: AINavigationState['status'],
): void => {
  refs.cancelStatusObservation();
  refs.navigationCommandGenerationRef.current += 1;
  refs.handoffCommandInFlightRef.current = false;
  refs.navigationStatusRef.current = status;
};

const handleStepEvent = (
  sessionId: string | null,
  refs: AINavigationRuntimeRefs,
  event: AINavigationStepEvent,
): void => {
  if (!isCurrentEvent(sessionId, refs, event.navigationId, event.sessionId)) return;
  if (refs.seenStepNumbersRef.current.has(event.stepNumber)) return;

  const step: AINavigationStep = {
    id: `step-${event.stepNumber}`,
    stepNumber: event.stepNumber,
    action: event.action,
    reasoning: event.reasoning,
    currentUrl: event.currentUrl,
    goalAchieved: event.goalAchieved,
    tokensUsed: event.tokensUsed,
    durationMs: event.durationMs,
    error: event.error,
    timestamp: new Date(event.timestamp),
  };

  refs.seenStepNumbersRef.current.add(event.stepNumber);
  const recoveredFromObservationLoss = refs.navigationStatusRef.current === 'observation_unavailable';
  if (recoveredFromObservationLoss) refs.navigationStatusRef.current = 'navigating';
  refs.setState((prev) => ({
    ...prev,
    ...(recoveredFromObservationLoss ? {
      isNavigating: true,
      status: 'navigating' as const,
      error: null,
    } : {}),
    steps: [...prev.steps, step],
    totalTokens: prev.totalTokens + event.tokensUsed.totalTokens,
  }));
  refs.onStepRef.current?.(step, event.navigationId);
};

const handleCompleteEvent = (
  sessionId: string | null,
  refs: AINavigationRuntimeRefs,
  event: AINavigationCompleteEvent,
): void => {
  logger.debug('Received complete event', {
    component: 'useAINavigation',
    eventNavigationId: event.navigationId,
    currentNavigationId: refs.navigationIdRef.current,
    eventStatus: event.status,
  });
  if (!isCurrentEvent(sessionId, refs, event.navigationId, event.sessionId)) {
    logger.debug('Ignoring complete event - navigationId mismatch', { component: 'useAINavigation' });
    return;
  }

  admitHandoffEvent(refs, event.status);
  refs.setState((prev) => ({
    ...prev,
    navigationId: null,
    isNavigating: false,
    status: event.status,
    totalTokens: event.totalTokens,
    totalDurationMs: event.totalDurationMs,
    finalUrl: event.finalUrl,
    error: event.error ?? null,
    humanIntervention: null,
  }));
  refs.navigationIdRef.current = null;
  refs.onCompleteRef.current?.(event.status, event.summary, event.navigationId);
};

const handleAwaitingHumanEvent = (
  sessionId: string | null,
  refs: AINavigationRuntimeRefs,
  event: AINavigationAwaitingHumanEvent,
): void => {
  if (!isCurrentEvent(sessionId, refs, event.navigationId, event.sessionId)) return;
  admitHandoffEvent(refs, 'awaiting_human');
  refs.setState((prev) => ({
    ...prev,
    isNavigating: false,
    status: 'awaiting_human',
    humanIntervention: {
      reason: event.reason,
      instructions: event.instructions,
      interventionType: event.interventionType,
      trigger: event.trigger,
      startedAt: new Date(event.timestamp),
    },
  }));
};

const handleResumedEvent = (
  sessionId: string | null,
  refs: AINavigationRuntimeRefs,
  event: AINavigationResumedEvent,
): void => {
  if (!isCurrentEvent(sessionId, refs, event.navigationId, event.sessionId)) return;
  admitHandoffEvent(refs, 'navigating');
  refs.setState((prev) => ({
    ...prev,
    isNavigating: true,
    status: 'navigating',
    humanIntervention: null,
  }));
};

export function useAINavigationEvents(
  sessionId: string | null,
  refs: AINavigationRuntimeRefs,
): void {
  useWebSocketMessage((lastMessage) => {
    const messageType = isRecord(lastMessage) && typeof lastMessage.type === 'string'
      ? lastMessage.type
      : null;
    if (messageType?.startsWith('ai_navigation')) {
      logger.debug('WebSocket message received', {
        component: 'useAINavigation',
        messageType,
      });
    }
    const event = parseNavigationEvent(lastMessage);
    if (!event) return;

    switch (event.type) {
      case 'ai_navigation_step':
        handleStepEvent(sessionId, refs, event);
        break;
      case 'ai_navigation_complete':
        handleCompleteEvent(sessionId, refs, event);
        break;
      case 'ai_navigation_awaiting_human':
        handleAwaitingHumanEvent(sessionId, refs, event);
        break;
      case 'ai_navigation_resumed':
        handleResumedEvent(sessionId, refs, event);
        break;
    }
  });
}
