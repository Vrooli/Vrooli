/**
 * useAINavigation Hook
 *
 * Manages AI-driven browser navigation sessions.
 * - Starts navigation via API
 * - Subscribes to WebSocket events for real-time updates
 * - Tracks step history for timeline display
 * - Handles abort functionality
 */

import { useState, useCallback, useEffect, useRef } from 'react';
import { useWebSocket } from '@/contexts/WebSocketContext';
import { getAIRequestHeadersSync } from '@/utils/apiHeaders';
import { recordingApi } from '../api';
import { logger } from '@/utils/logger';
import type {
  AINavigateRequest,
  AINavigationState,
  AINavigationStep,
  AINavigationStepEvent,
  AINavigationCompleteEvent,
  AINavigationAwaitingHumanEvent,
  AINavigationResumedEvent,
  VisionModelSpec,
  BrowserAction,
  TokenUsage,
} from './types';
import { VISION_MODELS } from './types';

// ============================================================================
// Helper Functions for WebSocket Message Parsing
// ============================================================================

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

const actionTypes = new Set<BrowserAction['type']>([
  'click',
  'type',
  'scroll',
  'navigate',
  'hover',
  'select',
  'wait',
  'keypress',
  'done',
  'request_human',
]);

const directionTypes = new Set<NonNullable<BrowserAction['direction']>>([
  'up',
  'down',
  'left',
  'right',
]);

const interventionTypes = new Set<NonNullable<BrowserAction['interventionType']>>([
  'captcha',
  'verification',
  'complex_interaction',
  'login_required',
  'other',
]);

const triggerTypes = new Set<AINavigationAwaitingHumanEvent['trigger']>([
  'programmatic',
  'ai_requested',
]);

const statusTypes = new Set<AINavigationCompleteEvent['status']>([
  'completed',
  'failed',
  'aborted',
  'max_steps_reached',
  'loop_detected',
  'awaiting_human',
]);

const parseTokensUsed = (value: unknown): TokenUsage => {
  if (!isRecord(value)) {
    return { promptTokens: 0, completionTokens: 0, totalTokens: 0 };
  }
  const promptTokens = typeof value.promptTokens === 'number' ? value.promptTokens : 0;
  const completionTokens = typeof value.completionTokens === 'number' ? value.completionTokens : 0;
  const totalTokens = typeof value.totalTokens === 'number' ? value.totalTokens : 0;
  return { promptTokens, completionTokens, totalTokens };
};

const parseBrowserAction = (value: unknown): BrowserAction => {
  if (!isRecord(value) || typeof value.type !== 'string' || !actionTypes.has(value.type as BrowserAction['type'])) {
    return { type: 'wait' };
  }

  const action: BrowserAction = { type: value.type as BrowserAction['type'] };

  if (typeof value.elementId === 'number') {
    action.elementId = value.elementId;
  }
  if (isRecord(value.coordinates) && typeof value.coordinates.x === 'number' && typeof value.coordinates.y === 'number') {
    action.coordinates = { x: value.coordinates.x, y: value.coordinates.y };
  }
  if (typeof value.text === 'string') {
    action.text = value.text;
  }
  if (typeof value.direction === 'string' && directionTypes.has(value.direction as NonNullable<BrowserAction['direction']>)) {
    action.direction = value.direction as NonNullable<BrowserAction['direction']>;
  }
  if (typeof value.url === 'string') {
    action.url = value.url;
  }
  if (typeof value.key === 'string') {
    action.key = value.key;
  }
  if (typeof value.result === 'string') {
    action.result = value.result;
  }
  if (typeof value.success === 'boolean') {
    action.success = value.success;
  }
  if (typeof value.reason === 'string') {
    action.reason = value.reason;
  }
  if (typeof value.instructions === 'string') {
    action.instructions = value.instructions;
  }
  if (typeof value.interventionType === 'string' && interventionTypes.has(value.interventionType as NonNullable<BrowserAction['interventionType']>)) {
    action.interventionType = value.interventionType as NonNullable<BrowserAction['interventionType']>;
  }
  return action;
};

const parseStepEvent = (value: unknown): AINavigationStepEvent | null => {
  if (!isRecord(value) || value.type !== 'ai_navigation_step') return null;
  if (typeof value.navigationId !== 'string' || typeof value.sessionId !== 'string') return null;
  const stepNumber = typeof value.stepNumber === 'number' ? value.stepNumber : 0;
  const action = parseBrowserAction(value.action);
  const reasoning = typeof value.reasoning === 'string' ? value.reasoning : '';
  const currentUrl = typeof value.currentUrl === 'string' ? value.currentUrl : '';
  const goalAchieved = typeof value.goalAchieved === 'boolean' ? value.goalAchieved : false;
  const tokensUsed = parseTokensUsed(value.tokensUsed);
  const durationMs = typeof value.durationMs === 'number' ? value.durationMs : 0;
  const error = typeof value.error === 'string' ? value.error : undefined;
  const timestamp = typeof value.timestamp === 'string' ? value.timestamp : new Date().toISOString();

  return {
    type: 'ai_navigation_step',
    navigationId: value.navigationId,
    sessionId: value.sessionId,
    stepNumber,
    action,
    reasoning,
    currentUrl,
    goalAchieved,
    tokensUsed,
    durationMs,
    error,
    timestamp,
  };
};

