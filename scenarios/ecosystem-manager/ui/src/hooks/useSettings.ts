/**
 * useSettings Hook
 * Hooks for managing application settings
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '@/lib/queryKeys';
import { api } from '@/lib/api';
import type { Settings } from '@/types/api';

/**
 * Fetch current settings
 */
export function useSettings() {
  return useQuery({
    queryKey: queryKeys.settings.get(),
    queryFn: () => api.getSettings(),
    staleTime: 60000, // Settings don't change often, cache for 1 minute
  });
}

/**
 * Save settings
 */
export function useSaveSettings() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (settings: Settings) => api.updateSettings(settings),
    onSuccess: (updatedSettings) => {
      // Update the cache with the new settings
      queryClient.setQueryData(queryKeys.settings.get(), updatedSettings);
    },
  });
}

/**
 * Reset settings to defaults
 */
export function useResetSettings() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => api.resetSettings(),
    onSuccess: (resetSettings) => {
      // Update the cache with the reset settings
      queryClient.setQueryData(queryKeys.settings.get(), resetSettings);
    },
  });
}

/**
 * Get recycler models for a specific provider
 */
export function useRecyclerModels(provider: string) {
  return useQuery({
    queryKey: queryKeys.settings.recyclerModels(provider),
    queryFn: () => api.getRecyclerModels(provider),
    enabled: !!provider && provider !== 'off',
    staleTime: 300000, // Cache model list for 5 minutes
  });
}
