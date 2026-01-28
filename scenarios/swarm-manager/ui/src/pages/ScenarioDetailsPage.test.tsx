import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { ScenarioDetailsPage } from "./ScenarioDetailsPage";

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
 * Mock the services layer.
 */
vi.mock("../services", () => ({
  scenariosService: {
    list: vi.fn(),
    get: vi.fn(),
    updateMetadata: vi.fn(),
  },
}));

import { scenariosService } from "../services";

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
    recommendationsEnabled: true,
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
  });

  const renderPage = (scenarioName = "test-scenario") => {
    return render(
      <MemoryRouter initialEntries={[`/scenarios/${scenarioName}`]}>
        <QueryClientProvider client={queryClient}>
          <Routes>
            <Route path="/scenarios/:name" element={<ScenarioDetailsPage />} />
          </Routes>
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

    it("shows breadcrumb navigation back to scenarios list", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-details-back")).toBeInTheDocument();
      });
      // Breadcrumb shows "Scenarios" link (Phase 29 experience improvement)
      expect(screen.getByText("Scenarios")).toBeInTheDocument();
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
        expect(screen.getByText("api")).toBeInTheDocument();
        expect(screen.getByText("backend")).toBeInTheDocument();
      });
    });

    it("displays completeness score when available", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      renderPage();

      await waitFor(() => {
        expect(screen.getByText("75%")).toBeInTheDocument();
      });
    });

    it("displays fallback when no description provided", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue({
        ...mockScenario,
        description: "",
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getByText("No description provided")).toBeInTheDocument();
      });
    });

    it("shows greenfield badge when scenario is greenfield", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue({
        ...mockScenario,
        isGreenfield: true,
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getByText("Greenfield")).toBeInTheDocument();
      });
    });
  });

  // [REQ:REQ-P0-007c] Test loading states
  describe("loading states", () => {
    it("shows loading message while fetching scenario", () => {
      vi.mocked(scenariosService.get).mockImplementation(() => new Promise(() => {}));
      renderPage();

      expect(screen.getByText("Loading scenario details...")).toBeInTheDocument();
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
      expect(screen.getByText("Scenario Settings")).toBeInTheDocument();
    });

    it("shows greenfield toggle", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-greenfield-toggle")).toBeInTheDocument();
      });
      expect(screen.getByText("Greenfield Mode")).toBeInTheDocument();
    });

    it("shows recommendations toggle", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue(mockScenario);
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-recommendations-toggle")).toBeInTheDocument();
      });
      expect(screen.getByText("Recommendations")).toBeInTheDocument();
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

    it("displays correct initial state for recommendations toggle", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue({
        ...mockScenario,
        recommendationsEnabled: true,
      });
      renderPage();

      await waitFor(() => {
        const toggle = screen.getByTestId("scenario-recommendations-toggle");
        expect(toggle).toHaveTextContent("Enabled");
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

    it("calls updateMetadata when recommendations toggle is clicked", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue({
        ...mockScenario,
        recommendationsEnabled: true,
      });
      vi.mocked(scenariosService.updateMetadata).mockResolvedValue({
        ...mockScenario,
        recommendationsEnabled: false,
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getByTestId("scenario-recommendations-toggle")).toBeInTheDocument();
      });

      fireEvent.click(screen.getByTestId("scenario-recommendations-toggle"));

      await waitFor(() => {
        expect(scenariosService.updateMetadata).toHaveBeenCalledWith(
          "test-scenario",
          { recommendationsEnabled: false }
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
        expect(
          screen.getByText("Failed to update settings. Please try again.")
        ).toBeInTheDocument();
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

  // [REQ:REQ-P0-007g] Test status display
  describe("status display", () => {
    it("displays running status correctly", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue({
        ...mockScenario,
        status: "running" as const,
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getByText("Running")).toBeInTheDocument();
      });
    });

    it("displays stopped status correctly", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue({
        ...mockScenario,
        status: "stopped" as const,
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getByText("Stopped")).toBeInTheDocument();
      });
    });

    it("displays error status correctly", async () => {
      vi.mocked(scenariosService.get).mockResolvedValue({
        ...mockScenario,
        status: "error" as const,
      });
      renderPage();

      await waitFor(() => {
        expect(screen.getByText("Error")).toBeInTheDocument();
      });
    });
  });

  // Edge cases
  describe("edge cases", () => {
    it("shows error when rendered without name parameter", async () => {
      render(
        <MemoryRouter initialEntries={["/scenarios/"]}>
          <QueryClientProvider client={queryClient}>
            <Routes>
              <Route path="/scenarios/" element={<ScenarioDetailsPage />} />
            </Routes>
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
