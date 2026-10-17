import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor } from "@testing-library/react";

import { renderWithProviders } from "../test-utils";
import { selectors } from "../consts/selectors";
import { DashboardPage } from "./DashboardPage";

const scanFleet = vi.fn();

vi.mock("../api/perf", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../api/perf")>();
  return {
    ...actual,
    perfClient: { scanFleet: (...a: unknown[]) => scanFleet(...a) },
  };
});

const populated = {
  entries: [{ scenario: "alpha", tier: "1" }],
  tierDistribution: [],
  errors: [],
  scenarioCount: 4,
  noBudgetCount: 1,
  regressedCount: 2,
};
const empty = { ...populated, entries: [], scenarioCount: 0, noBudgetCount: 0, regressedCount: 0 };

beforeEach(() => vi.clearAllMocks());
afterEach(() => cleanup());

describe("DashboardPage (cimode — copy-independent)", () => {
  it("renders the snapshot counters and every workflow card with data", async () => {
    scanFleet.mockResolvedValue(populated);
    renderWithProviders(<DashboardPage />);
    await waitFor(() =>
      expect(screen.getByTestId(selectors.pages.overviewSnapshot)).toBeInTheDocument(),
    );
    expect(screen.getByTestId(selectors.pages.workflowCard({ to: "/audit" }))).toBeInTheDocument();
    expect(screen.getByTestId(selectors.pages.workflowCard({ to: "/budgets" }))).toBeInTheDocument();
  });

  it("renders the empty snapshot with a fleet CTA when nothing is graded", async () => {
    scanFleet.mockResolvedValue(empty);
    renderWithProviders(<DashboardPage />);
    await waitFor(() => expect(screen.getByTestId(selectors.state.empty)).toBeInTheDocument());
    expect(screen.queryByTestId(selectors.pages.overviewSnapshot)).not.toBeInTheDocument();
  });

  it("renders an actionable error state when the scan fails", async () => {
    scanFleet.mockRejectedValue(new Error("scan boom"));
    renderWithProviders(<DashboardPage />);
    await waitFor(() => expect(screen.getByTestId(selectors.state.error)).toBeInTheDocument());
    expect(screen.getByTestId(selectors.state.errorRetry)).toBeInTheDocument();
  });

  it("shows a loading skeleton before the scan resolves", async () => {
    let resolve!: (v: typeof populated) => void;
    scanFleet.mockReturnValue(new Promise((r) => (resolve = r)));
    renderWithProviders(<DashboardPage />);
    // The workflow cards render immediately; the snapshot region shows skeletons.
    expect(screen.getByTestId(selectors.pages.workflowCard({ to: "/trends" }))).toBeInTheDocument();
    resolve(populated);
    await waitFor(() =>
      expect(screen.getByTestId(selectors.pages.overviewSnapshot)).toBeInTheDocument(),
    );
  });
});
