import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { queryKeys } from '@/lib/queryKeys';
import type { SkillResponse, SkillsSyncResult } from '@/types/api';

/**
 * Hook to fetch all skills from prompt-manager via the API.
 */
export function useSkills() {
  return useQuery<SkillResponse[]>({
    queryKey: queryKeys.skills.list(),
    queryFn: () => api.listSkills(),
  });
}

/**
 * Hook to trigger a sync of skills from prompt-manager.
 * Invalidates skills and phase names queries on success.
 */
export function useSyncSkills() {
  const queryClient = useQueryClient();

  return useMutation<SkillsSyncResult>({
    mutationFn: () => api.syncSkills(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.skills.list() });
    },
  });
}

/**
 * Hook to get only steer-mode skills (skills that have "Steer" in their modes).
 */
export function useSteerSkills() {
  const { data: skills, ...rest } = useSkills();

  const steerSkills = skills?.filter((skill) => skill.modes?.[0]?.toLowerCase() === 'steer');

  return {
    data: steerSkills,
    ...rest,
  };
}
