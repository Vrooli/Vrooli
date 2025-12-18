import { describe, it, expect, beforeEach, vi } from 'vitest';
import { act } from '@testing-library/react';
import { useProjectStore, type Project } from '@/domains/projects';

// Mock fetch globally
global.fetch = vi.fn();

function createFetchResponse<T>(data: T, ok = true, status = ok ? 200 : 400) {
  return Promise.resolve({
    ok,
    status,
    text: () => Promise.resolve(JSON.stringify(data)),
    json: () => Promise.resolve(data),
  } as Response);
}

const ts = (iso: string) => {
  const date = new Date(iso);
  return { seconds: Math.floor(date.getTime() / 1000), nanos: (date.getTime() % 1000) * 1_000_000 };
};

describe('projectStore [REQ:BAS-WORKFLOW-PERSIST-CRUD]', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Reset store state
    useProjectStore.setState({
      projects: [],
      selectedProject: null,
      isLoading: false,
      error: null,
    });
  });

  it('fetches projects successfully', async () => {
    const mockProjects: any[] = [
      {
        project: {
          id: 'project-1',
          name: 'Test Project 1',
          description: 'Description 1',
          folder_path: '/test/path1',
          created_at: ts('2025-01-01T00:00:00Z'),
          updated_at: ts('2025-01-01T00:00:00Z'),
        },
        stats: { workflow_count: 2, execution_count: 3 },
      },
      {
        project: {
          id: 'project-2',
          name: 'Test Project 2',
          description: '',
          folder_path: '/test/path2',
          created_at: ts('2025-01-02T00:00:00Z'),
          updated_at: ts('2025-01-02T00:00:00Z'),
        },
      },
    ];

    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      createFetchResponse({ projects: mockProjects })
    );

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

    const createdProjectProto = {
      id: 'new-project-id',
      ...newProjectData,
      created_at: ts('2025-01-03T00:00:00Z'),
      updated_at: ts('2025-01-03T00:00:00Z'),
    };

    const expectedProject: Project = {
      id: 'new-project-id',
      ...newProjectData,
      created_at: '2025-01-03T00:00:00.000Z',
      updated_at: '2025-01-03T00:00:00.000Z',
    };

    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      createFetchResponse(createdProjectProto)
    );

    let returnedProject: Project | undefined;
    await act(async () => {
      returnedProject = await useProjectStore.getState().createProject(newProjectData);
    });

    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/projects'),
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify(newProjectData),
      })
    );

    expect(returnedProject).toEqual(expectedProject);
    expect(useProjectStore.getState().projects).toContainEqual(expectedProject);
  });

  it('handles project creation errors [REQ:BAS-PROJECT-CREATE-VALIDATION]', async () => {
    const errorMessage = 'Project name already exists';

    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      createFetchResponse({ error: errorMessage }, false)
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

    const updates = {
      name: 'Updated Name',
      description: 'New Description',
      folder_path: '/test/path',
    };

    const updatedProjectProto = {
      ...existingProject,
      ...updates,
      created_at: ts('2025-01-01T00:00:00Z'),
      updated_at: ts('2025-01-04T00:00:00Z'),
    };

    const expectedUpdated: Project = {
      ...existingProject,
      ...updates,
      created_at: '2025-01-01T00:00:00.000Z',
      updated_at: '2025-01-04T00:00:00.000Z',
    };

    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(createFetchResponse(updatedProjectProto));

    await act(async () => {
      await useProjectStore.getState().updateProject('existing-id', updates);
    });

    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/projects/existing-id'),
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify(updates),
      })
    );

    const projectInStore = useProjectStore.getState().projects.find((p) => p.id === 'existing-id');
    expect(projectInStore).toEqual(expectedUpdated);
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

    (global.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      createFetchResponse({ success: true })
    );

    await act(async () => {
      await useProjectStore.getState().deleteProject('delete-id');
    });

    expect(global.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/projects/delete-id'),
      expect.objectContaining({
        method: 'DELETE',
      })
    );

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

    useProjectStore.setState({
      projects,
      selectedProject: projects[0],
    });

    act(() => {
      useProjectStore.getState().selectProject(null);
    });

    expect(useProjectStore.getState().selectedProject).toBeNull();
  });
});
