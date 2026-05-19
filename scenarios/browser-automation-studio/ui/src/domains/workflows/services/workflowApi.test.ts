import { beforeEach, describe, expect, it, vi } from 'vitest';
import { create } from '@bufbuild/protobuf';
import {
  GetWorkflowResponseSchema,
  ListWorkflowsResponseSchema,
  WorkflowSummarySchema,
} from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';

const listWorkflowsMock = vi.fn();
const getWorkflowMock = vi.fn();

vi.mock('@/api/workflows', () => ({
  workflowsClient: {
    listWorkflows: (...a: unknown[]) => listWorkflowsMock(...a),
    getWorkflow: (...a: unknown[]) => getWorkflowMock(...a),
  },
}));

import { fetchWorkflowList, fetchWorkflowProjectId } from './workflowApi';

describe('workflowApi', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('returns a workflow list mapped to legacy WorkflowItem shape', async () => {
    listWorkflowsMock.mockResolvedValueOnce(
      create(ListWorkflowsResponseSchema, {
        workflows: [
          create(WorkflowSummarySchema, {
            id: 'workflow-1',
            name: 'Test Workflow',
            projectId: 'project-1',
          }),
        ],
      }),
    );

    const workflows = await fetchWorkflowList(10);
    expect(workflows).toHaveLength(1);
    expect(workflows[0]?.id).toBe('workflow-1');
    expect(workflows[0]?.project_id).toBe('project-1');
    expect(listWorkflowsMock).toHaveBeenCalledWith({ limit: 10 });
  });

  it('returns the project id for a workflow', async () => {
    getWorkflowMock.mockResolvedValueOnce(
      create(GetWorkflowResponseSchema, {
        workflow: create(WorkflowSummarySchema, {
          id: 'workflow-1',
          projectId: 'project-1',
        }),
      }),
    );

    await expect(fetchWorkflowProjectId('workflow-1')).resolves.toBe('project-1');
  });

  it('throws when workflow project id is missing', async () => {
    getWorkflowMock.mockResolvedValueOnce(
      create(GetWorkflowResponseSchema, {
        workflow: create(WorkflowSummarySchema, { id: 'workflow-1' }),
      }),
    );

    await expect(fetchWorkflowProjectId('workflow-1')).rejects.toThrow('Workflow has no associated project');
  });
});
