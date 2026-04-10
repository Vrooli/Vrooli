import { describe, expect, it, vi } from 'vitest';

vi.mock('@/domains/projects/services/projectApi', () => ({
  fetchProjectsList: vi.fn(),
}));

vi.mock('@/domains/workflows/services/workflowApi', () => ({
  fetchWorkflowList: vi.fn(),
}));

vi.mock('@/domains/executions/services/executionApi', () => ({
  fetchExecutionsList: vi.fn(),
}));

import { loadGlobalExecutions } from './executionListController';
import { fetchProjectsList } from '@/domains/projects/services/projectApi';
import { fetchWorkflowList } from '@/domains/workflows/services/workflowApi';
import { fetchExecutionsList } from '@/domains/executions/services/executionApi';

describe('executionListController', () => {
  it('normalizes execution status and maps workflow names', async () => {
    vi.mocked(fetchProjectsList).mockResolvedValueOnce([
      {
        id: 'project-1',
        name: 'Project One',
        folder_path: '/project-one',
        created_at: '2024-01-01T00:00:00.000Z',
        updated_at: '2024-01-02T00:00:00.000Z',
      },
    ]);

    vi.mocked(fetchWorkflowList).mockResolvedValueOnce([
      {
        id: 'workflow-1',
        name: 'Workflow One',
        project_id: 'project-1',
        updated_at: '2024-01-03T00:00:00.000Z',
      },
    ]);

    vi.mocked(fetchExecutionsList).mockResolvedValueOnce([
      {
        execution_id: 'exec-1',
        workflow_id: 'workflow-1',
        status: 'EXECUTION_STATUS_RUNNING',
        started_at: '2024-01-04T00:00:00.000Z',
      },
      {
        execution_id: 'exec-2',
        workflow_id: 'workflow-1',
        status: 'completed',
        started_at: '2024-01-05T00:00:00.000Z',
        completed_at: '2024-01-05T00:10:00.000Z',
      },
    ]);

    const results = await loadGlobalExecutions(20);
    expect(results).toHaveLength(2);
    expect(results[0]?.status).toBe('completed');
    expect(results[0]?.workflowName).toBe('Workflow One');
    expect(results[1]?.status).toBe('running');
  });

  it('omits invalid completed_at timestamps', async () => {
    vi.mocked(fetchProjectsList).mockResolvedValueOnce([]);
    vi.mocked(fetchWorkflowList).mockResolvedValueOnce([
      {
        id: 'workflow-1',
        name: 'Workflow One',
        project_id: 'project-1',
        updated_at: '2024-01-03T00:00:00.000Z',
      },
    ]);
    vi.mocked(fetchExecutionsList).mockResolvedValueOnce([
      {
        execution_id: 'exec-3',
        workflow_id: 'workflow-1',
        status: 'completed',
        started_at: '2024-01-04T00:00:00.000Z',
        completed_at: 'not-a-date',
      },
    ]);

    const results = await loadGlobalExecutions(5);
    expect(results[0]?.completedAt).toBeUndefined();
  });
});
