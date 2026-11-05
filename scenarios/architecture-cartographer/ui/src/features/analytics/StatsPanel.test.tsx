import { cleanup, fireEvent, screen, within } from "@testing-library/react";
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

    const error = screen.getByTestId(selectors.features.analytics.stats.error);
    expect(error).toHaveTextContent("stats unavailable");
    fireEvent.click(within(error).getByRole("button"));
    expect(refetch).toHaveBeenCalledTimes(1);
  });

  it("renders default zero values when the API omits stats", () => {
    mockStats({ data: { stats: undefined } as never });

    renderWithProviders(<StatsPanel scenario="demo" />);

    const root = screen.getByTestId(selectors.features.analytics.stats.root);
    expect(root).toBeInTheDocument();
    expect(root.textContent.match(/0/g)?.length ?? 0).toBeGreaterThanOrEqual(6);
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

    const root = screen.getByTestId(selectors.features.analytics.stats.root);
    expect(root).toHaveTextContent("8");
    expect(root).toHaveTextContent("13");
    expect(root).toHaveTextContent("88%");
    expect(screen.queryByTestId(selectors.features.analytics.stats.suppressed)).not.toBeInTheDocument();
  });
});
