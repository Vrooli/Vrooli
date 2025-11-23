/**
 * useTasks Hook
 * Fetches tasks with optional filters using React Query
 */

import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import { queryKeys } from '../lib/queryKeys';
import type { TaskFilters, Task } from '../types/api';

export function useTasks(filters: TaskFilters = {}) {
  return useQuery<Task[]>({
    queryKey: queryKeys.tasks.list(filters),
    queryFn: async () => {
      const tasks = await api.getTasks(filters);
      // Ensure tasks is an array
      return Array.isArray(tasks) ? tasks : [];
    },
    staleTime: 30000, // 30 seconds
  });
}

/**
 * useTasksByStatus Hook
 * Convenience hook for fetching tasks by a specific status
 */
export function useTasksByStatus(status: string) {
  return useTasks({ status: status as any });
}
