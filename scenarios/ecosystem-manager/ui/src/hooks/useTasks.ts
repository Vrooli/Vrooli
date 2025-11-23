/**
 * useTasks Hook
 * Fetches tasks with optional filters using React Query
 */

import { useQuery } from '@tanstack/react-query';
import { api } from '../lib/api';
import { queryKeys } from '../lib/queryKeys';
import type { TaskFilters, Task, TaskStatus } from '../types/api';

const ALL_STATUSES: TaskStatus[] = [
  'pending',
  'in-progress',
  'completed',
  'completed-finalized',
  'failed',
  'failed-blocked',
  'archived',
  'review',
];

export function useTasks(filters: TaskFilters = {}) {
  return useQuery<Task[]>({
    queryKey: queryKeys.tasks.list(filters),
    queryFn: async () => {
      // When a specific status is provided, fetch just that set
      if (filters.status) {
        const tasks = await api.getTasks(filters);
        return Array.isArray(tasks) ? tasks : [];
      }

      // Otherwise fetch each status to mirror the legacy UI behavior
      const results = await Promise.all(
        ALL_STATUSES.map(async (status) => {
          const tasks = await api.getTasks({ ...filters, status });
          return Array.isArray(tasks) ? tasks : [];
        })
      );

      return results.flat();
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
