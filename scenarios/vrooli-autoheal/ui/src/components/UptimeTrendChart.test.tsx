// UptimeTrendChart component tests
// [REQ:UI-EVENTS-001] [REQ:PERSIST-HISTORY-001]
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { UptimeTrendChart } from "./UptimeTrendChart";
import * as api from "../lib/api";

// Mock the API module
vi.mock("../lib/api", async () => {
  const actual = await vi.importActual("../lib/api");
  return {
    ...actual,
    fetchUptimeHistory: vi.fn(),
  };
});

// Mock recharts to avoid SVG rendering issues in tests
vi.mock("recharts", () => ({
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="responsive-container">{children}</div>
  ),
  AreaChart: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="area-chart">{children}</div>
  ),
  Area: () => <div data-testid="area" />,
  XAxis: () => <div data-testid="x-axis" />,
  YAxis: () => <div data-testid="y-axis" />,
  Tooltip: () => <div data-testid="tooltip" />,
  CartesianGrid: () => <div data-testid="cartesian-grid" />,
}));

const createTestQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
    },
  });

const renderWithProviders = (ui: React.ReactElement) => {
  const queryClient = createTestQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>
  );
};

describe("[REQ:UI-EVENTS-001] UptimeTrendChart", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("shows loading state while fetching data", async () => {
    vi.mocked(api.fetchUptimeHistory).mockImplementation(
      () => new Promise(() => {})
    );

    renderWithProviders(<UptimeTrendChart />);

    expect(screen.getByText(/loading trend data/i)).toBeInTheDocument();
  });

  it("renders chart when data is available", async () => {
    vi.mocked(api.fetchUptimeHistory).mockResolvedValue({
      buckets: [
        {
          timestamp: "2024-01-01T12:00:00Z",
          total: 10,
          ok: 8,
          warning: 1,
          critical: 1,
        },
        {
          timestamp: "2024-01-01T13:00:00Z",
          total: 10,
          ok: 9,
          warning: 1,
          critical: 0,
        },
      ],
      overall: { uptimePercentage: 85, totalEvents: 20 },
      windowHours: 24,
      bucketCount: 24,
    });

    renderWithProviders(<UptimeTrendChart />);

    await waitFor(() => {
      expect(screen.getByTestId("autoheal-trends-chart")).toBeInTheDocument();
    });
  });

  it("shows empty state when no data available", async () => {
    vi.mocked(api.fetchUptimeHistory).mockResolvedValue({
      buckets: [],
      overall: { uptimePercentage: 100, totalEvents: 0 },
      windowHours: 24,
      bucketCount: 24,
    });

    renderWithProviders(<UptimeTrendChart />);

    await waitFor(() => {
      expect(screen.getByText(/no historical data available/i)).toBeInTheDocument();
    });
  });

  it("uses default window hours when not provided", async () => {
    vi.mocked(api.fetchUptimeHistory).mockResolvedValue({
      buckets: [],
      overall: { uptimePercentage: 100, totalEvents: 0 },
      windowHours: 24,
      bucketCount: 24,
    });

    renderWithProviders(<UptimeTrendChart />);

    await waitFor(() => {
      expect(api.fetchUptimeHistory).toHaveBeenCalledWith(24, 24);
    });
  });

  it("uses provided window hours and bucket count", async () => {
    vi.mocked(api.fetchUptimeHistory).mockResolvedValue({
      buckets: [],
      overall: { uptimePercentage: 100, totalEvents: 0 },
      windowHours: 6,
      bucketCount: 12,
    });

    renderWithProviders(<UptimeTrendChart windowHours={6} bucketCount={12} />);

    await waitFor(() => {
      expect(api.fetchUptimeHistory).toHaveBeenCalledWith(6, 12);
    });
  });

  it("shows error display when fetch fails", async () => {
    vi.mocked(api.fetchUptimeHistory).mockRejectedValue(new Error("Network error"));

    renderWithProviders(<UptimeTrendChart />);

    await waitFor(() => {
      expect(screen.getByText(/network error/i)).toBeInTheDocument();
    });
  });

  it("renders chart with correct test id", async () => {
    vi.mocked(api.fetchUptimeHistory).mockResolvedValue({
      buckets: [
        {
          timestamp: "2024-01-01T12:00:00Z",
          total: 10,
          ok: 10,
          warning: 0,
          critical: 0,
        },
      ],
      overall: { uptimePercentage: 100, totalEvents: 10 },
      windowHours: 24,
      bucketCount: 24,
    });

    renderWithProviders(<UptimeTrendChart />);

    await waitFor(() => {
      expect(screen.getByTestId("autoheal-trends-chart")).toBeInTheDocument();
    });
  });
});
