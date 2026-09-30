/**
 * Projects Domain Store
 *
 * Manages project state including:
 * - Projects list and CRUD operations
 * - Current/selected project
 * - Bulk workflow execution
 * - Last used project tracking
 *
 * All HTTP I/O is delegated to services/projectApi.ts, which talks to
 * ProjectsService Connect-RPC.
 */

import { create } from 'zustand';
import { logger } from '../../utils/logger';
import {
  createProjectViaApi,
  deleteProjectViaApi,
  executeAllProjectWorkflowsViaApi,
  fetchProject as fetchProjectViaApi,
  fetchProjectsList,
  updateProjectViaApi,
} from './services/projectApi';

const PROJECTS_ROOT = 'scenarios/browser-automation-studio/data/projects';
const normalizeFolderSegment = (value: string): string =>
  value.replace(/\\/g, '/').replace(/^\/+/, '').replace(/\/+$/, '');
export const buildProjectFolderPath = (folderName: string): string => {
  const safeName = normalizeFolderSegment(folderName || 'project');
  return `${PROJECTS_ROOT}/${safeName}`;
};

export interface Project {
  id: string;
  name: string;
  description?: string;
  folder_path: string;
  created_at: string;
  updated_at: string;
  stats?: {
    workflow_count: number;
    execution_count: number;
    last_execution?: string;
  };
}

interface BulkExecutionResult {
  message: string;
  executions: Array<{
    workflow_id: string;
    workflow_name: string;
    execution_id?: string;
    status: string;
    error?: string;
  }>;
}

interface ProjectState {
  projects: Project[];
  currentProject: Project | null;
  selectedProject: Project | null; // Alias for currentProject
  isLoading: boolean;
  error: string | null;
  bulkExecutionInProgress: Record<string, boolean>;
  isConnected: boolean;
  lastUsedProjectId: string | null;

  // Actions
  fetchProjects: () => Promise<void>;
  createProject: (
    project: Omit<Project, 'id' | 'created_at' | 'updated_at' | 'stats'> & {
      preset?: string;
      preset_paths?: string[];
    },
  ) => Promise<Project>;
  updateProject: (
    id: string,
    updates: Partial<Pick<Project, 'name' | 'description' | 'folder_path'>>,
  ) => Promise<Project>;
  deleteProject: (id: string, deleteFiles?: boolean) => Promise<void>;
  setCurrentProject: (project: Project | null) => void;
  selectProject: (idOrNull: string | null) => void; // Alias/helper for setCurrentProject
  getProject: (id: string) => Promise<Project | null>;
  executeAllWorkflows: (projectId: string) => Promise<BulkExecutionResult>;
  clearError: () => void;
  getOrCreateDefaultProject: () => Promise<Project>;
  getSmartDefaultProject: () => Project | null;
  setLastUsedProject: (projectId: string) => void;
  deleteAllProjects: () => Promise<{ deleted: number; errors: string[] }>;
}

// Load last used project from localStorage
const loadLastUsedProjectId = (): string | null => {
  try {
    if (typeof window === 'undefined') return null;
    return window.localStorage.getItem('browserAutomation.lastUsedProjectId');
  } catch {
    return null;
  }
};

const saveLastUsedProjectId = (projectId: string): void => {
  try {
    if (typeof window === 'undefined') return;
    window.localStorage.setItem('browserAutomation.lastUsedProjectId', projectId);
  } catch {
    // Ignore storage errors
  }
};

