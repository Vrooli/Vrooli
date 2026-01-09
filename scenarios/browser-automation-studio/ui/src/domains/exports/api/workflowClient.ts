/**
 * Workflow API Client
 *
 * Provides a testable seam for workflow-related API operations
 * used by the exports domain.
 */

import { getConfig } from '@/config';

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

  const data = await response.json();

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
