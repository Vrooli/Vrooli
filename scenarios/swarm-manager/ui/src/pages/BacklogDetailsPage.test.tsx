import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { screen, waitFor, fireEvent, within, cleanup, act } from "@testing-library/react";
import type { QueryClient } from "@tanstack/react-query";
import { Route, Routes } from "react-router-dom";
import { BacklogDetailsPage } from "./BacklogDetailsPage";
import { useBacklogStore, useAgentActivitiesStore, useBacklogDetailUIStore } from "../stores";
import {
  createTestQueryClient,
  installMatchMediaMock,
  renderWithProviders,
} from "../test-utils";

beforeAll(() => {
  installMatchMediaMock();
});

vi.mock("../config", () => ({
  dataFetchingConfig: {
    retryCount: 0,
    retryDelayMs: 0,
    staleTimeMs: 0,
    cacheTimeMs: 0,
    refetchOnWindowFocus: false,
  },
  displayLimitsConfig: {
    backlogCardMaxTags: 3,
    scenarioCardMaxTags: 5,
    descriptionLineClamp: 2,
    defaultPageSize: 20,
  },
  apiConfig: {
    requestTimeoutMs: 30000,
    apiVersion: "v1",
  },
  uiBehaviorConfig: {
    searchDebounceMs: 300,
    toastDurationMs: 5000,
  },
}));

// Mock useStorePolling to prevent real setInterval timers from firing in tests.
// Without this, each test starts a 6-second polling loop that calls unmocked
// services and accumulates memory until the worker OOMs after ~4 tests.
vi.mock("../hooks/useStorePolling", () => ({
  useStorePolling: vi.fn(),
}));

// Some hooks import services directly by file path (not through the barrel).
// We must mock those paths separately or the real service makes HTTP requests
// that hang until timeout, causing the test to appear stuck.
vi.mock("../services/execution-service", () => ({
  executionService: {
    list: vi.fn().mockResolvedValue([]),
    get: vi.fn(),
    create: vi.fn(),
    start: vi.fn(),
    cancel: vi.fn(),
    retry: vi.fn(),
    followUp: vi.fn(),
    triggerReview: vi.fn(),
  },
  createExecutionService: vi.fn(),
}));

vi.mock("../services/agent-activity-service", () => ({
  agentActivityService: {
    list: vi.fn().mockResolvedValue([]),
    get: vi.fn(),
    stopRun: vi.fn(),
  },
  createAgentActivityService: vi.fn(),
}));

// ClarificationPanel and InlineQuestionStepper import backlogService by file
// path, bypassing the barrel mock above.
vi.mock("../services/backlog-service", () => ({
  backlogService: {
    get: vi.fn(),
    getFiles: vi.fn(),
    getFileContent: vi.fn().mockResolvedValue(""),
    saveFileContent: vi.fn(),
    getClarification: vi.fn(),
    createClarification: vi.fn(),
    continueClarification: vi.fn(),
    clarificationAction: vi.fn(),
    workshopSave: vi.fn(),
    batchReview: vi.fn(),
  },
  createBacklogService: vi.fn(),
}));

// Safety net: mock the API client at the transport level so that ANY service
// that slipped through the per-file mocks above can't make real HTTP requests.
vi.mock("../lib/api-client", () => {
  const noopClient = {
    get: vi.fn().mockResolvedValue({}),
    post: vi.fn().mockResolvedValue({}),
    put: vi.fn().mockResolvedValue({}),
    patch: vi.fn().mockResolvedValue({}),
    delete: vi.fn().mockResolvedValue({}),
  };
  return {
    API_BASE: "http://localhost",
    DEFAULT_API_BASE: "http://localhost",
    defaultApiClient: noopClient,
    createApiClient: vi.fn(() => noopClient),
    ApiClient: vi.fn(() => noopClient),
    ApiError: class ApiError extends Error {
      status: number;
      userMessage: string;
      constructor(status: number, message: string) {
        super(message);
        this.status = status;
        this.userMessage = message;
      }
    },
    isApiError: vi.fn(() => false),
  };
});

