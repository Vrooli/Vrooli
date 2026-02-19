import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { TunnelStatusPanel } from "./TunnelStatus";

// [REQ:UI-001] Tunnel health overview panel
const mockFetchTunnelHealth = vi.fn();

vi.mock("../lib/api", () => ({
  fetchTunnelHealth: (...args: unknown[]) => mockFetchTunnelHealth(...args),
}));

const healthyStatus = {
  status: "healthy",
  systemd: "active",
  ready: "ok",
  ready_latency_ms: 12,
  score: 100,
  message: "",
  checked_at: "2026-02-19T00:00:00Z",
};

function wrapper({ children }: { children: React.ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("TunnelStatusPanel", () => {
  it("renders the tunnel health heading", () => {
    mockFetchTunnelHealth.mockResolvedValue(healthyStatus);
    render(<TunnelStatusPanel />, { wrapper });
    expect(screen.getByText("Tunnel Health")).toBeInTheDocument();
  });

  it("shows loading state initially", () => {
    mockFetchTunnelHealth.mockReturnValue(new Promise(() => {}));
    render(<TunnelStatusPanel />, { wrapper });
    expect(screen.getByText(/checking tunnel health/i)).toBeInTheDocument();
  });

  it("shows health score when loaded", async () => {
    mockFetchTunnelHealth.mockResolvedValue(healthyStatus);
    render(<TunnelStatusPanel />, { wrapper });
    expect(await screen.findByText("100/100")).toBeInTheDocument();
  });

  it("shows HEALTHY status badge", async () => {
    mockFetchTunnelHealth.mockResolvedValue(healthyStatus);
    render(<TunnelStatusPanel />, { wrapper });
    expect(await screen.findByText("HEALTHY")).toBeInTheDocument();
  });

  it("shows systemd and ready values", async () => {
    mockFetchTunnelHealth.mockResolvedValue(healthyStatus);
    render(<TunnelStatusPanel />, { wrapper });
    expect(await screen.findByText("active")).toBeInTheDocument();
    expect(screen.getByText(/ok/)).toBeInTheDocument();
  });

  it("shows error state when API unreachable", async () => {
    mockFetchTunnelHealth.mockRejectedValue(new Error("fail"));
    render(<TunnelStatusPanel />, { wrapper });
    expect(await screen.findByText(/unable to reach api/i)).toBeInTheDocument();
  });

  it("has retry button on error", async () => {
    mockFetchTunnelHealth.mockRejectedValue(new Error("fail"));
    render(<TunnelStatusPanel />, { wrapper });
    expect(await screen.findByText("Retry")).toBeInTheDocument();
  });

  it("has refresh button with aria-label", () => {
    mockFetchTunnelHealth.mockResolvedValue(healthyStatus);
    render(<TunnelStatusPanel />, { wrapper });
    expect(screen.getByLabelText("Refresh tunnel health")).toBeInTheDocument();
  });

  it("refetches on refresh click", async () => {
    mockFetchTunnelHealth.mockResolvedValue(healthyStatus);
    render(<TunnelStatusPanel />, { wrapper });
    await screen.findByText("100/100");
    fireEvent.click(screen.getByLabelText("Refresh tunnel health"));
    await waitFor(() => expect(mockFetchTunnelHealth).toHaveBeenCalledTimes(2));
  });

  it("renders data-testid attributes", async () => {
    mockFetchTunnelHealth.mockResolvedValue(healthyStatus);
    render(<TunnelStatusPanel />, { wrapper });
    await screen.findByText("100/100");
    expect(screen.getByTestId("tunnel-health-panel")).toBeInTheDocument();
    expect(screen.getByTestId("tunnel-score-value")).toBeInTheDocument();
  });

  it("shows message when present", async () => {
    mockFetchTunnelHealth.mockResolvedValue({ ...healthyStatus, message: "Degraded performance" });
    render(<TunnelStatusPanel />, { wrapper });
    expect(await screen.findByText("Degraded performance")).toBeInTheDocument();
  });

  it("shows green score color for healthy tunnel", async () => {
    mockFetchTunnelHealth.mockResolvedValue(healthyStatus);
    render(<TunnelStatusPanel />, { wrapper });
    const score = await screen.findByTestId("tunnel-score-value");
    expect(score.className).toContain("text-green-400");
  });

  it("shows yellow score color for degraded tunnel", async () => {
    mockFetchTunnelHealth.mockResolvedValue({ ...healthyStatus, score: 60 });
    render(<TunnelStatusPanel />, { wrapper });
    const score = await screen.findByTestId("tunnel-score-value");
    expect(score.className).toContain("text-yellow-400");
  });

  it("shows red score color for critical tunnel", async () => {
    mockFetchTunnelHealth.mockResolvedValue({ ...healthyStatus, score: 30 });
    render(<TunnelStatusPanel />, { wrapper });
    const score = await screen.findByTestId("tunnel-score-value");
    expect(score.className).toContain("text-red-400");
  });
});
