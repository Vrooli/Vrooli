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
      expect(screen.getByText("Loading backlog...")).toBeInTheDocument();
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
});
