import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RecoveryTimeline } from "./RecoveryTimeline";

// [REQ:UI-003] Recovery event timeline
vi.mock("../lib/api", () => ({
  fetchRecoveryEvents: vi.fn().mockResolvedValue([]),
}));

function wrapper({ children }: { children: React.ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

describe("RecoveryTimeline", () => {
  it("renders the recovery events heading", () => {
    render(<RecoveryTimeline />, { wrapper });
    expect(screen.getByText("Recovery Events")).toBeInTheDocument();
  });

  it("shows loading state initially", () => {
    render(<RecoveryTimeline />, { wrapper });
    expect(screen.getByText(/loading recovery events/i)).toBeInTheDocument();
  });
});
