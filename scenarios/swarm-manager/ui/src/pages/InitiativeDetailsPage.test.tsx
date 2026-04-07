import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { InitiativeDetailsPage } from "./InitiativeDetailsPage";
import { useBacklogStore, useDetailSelectionStore } from "../stores";

// jsdom doesn't provide matchMedia (needed by useIsMobile in DetailPageLayout).
beforeAll(() => {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(),
      removeListener: vi.fn(),
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });
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
}));

vi.mock("../services", () => ({
  initiativeService: {
    list: vi.fn(),
    get: vi.fn(),
    listFiles: vi.fn().mockResolvedValue([]),
    getFileContent: vi.fn(),
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
      archived: 0,
    },
  };

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });
    vi.clearAllMocks();

    // Pre-populate the detail selection store
    useDetailSelectionStore.getState().selectInitiative("test-initiative");

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

  const renderPage = () => {
    return render(
      <MemoryRouter initialEntries={["/graph"]}>
        <QueryClientProvider client={queryClient}>
          <InitiativeDetailsPage />
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
    // Title is shown in the DetailPageHeader
    expect(screen.getByTestId("detail-page-header")).toHaveTextContent("Test Initiative");
    const badges = screen.getAllByTestId("status-badge");
    expect(badges.length).toBeGreaterThanOrEqual(1);
    expect(badges[0]).toHaveTextContent("active");
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

  it("renders member item chips as clickable buttons", async () => {
    vi.mocked(initiativeService.get).mockResolvedValue(mockInitiativeData);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("initiative-details-items-list")).toBeInTheDocument();
    });

    const item1Button = screen.getByText("Execute Item 1");
    expect(item1Button.tagName).toBe("BUTTON");

    const item2Button = screen.getByText("Research Item 2");
    expect(item2Button.tagName).toBe("BUTTON");
  });

  it("clicking member item chip selects that backlog item", async () => {
    vi.mocked(initiativeService.get).mockResolvedValue(mockInitiativeData);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("initiative-details-items-list")).toBeInTheDocument();
    });

    const user = userEvent.setup();
    await user.click(screen.getByText("Execute Item 1"));

    const selection = useDetailSelectionStore.getState().selection;
    expect(selection?.entityType).toBe("backlog");
    expect(selection?.kind).toBe("execute");
    expect(selection?.name).toBe("item-1");
  });

  it("shows error state when fetch fails", async () => {
    vi.mocked(initiativeService.get).mockRejectedValue(new Error("Not found"));
    renderPage();

    await waitFor(() => {
      expect(screen.getByText("Failed to load initiative")).toBeInTheDocument();
    });
  });

  it("renders lens bar with navigation buttons", async () => {
    vi.mocked(initiativeService.get).mockResolvedValue(mockInitiativeData);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("lens-bar")).toBeInTheDocument();
    });
    expect(screen.getByTestId("lens-bar-focus")).toBeInTheDocument();
    expect(screen.getByTestId("lens-bar-topology")).toBeInTheDocument();
    expect(screen.getByTestId("lens-bar-operations")).toBeInTheDocument();
  });

  it("nav button clears detail selection on desktop", async () => {
    vi.mocked(initiativeService.get).mockResolvedValue(mockInitiativeData);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("detail-nav-button")).toBeInTheDocument();
    });

    const user = userEvent.setup();
    await user.click(screen.getByTestId("detail-nav-button"));

    expect(useDetailSelectionStore.getState().selection).toBeNull();
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
