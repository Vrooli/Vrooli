import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import type { ReactElement } from "react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  backlogStoreInitialState,
  captureStoreInitialState,
  useBacklogStore,
  useCaptureStore,
} from "../../../../stores";
import { BacklogTab } from "./BacklogTab";
import { CapturesTab } from "./CapturesTab";
import { Sidebar } from "./Sidebar";

const sidebarStateKey = "swarm-manager.sidebar.state.v1";

function renderWithProviders(ui: ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <MemoryRouter>
      <QueryClientProvider client={queryClient}>
        {ui}
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("sidebar create affordances", () => {
  beforeEach(() => {
    window.localStorage.clear();
    vi.restoreAllMocks();
    useBacklogStore.setState({
      ...backlogStoreInitialState,
      items: [],
      blockingMap: {},
      status: "success",
    });
    useCaptureStore.setState({
      ...captureStoreInitialState,
      captures: [],
      status: "success",
    });
  });

  it("wires the backlog empty-state action to the shared create handler", () => {
    const onCreateBacklog = vi.fn();

    renderWithProviders(
      <BacklogTab
        searchQuery=""
        filters={{
          statuses: [],
          kinds: [],
          priorityMin: null,
          priorityMax: null,
          showArchived: false,
        }}
        sort={{ field: "priority", direction: "asc" }}
        onItemClick={vi.fn()}
        onCreateBacklog={onCreateBacklog}
      />,
    );

    fireEvent.click(screen.getByTestId("backlog-tab-create-item"));

    expect(onCreateBacklog).toHaveBeenCalledOnce();
  });

  it("wires the captures empty-state action to the shared quick-capture handler", () => {
    const onCreateCapture = vi.fn();

    renderWithProviders(
      <CapturesTab
        searchQuery=""
        filters={{ statuses: [] }}
        sort={{ field: "recency", direction: "desc" }}
        onItemClick={vi.fn()}
        onCreateCapture={onCreateCapture}
      />,
    );

    fireEvent.click(screen.getByTestId("captures-tab-create-capture"));

    expect(onCreateCapture).toHaveBeenCalledOnce();
  });

  it("shows the inline create icon only for creatable active tabs", () => {
    const onQuickCapture = vi.fn();
    window.localStorage.setItem(sidebarStateKey, JSON.stringify({
      activeTab: "captures",
      searchQuery: "",
      searchMode: "plain",
      filters: {},
      sorts: {},
    }));

    renderWithProviders(
      <Sidebar
        onItemClick={vi.fn()}
        onSettingsOpen={vi.fn()}
        onGoHome={vi.fn()}
        onQuickCapture={onQuickCapture}
      />,
    );

    fireEvent.click(screen.getByTestId("sidebar-create-current"));

    expect(onQuickCapture).toHaveBeenCalledOnce();
  });

  it("does not render a create-from-plan icon in the sidebar", () => {
    renderWithProviders(
      <Sidebar
        onItemClick={vi.fn()}
        onSettingsOpen={vi.fn()}
        onGoHome={vi.fn()}
      />,
    );

    expect(screen.queryByTestId("sidebar-create-from-plan")).not.toBeInTheDocument();
  });
});
