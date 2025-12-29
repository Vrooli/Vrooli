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
import { getApiBase } from '@/config';
import { useWebSocket } from '@/contexts/WebSocketContext';
import type {
  AINavigateRequest,
  AINavigateResponse,
  AINavigationState,
  AINavigationStep,
  AINavigationStepEvent,
  AINavigationCompleteEvent,
  AINavigationAwaitingHumanEvent,
  AINavigationResumedEvent,
  VisionModelSpec,
} from './types';
import { VISION_MODELS } from './types';

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
  /** Start AI navigation with a prompt */
  startNavigation: (prompt: string, model: string, maxSteps?: number) => Promise<void>;
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

export function useAINavigation({
  sessionId,
  onStep,
  onComplete,
}: UseAINavigationOptions): UseAINavigationReturn {
  const apiUrl = getApiBase();
  const [state, setState] = useState<AINavigationState>(initialState);
  const { lastMessage } = useWebSocket();

  // Refs to track current navigation
  const navigationIdRef = useRef<string | null>(null);
  const onStepRef = useRef(onStep);
  const onCompleteRef = useRef(onComplete);
  onStepRef.current = onStep;
  onCompleteRef.current = onComplete;

  // Process WebSocket messages
  useEffect(() => {
    if (!lastMessage) return;

    const msg = lastMessage as unknown as Record<string, unknown>;

    // Handle AI navigation step events
    if (msg.type === 'ai_navigation_step') {
      const event = msg as unknown as AINavigationStepEvent;

      // Only process events for our current navigation
      if (event.navigationId !== navigationIdRef.current) return;

      // Defensive defaults for potentially missing fields
      const tokensUsed = event.tokensUsed ?? { promptTokens: 0, completionTokens: 0, totalTokens: 0 };
      const stepNumber = event.stepNumber ?? 0;
      const action = event.action ?? { type: 'wait' as const };
      const timestamp = event.timestamp ? new Date(event.timestamp) : new Date();

      const step: AINavigationStep = {
        id: `step-${stepNumber}`,
        stepNumber,
        action,
        reasoning: event.reasoning ?? '',
        currentUrl: event.currentUrl ?? '',
        goalAchieved: event.goalAchieved ?? false,
        tokensUsed,
        durationMs: event.durationMs ?? 0,
        error: event.error,
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
    if (msg.type === 'ai_navigation_complete') {
      const event = msg as unknown as AINavigationCompleteEvent;

      // Only process events for our current navigation
      if (event.navigationId !== navigationIdRef.current) return;

      // Defensive defaults for potentially missing fields
      const status = event.status ?? 'completed';
      const totalTokens = event.totalTokens ?? 0;

      setState((prev) => ({
        ...prev,
        isNavigating: false,
        status,
        totalTokens,
        error: event.error ?? null,
        humanIntervention: null,
      }));

      navigationIdRef.current = null;
      onCompleteRef.current?.(status, event.summary);
    }

    // Handle AI navigation awaiting human intervention events
    if (msg.type === 'ai_navigation_awaiting_human') {
      const event = msg as unknown as AINavigationAwaitingHumanEvent;

      // Only process events for our current navigation
      if (event.navigationId !== navigationIdRef.current) return;

      setState((prev) => ({
        ...prev,
        status: 'awaiting_human',
        humanIntervention: {
          reason: event.reason,
          instructions: event.instructions,
          interventionType: event.interventionType,
          trigger: event.trigger,
          startedAt: new Date(event.timestamp),
        },
      }));
    }

    // Handle AI navigation resumed events
    if (msg.type === 'ai_navigation_resumed') {
      const event = msg as unknown as AINavigationResumedEvent;

      // Only process events for our current navigation
      if (event.navigationId !== navigationIdRef.current) return;

      setState((prev) => ({
        ...prev,
        status: 'navigating',
        humanIntervention: null,
      }));
    }
  }, [lastMessage]);

  // Reset state when session changes
  useEffect(() => {
    setState(initialState);
    navigationIdRef.current = null;
  }, [sessionId]);

  const startNavigation = useCallback(
    async (prompt: string, model: string, maxSteps = 20) => {
      if (!sessionId) {
        setState((prev) => ({
          ...prev,
          error: 'No session available',
        }));
        return;
      }

      if (state.isNavigating) {
        setState((prev) => ({
          ...prev,
          error: 'Navigation already in progress',
        }));
        return;
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

        const response = await fetch(`${apiUrl}/ai-navigate`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            session_id: request.sessionId,
            prompt: request.prompt,
            model: request.model,
            max_steps: request.maxSteps,
          }),
        });

        if (!response.ok) {
          const errorData = await response.json().catch(() => ({}));
          throw new Error(errorData.message || `Failed to start navigation: ${response.statusText}`);
        }

        // API returns snake_case, convert to camelCase
        const rawData = await response.json();
        const data: AINavigateResponse = {
          navigationId: rawData.navigation_id,
          status: rawData.status,
          model: rawData.model,
          maxSteps: rawData.max_steps,
          estimatedCost: rawData.estimated_cost,
        };
        navigationIdRef.current = data.navigationId;

        setState((prev) => ({
          ...prev,
          navigationId: data.navigationId,
        }));
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
    [apiUrl, sessionId, state.isNavigating]
  );

  const abortNavigation = useCallback(async () => {
    if (!state.navigationId) {
      console.warn('[useAINavigation] Cannot abort: no navigationId');
      return;
    }

    try {
      const response = await fetch(`${apiUrl}/ai-navigate/${state.navigationId}/abort`, {
        method: 'POST',
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || 'Failed to abort navigation');
      }

      // Mark as aborted immediately and set isNavigating to false
      // Don't wait for WebSocket complete event since it may be delayed
      setState((prev) => ({
        ...prev,
        isNavigating: false,
        status: 'aborted',
        humanIntervention: null,
      }));
      navigationIdRef.current = null;

      // Trigger onComplete callback so message UI updates
      onCompleteRef.current?.('aborted', 'Navigation was aborted');
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to abort navigation';
      console.error('[useAINavigation] Abort failed:', message);
      setState((prev) => ({
        ...prev,
        error: message,
      }));
    }
  }, [apiUrl, state.navigationId]);

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

    try {
      const response = await fetch(`${apiUrl}/ai-navigate/${state.navigationId}/resume`, {
        method: 'POST',
      });

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}));
        throw new Error(errorData.message || 'Failed to resume navigation');
      }

      // The WebSocket resumed event will clear humanIntervention
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to resume navigation';
      setState((prev) => ({
        ...prev,
        error: message,
      }));
    }
  }, [apiUrl, state.navigationId, state.status]);

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
