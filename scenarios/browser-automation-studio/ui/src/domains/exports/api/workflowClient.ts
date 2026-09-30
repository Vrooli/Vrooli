/**
 * Workflow API Client
 *
 * Provides a testable seam for workflow-related API operations
 * used by the exports domain.
 */

import { getWorkflowViaApi } from '@/domains/workflows/services/workflowApi';

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
  _signal?: AbortSignal,
): Promise<WorkflowInfo> {
  const resp = await getWorkflowViaApi(workflowId);
  return {
    id: workflowId,
    name: resp.workflow?.name || 'Workflow',
    projectId: resp.workflow?.projectId || null,
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
