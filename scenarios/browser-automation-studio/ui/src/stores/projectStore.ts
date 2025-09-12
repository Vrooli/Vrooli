import { create } from 'zustand';
import { API_BASE } from '../config';

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
  isLoading: boolean;
  error: string | null;
  bulkExecutionInProgress: Record<string, boolean>;
  
  // Actions
  fetchProjects: () => Promise<void>;
  createProject: (project: Omit<Project, 'id' | 'created_at' | 'updated_at' | 'stats'>) => Promise<Project>;
  updateProject: (id: string, updates: Partial<Pick<Project, 'name' | 'description' | 'folder_path'>>) => Promise<void>;
  deleteProject: (id: string) => Promise<void>;
  setCurrentProject: (project: Project | null) => void;
  getProject: (id: string) => Promise<Project | null>;
  executeAllWorkflows: (projectId: string) => Promise<BulkExecutionResult>;
  clearError: () => void;
}

export const useProjectStore = create<ProjectState>((set) => ({
  projects: [],
  currentProject: null,
  isLoading: false,
  error: null,
  bulkExecutionInProgress: {},

  fetchProjects: async () => {
    set({ isLoading: true, error: null });
    try {
      const response = await fetch(`${API_BASE}/projects`);
      if (!response.ok) {
        throw new Error(`Failed to fetch projects: ${response.status}`);
      }
      const data = await response.json();
      set({ projects: data.projects || [], isLoading: false });
    } catch (error) {
      console.error('Failed to fetch projects:', error);
      set({ 
        error: error instanceof Error ? error.message : 'Failed to fetch projects',
        isLoading: false 
      });
    }
  },

  createProject: async (projectData) => {
    set({ isLoading: true, error: null });
    try {
      const response = await fetch(`${API_BASE}/projects`, {
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
      console.error('Failed to create project:', error);
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
      const response = await fetch(`${API_BASE}/projects/${id}`, {
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
      const updatedProject = data.project;

      set(state => ({
        projects: state.projects.map(p => p.id === id ? updatedProject : p),
        currentProject: state.currentProject?.id === id ? updatedProject : state.currentProject,
        isLoading: false
      }));
    } catch (error) {
      console.error('Failed to update project:', error);
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
      const response = await fetch(`${API_BASE}/projects/${id}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        const errorData = await response.text();
        throw new Error(errorData || `Failed to delete project: ${response.status}`);
      }

      set(state => ({
        projects: state.projects.filter(p => p.id !== id),
        currentProject: state.currentProject?.id === id ? null : state.currentProject,
        isLoading: false
      }));
    } catch (error) {
      console.error('Failed to delete project:', error);
      set({ 
        error: error instanceof Error ? error.message : 'Failed to delete project',
        isLoading: false 
      });
      throw error;
    }
  },

  getProject: async (id) => {
    try {
      const response = await fetch(`${API_BASE}/projects/${id}`);
      if (!response.ok) {
        throw new Error(`Failed to fetch project: ${response.status}`);
      }
      const project = await response.json();
      return project;
    } catch (error) {
      console.error('Failed to fetch project:', error);
      return null;
    }
  },

  setCurrentProject: (project) => {
    set({ currentProject: project });
  },

  executeAllWorkflows: async (projectId: string) => {
    set(state => ({ 
      bulkExecutionInProgress: { ...state.bulkExecutionInProgress, [projectId]: true },
      error: null 
    }));
    
    try {
      const response = await fetch(`${API_BASE}/projects/${projectId}/execute-all`, {
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
      console.error('Failed to execute all workflows:', error);
      set(state => ({
        bulkExecutionInProgress: { ...state.bulkExecutionInProgress, [projectId]: false },
        error: error instanceof Error ? error.message : 'Failed to execute workflows'
      }));
      throw error;
    }
  },

  clearError: () => {
    set({ error: null });
  },
}));