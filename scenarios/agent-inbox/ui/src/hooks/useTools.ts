/**
 * useTools Hook
 *
 * Provides tool configuration state management with react-query.
 * Supports both global tool configuration and per-chat overrides.
 *
 * ARCHITECTURE:
 * - Uses react-query for caching and state management
 * - Mutations extracted to useToolMutations.ts
 * - This file handles queries, derived data, and action wrappers
 *
 * TESTING SEAMS:
 * - All API calls go through the api.ts module
 * - Query client can be mocked in tests
 */

import { useCallback, useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  fetchToolSet,
  fetchScenarioStatuses,
  type ToolSet,
  type ScenarioStatus,
  type EffectiveTool,
  type ApprovalOverride,
  type DiscoveryResult,
} from "../lib/api";
import { useToolMutations } from "./useToolMutations";

// Query keys for cache management
export const toolQueryKeys = {
  all: ["tools"] as const,
  toolSet: (chatId?: string) => [...toolQueryKeys.all, "set", chatId ?? "global"] as const,
  scenarios: () => [...toolQueryKeys.all, "scenarios"] as const,
};

// Stable empty arrays/maps to prevent creating new references on every render.
const EMPTY_TOOLS: EffectiveTool[] = [];
const EMPTY_SCENARIO_MAP: Map<string, EffectiveTool[]> = new Map();
const EMPTY_CATEGORY_MAP: Map<string, EffectiveTool[]> = new Map();

export interface UseToolsOptions {
  chatId?: string;
  enabled?: boolean;
}

export interface UseToolsReturn {
  toolSet: ToolSet | undefined;
  scenarios: ScenarioStatus[] | undefined;
  enabledTools: EffectiveTool[];
  toolsByScenario: Map<string, EffectiveTool[]>;
  toolsByCategory: Map<string, EffectiveTool[]>;
  isLoading: boolean;
  isLoadingScenarios: boolean;
  isSyncing: boolean;
  isUpdating: boolean;
  error: Error | null;
  scenariosError: Error | null;
  toggleTool: (scenario: string, toolName: string, enabled: boolean) => Promise<void>;
  setApproval: (scenario: string, toolName: string, override: ApprovalOverride) => Promise<void>;
  resetTool: (scenario: string, toolName: string) => Promise<void>;
  syncDiscoveredTools: () => Promise<DiscoveryResult>;
  refetch: () => void;
  enableToolsByIds: (toolIds: string[]) => Promise<void>;
}

export function useTools(options: UseToolsOptions = {}): UseToolsReturn {
  const { chatId, enabled = true } = options;

  // Fetch tool set
  const {
    data: toolSet,
    isLoading,
    error,
    refetch,
  } = useQuery({
    queryKey: toolQueryKeys.toolSet(chatId),
    queryFn: () => fetchToolSet(chatId),
    enabled,
    staleTime: 60_000,
    gcTime: 300_000,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
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
    staleTime: 30_000,
    gcTime: 300_000,
    refetchOnMount: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });

  // Mutations from extracted module
  const { toggleMutation, approvalMutation, resetMutation, syncMutation } =
    useToolMutations({ chatId });

  // Derived data: enabled tools only
  const enabledTools = useMemo(() => {
    const tools = toolSet?.tools;
    if (!tools || tools.length === 0) return EMPTY_TOOLS;
    const filtered = tools.filter((t) => t.enabled);
    return filtered.length === 0 ? EMPTY_TOOLS : filtered;
  }, [toolSet?.tools]);

  // Derived data: tools grouped by scenario
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

  // Memoized action functions
  const toggleTool = useCallback(
    async (scenario: string, toolName: string, enabled: boolean) => {
      await toggleMutation.mutateAsync({ scenario, toolName, enabled });
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [toggleMutation.mutateAsync]
  );

  const setApproval = useCallback(
    async (scenario: string, toolName: string, override: ApprovalOverride) => {
      await approvalMutation.mutateAsync({ scenario, toolName, override });
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [approvalMutation.mutateAsync]
  );

  const resetTool = useCallback(
    async (scenario: string, toolName: string) => {
      await resetMutation.mutateAsync({ scenario, toolName });
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [resetMutation.mutateAsync]
  );

  const syncDiscoveredTools = useCallback(async () => {
    return await syncMutation.mutateAsync();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [syncMutation.mutateAsync]);

  const refetchTools = useCallback(() => {
    refetch();
  }, [refetch]);

  const enableToolsByIds = useCallback(
    async (toolIds: string[]) => {
      for (const toolId of toolIds) {
        const parts = toolId.split(':');
        let scenario: string | undefined;
        let toolName: string | undefined;

        if (parts.length === 2) {
          [scenario, toolName] = parts;
        } else {
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
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [toolsByScenario, toggleMutation.mutateAsync]
  );

  const isSyncing = syncMutation.isPending;
  const isUpdating = toggleMutation.isPending || approvalMutation.isPending || resetMutation.isPending;
  const errorValue = error as Error | null;
  const scenariosErrorValue = scenariosError as Error | null;

  return useMemo(
    () => ({
      toolSet,
      scenarios,
      enabledTools,
      toolsByScenario,
      toolsByCategory,
      isLoading,
      isLoadingScenarios,
      isSyncing,
      isUpdating,
      error: errorValue,
      scenariosError: scenariosErrorValue,
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
