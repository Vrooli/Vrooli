/**
 * Task Mutation Hooks
 * Handles task creation, updates, deletion, and status changes
 */

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';
import { queryKeys } from '../lib/queryKeys';
import type { CreateTaskInput, UpdateTaskInput, TaskStatus, Task } from '../types/api';

/**
 * Hook for creating a new task
 */
export function useCreateTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateTaskInput) => api.createTask(data),
    onSuccess: () => {
      // Invalidate all task queries to refetch
      queryClient.invalidateQueries({ queryKey: queryKeys.tasks.all });
    },
  });
}

/**
 * Hook for updating a task
 */
export function useUpdateTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, updates }: { id: string; updates: UpdateTaskInput }) =>
      api.updateTask(id, updates),
    onSuccess: (updatedTask) => {
      // Invalidate specific task and all lists
      queryClient.invalidateQueries({ queryKey: queryKeys.tasks.detail(updatedTask.id) });
      queryClient.invalidateQueries({ queryKey: queryKeys.tasks.lists() });
    },
  });
}

/**
 * Hook for updating task status (used in drag-and-drop)
 */
export function useUpdateTaskStatus() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, status }: { id: string; status: TaskStatus }) =>
      api.updateTaskStatus(id, status),
    onMutate: async ({ id, status }) => {
      // Cancel outgoing refetches
      await queryClient.cancelQueries({ queryKey: queryKeys.tasks.lists() });

      // Snapshot previous value
      const previousTasks = queryClient.getQueriesData({ queryKey: queryKeys.tasks.lists() });

      // Optimistically update to the new status
      queryClient.setQueriesData<Task[]>({ queryKey: queryKeys.tasks.lists() }, (old) => {
        if (!old) return old;
        return old.map(task =>
          task.id === id ? { ...task, status } : task
        );
      });

      return { previousTasks };
    },
    onError: (_err, _variables, context) => {
      // Rollback on error
      if (context?.previousTasks) {
        context.previousTasks.forEach(([queryKey, data]) => {
          queryClient.setQueryData(queryKey, data);
        });
      }
    },
    onSettled: () => {
      // Always refetch after error or success
      queryClient.invalidateQueries({ queryKey: queryKeys.tasks.lists() });
    },
  });
}

/**
 * Hook for deleting a task
 */
export function useDeleteTask() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.deleteTask(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.tasks.all });
    },
  });
}