const parseCompleteEvent = (value: unknown): AINavigationCompleteEvent | null => {
  if (!isRecord(value) || value.type !== 'ai_navigation_complete') return null;
  if (typeof value.navigationId !== 'string' || typeof value.sessionId !== 'string') return null;
  const status = typeof value.status === 'string' && statusTypes.has(value.status as AINavigationCompleteEvent['status'])
    ? (value.status as AINavigationCompleteEvent['status'])
    : 'completed';
  const totalSteps = typeof value.totalSteps === 'number' ? value.totalSteps : 0;
  const totalTokens = typeof value.totalTokens === 'number' ? value.totalTokens : 0;
  const totalDurationMs = typeof value.totalDurationMs === 'number' ? value.totalDurationMs : 0;
  const finalUrl = typeof value.finalUrl === 'string' ? value.finalUrl : '';
  const error = typeof value.error === 'string' ? value.error : undefined;
  const summary = typeof value.summary === 'string' ? value.summary : undefined;
  const timestamp = typeof value.timestamp === 'string' ? value.timestamp : new Date().toISOString();

  return {
    type: 'ai_navigation_complete',
    navigationId: value.navigationId,
    sessionId: value.sessionId,
    status,
    totalSteps,
    totalTokens,
    totalDurationMs,
    finalUrl,
    error,
    summary,
    timestamp,
  };
};

const parseAwaitingHumanEvent = (value: unknown): AINavigationAwaitingHumanEvent | null => {
  if (!isRecord(value) || value.type !== 'ai_navigation_awaiting_human') return null;
  if (typeof value.navigationId !== 'string' || typeof value.sessionId !== 'string') return null;
  const stepNumber = typeof value.stepNumber === 'number' ? value.stepNumber : 0;
  const reason = typeof value.reason === 'string' ? value.reason : 'Human intervention required';
  const instructions = typeof value.instructions === 'string' ? value.instructions : undefined;
  const interventionType = typeof value.interventionType === 'string' && interventionTypes.has(value.interventionType as AINavigationAwaitingHumanEvent['interventionType'])
    ? (value.interventionType as AINavigationAwaitingHumanEvent['interventionType'])
    : 'other';
  const trigger = typeof value.trigger === 'string' && triggerTypes.has(value.trigger as AINavigationAwaitingHumanEvent['trigger'])
    ? (value.trigger as AINavigationAwaitingHumanEvent['trigger'])
    : 'programmatic';
  const timestamp = typeof value.timestamp === 'string' ? value.timestamp : new Date().toISOString();

  return {
    type: 'ai_navigation_awaiting_human',
    navigationId: value.navigationId,
    sessionId: value.sessionId,
    stepNumber,
    reason,
    instructions,
    interventionType,
    trigger,
    timestamp,
  };
};

const parseResumedEvent = (value: unknown): AINavigationResumedEvent | null => {
  if (!isRecord(value) || value.type !== 'ai_navigation_resumed') return null;
  if (typeof value.navigationId !== 'string' || typeof value.sessionId !== 'string') return null;
  const timestamp = typeof value.timestamp === 'string' ? value.timestamp : new Date().toISOString();
  return {
    type: 'ai_navigation_resumed',
    navigationId: value.navigationId,
    sessionId: value.sessionId,
    timestamp,
  };
};

// ============================================================================
// Custom Error Class
// ============================================================================

/**
 * Custom error class for AI navigation errors.
 * Includes error code and additional details from the API.
 */
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

// ============================================================================
// Hook Interface
// ============================================================================

interface UseAINavigationOptions {
  sessionId: string | null;
  /** Callback when a step is received */
  onStep?: (step: AINavigationStep) => void;
  /** Callback when navigation completes */
  onComplete?: (status: string, summary?: string) => void;
}

interface UseAINavigationReturn {
  /** Current navigation state */
  state: AINavigationState;
  /** Start AI navigation with a prompt. Returns the navigationId on success, null on failure. */
  startNavigation: (prompt: string, model: string, maxSteps?: number) => Promise<string | null>;
  /** Abort the current navigation */
  abortNavigation: () => Promise<void>;
  /** Resume navigation after human intervention */
  resumeNavigation: () => Promise<void>;
  /** Reset the navigation state */
  reset: () => void;
  /** Available vision models */
  availableModels: VisionModelSpec[];
  /** Whether navigation is in progress */
  isNavigating: boolean;
  /** Whether navigation is awaiting human intervention */
  isAwaitingHuman: boolean;
}

