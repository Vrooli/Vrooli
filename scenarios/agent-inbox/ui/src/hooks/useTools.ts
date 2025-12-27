/**
 * useTools Hook
 *
 * Provides tool configuration state management with react-query.
 * Supports both global tool configuration and per-chat overrides.
 *
 * ARCHITECTURE:
 * - Uses react-query for caching and state management
 * - Separate queries for global vs chat-specific tool sets
 * - Optimistic updates for responsive UI
 * - Automatic cache invalidation on mutations
 *
 * TESTING SEAMS:
 * - All API calls go through the api.ts module
 * - Query client can be mocked in tests
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchToolSet,
  fetchScenarioStatuses,
  setToolEnabled,
  setToolApproval,
  resetToolConfig,
  refreshTools,
  type ToolSet,
  type ScenarioStatus,
  type EffectiveTool,
  type ApprovalOverride,
} from "../lib/api";

// Query keys for cache management
export const toolQueryKeys = {
  all: ["tools"] as const,
  toolSet: (chatId?: string) => [...toolQueryKeys.all, "set", chatId ?? "global"] as const,
  scenarios: () => [...toolQueryKeys.all, "scenarios"] as const,
};

export interface UseToolsOptions {
  /** Chat ID for chat-specific tool configurations */
  chatId?: string;
  /** Whether to fetch tools on mount (default: true) */
  enabled?: boolean;
}

export interface UseToolsReturn {
  // Data
  toolSet: ToolSet | undefined;
  scenarios: ScenarioStatus[] | undefined;
  enabledTools: EffectiveTool[];
  toolsByScenario: Map<string, EffectiveTool[]>;
  toolsByCategory: Map<string, EffectiveTool[]>;

  // Loading states
  isLoading: boolean;
  isLoadingScenarios: boolean;
  isRefreshing: boolean;
  isUpdating: boolean;

  // Error states
  error: Error | null;
  scenariosError: Error | null;

  // Actions
  toggleTool: (scenario: string, toolName: string, enabled: boolean) => Promise<void>;
  setApproval: (scenario: string, toolName: string, override: ApprovalOverride) => Promise<void>;
  resetTool: (scenario: string, toolName: string) => Promise<void>;
  refreshToolRegistry: () => Promise<void>;
  refetch: () => void;
}

/**
 * Hook for managing tool configurations.
 *
 * @example
 * // Global tools (Settings modal)
 * const { toolSet, toggleTool } = useTools();
 *
 * @example
 * // Chat-specific tools
 * const { toolSet, toggleTool } = useTools({ chatId: "abc-123" });
 */
export function useTools(options: UseToolsOptions = {}): UseToolsReturn {
  const { chatId, enabled = true } = options;
  const queryClient = useQueryClient();

  // Fetch tool set (global or chat-specific)
  const {
    data: toolSet,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: toolQueryKeys.toolSet(chatId),
    queryFn: () => fetchToolSet(chatId),
    enabled,
    staleTime: 60_000, // 1 minute
  });

  // Fetch scenario statuses
  const {
    data: scenarios,
    isLoading: isLoadingScenarios,
    error: scenariosError,
  } = useQuery({
    queryKey: toolQueryKeys.scenarios(),
    queryFn: fetchScenarioStatuses,
    enabled,
    staleTime: 30_000, // 30 seconds
  });

  // Toggle tool enabled state
  const toggleMutation = useMutation({
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
      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: toolQueryKeys.toolSet(chatId) });

      // Snapshot previous value
      const previousToolSet = queryClient.getQueryData<ToolSet>(toolQueryKeys.toolSet(chatId));

      // Optimistically update
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
      // Rollback on error
      if (context?.previousToolSet) {
        queryClient.setQueryData(toolQueryKeys.toolSet(chatId), context.previousToolSet);
      }
    },
    onSettled: () => {
      // Refetch to ensure consistency
      queryClient.invalidateQueries({ queryKey: toolQueryKeys.toolSet(chatId) });
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
      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: toolQueryKeys.toolSet(chatId) });

      // Snapshot previous value
      const previousToolSet = queryClient.getQueryData<ToolSet>(toolQueryKeys.toolSet(chatId));

      // Optimistically update
      if (previousToolSet) {
        queryClient.setQueryData<ToolSet>(toolQueryKeys.toolSet(chatId), {
          ...previousToolSet,
          tools: previousToolSet.tools.map((t) => {
            if (t.scenario === scenario && t.tool.name === toolName) {
              // Compute effective requires_approval based on override
              let requiresApproval = t.tool.metadata.requires_approval;
              if (override === "require") {
                requiresApproval = true;
              } else if (override === "skip") {
                requiresApproval = false;
              }
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
      // Rollback on error
      if (context?.previousToolSet) {
        queryClient.setQueryData(toolQueryKeys.toolSet(chatId), context.previousToolSet);
      }
    },
    onSettled: () => {
      // Refetch to ensure consistency
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

  // Refresh tool registry
  const refreshMutation = useMutation({
    mutationFn: refreshTools,
    onSuccess: () => {
      // Invalidate all tool queries
      queryClient.invalidateQueries({ queryKey: toolQueryKeys.all });
    },
  });

  // Derived data: enabled tools only
  const enabledTools = toolSet?.tools.filter((t) => t.enabled) ?? [];

  // Derived data: tools grouped by scenario
  const toolsByScenario = new Map<string, EffectiveTool[]>();
  for (const tool of toolSet?.tools ?? []) {
    const existing = toolsByScenario.get(tool.scenario) ?? [];
    toolsByScenario.set(tool.scenario, [...existing, tool]);
  }

  // Derived data: tools grouped by category
  const toolsByCategory = new Map<string, EffectiveTool[]>();
  for (const tool of toolSet?.tools ?? []) {
    const category = tool.tool.category ?? "uncategorized";
    const existing = toolsByCategory.get(category) ?? [];
    toolsByCategory.set(category, [...existing, tool]);
  }

  return {
    // Data
    toolSet,
    scenarios,
    enabledTools,
    toolsByScenario,
    toolsByCategory,

    // Loading states
    isLoading,
    isLoadingScenarios,
    isRefreshing: refreshMutation.isPending,
    isUpdating: toggleMutation.isPending || approvalMutation.isPending || resetMutation.isPending,

    // Error states
    error: error as Error | null,
    scenariosError: scenariosError as Error | null,

    // Actions
    toggleTool: async (scenario, toolName, enabled) => {
      await toggleMutation.mutateAsync({ scenario, toolName, enabled });
    },
    setApproval: async (scenario, toolName, override) => {
      await approvalMutation.mutateAsync({ scenario, toolName, override });
    },
    resetTool: async (scenario, toolName) => {
      await resetMutation.mutateAsync({ scenario, toolName });
    },
    refreshToolRegistry: async () => {
      await refreshMutation.mutateAsync();
    },
    refetch: () => {
      refetch();
    },
  };
}

/**
 * Get the count of enabled tools for display in UI badges.
 */
export function useEnabledToolCount(chatId?: string): number {
  const { enabledTools } = useTools({ chatId });
  return enabledTools.length;
}
