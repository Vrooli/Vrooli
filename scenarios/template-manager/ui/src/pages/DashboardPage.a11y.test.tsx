import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { expectNoA11yViolations, renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";

import { DashboardPage } from "./DashboardPage";

const fetchTemplateDashboard = vi.fn();

vi.mock("../api/templateDomain", () => ({
  fetchTemplateDashboard: () => fetchTemplateDashboard(),
}));

describe("DashboardPage accessibility", () => {
  it("renders the loaded dashboard without axe violations", async () => {
    fetchTemplateDashboard.mockResolvedValueOnce({
      templates: { templates: [{ id: "react-vite", kind: 1, version: "1.6.0", status: "active", versionLag: { lagCount: 0 } }] },
      runs: { runs: [] },
      drift: { snapshots: [] },
      debt: { entries: [] },
      monitor: {
        status: {
          enabled: true,
          intervalSeconds: 86400n,
          inFlight: false,
          lastStatus: "scheduled",
          lastRunId: "",
          greenStreak: 0n,
        },
      },
      openDebt: { count: 0n },
      deepStreak: { streak: 1n },
      standing: { buckets: [{ standing: "current", count: 1n }] },
      maxLag: { lag: 0n },
    });

    const { container } = renderWithProviders(<DashboardPage />);
    const dashboard = await screen.findByTestId(selectors.pages.dashboard);
    await waitFor(() => expect(dashboard).toHaveTextContent("react-vite"));

    await expectNoA11yViolations(container);
  });
});
