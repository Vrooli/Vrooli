/**
 * useAIConversation Hook
 *
 * Wraps useAINavigation to provide a chat-based conversation interface.
 * Manages message history and maps navigation events to chat messages.
 *
 * Key features:
 * - Maintains conversation history
 * - Creates user messages for prompts
 * - Creates/updates assistant messages for navigation sessions
 * - Supports abort and human intervention flows
 */

import { useState, useCallback, useEffect, useRef } from 'react';
import { useAINavigation, AINavigationError } from '../ai-navigation/useAINavigation';
import { useEntitlementStore } from '@stores/entitlementStore';
import { useAICapabilityStore } from '@stores/aiCapabilityStore';
import type { AINavigationState } from '../ai-navigation/types';
import type { AIMessage, AISettings, EntitlementErrorCode } from './types';
import { createUserMessage, createAssistantMessage, createSystemMessage, createEntitlementErrorMessage } from './types';

const entitlementErrorCodes = new Set<EntitlementErrorCode>([
  'AI_NOT_AVAILABLE',
  'INSUFFICIENT_CREDITS',
]);

const isEntitlementNavigationError = (
  error: unknown,
): error is AINavigationError & { code: EntitlementErrorCode } => (
  error instanceof AINavigationError
  && entitlementErrorCodes.has(error.code as EntitlementErrorCode)
);

const createNavigationStartErrorMessage = (error: unknown): AIMessage => {
  if (isEntitlementNavigationError(error)) {
    const entitlementStatus = useEntitlementStore.getState().status;
    const aiCapability = useAICapabilityStore.getState().capability;
    return createEntitlementErrorMessage(error.code, {
      remaining: parseInt(error.details?.remaining ?? '0', 10),
      creditsUsed: entitlementStatus?.ai_credits_used,
      creditsLimit: entitlementStatus?.ai_credits_limit,
      resetDate: aiCapability.resetDate ?? entitlementStatus?.ai_reset_date,
      tier: entitlementStatus?.tier,
    });
  }

  const errorMessage = error instanceof Error ? error.message : 'Failed to start navigation';
  return createSystemMessage(`Error: ${errorMessage}`);
};

type NavigationMessageState = Pick<AINavigationState, 'humanIntervention' | 'status' | 'error'>;

const getNavigationMessagePatch = (
  navigationState: NavigationMessageState,
): Partial<AIMessage> | null => {
  const intervention = navigationState.humanIntervention
    ? { humanIntervention: navigationState.humanIntervention }
    : {};
  if (navigationState.error) {
    if (navigationState.status === 'awaiting_human') {
      return { ...intervention, status: 'awaiting_human', error: navigationState.error };
    }
    if (navigationState.status === 'observation_unavailable') {
      return { ...intervention, status: 'observation_unavailable', error: navigationState.error, canAbort: true };
    }
    return { ...intervention, status: 'failed', error: navigationState.error, canAbort: false };
  }
  if (navigationState.status === 'aborting') return { status: 'aborting', canAbort: false };
  return navigationState.humanIntervention
    ? { ...intervention, status: 'awaiting_human' }
    : null;
};

// ============================================================================
// Types
// ============================================================================

export interface UseAIConversationOptions {
  /** Browser session ID for navigation */
  sessionId: string | null;
  /** AI settings (model, maxSteps) */
  settings: AISettings;
  /** Callback when a new timeline action should be added */
  onTimelineAction?: () => void;
}

