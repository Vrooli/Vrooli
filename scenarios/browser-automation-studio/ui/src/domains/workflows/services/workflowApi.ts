import { getConfig } from '@/config';
import { safeParse } from '@/shared/api/safeParse';
import {
  WorkflowsListResponseSchema,
  WorkflowResponseSchema,
  type WorkflowItem,
} from '@/shared/api/schemas';

/**
 * Fetch workflow list with Zod validation.
 */
export const fetchWorkflowList = async (limit = 100): Promise<WorkflowItem[]> => {
  const { API_URL } = await getConfig();
  const response = await fetch(`${API_URL}/workflows?limit=${limit}`);

  if (!response.ok) {
    const message = await response.text().catch(() => '');
    throw new Error(message || `Failed to fetch workflows (${response.status})`);
  }

  const payload = await response.json() as Record<string, unknown>;
  const normalized = {
    workflows: Array.isArray(payload?.workflows) ? payload.workflows : [],
  };

  const result = safeParse(WorkflowsListResponseSchema, normalized, 'WorkflowsList');
  if (!result.success) {
    throw new Error(result.error);
  }

  return result.data.workflows;
};

/**
 * Fetch a workflow and return its associated project ID.
 * Uses Zod validation to guard against malformed responses.
 */
export const fetchWorkflowProjectId = async (workflowId: string): Promise<string> => {
  const { API_URL } = await getConfig();
  const response = await fetch(`${API_URL}/workflows/${workflowId}`);

  if (!response.ok) {
    const message = await response.text().catch(() => '');
    throw new Error(message || `Failed to fetch workflow (${response.status})`);
  }

  const payload = await response.json();
  const result = safeParse(WorkflowResponseSchema, payload, 'WorkflowResponse');
  if (!result.success) {
    throw new Error(result.error);
  }

  const workflow = result.data.workflow ?? result.data;
  const projectId = workflow.project_id ?? workflow.projectId;
  if (!projectId) {
    throw new Error('Workflow has no associated project');
  }

  return projectId;
};
