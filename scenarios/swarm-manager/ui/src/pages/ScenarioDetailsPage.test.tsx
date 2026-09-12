import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor, fireEvent, within } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";
import { Route, Routes } from "react-router-dom";
import { ScenarioDetailsPage } from "./ScenarioDetailsPage";
import { useScenariosStore } from "../stores";
import { createTestQueryClient, installMatchMediaMock, renderWithProviders } from "../test-utils";

// jsdom doesn't provide matchMedia (needed by useIsMobile in DetailPageLayout).
beforeAll(() => {
  installMatchMediaMock();
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
  uiBehaviorConfig: {
    searchDebounceMs: 300,
    toastDurationMs: 5000,
  },
}));

/**
 * Mock the services layer.
 */
vi.mock("../services", () => ({
  scenariosService: {
    list: vi.fn(),
    get: vi.fn(),
    getContext: vi.fn(),
    getFiles: vi.fn(),
    updateMetadata: vi.fn(),
    delete: vi.fn(),
    start: vi.fn(),
    stop: vi.fn(),
    restart: vi.fn(),
    previewRemediation: vi.fn(),
    applyRemediation: vi.fn(),
    previewMaturityCampaign: vi.fn(),
    applyMaturityCampaign: vi.fn(),
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
    queryClient = createTestQueryClient();
    vi.clearAllMocks();
    window.localStorage.clear();
    vi.mocked(scenariosService.getFiles).mockResolvedValue([]);
    vi.mocked(scenariosService.getContext).mockResolvedValue({
      scenarioName: "test-scenario",
      goals: [],
      orphanItems: [],
      rollup: { total: 0, completed: 0, inProgress: 0, failed: 0, pending: 0, archived: 0 },
      fixes: { active: [], archived: [] },
    });
    useScenariosStore.getState().reset();
  });

  const renderPage = (scenarioName = "test-scenario") => {
    return renderWithProviders(
      <Routes>
        <Route path="/scenarios/:name" element={<ScenarioDetailsPage />} />
        <Route path="/graph" element={<div data-testid="graph-route" />} />
      </Routes>,
      {
        queryClient,
        initialEntries: [`/scenarios/${scenarioName}`],
      },
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
    it("previews before it applies a fresh phase remediation", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue({ ...mockScenario, health: { evidenceState: "fresh", phases: [{ phase: "unit", priorityCapabilityId: "coverage", priorityCapabilityLabel: "Coverage", blockingCodes: [] }] } as never });
      vi.mocked(scenariosService.previewRemediation).mockResolvedValue({ proposal: { target: { scenarioName: "test-scenario", providerPhase: "unit", capabilityId: "coverage" }, fingerprint: "srh:test", title: "Improve coverage", description: "Preview", acceptanceCriteria: ["Given evidence"], acceptanceAllow: [], recommendedWorkflows: [] } });
      renderPage();
      await waitFor(() => expect(screen.getAllByTestId("scenario-detail-tab-quality")).not.toHaveLength(0));
      fireEvent.click(screen.getAllByTestId("scenario-detail-tab-quality").at(-1)!);
      await waitFor(() => expect(screen.getByRole("button", { name: "Preview remediation" })).toBeInTheDocument());
      fireEvent.click(screen.getByRole("button", { name: "Preview remediation" }));
      await waitFor(() => expect(screen.getByTestId("scenario-remediation-preview")).toBeInTheDocument());
      expect(scenariosService.applyRemediation).not.toHaveBeenCalled();
    });

    it("requires a separate preview and confirmation for a maturity campaign", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue({ ...mockScenario, health: { evidenceState: "fresh", phases: [{ phase: "unit", priorityCapabilityId: "coverage", priorityCapabilityLabel: "Coverage", blockingCodes: [] }] } as never });
      vi.mocked(scenariosService.previewMaturityCampaign).mockResolvedValue({ proposal: { target: { scenarioName: "test-scenario", maturityTarget: "operator-selected maturity outcome", providerPhases: ["unit"] }, fingerprint: "smc:test", title: "Raise maturity", description: "Preview", acceptanceCriteria: ["Given evidence"], declaredWorkflow: "scenario-improvement-campaign", trackerAvailability: "unavailable" } });
      vi.mocked(scenariosService.applyMaturityCampaign).mockResolvedValue({ proposal: { target: { scenarioName: "test-scenario", maturityTarget: "operator-selected maturity outcome", providerPhases: ["unit"] }, fingerprint: "smc:test", title: "Raise maturity", description: "Preview", acceptanceCriteria: ["Given evidence"], declaredWorkflow: "scenario-improvement-campaign", trackerAvailability: "unavailable" }, goalRef: "scenario-maturity-test", created: true, trackerAvailability: "unavailable" });
      renderPage();

      await waitFor(() => expect(screen.getAllByTestId("scenario-detail-tab-quality")).not.toHaveLength(0));
      fireEvent.click(screen.getAllByTestId("scenario-detail-tab-quality").at(-1)!);
      await waitFor(() => expect(screen.getByTestId("scenario-maturity-campaign-preview-button")).toBeInTheDocument());
      fireEvent.click(screen.getByTestId("scenario-maturity-campaign-preview-button"));
      await waitFor(() => expect(screen.getByTestId("scenario-maturity-campaign-preview")).toBeInTheDocument());
      expect(scenariosService.applyMaturityCampaign).not.toHaveBeenCalled();

      fireEvent.click(screen.getByTestId("scenario-maturity-campaign-confirm-button"));
      await waitFor(() => expect(scenariosService.applyMaturityCampaign).toHaveBeenCalledTimes(1));
      expect(screen.getByText("Governed goal: scenario-maturity-test")).toBeInTheDocument();
    });

    it("renders provider-owned stale health without claiming a verdict", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue({
        ...mockScenario,
        health: { evidenceState: "stale", reason: "Evidence is older than the freshness window.", sourceRunId: "run-42", phases: [] } as never,
      });
      renderPage();

      await waitFor(() => expect(screen.getAllByTestId("scenario-detail-tab-quality")).not.toHaveLength(0));
      fireEvent.click(screen.getAllByTestId("scenario-detail-tab-quality").at(-1)!);

      await waitFor(() => {
        expect(screen.getByTestId("scenario-health-section")).toBeInTheDocument();
        expect(screen.getByTestId("scenario-health-state")).toHaveTextContent("stale");
        expect(screen.getByTestId("scenario-health-reason")).toHaveTextContent("older than the freshness window");
      });
    });

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

      await waitFor(() => expect(screen.getAllByTestId("scenario-detail-tab-work")).not.toHaveLength(0));
      fireEvent.click(screen.getAllByTestId("scenario-detail-tab-work").at(-1)!);

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
      await waitFor(() => {
        expect(screen.getByText("No files selected for archive.")).toBeInTheDocument();
      });

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
        expect(screen.getAllByTestId("scenario-details-start").length).toBeGreaterThan(0);
      });
      expect(screen.getAllByTestId("scenario-details-stop").length).toBeGreaterThan(0);
      expect(screen.getAllByTestId("scenario-details-restart").length).toBeGreaterThan(0);
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
        expect(screen.getAllByTestId("scenario-details-start").length).toBeGreaterThan(0);
      });

      const startButtons = screen.getAllByTestId("scenario-details-start");
      fireEvent.click(startButtons[0] as HTMLElement);

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

      await waitFor(() => expect(screen.getAllByTestId("scenario-detail-tab-manage")).not.toHaveLength(0));
      fireEvent.click(screen.getAllByTestId("scenario-detail-tab-manage")[0]!);

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
      renderWithProviders(
        <Routes>
          <Route path="/graph" element={<ScenarioDetailsPage />} />
        </Routes>,
        {
          queryClient,
          initialEntries: ["/graph"],
        },
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

  describe("associated goals & backlog coverage", () => {
    it("renders the coverage section heading on desktop", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      renderPage();

      await waitFor(() => expect(screen.getAllByTestId("scenario-detail-tab-work")).not.toHaveLength(0));
      fireEvent.click(screen.getAllByTestId("scenario-detail-tab-work").at(-1)!);

      await waitFor(() => {
        expect(screen.getByTestId("scenario-coverage-section")).toBeInTheDocument();
      });
      expect(screen.getByText("Associated Goals & Backlog")).toBeInTheDocument();
    });

    it("renders goals subsection when context has goals", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      vi.mocked(scenariosService.getContext).mockResolvedValue({
        scenarioName: "test-scenario",
        goals: [
          {
            name: "audio-platform",
            title: "Audio Platform",
            status: "active",
            priority: 3,
            rollup: { total: 3, completed: 1, inProgress: 1, failed: 0, pending: 1, archived: 0 },
          },
        ],
        orphanItems: [],
        rollup: { total: 3, completed: 1, inProgress: 1, failed: 0, pending: 1, archived: 0 },
        fixes: { active: [], archived: [] },
      });
      renderPage();

      await waitFor(() => expect(screen.getAllByTestId("scenario-detail-tab-work")).not.toHaveLength(0));
      fireEvent.click(screen.getAllByTestId("scenario-detail-tab-work").at(-1)!);

      await waitFor(() => {
        expect(screen.getByTestId("scenario-coverage-goals")).toBeInTheDocument();
      });
      expect(screen.getByText("Audio Platform")).toBeInTheDocument();
      // Orphans subsection should not render when empty.
      expect(screen.queryByTestId("scenario-coverage-orphans")).not.toBeInTheDocument();
    });

    it("renders orphan items subsection when context has orphans", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      vi.mocked(scenariosService.getContext).mockResolvedValue({
        scenarioName: "test-scenario",
        goals: [],
        orphanItems: [
          {
            kind: "execute",
            name: "orphan-one",
            title: "Orphan One",
            status: "backlog",
            priority: 3,
          },
        ],
        rollup: { total: 1, completed: 0, inProgress: 0, failed: 0, pending: 1, archived: 0 },
        fixes: { active: [], archived: [] },
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-coverage-orphans")).toBeInTheDocument();
      });
      expect(screen.getByText("Orphan One")).toBeInTheDocument();
      // Goals subsection should not render when empty.
      expect(screen.queryByTestId("scenario-coverage-goals")).not.toBeInTheDocument();
    });

    it("renders empty-state copy when no coverage exists", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      // Default getContext mock in beforeEach returns empty — confirm the empty state renders.
      renderPage();

      await waitFor(() => expect(screen.getAllByTestId("scenario-detail-tab-work")).not.toHaveLength(0));
      fireEvent.click(screen.getAllByTestId("scenario-detail-tab-work").at(-1)!);

      await waitFor(() => {
        expect(screen.getByTestId("scenario-coverage-empty")).toBeInTheDocument();
      });
    });

    it("renders Fix History section with active partition by default", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      vi.mocked(scenariosService.getContext).mockResolvedValue({
        scenarioName: "test-scenario",
        goals: [],
        orphanItems: [],
        rollup: { total: 0, completed: 0, inProgress: 0, failed: 0, pending: 0, archived: 0 },
        fixes: {
          active: [
            { name: "fix-active", title: "Active fix", status: "backlog", priority: 2, path: "fix/fix-active" },
          ],
          archived: [
            { name: "fix-old", title: "Old fix", status: "completed", priority: 1, archivedAt: "2026-04-15T10:00:00Z", path: "fix/fix-old" },
          ],
        },
      });
      renderPage();

      await waitFor(() => expect(screen.getAllByTestId("scenario-detail-tab-work")).not.toHaveLength(0));
      fireEvent.click(screen.getAllByTestId("scenario-detail-tab-work").at(-1)!);

      await waitFor(() => {
        expect(screen.getByTestId("scenario-fix-history")).toBeInTheDocument();
      });
      // Default scope=active: shows the active fix, hides the archived one.
      expect(screen.getByText("Active fix")).toBeInTheDocument();
      expect(screen.queryByText("Old fix")).not.toBeInTheDocument();
    });

    it("toggling Archived scope shows archived fixes only", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      vi.mocked(scenariosService.getContext).mockResolvedValue({
        scenarioName: "test-scenario",
        goals: [],
        orphanItems: [],
        rollup: { total: 0, completed: 0, inProgress: 0, failed: 0, pending: 0, archived: 0 },
        fixes: {
          active: [
            { name: "fix-active", title: "Active fix", status: "backlog", priority: 2, path: "fix/fix-active" },
          ],
          archived: [
            { name: "fix-old", title: "Old fix", status: "completed", priority: 1, archivedAt: "2026-04-15T10:00:00Z", path: "fix/fix-old" },
          ],
        },
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-fix-history")).toBeInTheDocument();
      });

      const archivedToggle = screen.getByTestId("scenario-fix-history-scope-archived");
      fireEvent.click(archivedToggle);

      await waitFor(() => {
        expect(screen.getByText("Old fix")).toBeInTheDocument();
      });
      expect(screen.queryByText("Active fix")).not.toBeInTheDocument();
    });

    it("search narrows fix history by title", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      vi.mocked(scenariosService.getContext).mockResolvedValue({
        scenarioName: "test-scenario",
        goals: [],
        orphanItems: [],
        rollup: { total: 0, completed: 0, inProgress: 0, failed: 0, pending: 0, archived: 0 },
        fixes: {
          active: [
            { name: "fix-foo", title: "Foo crashes", status: "backlog", priority: 2, path: "fix/fix-foo" },
            { name: "fix-bar", title: "Bar timeout", status: "backlog", priority: 1, path: "fix/fix-bar" },
          ],
          archived: [],
        },
      });
      renderPage();

      const search = await screen.findByTestId("scenario-fix-history-search");
      fireEvent.change(search, { target: { value: "foo" } });

      await waitFor(() => {
        expect(screen.queryByText("Bar timeout")).not.toBeInTheDocument();
      });
      expect(screen.getByText("Foo crashes")).toBeInTheDocument();
    });

    it("renders Fix History section exactly once across mobile and desktop layouts", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-fix-history")).toBeInTheDocument();
      });
      // Single instance — no duplicate rendering across layouts.
      expect(screen.getAllByTestId("scenario-fix-history")).toHaveLength(1);
    });
  });
});
