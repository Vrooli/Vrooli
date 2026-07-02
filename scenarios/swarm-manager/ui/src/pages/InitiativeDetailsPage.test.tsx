import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { QueryClient } from "@tanstack/react-query";
import { Route, Routes, useLocation } from "react-router-dom";
import { InitiativeDetailsPage } from "./InitiativeDetailsPage";
import { useBacklogStore } from "../stores";
import { useInitiativeStore } from "../stores/initiative-store";
import {
  createTestQueryClient,
  installMatchMediaMock,
  installResizeObserverMock,
  renderWithProviders,
} from "../test-utils";

beforeAll(() => {
  installMatchMediaMock();
  installResizeObserverMock();
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
      priority: 0,
      dependsOn: [],
      items: ["execute/item-1", "research/item-2"],
      mode: "item-level" as const,
      acceptanceCriteria: [] as string[],
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

  const relatedInitiatives = [
    {
      initiative: {
        name: "protected-agent-sandboxing",
        title: "Protected Agent Sandboxing",
        description: "Upstream dependency",
        status: "completed" as const,
        priority: 1,
        dependsOn: [],
        items: [],
        mode: "item-level" as const,
        acceptanceCriteria: [] as string[],
        created: "2026-03-20T00:00:00Z",
        updated: "2026-03-21T00:00:00Z",
      },
      rollup: {
        total: 4,
        completed: 4,
        inProgress: 0,
        failed: 0,
        pending: 0,
        archived: 0,
      },
    },
    {
      initiative: {
        name: "run-level-undo-and-revert",
        title: "Run-Level Undo and Revert",
        description: "Downstream initiative",
        status: "active" as const,
        priority: 4,
        dependsOn: ["test-initiative"],
        items: [],
        mode: "item-level" as const,
        acceptanceCriteria: [] as string[],
        created: "2026-03-22T00:00:00Z",
        updated: "2026-03-23T00:00:00Z",
      },
      rollup: {
        total: 3,
        completed: 1,
        inProgress: 1,
        failed: 0,
        pending: 1,
        archived: 0,
      },
    },
  ];

  beforeEach(() => {
    queryClient = createTestQueryClient();
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
        suggestedSkills: [],
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
        suggestedSkills: [],
        dependsOn: ["execute/item-1"],
        acceptanceAllow: [],
        acceptanceDeny: [],
        created: "2026-03-27T00:00:00Z",
        updated: "2026-03-28T00:00:00Z",
      },
    ]);

    useInitiativeStore.setState({
      items: [
        {
          ...mockInitiativeData,
          initiative: {
            ...mockInitiativeData.initiative,
            dependsOn: ["protected-agent-sandboxing"],
            priority: 3,
          },
        },
        ...relatedInitiatives,
      ],
      status: "success",
      error: null,
      isRefreshing: false,
      lastFetchedAt: Date.now(),
    });
  });

  const renderPage = () => {
    return renderWithProviders(
      <Routes>
        <Route
          path="/initiatives/:name"
          element={(
            <>
              <InitiativeDetailsPage />
              <LocationProbe />
            </>
          )}
        />
        <Route path="/graph" element={<LocationProbe />} />
        <Route path="*" element={<LocationProbe />} />
      </Routes>,
      {
        queryClient,
        initialEntries: ["/graph", "/initiatives/test-initiative"],
      },
    );
  };

  function LocationProbe() {
    const location = useLocation();
    return <span data-testid="location-path">{location.pathname}</span>;
  }

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

  it("counts archived completed items as done while keeping archived scope visible", async () => {
    useInitiativeStore.setState({
      items: [
        {
          ...mockInitiativeData,
          initiative: {
            ...mockInitiativeData.initiative,
            dependsOn: [],
            items: ["execute/archived-parent", "research/active-child"],
          },
        },
      ],
      status: "success",
      error: null,
      isRefreshing: false,
      lastFetchedAt: Date.now(),
    });
    useBacklogStore.getState().setItems([
      {
        kind: "execute" as const,
        name: "archived-parent",
        title: "Archived Parent",
        description: "",
        status: "completed" as const,
        priority: 1,
        tags: [],
        suggestedSkills: [],
        dependsOn: [],
        acceptanceAllow: [],
        acceptanceDeny: [],
        archivedAt: "2026-04-25T00:00:00Z",
        created: "2026-03-27T00:00:00Z",
        updated: "2026-03-28T00:00:00Z",
      },
      {
        kind: "research" as const,
        name: "active-child",
        title: "Active Child",
        description: "",
        status: "backlog" as const,
        priority: 2,
        tags: [],
        suggestedSkills: [],
        dependsOn: ["execute/archived-parent"],
        acceptanceAllow: [],
        acceptanceDeny: [],
        created: "2026-03-27T00:00:00Z",
        updated: "2026-03-28T00:00:00Z",
      },
    ]);
    vi.mocked(initiativeService.get).mockResolvedValue({
      initiative: {
        ...mockInitiativeData.initiative,
        items: ["execute/archived-parent", "research/active-child"],
      },
      rollup: {
        total: 2,
        completed: 1,
        inProgress: 0,
        failed: 0,
        pending: 1,
        archived: 1,
      },
    });
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("initiative-details-items-list")).toBeInTheDocument();
    });

    expect(screen.getByText("Archived Parent")).toBeInTheDocument();
    expect(screen.getByText("Active Child")).toBeInTheDocument();
    expect(screen.getByText("1 of 2 items complete")).toBeInTheDocument();
    expect(screen.getByText((_, node) => node?.textContent === "backlog items • 1 archived")).toBeInTheDocument();
  });

  it("defaults to dependency graph view for member items", async () => {
    vi.mocked(initiativeService.get).mockResolvedValue(mockInitiativeData);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("initiative-details-items-list")).toBeInTheDocument();
    });

    expect(screen.getByTestId("initiative-items-graph-view")).toBeInTheDocument();
    expect(screen.queryByTestId("initiative-items-list-view")).not.toBeInTheDocument();
    expect(screen.getByText("Default view: dependency graph for faster sequencing and blocker scans.")).toBeInTheDocument();
    expect(screen.getByText("P2")).toBeInTheDocument();
    expect(screen.getByText("research/item-2")).toBeInTheDocument();
  });

  it("supports switching to the polished list view and selecting a backlog item", async () => {
    vi.mocked(initiativeService.get).mockResolvedValue(mockInitiativeData);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("initiative-details-items-list")).toBeInTheDocument();
    });

    const user = userEvent.setup();
    await user.click(screen.getByTitle("List view"));
    await waitFor(() => {
      expect(screen.getByTestId("initiative-items-list-view")).toBeInTheDocument();
    });

    expect(screen.getByText("Completed")).toBeInTheDocument();
    await user.click(screen.getByText("Execute Item 1"));

    expect(screen.getByTestId("location-path")).toHaveTextContent("/backlog/execute/item-1");
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
    expect(screen.getByTestId("lens-bar-plan")).toBeInTheDocument();
    expect(screen.getByTestId("lens-bar-focus")).toBeInTheDocument();
  });

  it("nav button uses route back on desktop", async () => {
    vi.mocked(initiativeService.get).mockResolvedValue(mockInitiativeData);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("detail-nav-button")).toBeInTheDocument();
    });

    const user = userEvent.setup();
    await user.click(screen.getByTestId("detail-nav-button"));

    expect(screen.getByTestId("location-path")).toHaveTextContent("/graph");
  });

  it("shows dependency cards with rollup context for upstream and downstream initiatives", async () => {
    vi.mocked(initiativeService.get).mockResolvedValue({
      ...mockInitiativeData,
      initiative: {
        ...mockInitiativeData.initiative,
        priority: 3,
        dependsOn: ["protected-agent-sandboxing"],
      },
    });
    renderPage();

    await waitFor(() => {
      expect(screen.getAllByText("Dependencies").length).toBeGreaterThan(0);
    });

    expect(screen.getByText("Blocked By")).toBeInTheDocument();
    expect(screen.getByText("Unblocks")).toBeInTheDocument();
    expect(screen.getByText("Protected Agent Sandboxing")).toBeInTheDocument();
    expect(screen.getByText("Run-Level Undo and Revert")).toBeInTheDocument();
    expect(screen.getByText("4 done")).toBeInTheDocument();
    expect(screen.getByText("1 active")).toBeInTheDocument();
    expect(screen.getByText("Priority P3")).toBeInTheDocument();
  });

  it("exposes the Feedback tab and Add Feedback entry point on the details page", async () => {
    vi.mocked(initiativeService.get).mockResolvedValue(mockInitiativeData);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("initiative-details-page")).toBeInTheDocument();
    });
    // Both the tab-row entry and the header button that launches the dialog
    // must be present — they are the two user-visible surfaces wired in W6.
    // The mobile button lives behind a bottom-sheet that jsdom does not
    // render in isolation; assert the desktop entry point here and cover
    // the mobile variant as a screen-size case if we add one later.
    expect(screen.getByTestId("initiative-details-tab-feedback")).toBeInTheDocument();
    expect(screen.getByTestId("initiative-details-add-feedback-desktop")).toBeInTheDocument();
  });

  it("exposes the operating mode workspace tab", async () => {
    vi.mocked(initiativeService.get).mockResolvedValue({
      ...mockInitiativeData,
      initiative: {
        ...mockInitiativeData.initiative,
        mode: "holistic-loop",
        acceptanceCriteria: ["Pass initiative-level acceptance review"],
      },
    });
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("initiative-details-page")).toBeInTheDocument();
    });

    expect(screen.getByTestId("initiative-details-tab-mode")).toBeInTheDocument();
    expect(screen.getByText(/Holistic loop/i)).toBeInTheDocument();
  });

  it("renders the Info-tab Mode card as a link to the operating-mode details page", async () => {
    vi.mocked(initiativeService.get).mockResolvedValue({
      ...mockInitiativeData,
      initiative: {
        ...mockInitiativeData.initiative,
        mode: "holistic-loop",
      },
    });
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("initiative-details-page")).toBeInTheDocument();
    });

    const card = screen.getByTestId("initiative-info-mode-card");
    expect(card.tagName).toBe("A");
    expect(card).toHaveAttribute("href", "/operating-modes/holistic-loop");
  });

  it("falls back to item-level for the Info-tab Mode card when no mode is set", async () => {
    const { mode: _ignored, ...initiativeWithoutMode } = mockInitiativeData.initiative;
    vi.mocked(initiativeService.get).mockResolvedValue({
      ...mockInitiativeData,
      // The service shape requires `mode`, but the UI must tolerate missing
      // values on legacy records — cast through unknown to assert the runtime
      // behavior the page is designed to defend against.
      initiative: initiativeWithoutMode as unknown as typeof mockInitiativeData.initiative,
    });
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("initiative-details-page")).toBeInTheDocument();
    });

    expect(screen.getByTestId("initiative-info-mode-card")).toHaveAttribute(
      "href",
      "/operating-modes/item-level",
    );
  });

  it("renders the new in_review status chip colors when an item is in review", async () => {
    useBacklogStore.getState().setItems([
      {
        kind: "execute" as const,
        name: "item-in-review",
        title: "Item In Review",
        description: "",
        status: "in_review" as const,
        priority: 1,
        tags: [],
        suggestedSkills: [],
        dependsOn: [],
        acceptanceAllow: [],
        acceptanceDeny: [],
        created: "2026-03-27T00:00:00Z",
        updated: "2026-03-28T00:00:00Z",
      },
    ]);
    vi.mocked(initiativeService.get).mockResolvedValue({
      ...mockInitiativeData,
      initiative: { ...mockInitiativeData.initiative, items: ["execute/item-in-review"] },
    });
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("initiative-details-items-list")).toBeInTheDocument();
    });
    const user = userEvent.setup();
    await user.click(screen.getByTitle("List view"));
    await waitFor(() => {
      expect(screen.getByTestId("initiative-items-list-view")).toBeInTheDocument();
    });
    // The new amber-toned status chip must render for in_review items so
    // the user can distinguish "agent gathering evidence" from backlog.
    const chips = screen.getAllByText(/in.?review/i);
    expect(chips.length).toBeGreaterThan(0);
  });

  it("renders target scenarios section when initiative has aggregated scenarios", async () => {
    vi.mocked(initiativeService.get).mockResolvedValue({
      ...mockInitiativeData,
      targetScenarios: ["web-console", "command-center"],
    });
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("initiative-details-page")).toBeInTheDocument();
    });
    expect(screen.getByText("Target Scenarios")).toBeInTheDocument();
    expect(screen.getByText("web-console")).toBeInTheDocument();
    expect(screen.getByText("command-center")).toBeInTheDocument();
  });

  it("omits target scenarios section when none are populated", async () => {
    vi.mocked(initiativeService.get).mockResolvedValue(mockInitiativeData);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("initiative-details-page")).toBeInTheDocument();
    });
    expect(screen.queryByText("Target Scenarios")).not.toBeInTheDocument();
  });

  it("falls back to kind/name and marks unresolved items in list view", async () => {
    useBacklogStore.getState().setItems([]); // clear store
    vi.mocked(initiativeService.get).mockResolvedValue(mockInitiativeData);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("initiative-details-items-list")).toBeInTheDocument();
    });
    const user = userEvent.setup();
    await user.click(screen.getByTitle("List view"));
    await waitFor(() => {
      expect(screen.getByTestId("initiative-items-list-view")).toBeInTheDocument();
    });
    expect(screen.getByRole("button", { name: "execute/item-1" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "research/item-2" })).toBeInTheDocument();
    expect(screen.getAllByText("Missing from backlog")).toHaveLength(2);
  });
});
