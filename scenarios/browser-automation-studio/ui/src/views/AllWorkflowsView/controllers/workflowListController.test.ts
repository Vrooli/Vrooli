import { describe, expect, it, vi } from 'vitest';

vi.mock('@/domains/projects/services/projectApi', () => ({
  fetchProjectsList: vi.fn(),
}));

vi.mock('@/domains/workflows/services/workflowApi', () => ({
  fetchWorkflowList: vi.fn(),
}));

import { loadGlobalWorkflows } from './workflowListController';
import { fetchProjectsList } from '@/domains/projects/services/projectApi';
import { fetchWorkflowList } from '@/domains/workflows/services/workflowApi';

describe('workflowListController', () => {
  it('maps workflows with project names and defaults', async () => {
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
        folder_path: '/flows',
        execution_count: 5,
      },
    ]);

    const results = await loadGlobalWorkflows(10);
    expect(results).toHaveLength(1);
    expect(results[0]?.projectName).toBe('Project One');
    expect(results[0]?.executionCount).toBe(5);
  });

  it('falls back to unknown project when missing mapping', async () => {
    vi.mocked(fetchProjectsList).mockResolvedValueOnce([]);
    vi.mocked(fetchWorkflowList).mockResolvedValueOnce([
      {
        id: 'workflow-2',
        name: 'Workflow Two',
        project_id: 'missing-project',
        updated_at: '2024-01-03T00:00:00.000Z',
      },
    ]);

    const results = await loadGlobalWorkflows(10);
    expect(results[0]?.projectName).toBe('Unknown Project');
    expect(results[0]?.folderPath).toBe('/');
  });
});
