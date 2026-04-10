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
import type { AIMessage, AISettings } from './types';
import { createUserMessage, createAssistantMessage, createSystemMessage, createEntitlementErrorMessage } from './types';

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
  resumeNavigation: () => Promise<void>;
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
    onStep: (step) => {
      // Update the current assistant message with new step
      if (currentAssistantIdRef.current) {
        setMessages((prev) =>
          prev.map((msg) => {
            if (msg.id !== currentAssistantIdRef.current) return msg;
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
    onComplete: (status, summary) => {
      console.log('[useAIConversation] onComplete called:', {
        status,
        summary,
        currentAssistantId: currentAssistantIdRef.current,
      });

      // Update the current assistant message with final status
      if (currentAssistantIdRef.current) {
        setMessages((prev) => {
          console.log('[useAIConversation] Messages in state:', prev.map(m => ({ id: m.id, role: m.role, status: m.status })));
          return prev.map((msg) => {
            if (msg.id !== currentAssistantIdRef.current) return msg;

            const finalStatus =
              status === 'completed'
                ? 'completed'
                : status === 'aborted'
                  ? 'aborted'
                  : 'failed';

            console.log('[useAIConversation] Setting message status to:', finalStatus);

            return {
              ...msg,
              status: finalStatus,
              content: summary || msg.content,
              canAbort: false,
            };
          });
        });
        currentAssistantIdRef.current = null;
      }
    },
  });

  // Sync human intervention state to current message
  useEffect(() => {
    if (navState.humanIntervention && currentAssistantIdRef.current) {
      setMessages((prev) =>
        prev.map((msg) => {
          if (msg.id !== currentAssistantIdRef.current) return msg;
          return {
            ...msg,
            status: 'awaiting_human' as const,
            humanIntervention: navState.humanIntervention ?? undefined,
          };
        })
      );
    }
  }, [navState.humanIntervention]);

  // Sync aborting state to current message
  useEffect(() => {
    if (navState.status === 'aborting' && currentAssistantIdRef.current) {
      setMessages((prev) =>
        prev.map((msg) => {
          if (msg.id !== currentAssistantIdRef.current) return msg;
          return {
            ...msg,
            status: 'aborting' as const,
            canAbort: false, // Can't abort twice
          };
        })
      );
    }
  }, [navState.status]);

  // Sync error state to current message
  useEffect(() => {
    if (navState.error && currentAssistantIdRef.current) {
      setMessages((prev) =>
        prev.map((msg) => {
          if (msg.id !== currentAssistantIdRef.current) return msg;
          return {
            ...msg,
            status: 'failed',
            error: navState.error ?? undefined,
            canAbort: false,
          };
        })
      );
    }
  }, [navState.error]);

  // Reset conversation when session changes
  useEffect(() => {
    setMessages([]);
    currentAssistantIdRef.current = null;
  }, [sessionId]);

  const sendMessage = useCallback(
    async (prompt: string) => {
      if (!prompt.trim()) return;
      if (isNavigating) return;

      // Create user message
      const userMessage = createUserMessage(prompt);
      setMessages((prev) => [...prev, userMessage]);

      // Start navigation and create assistant message placeholder
      try {
        const navigationId = await startNavigation(prompt, settings.model, settings.maxSteps);

        if (!navigationId) {
          // startNavigation failed but didn't throw - error already set in state
          return;
        }

        // Create assistant message with the actual navigation ID
        const assistantMessage = createAssistantMessage(navigationId);
        currentAssistantIdRef.current = assistantMessage.id;
        setMessages((prev) => [...prev, assistantMessage]);
      } catch (err) {
        // Check if this is an entitlement-related error
        if (err instanceof AINavigationError) {
          const errorCode = err.code;

          if (errorCode === 'AI_NOT_AVAILABLE' || errorCode === 'INSUFFICIENT_CREDITS') {
            // Get current entitlement info for richer error display
            const entitlementStatus = useEntitlementStore.getState().status;
            const aiCapability = useAICapabilityStore.getState().capability;

            const errorMessage = createEntitlementErrorMessage(errorCode, {
              remaining: err.details?.remaining ? parseInt(err.details.remaining, 10) : 0,
              creditsUsed: entitlementStatus?.ai_credits_used,
              creditsLimit: entitlementStatus?.ai_credits_limit,
              resetDate: aiCapability.resetDate || entitlementStatus?.ai_reset_date,
              tier: entitlementStatus?.tier,
            });
            setMessages((prev) => [...prev, errorMessage]);
            return;
          }
        }

        // Add generic error as system message
        const errorMessage =
          err instanceof Error ? err.message : 'Failed to start navigation';
        const systemMessage = createSystemMessage(`Error: ${errorMessage}`);
        setMessages((prev) => [...prev, systemMessage]);
      }
    },
    [isNavigating, settings.model, settings.maxSteps, startNavigation]
  );

  const abortNavigation = useCallback(async () => {
    await navAbort();
  }, [navAbort]);

  const resumeNavigation = useCallback(async () => {
    await navResume();
    // Update message status back to running
    if (currentAssistantIdRef.current) {
      setMessages((prev) =>
        prev.map((msg) => {
          if (msg.id !== currentAssistantIdRef.current) return msg;
          return {
            ...msg,
            status: 'running',
            humanIntervention: undefined,
          };
        })
      );
    }
  }, [navResume]);

  const clearConversation = useCallback(() => {
    navReset();
    setMessages([]);
    currentAssistantIdRef.current = null;
  }, [navReset]);

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
