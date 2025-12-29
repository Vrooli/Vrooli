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
import { useAINavigation } from '../ai-navigation/useAINavigation';
import type { AIMessage, AISettings } from './types';
import { createUserMessage, createAssistantMessage, createSystemMessage } from './types';

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
            return {
              ...msg,
              status: 'running',
              steps: [...(msg.steps || []), step],
              totalTokens: (msg.totalTokens || 0) + step.tokensUsed.totalTokens,
            };
          })
        );
        onTimelineAction?.();
      }
    },
    onComplete: (status, summary) => {
      // Update the current assistant message with final status
      if (currentAssistantIdRef.current) {
        setMessages((prev) =>
          prev.map((msg) => {
            if (msg.id !== currentAssistantIdRef.current) return msg;

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
          })
        );
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
        await startNavigation(prompt, settings.model, settings.maxSteps);

        // Create assistant message after navigation starts successfully
        // Use the navigation ID from navState (it will be set after startNavigation)
        const assistantMessage = createAssistantMessage(navState.navigationId || `nav-${Date.now()}`);
        currentAssistantIdRef.current = assistantMessage.id;
        setMessages((prev) => [...prev, assistantMessage]);
      } catch (err) {
        // Add error as system message
        const errorMessage =
          err instanceof Error ? err.message : 'Failed to start navigation';
        const systemMessage = createSystemMessage(`Error: ${errorMessage}`);
        setMessages((prev) => [...prev, systemMessage]);
      }
    },
    [isNavigating, settings.model, settings.maxSteps, startNavigation, navState.navigationId]
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
