import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { BacklogPage } from "./BacklogPage";
import { useBacklogStore } from "../stores";

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
}));

vi.mock("../services", () => ({
  backlogService: {
    list: vi.fn(),
    get: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    queue: vi.fn(),
  },
  executionService: {
    list: vi.fn().mockResolvedValue([]),
  },
}));

import { backlogService } from "../services";

describe("BacklogPage", () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
        },
      },
    });
    vi.clearAllMocks();
    useBacklogStore.getState().reset();
    vi.mocked(backlogService.queue).mockResolvedValue({
      item: {
        name: "queued-item",
        title: "Queued Item",
        description: "",
        status: "queued",
        priority: 5,
        tags: [],
        kind: "idea",
        created: "2026-01-28T00:00:00Z",
        updated: "2026-01-28T00:00:00Z",
      },
      taskId: "task_1",
      runId: "run_1",
      baseUrl: "",
      created: "2026-01-28T00:00:00Z",
      dryRun: false,
      queued: true,
      message: "",
      blockingReasons: [],
      unansweredQuestions: 0,
      pendingSuggestions: 0,
    });
  });

  const renderPage = () => {
    return render(
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <BacklogPage />
        </QueryClientProvider>
      </MemoryRouter>
    );
  };

  it("renders the backlog page with search and create button", () => {
    vi.mocked(backlogService.list).mockResolvedValue([]);
    renderPage();

    expect(screen.getByTestId("backlog-page")).toBeInTheDocument();
    expect(screen.getByTestId("backlog-search")).toBeInTheDocument();
    expect(screen.getByTestId("create-backlog")).toBeInTheDocument();
    expect(screen.getByTestId("backlog-filter")).toBeInTheDocument();
  });

  it("shows empty state when no backlog items exist", async () => {
    vi.mocked(backlogService.list).mockResolvedValue([]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("backlog-empty")).toBeInTheDocument();
    });

    expect(screen.getByText("No ideas yet")).toBeInTheDocument();
    expect(screen.getByTestId("create-first-backlog")).toBeInTheDocument();
  });

  it("renders backlog cards with correct data", async () => {
    const mockItems = [
      {
        name: "test-idea",
        title: "Test Idea",
        description: "A test idea description",
        status: "backlog" as const,
        priority: 1,
        tags: ["test", "automation"],
        created: "2026-01-28T00:00:00Z",
        updated: "2026-01-28T00:00:00Z",
        kind: "idea" as const,
      },
    ];
    vi.mocked(backlogService.list).mockResolvedValue(mockItems);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("backlog-grid")).toBeInTheDocument();
    });

    const titleElements = screen.getAllByText("Test Idea");
    expect(titleElements.length).toBeGreaterThanOrEqual(1);
    const descElements = screen.getAllByText("A test idea description");
    expect(descElements.length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("P1")).toBeInTheDocument();
    expect(screen.getByText("test")).toBeInTheDocument();
    expect(screen.getByText("automation")).toBeInTheDocument();
  });

  it("displays correct status indicator", async () => {
    const mockItems = [
      {
        name: "ready-idea",
        title: "Ready Idea",
        description: "An idea ready for development",
        status: "ready" as const,
        priority: 1,
        tags: [],
        created: "2026-01-28T00:00:00Z",
        updated: "2026-01-28T00:00:00Z",
        kind: "idea" as const,
      },
    ];
    vi.mocked(backlogService.list).mockResolvedValue(mockItems);
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("Ready")).toBeInTheDocument();
    });
  });

  it("shows loading state while fetching backlog", async () => {
    vi.mocked(backlogService.list).mockImplementation(() => new Promise(() => {}));
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("backlog-loading-state")).toBeInTheDocument();
    });
  });

  it("shows error state when API fails", async () => {
    vi.mocked(backlogService.list).mockRejectedValue(new Error("API error"));
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("error-state")).toBeInTheDocument();
    });

    expect(screen.getByText("Unable to load backlog")).toBeInTheDocument();
    expect(screen.queryByText("No ideas yet")).not.toBeInTheDocument();
  });

  it("shows retry button on error state", async () => {
    vi.mocked(backlogService.list).mockRejectedValue(new Error("Network error"));
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("error-retry")).toBeInTheDocument();
    });

    expect(screen.getByText("Try again")).toBeInTheDocument();
  });

  it("retry button triggers refetch", async () => {
    vi.mocked(backlogService.list)
      .mockRejectedValueOnce(new Error("Network error"))
      .mockResolvedValueOnce([]);

    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("error-state")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByTestId("error-retry"));

    await waitFor(() => {
      expect(screen.getByTestId("backlog-empty")).toBeInTheDocument();
    });

    expect(backlogService.list).toHaveBeenCalledTimes(2);
  });

  it("shows a reason when an item is not queueable", async () => {
    vi.mocked(backlogService.list).mockResolvedValue([
      {
        name: "done-idea",
        title: "Done Idea",
        description: "Already complete",
        status: "completed" as const,
        priority: 1,
        tags: [],
        created: "2026-01-28T00:00:00Z",
        updated: "2026-01-28T00:00:00Z",
        kind: "idea" as const,
      },
    ]);

    renderPage();

    await waitFor(() => {
      expect(screen.getByText("Completed items cannot be queued again.")).toBeInTheDocument();
    });
  });

  it("allows archived ideas to be queued", async () => {
    vi.mocked(backlogService.list).mockResolvedValue([
      {
        name: "archived-idea",
        title: "Archived Idea",
        description: "Archived but should be queueable",
        status: "archived" as const,
        priority: 1,
        tags: [],
        created: "2026-01-28T00:00:00Z",
        updated: "2026-01-28T00:00:00Z",
        kind: "idea" as const,
      },
    ]);

    renderPage();

    await waitFor(() => {
      expect(screen.getByText("Run")).toBeInTheDocument();
    });
    expect(screen.queryByText("Only archived ideas can be queued directly.")).not.toBeInTheDocument();
  });

  it("enables batch selection and shows run button when items are selected", async () => {
    vi.mocked(backlogService.list).mockResolvedValue([
      {
        name: "idea-1",
        title: "Idea One",
        description: "desc",
        status: "ready" as const,
        priority: 1,
        tags: [],
        created: "2026-01-28T00:00:00Z",
        updated: "2026-01-28T00:00:00Z",
        kind: "idea" as const,
      },
      {
        name: "idea-2",
        title: "Idea Two",
        description: "desc",
        status: "backlog" as const,
        priority: 2,
        tags: [],
        created: "2026-01-28T00:00:00Z",
        updated: "2026-01-28T00:00:00Z",
        kind: "idea" as const,
      },
    ]);

    renderPage();

    // Enter batch mode to reveal the "Run Selected" button and item checkboxes
    await waitFor(() => {
      expect(screen.getByLabelText("Toggle batch mode")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByLabelText("Toggle batch mode"));

    // "Run Selected" button appears (disabled since nothing selected yet)
    await waitFor(() => {
      expect(screen.getByText("Run Selected")).toBeInTheDocument();
    });

    // Checkboxes should appear for each queueable item
    expect(screen.getByLabelText("Select backlog item Idea One")).toBeInTheDocument();
    expect(screen.getByLabelText("Select backlog item Idea Two")).toBeInTheDocument();

    // Select items — button should become enabled
    fireEvent.click(screen.getByLabelText("Select backlog item Idea One"));
    fireEvent.click(screen.getByLabelText("Select backlog item Idea Two"));

    await waitFor(() => {
      const el = screen.getByText("Run Selected");
      const btn = el.closest("button");
      expect(btn).not.toBeNull();
      expect(btn).not.toBeDisabled();
    });
  });
});
