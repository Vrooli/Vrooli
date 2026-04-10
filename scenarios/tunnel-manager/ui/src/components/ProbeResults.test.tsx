import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { ProbeResults } from "./ProbeResults";

// [REQ:PROBE-001] [REQ:PROBE-002] ProbeResults component tests

const mockRunProbes = vi.fn();

vi.mock("../lib/api", () => ({
  runProbes: (...args: unknown[]) => mockRunProbes(...args),
}));

function wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <QueryClientProvider client={qc}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  );
}

const probeResponse = {
  results: [
    { route_id: 1, subdomain: "app1", probe_type: "internal", status: "up", latency_ms: 25, error_msg: "" },
    { route_id: 2, subdomain: "app2", probe_type: "external", status: "down", latency_ms: 0, error_msg: "timeout" },
  ],
  summary: { total: 2, up: 1, down: 1 },
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe("ProbeResults", () => {
  it("renders liveness probes heading", () => {
    render(<ProbeResults />, { wrapper });
    expect(screen.getByText("Liveness Probes")).toBeDefined();
  });

  it("shows run probes button", () => {
    render(<ProbeResults />, { wrapper });
    expect(screen.getByText("Run Probes")).toBeDefined();
  });

  it("shows initial prompt when no data", () => {
    render(<ProbeResults />, { wrapper });
    expect(screen.getByText(/No probe results yet/)).toBeDefined();
  });

  it("calls runProbes on button click", async () => {
    mockRunProbes.mockResolvedValue(probeResponse);
    render(<ProbeResults />, { wrapper });
    fireEvent.click(screen.getByText("Run Probes"));
    await waitFor(() => expect(mockRunProbes).toHaveBeenCalledTimes(1));
  });

  it("shows summary after probe run", async () => {
    mockRunProbes.mockResolvedValue(probeResponse);
    render(<ProbeResults />, { wrapper });
    fireEvent.click(screen.getByText("Run Probes"));
    expect(await screen.findByText("1 up")).toBeDefined();
    expect(screen.getByText("1 down")).toBeDefined();
    expect(screen.getByText("2 total")).toBeDefined();
  });

  it("renders results with subdomains", async () => {
    mockRunProbes.mockResolvedValue(probeResponse);
    render(<ProbeResults />, { wrapper });
    fireEvent.click(screen.getByText("Run Probes"));
    const app1 = await screen.findAllByText("app1");
    expect(app1.length).toBeGreaterThanOrEqual(1);
    const app2 = screen.getAllByText("app2");
    expect(app2.length).toBeGreaterThanOrEqual(1);
  });

  it("renders probe types", async () => {
    mockRunProbes.mockResolvedValue(probeResponse);
    render(<ProbeResults />, { wrapper });
    fireEvent.click(screen.getByText("Run Probes"));
    const internals = await screen.findAllByText("internal");
    expect(internals.length).toBeGreaterThanOrEqual(1);
    const externals = screen.getAllByText("external");
    expect(externals.length).toBeGreaterThanOrEqual(1);
  });

  it("renders status badges", async () => {
    mockRunProbes.mockResolvedValue(probeResponse);
    render(<ProbeResults />, { wrapper });
    fireEvent.click(screen.getByText("Run Probes"));
    const ups = await screen.findAllByText("up");
    expect(ups.length).toBeGreaterThanOrEqual(1);
    const downs = screen.getAllByText("down");
    expect(downs.length).toBeGreaterThanOrEqual(1);
  });

  it("formats latency correctly", async () => {
    mockRunProbes.mockResolvedValue(probeResponse);
    render(<ProbeResults />, { wrapper });
    fireEvent.click(screen.getByText("Run Probes"));
    const latency = await screen.findAllByText("25ms");
    expect(latency.length).toBeGreaterThanOrEqual(1);
  });

  it("shows error message for failed probes", async () => {
    mockRunProbes.mockResolvedValue(probeResponse);
    render(<ProbeResults />, { wrapper });
    fireEvent.click(screen.getByText("Run Probes"));
    const errors = await screen.findAllByText("timeout");
    expect(errors.length).toBeGreaterThanOrEqual(1);
  });
});
