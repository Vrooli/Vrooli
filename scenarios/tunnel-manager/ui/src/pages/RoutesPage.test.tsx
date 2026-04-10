import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import userEvent from "@testing-library/user-event";
import RoutesPage from "./RoutesPage";

// [REQ:UI-005] [REQ:RM-001] Routes page tests

const mockFetchRoutes = vi.fn();
const mockCreateRoute = vi.fn();
const mockUpdateRoute = vi.fn();
const mockDeleteRoute = vi.fn();

vi.mock("../lib/api", () => ({
  fetchRoutes: (...args: unknown[]) => mockFetchRoutes(...args),
  createRoute: (...args: unknown[]) => mockCreateRoute(...args),
  updateRoute: (...args: unknown[]) => mockUpdateRoute(...args),
  deleteRoute: (...args: unknown[]) => mockDeleteRoute(...args),
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
  mockFetchRoutes.mockResolvedValue([
    {
      id: 1, subdomain: "app-one", scenario_name: "app-one-scenario",
      local_port: 3000, health_path: "/health", public_url: "https://app-one.example.com",
      enabled: true, created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z",
    },
    {
      id: 2, subdomain: "app-two", scenario_name: "app-two-scenario",
      local_port: 4000, health_path: "/health", public_url: "",
      enabled: false, created_at: "2026-01-01T00:00:00Z", updated_at: "2026-01-01T00:00:00Z",
    },
  ]);
  mockCreateRoute.mockResolvedValue({});
});

describe("RoutesPage", () => {
  it("renders both RouteTable and RouteManagement sections", async () => {
    render(<RoutesPage />, { wrapper });
    expect(await screen.findByText("Route Manifest")).toBeDefined();
    expect(screen.getByText("Add & Edit Routes")).toBeDefined();
  });

  it("displays routes from the API", async () => {
    render(<RoutesPage />, { wrapper });
    const appOnes = await screen.findAllByText("app-one");
    expect(appOnes.length).toBeGreaterThanOrEqual(1);
    const appTwos = screen.getAllByText("app-two");
    expect(appTwos.length).toBeGreaterThanOrEqual(1);
  });

  it("shows enabled/disabled status badges", async () => {
    render(<RoutesPage />, { wrapper });
    const enabledBadges = await screen.findAllByText("enabled");
    expect(enabledBadges.length).toBeGreaterThanOrEqual(1);
    const disabledBadges = screen.getAllByText("disabled");
    expect(disabledBadges.length).toBeGreaterThanOrEqual(1);
  });

  it("shows Add Route button in management section", async () => {
    render(<RoutesPage />, { wrapper });
    expect(await screen.findByText("Add Route")).toBeDefined();
  });

  it("opens route form when Add Route is clicked", async () => {
    const user = userEvent.setup();
    render(<RoutesPage />, { wrapper });
    await screen.findByText("Add Route");
    await user.click(screen.getByText("Add Route"));
    expect(screen.getByRole("textbox", { name: /Subdomain/ })).toBeDefined();
    expect(screen.getByRole("textbox", { name: /Scenario Name/ })).toBeDefined();
    expect(screen.getByRole("spinbutton", { name: /Local Port/ })).toBeDefined();
  });

  it("shows public URL as link for routes that have one", async () => {
    render(<RoutesPage />, { wrapper });
    const urlElements = await screen.findAllByText("https://app-one.example.com");
    const link = urlElements[0].closest("a");
    expect(link).not.toBeNull();
    expect(link!.getAttribute("href")).toBe("https://app-one.example.com");
  });

  it("shows refresh button for route table", async () => {
    render(<RoutesPage />, { wrapper });
    expect(await screen.findByLabelText("Refresh routes")).toBeDefined();
  });
});
