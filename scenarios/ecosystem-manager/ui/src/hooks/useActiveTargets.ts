/**
 * useActiveTargets Hook
 * Fetches available targets for improver tasks
 */

import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';
import { queryKeys } from '@/lib/queryKeys';

export function useActiveTargets(type?: string, operation?: string) {
  return useQuery({
    queryKey: queryKeys.tasks.activeTargets(type, operation),
    queryFn: async () => {
      const targets = await api.getActiveTargets(type, operation);
      return Array.isArray(targets) ? targets : [];
    },
    enabled: !!type && !!operation,
    staleTime: 30000, // 30 seconds
  });
}
