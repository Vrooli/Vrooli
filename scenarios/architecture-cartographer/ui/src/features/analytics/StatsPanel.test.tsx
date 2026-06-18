import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("./controllers/useAnalyticsController", () => ({
  useStats: vi.fn(),
}));

import { selectors } from "../../consts/selectors";
import { renderWithProviders } from "../../test-utils";
import { StatsPanel } from "./StatsPanel";
import { useStats } from "./controllers/useAnalyticsController";

afterEach(() => {
  cleanup();
  vi.mocked(useStats).mockReset();
});

function mockStats(state: Partial<ReturnType<typeof useStats>>) {
  vi.mocked(useStats).mockReturnValue({
    isPending: false,
    isError: false,
    data: { stats: undefined },
    error: null,
    refetch: vi.fn(),
    ...state,
  } as unknown as ReturnType<typeof useStats>);
}

describe("StatsPanel", () => {
  it("renders loading and retryable error states", () => {
    mockStats({ isPending: true });
    const { rerender } = renderWithProviders(<StatsPanel scenario="demo" />);
    expect(screen.getByTestId(selectors.features.analytics.stats.loading)).toBeInTheDocument();

    const refetch = vi.fn();
    mockStats({ isError: true, error: new Error("stats unavailable"), refetch });
    rerender(<StatsPanel scenario="demo" />);

    expect(screen.getByTestId(selectors.features.analytics.stats.error)).toBeInTheDocument();
    expect(screen.getByText("stats unavailable")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /retry/i }));
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it("renders default zero values when the API omits stats", () => {
    mockStats({ data: { stats: undefined } as never });

    renderWithProviders(<StatsPanel scenario="demo" />);

    expect(screen.getByTestId(selectors.features.analytics.stats.root)).toBeInTheDocument();
    expect(screen.getAllByText("0").length).toBeGreaterThanOrEqual(6);
    expect(screen.getByTestId(selectors.features.analytics.stats.suppressed)).toBeInTheDocument();
  });

  it("renders concrete stats and unsuppressed verdict rate", () => {
    mockStats({
      data: {
        stats: {
          conflictsDetected: 8,
          conflictsResolved: 5,
          conflictsForceResolved: 1,
          placementsAuto: 13,
          placementsSuggest: 2,
          overrides: 3,
          verdictSuccessRate: 0.875,
          verdictSuccessRateSuppressed: false,
          verdictObservationCount: 64,
        },
      } as never,
    });

    renderWithProviders(<StatsPanel scenario="demo" />);

    expect(screen.getByText("8")).toBeInTheDocument();
    expect(screen.getByText("13")).toBeInTheDocument();
    expect(screen.getByText("88%")).toBeInTheDocument();
    expect(screen.queryByTestId(selectors.features.analytics.stats.suppressed)).not.toBeInTheDocument();
  });
});
