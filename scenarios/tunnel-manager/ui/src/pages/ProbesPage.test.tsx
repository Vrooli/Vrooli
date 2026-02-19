import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import ProbesPage from "./ProbesPage";

// [REQ:PROBE-001] [REQ:PROBE-003] Probes page tests

const mockRunProbes = vi.fn();
const mockFetchProbeHistory = vi.fn();

vi.mock("../lib/api", () => ({
  runProbes: (...args: unknown[]) => mockRunProbes(...args),
  fetchProbeHistory: (...args: unknown[]) => mockFetchProbeHistory(...args),
}));

function wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <QueryClientProvider client={qc}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  );
}

const probeHistoryData = [
  { route_id: 1, subdomain: "app1", probe_type: "internal", status: "up", latency_ms: 42, error_msg: "" },
  { route_id: 2, subdomain: "app2", probe_type: "external", status: "down", latency_ms: 0, error_msg: "connection refused" },
  { route_id: 3, subdomain: "app3", probe_type: "internal", status: "timeout", latency_ms: 0, error_msg: "context deadline exceeded" },
];

beforeEach(() => {
  vi.clearAllMocks();
});

describe("ProbesPage", () => {
  it("renders liveness probes section", () => {
    mockFetchProbeHistory.mockResolvedValue([]);
    render(<ProbesPage />, { wrapper });
    expect(screen.getByText("Liveness Probes")).toBeDefined();
  });

  it("renders probe history section", async () => {
    mockFetchProbeHistory.mockResolvedValue([]);
    render(<ProbesPage />, { wrapper });
    expect(screen.getByText("Probe History")).toBeDefined();
  });

  it("shows run probes button", () => {
    mockFetchProbeHistory.mockResolvedValue([]);
    render(<ProbesPage />, { wrapper });
    expect(screen.getByText("Run Probes")).toBeDefined();
  });

  it("shows empty probe history message", async () => {
    mockFetchProbeHistory.mockResolvedValue([]);
    render(<ProbesPage />, { wrapper });
    expect(await screen.findByText(/No probe history recorded/)).toBeDefined();
  });

  it("shows probe history error state", async () => {
    mockFetchProbeHistory.mockRejectedValue(new Error("Network error"));
    render(<ProbesPage />, { wrapper });
    expect(await screen.findByText("Failed to load probe history.")).toBeDefined();
  });

  it("renders probe history with data", async () => {
    mockFetchProbeHistory.mockResolvedValue(probeHistoryData);
    render(<ProbesPage />, { wrapper });
    const app1 = await screen.findAllByText("app1");
    expect(app1.length).toBeGreaterThanOrEqual(1);
    const app2 = screen.getAllByText("app2");
    expect(app2.length).toBeGreaterThanOrEqual(1);
    const app3 = screen.getAllByText("app3");
    expect(app3.length).toBeGreaterThanOrEqual(1);
  });

  it("shows probe type in history", async () => {
    mockFetchProbeHistory.mockResolvedValue(probeHistoryData);
    render(<ProbesPage />, { wrapper });
    const internals = await screen.findAllByText("internal");
    expect(internals.length).toBeGreaterThanOrEqual(1);
    const externals = screen.getAllByText("external");
    expect(externals.length).toBeGreaterThanOrEqual(1);
  });

  it("shows probe status badges", async () => {
    mockFetchProbeHistory.mockResolvedValue(probeHistoryData);
    render(<ProbesPage />, { wrapper });
    const ups = await screen.findAllByText("up");
    expect(ups.length).toBeGreaterThanOrEqual(1);
    const downs = screen.getAllByText("down");
    expect(downs.length).toBeGreaterThanOrEqual(1);
    const timeouts = screen.getAllByText("timeout");
    expect(timeouts.length).toBeGreaterThanOrEqual(1);
  });

  it("formats latency in history", async () => {
    mockFetchProbeHistory.mockResolvedValue(probeHistoryData);
    render(<ProbesPage />, { wrapper });
    const latency = await screen.findAllByText("42ms");
    expect(latency.length).toBeGreaterThanOrEqual(1);
  });

  it("shows error messages in history", async () => {
    mockFetchProbeHistory.mockResolvedValue(probeHistoryData);
    render(<ProbesPage />, { wrapper });
    const errors = await screen.findAllByText("connection refused");
    expect(errors.length).toBeGreaterThanOrEqual(1);
  });

  it("has refresh button for probe history", async () => {
    mockFetchProbeHistory.mockResolvedValue([]);
    render(<ProbesPage />, { wrapper });
    expect(await screen.findByLabelText("Refresh probe history")).toBeDefined();
  });

  it("shows initial prompt to run probes", () => {
    mockFetchProbeHistory.mockResolvedValue([]);
    render(<ProbesPage />, { wrapper });
    expect(screen.getByText(/No probe results yet/)).toBeDefined();
  });
});
