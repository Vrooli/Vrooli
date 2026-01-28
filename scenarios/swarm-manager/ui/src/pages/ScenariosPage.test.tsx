import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter } from "react-router-dom";
import { ScenariosPage } from "./ScenariosPage";
import type { Scenario } from "../types";

/**
 * Mock the config module for testing.
 * This allows tests to control configuration values without retries.
 */
vi.mock("../config", () => ({
  dataFetchingConfig: {
    retryCount: 0, // Disable retries for faster test failures
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
  scenariosService: {
    list: vi.fn(),
    get: vi.fn(),
    updateMetadata: vi.fn(),
  },
}));

import { scenariosService } from "../services";

// Sample scenario data for tests
const mockScenarios: Scenario[] = [
  {
    name: "api-server",
    displayName: "API Server",
    description: "Backend REST API for the application",
    status: "running",
    priority: 1,
    completenessScore: 85,
    isGreenfield: false,
    tags: ["api", "backend", "go"],
    recommendationsEnabled: true,
  },
  {
    name: "ui-dashboard",
    displayName: "UI Dashboard",
    description: "Frontend dashboard built with React",
    status: "stopped",
    priority: 2,
    completenessScore: 65,
    isGreenfield: false,
    tags: ["ui", "frontend", "react"],
    recommendationsEnabled: true,
  },
  {
    name: "data-pipeline",
    displayName: "Data Pipeline",
    description: "ETL data processing pipeline",
    status: "error",
    priority: 3,
    isGreenfield: true,
    tags: ["data", "etl"],
    recommendationsEnabled: false,
  },
];

// [REQ:REQ-P0-006] Test scenarios catalog UI with search and filter
describe("ScenariosPage", () => {
  let queryClient: QueryClient;

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

  const renderPage = () => {
    return render(
      <BrowserRouter>
        <QueryClientProvider client={queryClient}>
          <ScenariosPage />
        </QueryClientProvider>
      </BrowserRouter>
    );
  };

  // [REQ:REQ-P0-006] Test scenarios page renders with search and filter controls
  it("renders the scenarios page with search and filter controls", () => {
    vi.mocked(scenariosService.list).mockResolvedValue([]);
    renderPage();

    expect(screen.getByTestId("scenarios-page")).toBeInTheDocument();
    expect(screen.getByTestId("scenarios-search")).toBeInTheDocument();
    expect(screen.getByTestId("scenarios-filter")).toBeInTheDocument();
  });

  // [REQ:REQ-P0-006] Test empty state when no scenarios exist
  it("shows empty state when no scenarios exist", async () => {
    vi.mocked(scenariosService.list).mockResolvedValue([]);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("scenarios-empty")).toBeInTheDocument();
    });

    expect(screen.getByText("No scenarios found")).toBeInTheDocument();
  });

  // [REQ:REQ-P0-006] Test scenario cards display correctly
  it("renders scenario cards with correct data", async () => {
    vi.mocked(scenariosService.list).mockResolvedValue(mockScenarios);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("scenarios-list")).toBeInTheDocument();
    });

    expect(screen.getByText("API Server")).toBeInTheDocument();
    expect(screen.getByText("Backend REST API for the application")).toBeInTheDocument();
    expect(screen.getByText("P1")).toBeInTheDocument();
    expect(screen.getByText("85%")).toBeInTheDocument();
  });

  // [REQ:REQ-P0-006] Test scenario count display
  it("displays scenario count", async () => {
    vi.mocked(scenariosService.list).mockResolvedValue(mockScenarios);
    renderPage();

    // Wait for the list to load (not just the count element which shows 0 during loading)
    await waitFor(() => {
      expect(screen.getByTestId("scenarios-list")).toBeInTheDocument();
    });

    // The count element should show "3 scenarios"
    const countElement = screen.getByTestId("scenarios-count");
    expect(countElement).toHaveTextContent("3 scenarios");
  });

  // [REQ:REQ-P0-006] Test priority sorting (scenarios should be sorted by priority)
  it("displays scenarios sorted by priority", async () => {
    vi.mocked(scenariosService.list).mockResolvedValue(mockScenarios);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("scenarios-list")).toBeInTheDocument();
    });

    // Get all scenario cards
    const cards = screen.getByTestId("scenarios-list").children;
    expect(cards).toHaveLength(3);

    // First should be P1 (API Server)
    expect(cards[0]).toHaveTextContent("API Server");
    expect(cards[0]).toHaveTextContent("P1");

    // Second should be P2 (UI Dashboard)
    expect(cards[1]).toHaveTextContent("UI Dashboard");
    expect(cards[1]).toHaveTextContent("P2");
  });

  // [REQ:REQ-P0-006] Test search functionality filters by name
  it("filters scenarios by search term in name", async () => {
    vi.mocked(scenariosService.list).mockResolvedValue(mockScenarios);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("scenarios-list")).toBeInTheDocument();
    });

    // Type in search
    const searchInput = screen.getByPlaceholderText("Search scenarios...");
    fireEvent.change(searchInput, { target: { value: "api" } });

    // Should only show api-server
    await waitFor(() => {
      expect(screen.getByText("1 scenario")).toBeInTheDocument();
    });

    expect(screen.getByText("API Server")).toBeInTheDocument();
    expect(screen.queryByText("UI Dashboard")).not.toBeInTheDocument();
  });

  // [REQ:REQ-P0-006] Test search functionality filters by description
  it("filters scenarios by search term in description", async () => {
    vi.mocked(scenariosService.list).mockResolvedValue(mockScenarios);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("scenarios-list")).toBeInTheDocument();
    });

    // Search for "frontend"
    const searchInput = screen.getByPlaceholderText("Search scenarios...");
    fireEvent.change(searchInput, { target: { value: "frontend" } });

    // Should only show ui-dashboard
    await waitFor(() => {
      expect(screen.getByText("1 scenario")).toBeInTheDocument();
    });

    expect(screen.getByText("UI Dashboard")).toBeInTheDocument();
    expect(screen.queryByText("API Server")).not.toBeInTheDocument();
  });

  // [REQ:REQ-P0-006] Test search is case insensitive
  it("search is case insensitive", async () => {
    vi.mocked(scenariosService.list).mockResolvedValue(mockScenarios);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("scenarios-list")).toBeInTheDocument();
    });

    // Search with uppercase
    const searchInput = screen.getByPlaceholderText("Search scenarios...");
    fireEvent.change(searchInput, { target: { value: "API" } });

    // Should still match api-server
    await waitFor(() => {
      expect(screen.getByText("API Server")).toBeInTheDocument();
    });
  });

  // [REQ:REQ-P0-006] Test no results state after filtering
  it("shows no results state when search matches nothing", async () => {
    vi.mocked(scenariosService.list).mockResolvedValue(mockScenarios);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("scenarios-list")).toBeInTheDocument();
    });

    // Search for something that doesn't exist
    const searchInput = screen.getByPlaceholderText("Search scenarios...");
    fireEvent.change(searchInput, { target: { value: "nonexistent" } });

    await waitFor(() => {
      expect(screen.getByTestId("scenarios-no-results")).toBeInTheDocument();
    });

    expect(screen.getByText("No matching scenarios")).toBeInTheDocument();
    expect(screen.getByText("Clear filters")).toBeInTheDocument();
  });

  // [REQ:REQ-P0-006] Test clear filters button works
  it("clear filters button resets search", async () => {
    vi.mocked(scenariosService.list).mockResolvedValue(mockScenarios);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("scenarios-list")).toBeInTheDocument();
    });

    // Search for something that doesn't exist
    const searchInput = screen.getByPlaceholderText("Search scenarios...");
    fireEvent.change(searchInput, { target: { value: "nonexistent" } });

    await waitFor(() => {
      expect(screen.getByTestId("scenarios-no-results")).toBeInTheDocument();
    });

    // Click clear filters
    fireEvent.click(screen.getByText("Clear filters"));

    // Should show all scenarios again
    await waitFor(() => {
      expect(screen.getByTestId("scenarios-list")).toBeInTheDocument();
      expect(screen.getByText("3 scenarios")).toBeInTheDocument();
    });
  });

  // [REQ:REQ-P0-006] Test filter dropdown opens
  it("opens filter dropdown when filter button is clicked", async () => {
    vi.mocked(scenariosService.list).mockResolvedValue(mockScenarios);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("scenarios-list")).toBeInTheDocument();
    });

    // Click filter button
    fireEvent.click(screen.getByTestId("scenarios-filter"));

    // Dropdown should appear
    await waitFor(() => {
      expect(screen.getByTestId("scenarios-filter-dropdown")).toBeInTheDocument();
    });

    expect(screen.getByText("Filters")).toBeInTheDocument();
    expect(screen.getByTestId("scenarios-status-filter")).toBeInTheDocument();
  });

  // [REQ:REQ-P0-006] Test loading state
  it("shows loading state while fetching scenarios", () => {
    vi.mocked(scenariosService.list).mockImplementation(
      () => new Promise(() => {}) // Never resolves
    );
    renderPage();

    expect(screen.getByText("Loading scenarios...")).toBeInTheDocument();
  });

  // [REQ:REQ-P0-006] Test error state shows ErrorState component
  it("shows error state when API fails", async () => {
    vi.mocked(scenariosService.list).mockRejectedValue(new Error("API error"));
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("error-state")).toBeInTheDocument();
    });

    expect(screen.getByText("Unable to load scenarios")).toBeInTheDocument();
  });

  // [REQ:REQ-P0-006] Test greenfield badge displays
  it("displays greenfield badge for new scenarios", async () => {
    vi.mocked(scenariosService.list).mockResolvedValue(mockScenarios);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("scenarios-list")).toBeInTheDocument();
    });

    // Data Pipeline is greenfield
    expect(screen.getByText("Greenfield")).toBeInTheDocument();
  });

  // [REQ:REQ-P0-006] Test filtered count shows "of total" when filtering
  it("shows filtered count with total when filtering", async () => {
    vi.mocked(scenariosService.list).mockResolvedValue(mockScenarios);
    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId("scenarios-list")).toBeInTheDocument();
    });

    // Search to filter
    const searchInput = screen.getByPlaceholderText("Search scenarios...");
    fireEvent.change(searchInput, { target: { value: "api" } });

    await waitFor(() => {
      expect(screen.getByText("1 scenario")).toBeInTheDocument();
      expect(screen.getByText("of 3")).toBeInTheDocument();
    });
  });
});
