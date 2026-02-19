import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { TunnelStatusPanel } from "./TunnelStatus";

// [REQ:UI-001] Tunnel health overview panel
vi.mock("../lib/api", () => ({
  fetchTunnelHealth: vi.fn().mockResolvedValue({
    status: "healthy",
    systemd: "active",
    ready: "ok",
    ready_latency_ms: 12,
    score: 100,
    message: "",
    checked_at: "2026-02-19T00:00:00Z",
  }),
}));

function wrapper({ children }: { children: React.ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

describe("TunnelStatusPanel", () => {
  it("renders the tunnel health heading", () => {
    render(<TunnelStatusPanel />, { wrapper });
    expect(screen.getByText("Tunnel Health")).toBeInTheDocument();
  });

  it("shows loading state initially", () => {
    render(<TunnelStatusPanel />, { wrapper });
    expect(screen.getByText(/checking tunnel health/i)).toBeInTheDocument();
  });
});
