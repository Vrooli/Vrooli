/**
 * projectStore Test Suite (Connect-RPC)
 *
 * Tests project store functionality after migration to ProjectsService
 * Connect-RPC client. Mocks the generated client; verifies that the store
 * adapts proto responses into its snake_case consumer shape and routes
 * actions to the right RPC.
 */

import { describe, it, expect, beforeEach, vi } from 'vitest';
import { act } from '@testing-library/react';
import { ConnectError, Code } from '@connectrpc/connect';

// ----------------------------------------------------------------------------
// Client mock — installed BEFORE importing the store.
// ----------------------------------------------------------------------------

const listProjectsMock = vi.fn();
const createProjectMock = vi.fn();
const getProjectMock = vi.fn();
const updateProjectMock = vi.fn();
const deleteProjectMock = vi.fn();
const listProjectWorkflowsMock = vi.fn();
const bulkDeleteProjectWorkflowsMock = vi.fn();
const executeAllProjectWorkflowsMock = vi.fn();

vi.mock('../api/projects', () => ({
  projectsClient: {
    listProjects: (...a: unknown[]) => listProjectsMock(...a),
    createProject: (...a: unknown[]) => createProjectMock(...a),
    getProject: (...a: unknown[]) => getProjectMock(...a),
    updateProject: (...a: unknown[]) => updateProjectMock(...a),
    deleteProject: (...a: unknown[]) => deleteProjectMock(...a),
    listProjectWorkflows: (...a: unknown[]) => listProjectWorkflowsMock(...a),
    bulkDeleteProjectWorkflows: (...a: unknown[]) => bulkDeleteProjectWorkflowsMock(...a),
    executeAllProjectWorkflows: (...a: unknown[]) => executeAllProjectWorkflowsMock(...a),
  },
}));

import { useProjectStore, type Project } from '@/domains/projects';

const ts = (iso: string) => {
  const date = new Date(iso);
  return { seconds: BigInt(Math.floor(date.getTime() / 1000)), nanos: (date.getTime() % 1000) * 1_000_000 };
};

describe('projectStore [REQ:BAS-WORKFLOW-PERSIST-CRUD]', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    useProjectStore.setState({
      projects: [],
      selectedProject: null,
      currentProject: null,
      isLoading: false,
      error: null,
    });
  });

  it('fetches projects successfully', async () => {
    listProjectsMock.mockResolvedValueOnce({
      projects: [
        {
          project: {
            id: 'project-1',
            name: 'Test Project 1',
            description: 'Description 1',
            folderPath: '/test/path1',
            createdAt: ts('2025-01-01T00:00:00Z'),
            updatedAt: ts('2025-01-01T00:00:00Z'),
          },
          stats: { workflowCount: 2, executionCount: 3 },
        },
        {
          project: {
            id: 'project-2',
            name: 'Test Project 2',
            description: '',
            folderPath: '/test/path2',
            createdAt: ts('2025-01-02T00:00:00Z'),
            updatedAt: ts('2025-01-02T00:00:00Z'),
          },
        },
      ],
    });

    await act(async () => {
      await useProjectStore.getState().fetchProjects();
    });

    const state = useProjectStore.getState();
    expect(state.projects).toHaveLength(2);
    expect(state.projects[0].name).toBe('Test Project 1');
    expect(state.isLoading).toBe(false);
    expect(state.error).toBeNull();
  });

  it('creates a new project successfully [REQ:BAS-PROJECT-CREATE-SUCCESS]', async () => {
    const newProjectData = {
      name: 'New Project',
      description: 'Test Description',
      folder_path: '/test/new',
    };

    createProjectMock.mockResolvedValueOnce({
      project: {
        id: 'new-project-id',
        name: 'New Project',
        description: 'Test Description',
        folderPath: '/test/new',
        createdAt: ts('2025-01-03T00:00:00Z'),
        updatedAt: ts('2025-01-03T00:00:00Z'),
      },
    });

    let returnedProject: Project | undefined;
    await act(async () => {
      returnedProject = await useProjectStore.getState().createProject(newProjectData);
    });

    expect(createProjectMock).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'New Project',
        folderPath: '/test/new',
      }),
    );
    expect(returnedProject?.id).toBe('new-project-id');
    expect(useProjectStore.getState().projects.map((p) => p.id)).toContain('new-project-id');
  });

  it('handles project creation errors [REQ:BAS-PROJECT-CREATE-VALIDATION]', async () => {
    createProjectMock.mockRejectedValueOnce(
      new ConnectError('Project name already exists', Code.AlreadyExists),
    );

    await expect(async () => {
      await act(async () => {
        await useProjectStore.getState().createProject({
          name: 'Duplicate Name',
          folder_path: '/test/path',
        });
      });
    }).rejects.toThrow();

    expect(useProjectStore.getState().error).toBeTruthy();
  });

  it('updates an existing project successfully', async () => {
    const existingProject: Project = {
      id: 'existing-id',
      name: 'Original Name',
      description: '',
      folder_path: '/test/path',
      created_at: '2025-01-01',
      updated_at: '2025-01-01',
    };
    useProjectStore.setState({ projects: [existingProject] });

    updateProjectMock.mockResolvedValueOnce({
      project: {
        id: 'existing-id',
        name: 'Updated Name',
        description: 'New Description',
        folderPath: '/test/path',
        createdAt: ts('2025-01-01T00:00:00Z'),
        updatedAt: ts('2025-01-04T00:00:00Z'),
      },
    });

    await act(async () => {
      await useProjectStore.getState().updateProject('existing-id', {
        name: 'Updated Name',
        description: 'New Description',
        folder_path: '/test/path',
      });
    });

    expect(updateProjectMock).toHaveBeenCalledWith(
      expect.objectContaining({ id: 'existing-id', name: 'Updated Name' }),
    );
    const projectInStore = useProjectStore.getState().projects.find((p) => p.id === 'existing-id');
    expect(projectInStore?.name).toBe('Updated Name');
  });

  it('deletes a project successfully', async () => {
    const projectToDelete: Project = {
      id: 'delete-id',
      name: 'To Be Deleted',
      description: '',
      folder_path: '/test/delete',
      created_at: '2025-01-01',
      updated_at: '2025-01-01',
    };
    useProjectStore.setState({ projects: [projectToDelete] });

    deleteProjectMock.mockResolvedValueOnce({ filesDeleted: false });

    await act(async () => {
      await useProjectStore.getState().deleteProject('delete-id');
    });

    expect(deleteProjectMock).toHaveBeenCalledWith({ id: 'delete-id', deleteFiles: false });
    expect(useProjectStore.getState().projects).toHaveLength(0);
  });

  it('selects a project', () => {
    const projects: Project[] = [
      {
        id: 'project-1',
        name: 'Project 1',
        description: '',
        folder_path: '/test/path1',
        created_at: '2025-01-01',
        updated_at: '2025-01-01',
      },
    ];

    useProjectStore.setState({ projects });

    act(() => {
      useProjectStore.getState().selectProject('project-1');
    });

    expect(useProjectStore.getState().selectedProject?.id).toBe('project-1');
  });

  it('clears selection when selecting null', () => {
    const projects: Project[] = [
      {
        id: 'project-1',
        name: 'Project 1',
        description: '',
        folder_path: '/test/path1',
        created_at: '2025-01-01',
        updated_at: '2025-01-01',
      },
    ];

    useProjectStore.setState({ projects, selectedProject: projects[0] });

    act(() => {
      useProjectStore.getState().selectProject(null);
    });

    expect(useProjectStore.getState().selectedProject).toBeNull();
  });
});
