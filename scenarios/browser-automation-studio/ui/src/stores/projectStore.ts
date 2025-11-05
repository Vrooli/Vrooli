import { create } from 'zustand';
import { getConfig } from '../config';
import { logger } from '../utils/logger';

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

  // Actions
  fetchProjects: () => Promise<void>;
  createProject: (project: Omit<Project, 'id' | 'created_at' | 'updated_at' | 'stats'>) => Promise<Project>;
  updateProject: (id: string, updates: Partial<Pick<Project, 'name' | 'description' | 'folder_path'>>) => Promise<Project>;
  deleteProject: (id: string) => Promise<void>;
  setCurrentProject: (project: Project | null) => void;
  selectProject: (idOrNull: string | null) => void; // Alias/helper for setCurrentProject
  getProject: (id: string) => Promise<Project | null>;
  executeAllWorkflows: (projectId: string) => Promise<BulkExecutionResult>;
  clearError: () => void;
}

export const useProjectStore = create<ProjectState>((set, get) => ({
  projects: [],
  currentProject: null,
  selectedProject: null, // Will be kept in sync with currentProject
  isLoading: false,
  error: null,
  bulkExecutionInProgress: {},
  isConnected: true,

  fetchProjects: async () => {
    set({ isLoading: true, error: null });
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/projects`);
      if (!response.ok) {
        throw new Error(`Failed to fetch projects: ${response.status}`);
      }
      const data = await response.json();
      set({ projects: data.projects || [], isLoading: false, isConnected: true });
    } catch (error) {
      logger.error('Failed to fetch projects', { component: 'ProjectStore', action: 'fetchProjects' }, error);
      set({
        error: error instanceof Error ? error.message : 'Failed to fetch projects',
        isLoading: false,
        isConnected: false
      });
    }
  },

  createProject: async (projectData) => {
    set({ isLoading: true, error: null });
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/projects`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(projectData),
      });

      if (!response.ok) {
        const errorData = await response.text();
        throw new Error(errorData || `Failed to create project: ${response.status}`);
      }

      const data = await response.json();
      const newProject = data.project;

      set(state => ({
        projects: [newProject, ...state.projects],
        isLoading: false
      }));

      return newProject;
    } catch (error) {
      logger.error('Failed to create project', { component: 'ProjectStore', action: 'createProject' }, error);
      set({
        error: error instanceof Error ? error.message : 'Failed to create project',
        isLoading: false
      });
      throw error;
    }
  },

  updateProject: async (id, updates) => {
    set({ isLoading: true, error: null });
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/projects/${id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(updates),
      });

      if (!response.ok) {
        const errorData = await response.text();
        throw new Error(errorData || `Failed to update project: ${response.status}`);
      }

      const data = await response.json();
      const updatedProject = data.project as Project;

      set(state => ({
        projects: state.projects.map(p => p.id === id ? updatedProject : p),
        currentProject: state.currentProject?.id === id ? updatedProject : state.currentProject,
        selectedProject: state.selectedProject?.id === id ? updatedProject : state.selectedProject,
        isLoading: false
      }));

      return updatedProject;
    } catch (error) {
      logger.error('Failed to update project', { component: 'ProjectStore', action: 'updateProject' }, error);
      set({
        error: error instanceof Error ? error.message : 'Failed to update project',
        isLoading: false
      });
      throw error;
    }
  },

  deleteProject: async (id) => {
    set({ isLoading: true, error: null });
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/projects/${id}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        const errorData = await response.text();
        throw new Error(errorData || `Failed to delete project: ${response.status}`);
      }

      set(state => ({
        projects: state.projects.filter(p => p.id !== id),
        currentProject: state.currentProject?.id === id ? null : state.currentProject,
        selectedProject: state.selectedProject?.id === id ? null : state.selectedProject,
        isLoading: false
      }));
    } catch (error) {
      logger.error('Failed to delete project', { component: 'ProjectStore', action: 'deleteProject' }, error);
      set({
        error: error instanceof Error ? error.message : 'Failed to delete project',
        isLoading: false
      });
      throw error;
    }
  },

  getProject: async (id) => {
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/projects/${id}`);
      if (!response.ok) {
        throw new Error(`Failed to fetch project: ${response.status}`);
      }
      const project = await response.json();
      return project;
    } catch (error) {
      logger.error('Failed to fetch project', { component: 'ProjectStore', action: 'getProject', projectId: id }, error);
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

    const project = get().projects.find(p => p.id === idOrNull);
    if (project) {
      set({ currentProject: project, selectedProject: project });
    }
  },

  executeAllWorkflows: async (projectId: string) => {
    set(state => ({ 
      bulkExecutionInProgress: { ...state.bulkExecutionInProgress, [projectId]: true },
      error: null 
    }));
    
    try {
      const config = await getConfig();
      const response = await fetch(`${config.API_URL}/projects/${projectId}/execute-all`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        const errorData = await response.text();
        throw new Error(errorData || `Failed to execute workflows: ${response.status}`);
      }

      const result = await response.json();

      set(state => ({
        bulkExecutionInProgress: { ...state.bulkExecutionInProgress, [projectId]: false }
      }));

      return result;
    } catch (error) {
      logger.error('Failed to execute all workflows', { component: 'ProjectStore', action: 'executeAllWorkflows', projectId }, error);
      set(state => ({
        bulkExecutionInProgress: { ...state.bulkExecutionInProgress, [projectId]: false },
        error: error instanceof Error ? error.message : 'Failed to execute workflows'
      }));
      throw error;
    }
  },

  clearError: () => {
    set({ error: null, isConnected: true });
  },
}));
