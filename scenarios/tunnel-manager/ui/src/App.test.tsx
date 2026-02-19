import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import App from "./App";

// [REQ:UI-001] App routing and navigation tests

vi.mock("./lib/api", () => ({
  fetchTunnelHealth: vi.fn().mockResolvedValue({
    status: "healthy", systemd: "active", ready: "ok", ready_latency_ms: 10, score: 100, message: "OK", checked_at: "2026-02-19T08:00:00Z",
  }),
  fetchRoutes: vi.fn().mockResolvedValue([]),
  runProbes: vi.fn().mockResolvedValue({ results: [], summary: { total: 0, up: 0, down: 0 } }),
  fetchAudit: vi.fn().mockResolvedValue({ results: [], total: 0, violations: 0, compliant: 0 }),
  fetchRecoveryEvents: vi.fn().mockResolvedValue([]),
  fetchRecoveryState: vi.fn().mockResolvedValue({ status: "idle", circuit_open: false, consecutive_failures: 0, backoff_retries: 0 }),
  fetchProbeHistory: vi.fn().mockResolvedValue([]),
  fetchDetailedHealth: vi.fn().mockResolvedValue({ status: "healthy", tunnel: { ready: "ok", systemd: "active", score: 100 }, routes: [], timestamp: "2026-02-19T08:00:00Z" }),
  fetchMetricsHistory: vi.fn().mockResolvedValue([]),
  fetchMetricsLatest: vi.fn().mockResolvedValue({ status: "no_data" }),
}));

function renderApp() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <App />
    </QueryClientProvider>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("App", () => {
  it("renders header with title", () => {
    renderApp();
    expect(screen.getByTestId("app-title")).toBeDefined();
    expect(screen.getByText("Tunnel Manager")).toBeDefined();
  });

  it("renders subtitle", () => {
    renderApp();
    expect(screen.getByText(/Cloudflare tunnel monitoring/)).toBeDefined();
  });

  it("renders desktop navigation links", () => {
    renderApp();
    expect(screen.getByText("Dashboard")).toBeDefined();
    expect(screen.getByText("Routes")).toBeDefined();
    expect(screen.getByText("Probes")).toBeDefined();
    expect(screen.getByText("Metrics")).toBeDefined();
    expect(screen.getByText("Health")).toBeDefined();
    expect(screen.getByText("Recovery")).toBeDefined();
  });

  it("has main navigation landmark", () => {
    renderApp();
    expect(screen.getByRole("navigation", { name: "Main navigation" })).toBeDefined();
  });

  it("renders dashboard as default route", async () => {
    renderApp();
    expect(await screen.findByText("Tunnel Health")).toBeDefined();
  });

  it("renders desktop nav with 6 navigation items", () => {
    renderApp();
    const nav = screen.getByTestId("desktop-nav");
    const links = nav.querySelectorAll("a");
    expect(links.length).toBe(6);
  });

  it("renders mobile menu toggle button", () => {
    renderApp();
    expect(screen.getByTestId("mobile-menu-toggle")).toBeDefined();
  });

  it("has sticky header", () => {
    renderApp();
    const header = screen.getByTestId("app-header");
    expect(header.className).toContain("sticky");
  });

  it("has main content area", () => {
    renderApp();
    expect(screen.getByTestId("main-content")).toBeDefined();
  });

  it("has skip-to-content link for accessibility", () => {
    renderApp();
    const skipLink = screen.getByTestId("skip-link");
    expect(skipLink).toBeDefined();
    expect(skipLink.getAttribute("href")).toBe("#main-content");
  });

  it("has main-content id for skip link target", () => {
    renderApp();
    const main = screen.getByTestId("main-content");
    expect(main.getAttribute("id")).toBe("main-content");
  });

  it("has route announcer for screen readers", () => {
    renderApp();
    const announcers = screen.getAllByRole("status");
    const routeAnnouncer = announcers.find(el => el.getAttribute("aria-live") === "assertive");
    expect(routeAnnouncer).toBeDefined();
    expect(routeAnnouncer!.className).toContain("sr-only");
  });
});
