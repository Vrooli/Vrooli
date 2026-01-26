/**
 * useConfigUpdate Hook
 *
 * Provides functions to update runtime configuration values.
 * Only options marked as editable can be changed at runtime.
 */

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useCallback } from 'react';
import { getConfig } from '@/config';
import { logger } from '@/utils/logger';
import type { ConfigUpdateResult, ConfigUpdateRequest } from '../types';

const OBSERVABILITY_QUERY_KEY = 'observability';

interface UseConfigUpdateOptions {
  /** Callback when config update succeeds */
  onSuccess?: (result: ConfigUpdateResult) => void;
  /** Callback when config update fails */
  onError?: (error: Error) => void;
}

interface UseConfigUpdateReturn {
  /** Update a config value */
  updateConfig: (envVar: string, value: string) => Promise<ConfigUpdateResult>;
  /** Reset a config value to default */
  resetConfig: (envVar: string) => Promise<{ success: boolean; current_value?: string }>;
  /** Whether an update is in progress */
  isUpdating: boolean;
  /** Last error */
  error: Error | null;
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null;

const isString = (value: unknown): value is string => typeof value === 'string';
const isBoolean = (value: unknown): value is boolean => typeof value === 'boolean';

const isConfigUpdateResult = (value: unknown): value is ConfigUpdateResult => {
  if (!isRecord(value)) return false;
  if (!isBoolean(value.success)) return false;
  if (value.env_var !== undefined && !isString(value.env_var)) return false;
  if (value.new_value !== undefined && !isString(value.new_value)) return false;
  if (value.previous_value !== undefined && !isString(value.previous_value)) return false;
  if (value.error !== undefined && !isString(value.error)) return false;
  return true;
};

const isResetResult = (value: unknown): value is { success: boolean; current_value?: string } => {
  if (!isRecord(value)) return false;
  if (!isBoolean(value.success)) return false;
  if (value.current_value !== undefined && !isString(value.current_value)) return false;
  return true;
};

async function updateConfigRequest(envVar: string, value: string): Promise<ConfigUpdateResult> {
  const config = await getConfig();
  const request: ConfigUpdateRequest = { value };

  const response = await fetch(`${config.API_URL}/observability/config/${envVar}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(request),
  });

  const result: unknown = await response.json();
  if (!isConfigUpdateResult(result)) {
    throw new Error(`Failed to update config: ${response.statusText}`);
  }

  if (!response.ok && result.success !== true) {
    const message = typeof result.error === 'string'
      ? result.error
      : `Failed to update config: ${response.statusText}`;
    throw new Error(message);
  }

  return result;
}

async function resetConfigRequest(envVar: string): Promise<{ success: boolean; current_value?: string }> {
  const config = await getConfig();

  const response = await fetch(`${config.API_URL}/observability/config/${envVar}`, {
    method: 'DELETE',
  });

  const result: unknown = await response.json();
  if (!response.ok) {
    const message = isRecord(result) && typeof result.error === 'string'
      ? result.error
      : `Failed to reset config: ${response.statusText}`;
    throw new Error(message);
  }

  if (!isResetResult(result)) {
    throw new Error(`Failed to reset config: ${response.statusText}`);
  }

  return result;
}

export function useConfigUpdate(options: UseConfigUpdateOptions = {}): UseConfigUpdateReturn {
  const { onSuccess, onError } = options;
  const queryClient = useQueryClient();

  const updateMutation = useMutation({
    mutationFn: ({ envVar, value }: { envVar: string; value: string }) =>
      updateConfigRequest(envVar, value),
    onSuccess: (result) => {
      // Invalidate observability queries to pick up new data
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
