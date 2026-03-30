import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { BacklogDetailsPage } from "./BacklogDetailsPage";
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
  },
}));

import { backlogService } from "../services";

describe("BacklogDetailsPage", () => {
  let queryClient: QueryClient;

  const mockItem = {
    name: "test-idea",
    title: "Test Idea Title",
    description: "A detailed description of the test idea",
    status: "backlog" as const,
    priority: 2,
    tags: ["feature", "api"],
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
    queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
        },
      },
    });
    vi.clearAllMocks();
    useBacklogStore.getState().reset();
    vi.mocked(backlogService.getFileContent).mockResolvedValue("Spec content");
  });

  const renderPage = (kind = "idea", name = "test-idea", tab?: string) => {
    useDetailSelectionStore.getState().selectBacklog(kind, name, tab);
    const search = tab ? `?tab=${tab}` : "";
    return render(
      <MemoryRouter initialEntries={[`/graph${search}`]}>
        <QueryClientProvider client={queryClient}>
          <BacklogDetailsPage />
        </QueryClientProvider>
      </MemoryRouter>
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

  it("shows close button to return to graph", async () => {
    vi.mocked(backlogService.get).mockResolvedValue(mockItem);
    vi.mocked(backlogService.getFiles).mockResolvedValue(mockFiles);

    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("backlog-details-back")).toBeInTheDocument();
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
    expect(screen.getAllByText("Post-run checks will run after execution").length).toBeGreaterThan(0);
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
