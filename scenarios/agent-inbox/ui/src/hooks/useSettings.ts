/**
 * useSettings Hook
 *
 * Provides settings state management with react-query.
 * Currently handles YOLO mode setting.
 *
 * ARCHITECTURE:
 * - Uses react-query for caching and state management
 * - Optimistic updates for responsive UI
 * - Automatic cache invalidation on mutations
 */

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { getYoloMode, setYoloMode as setYoloModeApi } from "../lib/api";

// Query keys for cache management
export const settingsQueryKeys = {
  all: ["settings"] as const,
  yoloMode: () => [...settingsQueryKeys.all, "yolo-mode"] as const,
};

export interface UseYoloModeReturn {
  yoloMode: boolean;
  isLoading: boolean;
  isUpdating: boolean;
  error: Error | null;
  setYoloMode: (enabled: boolean) => Promise<void>;
}

/**
 * Hook for managing YOLO mode setting.
 *
 * YOLO mode bypasses all tool approval requirements when enabled.
 *
 * @param enabled - Whether to fetch the setting on mount (default: true)
 *
 * @example
 * const { yoloMode, setYoloMode, isLoading } = useYoloMode();
 */
export function useYoloMode(enabled = true): UseYoloModeReturn {
  const queryClient = useQueryClient();

  // Fetch YOLO mode setting
  const {
    data: yoloMode = false,
    isLoading,
    error,
  } = useQuery({
    queryKey: settingsQueryKeys.yoloMode(),
    queryFn: getYoloMode,
    enabled,
    staleTime: 60_000, // 1 minute
  });

  // Update YOLO mode
  const mutation = useMutation({
    mutationFn: setYoloModeApi,
    onMutate: async (newEnabled) => {
      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: settingsQueryKeys.yoloMode() });

      // Snapshot previous value
      const previousValue = queryClient.getQueryData<boolean>(settingsQueryKeys.yoloMode());

      // Optimistically update
      queryClient.setQueryData(settingsQueryKeys.yoloMode(), newEnabled);

      return { previousValue };
    },
    onError: (_err, _variables, context) => {
      // Rollback on error
      if (context?.previousValue !== undefined) {
        queryClient.setQueryData(settingsQueryKeys.yoloMode(), context.previousValue);
      }
    },
    onSettled: () => {
      // Refetch to ensure consistency
      queryClient.invalidateQueries({ queryKey: settingsQueryKeys.yoloMode() });
    },
  });

  return {
    yoloMode,
    isLoading,
    isUpdating: mutation.isPending,
    error: error as Error | null,
    setYoloMode: async (enabled) => {
      await mutation.mutateAsync(enabled);
    },
  };
}
