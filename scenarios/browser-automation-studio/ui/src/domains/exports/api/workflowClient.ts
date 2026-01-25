/**
 * Workflow API Client
 *
 * Provides a testable seam for workflow-related API operations
 * used by the exports domain.
 */

import { z } from 'zod';
import { getConfig } from '@/config';
import { safeParse } from '@/shared/api';
import { logger } from '@/utils/logger';

// =============================================================================
// Schemas (Runtime Validation)
// =============================================================================

/**
 * Zod schema for workflow API response validation.
 * Validates the shape of data received from the API at runtime.
 */
const WorkflowResponseSchema = z.object({
  id: z.string().optional(),
  name: z.string().optional(),
  project_id: z.string().nullable().optional(),
  projectId: z.string().nullable().optional(),
});

type WorkflowResponse = z.infer<typeof WorkflowResponseSchema>;

// =============================================================================
// Types
// =============================================================================

export interface WorkflowInfo {
  id: string;
  name: string;
  projectId: string | null;
}

export interface WorkflowApiClient {
  /**
   * Fetches workflow information by ID.
   */
  fetchWorkflow(workflowId: string, signal?: AbortSignal): Promise<WorkflowInfo>;
}

// =============================================================================
// Default Implementation
// =============================================================================

async function fetchWorkflow(
  workflowId: string,
  signal?: AbortSignal,
): Promise<WorkflowInfo> {
  const config = await getConfig();
  const response = await fetch(`${config.API_URL}/workflows/${workflowId}`, {
    signal,
  });

  if (!response.ok) {
    throw new Error(`Failed to fetch workflow: ${response.status}`);
  }

  const raw: unknown = await response.json();

  // Validate response at runtime using Zod
  const result = safeParse(WorkflowResponseSchema, raw, 'fetchWorkflow');
  if (!result.success) {
    // Log validation failure but continue with defensive defaults
    logger.warn('Workflow response validation failed', {
      component: 'workflowClient',
      action: 'fetchWorkflow',
      workflowId,
      error: result.error,
    });
  }

  // Use validated data if available, fall back to raw with defensive defaults
  const data: WorkflowResponse = result.success ? result.data : (raw as WorkflowResponse);

  return {
    id: workflowId,
    name: data.name ?? 'Workflow',
    projectId: data.project_id ?? data.projectId ?? null,
  };
}

/**
 * Default workflow API client implementation.
 */
export const defaultWorkflowApiClient: WorkflowApiClient = {
  fetchWorkflow,
};

// =============================================================================
// Testing Utilities
// =============================================================================

/**
 * Creates a mock workflow API client for testing.
 */
export function createMockWorkflowApiClient(
  overrides: Partial<WorkflowApiClient> = {},
): WorkflowApiClient {
  return {
    fetchWorkflow: overrides.fetchWorkflow ?? (async (id) => ({
      id,
      name: 'Test Workflow',
      projectId: null,
    })),
  };
}
