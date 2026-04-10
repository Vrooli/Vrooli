/**
 * useToolMutations - React Query mutations for tool CRUD operations.
 *
 * Extracted from useTools.ts. Handles toggle, approval, reset, and sync mutations
 * with optimistic updates and rollback.
 */

import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  setToolEnabled,
  setToolApproval,
  resetToolConfig,
  syncTools,
  type ToolSet,
  type ApprovalOverride,
} from "../lib/api";
import { toolQueryKeys } from "./useTools";

export interface UseToolMutationsOptions {
  chatId?: string;
}

export function useToolMutations({ chatId }: UseToolMutationsOptions = {}) {
  const queryClient = useQueryClient();

  // Toggle tool enabled state
  const toggleMutationKey = ["tools", "toggle", chatId ?? "global"] as const;
  const toggleMutation = useMutation({
    mutationKey: toggleMutationKey,
    mutationFn: async ({
      scenario,
      toolName,
      enabled: newEnabled,
    }: {
      scenario: string;
      toolName: string;
      enabled: boolean;
    }) => {
      await setToolEnabled({
        chat_id: chatId,
        scenario,
        tool_name: toolName,
        enabled: newEnabled,
      });
    },
    onMutate: async ({ scenario, toolName, enabled: newEnabled }) => {
      await queryClient.cancelQueries({ queryKey: toolQueryKeys.toolSet(chatId) });
      const previousToolSet = queryClient.getQueryData<ToolSet>(toolQueryKeys.toolSet(chatId));

      if (previousToolSet) {
        queryClient.setQueryData<ToolSet>(toolQueryKeys.toolSet(chatId), {
          ...previousToolSet,
          tools: previousToolSet.tools.map((t) =>
            t.scenario === scenario && t.tool.name === toolName
              ? { ...t, enabled: newEnabled, source: chatId ? "chat" : "global" }
              : t
          ),
        });
      }

      return { previousToolSet };
    },
    onError: (_err, _variables, context) => {
      if (context?.previousToolSet) {
        queryClient.setQueryData(toolQueryKeys.toolSet(chatId), context.previousToolSet);
      }
    },
    onSettled: () => {
      setTimeout(() => {
        const pendingCount = queryClient.isMutating({ mutationKey: toggleMutationKey });
        if (pendingCount === 0) {
          queryClient.invalidateQueries({ queryKey: toolQueryKeys.toolSet(chatId) });
        }
      }, 0);
    },
  });

  // Set tool approval override
  const approvalMutation = useMutation({
    mutationFn: async ({
      scenario,
      toolName,
      override,
    }: {
      scenario: string;
      toolName: string;
      override: ApprovalOverride;
    }) => {
      await setToolApproval(scenario, toolName, override, chatId);
    },
    onMutate: async ({ scenario, toolName, override }) => {
      await queryClient.cancelQueries({ queryKey: toolQueryKeys.toolSet(chatId) });
      const previousToolSet = queryClient.getQueryData<ToolSet>(toolQueryKeys.toolSet(chatId));

      if (previousToolSet) {
        queryClient.setQueryData<ToolSet>(toolQueryKeys.toolSet(chatId), {
          ...previousToolSet,
          tools: previousToolSet.tools.map((t) => {
            if (t.scenario === scenario && t.tool.name === toolName) {
              let requiresApproval = t.tool.metadata.requires_approval;
              if (override === "require") requiresApproval = true;
              else if (override === "skip") requiresApproval = false;
              return {
                ...t,
                requires_approval: requiresApproval,
                approval_override: override,
                approval_source: chatId ? "chat" : "global",
              };
            }
            return t;
          }),
        });
      }

      return { previousToolSet };
    },
    onError: (_err, _variables, context) => {
      if (context?.previousToolSet) {
        queryClient.setQueryData(toolQueryKeys.toolSet(chatId), context.previousToolSet);
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: toolQueryKeys.toolSet(chatId) });
    },
  });

  // Reset tool to default
  const resetMutation = useMutation({
    mutationFn: async ({ scenario, toolName }: { scenario: string; toolName: string }) => {
      await resetToolConfig(scenario, toolName, chatId);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: toolQueryKeys.toolSet(chatId) });
    },
  });

  // Sync tools (discover from all running scenarios)
  const syncMutation = useMutation({
    mutationFn: syncTools,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: toolQueryKeys.all });
    },
  });

  return {
    toggleMutation,
    approvalMutation,
    resetMutation,
    syncMutation,
  };
}
