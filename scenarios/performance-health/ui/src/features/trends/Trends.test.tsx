import { describe, expect, it, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { ScenarioProvider } from "../perf/ScenarioContext";
import { TrendsView } from "./TrendsView";

const scanFleet = vi.fn();
const getTrend = vi.fn();
const getStartupTrend = vi.fn();

vi.mock("../../api/perf", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/perf")>();
  return {
    ...actual,
    perfClient: {
      scanFleet: (...a: unknown[]) => scanFleet(...a),
      getTrend: (...a: unknown[]) => getTrend(...a),
      getStartupTrend: (...a: unknown[]) => getStartupTrend(...a),
    },
  };
});

const renderTrends = () =>
  renderWithProviders(
    <ScenarioProvider>
      <TrendsView />
    </ScenarioProvider>,
  );

beforeEach(() => {
  vi.clearAllMocks();
  scanFleet.mockResolvedValue({
    entries: [{ scenario: "performance-health", tier: "1" }],
    tierDistribution: [],
    errors: [],
    scenarioCount: 1,
    noBudgetCount: 0,
    regressedCount: 0,
  });
  getStartupTrend.mockResolvedValue({ measurements: [] });
});

describe("TrendsView (cimode — copy-independent)", () => {
  it("renders metric cards + samples table from GetTrend", async () => {
    getTrend.mockResolvedValue({
      samples: [
        {
          capturedAt: "2026-06-21T00:00:00Z",
          goBuildMs: 1200n,
          uiBuildMs: 5000n,
          bundleBytes: 2_000_000n,
          lcpMs: 800n,
          p95Ms: 120n,
          slowestComponent: "List",
          slowestComponentAvgMs: 4.2,
          note: "first sample",
        },
      ],
    });
    getStartupTrend.mockResolvedValue({
      measurements: [{ timeToHealthyMs: 3000n }],
    });

    renderTrends();
    await waitFor(() =>
      expect(screen.getByTestId(selectors.trends.charts)).toBeInTheDocument(),
    );
    expect(screen.getByTestId(selectors.trends.cardGoBuild)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.trends.cardComponent)).toHaveTextContent("List");
    expect(screen.getByTestId(selectors.trends.cardStartup)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.trends.samples)).toHaveTextContent("first sample");
  });

  it("shows the empty state with a CTA when there are no samples", async () => {
    getTrend.mockResolvedValue({ samples: [] });
    renderTrends();
    const block = await screen.findByTestId(selectors.trends.empty);
    // The empty state offers the run-an-audit CTA (rendered as a router Link).
    expect(block.querySelector('a[href="/audit"]')).not.toBeNull();
  });

  it("shows an actionable error state when GetTrend fails", async () => {
    getTrend.mockRejectedValue(new Error("trend boom"));
    renderTrends();
    await waitFor(() =>
      expect(screen.getByTestId(selectors.trends.error)).toBeInTheDocument(),
    );
    expect(screen.getByTestId(selectors.state.errorRetry)).toBeInTheDocument();
  });

  it("renders a loading skeleton before data resolves", async () => {
    let resolve!: (v: { samples: never[] }) => void;
    getTrend.mockReturnValue(new Promise((r) => (resolve = r)));
    renderTrends();
    expect(await screen.findByTestId(selectors.state.loading)).toBeInTheDocument();
    resolve({ samples: [] });
  });
});
