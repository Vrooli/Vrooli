import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../lib/queryKeys';
import { api } from '../lib/api';

/**
 * Hook to fetch queue/processor status with automatic polling
 */
export function useQueueStatus() {
  return useQuery({
    queryKey: queryKeys.queue.status(),
    queryFn: () => api.getQueueStatus(),
    refetchInterval: 5000, // Poll every 5 seconds
    staleTime: 4000, // Consider stale after 4 seconds
  });
}

/**
 * Hook to toggle processor (start/stop)
 */
export function useToggleProcessor() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: async (action: 'start' | 'stop') => {
      if (action === 'start') {
        return api.startProcessor();
      } else {
        return api.stopProcessor();
      }
    },
    onSuccess: () => {
      // Invalidate queue status to reflect new state
      queryClient.invalidateQueries({ queryKey: queryKeys.queue.status() });
    },
  });
}

/**
 * Hook to trigger manual queue processing
 */
export function useTriggerQueue() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => api.triggerQueue(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.queue.status() });
      queryClient.invalidateQueries({ queryKey: queryKeys.tasks.all });
    },
  });
}

/**
 * Hook to reset rate limit
 */
export function useResetRateLimit() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: () => api.resetRateLimit(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.queue.status() });
    },
  });
}
