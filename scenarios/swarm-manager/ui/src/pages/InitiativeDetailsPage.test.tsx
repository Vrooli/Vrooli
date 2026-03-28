import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { InitiativeDetailsPage } from "./InitiativeDetailsPage";
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
  initiativeService: {
    list: vi.fn(),
    get: vi.fn(),
  },
}));

import { initiativeService } from "../services";

describe("InitiativeDetailsPage", () => {
  let queryClient: QueryClient;

  const mockInitiativeData = {
    initiative: {
      name: "test-initiative",
      title: "Test Initiative",
      description: "A test initiative for unit testing",
      status: "active",
      items: ["execute/item-1", "research/item-2"],
      created: "2026-03-27T00:00:00Z",
      updated: "2026-03-28T00:00:00Z",
    },
    rollup: {
      total: 2,
      completed: 1,
      inProgress: 1,
      failed: 0,
      pending: 0,
    },
  };

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    vi.clearAllMocks();
    useBacklogStore.getState().setItems([
      {
        kind: "execute" as const,
        name: "item-1",
        title: "Execute Item 1",
        description: "",
        status: "completed" as const,
        priority: 1,
        tags: [],
        dependsOn: [],
        acceptanceAllow: [],
        acceptanceDeny: [],
        created: "2026-03-27T00:00:00Z",
        updated: "2026-03-28T00:00:00Z",
      },
      {
        kind: "research" as const,
        name: "item-2",
        title: "Research Item 2",
        description: "",
        status: "in_progress" as const,
        priority: 2,
        tags: [],
        dependsOn: [],
        acceptanceAllow: [],
        acceptanceDeny: [],
        created: "2026-03-27T00:00:00Z",
        updated: "2026-03-28T00:00:00Z",
      },
    ]);
  });

  const renderPage = (name = "test-initiative", search = "") => {
    return render(
      <MemoryRouter initialEntries={[`/details/initiative/${name}${search}`]}>
        <QueryClientProvider client={queryClient}>
          <Routes>
            <Route path="/details/initiative/:name" element={<InitiativeDetailsPage />} />
            <Route path="/graph" element={<div data-testid="graph-page" />} />
          </Routes>
        </QueryClientProvider>
      </MemoryRouter>,
    );
  };

  it("renders initiative title and status on successful load", async () => {
    vi.mocked(initiativeService.get).mockResolvedValue(mockInitiativeData);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("initiative-details-page")).toBeInTheDocument();
    });
    expect(screen.getByTestId("initiative-details-title")).toHaveTextContent("Test Initiative");
    expect(screen.getByTestId("initiative-details-status")).toHaveTextContent("active");
  });

  it("renders description", async () => {
    vi.mocked(initiativeService.get).mockResolvedValue(mockInitiativeData);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("initiative-details-description")).toBeInTheDocument();
    });
    expect(screen.getByText("A test initiative for unit testing")).toBeInTheDocument();
  });

  it("renders rollup progress", async () => {
    vi.mocked(initiativeService.get).mockResolvedValue(mockInitiativeData);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("initiative-details-rollup")).toBeInTheDocument();
    });
    expect(screen.getByText("1 completed")).toBeInTheDocument();
    expect(screen.getByText("1 in progress")).toBeInTheDocument();
    expect(screen.getByText("2 total")).toBeInTheDocument();
  });

  it("renders member item chips as links", async () => {
    vi.mocked(initiativeService.get).mockResolvedValue(mockInitiativeData);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("initiative-details-items-list")).toBeInTheDocument();
    });

    const item1Link = screen.getByText("Execute Item 1");
    expect(item1Link.closest("a")).toHaveAttribute(
      "href",
      expect.stringContaining("/details/backlog/execute/item-1"),
    );

    const item2Link = screen.getByText("Research Item 2");
    expect(item2Link.closest("a")).toHaveAttribute(
      "href",
      expect.stringContaining("/details/backlog/research/item-2"),
    );
  });

  it("shows error state when fetch fails", async () => {
    vi.mocked(initiativeService.get).mockRejectedValue(new Error("Not found"));
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("Failed to load initiative")).toBeInTheDocument();
    });
  });

  it("back link uses returnTo param when present", async () => {
    vi.mocked(initiativeService.get).mockResolvedValue(mockInitiativeData);
    renderPage("test-initiative", "?returnTo=/details/backlog/execute/item-1");

    await waitFor(() => {
      expect(screen.getByTestId("initiative-details-back-link")).toHaveAttribute(
        "href",
        "/details/backlog/execute/item-1",
      );
    });
  });

  it("back link defaults to /graph when no returnTo", async () => {
    vi.mocked(initiativeService.get).mockResolvedValue(mockInitiativeData);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("initiative-details-back-link")).toHaveAttribute("href", "/graph");
    });
  });

  it("falls back to kind/name when item not in backlog store", async () => {
    useBacklogStore.getState().setItems([]); // clear store
    vi.mocked(initiativeService.get).mockResolvedValue(mockInitiativeData);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("initiative-details-items-list")).toBeInTheDocument();
    });
    expect(screen.getByText("execute/item-1")).toBeInTheDocument();
    expect(screen.getByText("research/item-2")).toBeInTheDocument();
  });
});
