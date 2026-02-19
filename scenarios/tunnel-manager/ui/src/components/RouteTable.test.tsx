import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RouteTable } from "./RouteTable";

// [REQ:UI-002] Route status table
vi.mock("../lib/api", () => ({
  fetchRoutes: vi.fn().mockResolvedValue([]),
}));

function wrapper({ children }: { children: React.ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

describe("RouteTable", () => {
  it("renders the route manifest heading", () => {
    render(<RouteTable />, { wrapper });
    expect(screen.getByText("Route Manifest")).toBeInTheDocument();
  });

  it("shows loading state initially", () => {
    render(<RouteTable />, { wrapper });
    expect(screen.getByText(/loading routes/i)).toBeInTheDocument();
  });
});
