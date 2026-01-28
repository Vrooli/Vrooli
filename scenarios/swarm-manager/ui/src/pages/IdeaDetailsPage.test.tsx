import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { IdeaDetailsPage } from "./IdeaDetailsPage";

/**
 * Mock the config module for testing.
 * This allows tests to control configuration values without retries.
 */
vi.mock("../config", () => ({
  dataFetchingConfig: {
    retryCount: 0,
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
    getFiles: vi.fn(),
    getFileContent: vi.fn(),
    uploadFile: vi.fn(),
    queue: vi.fn(),
    research: vi.fn(),
  },
}));

import { ideasService } from "../services";

// [REQ:REQ-P0-004] Test idea details page functionality
describe("IdeaDetailsPage", () => {
  let queryClient: QueryClient;

  const mockIdea = {
    name: "test-idea",
    title: "Test Idea Title",
    description: "A detailed description of the test idea",
    status: "backlog" as const,
    priority: 2,
    tags: ["feature", "api"],
    created: "2026-01-20T10:00:00Z",
    updated: "2026-01-25T15:30:00Z",
  };

  const mockFiles = [
    {
      name: "spec.json",
      path: "spec.json",
      type: "file" as const,
      size: 256,
    },
    {
      name: "notes",
      path: "notes",
      type: "directory" as const,
      children: [
        {
          name: "research.md",
          path: "notes/research.md",
          type: "file" as const,
          size: 1024,
        },
      ],
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
  });

  const renderPage = (ideaName = "test-idea") => {
    return render(
      <MemoryRouter initialEntries={[`/ideas/${ideaName}`]}>
        <QueryClientProvider client={queryClient}>
          <Routes>
            <Route path="/ideas/:name" element={<IdeaDetailsPage />} />
          </Routes>
        </QueryClientProvider>
      </MemoryRouter>
    );
  };

  // [REQ:REQ-P0-004a] Test page renders with correct structure
  describe("page structure", () => {
    it("renders the idea details page container", async () => {
      vi.mocked(ideasService.get).mockResolvedValue(mockIdea);
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("idea-details-page")).toBeInTheDocument();
      });
    });

    it("shows breadcrumb navigation back to ideas list", async () => {
      vi.mocked(ideasService.get).mockResolvedValue(mockIdea);
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("idea-details-back")).toBeInTheDocument();
      });
      // Breadcrumb shows "Ideas" link (Phase 29 experience improvement)
      expect(screen.getByText("Ideas")).toBeInTheDocument();
    });
  });

  // [REQ:REQ-P0-004b] Test idea metadata display
  describe("idea metadata display", () => {
    it("displays idea title correctly", async () => {
      vi.mocked(ideasService.get).mockResolvedValue(mockIdea);
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("idea-details-title")).toHaveTextContent(
          "Test Idea Title"
        );
      });
    });

    it("displays idea description", async () => {
      vi.mocked(ideasService.get).mockResolvedValue(mockIdea);
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("idea-details-description")).toHaveTextContent(
          "A detailed description of the test idea"
        );
      });
    });

    it("displays formatted status", async () => {
      vi.mocked(ideasService.get).mockResolvedValue(mockIdea);
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        // formatIdeaStatus capitalizes: "backlog" -> "Backlog"
        expect(screen.getByText("Backlog")).toBeInTheDocument();
      });
    });

    it("displays priority badge", async () => {
      vi.mocked(ideasService.get).mockResolvedValue(mockIdea);
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        expect(screen.getByText("Priority 2")).toBeInTheDocument();
      });
    });

    it("displays tags", async () => {
      vi.mocked(ideasService.get).mockResolvedValue(mockIdea);
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        expect(screen.getByText("feature")).toBeInTheDocument();
        expect(screen.getByText("api")).toBeInTheDocument();
      });
    });

    it("displays fallback when no description provided", async () => {
      vi.mocked(ideasService.get).mockResolvedValue({
        ...mockIdea,
        description: "",
      });
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        expect(screen.getByText("No description provided")).toBeInTheDocument();
      });
    });
  });

  // [REQ:REQ-P0-004c] Test loading states
  describe("loading states", () => {
    it("shows loading message while fetching idea", () => {
      vi.mocked(ideasService.get).mockImplementation(() => new Promise(() => {}));
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      expect(screen.getByText("Loading idea details...")).toBeInTheDocument();
    });

    it("shows loading message for files section", async () => {
      vi.mocked(ideasService.get).mockResolvedValue(mockIdea);
      vi.mocked(ideasService.getFiles).mockImplementation(() => new Promise(() => {}));
      renderPage();

      await waitFor(() => {
        expect(screen.getByText("Loading files...")).toBeInTheDocument();
      });
    });
  });

  // [REQ:REQ-P0-004d] Test error handling
  describe("error states", () => {
    it("shows error state when idea fetch fails", async () => {
      vi.mocked(ideasService.get).mockRejectedValue(new Error("API error"));
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("error-state")).toBeInTheDocument();
      });
      expect(screen.getByText("Unable to load idea")).toBeInTheDocument();
    });

    it("shows error state when files fetch fails", async () => {
      vi.mocked(ideasService.get).mockResolvedValue(mockIdea);
      vi.mocked(ideasService.getFiles).mockRejectedValue(new Error("Files error"));
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("error-state")).toBeInTheDocument();
      });
    });

    it("provides retry button on error", async () => {
      vi.mocked(ideasService.get).mockRejectedValue(new Error("Network error"));
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("error-retry")).toBeInTheDocument();
      });
    });

    it("retry button triggers refetch", async () => {
      vi.mocked(ideasService.get)
        .mockRejectedValueOnce(new Error("Network error"))
        .mockResolvedValueOnce(mockIdea);
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("error-state")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByTestId("error-retry"));

      await waitFor(() => {
        expect(screen.getByTestId("idea-details-title")).toBeInTheDocument();
      });
      expect(ideasService.get).toHaveBeenCalledTimes(2);
    });
  });

  // [REQ:REQ-P0-004e] Test file tree functionality
  describe("file tree", () => {
    it("renders file tree component", async () => {
      vi.mocked(ideasService.get).mockResolvedValue(mockIdea);
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("idea-details-file-tree")).toBeInTheDocument();
      });
    });

    it("shows prompt to select file when none selected", async () => {
      vi.mocked(ideasService.get).mockResolvedValue(mockIdea);
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        expect(
          screen.getByText("Click on a file in the tree to preview its contents")
        ).toBeInTheDocument();
      });
    });
  });

  // [REQ:REQ-P0-004f] Test upload toggle
  describe("file upload", () => {
    it("shows upload toggle button", async () => {
      vi.mocked(ideasService.get).mockResolvedValue(mockIdea);
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("toggle-upload")).toBeInTheDocument();
      });
      expect(screen.getByText("Upload Files")).toBeInTheDocument();
    });

    it("toggles upload area visibility", async () => {
      vi.mocked(ideasService.get).mockResolvedValue(mockIdea);
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("toggle-upload")).toBeInTheDocument();
      });

      // Initially upload area not visible
      expect(screen.queryByTestId("file-upload-dropzone")).not.toBeInTheDocument();

      // Click to show
      fireEvent.click(screen.getByTestId("toggle-upload"));
      expect(screen.getByTestId("file-upload-dropzone")).toBeInTheDocument();
      expect(screen.getByText("Hide Upload")).toBeInTheDocument();

      // Click to hide
      fireEvent.click(screen.getByTestId("toggle-upload"));
      expect(screen.queryByTestId("file-upload-dropzone")).not.toBeInTheDocument();
    });
  });

  // [REQ:REQ-P0-005] Test queue functionality
  describe("queue for processing", () => {
    it("shows queue button for queueable statuses", async () => {
      vi.mocked(ideasService.get).mockResolvedValue({
        ...mockIdea,
        status: "ready" as const,
      });
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("idea-details-queue")).toBeInTheDocument();
      });
      expect(screen.getByText("Queue for Processing")).toBeInTheDocument();
    });

    it("hides queue button for non-queueable statuses", async () => {
      vi.mocked(ideasService.get).mockResolvedValue({
        ...mockIdea,
        status: "queued" as const,
      });
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("idea-details-header")).toBeInTheDocument();
      });
      expect(screen.queryByTestId("idea-details-queue")).not.toBeInTheDocument();
    });

    it("shows queue button for backlog status", async () => {
      vi.mocked(ideasService.get).mockResolvedValue({
        ...mockIdea,
        status: "backlog" as const,
      });
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("idea-details-queue")).toBeInTheDocument();
      });
    });

    it("shows queue button for researching status", async () => {
      vi.mocked(ideasService.get).mockResolvedValue({
        ...mockIdea,
        status: "researching" as const,
      });
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("idea-details-queue")).toBeInTheDocument();
      });
    });

    it("hides queue button for in_progress status", async () => {
      vi.mocked(ideasService.get).mockResolvedValue({
        ...mockIdea,
        status: "in_progress" as const,
      });
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("idea-details-header")).toBeInTheDocument();
      });
      expect(screen.queryByTestId("idea-details-queue")).not.toBeInTheDocument();
    });

    it("hides queue button for completed status", async () => {
      vi.mocked(ideasService.get).mockResolvedValue({
        ...mockIdea,
        status: "completed" as const,
      });
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("idea-details-header")).toBeInTheDocument();
      });
      expect(screen.queryByTestId("idea-details-queue")).not.toBeInTheDocument();
    });

    it("calls queue service when queue button clicked", async () => {
      vi.mocked(ideasService.get).mockResolvedValue({
        ...mockIdea,
        status: "ready" as const,
      });
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      vi.mocked(ideasService.queue).mockResolvedValue({
        idea: { ...mockIdea, status: "queued" as const },
        taskId: "task-123",
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("idea-details-queue")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByTestId("idea-details-queue"));

      await waitFor(() => {
        expect(ideasService.queue).toHaveBeenCalledWith("test-idea");
      });
    });
  });

  // [REQ:REQ-P0-004g] Test action buttons
  describe("action buttons", () => {
    it("shows edit button", async () => {
      vi.mocked(ideasService.get).mockResolvedValue(mockIdea);
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("idea-details-edit")).toBeInTheDocument();
      });
    });

    it("shows delete button", async () => {
      vi.mocked(ideasService.get).mockResolvedValue(mockIdea);
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("idea-details-delete")).toBeInTheDocument();
      });
    });

    it("opens research dialog when research button clicked", async () => {
      vi.mocked(ideasService.get).mockResolvedValue(mockIdea);
      vi.mocked(ideasService.getFiles).mockResolvedValue(mockFiles);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("idea-details-research")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByTestId("idea-details-research"));

      await waitFor(() => {
        expect(screen.getByTestId("idea-research-dialog")).toBeInTheDocument();
        expect(screen.getByTestId("idea-research-prompt")).toBeInTheDocument();
      });
    });
  });

  // Test edge case: missing name parameter
  describe("edge cases", () => {
    it("shows error when rendered without name parameter", async () => {
      render(
        <MemoryRouter initialEntries={["/ideas/"]}>
          <QueryClientProvider client={queryClient}>
            <Routes>
              <Route path="/ideas/" element={<IdeaDetailsPage />} />
            </Routes>
          </QueryClientProvider>
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(screen.getByTestId("idea-details-page")).toBeInTheDocument();
      });
      expect(screen.getByText("Invalid URL")).toBeInTheDocument();
    });
  });
});
