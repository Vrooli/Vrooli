import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { BacklogDetailsPage } from "./BacklogDetailsPage";
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

  const renderPage = (kind = "idea", name = "test-idea") => {
    return render(
      <MemoryRouter initialEntries={[`/backlog/${kind}/${name}`]}>
        <QueryClientProvider client={queryClient}>
          <Routes>
            <Route path="/backlog/:kind/:name" element={<BacklogDetailsPage />} />
          </Routes>
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

  it("shows breadcrumb navigation back to backlog list", async () => {
    vi.mocked(backlogService.get).mockResolvedValue(mockItem);
    vi.mocked(backlogService.getFiles).mockResolvedValue(mockFiles);

    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("backlog-details-back")).toBeInTheDocument();
    });

    expect(screen.getByText("Backlog")).toBeInTheDocument();
  });

  it("shows queue button for idea backlog items", async () => {
    vi.mocked(backlogService.get).mockResolvedValue(mockItem);
    vi.mocked(backlogService.getFiles).mockResolvedValue(mockFiles);

    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("backlog-details-queue")).toBeInTheDocument();
    });
  });

  it("hides queue button for research items", async () => {
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

    expect(screen.queryByTestId("backlog-details-queue")).not.toBeInTheDocument();
  });

  it("renders file tree", async () => {
    vi.mocked(backlogService.get).mockResolvedValue(mockItem);
    vi.mocked(backlogService.getFiles).mockResolvedValue(mockFiles);

    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("backlog-details-file-tree")).toBeInTheDocument();
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

    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("file-header-actions-trigger")).toBeInTheDocument();
    });
  });

  it("opens file header actions menu when trigger is clicked", async () => {
    vi.mocked(backlogService.get).mockResolvedValue(mockItem);
    vi.mocked(backlogService.getFiles).mockResolvedValue(mockFiles);

    renderPage();

    const trigger = await screen.findByTestId("file-header-actions-trigger");
    fireEvent.click(trigger);

    await waitFor(() => {
      expect(screen.getByTestId("file-header-actions-popover")).toBeInTheDocument();
      expect(screen.getByTestId("backlog-file-actions-menu")).toBeInTheDocument();
    });
  });
});
