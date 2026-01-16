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

import { useCallback, useMemo } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchToolSet,
  fetchScenarioStatuses,
  setToolEnabled,
  setToolApproval,
  resetToolConfig,
  syncTools,
  type ToolSet,
  type ScenarioStatus,
  type EffectiveTool,
  type ApprovalOverride,
  type DiscoveryResult,
} from "../lib/api";

// Query keys for cache management
export const toolQueryKeys = {
  all: ["tools"] as const,
  toolSet: (chatId?: string) => [...toolQueryKeys.all, "set", chatId ?? "global"] as const,
  scenarios: () => [...toolQueryKeys.all, "scenarios"] as const,
};

// Stable empty arrays/maps to prevent creating new references on every render.
// CRITICAL: Using inline [] or new Map() creates new references that can trigger
// infinite re-render loops when used in dependencies or passed to memoized components.
const EMPTY_TOOLS: EffectiveTool[] = [];
const EMPTY_SCENARIO_MAP: Map<string, EffectiveTool[]> = new Map();
const EMPTY_CATEGORY_MAP: Map<string, EffectiveTool[]> = new Map();

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
  isSyncing: boolean;
  isUpdating: boolean;

  // Error states
  error: Error | null;
  scenariosError: Error | null;

  // Actions
  toggleTool: (scenario: string, toolName: string, enabled: boolean) => Promise<void>;
  setApproval: (scenario: string, toolName: string, override: ApprovalOverride) => Promise<void>;
  resetTool: (scenario: string, toolName: string) => Promise<void>;
  /** Discover tools from all running scenarios via vrooli CLI */
  syncDiscoveredTools: () => Promise<DiscoveryResult>;
  refetch: () => void;
  /** Enable multiple tools by their IDs (format: "scenario:tool_name") */
  enableToolsByIds: (toolIds: string[]) => Promise<void>;
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
// DEBUG: Track renders
let useToolsRenderCount = 0;

