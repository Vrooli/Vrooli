import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ConnectError, Code } from '@connectrpc/connect';

const listProjectsMock = vi.fn();
const listProjectWorkflowsMock = vi.fn();
const getProjectFileTreeMock = vi.fn();

vi.mock('@/api/projects', () => ({
  projectsClient: {
    listProjects: (...a: unknown[]) => listProjectsMock(...a),
    listProjectWorkflows: (...a: unknown[]) => listProjectWorkflowsMock(...a),
  },
}));

vi.mock('@/api/projectFiles', () => ({
  projectFilesClient: {
    getProjectFileTree: (...a: unknown[]) => getProjectFileTreeMock(...a),
  },
}));

vi.mock('@/utils/logger', () => ({
  logger: { warn: vi.fn() },
}));

import { fetchProjectEntries, fetchProjectWorkflows, fetchProjectsList } from './projectApi';

const ts = (iso: string) => {
  const d = new Date(iso);
  return { seconds: BigInt(Math.floor(d.getTime() / 1000)), nanos: 0 };
};

describe('projectApi', () => {
  beforeEach(() => vi.clearAllMocks());

  it('returns parsed projects', async () => {
    listProjectsMock.mockResolvedValueOnce({
      projects: [
        {
          project: {
            id: 'project-1',
            name: 'Project One',
            folderPath: '/project-one',
            createdAt: ts('2024-01-01T00:00:00.000Z'),
            updatedAt: ts('2024-01-02T00:00:00.000Z'),
          },
        },
      ],
    });

    const projects = await fetchProjectsList();
    expect(projects).toHaveLength(1);
    expect(projects[0]?.id).toBe('project-1');
  });

  it('propagates Connect errors', async () => {
    listProjectsMock.mockRejectedValueOnce(new ConnectError('boom', Code.Internal));
    await expect(fetchProjectsList()).rejects.toThrow(/boom/);
  });

  it('returns project workflows', async () => {
    listProjectWorkflowsMock.mockResolvedValueOnce({
      workflows: [
        {
          id: 'workflow-1',
          name: 'Demo Workflow',
          folderPath: '/demo',
          projectId: 'project-1',
          version: 1,
        },
      ],
    });

    const workflows = await fetchProjectWorkflows('project-1');
    expect(workflows).toHaveLength(1);
    expect(workflows[0]?.id).toBe('workflow-1');
  });

  it('returns project entries via ProjectFilesService', async () => {
    getProjectFileTreeMock.mockResolvedValueOnce({
      entries: [
        {
          id: 'entry-1',
          projectId: 'project-1',
          path: 'a/b.json',
          kind: 2, // WORKFLOW_FILE
          workflowId: 'workflow-1',
          metadata: undefined,
        },
      ],
    });

    const entries = await fetchProjectEntries('project-1');
    expect(entries).toHaveLength(1);
    expect(entries[0]?.id).toBe('entry-1');
    expect(entries[0]?.kind).toBe('workflow_file');
  });
});
