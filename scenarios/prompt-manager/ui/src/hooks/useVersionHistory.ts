/**
 * useVersionHistory - Data fetching hooks for skill version history.
 *
 * Handles:
 * - Fetching version history via react-query
 * - Revert mutation with cache invalidation
 */

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '@/lib/api'
import type { VersionsResponse, RevertResponse } from '@/lib/schemas'

const QUERY_KEYS = {
  versions: (skillId: string) => ['versions', skillId] as const,
}

/**
 * Fetch version history for a skill.
 */
export function useVersionHistory(skillId: string | null) {
  return useQuery<VersionsResponse>({
    queryKey: QUERY_KEYS.versions(skillId ?? ''),
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion -- guarded by enabled: !!skillId
    queryFn: () => api.getSkillVersions(skillId!),
    enabled: !!skillId,
    staleTime: 10_000,
  })
}

/**
 * Mutation to revert a skill to a previous version.
 */
export function useRevertVersion() {
  const queryClient = useQueryClient()

  return useMutation<RevertResponse, Error, { skillId: string; version: number }>({
    mutationFn: ({ skillId, version }) => api.revertSkillVersion(skillId, version),
    onSuccess: (_data, variables) => {
      void queryClient.invalidateQueries({ queryKey: QUERY_KEYS.versions(variables.skillId) })
      void queryClient.invalidateQueries({ queryKey: ['skills'] })
    },
  })
}
