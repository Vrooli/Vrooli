import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";

import { DashboardPage } from "./DashboardPage";

const fetchTemplateDashboard = vi.fn();

vi.mock("../api/templateDomain", () => ({
  fetchTemplateDashboard: () => fetchTemplateDashboard(),
}));

describe("DashboardPage", () => {
  it("renders live template dashboard surfaces", async () => {
    fetchTemplateDashboard.mockResolvedValueOnce({
      templates: {
        templates: [
          { id: "react-vite", kind: 1, version: "1.6.0", status: "active", versionLag: { lagCount: 0 } },
          { id: "minimal-resource", kind: 3, version: "1.0.0", status: "active", versionLag: { lagCount: 1 } },
        ],
      },
      runs: {
        runs: [{ id: "validation-1", templateId: "react-vite", mode: 2, status: "passed", findings: [] }],
      },
      drift: {
        snapshots: [{ id: "drift-1", templateId: "react-vite", target: "fleet", status: "drifted", driftCount: 4 }],
      },
      debt: {
        entries: [{ key: "react-vite.aria", severity: "medium", status: "open", title: "Missing aria label" }],
      },
      monitor: {
        status: {
          enabled: true,
          intervalSeconds: 86400n,
          inFlight: false,
          lastStatus: "passed",
          lastRunId: "validation-1",
          greenStreak: 3n,
        },
      },
      openDebt: { count: 1n },
      deepStreak: { streak: 3n },
      standing: {
        buckets: [
          { standing: "current", count: 1n },
          { standing: "open_debt", count: 1n },
        ],
      },
      maxLag: { lag: 1n },
    });

    renderWithProviders(<DashboardPage />);

    const dashboard = await screen.findByTestId(selectors.pages.dashboard);
    await waitFor(() => expect(dashboard).toHaveTextContent("react-vite"));
    expect(dashboard).toHaveTextContent("react-vite");
    expect(dashboard).toHaveTextContent("Missing aria label");
    expect(dashboard).toHaveTextContent("validation-1");
    expect(dashboard).toHaveTextContent("fleet");
  });

  it("renders loading state", () => {
    fetchTemplateDashboard.mockReturnValueOnce(new Promise(() => {}));

    renderWithProviders(<DashboardPage />);

    expect(screen.getByText(strings.dashboard.loadingTitle)).toBeInTheDocument();
  });

  it("renders error state", async () => {
    fetchTemplateDashboard.mockRejectedValueOnce(new Error("boom"));

    renderWithProviders(<DashboardPage />);

    await waitFor(() => expect(screen.getByText(strings.dashboard.errorTitle)).toBeInTheDocument());
  });
});