export const useProjectStore = create<ProjectState>((set, get) => ({
  projects: [],
  currentProject: null,
  selectedProject: null, // Will be kept in sync with currentProject
  isLoading: false,
  error: null,
  bulkExecutionInProgress: {},
  isConnected: true,
  lastUsedProjectId: loadLastUsedProjectId(),

  fetchProjects: async () => {
    set({ isLoading: true, error: null });
    try {
      const parsed = await fetchProjectsList();
      set({ projects: parsed, isLoading: false, isConnected: true });
    } catch (error) {
      logger.error(
        'Failed to fetch projects',
        { component: 'ProjectStore', action: 'fetchProjects' },
        error,
      );
      set({
        error: error instanceof Error ? error.message : 'Failed to fetch projects',
        isLoading: false,
        isConnected: false,
      });
    }
  },

  createProject: async (projectData) => {
    set({ isLoading: true, error: null });
    try {
      const newProject = await createProjectViaApi({
        name: projectData.name,
        description: projectData.description ?? '',
        folder_path: projectData.folder_path,
        preset: projectData.preset,
        preset_paths: projectData.preset_paths,
      });
      set((state) => ({
        projects: [newProject, ...state.projects],
        isLoading: false,
      }));
      return newProject;
    } catch (error) {
      logger.error(
        'Failed to create project',
        { component: 'ProjectStore', action: 'createProject' },
        error,
      );
      set({
        error: error instanceof Error ? error.message : 'Failed to create project',
        isLoading: false,
      });
      throw error;
    }
  },

  updateProject: async (id, updates) => {
    set({ isLoading: true, error: null });
    try {
      const updatedProject = await updateProjectViaApi(id, updates);
      set((state) => ({
        projects: state.projects.map((p) => (p.id === id ? updatedProject : p)),
        currentProject: state.currentProject?.id === id ? updatedProject : state.currentProject,
        selectedProject: state.selectedProject?.id === id ? updatedProject : state.selectedProject,
        isLoading: false,
      }));
      return updatedProject;
    } catch (error) {
      logger.error(
        'Failed to update project',
        { component: 'ProjectStore', action: 'updateProject' },
        error,
      );
      set({
        error: error instanceof Error ? error.message : 'Failed to update project',
        isLoading: false,
      });
      throw error;
    }
  },

  deleteProject: async (id, deleteFiles = false) => {
    set({ isLoading: true, error: null });
    try {
      await deleteProjectViaApi(id, deleteFiles);
      set((state) => ({
        projects: state.projects.filter((p) => p.id !== id),
        currentProject: state.currentProject?.id === id ? null : state.currentProject,
        selectedProject: state.selectedProject?.id === id ? null : state.selectedProject,
        isLoading: false,
      }));
    } catch (error) {
      logger.error(
        'Failed to delete project',
        { component: 'ProjectStore', action: 'deleteProject', deleteFiles },
        error,
      );
      set({
        error: error instanceof Error ? error.message : 'Failed to delete project',
        isLoading: false,
      });
      throw error;
    }
  },

  getProject: async (id) => {
    try {
      return await fetchProjectViaApi(id);
    } catch (error) {
      logger.error(
        'Failed to fetch project',
        { component: 'ProjectStore', action: 'getProject', projectId: id },
        error,
      );
      return null;
    }
  },

  setCurrentProject: (project) => {
    set({ currentProject: project, selectedProject: project });
  },

  selectProject: (idOrNull) => {
    if (idOrNull === null) {
      set({ currentProject: null, selectedProject: null });
      return;
    }
    const project = get().projects.find((p) => p.id === idOrNull);
    if (project) {
      set({ currentProject: project, selectedProject: project });
    }
  },

  executeAllWorkflows: async (projectId: string) => {
    set((state) => ({
      bulkExecutionInProgress: { ...state.bulkExecutionInProgress, [projectId]: true },
      error: null,
    }));
    try {
      const result = await executeAllProjectWorkflowsViaApi(projectId);
      set((state) => ({
        bulkExecutionInProgress: { ...state.bulkExecutionInProgress, [projectId]: false },
      }));
      return result;
    } catch (error) {
      logger.error(
        'Failed to execute all workflows',
        { component: 'ProjectStore', action: 'executeAllWorkflows', projectId },
        error,
      );
      set((state) => ({
        bulkExecutionInProgress: { ...state.bulkExecutionInProgress, [projectId]: false },
        error: error instanceof Error ? error.message : 'Failed to execute workflows',
      }));
      throw error;
    }
  },

  clearError: () => {
    set({ error: null, isConnected: true });
  },

  getOrCreateDefaultProject: async () => {
    const { projects, createProject, fetchProjects } = get();
    if (projects.length === 0) {
      await fetchProjects();
    }
    const currentProjects = get().projects;
    if (currentProjects.length > 0) {
      const defaultProject = get().getSmartDefaultProject();
      if (defaultProject) {
        return defaultProject;
      }
    }
    const newProject = await createProject({
      name: 'My Automations',
      description: 'Default project for automation workflows',
      folder_path: buildProjectFolderPath('my-automations'),
    });
    get().setLastUsedProject(newProject.id);
    return newProject;
  },

  getSmartDefaultProject: () => {
    const { projects, lastUsedProjectId } = get();
    if (projects.length === 0) return null;
    if (projects.length === 1) return projects[0] ?? null;
    if (lastUsedProjectId) {
      const lastUsed = projects.find((p) => p.id === lastUsedProjectId);
      if (lastUsed) return lastUsed;
    }
    const sortedByUpdate = [...projects].sort(
      (a, b) => new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime(),
    );
    return sortedByUpdate[0] ?? null;
  },

  setLastUsedProject: (projectId: string) => {
    saveLastUsedProjectId(projectId);
    set({ lastUsedProjectId: projectId });
  },

  deleteAllProjects: async () => {
    const { projects } = get();
    const errors: string[] = [];
    let deleted = 0;

    set({ isLoading: true, error: null });

    for (const project of projects) {
      try {
        await deleteProjectViaApi(project.id, false);
        deleted++;
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Unknown error';
        errors.push(`Failed to delete "${project.name}": ${message}`);
        logger.error(
          'Failed to delete project during bulk delete',
          { component: 'ProjectStore', action: 'deleteAllProjects', projectId: project.id },
          error,
        );
      }
    }

    set({
      projects: [],
      currentProject: null,
      selectedProject: null,
      isLoading: false,
      lastUsedProjectId: null,
    });

    try {
      if (typeof window !== 'undefined') {
        window.localStorage.removeItem('browserAutomation.lastUsedProjectId');
      }
    } catch {
      // Ignore storage errors
    }

    return { deleted, errors };
  },
}));

// Export types for consumers
export type { ProjectState };
