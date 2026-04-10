import { getConfig } from '@/config';
import { safeParse } from '@/shared/api/safeParse';
import { ExecutionsListResponseSchema, type ExecutionItem } from '@/shared/api/schemas';

/**
 * Fetch execution list with Zod validation.
 */
export const fetchExecutionsList = async (limit = 100): Promise<ExecutionItem[]> => {
  const { API_URL } = await getConfig();
  const response = await fetch(`${API_URL}/executions?limit=${limit}`);

  if (!response.ok) {
    const message = await response.text().catch(() => '');
    throw new Error(message || `Failed to fetch executions (${response.status})`);
  }

  const payload = await response.json() as Record<string, unknown>;
  const normalized = {
    executions: Array.isArray(payload?.executions) ? payload.executions : [],
  };

  const result = safeParse(ExecutionsListResponseSchema, normalized, 'ExecutionsList');
  if (!result.success) {
    throw new Error(result.error);
  }

  return result.data.executions;
};