vi.mock("../services", () => ({
  backlogService: {
    list: vi.fn(),
    get: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    getFiles: vi.fn(),
    getFileContent: vi.fn(),
    uploadFile: vi.fn(),
    saveFileContent: vi.fn(),
    renameFile: vi.fn(),
    moveFile: vi.fn(),
    copyFile: vi.fn(),
    deleteFile: vi.fn(),
    queue: vi.fn(),
    research: vi.fn(),
    convert: vi.fn(),
    listBySpawnedFrom: vi.fn(),
    getMaturitySummary: vi.fn(),
    getArchiveTargets: vi.fn(),
    workshopSave: vi.fn(),
    workshopDeleteRound: vi.fn(),
    updateModuleRequirements: vi.fn(),
    createModule: vi.fn(),
    updateModuleMeta: vi.fn(),
    deleteModule: vi.fn(),
    createArchiveTarget: vi.fn(),
    updateArchiveTarget: vi.fn(),
    deleteArchiveTarget: vi.fn(),
    batchReview: vi.fn(),
  },
  executionService: {
    list: vi.fn(),
    get: vi.fn(),
    create: vi.fn(),
    start: vi.fn(),
    cancel: vi.fn(),
    retry: vi.fn(),
    followUp: vi.fn(),
    triggerReview: vi.fn(),
  },
  agentActivityService: {
    list: vi.fn(),
    get: vi.fn(),
    stopRun: vi.fn(),
  },
}));

import { backlogService, executionService } from "../services";

