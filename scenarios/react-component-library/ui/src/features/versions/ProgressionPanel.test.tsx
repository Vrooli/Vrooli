import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import { QueryClient } from "@tanstack/react-query";

import { selectors } from "../../consts/selectors";
import { listVersionLedger } from "../../api/versionLedger";
import { ProgressionPanel } from "./ProgressionPanel";
import { renderWithProviders } from "../../test-utils";

vi.mock("../../api/versionLedger", () => ({
  listVersionLedger: vi.fn(),
}));

function renderPanel() {
  return renderWithProviders(<ProgressionPanel libraryId="react-component-library:Chart" />, {
    queryClient: new QueryClient({
      defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
    }),
  });
}

describe("ProgressionPanel", () => {
  beforeEach(() => vi.mocked(listVersionLedger).mockReset());
  afterEach(() => cleanup());

  it("keeps a retired version on the chart and labels it", async () => {
    vi.mocked(listVersionLedger).mockResolvedValue([
      {
        libraryId: "react-component-library:Chart",
        version: "1.0.0",
        createdAt: "2026-01-01T00:00:00Z",
        releasedAt: "2026-01-02T00:00:00Z",
        retiredAt: "",
        lifecycleState: "released",
        gatePassCount: 4,
        gateFailCount: 0,
        testRuns: 3,
        testPassRate: 1,
        adoptionCurrent: 2,
        adoptionPeak: 2,
        fileCount: 1,
        linesOfCode: 40,
        dependencyCount: 0,
        presence: "materialized",
      },
      {
        libraryId: "react-component-library:Chart",
        version: "0.9.0",
        createdAt: "2025-12-01T00:00:00Z",
        releasedAt: "2025-12-01T00:00:00Z",
        retiredAt: "2026-01-03T00:00:00Z",
        lifecycleState: "retired",
        gatePassCount: 3,
        gateFailCount: 1,
        testRuns: 4,
        testPassRate: 0.75,
        adoptionCurrent: 0,
        adoptionPeak: 1,
        fileCount: 1,
        linesOfCode: 32,
        dependencyCount: 0,
        presence: "evicted",
      },
    ]);

    renderPanel();

    const panel = await screen.findByTestId(selectors.versions.progressionPanel);
    await waitFor(() => expect(panel.querySelector("[data-rcl-chart]")).toBeTruthy());
    const chart = panel.querySelector("[data-rcl-chart]") as HTMLElement;
    expect(chart).toHaveTextContent("0.9.0");
    expect(panel).toHaveTextContent("retired");
    expect(within(chart).getByRole("button", { name: /0\.9\.0/ })).toBeInTheDocument();
    expect(within(chart).getByRole("table")).toBeInTheDocument();
    expect(listVersionLedger).toHaveBeenCalledWith("react-component-library:Chart");
  });

  it("reports loading while the provider is pending", async () => {
    vi.mocked(listVersionLedger).mockImplementation(
      () => new Promise((resolve) => setTimeout(() => resolve([]), 25)),
    );
    renderPanel();
    expect(screen.getByRole("status")).toHaveTextContent("componentDetail.progression.loading");
    const panel = await screen.findByTestId(selectors.versions.progressionPanel);
    await waitFor(() => expect(panel.querySelector("[data-rcl-chart]")).toBeTruthy());
  });

  it("reports provider failures", async () => {
    vi.mocked(listVersionLedger).mockRejectedValueOnce(new Error("offline"));
    renderPanel();
    expect(await screen.findByRole("alert")).toHaveTextContent("componentDetail.progression.error");
  });
});
