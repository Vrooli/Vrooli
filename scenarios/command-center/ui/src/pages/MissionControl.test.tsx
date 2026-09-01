import { describe, expect, it, vi } from "vitest";
import { screen, waitFor } from "@testing-library/react";
import { renderWithProviders as render } from "../test-utils/renderWithProviders";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import type { DashboardResponse } from "../lib/api";

vi.mock("@react-three/fiber", () => ({
  Canvas: ({ children }: { children: ReactNode }) => (
    <div data-testid="canvas">{children}</div>
  ),
  useFrame: () => undefined,
}));

vi.mock("@react-three/drei", () => ({
  Points: ({ children }: { children?: ReactNode }) => <div>{children}</div>,
  PointMaterial: () => null,
}));

import MissionControl from "./MissionControl";

const seededDashboard: DashboardResponse = {
  dashboard: "mission-control",
  generated_at: "2026-04-18T12:00:00Z",
  metrics: [
    {
      id: "active_scenarios",
      label: "Active scenarios",
      dataSource: "live",
      upstreamSource: "vrooli",
      description: "Count of scenarios currently running.",
    },
    {
      id: "revenue_mrr",
      label: "Monthly recurring revenue",
      dataSource: "gap",
      upstreamSource: "lpbs",
      description: "Rolling MRR from Stripe subscriptions.",
      whatIsNeeded: "LPBS admin dashboard endpoint pending.",
    },
  ],
  sources: {
    vrooli: { from_cache: false, staleness_ts: null },
    lpbs: { from_cache: false, staleness_ts: null },
  },
};

function renderWithClient(): QueryClient {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  client.setQueryData(["dashboard", "mission-control"], seededDashboard);
  render(
    <QueryClientProvider client={client}>
      <MissionControl />
    </QueryClientProvider>,
  );
  return client;
}

describe("MissionControl page", () => {
  it("renders the title and at least one gap badge for non-live metrics", async () => {
    renderWithClient();
    await waitFor(() => {
      expect(screen.getByText("Mission Control")).toBeInTheDocument();
    });
    const gapBadges = screen.getAllByTestId("gap-badge");
    expect(gapBadges.length).toBeGreaterThanOrEqual(1);
  });
});