describe("BacklogDetailsPage", () => {
  let queryClient: QueryClient;

  const mockItem = {
    name: "test-idea",
    title: "Test Idea Title",
    description: "A detailed description of the test idea",
    status: "backlog" as const,
    priority: 2,
    tags: ["feature", "api"],
    suggestedSkills: [],
    created: "2026-01-20T10:00:00Z",
    updated: "2026-01-25T15:30:00Z",
    kind: "idea" as const,
  };

  const mockFiles = [
    {
      name: "spec.json",
      path: "spec.json",
      type: "file" as const,
      size: 256,
    },
  ];

  beforeEach(() => {
    queryClient = createTestQueryClient();
    vi.clearAllMocks();
    useBacklogStore.getState().reset();
    useBacklogDetailUIStore.getState().reset();
    vi.mocked(backlogService.getFileContent).mockResolvedValue("Spec content");
    // Provide default resolved values for queries that BacklogDetailsPage fires
    // via useBacklogDetailData. Without these, vi.fn() returns undefined which
    // React Query treats as an error, causing repeated retries and stderr noise.
    vi.mocked(backlogService.listBySpawnedFrom).mockResolvedValue([]);
    vi.mocked(backlogService.getMaturitySummary).mockResolvedValue({ items: [] });
    vi.mocked(backlogService.getArchiveTargets).mockResolvedValue({ targets: [], requirements: [], has_archive: false });
    vi.mocked(executionService.list).mockResolvedValue([]);
  });

  afterEach(() => {
    cleanup();
    queryClient.clear();
    useBacklogStore.getState().reset();
    useAgentActivitiesStore.setState({ activities: [], isRefreshing: false });
    useBacklogDetailUIStore.getState().reset();
    vi.clearAllTimers();
  });

  const renderPage = (kind = "idea", name = "test-idea", tab?: string) => {
    const search = tab ? `?tab=${tab}` : "";
    return renderWithProviders(
      <Routes>
        <Route path="/backlog/:kind/:name" element={<BacklogDetailsPage />} />
      </Routes>,
      {
        queryClient,
        initialEntries: [`/backlog/${kind}/${name}${search}`],
      },
    );
  };

  it("renders the backlog details page container", async () => {
    vi.mocked(backlogService.get).mockResolvedValue(mockItem);
    vi.mocked(backlogService.getFiles).mockResolvedValue(mockFiles);

    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("backlog-details-page")).toBeInTheDocument();
    });
  });

  it("shows nav button in shared header", async () => {
    vi.mocked(backlogService.get).mockResolvedValue(mockItem);
    vi.mocked(backlogService.getFiles).mockResolvedValue(mockFiles);

    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("detail-nav-button")).toBeInTheDocument();
    });
  });

  it("shows queue button for idea backlog items", async () => {
    vi.mocked(backlogService.get).mockResolvedValue(mockItem);
    vi.mocked(backlogService.getFiles).mockResolvedValue(mockFiles);

    renderPage();

    await waitFor(() => {
      expect(screen.getAllByTestId("backlog-details-queue").length).toBeGreaterThan(0);
    });
  });

  it("shows only the primary CTA in the header", async () => {
    vi.mocked(backlogService.get).mockResolvedValue(mockItem);
    vi.mocked(backlogService.getFiles).mockResolvedValue(mockFiles);

    renderPage();

    const header = await screen.findByTestId("backlog-details-header");
    expect(within(header).getByRole("button", { name: "Run" })).toBeInTheDocument();
    expect(within(header).queryByRole("button", { name: "Workshop" })).not.toBeInTheDocument();
    expect(within(header).queryByRole("button", { name: /Next Round/i })).not.toBeInTheDocument();
  });

  it("reconciles post-approval finalize state and switches the header CTA from Finalize to Run", async () => {
    const executeItem = {
      ...mockItem,
      kind: "execute" as const,
      name: "agent-manager-sandbox-auto-apply-defaults",
      title: "Make sandboxed agent-manager runs auto-apply accepted changes by default",
    };
    const roundOne = JSON.stringify({
      round: 1,
      mode: "workshop",
      items: [],
      readiness: {},
      pending_synthesis: false,
    });
    const roundTwo = JSON.stringify({
      round: 2,
      mode: "workshop",
      items: [{ id: "d1", type: "decision", selected: "A" }],
      readiness: {},
      pending_synthesis: true,
    });
    const roundThree = JSON.stringify({
      round: 3,
      mode: "finalize",
      items: [],
      readiness: {},
      pending_synthesis: false,
    });

    vi.mocked(backlogService.get).mockResolvedValue(executeItem);
    vi.mocked(backlogService.getFiles)
      .mockResolvedValueOnce([
        {
          name: "workshop",
          path: "workshop",
          type: "directory" as const,
          children: [
            { name: "round-001.json", path: "workshop/round-001.json", type: "file" as const, size: 100 },
            { name: "round-002.json", path: "workshop/round-002.json", type: "file" as const, size: 100 },
          ],
        },
      ])
      .mockResolvedValue([
        {
          name: "workshop",
          path: "workshop",
          type: "directory" as const,
          children: [
            { name: "round-001.json", path: "workshop/round-001.json", type: "file" as const, size: 100 },
            { name: "round-002.json", path: "workshop/round-002.json", type: "file" as const, size: 100 },
            { name: "round-003.json", path: "workshop/round-003.json", type: "file" as const, size: 100 },
          ],
        },
      ]);
    vi.mocked(backlogService.getFileContent).mockImplementation(async (_kind, _name, path) => {
      if (path === "workshop/round-001.json") return roundOne;
      if (path === "workshop/round-002.json") return roundTwo;
      if (path === "workshop/round-003.json") return roundThree;
      return "";
    });
    vi.mocked(backlogService.getMaturitySummary)
      .mockResolvedValueOnce({
        items: [
          {
            kind: "execute",
            name: "agent-manager-sandbox-auto-apply-defaults",
            title: executeItem.title,
            rounds_completed: 2,
            raw_scores: {
              problem_clarity: 3,
              scope_defined: 3,
              approach_solid: 3,
              testable: 3,
              risk_awareness: 3,
            },
            effective_scores: {
              problem_clarity: 3,
              scope_defined: 3,
              approach_solid: 3,
              testable: 3,
              risk_awareness: 3,
            },
            ready: true,
            pending_items: 0,
            pending_synthesis: true,
            has_plan: true,
          },
        ],
      })
      .mockResolvedValue({
        items: [
          {
            kind: "execute",
            name: "agent-manager-sandbox-auto-apply-defaults",
            title: executeItem.title,
            rounds_completed: 3,
            raw_scores: {
              problem_clarity: 3,
              scope_defined: 3,
              approach_solid: 3,
              testable: 3,
              risk_awareness: 3,
            },
            effective_scores: {
              problem_clarity: 3,
              scope_defined: 3,
              approach_solid: 3,
              testable: 3,
              risk_awareness: 3,
            },
            ready: true,
            pending_items: 0,
            pending_synthesis: false,
            has_plan: true,
          },
        ],
      });

    useAgentActivitiesStore.setState({
      activities: [
        {
          activityId: "act-finalize",
          ownerType: "backlog",
          ownerKind: "execute",
          ownerName: "agent-manager-sandbox-auto-apply-defaults",
          ownerTitle: executeItem.title,
          purpose: "finalize",
          interactionType: "spawn",
          status: "needs_review",
          requestedAt: "2026-04-27T02:47:43Z",
          startedAt: "2026-04-27T02:47:44Z",
          updatedAt: "2026-04-27T02:47:44Z",
          runId: "run-finalize",
          isStopping: false,
        },
      ],
      isRefreshing: false,
    });

    renderPage("execute", "agent-manager-sandbox-auto-apply-defaults");

    const header = await screen.findByTestId("backlog-details-header");
    await waitFor(() => {
      expect(within(header).getByRole("button", { name: "Finalize" })).toBeInTheDocument();
    });

    await act(async () => {
      useAgentActivitiesStore.setState({ activities: [], isRefreshing: false });
    });

    await waitFor(() => {
      expect(within(header).getByRole("button", { name: "Run" })).toBeInTheDocument();
    });
    expect(within(header).queryByRole("button", { name: "Finalize" })).not.toBeInTheDocument();
  });

  it("shows queue button for research items (research items are queueable)", async () => {
    vi.mocked(backlogService.get).mockResolvedValue({
      ...mockItem,
      kind: "research",
      status: "researching",
    });
    vi.mocked(backlogService.getFiles).mockResolvedValue(mockFiles);

    renderPage("research", "test-research");

    await waitFor(() => {
      expect(screen.getByTestId("backlog-details-header")).toBeInTheDocument();
    });

    // Research items follow the normal CTA funnel and can be queued.
    expect(screen.getAllByTestId("backlog-details-queue").length).toBeGreaterThan(0);
  });

  it("renders file tree on files tab", async () => {
    vi.mocked(backlogService.get).mockResolvedValue(mockItem);
    vi.mocked(backlogService.getFiles).mockResolvedValue(mockFiles);

    renderPage("idea", "test-idea", "files");

    await waitFor(() => {
      expect(screen.getAllByTestId("backlog-details-file-tree").length).toBeGreaterThan(0);
    });
  });

  it("shows error state when fetch fails", async () => {
    vi.mocked(backlogService.get).mockRejectedValue(new Error("API error"));
    vi.mocked(backlogService.getFiles).mockResolvedValue(mockFiles);

    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("error-state")).toBeInTheDocument();
    });

    expect(screen.getByText("Unable to load backlog item")).toBeInTheDocument();
  });

  it("renders file header actions menu trigger for selected file", async () => {
    vi.mocked(backlogService.get).mockResolvedValue(mockItem);
    vi.mocked(backlogService.getFiles).mockResolvedValue(mockFiles);

    renderPage("idea", "test-idea", "files");

    await waitFor(() => {
      expect(screen.getAllByTestId("file-header-actions-trigger").length).toBeGreaterThan(0);
    });
  });

  it("opens file header actions menu when trigger is clicked", async () => {
    vi.mocked(backlogService.get).mockResolvedValue(mockItem);
    vi.mocked(backlogService.getFiles).mockResolvedValue(mockFiles);

    renderPage("idea", "test-idea", "files");

    await waitFor(() => {
      expect(screen.getAllByTestId("file-header-actions-trigger").length).toBeGreaterThan(0);
    });
    const triggers = screen.getAllByTestId("file-header-actions-trigger");
    const firstTrigger = triggers[0];
    expect(firstTrigger).toBeDefined();
    if (!firstTrigger) {
      throw new Error("File header actions trigger not found");
    }
    fireEvent.click(firstTrigger);

    await waitFor(() => {
      expect(screen.getAllByTestId("file-header-actions-popover").length).toBeGreaterThan(0);
      expect(screen.getAllByTestId("backlog-file-actions-menu").length).toBeGreaterThan(0);
    });
  });

  it("shows target scenarios panel when acceptanceAllow has scenario globs", async () => {
    vi.mocked(backlogService.get).mockResolvedValue({
      ...mockItem,
      acceptanceAllow: ["scenarios/web-console/api/**"],
    });
    vi.mocked(backlogService.getFiles).mockResolvedValue(mockFiles);

    renderPage();

    await waitFor(() => {
      expect(screen.getAllByText("Target Scenarios").length).toBeGreaterThan(0);
    });
    expect(screen.getAllByText("web-console").length).toBeGreaterThan(0);
  });

  it("hides target scenarios panel when no acceptanceAllow", async () => {
    vi.mocked(backlogService.get).mockResolvedValue(mockItem);
    vi.mocked(backlogService.getFiles).mockResolvedValue(mockFiles);

    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("backlog-details-page")).toBeInTheDocument();
    });
    expect(screen.queryByText("Target Scenarios")).not.toBeInTheDocument();
  });
});
