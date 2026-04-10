import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RouteManagement } from "./RouteForm";
import userEvent from "@testing-library/user-event";

// [REQ:UI-005] Route management form
vi.mock("../lib/api", () => ({
  createRoute: vi.fn().mockResolvedValue({}),
  updateRoute: vi.fn().mockResolvedValue({}),
  deleteRoute: vi.fn().mockResolvedValue(undefined),
  fetchRoutes: vi.fn().mockResolvedValue([]),
}));

function wrapper({ children }: { children: React.ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

describe("RouteManagement", () => {
  it("renders the route management heading", () => {
    render(<RouteManagement />, { wrapper });
    expect(screen.getByText("Add & Edit Routes")).toBeInTheDocument();
  });

  it("shows Add Route button", () => {
    render(<RouteManagement />, { wrapper });
    expect(screen.getByText("Add Route")).toBeInTheDocument();
  });

  it("opens the form when Add Route is clicked", async () => {
    const user = userEvent.setup();
    render(<RouteManagement />, { wrapper });
    await user.click(screen.getByText("Add Route"));
    expect(screen.getByText(/Subdomain/)).toBeInTheDocument();
    expect(screen.getByText(/Scenario Name/)).toBeInTheDocument();
    expect(screen.getByText(/Local Port/)).toBeInTheDocument();
  });
});
