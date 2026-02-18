// TrendsPage component tests
// [REQ:UI-EVENTS-001] [REQ:PERSIST-HISTORY-001]
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent, waitFor } from "@testing-library/react";
import { TrendsPage } from "./TrendsPage";
import * as api from "../../lib/api";
import { renderWithProviders } from "../../test-utils";
import {
  createCheckTrendsResponse,
  createIncident,
  createIncidentsResponse,
  createTimelineResponse,
  createUptimeHistoryResponse,
  createUptimeStatsResponse,
} from "../../test-utils";

vi.mock("../../shared/contexts/CheckMetadataContext", async () => {
  const { useMockCheckMetadata } = await import(
    "../../test-utils/mocks/checkMetadataContext"
  );
  return {
    useCheckMetadata: useMockCheckMetadata,
  };
});

// Mock the API module
vi.mock("../../lib/api", async () => {
  const actual = await vi.importActual("../../lib/api");
  return {
    ...actual,
    fetchUptimeStats: vi.fn(),
    fetchCheckTrends: vi.fn(),
    fetchIncidents: vi.fn(),
    fetchTimeline: vi.fn(),
    fetchUptimeHistory: vi.fn(),
  };
});

// Mock the export module
vi.mock("../../lib/export", () => ({
  exportTrendDataToCSV: vi.fn(),
}));

describe("[REQ:UI-EVENTS-001] TrendsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Default mocks
    vi.mocked(api.fetchUptimeStats).mockResolvedValue(createUptimeStatsResponse());
    vi.mocked(api.fetchCheckTrends).mockResolvedValue(createCheckTrendsResponse());
    vi.mocked(api.fetchIncidents).mockResolvedValue(createIncidentsResponse());
    vi.mocked(api.fetchTimeline).mockResolvedValue(createTimelineResponse());
    vi.mocked(api.fetchUptimeHistory).mockResolvedValue(createUptimeHistoryResponse());
  });

  it("renders the page with header", async () => {
    renderWithProviders(<TrendsPage />);

    expect(screen.getByText("Health Trends")).toBeInTheDocument();
    expect(screen.getByText(/historical analysis/i)).toBeInTheDocument();
  });

  it("displays time window selector buttons", async () => {
    renderWithProviders(<TrendsPage />);

    expect(screen.getByTestId("time-window-selector")).toBeInTheDocument();
    expect(screen.getByTestId("time-window-6h")).toBeInTheDocument();
    expect(screen.getByTestId("time-window-12h")).toBeInTheDocument();
    expect(screen.getByTestId("time-window-24h")).toBeInTheDocument();
    expect(screen.getByTestId("time-window-7d")).toBeInTheDocument();
  });

  it("defaults to 24h time window", async () => {
    renderWithProviders(<TrendsPage />);

    await waitFor(() => {
      expect(api.fetchCheckTrends).toHaveBeenCalledWith(24);
    });
  });

  it("changes time window when button clicked", async () => {
    renderWithProviders(<TrendsPage />);

    const button6h = screen.getByTestId("time-window-6h");
    fireEvent.click(button6h);

    await waitFor(() => {
      expect(api.fetchCheckTrends).toHaveBeenCalledWith(6);
    });
  });

  it("displays uptime percentage from API", async () => {
    renderWithProviders(<TrendsPage />);

    await waitFor(() => {
      expect(screen.getByText("95.0%")).toBeInTheDocument();
    });
  });

  it("displays export button", async () => {
    renderWithProviders(<TrendsPage />);

    expect(screen.getByTestId("export-csv-button")).toBeInTheDocument();
  });

  it("shows loading state for trend data", async () => {
    // Never resolve to keep loading state
    vi.mocked(api.fetchCheckTrends).mockImplementation(
      () => new Promise(() => {})
    );

    renderWithProviders(<TrendsPage />);

    expect(screen.getByText(/loading check data/i)).toBeInTheDocument();
  });

  it("shows loading state for incidents", async () => {
    vi.mocked(api.fetchIncidents).mockImplementation(
      () => new Promise(() => {})
    );

    renderWithProviders(<TrendsPage />);

    expect(screen.getByText(/loading incidents/i)).toBeInTheDocument();
  });

  it("displays no status transitions message when none exist", async () => {
    renderWithProviders(<TrendsPage />);

    await waitFor(() => {
      expect(screen.getByText(/no status transitions detected/i)).toBeInTheDocument();
    });
  });

  it("displays incidents when available", async () => {
    vi.mocked(api.fetchIncidents).mockResolvedValue(
      createIncidentsResponse({
        incidents: [
          createIncident({
            checkId: "test-check",
            message: "Test incident",
          }),
        ],
      })
    );

    renderWithProviders(<TrendsPage />);

    await waitFor(() => {
      expect(screen.getByText("test-check")).toBeInTheDocument();
    });
  });

  it("shows Per-Check Health section", async () => {
    renderWithProviders(<TrendsPage />);

    expect(screen.getByText("Per-Check Health")).toBeInTheDocument();
  });

  it("shows Status Transitions section", async () => {
    renderWithProviders(<TrendsPage />);

    expect(screen.getByText("Status Transitions")).toBeInTheDocument();
  });
});
