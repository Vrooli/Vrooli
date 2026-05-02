import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fetchJsonResponse, installFetchMock, type FetchMock } from '@/test-utils';

vi.mock('@/config', () => ({
  getConfig: vi.fn(() => Promise.resolve({ API_URL: 'http://localhost:8080' })),
}));

import { fetchWorkflowList, fetchWorkflowProjectId } from './workflowApi';

describe('workflowApi', () => {
  let fetchMock: FetchMock;

  beforeEach(() => {
    vi.clearAllMocks();
    fetchMock = installFetchMock();
  });

  it('returns a validated workflow list', async () => {
    fetchMock.mockResolvedValueOnce(
      fetchJsonResponse({
        workflows: [
          {
            id: 'workflow-1',
            name: 'Test Workflow',
            project_id: 'project-1',
            updated_at: '2024-01-01T00:00:00.000Z',
          },
        ],
      })
    );

    const workflows = await fetchWorkflowList(10);
    expect(workflows).toHaveLength(1);
    expect(workflows[0]?.id).toBe('workflow-1');
  });

  it('throws when workflow list validation fails', async () => {
    fetchMock.mockResolvedValueOnce(
      fetchJsonResponse({
        workflows: [{ id: 123 }],
      })
    );

    await expect(fetchWorkflowList()).rejects.toThrow('Invalid WorkflowsList response');
  });

  it('returns the project id for a workflow', async () => {
    fetchMock.mockResolvedValueOnce(
      fetchJsonResponse({
        workflow: {
          id: 'workflow-1',
          project_id: 'project-1',
        },
      })
    );

    await expect(fetchWorkflowProjectId('workflow-1')).resolves.toBe('project-1');
  });

  it('throws when workflow project id is missing', async () => {
    fetchMock.mockResolvedValueOnce(
      fetchJsonResponse({
        workflow: {
          id: 'workflow-1',
        },
      })
    );

    await expect(fetchWorkflowProjectId('workflow-1')).rejects.toThrow('Workflow has no associated project');
  });
});