const initialState: AINavigationState = {
  isNavigating: false,
  navigationId: null,
  prompt: '',
  model: 'qwen3-vl-30b',
  steps: [],
  status: 'idle',
  totalTokens: 0,
  error: null,
  humanIntervention: null,
};

// ============================================================================
// Hook Implementation
// ============================================================================

export function useAINavigation({
  sessionId,
  onStep,
  onComplete,
}: UseAINavigationOptions): UseAINavigationReturn {
  const [state, setState] = useState<AINavigationState>(initialState);
  const { lastMessage } = useWebSocket();

  // Refs to track current navigation
  const navigationIdRef = useRef<string | null>(null);
  const onStepRef = useRef(onStep);
  const onCompleteRef = useRef(onComplete);
  onStepRef.current = onStep;
  onCompleteRef.current = onComplete;

  // AbortController for request cancellation
  const abortControllerRef = useRef<AbortController | null>(null);

  // Clean up on unmount
  useEffect(() => {
    return () => {
      abortControllerRef.current?.abort();
    };
  }, []);

  // Process WebSocket messages
  useEffect(() => {
    if (!lastMessage) return;

    if (!isRecord(lastMessage) || typeof lastMessage.type !== 'string') return;
    const msg = lastMessage;

    // Log all AI navigation related messages
    if (typeof msg.type === 'string' && msg.type.startsWith('ai_navigation')) {
      logger.debug('WebSocket message received', {
        component: 'useAINavigation',
        messageType: msg.type,
      });
    }

    // Handle AI navigation step events
    const stepEvent = parseStepEvent(msg);
    if (stepEvent) {

      // Only process events for our current navigation
      if (stepEvent.navigationId !== navigationIdRef.current) return;

      // Defensive defaults for potentially missing fields
      const tokensUsed = stepEvent.tokensUsed ?? { promptTokens: 0, completionTokens: 0, totalTokens: 0 };
      const stepNumber = stepEvent.stepNumber ?? 0;
      const action = stepEvent.action ?? { type: 'wait' as const };
      const timestamp = stepEvent.timestamp ? new Date(stepEvent.timestamp) : new Date();

      const step: AINavigationStep = {
        id: `step-${stepNumber}`,
        stepNumber,
        action,
        reasoning: stepEvent.reasoning ?? '',
        currentUrl: stepEvent.currentUrl ?? '',
        goalAchieved: stepEvent.goalAchieved ?? false,
        tokensUsed,
        durationMs: stepEvent.durationMs ?? 0,
        error: stepEvent.error,
        timestamp,
      };

      setState((prev) => ({
        ...prev,
        steps: [...prev.steps, step],
        totalTokens: prev.totalTokens + tokensUsed.totalTokens,
      }));

      onStepRef.current?.(step);
    }

    // Handle AI navigation complete events
    const completeEvent = parseCompleteEvent(msg);
    if (completeEvent) {

      logger.debug('Received complete event', {
        component: 'useAINavigation',
        eventNavigationId: completeEvent.navigationId,
        currentNavigationId: navigationIdRef.current,
        eventStatus: completeEvent.status,
      });

      // Only process events for our current navigation
      if (completeEvent.navigationId !== navigationIdRef.current) {
        logger.debug('Ignoring complete event - navigationId mismatch', { component: 'useAINavigation' });
        return;
      }

      // Defensive defaults for potentially missing fields
      const status = completeEvent.status ?? 'completed';
      const totalTokens = completeEvent.totalTokens ?? 0;

      logger.debug('Processing complete event', {
        component: 'useAINavigation',
        status,
      });

      setState((prev) => ({
        ...prev,
        isNavigating: false,
        status,
        totalTokens,
        error: completeEvent.error ?? null,
        humanIntervention: null,
      }));

      navigationIdRef.current = null;
      onCompleteRef.current?.(status, completeEvent.summary);
    }

    // Handle AI navigation awaiting human intervention events
    const awaitingEvent = parseAwaitingHumanEvent(msg);
    if (awaitingEvent) {

      // Only process events for our current navigation
      if (awaitingEvent.navigationId !== navigationIdRef.current) return;

      setState((prev) => ({
        ...prev,
        status: 'awaiting_human',
        humanIntervention: {
          reason: awaitingEvent.reason,
          instructions: awaitingEvent.instructions,
          interventionType: awaitingEvent.interventionType,
          trigger: awaitingEvent.trigger,
          startedAt: new Date(awaitingEvent.timestamp),
        },
      }));
    }

    // Handle AI navigation resumed events
    const resumedEvent = parseResumedEvent(msg);
    if (resumedEvent) {

      // Only process events for our current navigation
      if (resumedEvent.navigationId !== navigationIdRef.current) return;

      setState((prev) => ({
        ...prev,
        status: 'navigating',
        humanIntervention: null,
      }));
    }
  }, [lastMessage]);

  // Reset state when session changes
  useEffect(() => {
    // Abort any pending requests when session changes
    abortControllerRef.current?.abort();
    abortControllerRef.current = new AbortController();

    setState(initialState);
    navigationIdRef.current = null;
  }, [sessionId]);

  const startNavigation = useCallback(
    async (prompt: string, model: string, maxSteps = 20): Promise<string | null> => {
      if (!sessionId) {
        setState((prev) => ({
          ...prev,
          error: 'No session available',
        }));
        return null;
      }

      if (state.isNavigating) {
        setState((prev) => ({
          ...prev,
          error: 'Navigation already in progress',
        }));
        return null;
      }

      setState((prev) => ({
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
        const request: AINavigateRequest = {
          sessionId,
          prompt,
          model,
          maxSteps,
        };

        const result = await recordingApi.startAINavigation(
          {
            sessionId: request.sessionId,
            prompt: request.prompt,
            model: request.model,
            maxSteps: request.maxSteps,
          },
          getAIRequestHeadersSync(),
          { signal: abortControllerRef.current?.signal }
        );

        if (!result.success) {
          // Parse enriched error from the service
          let code = 'UNKNOWN_ERROR';
          let message = result.error;
          let details: Record<string, string> | undefined;

          try {
            const errorData: unknown = JSON.parse(result.error);
            if (isRecord(errorData)) {
              if (typeof errorData.code === 'string') code = errorData.code;
              if (typeof errorData.message === 'string') message = errorData.message;
              if (isRecord(errorData.details)) {
                // Validate that all values are strings
                const d = errorData.details;
                const allStrings = Object.values(d).every((v) => typeof v === 'string');
                if (allStrings) {
                  details = d as Record<string, string>;
                }
              }
            }
          } catch {
            // Not JSON, use raw error
          }

          throw new AINavigationError(code, message, details);
        }

        navigationIdRef.current = result.data.navigation_id;

        setState((prev) => ({
          ...prev,
          navigationId: result.data.navigation_id,
        }));

        return result.data.navigation_id;
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to start navigation';
        setState((prev) => ({
          ...prev,
          isNavigating: false,
          status: 'failed',
          error: message,
        }));
        throw err;
      }
    },
    [sessionId, state.isNavigating]
  );

  const abortNavigation = useCallback(async () => {
    // Use ref for the most current navigationId (avoids stale closure issues)
    const navId = navigationIdRef.current;
    if (!navId) {
      logger.warn('Cannot abort: no navigationId in ref', { component: 'useAINavigation' });
      return;
    }

    logger.info('Aborting navigation', { component: 'useAINavigation', navigationId: navId });

    const result = await recordingApi.abortAINavigation(navId, {
      signal: abortControllerRef.current?.signal,
    });

    if (!result.success) {
      logger.error('Abort failed', { component: 'useAINavigation' }, new Error(result.error));
      setState((prev) => ({
        ...prev,
        error: result.error,
      }));
      return;
    }

    logger.info('Abort request sent, waiting for completion', { component: 'useAINavigation' });

    // Set status to 'aborting' - navigation is still in progress until server confirms
    // The WebSocket ai_navigation_complete event will set the final 'aborted' status
    setState((prev) => ({
      ...prev,
      status: 'aborting',
      humanIntervention: null,
    }));

    // Note: Don't set isNavigating to false or clear navigationIdRef yet
    // The WebSocket complete event will handle final cleanup
  }, []);

  const resumeNavigation = useCallback(async () => {
    if (!state.navigationId) {
      return;
    }

    if (state.status !== 'awaiting_human') {
      setState((prev) => ({
        ...prev,
        error: 'Navigation is not awaiting human intervention',
      }));
      return;
    }

    const result = await recordingApi.resumeAINavigation(state.navigationId, {
      signal: abortControllerRef.current?.signal,
    });

    if (!result.success) {
      setState((prev) => ({
        ...prev,
        error: result.error,
      }));
      return;
    }

    // The WebSocket resumed event will clear humanIntervention
  }, [state.navigationId, state.status]);

  const reset = useCallback(() => {
    setState(initialState);
    navigationIdRef.current = null;
  }, []);

  return {
    state,
    startNavigation,
    abortNavigation,
    resumeNavigation,
    reset,
    availableModels: VISION_MODELS,
    isNavigating: state.isNavigating,
    isAwaitingHuman: state.status === 'awaiting_human',
  };
}
