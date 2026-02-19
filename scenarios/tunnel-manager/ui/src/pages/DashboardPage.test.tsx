import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import DashboardPage from "./DashboardPage";

// [REQ:UI-001] [REQ:DASH-001] Dashboard page tests

const mockFetchTunnelHealth = vi.fn();
const mockFetchRoutes = vi.fn();
const mockRunProbes = vi.fn();
const mockFetchAudit = vi.fn();
const mockFetchRecoveryEvents = vi.fn();

vi.mock("../lib/api", () => ({
  fetchTunnelHealth: (...args: unknown[]) => mockFetchTunnelHealth(...args),
  fetchRoutes: (...args: unknown[]) => mockFetchRoutes(...args),
  runProbes: (...args: unknown[]) => mockRunProbes(...args),
  fetchAudit: (...args: unknown[]) => mockFetchAudit(...args),
  fetchRecoveryEvents: (...args: unknown[]) => mockFetchRecoveryEvents(...args),
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
  mockFetchTunnelHealth.mockResolvedValue({
    status: "healthy", systemd: "active", ready: "ok", ready_latency_ms: 10, score: 100, message: "OK", checked_at: "2026-02-19T08:00:00Z",
  });
  mockFetchRoutes.mockResolvedValue([
    { id: 1, subdomain: "test", scenario_name: "test-app", local_port: 3000, health_path: "/health", public_url: "https://test.example.com", enabled: true, created_at: "2026-02-19T00:00:00Z", updated_at: "2026-02-19T00:00:00Z" },
  ]);
  mockFetchAudit.mockResolvedValue({ results: [], total: 0, violations: 0, compliant: 0 });
  mockFetchRecoveryEvents.mockResolvedValue([]);
});

describe("DashboardPage", () => {
  it("renders tunnel status section", async () => {
    render(<DashboardPage />, { wrapper });
    expect(await screen.findByText("Tunnel Health")).toBeDefined();
  });

  it("renders route table section", async () => {
    render(<DashboardPage />, { wrapper });
    expect(await screen.findByText("Route Manifest")).toBeDefined();
  });

  it("renders liveness probes section", async () => {
    render(<DashboardPage />, { wrapper });
    expect(await screen.findByText("Liveness Probes")).toBeDefined();
  });

  it("renders port audit section", async () => {
    render(<DashboardPage />, { wrapper });
    expect(await screen.findByText("Port Compliance Audit")).toBeDefined();
  });

  it("renders recovery timeline section", async () => {
    render(<DashboardPage />, { wrapper });
    expect(await screen.findByText("Recovery Events")).toBeDefined();
  });

  it("shows tunnel health score", async () => {
    render(<DashboardPage />, { wrapper });
    expect(await screen.findByText("100/100")).toBeDefined();
  });

  it("shows route data in table", async () => {
    render(<DashboardPage />, { wrapper });
    const tests = await screen.findAllByText("test");
    expect(tests.length).toBeGreaterThanOrEqual(1);
    const apps = screen.getAllByText("test-app");
    expect(apps.length).toBeGreaterThanOrEqual(1);
  });

  it("shows audit summary counts", async () => {
    render(<DashboardPage />, { wrapper });
    expect(await screen.findByText("0 compliant")).toBeDefined();
    expect(screen.getByText("0 total")).toBeDefined();
  });

  it("shows healthy tunnel status", async () => {
    render(<DashboardPage />, { wrapper });
    expect(await screen.findByText("HEALTHY")).toBeDefined();
  });
});
