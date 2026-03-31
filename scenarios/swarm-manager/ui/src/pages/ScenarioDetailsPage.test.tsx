import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { ScenarioDetailsPage } from "./ScenarioDetailsPage";
import { useScenariosStore, useDetailSelectionStore } from "../stores";

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

/**
 * Mock the config module for testing.
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

/**
 * Mock the services layer.
 */
vi.mock("../services", () => ({
  scenariosService: {
    list: vi.fn(),
    get: vi.fn(),
    getFiles: vi.fn(),
    updateMetadata: vi.fn(),
    delete: vi.fn(),
    start: vi.fn(),
    stop: vi.fn(),
    restart: vi.fn(),
  },
}));

import { scenariosService } from "../services";

const ARCHIVE_PREFERENCES_STORAGE_KEY = "swarm-manager.archive.preferences.v1";

// [REQ:REQ-P0-007] Test scenario details page functionality
describe("ScenarioDetailsPage", () => {
  let queryClient: QueryClient;

  const mockScenario = {
    name: "test-scenario",
    displayName: "Test Scenario",
    description: "A test scenario for unit testing",
    status: "running" as const,
    priority: 2,
    tags: ["api", "backend"],
    isGreenfield: false,
    completenessScore: 75,
  };

  beforeEach(() => {
    queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
        },
      },
    });
    vi.clearAllMocks();
    window.localStorage.clear();
    vi.mocked(scenariosService.getFiles).mockResolvedValue([]);
    useScenariosStore.getState().reset();
  });

  const renderPage = (scenarioName = "test-scenario") => {
    useDetailSelectionStore.getState().selectScenario(scenarioName);
    return render(
      <MemoryRouter initialEntries={["/graph"]}>
        <QueryClientProvider client={queryClient}>
          <ScenarioDetailsPage />
        </QueryClientProvider>
      </MemoryRouter>
    );
  };

  // [REQ:REQ-P0-007a] Test page structure
  describe("page structure", () => {
    it("renders the scenario details page container", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-details-page")).toBeInTheDocument();
      });
    });

    it("shows nav button in shared header", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("detail-nav-button")).toBeInTheDocument();
      });
    });
  });

  // [REQ:REQ-P0-007b] Test scenario metadata display
  describe("scenario metadata display", () => {
    it("displays scenario title correctly", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-details-title")).toHaveTextContent(
          "Test Scenario"
        );
      });
    });

    it("displays scenario description", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-details-description")).toHaveTextContent(
          "A test scenario for unit testing"
        );
      });
    });

    it("displays priority badge", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-details-priority")).toHaveTextContent(
          "P2"
        );
      });
    });

    it("displays tags", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      renderPage();

      await waitFor(() => {
        expect(screen.getAllByText("api").length).toBeGreaterThan(0);
        expect(screen.getAllByText("backend").length).toBeGreaterThan(0);
      });
    });

    it("displays completeness score when available", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      renderPage();

      await waitFor(() => {
        expect(screen.getAllByText("75%").length).toBeGreaterThan(0);
      });
    });

    it("displays fallback when no description provided", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue({
        ...mockScenario,
        description: "",
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getAllByText("No description provided").length).toBeGreaterThan(0);
      });
    });

    it("shows greenfield badge when scenario is greenfield", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue({
        ...mockScenario,
        isGreenfield: true,
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getAllByText("Greenfield").length).toBeGreaterThan(0);
      });
    });
  });

  // [REQ:REQ-P0-007c] Test loading states
  describe("loading states", () => {
    it("shows loading message while fetching scenario", () => {
      vi.mocked(scenariosService.get).mockImplementation(() => new Promise(() => {}));
      renderPage();

      expect(screen.getByTestId("scenario-details-loading-state")).toBeInTheDocument();
    });
  });

  // [REQ:REQ-P0-007d] Test error handling
  describe("error states", () => {
    it("shows error state when scenario fetch fails", async () => {
      vi.mocked(scenariosService.get).mockRejectedValue(new Error("API error"));
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("error-state")).toBeInTheDocument();
      });
      expect(screen.getByText("Unable to load scenario")).toBeInTheDocument();
    });

    it("provides retry button on error", async () => {
      vi.mocked(scenariosService.get).mockRejectedValue(new Error("Network error"));
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("error-retry")).toBeInTheDocument();
      });
    });

    it("retry button triggers refetch", async () => {
      vi.mocked(scenariosService.get)
        .mockRejectedValueOnce(new Error("Network error"))
        .mockResolvedValueOnce(mockScenario);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("error-state")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByTestId("error-retry"));

      await waitFor(() => {
        expect(screen.getByTestId("scenario-details-title")).toBeInTheDocument();
      });
      expect(scenariosService.get).toHaveBeenCalledTimes(2);
    });
  });

  // [REQ:REQ-P0-007e] Test metadata management section
  describe("metadata management", () => {
    it("renders metadata section", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-details-metadata")).toBeInTheDocument();
      });
      expect(screen.getAllByText("Scenario Settings").length).toBeGreaterThan(0);
    });

    it("shows greenfield toggle", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-greenfield-toggle")).toBeInTheDocument();
      });
      expect(screen.getAllByText("Greenfield Mode").length).toBeGreaterThan(0);
    });

    it("displays correct initial state for greenfield toggle", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue({
        ...mockScenario,
        isGreenfield: false,
      });
      renderPage();

      await waitFor(() => {
        const toggle = screen.getByTestId("scenario-greenfield-toggle");
        expect(toggle).toHaveTextContent("Disabled");
      });
    });

  });

  // [REQ:REQ-P0-007f] Test toggle interactions
  describe("toggle interactions", () => {
    it("calls updateMetadata when greenfield toggle is clicked", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue({
        ...mockScenario,
        isGreenfield: false,
      });
      vi.mocked(scenariosService.updateMetadata).mockResolvedValue({
        ...mockScenario,
        isGreenfield: true,
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-greenfield-toggle")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByTestId("scenario-greenfield-toggle"));

      await waitFor(() => {
        expect(scenariosService.updateMetadata).toHaveBeenCalledWith(
          "test-scenario",
          { isGreenfield: true }
        );
      });
    });

    it("shows optimistic update immediately on toggle", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue({
        ...mockScenario,
        isGreenfield: false,
      });
      // Slow response to test optimistic update
      vi.mocked(scenariosService.updateMetadata).mockImplementation(
        () =>
          new Promise((resolve) =>
            setTimeout(() => resolve({ ...mockScenario, isGreenfield: true }), 1000)
          )
      );
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-greenfield-toggle")).toHaveTextContent(
          "Disabled"
        );
      });

      fireEvent.click(screen.getByTestId("scenario-greenfield-toggle"));

      // Optimistic update should show "Enabled" immediately
      await waitFor(() => {
        expect(screen.getByTestId("scenario-greenfield-toggle")).toHaveTextContent(
          "Enabled"
        );
      });
    });

    it("shows error feedback when update fails", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      vi.mocked(scenariosService.updateMetadata).mockRejectedValue(
        new Error("Update failed")
      );
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-greenfield-toggle")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByTestId("scenario-greenfield-toggle"));

      await waitFor(() => {
        expect(screen.getAllByText("Failed to update settings. Please try again.").length).toBeGreaterThan(0);
      });
    });

    it("reverts to original value when update fails", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue({
        ...mockScenario,
        isGreenfield: false,
      });
      vi.mocked(scenariosService.updateMetadata).mockRejectedValue(
        new Error("Update failed")
      );
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-greenfield-toggle")).toHaveTextContent(
          "Disabled"
        );
      });

      fireEvent.click(screen.getByTestId("scenario-greenfield-toggle"));

      // After error, should revert to original value
      await waitFor(() => {
        expect(screen.getByTestId("scenario-greenfield-toggle")).toHaveTextContent(
          "Disabled"
        );
      });
    });
  });

  describe("scenario deletion", () => {
    it("renders delete button in danger zone", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-details-delete")).toBeInTheDocument();
      });
    });

    it("opens delete dialog with archive enabled by default", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      vi.mocked(scenariosService.getFiles).mockResolvedValue([
        { name: "PRD.md", path: "PRD.md", type: "file", size: 100 },
      ]);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-details-delete")).toBeInTheDocument();
      });
      fireEvent.click(screen.getByTestId("scenario-details-delete"));

      expect(screen.getByTestId("scenario-delete-dialog")).toBeInTheDocument();
      expect(screen.getByTestId("scenario-delete-archive")).toBeChecked();
      expect(screen.getByTestId("scenario-delete-confirm")).toBeDisabled();
      expect(screen.getByTestId("archive-preview-panel")).toBeInTheDocument();
      await waitFor(() => {
        expect(screen.getByTestId("archive-preview-count")).toHaveTextContent("1 files");
      });
      expect(screen.getByTestId("customize-files-link")).toBeInTheDocument();
    });

    it("requires exact scenario name before enabling delete confirmation", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-details-delete")).toBeInTheDocument();
      });
      fireEvent.click(screen.getByTestId("scenario-details-delete"));

      const confirmButton = screen.getByTestId("scenario-delete-confirm");
      const confirmInput = screen.getByPlaceholderText("test-scenario");

      expect(confirmButton).toBeDisabled();

      fireEvent.change(confirmInput, { target: { value: "wrong-name" } });
      expect(confirmButton).toBeDisabled();

      fireEvent.change(confirmInput, { target: { value: "test-scenario" } });
      expect(confirmButton).toBeEnabled();
    });

    it("calls delete with archive enabled by default", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      vi.mocked(scenariosService.getFiles).mockResolvedValue([
        { name: "PRD.md", path: "PRD.md", type: "file", size: 100 },
      ]);
      vi.mocked(scenariosService.delete).mockResolvedValue({
        name: "test-scenario",
        archived: true,
        message: "Scenario archived to backlog (idea) and deleted",
        preservedFiles: [],
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-details-delete")).toBeInTheDocument();
      });
      fireEvent.click(screen.getByTestId("scenario-details-delete"));
      fireEvent.change(screen.getByPlaceholderText("test-scenario"), {
        target: { value: "test-scenario" },
      });
      fireEvent.click(screen.getByTestId("scenario-delete-confirm"));

      await waitFor(() => {
        expect(scenariosService.delete).toHaveBeenCalledWith("test-scenario", {
          archive: true,
          preserveFiles: { preset: "planning" },
        });
      });
    });

    it("calls delete with archive disabled when checkbox is unchecked", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      vi.mocked(scenariosService.delete).mockResolvedValue({
        name: "test-scenario",
        archived: false,
        message: "Scenario permanently deleted",
        preservedFiles: [],
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-details-delete")).toBeInTheDocument();
      });
      fireEvent.click(screen.getByTestId("scenario-details-delete"));
      fireEvent.click(screen.getByTestId("scenario-delete-archive"));

      expect(screen.queryByTestId("customize-files-link")).not.toBeInTheDocument();

      fireEvent.change(screen.getByPlaceholderText("test-scenario"), {
        target: { value: "test-scenario" },
      });
      fireEvent.click(screen.getByTestId("scenario-delete-confirm"));

      await waitFor(() => {
        expect(scenariosService.delete).toHaveBeenCalledWith("test-scenario", {
          archive: false,
          preserveFiles: undefined,
        });
      });
    });

    it("updates preview and payload when preset changes", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      vi.mocked(scenariosService.getFiles).mockResolvedValue([
        { name: "PRD.md", path: "PRD.md", type: "file", size: 100 },
        { name: "README.md", path: "README.md", type: "file", size: 120 },
        { name: "service.json", path: ".vrooli/service.json", type: "file", size: 80 },
        { name: "README.md", path: "node_modules/pkg/README.md", type: "file", size: 50 },
      ]);
      vi.mocked(scenariosService.delete).mockResolvedValue({
        name: "test-scenario",
        archived: true,
        message: "Scenario archived to backlog (idea) and deleted",
        preservedFiles: [],
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-details-delete")).toBeInTheDocument();
      });
      fireEvent.click(screen.getByTestId("scenario-details-delete"));

      await waitFor(() => {
        expect(screen.getByTestId("archive-preview-count")).toHaveTextContent("2 files");
      });
      fireEvent.change(screen.getByTestId("archive-preset-select"), {
        target: { value: "documentation" },
      });
      await waitFor(() => {
        expect(screen.getByTestId("archive-preview-count")).toHaveTextContent("2 files");
      });

      fireEvent.change(screen.getByPlaceholderText("test-scenario"), {
        target: { value: "test-scenario" },
      });
      fireEvent.click(screen.getByTestId("scenario-delete-confirm"));

      await waitFor(() => {
        expect(scenariosService.delete).toHaveBeenCalledWith("test-scenario", {
          archive: true,
          preserveFiles: { preset: "documentation" },
        });
      });
    });

    it("uses persisted preset when modal opens", async () => {
      window.localStorage.setItem(ARCHIVE_PREFERENCES_STORAGE_KEY, JSON.stringify({
        mode: "preset",
        preset: "documentation",
        customPaths: [],
      }));
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      vi.mocked(scenariosService.getFiles).mockResolvedValue([
        { name: "PRD.md", path: "PRD.md", type: "file", size: 100 },
        { name: "README.md", path: "README.md", type: "file", size: 120 },
      ]);
      vi.mocked(scenariosService.delete).mockResolvedValue({
        name: "test-scenario",
        archived: true,
        message: "Scenario archived to backlog (idea) and deleted",
        preservedFiles: [],
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-details-delete")).toBeInTheDocument();
      });
      fireEvent.click(screen.getByTestId("scenario-details-delete"));

      await waitFor(() => {
        expect(screen.getByTestId("archive-preset-select")).toHaveValue("documentation");
      });

      fireEvent.change(screen.getByPlaceholderText("test-scenario"), {
        target: { value: "test-scenario" },
      });
      fireEvent.click(screen.getByTestId("scenario-delete-confirm"));

      await waitFor(() => {
        expect(scenariosService.delete).toHaveBeenCalledWith("test-scenario", {
          archive: true,
          preserveFiles: { preset: "documentation" },
        });
      });
    });

    it("ignores node_modules files when selecting documentation preset in file dialog", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      vi.mocked(scenariosService.getFiles).mockResolvedValue([
        { name: "PRD.md", path: "PRD.md", type: "file", size: 100 },
        { name: "README.md", path: "README.md", type: "file", size: 120 },
        { name: "README.md", path: "node_modules/pkg/README.md", type: "file", size: 120 },
      ]);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-details-delete")).toBeInTheDocument();
      });
      fireEvent.click(screen.getByTestId("scenario-details-delete"));
      fireEvent.click(screen.getByTestId("customize-files-link"));

      await waitFor(() => {
        expect(screen.getByTestId("file-selection-dialog")).toBeInTheDocument();
      });
      fireEvent.change(screen.getByTestId("preset-select"), {
        target: { value: "documentation" },
      });

      await waitFor(() => {
        expect(screen.getByTestId("confirm-selection-button")).toHaveTextContent("(2 files)");
      });
    });

    it("loads files for archive customization and sends custom preserveFiles paths", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      vi.mocked(scenariosService.getFiles).mockResolvedValue([
        { name: "PRD.md", path: "PRD.md", type: "file", size: 100 },
      ]);
      vi.mocked(scenariosService.delete).mockResolvedValue({
        name: "test-scenario",
        archived: true,
        message: "Scenario archived to backlog (idea) with 1 preserved files and deleted",
        preservedFiles: ["PRD.md"],
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-details-delete")).toBeInTheDocument();
      });
      fireEvent.click(screen.getByTestId("scenario-details-delete"));
      fireEvent.click(screen.getByTestId("customize-files-link"));

      await waitFor(() => {
        expect(scenariosService.getFiles).toHaveBeenCalledWith("test-scenario");
      });
      expect(screen.getByTestId("file-selection-dialog")).toBeInTheDocument();

      fireEvent.click(screen.getByTestId("select-all-button"));
      await waitFor(() => {
        expect(screen.getByTestId("confirm-selection-button")).toHaveTextContent("(1 files)");
      });
      fireEvent.click(screen.getByTestId("confirm-selection-button"));

      fireEvent.change(screen.getByPlaceholderText("test-scenario"), {
        target: { value: "test-scenario" },
      });
      fireEvent.click(screen.getByTestId("scenario-delete-confirm"));

      await waitFor(() => {
        expect(scenariosService.delete).toHaveBeenCalledWith("test-scenario", {
          archive: true,
          preserveFiles: {
            paths: ["PRD.md"],
          },
        });
      });
    });

    it("uses persisted custom selection when all selected files exist", async () => {
      window.localStorage.setItem(ARCHIVE_PREFERENCES_STORAGE_KEY, JSON.stringify({
        mode: "custom",
        preset: "documentation",
        customPaths: ["README.md", "PRD.md", "requirements/index.json", ".vrooli/service.json"],
      }));
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      vi.mocked(scenariosService.getFiles).mockResolvedValue([
        { name: "README.md", path: "README.md", type: "file", size: 100 },
        { name: "PRD.md", path: "PRD.md", type: "file", size: 120 },
        { name: "index.json", path: "requirements/index.json", type: "file", size: 80 },
        { name: "service.json", path: ".vrooli/service.json", type: "file", size: 80 },
      ]);
      vi.mocked(scenariosService.delete).mockResolvedValue({
        name: "test-scenario",
        archived: true,
        message: "Scenario archived to backlog (idea) and deleted",
        preservedFiles: [],
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-details-delete")).toBeInTheDocument();
      });
      fireEvent.click(screen.getByTestId("scenario-details-delete"));

      await waitFor(() => {
        expect(screen.getByTestId("archive-preview-count")).toHaveTextContent("4 files");
      });

      fireEvent.change(screen.getByPlaceholderText("test-scenario"), {
        target: { value: "test-scenario" },
      });
      fireEvent.click(screen.getByTestId("scenario-delete-confirm"));

      await waitFor(() => {
        expect(scenariosService.delete).toHaveBeenCalledWith("test-scenario", {
          archive: true,
          preserveFiles: {
            paths: ["README.md", "PRD.md", "requirements/index.json", ".vrooli/service.json"],
          },
        });
      });
    });

    it("falls back to persisted preset when custom selection does not match scenario files", async () => {
      window.localStorage.setItem(ARCHIVE_PREFERENCES_STORAGE_KEY, JSON.stringify({
        mode: "custom",
        preset: "documentation",
        customPaths: ["README.md", "PRD.md", "requirements/index.json", ".vrooli/service.json"],
      }));
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      vi.mocked(scenariosService.getFiles).mockResolvedValue([
        { name: "README.md", path: "README.md", type: "file", size: 100 },
        { name: "PRD.md", path: "PRD.md", type: "file", size: 120 },
      ]);
      vi.mocked(scenariosService.delete).mockResolvedValue({
        name: "test-scenario",
        archived: true,
        message: "Scenario archived to backlog (idea) and deleted",
        preservedFiles: [],
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-details-delete")).toBeInTheDocument();
      });
      fireEvent.click(screen.getByTestId("scenario-details-delete"));

      await waitFor(() => {
        expect(screen.getByTestId("archive-preview-count")).toHaveTextContent("2 files");
      });
      expect(screen.getByText("Included Files")).toBeInTheDocument();

      fireEvent.change(screen.getByPlaceholderText("test-scenario"), {
        target: { value: "test-scenario" },
      });
      fireEvent.click(screen.getByTestId("scenario-delete-confirm"));

      await waitFor(() => {
        expect(scenariosService.delete).toHaveBeenCalledWith("test-scenario", {
          archive: true,
          preserveFiles: { preset: "documentation" },
        });
      });
    });

    it("reopens custom file selection with previous files preselected", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      vi.mocked(scenariosService.getFiles).mockResolvedValue([
        { name: "PRD.md", path: "PRD.md", type: "file", size: 100 },
      ]);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-details-delete")).toBeInTheDocument();
      });
      fireEvent.click(screen.getByTestId("scenario-details-delete"));
      fireEvent.click(screen.getByTestId("customize-files-link"));

      await waitFor(() => {
        expect(screen.getByTestId("file-selection-dialog")).toBeInTheDocument();
      });
      fireEvent.click(screen.getByTestId("select-all-button"));
      await waitFor(() => {
        expect(screen.getByTestId("confirm-selection-button")).toHaveTextContent("(1 files)");
      });
      fireEvent.click(screen.getByTestId("confirm-selection-button"));

      fireEvent.click(screen.getByTestId("customize-files-link"));

      await waitFor(() => {
        expect(screen.getByText("1 of 1 files selected")).toBeInTheDocument();
      });
      expect(screen.getByTestId("confirm-selection-button")).toHaveTextContent("(1 files)");
    });
  });

  // [REQ:REQ-P0-007g] Test status display
  describe("status display", () => {
    it("displays running status correctly", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue({
        ...mockScenario,
        status: "running" as const,
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getAllByText("Running").length).toBeGreaterThan(0);
      });
    });

    it("displays stopped status correctly", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue({
        ...mockScenario,
        status: "stopped" as const,
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getAllByText("Stopped").length).toBeGreaterThan(0);
      });
    });

    it("displays error status correctly", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue({
        ...mockScenario,
        status: "error" as const,
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getAllByText("Error").length).toBeGreaterThan(0);
      });
    });
  });

  describe("scenario actions", () => {
    it("renders start/stop/restart buttons", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-details-start")).toBeInTheDocument();
      });
      expect(screen.getByTestId("scenario-details-stop")).toBeInTheDocument();
      expect(screen.getByTestId("scenario-details-restart")).toBeInTheDocument();
    });

    it("calls start when start button is clicked", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue({
        ...mockScenario,
        status: "stopped" as const,
      });
      vi.mocked(scenariosService.start).mockResolvedValue({
        ...mockScenario,
        status: "running" as const,
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-details-start")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByTestId("scenario-details-start"));

      await waitFor(() => {
        expect(scenariosService.start).toHaveBeenCalledWith("test-scenario");
      });
    });

    it("opens and closes mobile actions sheet", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      renderPage();

      await waitFor(() => {
        expect(screen.getByLabelText("Open scenario actions")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByLabelText("Open scenario actions"));
      expect(screen.getByTestId("scenario-mobile-actions-sheet")).toBeInTheDocument();

      fireEvent.click(screen.getByLabelText("Close dialog"));
      await waitFor(() => {
        expect(screen.queryByTestId("scenario-mobile-actions-sheet")).not.toBeInTheDocument();
      });
    });

    it("runs start from mobile actions sheet", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue({
        ...mockScenario,
        status: "stopped" as const,
      });
      vi.mocked(scenariosService.start).mockResolvedValue({
        ...mockScenario,
        status: "running" as const,
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getByLabelText("Open scenario actions")).toBeInTheDocument();
      });
      fireEvent.click(screen.getByLabelText("Open scenario actions"));

      const actionsSheet = screen.getByTestId("scenario-mobile-actions-sheet");
      fireEvent.click(within(actionsSheet).getByRole("button", { name: "Start" }));

      await waitFor(() => {
        expect(scenariosService.start).toHaveBeenCalledWith("test-scenario");
      });
    });
  });

  describe("mobile danger zone", () => {
    it("starts collapsed and expands on demand", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      renderPage();

      await waitFor(() => {
        expect(screen.getByRole("button", { name: /Danger Zone/i })).toBeInTheDocument();
      });
      expect(screen.queryByRole("button", { name: "Delete Scenario" })).not.toBeInTheDocument();

      fireEvent.click(screen.getByRole("button", { name: /Danger Zone/i }));
      expect(screen.getByRole("button", { name: "Delete Scenario" })).toBeInTheDocument();
    });
  });

  // Edge cases
  describe("edge cases", () => {
    it("shows error when rendered without name in selection", async () => {
      // Clear selection to simulate no name
      useDetailSelectionStore.getState().clearSelection();
      render(
        <MemoryRouter initialEntries={["/graph"]}>
          <QueryClientProvider client={queryClient}>
            <ScenarioDetailsPage />
          </QueryClientProvider>
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(screen.getByTestId("scenario-details-page")).toBeInTheDocument();
      });
      expect(screen.getByText("Invalid URL")).toBeInTheDocument();
    });

    it("handles scenario without completeness score", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue({
        ...mockScenario,
        completenessScore: undefined,
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-details-title")).toBeInTheDocument();
      });
      // Should not crash and score should not appear
      expect(screen.queryByText("%")).not.toBeInTheDocument();
    });
  });
});
