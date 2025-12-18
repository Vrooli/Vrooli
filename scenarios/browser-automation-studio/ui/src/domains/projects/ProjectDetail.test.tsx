import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithProviders } from "@/test-utils/testHelpers";
import type { Project } from "./store";
import ProjectDetail from "./ProjectDetail";
import { selectors } from "@constants/selectors";

const mockDeleteProject = vi.fn();
const mockBulkDeleteWorkflows = vi.fn();
const executionStoreState = {
  loadExecution: vi.fn(),
  loadExecutions: vi.fn(),
  startExecution: vi.fn(),
  closeViewer: vi.fn(),
  currentExecution: null,
  viewerWorkflowId: null,
  executions: [],
};

vi.mock("./store", () => ({
  __esModule: true,
  useProjectStore: vi.fn(
    (
      selector?: (state: {
        deleteProject: typeof mockDeleteProject;
      }) => unknown,
    ) => {
      const state = { deleteProject: mockDeleteProject };
      return typeof selector === "function" ? selector(state) : state;
    },
  ),
}));

vi.mock("@stores/workflowStore", () => ({
  __esModule: true,
  useWorkflowStore: vi.fn(
    (
      selector?: (state: {
        bulkDeleteWorkflows: typeof mockBulkDeleteWorkflows;
      }) => unknown,
    ) => {
      const state = { bulkDeleteWorkflows: mockBulkDeleteWorkflows };
      return typeof selector === "function" ? selector(state) : state;
    },
  ),
}));

vi.mock("@/domains/executions", () => {
  const mockUseExecutionStore = vi.fn(
    (selector?: (state: typeof executionStoreState) => unknown) => {
      return typeof selector === "function"
        ? selector(executionStoreState)
        : executionStoreState;
    },
  );
  // Add getState method for components that access state outside React hooks
  mockUseExecutionStore.getState = () => executionStoreState;
  return {
    __esModule: true,
    useExecutionStore: mockUseExecutionStore,
    ExecutionHistory: () => null,
    ExecutionViewer: () => null,
  };
});

const getConfigMock = vi.hoisted(() => vi.fn());
vi.mock("@/config", () => ({
  __esModule: true,
  getConfig: getConfigMock,
}));

const toastMock = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
}));
vi.mock("react-hot-toast", () => ({
  __esModule: true,
  default: toastMock,
}));

vi.mock("./ExecutionViewer", () => ({
  __esModule: true,
  default: () => <div data-testid={selectors.executions.mock.viewer} />,
}));

vi.mock("./ExecutionHistory", () => ({
  __esModule: true,
  default: () => <div data-testid={selectors.executions.mock.history} />,
}));

vi.mock("@utils/logger", () => ({
  __esModule: true,
  logger: {
    info: vi.fn(),
    error: vi.fn(),
    warn: vi.fn(),
    debug: vi.fn(),
  },
}));

describe("ProjectDetail workflow execution [REQ:BAS-EXEC-TELEMETRY-AUTOMATION]", () => {
  const apiBase = "http://localhost:19770/api/v1";
  const project: Project = {
    id: "project-demo",
    name: "Demo Project",
    description: "Demo description",
    folder_path: "/demo",
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-01T00:00:00Z",
  };

  const workflow = {
    id: "workflow-demo",
    name: "Telemetry Workflow",
    description: "Ensures telemetry render",
    folder_path: "/demo",
    created_at: "2025-01-01T00:00:00Z",
    updated_at: "2025-01-02T00:00:00Z",
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

    fetchMock = vi.fn((input: RequestInfo | URL, _init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      if (url === `${apiBase}/projects/${project.id}/workflows`) {
        return Promise.resolve({
          ok: true,
          json: async () => ({ workflows: [workflow] }),
        } as Response);
      }

      if (url === `${apiBase}/workflows/${workflow.id}/execute`) {
        return Promise.resolve({
          ok: true,
          json: async () => ({ execution_id: "exec-123", status: "pending" }),
        } as Response);
      }

      return Promise.reject(new Error(`Unhandled fetch call for ${url}`));
    });

    const mockFetch = fetchMock as unknown as typeof fetch;
    global.fetch = mockFetch;
    window.fetch = mockFetch;
  });

  afterEach(() => {
    global.fetch = originalFetch;
  });

  it("starts executions through the execution store when executing a workflow", async () => {
    const user = userEvent.setup();

    renderWithProviders(
      <ProjectDetail
        project={project}
        onBack={() => {}}
        onWorkflowSelect={async () => {}}
        onCreateWorkflow={() => {}}
      />,
    );

    // Wait for workflows to load first (workflow card should appear)
    await screen.findAllByTestId(selectors.workflows.card);

    // Open the workflow actions dropdown menu first (card view has execute in dropdown)
    const actionsButton = await screen.findByLabelText("Workflow actions");
    await user.click(actionsButton);

    // Now find and click the execute button in the dropdown
    const executeButton = await screen.findByTestId(
      selectors.workflowBuilder.executeButton,
    );
    await user.click(executeButton);

    await waitFor(() => {
      expect(executionStoreState.startExecution).toHaveBeenCalledWith(
        workflow.id,
      );
    });
  });
});
