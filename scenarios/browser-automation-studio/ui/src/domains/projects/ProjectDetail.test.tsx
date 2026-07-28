import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { act, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {
  fetchJsonResponse,
  installFetchMock,
  renderWithProviders,
  type FetchMock,
} from "@/test-utils";
import type { Project } from "./store";
import ProjectDetail from "./ProjectDetail";
import { useProjectDetailStore } from "./hooks/useProjectDetailStore";
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

// Stub the Connect-RPC project adapter used by useProjectDetailStore.
const fetchProjectWorkflowsMock = vi.fn();
const fetchProjectEntriesMock = vi.fn().mockResolvedValue([]);
vi.mock("@/domains/projects/services/projectApi", () => ({
  fetchProjectWorkflows: (...a: unknown[]) => fetchProjectWorkflowsMock(...a),
  fetchProjectEntries: (...a: unknown[]) => fetchProjectEntriesMock(...a),
}));

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

vi.mock("@/domains/executions/store", () => {
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
  };
});

vi.mock("@/domains/executions/hooks/useStartWorkflow", () => {
  return {
    __esModule: true,
    useStartWorkflow: () => ({
      startWorkflow: async ({ workflowId }: { workflowId: string }) => {
        await executionStoreState.startExecution(workflowId);
        return "mock-execution-id";
      },
      promptDialogProps: {
        open: false,
        title: "",
        description: "",
        defaultValue: "",
        onSubmit: vi.fn(),
        onClose: vi.fn(),
      },
    }),
  };
});

const getConfigMock = vi.hoisted(() => vi.fn());
vi.mock("@/config", () => ({
  __esModule: true,
  getConfig: getConfigMock,
  getCachedConfig: vi.fn(() => ({ API_URL: "", WS_URL: "" })),
  config: vi.fn(() => ({ API_URL: "", WS_URL: "" })),
  getApiBase: vi.fn(() => ""),
  getWsBase: vi.fn(() => ""),
  API_BASE: "",
  WS_BASE: "",
}));

const toastMock = vi.hoisted(() => ({
  success: vi.fn(),
  error: vi.fn(),
}));
vi.mock("react-hot-toast", () => ({
  __esModule: true,
  default: toastMock,
}));

vi.mock("@/domains/executions/InlineExecutionViewer", () => ({
  __esModule: true,
  default: () => <div data-testid={selectors.executions.mock.viewer} />,
}));

vi.mock("@/domains/executions/history/ExecutionHistory", () => ({
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

vi.mock("@shared/modals", () => ({
  __esModule: true,
  useModals: () => ({
    showAIModal: false,
    showProjectModal: false,
    showWorkflowCreationModal: false,
    showAssetUploadModal: false,
    assetUploadConfig: null,
    showDocs: false,
    docsInitialTab: "getting-started" as const,
    openAIModal: vi.fn(),
    closeAIModal: vi.fn(),
    openProjectModal: vi.fn(),
    closeProjectModal: vi.fn(),
    openWorkflowCreationModal: vi.fn(),
    closeWorkflowCreationModal: vi.fn(),
    openAssetUploadModal: vi.fn(),
    closeAssetUploadModal: vi.fn(),
    openDocs: vi.fn(),
    closeDocs: vi.fn(),
    closeAllModals: vi.fn(),
  }),
  useIsAnyModalOpen: () => false,
  ModalProvider: ({ children }: { children: React.ReactNode }) => children,
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

  let fetchMock: FetchMock;

  beforeEach(() => {
    vi.clearAllMocks();
    getConfigMock.mockResolvedValue({ API_URL: apiBase });
    executionStoreState.startExecution.mockResolvedValue(undefined);

    fetchMock = installFetchMock();
    fetchProjectWorkflowsMock.mockResolvedValue([
      {
        id: workflow.id,
        name: workflow.name,
        description: workflow.description,
        project_id: workflow.project_id,
        folder_path: workflow.folder_path ?? '/',
        version: workflow.version ?? 1,
        created_at: workflow.created_at,
        updated_at: workflow.updated_at,
      },
    ]);
    fetchMock.mockImplementation((input: RequestInfo | URL, _init?: RequestInit) => {
      const url = typeof input === "string" ? input : input.toString();
      if (url === `${apiBase}/workflows/${workflow.id}/execute`) {
        return Promise.resolve(
          fetchJsonResponse({ execution_id: "exec-123", status: "pending" }),
        );
      }
      return Promise.reject(new Error(`Unhandled fetch call for ${url}`));
    });
  });

  afterEach(() => {
    vi.unstubAllGlobals();
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

    // Default viewMode is "tree" after the component's initializeForProject effect
    // runs; switch to "card" so this test can interact with the workflow card UI.
    await waitFor(() => {
      expect(useProjectDetailStore.getState().projectId).toBe(project.id);
    });
    act(() => {
      useProjectDetailStore.getState().setViewMode("card");
    });

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
