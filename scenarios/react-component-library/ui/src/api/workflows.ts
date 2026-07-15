import { buildApiUrl } from "@vrooli/api-base";

import { API_BASE, decodeApiError } from "./client";

export interface Workflow {
  id: string;
  assetId: string;
  sourceScenario: string;
  targetScenario: string;
  sourcePath: string;
  status: number;
  summary: string;
  error: string;
  canStop: boolean;
  canRetry: boolean;
}

function normalizeWorkflowsResponse(value: unknown): { workflows: Workflow[] } {
  if (!value || typeof value !== "object") return { workflows: [] };

  const { workflows } = value as { workflows?: unknown };
  return { workflows: Array.isArray(workflows) ? workflows as Workflow[] : [] };
}

async function request<T>(procedure: string, input: object): Promise<T> {
  const response = await fetch(buildApiUrl(`/vrooli.react_component_library.v1.workflows.WorkflowsService/${procedure}`, { baseUrl: API_BASE }), {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  if (!response.ok) throw await decodeApiError(response);
  return await response.json() as T;
}

export const workflowsClient = {
  startWorkflow: (input: { kind: 1 | 2; sourceScenario?: string; targetScenario?: string; sourcePath?: string; assetId?: string; idempotencyKey: string }) => request<{ workflow: Workflow }>("StartWorkflow", input),
  listWorkflows: async (input: { activeOnly?: boolean; limit?: number }) => normalizeWorkflowsResponse(await request<unknown>("ListWorkflows", input)),
  stopWorkflow: (input: { id: string }) => request<{ workflow: Workflow }>("StopWorkflow", input),
  retryWorkflow: (input: { id: string; idempotencyKey: string }) => request<{ workflow: Workflow }>("RetryWorkflow", input),
};