export interface UseAIConversationReturn {
  /** All messages in the conversation */
  messages: AIMessage[];
  /** Send a new message (starts navigation) */
  sendMessage: (prompt: string) => Promise<void>;
  /** Abort the current navigation */
  abortNavigation: () => Promise<void>;
  /** Resume navigation after human intervention */
  resumeNavigation: () => Promise<boolean>;
  /** Clear all messages */
  clearConversation: () => void;
  /** Whether navigation is currently in progress */
  isNavigating: boolean;
  /** Current navigation ID (if any) */
  currentNavigationId: string | null;
  /** Whether awaiting human intervention */
  isAwaitingHuman: boolean;
  /** Add a system message to the conversation */
  addSystemMessage: (content: string) => void;
  /** Navigation steps from the current/last navigation (for timeline merging) */
  navigationSteps: import('../ai-navigation/types').AINavigationStep[];
  /** Available AI models */
  availableModels: import('../ai-navigation/types').VisionModelSpec[];
  /** Human intervention state */
  humanIntervention: import('../ai-navigation/types').HumanInterventionState | null;
}

// ============================================================================
// Hook Implementation
// ============================================================================

export function useAIConversation({
  sessionId,
  settings,
  onTimelineAction,
}: UseAIConversationOptions): UseAIConversationReturn {
  const [messages, setMessages] = useState<AIMessage[]>([]);
  const currentAssistantIdRef = useRef<string | null>(null);
  const currentNavigationIdRef = useRef<string | null>(null);
  const conversationGenerationRef = useRef(0);
  const pendingStartGenerationRef = useRef<number | null>(null);
  const admittedNavigationIdsRef = useRef(new Set<string>());
  const sendInFlightRef = useRef(false);
  const latestStartedAttemptRef = useRef(0);

  // Use the underlying navigation hook
  const {
    state: navState,
    startNavigation,
    abortNavigation: navAbort,
    resumeNavigation: navResume,
    reset: navReset,
    isNavigating,
    isAwaitingHuman,
    availableModels,
  } = useAINavigation({
    sessionId,
    onStarted: (navigationId, startAttempt) => {
      if (
        pendingStartGenerationRef.current !== conversationGenerationRef.current
        || (startAttempt !== undefined && startAttempt < latestStartedAttemptRef.current)
        || admittedNavigationIdsRef.current.has(navigationId)
      ) {
        return;
      }
      if (startAttempt !== undefined) latestStartedAttemptRef.current = startAttempt;
      admittedNavigationIdsRef.current.add(navigationId);
      const assistantMessage = createAssistantMessage(navigationId);
      currentAssistantIdRef.current = assistantMessage.id;
      currentNavigationIdRef.current = navigationId;
      setMessages((prev) => [...prev, assistantMessage]);
    },
    onStep: (step, navigationId) => {
      // Update the current assistant message with new step
      const assistantId = currentAssistantIdRef.current;
      if (assistantId && navigationId && currentNavigationIdRef.current === navigationId) {
        setMessages((prev) =>
          prev.map((msg) => {
            if (msg.id !== assistantId) return msg;
            // Preserve 'aborting' status - don't overwrite with 'running'
            const newStatus = msg.status === 'aborting' ? 'aborting' : 'running';
            return {
              ...msg,
              status: newStatus,
              steps: [...(msg.steps || []), step],
              totalTokens: (msg.totalTokens || 0) + step.tokensUsed.totalTokens,
            };
          })
        );
        onTimelineAction?.();
      }
    },
    onComplete: (status, summary, navigationId) => {
      const assistantId = currentAssistantIdRef.current;

      // Update the current assistant message with final status
      if (assistantId && navigationId && currentNavigationIdRef.current === navigationId) {
        setMessages((prev) => {
          return prev.map((msg) => {
            if (msg.id !== assistantId) return msg;

            const finalStatus =
              status === 'completed'
                ? 'completed'
                : status === 'aborted'
                  ? 'aborted'
                  : 'failed';

            return {
              ...msg,
              status: finalStatus,
              content: summary || msg.content,
              canAbort: false,
            };
          });
        });
        currentAssistantIdRef.current = null;
        currentNavigationIdRef.current = null;
      }
    },
  });

  const resetConversationState = useCallback((resetNavigation: boolean) => {
    conversationGenerationRef.current += 1;
    pendingStartGenerationRef.current = null;
    admittedNavigationIdsRef.current.clear();
    sendInFlightRef.current = false;
    if (resetNavigation) navReset();
    setMessages([]);
    currentAssistantIdRef.current = null;
    currentNavigationIdRef.current = null;
  }, [navReset]);

  const updateCurrentAssistantMessage = useCallback((
    navigationId: string | null,
    update: (message: AIMessage) => AIMessage,
    expectedAssistantId: string | null = currentAssistantIdRef.current,
  ) => {
    if (
      !expectedAssistantId
      || expectedAssistantId !== currentAssistantIdRef.current
      || navigationId !== currentNavigationIdRef.current
    ) return;
    setMessages((prev) => prev.map((msg) => (msg.id === expectedAssistantId ? update(msg) : msg)));
  }, []);

  // Project navigation lifecycle state once per update. Keeping the precedence
  // here avoids three effects racing separate message projections.
  const {
    error: navigationError,
    humanIntervention,
    navigationId,
    status: navigationStatus,
  } = navState;
  useEffect(() => {
    if (!humanIntervention && navigationStatus !== 'aborting' && !navigationError) return;
    const messagePatch = getNavigationMessagePatch({
      error: navigationError,
      humanIntervention,
      status: navigationStatus,
    });
    if (!messagePatch) return;
    updateCurrentAssistantMessage(navigationId, (msg) => ({ ...msg, ...messagePatch }));
  }, [humanIntervention, navigationError, navigationId, navigationStatus, updateCurrentAssistantMessage]);

  // Reset conversation when session changes
  useEffect(() => {
    resetConversationState(false);
  }, [resetConversationState, sessionId]);

  const sendMessage = useCallback(
    async (prompt: string) => {
      if (!prompt.trim()) return;
      if (isNavigating || sendInFlightRef.current) return;
      sendInFlightRef.current = true;

      const conversationGeneration = conversationGenerationRef.current + 1;
      conversationGenerationRef.current = conversationGeneration;
      pendingStartGenerationRef.current = conversationGeneration;

      // Create user message
      const userMessage = createUserMessage(prompt);
      setMessages((prev) => [...prev, userMessage]);

      // Start navigation and create assistant message placeholder
      try {
        const navigationId = await startNavigation(prompt, settings.model, settings.maxSteps);

        if (!navigationId || conversationGenerationRef.current !== conversationGeneration) {
          // startNavigation failed but didn't throw - error already set in state
          return;
        }

      } catch (err) {
        if (conversationGenerationRef.current !== conversationGeneration) return;
        setMessages((prev) => [...prev, createNavigationStartErrorMessage(err)]);
      } finally {
        if (conversationGenerationRef.current === conversationGeneration) {
          sendInFlightRef.current = false;
        }
      }
    },
    [isNavigating, settings.model, settings.maxSteps, startNavigation]
  );

  const abortNavigation = useCallback(async () => {
    await navAbort();
  }, [navAbort]);

  const resumeNavigation = useCallback(async () => {
    const assistantId = currentAssistantIdRef.current;
    const navigationId = currentNavigationIdRef.current;
    const resumed = await navResume();
    if (!resumed) return false;
    // Update message status back to running
    updateCurrentAssistantMessage(navigationId, (msg) => ({
      ...msg,
      status: 'running',
      humanIntervention: undefined,
    }), assistantId);
    return true;
  }, [navResume, updateCurrentAssistantMessage]);

  const clearConversation = useCallback(() => {
    resetConversationState(true);
  }, [resetConversationState]);

  const addSystemMessage = useCallback((content: string) => {
    const message = createSystemMessage(content);
    setMessages((prev) => [...prev, message]);
  }, []);

  return {
    messages,
    sendMessage,
    abortNavigation,
    resumeNavigation,
    clearConversation,
    isNavigating,
    currentNavigationId: navState.navigationId,
    isAwaitingHuman,
    addSystemMessage,
    navigationSteps: navState.steps,
    availableModels,
    humanIntervention: navState.humanIntervention,
  };
}
