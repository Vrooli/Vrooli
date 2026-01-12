/**
 * useActiveTemplate Hook
 *
 * Manages the "active template" state at the chat level.
 * An active template has its suggested tools auto-enabled and remains
 * active until the user deactivates it OR the agent uses one of its tools.
 *
 * ARCHITECTURE:
 * - State is persisted in the chat record (active_template_id, active_template_tool_ids)
 * - Uses react-query for synchronization with backend
 * - SSE events signal when to deactivate (tool_call_result with deactivate_template: true)
 *
 * USAGE:
 * - Call activate() when user selects a template with suggested tools
 * - Call deactivate() when user manually clears OR SSE signals tool usage
 * - Check isActive to show UI indicators
 */

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { setActiveTemplate, type Chat } from "../lib/api";

export interface UseActiveTemplateReturn {
  /** ID of the currently active template, or null if none */
  activeTemplateId: string | null;
  /** Tool IDs associated with the active template */
  activeToolIds: string[];
  /** Whether a template is currently active */
  isActive: boolean;
  /** Activate a template with its suggested tools */
  activate: (templateId: string, toolIds: string[]) => Promise<void>;
  /** Deactivate the current template (clears active state) */
  deactivate: () => Promise<void>;
  /** Whether an activation/deactivation is in progress */
  isUpdating: boolean;
}

/**
 * Hook for managing active template state for a chat.
 *
 * @param chatId - The chat ID to manage active template for
 * @param chat - Optional chat object (avoids extra fetch if already available)
 * @returns Active template state and mutation functions
 *
 * @example
 * const activeTemplate = useActiveTemplate(chatId, chatData?.chat);
 *
 * // Activate when user selects a template
 * const handleTemplateSelect = (template) => {
 *   if (template.suggestedToolIds?.length) {
 *     await tools.enableToolsByIds(template.suggestedToolIds);
 *     await activeTemplate.activate(template.id, template.suggestedToolIds);
 *   }
 * };
 *
 * // Deactivate when SSE signals tool usage
 * if (event.deactivate_template) {
 *   activeTemplate.deactivate();
 * }
 */
export function useActiveTemplate(
  chatId: string | undefined,
  chat?: Chat
): UseActiveTemplateReturn {
  const queryClient = useQueryClient();

  // Extract active template state from chat
  const activeTemplateId = chat?.active_template_id ?? null;
  const activeToolIds = chat?.active_template_tool_ids ?? [];
  const isActive = Boolean(activeTemplateId);

  // Mutation to activate a template
  const activateMutation = useMutation({
    mutationFn: async ({ templateId, toolIds }: { templateId: string; toolIds: string[] }) => {
      if (!chatId) throw new Error("Chat ID required to activate template");
      return setActiveTemplate(chatId, templateId, toolIds);
    },
    onMutate: async ({ templateId, toolIds }) => {
      // Optimistically update the chat cache
      const queryKey = ["chat", chatId];
      await queryClient.cancelQueries({ queryKey });

      const previousData = queryClient.getQueryData(queryKey);

      queryClient.setQueryData(queryKey, (old: { chat: Chat; messages: unknown[] } | undefined) => {
        if (!old) return old;
        return {
          ...old,
          chat: {
            ...old.chat,
            active_template_id: templateId,
            active_template_tool_ids: toolIds,
          },
        };
      });

      return { previousData };
    },
    onError: (_err, _variables, context) => {
      // Rollback on error
      if (context?.previousData) {
        queryClient.setQueryData(["chat", chatId], context.previousData);
      }
    },
    onSettled: () => {
      // Refetch to ensure consistency
      queryClient.invalidateQueries({ queryKey: ["chat", chatId] });
    },
  });

  // Mutation to deactivate the active template
  const deactivateMutation = useMutation({
    mutationFn: async () => {
      if (!chatId) throw new Error("Chat ID required to deactivate template");
      return setActiveTemplate(chatId, null, []);
    },
    onMutate: async () => {
      // Optimistically update the chat cache
      const queryKey = ["chat", chatId];
      await queryClient.cancelQueries({ queryKey });

      const previousData = queryClient.getQueryData(queryKey);

      queryClient.setQueryData(queryKey, (old: { chat: Chat; messages: unknown[] } | undefined) => {
        if (!old) return old;
        return {
          ...old,
          chat: {
            ...old.chat,
            active_template_id: undefined,
            active_template_tool_ids: undefined,
          },
        };
      });

      return { previousData };
    },
    onError: (_err, _variables, context) => {
      // Rollback on error
      if (context?.previousData) {
        queryClient.setQueryData(["chat", chatId], context.previousData);
      }
    },
    onSettled: () => {
      // Refetch to ensure consistency
      queryClient.invalidateQueries({ queryKey: ["chat", chatId] });
    },
  });

  return {
    activeTemplateId,
    activeToolIds,
    isActive,
    activate: async (templateId: string, toolIds: string[]) => {
      await activateMutation.mutateAsync({ templateId, toolIds });
    },
    deactivate: async () => {
      await deactivateMutation.mutateAsync();
    },
    isUpdating: activateMutation.isPending || deactivateMutation.isPending,
  };
}
