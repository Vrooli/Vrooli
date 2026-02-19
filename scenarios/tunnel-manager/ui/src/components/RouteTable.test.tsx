import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { RouteTable } from "./RouteTable";

// [REQ:UI-002] Route status table
const mockFetchRoutes = vi.fn();

vi.mock("../lib/api", () => ({
  fetchRoutes: (...args: unknown[]) => mockFetchRoutes(...args),
}));

function wrapper({ children }: { children: React.ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>
  );
}

const mockRoutes = [
  { id: 1, subdomain: "api", scenario_name: "my-api", local_port: 3000, enabled: true, health_path: "/health", public_url: "https://api.example.com", created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z" },
  { id: 2, subdomain: "web", scenario_name: "my-web", local_port: 8080, enabled: false, health_path: "/health", public_url: "", created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z" },
];

beforeEach(() => {
  vi.clearAllMocks();
});

describe("RouteTable", () => {
  it("renders the route manifest heading", () => {
    mockFetchRoutes.mockResolvedValue([]);
    render(<RouteTable />, { wrapper });
    expect(screen.getByText("Route Manifest")).toBeInTheDocument();
  });

  it("shows loading state initially", () => {
    mockFetchRoutes.mockReturnValue(new Promise(() => {}));
    render(<RouteTable />, { wrapper });
    expect(screen.getByText(/loading routes/i)).toBeInTheDocument();
  });

  it("shows empty state when no routes", async () => {
    mockFetchRoutes.mockResolvedValue([]);
    render(<RouteTable />, { wrapper });
    expect(await screen.findByText(/no routes configured/i)).toBeInTheDocument();
  });

  it("renders route subdomains", async () => {
    mockFetchRoutes.mockResolvedValue(mockRoutes);
    render(<RouteTable />, { wrapper });
    const apis = await screen.findAllByText("api");
    expect(apis.length).toBeGreaterThanOrEqual(1);
  });

  it("shows search input when routes exist", async () => {
    mockFetchRoutes.mockResolvedValue(mockRoutes);
    render(<RouteTable />, { wrapper });
    await screen.findAllByText("api");
    expect(screen.getByLabelText("Filter routes")).toBeInTheDocument();
  });

  it("filters routes by search input", async () => {
    mockFetchRoutes.mockResolvedValue(mockRoutes);
    render(<RouteTable />, { wrapper });
    await screen.findAllByText("api");
    fireEvent.change(screen.getByLabelText("Filter routes"), { target: { value: "web" } });
    await waitFor(() => {
      expect(screen.queryAllByText("api").length).toBe(0);
      expect(screen.getAllByText("web").length).toBeGreaterThanOrEqual(1);
    });
  });

  it("shows no-match message when filter has no results", async () => {
    mockFetchRoutes.mockResolvedValue(mockRoutes);
    render(<RouteTable />, { wrapper });
    await screen.findAllByText("api");
    fireEvent.change(screen.getByLabelText("Filter routes"), { target: { value: "nonexistent" } });
    expect(await screen.findByText(/no routes match/i)).toBeInTheDocument();
  });

  it("shows enabled/disabled badges", async () => {
    mockFetchRoutes.mockResolvedValue(mockRoutes);
    render(<RouteTable />, { wrapper });
    const enabled = await screen.findAllByText("enabled");
    expect(enabled.length).toBeGreaterThanOrEqual(1);
    const disabled = screen.getAllByText("disabled");
    expect(disabled.length).toBeGreaterThanOrEqual(1);
  });

  it("shows error state when fetch fails", async () => {
    mockFetchRoutes.mockRejectedValue(new Error("fail"));
    render(<RouteTable />, { wrapper });
    expect(await screen.findByText(/failed to load routes/i)).toBeInTheDocument();
  });

  it("has Add Route CTA in empty state", async () => {
    mockFetchRoutes.mockResolvedValue([]);
    render(<RouteTable />, { wrapper });
    expect(await screen.findByText("Add Route")).toBeInTheDocument();
  });
});
