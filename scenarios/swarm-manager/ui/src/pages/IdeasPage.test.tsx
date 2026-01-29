import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { IdeasPage } from "./IdeasPage";
import { useIdeasStore } from "../stores";

/**
 * Mock the config module for testing.
 * This allows tests to control configuration values without retries.
 */
vi.mock("../config", () => ({
  dataFetchingConfig: {
    retryCount: 0, // Disable retries for faster test failures
    retryDelayMs: 0,
    staleTimeMs: 0,
    cacheTimeMs: 0,
    refetchOnWindowFocus: false,
  },
  displayLimitsConfig: {
    ideaCardMaxTags: 3,
    scenarioCardMaxTags: 5,
    descriptionLineClamp: 2,
    defaultPageSize: 20,
  },
  apiConfig: {
    requestTimeoutMs: 30000,
    apiVersion: "v1",
  },
}));

/**
 * Mock the services layer - this is the seam we use for testing.
 * By mocking at the service level instead of the raw API client,
 * tests are more focused on behavior and less coupled to HTTP details.
 */
vi.mock("../services", () => ({
  ideasService: {
    list: vi.fn(),
    get: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}));

import { ideasService } from "../services";

// [REQ:MOD-P0-001] Test ideas UI components
describe("IdeasPage", () => {
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
    useIdeasStore.getState().reset();
  });

  const renderPage = () => {
    return render(
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <IdeasPage />
        </QueryClientProvider>
      </MemoryRouter>
    );
  };

  // [REQ:REQ-P0-003a] Test ideas list UI renders correctly
  it("renders the ideas page with search and create button", () => {
    vi.mocked(ideasService.list).mockResolvedValue([]);
    renderPage();

    expect(screen.getByTestId("ideas-page")).toBeInTheDocument();
    expect(screen.getByTestId("ideas-search")).toBeInTheDocument();
    expect(screen.getByTestId("create-idea")).toBeInTheDocument();
    expect(screen.getByTestId("ideas-filter")).toBeInTheDocument();
  });

  // [REQ:REQ-P0-003a] Test empty state when no ideas exist
  it("shows empty state when no ideas exist", async () => {
    vi.mocked(ideasService.list).mockResolvedValue([]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("ideas-empty")).toBeInTheDocument();
    });

    expect(screen.getByText("No ideas yet")).toBeInTheDocument();
    expect(screen.getByTestId("create-first-idea")).toBeInTheDocument();
  });

  // [REQ:REQ-P0-003a] Test idea cards display correctly
  it("renders idea cards with correct data", async () => {
    const mockIdeas = [
      {
        name: "test-idea",
        title: "Test Idea",
        description: "A test idea description",
        status: "backlog" as const,
        priority: 1,
        tags: ["test", "automation"],
        created: "2026-01-28T00:00:00Z",
        updated: "2026-01-28T00:00:00Z",
      },
    ];
    vi.mocked(ideasService.list).mockResolvedValue(mockIdeas);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("ideas-grid")).toBeInTheDocument();
    });

    // Note: with Continue Working section (Phase 29), non-completed ideas appear in both
    // the Continue Working section and the All Ideas grid. Use getAllByText for text that
    // appears in both places, getByText for unique elements (like priority and tags).
    const titleElements = screen.getAllByText("Test Idea");
    expect(titleElements.length).toBeGreaterThanOrEqual(1);
    const descElements = screen.getAllByText("A test idea description");
    expect(descElements.length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText("P1")).toBeInTheDocument();
    expect(screen.getByText("test")).toBeInTheDocument();
    expect(screen.getByText("automation")).toBeInTheDocument();
  });

  // [REQ:REQ-P0-003a] Test idea status colors display
  it("displays correct status indicator", async () => {
    const mockIdeas = [
      {
        name: "ready-idea",
        title: "Ready Idea",
        description: "An idea ready for development",
        status: "ready" as const,
        priority: 1,
        tags: [],
        created: "2026-01-28T00:00:00Z",
        updated: "2026-01-28T00:00:00Z",
      },
    ];
    vi.mocked(ideasService.list).mockResolvedValue(mockIdeas);
    renderPage();

    await waitFor(() => {
      // formatIdeaStatus now capitalizes status text (displayed as uppercase via CSS)
      expect(screen.getByText("Ready")).toBeInTheDocument();
    });
  });

  // [REQ:REQ-P0-003a] Test loading state
  it("shows loading state while fetching ideas", async () => {
    vi.mocked(ideasService.list).mockImplementation(
      () => new Promise(() => {}) // Never resolves
    );
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("Loading ideas...")).toBeInTheDocument();
    });
  });

  // [REQ:REQ-P0-003a] Test error state shows ErrorState component (not empty state)
  it("shows error state when API fails", async () => {
    vi.mocked(ideasService.list).mockRejectedValue(new Error("API error"));
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("error-state")).toBeInTheDocument();
    });

    // Error state should show user-friendly message, not "No ideas yet"
    expect(screen.getByText("Unable to load ideas")).toBeInTheDocument();
    expect(screen.queryByText("No ideas yet")).not.toBeInTheDocument();
  });

  // [REQ:REQ-P0-003a] Test error state has retry button
  it("shows retry button on error state", async () => {
    vi.mocked(ideasService.list).mockRejectedValue(new Error("Network error"));
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("error-retry")).toBeInTheDocument();
    });

    expect(screen.getByText("Try again")).toBeInTheDocument();
  });

  // [REQ:REQ-P0-003a] Test retry button triggers refetch
  it("retry button triggers refetch", async () => {
    // First call fails, second succeeds
    vi.mocked(ideasService.list)
      .mockRejectedValueOnce(new Error("Network error"))
      .mockResolvedValueOnce([]);

    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("error-state")).toBeInTheDocument();
    });

    // Click retry
    fireEvent.click(screen.getByTestId("error-retry"));

    // Should refetch and show empty state (not error state)
    await waitFor(() => {
      expect(screen.getByTestId("ideas-empty")).toBeInTheDocument();
    });

    expect(ideasService.list).toHaveBeenCalledTimes(2);
  });
});
