import type { ReactElement, ReactNode } from 'react';
import { render, RenderOptions } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { vi } from 'vitest';

/**
 * Creates a QueryClient instance for testing with disabled retries and caching.
 */
export function createTestQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
        staleTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  });
}

/**
 * Custom render function that wraps components with necessary providers.
 */
export function renderWithProviders(
  ui: ReactElement,
  options?: RenderOptions & { queryClient?: QueryClient }
) {
  const { queryClient, ...renderOptions } = options ?? {};
  const resolvedClient = queryClient ?? createTestQueryClient();

  const Wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={resolvedClient}>{children}</QueryClientProvider>
  );

  return render(ui, {
    wrapper: Wrapper,
    ...renderOptions,
  });
}

/**
 * Mock functions for common zustand stores used in tests.
 */
export const mockProjectStore = {
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

export const mockWorkflowStore = {
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

export const mockExecutionStore = {
  executions: [],
  activeExecution: null,
  isExecuting: false,
  error: null,
  startExecution: vi.fn(),
  stopExecution: vi.fn(),
  fetchExecutions: vi.fn(),
};
