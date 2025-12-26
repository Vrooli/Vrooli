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
  /** Reset the navigation state */
  reset: () => void;
  /** Available vision models */
  availableModels: VisionModelSpec[];
  /** Whether navigation is in progress */
  isNavigating: boolean;
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

      setState((prev) => ({
        ...prev,
        steps: [...prev.steps, step],
        totalTokens: prev.totalTokens + event.tokensUsed.totalTokens,
      }));

      onStepRef.current?.(step);
    }

    // Handle AI navigation complete events
    if (msg.type === 'ai_navigation_complete') {
      const event = msg as unknown as AINavigationCompleteEvent;

      // Only process events for our current navigation
      if (event.navigationId !== navigationIdRef.current) return;

      setState((prev) => ({
        ...prev,
        isNavigating: false,
        status: event.status,
        totalTokens: event.totalTokens,
        error: event.error ?? null,
      }));

      navigationIdRef.current = null;
      onCompleteRef.current?.(event.status, event.summary);
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

        const data: AINavigateResponse = await response.json();
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

      setState((prev) => ({
        ...prev,
        status: 'aborted',
      }));
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to abort navigation';
      setState((prev) => ({
        ...prev,
        error: message,
      }));
    }
  }, [apiUrl, state.navigationId]);

  const reset = useCallback(() => {
    setState(initialState);
    navigationIdRef.current = null;
  }, []);

  return {
    state,
    startNavigation,
    abortNavigation,
    reset,
    availableModels: VISION_MODELS,
    isNavigating: state.isNavigating,
  };
}
