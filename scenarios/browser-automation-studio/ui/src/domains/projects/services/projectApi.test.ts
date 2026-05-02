import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fetchJsonResponse, fetchTextResponse, installFetchMock, type FetchMock } from '@/test-utils';

vi.mock('@/config', () => ({
  getConfig: vi.fn(() => Promise.resolve({ API_URL: 'http://localhost:8080' })),
}));

vi.mock('@/utils/logger', () => ({
  logger: {
    warn: vi.fn(),
  },
}));

import { fetchProjectEntries, fetchProjectWorkflows, fetchProjectsList } from './projectApi';

describe('projectApi', () => {
  let fetchMock: FetchMock;

  beforeEach(() => {
    vi.clearAllMocks();
    fetchMock = installFetchMock();
  });

  it('returns parsed projects', async () => {
    fetchMock.mockResolvedValueOnce(
      fetchJsonResponse({
        projects: [
          {
            id: 'project-1',
            name: 'Project One',
            folder_path: '/project-one',
            created_at: '2024-01-01T00:00:00.000Z',
            updated_at: '2024-01-02T00:00:00.000Z',
          },
        ],
      })
    );

    const projects = await fetchProjectsList();
    expect(projects).toHaveLength(1);
    expect(projects[0]?.id).toBe('project-1');
  });

  it('throws when the response is not ok', async () => {
    fetchMock.mockResolvedValueOnce(fetchTextResponse('boom', { status: 500 }));

    await expect(fetchProjectsList()).rejects.toThrow('boom');
  });

  it('returns validated project workflows', async () => {
    fetchMock.mockResolvedValueOnce(
      fetchJsonResponse({
        workflows: [
          {
            id: 'workflow-1',
            name: 'Demo Workflow',
            folder_path: '/demo',
            created_at: '2025-01-01T00:00:00Z',
            updated_at: '2025-01-02T00:00:00Z',
            stats: {
              execution_count: 2,
              success_rate: 100,
            },
          },
        ],
      })
    );

    const workflows = await fetchProjectWorkflows('project-1');
    expect(workflows).toHaveLength(1);
    expect(workflows[0]?.id).toBe('workflow-1');
  });

  it('returns validated project entries', async () => {
    fetchMock.mockResolvedValueOnce(
      fetchJsonResponse({
        entries: [
          {
            id: 'entry-1',
            project_id: 'project-1',
            path: '/demo',
            kind: 'folder',
          },
        ],
      })
    );

    const entries = await fetchProjectEntries('project-1');
    expect(entries).toHaveLength(1);
    expect(entries[0]?.id).toBe('entry-1');
  });
});
