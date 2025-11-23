import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../lib/queryKeys';
import { api } from '../lib/api';

/**
 * Hook to fetch running processes with automatic polling
 */
export function useRunningProcesses() {
  return useQuery({
    queryKey: queryKeys.processes.running(),
    queryFn: async () => {
      const processes = await api.getRunningProcesses();
      // Ensure processes is an array
      return Array.isArray(processes) ? processes : [];
    },
    refetchInterval: 5000, // Poll every 5 seconds
    staleTime: 4000, // Consider stale after 4 seconds
  });
}

/**
 * Hook to terminate a running process
 */
export function useTerminateProcess() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (processId: string) => api.terminateProcess(processId),
    onSuccess: () => {
      // Invalidate processes to update the list
      queryClient.invalidateQueries({ queryKey: queryKeys.processes.running() });
      // Also invalidate tasks in case status changed
      queryClient.invalidateQueries({ queryKey: queryKeys.tasks.all });
    },
  });
}
