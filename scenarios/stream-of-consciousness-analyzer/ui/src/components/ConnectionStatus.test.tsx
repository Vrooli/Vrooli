import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import "@testing-library/jest-dom/vitest";

const mockFetchHealth = vi.fn();

vi.mock("../lib/api", () => ({
  fetchHealth: (...args: unknown[]) => mockFetchHealth(...args) as unknown,
}));

vi.mock("@vrooli/api-base", () => ({
  resolveApiBase: () => "http://localhost:3000/api/v1",
  buildApiUrl: (path: string) => `http://localhost:3000/api/v1${path}`,
}));

import { ConnectionStatus } from "./ConnectionStatus";

function renderComponent() {
  // Disable retries AND set gcTime to 0 to avoid stale query cache between tests
  const qc = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
    },
  });
  return render(
    <QueryClientProvider client={qc}>
      <ConnectionStatus />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  mockFetchHealth.mockReset();
});

// [REQ:P0-001] Connection status shows degraded state when API unreachable
describe("ConnectionStatus", () => {
  it("shows nothing when API is healthy", async () => {
    mockFetchHealth.mockResolvedValue({ status: "ok" });
    renderComponent();
    await waitFor(() => expect(mockFetchHealth).toHaveBeenCalled());
    expect(screen.queryByTestId("connection-status")).not.toBeInTheDocument();
  });

  it("shows warning banner when API is unreachable", async () => {
    mockFetchHealth.mockRejectedValue(new Error("connection refused"));
    renderComponent();
    await waitFor(
      () => {
        expect(screen.getByTestId("connection-status")).toBeInTheDocument();
      },
      { timeout: 3000 },
    );
    expect(screen.getByText(/API is unreachable/)).toBeInTheDocument();
  });

  it("has accessible role=status when degraded", async () => {
    mockFetchHealth.mockRejectedValue(new Error("timeout"));
    renderComponent();
    await waitFor(
      () => {
        expect(screen.getByRole("status")).toBeInTheDocument();
      },
      { timeout: 3000 },
    );
  });
});
