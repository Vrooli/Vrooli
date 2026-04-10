import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import HealthPage from "./HealthPage";

// [REQ:UI-001] [REQ:OBS-004] Health page tests

const mockFetchDetailedHealth = vi.fn();

vi.mock("../lib/api", () => ({
  fetchDetailedHealth: (...args: unknown[]) => mockFetchDetailedHealth(...args),
}));

function wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <QueryClientProvider client={qc}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  );
}

const healthyData = {
  status: "healthy",
  tunnel: { ready: "ok", systemd: "active", score: 95, ready_latency_ms: 12 },
  routes: [
    { subdomain: "test", scenario_name: "test-scenario", enabled: true, internal_up: true },
  ],
  timestamp: "2026-02-19T08:00:00Z",
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe("HealthPage", () => {
  it("renders detailed health heading", async () => {
    mockFetchDetailedHealth.mockResolvedValue(healthyData);
    render(<HealthPage />, { wrapper });
    expect(await screen.findByText("Detailed Health")).toBeDefined();
  });

  it("shows tunnel overview with score", async () => {
    mockFetchDetailedHealth.mockResolvedValue(healthyData);
    render(<HealthPage />, { wrapper });
    expect(await screen.findByText("95/100")).toBeDefined();
  });

  it("shows tunnel ready status", async () => {
    mockFetchDetailedHealth.mockResolvedValue(healthyData);
    render(<HealthPage />, { wrapper });
    expect(await screen.findByText("ok")).toBeDefined();
  });

  it("shows tunnel systemd status", async () => {
    mockFetchDetailedHealth.mockResolvedValue(healthyData);
    render(<HealthPage />, { wrapper });
    const actives = await screen.findAllByText("active");
    expect(actives.length).toBeGreaterThanOrEqual(1);
  });

  it("shows route health table with route", async () => {
    mockFetchDetailedHealth.mockResolvedValue(healthyData);
    render(<HealthPage />, { wrapper });
    expect(await screen.findByText("Route Health")).toBeDefined();
    const tests = await screen.findAllByText("test");
    expect(tests.length).toBeGreaterThanOrEqual(1);
    const scenarios = screen.getAllByText("test-scenario");
    expect(scenarios.length).toBeGreaterThanOrEqual(1);
  });

  it("shows no routes message when empty", async () => {
    mockFetchDetailedHealth.mockResolvedValue({
      ...healthyData,
      routes: [],
    });
    render(<HealthPage />, { wrapper });
    expect(await screen.findByText("No routes configured.")).toBeDefined();
  });

  it("shows error state when fetch fails", async () => {
    mockFetchDetailedHealth.mockRejectedValue(new Error("Network error"));
    render(<HealthPage />, { wrapper });
    expect(await screen.findByText("Failed to load detailed health.")).toBeDefined();
  });

  it("shows degraded status badge", async () => {
    mockFetchDetailedHealth.mockResolvedValue({
      ...healthyData,
      status: "degraded",
    });
    render(<HealthPage />, { wrapper });
    expect(await screen.findByText("degraded")).toBeDefined();
  });

  it("shows unhealthy status badge", async () => {
    mockFetchDetailedHealth.mockResolvedValue({
      ...healthyData,
      status: "unhealthy",
      tunnel: { ready: "error", systemd: "inactive", score: 0 },
    });
    render(<HealthPage />, { wrapper });
    expect(await screen.findByText("unhealthy")).toBeDefined();
  });

  it("has refresh button", async () => {
    mockFetchDetailedHealth.mockResolvedValue(healthyData);
    render(<HealthPage />, { wrapper });
    expect(await screen.findByLabelText("Refresh health data")).toBeDefined();
  });

  it("shows ready latency when available", async () => {
    mockFetchDetailedHealth.mockResolvedValue(healthyData);
    render(<HealthPage />, { wrapper });
    expect(await screen.findByText("12ms")).toBeDefined();
  });

  it("shows route enabled/disabled status", async () => {
    mockFetchDetailedHealth.mockResolvedValue({
      ...healthyData,
      routes: [
        { subdomain: "enabled-app", scenario_name: "test", enabled: true },
        { subdomain: "disabled-app", scenario_name: "test2", enabled: false },
      ],
    });
    render(<HealthPage />, { wrapper });
    const yesElements = await screen.findAllByText("yes");
    expect(yesElements.length).toBeGreaterThanOrEqual(1);
    const noElements = await screen.findAllByText("no");
    expect(noElements.length).toBeGreaterThanOrEqual(1);
  });
});
