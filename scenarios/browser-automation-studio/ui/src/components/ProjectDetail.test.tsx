import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { renderWithProviders } from '@/test-utils/testHelpers';
import type { Project } from '@stores/projectStore';
import ProjectDetail from '@components/ProjectDetail';

const mockDeleteProject = vi.fn();
const mockBulkDeleteWorkflows = vi.fn();
const executionStoreState = {
  loadExecution: vi.fn(),
  startExecution: vi.fn(),
  closeViewer: vi.fn(),
  currentExecution: null,
  viewerWorkflowId: null,
};

vi.mock('@stores/projectStore', () => ({
  __esModule: true,
  useProjectStore: vi.fn((selector?: (state: { deleteProject: typeof mockDeleteProject }) => unknown) => {
    const state = { deleteProject: mockDeleteProject };
    return typeof selector === 'function' ? selector(state) : state;
  }),
}));

vi.mock('@stores/workflowStore', () => ({
  __esModule: true,
  useWorkflowStore: vi.fn((selector?: (state: { bulkDeleteWorkflows: typeof mockBulkDeleteWorkflows }) => unknown) => {
    const state = { bulkDeleteWorkflows: mockBulkDeleteWorkflows };
    return typeof selector === 'function' ? selector(state) : state;
  }),
}));

vi.mock('@stores/executionStore', () => ({
  __esModule: true,
  useExecutionStore: vi.fn((selector?: (state: typeof executionStoreState) => unknown) => {
    return typeof selector === 'function' ? selector(executionStoreState) : executionStoreState;
  }),
}));

const getConfigMock = vi.hoisted(() => vi.fn());
vi.mock('../config', () => ({
  __esModule: true,
  getConfig: getConfigMock,
}));

const toastMock = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
}));
vi.mock('react-hot-toast', () => ({
  __esModule: true,
  default: toastMock,
}));

vi.mock('./ExecutionViewer', () => ({
  __esModule: true,
  default: () => <div data-testid="execution-viewer-mock" />,
}));

vi.mock('./ExecutionHistory', () => ({
  __esModule: true,
  default: () => <div data-testid="execution-history-mock" />,
}));

vi.mock('../utils/logger', () => ({
  __esModule: true,
  logger: {
    info: vi.fn(),
    error: vi.fn(),
    warn: vi.fn(),
    debug: vi.fn(),
  },
}));

describe('ProjectDetail workflow execution [REQ:BAS-EXEC-TELEMETRY-AUTOMATION]', () => {
  const apiBase = 'http://localhost:19770/api/v1';
  const project: Project = {
    id: 'project-demo',
    name: 'Demo Project',
    description: 'Demo description',
    folder_path: '/demo',
    created_at: '2025-01-01T00:00:00Z',
    updated_at: '2025-01-01T00:00:00Z',
  };

  const workflow = {
    id: 'workflow-demo',
    name: 'Telemetry Workflow',
    description: 'Ensures telemetry render',
    folder_path: '/demo',
    created_at: '2025-01-01T00:00:00Z',
    updated_at: '2025-01-02T00:00:00Z',
    stats: {
      execution_count: 3,
      success_rate: 100,
    },
  };

  const originalFetch = global.fetch;
  let fetchMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    vi.clearAllMocks();
    getConfigMock.mockResolvedValue({ API_URL: apiBase });
    executionStoreState.startExecution.mockResolvedValue(undefined);

    fetchMock = vi.fn((input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === 'string' ? input : input.toString();
      if (url === `${apiBase}/projects/${project.id}/workflows`) {
        return Promise.resolve({
          ok: true,
          json: async () => ({ workflows: [workflow] }),
        } as Response);
      }

      if (url === `${apiBase}/workflows/${workflow.id}/execute`) {
        return Promise.resolve({
          ok: true,
          json: async () => ({ execution_id: 'exec-123', status: 'pending' }),
        } as Response);
      }

      return Promise.reject(new Error(`Unhandled fetch call for ${url}`));
    });

    global.fetch = fetchMock as unknown as typeof fetch;
  });

  afterEach(() => {
    global.fetch = originalFetch;
  });

  it('starts executions through the execution store when executing a workflow', async () => {
    const user = userEvent.setup();

    renderWithProviders(
      <ProjectDetail
        project={project}
        onBack={() => {}}
        onWorkflowSelect={async () => {}}
        onCreateWorkflow={() => {}}
      />,
    );

    const executeButtons = await screen.findAllByTestId('workflow-execute-button');
    await user.click(executeButtons[0]);

    await waitFor(() => {
      expect(executionStoreState.startExecution).toHaveBeenCalledWith(workflow.id);
    });
  });
});
