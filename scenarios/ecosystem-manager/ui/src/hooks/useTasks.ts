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
        return applyClientFilters(Array.isArray(tasks) ? tasks : [], filters);
      }

      // Otherwise fetch each status to mirror the legacy UI behavior
      const results = await Promise.all(
        ALL_STATUSES.map(async (status) => {
          const tasks = await api.getTasks({ ...filters, status });
          return Array.isArray(tasks) ? tasks : [];
        })
      );

      return applyClientFilters(results.flat(), filters);
    },
    staleTime: 30000, // 30 seconds
  });
}

function applyClientFilters(tasks: Task[], filters: TaskFilters) {
  const search = filters.search?.trim().toLowerCase() ?? '';
  const priority = filters.priority;
  const type = filters.type;
  const operation = filters.operation;

  return tasks.filter((task) => {
    if (priority && task.priority !== priority) return false;
    if (type && task.type !== type) return false;
    if (operation && task.operation !== operation) return false;

    if (search) {
      const haystack = [
        task.title,
        task.notes,
        task.id,
        ...(task.target || []),
      ]
        .filter(Boolean)
        .join(' ')
        .toLowerCase();

      if (!haystack.includes(search)) return false;
    }

    return true;
  });
}

/**
 * useTasksByStatus Hook
 * Convenience hook for fetching tasks by a specific status
 */
export function useTasksByStatus(status: string) {
  return useTasks({ status: status as any });
}
