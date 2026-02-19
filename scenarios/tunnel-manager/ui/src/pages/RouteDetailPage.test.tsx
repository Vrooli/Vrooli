import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import RouteDetailPage from "./RouteDetailPage";

// [REQ:UI-002] [REQ:ROUTE-002] Route detail page tests

const mockFetchRoute = vi.fn();
const mockFetchProbeHistory = vi.fn();

vi.mock("../lib/api", () => ({
  fetchRoute: (...args: unknown[]) => mockFetchRoute(...args),
  fetchProbeHistory: (...args: unknown[]) => mockFetchProbeHistory(...args),
}));

function wrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={["/routes/1"]}>
        <Routes>
          <Route path="/routes/:id" element={children} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

function invalidIdWrapper({ children }: { children: React.ReactNode }) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={["/routes/abc"]}>
        <Routes>
          <Route path="/routes/:id" element={children} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

const routeData = {
  id: 1,
  subdomain: "app",
  scenario_name: "test-app",
  local_port: 8080,
  health_path: "/health",
  public_url: "https://app.example.com",
  enabled: true,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-02-19T00:00:00Z",
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe("RouteDetailPage", () => {
  it("renders route detail with subdomain", async () => {
    mockFetchRoute.mockResolvedValue(routeData);
    mockFetchProbeHistory.mockResolvedValue([]);
    render(<RouteDetailPage />, { wrapper });
    expect(await screen.findByText("app")).toBeDefined();
  });

  it("shows route info fields", async () => {
    mockFetchRoute.mockResolvedValue(routeData);
    mockFetchProbeHistory.mockResolvedValue([]);
    render(<RouteDetailPage />, { wrapper });
    expect(await screen.findByText("test-app")).toBeDefined();
    expect(await screen.findByText("8080")).toBeDefined();
  });

  it("shows health path", async () => {
    mockFetchRoute.mockResolvedValue(routeData);
    mockFetchProbeHistory.mockResolvedValue([]);
    render(<RouteDetailPage />, { wrapper });
    expect(await screen.findByText("/health")).toBeDefined();
  });

  it("shows public URL as link", async () => {
    mockFetchRoute.mockResolvedValue(routeData);
    mockFetchProbeHistory.mockResolvedValue([]);
    render(<RouteDetailPage />, { wrapper });
    const urlElement = await screen.findByText("https://app.example.com");
    const link = urlElement.closest("a");
    expect(link).not.toBeNull();
    expect(link!.getAttribute("href")).toBe("https://app.example.com");
  });

  it("shows enabled badge", async () => {
    mockFetchRoute.mockResolvedValue(routeData);
    mockFetchProbeHistory.mockResolvedValue([]);
    render(<RouteDetailPage />, { wrapper });
    expect(await screen.findByText("enabled")).toBeDefined();
  });

  it("shows disabled badge for disabled route", async () => {
    mockFetchRoute.mockResolvedValue({ ...routeData, enabled: false });
    mockFetchProbeHistory.mockResolvedValue([]);
    render(<RouteDetailPage />, { wrapper });
    expect(await screen.findByText("disabled")).toBeDefined();
  });

  it("shows error state when route fetch fails", async () => {
    mockFetchRoute.mockRejectedValue(new Error("Not found"));
    mockFetchProbeHistory.mockResolvedValue([]);
    render(<RouteDetailPage />, { wrapper });
    expect(await screen.findByText("Failed to load route.")).toBeDefined();
  });

  it("shows back to routes link", async () => {
    mockFetchRoute.mockResolvedValue(routeData);
    mockFetchProbeHistory.mockResolvedValue([]);
    render(<RouteDetailPage />, { wrapper });
    expect(await screen.findByText("Back to routes")).toBeDefined();
  });

  it("shows no probe results message when empty", async () => {
    mockFetchRoute.mockResolvedValue(routeData);
    mockFetchProbeHistory.mockResolvedValue([]);
    render(<RouteDetailPage />, { wrapper });
    expect(await screen.findByText("No probe results for this route.")).toBeDefined();
  });

  it("shows probe history with data", async () => {
    mockFetchRoute.mockResolvedValue(routeData);
    mockFetchProbeHistory.mockResolvedValue([
      { route_id: 1, probe_type: "internal", status: "up", latency_ms: 42, error_msg: "" },
      { route_id: 1, probe_type: "external", status: "down", latency_ms: 0, error_msg: "timeout" },
    ]);
    render(<RouteDetailPage />, { wrapper });
    expect(await screen.findByText("Probe History")).toBeDefined();
    const internals = await screen.findAllByText("internal");
    expect(internals.length).toBeGreaterThanOrEqual(1);
    const externals = screen.getAllByText("external");
    expect(externals.length).toBeGreaterThanOrEqual(1);
    const latency = screen.getAllByText("42ms");
    expect(latency.length).toBeGreaterThanOrEqual(1);
  });

  it("shows error when probe history fetch fails", async () => {
    mockFetchRoute.mockResolvedValue(routeData);
    mockFetchProbeHistory.mockRejectedValue(new Error("Network error"));
    render(<RouteDetailPage />, { wrapper });
    expect(await screen.findByText("Failed to load probe history.")).toBeDefined();
  });

  it("has refresh button for probe history", async () => {
    mockFetchRoute.mockResolvedValue(routeData);
    mockFetchProbeHistory.mockResolvedValue([]);
    render(<RouteDetailPage />, { wrapper });
    expect(await screen.findByLabelText("Refresh probe history")).toBeDefined();
  });

  it("shows invalid route ID message", async () => {
    render(<RouteDetailPage />, { wrapper: invalidIdWrapper });
    expect(await screen.findByText("Invalid route ID.")).toBeDefined();
  });
});
