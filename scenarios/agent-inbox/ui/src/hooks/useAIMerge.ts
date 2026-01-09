/**
 * Hook for AI-powered message merging.
 * Combines existing message content with a template using AI.
 *
 * NOTE: This currently uses a simple text combination approach.
 * Full AI merging would require a dedicated backend endpoint for
 * ephemeral completions that don't persist to chat history.
 */

import { useCallback, useState } from "react";

export interface UseAIMergeReturn {
  mergeMessages: (
    existingMessage: string,
    templateContent: string,
    model: string,
    chatId: string
  ) => Promise<string>;
  isMerging: boolean;
  error: Error | null;
  clearError: () => void;
}

export function useAIMerge(): UseAIMergeReturn {
  const [isMerging, setIsMerging] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const clearError = useCallback(() => {
    setError(null);
  }, []);

  const mergeMessages = useCallback(
    async (
      existingMessage: string,
      templateContent: string,
      _model: string,
      _chatId: string
    ): Promise<string> => {
      setIsMerging(true);
      setError(null);

      try {
        // TODO: Implement proper AI merging via backend endpoint
        // For now, we do a simple combination:
        // - If template has variables, prepend the user's message as context
        // - Otherwise just append the draft to the template

        // Simulate a brief delay for UX consistency
        await new Promise((resolve) => setTimeout(resolve, 500));

        // Simple merge: prepend user's message as context
        const merged = `${existingMessage.trim()}\n\n---\n\n${templateContent.trim()}`;

        return merged;
      } catch (err) {
        const error =
          err instanceof Error ? err : new Error("Failed to merge messages");
        setError(error);
        throw error;
      } finally {
        setIsMerging(false);
      }
    },
    []
  );

  return {
    mergeMessages,
    isMerging,
    error,
    clearError,
  };
}
