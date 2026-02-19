import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import MetricsPage from "./MetricsPage";

// [REQ:UI-001] [REQ:OBS-001] Metrics page tests

const mockFetchMetricsLatest = vi.fn();
const mockFetchMetricsHistory = vi.fn();

vi.mock("../lib/api", () => ({
  fetchMetricsLatest: (...args: unknown[]) => mockFetchMetricsLatest(...args),
  fetchMetricsHistory: (...args: unknown[]) => mockFetchMetricsHistory(...args),
}));

function wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <QueryClientProvider client={qc}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("MetricsPage", () => {
  it("renders metrics page headings", async () => {
    mockFetchMetricsLatest.mockResolvedValue({
      id: 1, ha_connections: 4, request_errors: 0, active_streams: 12,
      smoothed_rtt_ms: 3.5, scraped_at: "2026-02-19T08:00:00Z",
    });
    mockFetchMetricsHistory.mockResolvedValue([]);
    render(<MetricsPage />, { wrapper });
    expect(await screen.findByText("Current Metrics")).toBeDefined();
  });

  it("shows latest metrics values", async () => {
    mockFetchMetricsLatest.mockResolvedValue({
      id: 1, ha_connections: 4, request_errors: 0, active_streams: 12,
      smoothed_rtt_ms: 3.5, scraped_at: "2026-02-19T08:00:00Z",
    });
    mockFetchMetricsHistory.mockResolvedValue([]);
    render(<MetricsPage />, { wrapper });
    expect(await screen.findByText("HA Connections")).toBeDefined();
    expect(await screen.findByText("4")).toBeDefined();
  });

  it("shows active streams count", async () => {
    mockFetchMetricsLatest.mockResolvedValue({
      id: 1, ha_connections: 4, request_errors: 0, active_streams: 12,
      smoothed_rtt_ms: 3.5, scraped_at: "2026-02-19T08:00:00Z",
    });
    mockFetchMetricsHistory.mockResolvedValue([]);
    render(<MetricsPage />, { wrapper });
    expect(await screen.findByText("Active Streams")).toBeDefined();
    expect(await screen.findByText("12")).toBeDefined();
  });

  it("shows RTT value formatted", async () => {
    mockFetchMetricsLatest.mockResolvedValue({
      id: 1, ha_connections: 4, request_errors: 0, active_streams: 12,
      smoothed_rtt_ms: 3.5, scraped_at: "2026-02-19T08:00:00Z",
    });
    mockFetchMetricsHistory.mockResolvedValue([]);
    render(<MetricsPage />, { wrapper });
    expect(await screen.findByText("RTT")).toBeDefined();
    expect(await screen.findByText("3.5ms")).toBeDefined();
  });

  it("shows no data message when latest returns status object", async () => {
    mockFetchMetricsLatest.mockResolvedValue({ status: "no_data" });
    mockFetchMetricsHistory.mockResolvedValue([]);
    render(<MetricsPage />, { wrapper });
    expect(await screen.findByText("No metrics data yet")).toBeDefined();
  });

  it("shows error state when latest fetch fails", async () => {
    mockFetchMetricsLatest.mockRejectedValue(new Error("Network error"));
    mockFetchMetricsHistory.mockResolvedValue([]);
    render(<MetricsPage />, { wrapper });
    expect(await screen.findByText("Failed to load latest metrics.")).toBeDefined();
  });

  it("shows empty history message", async () => {
    mockFetchMetricsLatest.mockResolvedValue({ status: "no_data" });
    mockFetchMetricsHistory.mockResolvedValue([]);
    render(<MetricsPage />, { wrapper });
    expect(await screen.findByText("No metrics recorded in the last 24 hours.")).toBeDefined();
  });

  it("shows history table with data", async () => {
    mockFetchMetricsLatest.mockResolvedValue({ status: "no_data" });
    mockFetchMetricsHistory.mockResolvedValue([
      { id: 1, ha_connections: 4, active_streams: 10, smoothed_rtt_ms: 2.0, request_errors: 0, scraped_at: "2026-02-19T08:00:00Z" },
    ]);
    render(<MetricsPage />, { wrapper });
    expect(await screen.findByText("Metrics History (24h)")).toBeDefined();
    expect(await screen.findByText("HA Conns")).toBeDefined();
  });

  it("shows error state when history fetch fails", async () => {
    mockFetchMetricsLatest.mockResolvedValue({ status: "no_data" });
    mockFetchMetricsHistory.mockRejectedValue(new Error("Network error"));
    render(<MetricsPage />, { wrapper });
    expect(await screen.findByText("Failed to load metrics history.")).toBeDefined();
  });

  it("has refresh button for history", async () => {
    mockFetchMetricsLatest.mockResolvedValue({ status: "no_data" });
    mockFetchMetricsHistory.mockResolvedValue([]);
    render(<MetricsPage />, { wrapper });
    expect(await screen.findByLabelText("Refresh metrics history")).toBeDefined();
  });
});
