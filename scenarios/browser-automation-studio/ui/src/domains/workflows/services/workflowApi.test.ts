import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('@/config', () => ({
  getConfig: vi.fn(() => Promise.resolve({ API_URL: 'http://localhost:8080' })),
}));

import { fetchWorkflowList, fetchWorkflowProjectId } from './workflowApi';

describe('workflowApi', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    global.fetch = vi.fn();
  });

  it('returns a validated workflow list', async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        workflows: [
          {
            id: 'workflow-1',
            name: 'Test Workflow',
            project_id: 'project-1',
            updated_at: '2024-01-01T00:00:00.000Z',
          },
        ],
      }),
    } as Response);

    const workflows = await fetchWorkflowList(10);
    expect(workflows).toHaveLength(1);
    expect(workflows[0]?.id).toBe('workflow-1');
  });

  it('throws when workflow list validation fails', async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        workflows: [{ id: 123 }],
      }),
    } as Response);

    await expect(fetchWorkflowList()).rejects.toThrow('Invalid WorkflowsList response');
  });

  it('returns the project id for a workflow', async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        workflow: {
          id: 'workflow-1',
          project_id: 'project-1',
        },
      }),
    } as Response);

    await expect(fetchWorkflowProjectId('workflow-1')).resolves.toBe('project-1');
  });

  it('throws when workflow project id is missing', async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        workflow: {
          id: 'workflow-1',
        },
      }),
    } as Response);

    await expect(fetchWorkflowProjectId('workflow-1')).rejects.toThrow('Workflow has no associated project');
  });
});
