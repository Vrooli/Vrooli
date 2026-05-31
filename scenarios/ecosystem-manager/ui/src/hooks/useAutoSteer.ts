/**
 * Auto Steer Hooks
 * Hooks for managing Auto Steer profiles
 */

import { useMemo } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api, ApiError } from '@/lib/api';
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
        if (err instanceof ApiError && err.status === 404) return undefined;
        throw err;
      }
    },
    enabled: !!taskId,
    staleTime: 15000,
    retry: 1,
  });
}

/**
 * Fetch the controller's decision trace for a task (survives finalization).
 */
export function useAutoSteerDecisionTrace(taskId?: string) {
  return useQuery({
    queryKey: queryKeys.autoSteer.decisionTrace(taskId ?? 'none'),
    queryFn: async () => {
      const resp = await api.getAutoSteerDecisionTrace(taskId as string);
      return resp?.trace ?? [];
    },
    enabled: !!taskId,
    staleTime: 15000,
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
 * Initialize the controller for a task: runs the first audit and selects the
 * first skill. The controller derives everything else — there is no phase cursor
 * to seek to.
 */
export function useStartAutoSteerExecution() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({
      taskId,
      profileId,
      scenarioName,
    }: {
      taskId: string;
      profileId: string;
      scenarioName: string;
    }) => api.startAutoSteerExecution(taskId, profileId, scenarioName),
    onSuccess: (state) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.autoSteer.executionState(state?.task_id ?? 'none') });
    },
  });
}

/**
 * Fetch the canonical improvement-dimension vocabulary (SSOT) for the editor.
 */
export function useAutoSteerDimensions() {
  return useQuery({
    queryKey: queryKeys.autoSteer.dimensions(),
    queryFn: () => api.getAutoSteerDimensions(),
    staleTime: Infinity, // vocabulary is effectively static for a session
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

/**
 * Fetch all Auto Steer profiles AND templates merged into a single list.
 * Templates whose ID already appears in profiles are deduplicated.
 * Use this for display/lookup where both custom profiles and built-in templates must resolve.
 */
export function useAllAutoSteerProfiles() {
  const { data: profiles = [], isLoading: isLoadingProfiles } = useAutoSteerProfiles();
  const { data: templates = [], isLoading: isLoadingTemplates } = useAutoSteerTemplates();

  const merged = useMemo<AutoSteerProfile[]>(() => {
    const profileIds = new Set(profiles.map((p) => p.id));
    const fromTemplates = templates.filter((t) => !profileIds.has(t.id));
    return [...profiles, ...fromTemplates];
  }, [profiles, templates]);

  return { data: merged, isLoading: isLoadingProfiles || isLoadingTemplates };
}