export function useTools(options: UseToolsOptions = {}): UseToolsReturn {
  const { chatId, enabled = true } = options;
  const queryClient = useQueryClient();

  // DEBUG: Track renders and identify call sources
  useToolsRenderCount++;
  const stackTrace = chatId === undefined ? new Error().stack?.split('\n').slice(1, 4).join(' <- ') : undefined;
  console.log(`[useTools] Render #${useToolsRenderCount}`, {
    chatId,
    chatIdIsUndefined: chatId === undefined,
    enabled,
    // Only log stack trace for undefined chatId calls to help identify source
    ...(chatId === undefined && { callStack: stackTrace })
  });

  // Fetch tool set (global or chat-specific)
  // CRITICAL: Use aggressive caching to prevent cascading re-renders.
  // Multiple components use useTools with different chatId values. Without
  // these options, each query refetch triggers re-renders in ALL useTools
  // instances, creating exponential render storms during rapid state transitions.
  const {
    data: toolSet,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: toolQueryKeys.toolSet(chatId),
    queryFn: () => fetchToolSet(chatId),
    enabled,
    staleTime: 60_000, // 1 minute - data considered fresh
    gcTime: 300_000, // 5 minutes - keep in cache
    refetchOnMount: false, // Don't refetch when component mounts
    refetchOnWindowFocus: false, // Don't refetch on window focus
    refetchOnReconnect: false, // Don't refetch on reconnect
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
    staleTime: 30_000, // 30 seconds - data considered fresh
    gcTime: 300_000, // 5 minutes - keep in cache
    refetchOnMount: false, // Don't refetch when component mounts
    refetchOnWindowFocus: false, // Don't refetch on window focus
    refetchOnReconnect: false, // Don't refetch on reconnect
  });

  // Toggle tool enabled state
  // CRITICAL: Uses a mutation key so we can track pending mutations and avoid
  // race conditions during batch operations (like scenario-level toggles)
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
      // Cancel outgoing refetches to prevent them from overwriting optimistic updates
      await queryClient.cancelQueries({ queryKey: toolQueryKeys.toolSet(chatId) });

      // Snapshot previous value for rollback
      const previousToolSet = queryClient.getQueryData<ToolSet>(toolQueryKeys.toolSet(chatId));

      // Optimistically update the cache
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
      // CRITICAL: Only invalidate if there are no more pending toggle mutations.
      // This prevents race conditions during batch operations where multiple toggles
      // fire in sequence (e.g., scenario-level toggle). Without this check, each
      // mutation's onSettled would trigger a refetch that could overwrite subsequent
      // optimistic updates.
      //
      // We use setTimeout to defer the check until after React's state updates,
      // ensuring we see the correct count of pending mutations.
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

  // Sync tools (discover from all running scenarios)
  const syncMutation = useMutation({
    mutationFn: syncTools,
    onSuccess: () => {
      // Invalidate all tool queries to reload discovered tools
      queryClient.invalidateQueries({ queryKey: toolQueryKeys.all });
    },
  });

  // Derived data: enabled tools only
  // CRITICAL: Must memoize AND use stable empty array to prevent creating
  // new references on every render which causes infinite re-render loops.
  const enabledTools = useMemo(() => {
    const tools = toolSet?.tools;
    if (!tools || tools.length === 0) return EMPTY_TOOLS;
    const filtered = tools.filter((t) => t.enabled);
    return filtered.length === 0 ? EMPTY_TOOLS : filtered;
  }, [toolSet?.tools]);

  // Derived data: tools grouped by scenario
  // CRITICAL: Must memoize AND use stable empty map to prevent new references on every render
  const toolsByScenario = useMemo(() => {
    const tools = toolSet?.tools;
    if (!tools || tools.length === 0) return EMPTY_SCENARIO_MAP;
    const map = new Map<string, EffectiveTool[]>();
    for (const tool of tools) {
      const existing = map.get(tool.scenario) ?? [];
      map.set(tool.scenario, [...existing, tool]);
    }
    return map;
  }, [toolSet?.tools]);

  // Derived data: tools grouped by category
  // CRITICAL: Must memoize AND use stable empty map to prevent new references on every render
  const toolsByCategory = useMemo(() => {
    const tools = toolSet?.tools;
    if (!tools || tools.length === 0) return EMPTY_CATEGORY_MAP;
    const map = new Map<string, EffectiveTool[]>();
    for (const tool of tools) {
      const category = tool.tool.category ?? "uncategorized";
      const existing = map.get(category) ?? [];
      map.set(category, [...existing, tool]);
    }
    return map;
  }, [toolSet?.tools]);

  // CRITICAL: Memoize action functions to prevent creating new references on every render.
  // These functions are used as dependencies in consuming components (e.g., App.tsx's
  // handleTemplateActivated depends on enableToolsByIds). Without memoization, every
  // render creates new function references, causing cascading re-renders.
  const toggleTool = useCallback(
    async (scenario: string, toolName: string, enabled: boolean) => {
      await toggleMutation.mutateAsync({ scenario, toolName, enabled });
    },
    [toggleMutation.mutateAsync]
  );

  const setApproval = useCallback(
    async (scenario: string, toolName: string, override: ApprovalOverride) => {
      await approvalMutation.mutateAsync({ scenario, toolName, override });
    },
    [approvalMutation.mutateAsync]
  );

  const resetTool = useCallback(
    async (scenario: string, toolName: string) => {
      await resetMutation.mutateAsync({ scenario, toolName });
    },
    [resetMutation.mutateAsync]
  );

  const syncDiscoveredTools = useCallback(async () => {
    return await syncMutation.mutateAsync();
  }, [syncMutation.mutateAsync]);

  const refetchTools = useCallback(() => {
    refetch();
  }, [refetch]);

  const enableToolsByIds = useCallback(
    async (toolIds: string[]) => {
      // Enable tools in sequence to avoid race conditions with optimistic updates
      for (const toolId of toolIds) {
        const parts = toolId.split(':');
        let scenario: string | undefined;
        let toolName: string | undefined;

        if (parts.length === 2) {
          // New format: "scenario:toolName"
          [scenario, toolName] = parts;
        } else {
          // Old format: just "toolName" - need to find the scenario
          toolName = toolId;
          for (const [scenarioKey, tools] of toolsByScenario.entries()) {
            if (tools.some(t => t.tool.name === toolId)) {
              scenario = scenarioKey;
              break;
            }
          }
        }

        if (scenario && toolName) {
          await toggleMutation.mutateAsync({ scenario, toolName, enabled: true });
        }
      }
    },
    [toolsByScenario, toggleMutation.mutateAsync]
  );

  // Compute derived values outside useMemo to avoid including mutation objects in deps
  const isSyncing = syncMutation.isPending;
  const isUpdating = toggleMutation.isPending || approvalMutation.isPending || resetMutation.isPending;
  const errorValue = error as Error | null;
  const scenariosErrorValue = scenariosError as Error | null;

  // CRITICAL: Memoize the return object to prevent creating new object references
  // on every render. Without this, every render creates a new object that triggers
  // re-renders in consuming components, potentially causing "too many re-renders" errors.
  return useMemo(
    () => ({
      // Data
      toolSet,
      scenarios,
      enabledTools,
      toolsByScenario,
      toolsByCategory,

      // Loading states
      isLoading,
      isLoadingScenarios,
      isSyncing,
      isUpdating,

      // Error states
      error: errorValue,
      scenariosError: scenariosErrorValue,

      // Actions - use memoized callbacks defined above
      toggleTool,
      setApproval,
      resetTool,
      syncDiscoveredTools,
      refetch: refetchTools,
      enableToolsByIds,
    }),
    [
      toolSet,
      scenarios,
      enabledTools,
      toolsByScenario,
      toolsByCategory,
      isLoading,
      isLoadingScenarios,
      isSyncing,
      isUpdating,
      errorValue,
      scenariosErrorValue,
      toggleTool,
      setApproval,
      resetTool,
      syncDiscoveredTools,
      refetchTools,
      enableToolsByIds,
    ]
  );
}

/**
 * Get the count of enabled tools for display in UI badges.
 */
export function useEnabledToolCount(chatId?: string): number {
  const { enabledTools } = useTools({ chatId });
  return enabledTools.length;
}
