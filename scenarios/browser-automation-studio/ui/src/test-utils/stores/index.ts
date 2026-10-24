import { vi } from 'vitest';

export function createMockProjectStore() {
  return {
    projects: [],
    selectedProject: null,
    isLoading: false,
    error: null,
    fetchProjects: vi.fn(),
    createProject: vi.fn(),
    updateProject: vi.fn(),
    deleteProject: vi.fn(),
    selectProject: vi.fn(),
  };
}

export function createMockWorkflowStore() {
  return {
    workflows: [],
    selectedWorkflow: null,
    isLoading: false,
    error: null,
    fetchWorkflows: vi.fn(),
    createWorkflow: vi.fn(),
    updateWorkflow: vi.fn(),
    deleteWorkflow: vi.fn(),
    selectWorkflow: vi.fn(),
  };
}

export function createMockExecutionStore() {
  return {
    executions: [],
    activeExecution: null,
    isExecuting: false,
    error: null,
    startExecution: vi.fn(),
    stopExecution: vi.fn(),
    fetchExecutions: vi.fn(),
  };
}

export const mockProjectStore = createMockProjectStore();
export const mockWorkflowStore = createMockWorkflowStore();
export const mockExecutionStore = createMockExecutionStore();
