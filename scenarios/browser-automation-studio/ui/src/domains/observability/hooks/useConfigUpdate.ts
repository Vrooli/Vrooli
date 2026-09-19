/**
 * useConfigUpdate Hook
 *
 * Provides functions to update runtime configuration values.
 * Only options marked as editable can be changed at runtime.
 *
 * Transport: Connect-RPC via `observability` client in src/api/observability.ts.
 */

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useCallback } from 'react';
import { observability } from '@/api/observability';
import { logger } from '@/utils/logger';
import type { ConfigUpdateResult } from '../types';

const OBSERVABILITY_QUERY_KEY = 'observability';

interface UseConfigUpdateOptions {
  onSuccess?: (result: ConfigUpdateResult) => void;
  onError?: (error: Error) => void;
}

interface UseConfigUpdateReturn {
  updateConfig: (envVar: string, value: string) => Promise<ConfigUpdateResult>;
  resetConfig: (envVar: string) => Promise<{ success: boolean; current_value?: string }>;
  isUpdating: boolean;
  error: Error | null;
}

async function updateConfigRequest(envVar: string, value: string): Promise<ConfigUpdateResult> {
  return observability.updateConfig<ConfigUpdateResult>(envVar, value);
}

async function resetConfigRequest(envVar: string): Promise<{ success: boolean; current_value?: string }> {
  return observability.resetConfig<{ success: boolean; current_value?: string }>(envVar);
}

export function useConfigUpdate(options: UseConfigUpdateOptions = {}): UseConfigUpdateReturn {
  const { onSuccess, onError } = options;
  const queryClient = useQueryClient();

  const updateMutation = useMutation({
    mutationFn: ({ envVar, value }: { envVar: string; value: string }) =>
      updateConfigRequest(envVar, value),
    onSuccess: (result) => {
      queryClient.invalidateQueries({ queryKey: [OBSERVABILITY_QUERY_KEY] });
      onSuccess?.(result);
    },
    onError: (error: Error) => {
      logger.error('Config update failed', { component: 'useConfigUpdate', action: 'updateConfig' }, error);
      onError?.(error);
    },
  });

  const resetMutation = useMutation({
    mutationFn: (envVar: string) => resetConfigRequest(envVar),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [OBSERVABILITY_QUERY_KEY] });
    },
    onError: (error: Error) => {
      logger.error('Config reset failed', { component: 'useConfigUpdate', action: 'resetConfig' }, error);
      onError?.(error);
    },
  });

  const updateConfig = useCallback(
    async (envVar: string, value: string): Promise<ConfigUpdateResult> => {
      return updateMutation.mutateAsync({ envVar, value });
    },
    [updateMutation]
  );

  const resetConfig = useCallback(
    async (envVar: string): Promise<{ success: boolean; current_value?: string }> => {
      return resetMutation.mutateAsync(envVar);
    },
    [resetMutation]
  );

  return {
    updateConfig,
    resetConfig,
    isUpdating: updateMutation.isPending || resetMutation.isPending,
    error: updateMutation.error || resetMutation.error,
  };
}

export default useConfigUpdate;
