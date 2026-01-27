import { useMemo } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { queryKeys } from '@/lib/queryKeys';
import type { PromptFileInfo, PromptFile, PhaseInfo } from '@/types/api';
import { normalizeSteerMode } from '@/lib/utils';

export function usePromptFiles() {
  return useQuery<PromptFileInfo[]>({
    queryKey: queryKeys.prompts.list(),
    queryFn: () => api.listPromptFiles(),
  });
}

export function usePromptFile(id?: string) {
  return useQuery<PromptFile>({
    queryKey: queryKeys.prompts.file(id || 'none'),
    queryFn: () => api.getPromptFile(id as string),
    enabled: !!id,
  });
}

export function useSavePromptFile() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, content }: { id: string; content: string }) => api.updatePromptFile(id, content),
    onSuccess: (data) => {
      queryClient.setQueryData(queryKeys.prompts.file(data.id), data);
    },
  });
}

export function useCreatePromptFile() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ path, content }: { path: string; content: string }) => api.createPromptFile(path, content),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.prompts.list() });
      queryClient.setQueryData(queryKeys.prompts.file(data.id), data);
    },
  });
}

/**
 * Hook to fetch phase names from prompt-manager skills.
 * Skills with "Steer" mode are converted to PhaseInfo format.
 */
export function useMergedPhaseNames() {
  const skillsQuery = useQuery({
    queryKey: queryKeys.skills.list(),
    queryFn: () => api.listSkills(),
  });

  const phases = useMemo(() => {
    const skills = skillsQuery.data ?? [];

    return skills
      .filter((skill) => normalizeSteerMode(skill.modes?.[0]) === 'steer')
      .map((skill) => {
        const modes = (skill.modes ?? []).slice(1).map((mode) => mode.trim()).filter(Boolean);
        return {
          id: normalizeSteerMode(skill.id),
          name: skill.name,
          description: skill.description,
          modes,
          source: 'prompt-manager' as const,
        };
      })
      .filter((phase) => phase.id && normalizeSteerMode(phase.name));
  }, [skillsQuery.data]);

  return {
    data: phases,
    isLoading: skillsQuery.isLoading,
    isError: skillsQuery.isError,
    error: skillsQuery.error,
  };
}
