/**
 * Auto Steer Hooks
 * Hooks for managing Auto Steer profiles
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { queryKeys } from '@/lib/queryKeys';
import type { AutoSteerProfile, AutoSteerExecutionState } from '@/types/api';

/**
 * Fetch all Auto Steer profiles
 */
export function useAutoSteerProfiles() {
  return useQuery({
    queryKey: queryKeys.autoSteer.profiles(),
    queryFn: async () => {
      const profiles = await api.getAutoSteerProfiles();
      return Array.isArray(profiles) ? profiles : [];
    },
    staleTime: 60000, // 1 minute
  });
}

/**
 * Fetch a single Auto Steer profile by ID
 */
export function useAutoSteerProfile(id: string) {
  return useQuery({
    queryKey: queryKeys.autoSteer.profile(id),
    queryFn: () => api.getAutoSteerProfile(id),
    enabled: !!id,
  });
}

/**
 * Fetch Auto Steer execution state for a task
 */
export function useAutoSteerExecutionState(taskId?: string) {
  return useQuery({
    queryKey: queryKeys.autoSteer.executionState(taskId ?? 'none'),
    queryFn: async (): Promise<AutoSteerExecutionState | undefined> => {
      try {
        return await api.getAutoSteerExecutionState(taskId as string);
      } catch (err: any) {
        // Treat missing state as undefined instead of leaving stale data around.
        const status = err?.status ?? err?.response?.status;
        if (status === 404) return undefined;
        throw err;
      }
    },
    enabled: !!taskId,
    staleTime: 15000,
    retry: 1,
  });
}

/**
 * Reset Auto Steer execution state for a task
 */
export function useResetAutoSteerExecution() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (taskId: string) => api.resetAutoSteerExecution(taskId),
    onSuccess: (_, taskId) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.autoSteer.executionState(taskId ?? 'none') });
      queryClient.invalidateQueries({ queryKey: queryKeys.tasks.executions(taskId ?? '') });
    },
  });
}

/**
 * Seek Auto Steer execution cursor to a specific phase/iteration
 */
export function useSeekAutoSteerExecution() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ taskId, phaseIndex, phaseIteration }: { taskId: string; phaseIndex: number; phaseIteration: number }) =>
      api.seekAutoSteerExecution(taskId, phaseIndex, phaseIteration),
    onSuccess: (state) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.autoSteer.executionState(state?.task_id ?? 'none') });
    },
  });
}

/**
 * Create a new Auto Steer profile
 */
export function useCreateAutoSteerProfile() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (profile: Omit<AutoSteerProfile, 'id' | 'created_at' | 'updated_at'>) =>
      api.createAutoSteerProfile(profile),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.autoSteer.profiles() });
    },
  });
}

/**
 * Update an existing Auto Steer profile
 */
export function useUpdateAutoSteerProfile() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, updates }: { id: string; updates: Partial<AutoSteerProfile> }) =>
      api.updateAutoSteerProfile(id, updates),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.autoSteer.profile(variables.id) });
      queryClient.invalidateQueries({ queryKey: queryKeys.autoSteer.profiles() });
    },
  });
}

/**
 * Delete an Auto Steer profile
 */
export function useDeleteAutoSteerProfile() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.deleteAutoSteerProfile(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.autoSteer.profiles() });
    },
  });
}

/**
 * Fetch Auto Steer templates
 */
export function useAutoSteerTemplates() {
  return useQuery({
    queryKey: queryKeys.autoSteer.templates(),
    queryFn: async () => {
      const templates = await api.getAutoSteerTemplates();
      return Array.isArray(templates) ? templates : [];
    },
    staleTime: 300000, // 5 minutes
  });
}
