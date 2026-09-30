import { screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../test-utils";
import { defaultRoutes, makeGate, makeOverview, stubConsoleFetch } from "../test-utils/consoleFixtures";
import { DashboardPage } from "./DashboardPage";

describe("DashboardPage", () => {
  afterEach(() => vi.unstubAllGlobals());

  // [REQ:SWBD-P1-005]
  it("says nothing needs attention when no gate is pending", async () => {
    stubConsoleFetch(defaultRoutes());
    renderWithProviders(<DashboardPage />);
    expect(await screen.findByTestId("overview-all-clear")).toBeInTheDocument();
    expect(screen.getByTestId("dashboard-attention-region")).toHaveAttribute("data-experience-state", "empty");
    expect(screen.getByTestId("overview-channel-health")).toHaveTextContent("Switchboard app");
    expect(screen.getByTestId("dashboard-refusals-region")).toHaveAttribute("data-experience-state", "empty");
  });

  // [REQ:SWBD-P1-006]
  it("surfaces pending gates, refusals and budget pressure", async () => {
    const routes = defaultRoutes();
    routes["/api/v1/overview"] = makeOverview({
      gates: [makeGate()],
      refusals: [{ thread_id: "thread-2", channel_id: "telegram", channel_display_name: "Telegram", thread_key: "chat-9", sender_address: "@stranger", agent_id: "household-planner", reason: "no permitted scope remains", at: new Date().toISOString() }],
      budget: { threads_under_pressure: [{ thread_id: "thread-1", channel_id: "in-app", thread_key: "5f2c", agent_id: "household-planner", turn_budget: 20, used: 19, spend_cap_cents: 0, spent_cents: 0, window_started_at: new Date().toISOString(), exhausted: false }] },
    });
    stubConsoleFetch(routes);
    renderWithProviders(<DashboardPage />);
    expect(await screen.findByTestId("overview-gate-item")).toBeInTheDocument();
    expect(screen.getByTestId("dashboard-attention-region")).toHaveAttribute("data-experience-state", "ready");
    expect(screen.getByTestId("overview-refusals")).toHaveTextContent("@stranger");
    expect(screen.getByTestId("overview-budget-meter")).toBeInTheDocument();
    expect(screen.queryByTestId("overview-all-clear")).not.toBeInTheDocument();
  });

  it("renders the error state with a retry when the overview fails", async () => {
    const routes = defaultRoutes();
    routes["/api/v1/overview"] = new Response("boom", { status: 500 });
    stubConsoleFetch(routes);
    renderWithProviders(<DashboardPage />);
    await waitFor(() => expect(screen.getByTestId("dashboard-attention-region")).toHaveAttribute("data-experience-state", "error"));
    expect(screen.getByTestId("dashboard-attention-region-retry")).toBeInTheDocument();
  });
});
