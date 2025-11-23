/**
 * System Logs Query Hook
 * Fetches and manages system logs with optional polling
 */

import { useQuery } from '@tanstack/react-query';
import { queryKeys } from '../lib/queryKeys';
import { api } from '../lib/api';

export interface UseSystemLogsOptions {
  limit?: number;
  level?: 'all' | 'info' | 'warning' | 'error';
  refetchInterval?: number;
}

export function useSystemLogs(options: UseSystemLogsOptions = {}) {
  const { limit = 500, level = 'all', refetchInterval } = options;

  const query = useQuery({
    queryKey: [queryKeys.logs.system(limit), { level }],
    queryFn: async () => {
      try {
        const logs = await api.getSystemLogs(limit);

        // Ensure logs is an array
        if (!Array.isArray(logs)) {
          console.error('getSystemLogs did not return an array:', logs);
          return [];
        }

        // Filter by level if not 'all'
        if (level === 'all') {
          return logs;
        }

        return logs.filter(log => log.level === level);
      } catch (error) {
        console.error('Error fetching system logs:', error);
        return [];
      }
    },
    refetchInterval,
    staleTime: 5000, // Consider stale after 5 seconds
  });

  return query;
}
