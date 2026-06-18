import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { ApplyStatus } from "@vrooli/proto-types/architecture-cartographer/v1/apply/apply_pb";

vi.mock("./controllers/useApplyController", () => ({
  useApplyHistory: vi.fn(),
}));

import { selectors } from "../../consts/selectors";
import { renderWithProviders } from "../../test-utils";
import { ApplyHistoryPanel } from "./ApplyHistoryPanel";
import { useApplyHistory } from "./controllers/useApplyController";

afterEach(() => {
  cleanup();
  vi.mocked(useApplyHistory).mockReset();
});

function mockHistory(state: Partial<ReturnType<typeof useApplyHistory>>) {
  vi.mocked(useApplyHistory).mockReturnValue({
    isPending: false,
    isError: false,
    data: { runs: [] },
    error: null,
    refetch: vi.fn(),
    ...state,
  } as unknown as ReturnType<typeof useApplyHistory>);
}

describe("ApplyHistoryPanel", () => {
  it("renders loading, retryable error, and empty states", () => {
    mockHistory({ isPending: true });
    const { rerender } = renderWithProviders(<ApplyHistoryPanel scenario="demo" domain="graph" />);
    expect(screen.getByTestId(selectors.features.apply.history.loading)).toBeInTheDocument();

    const refetch = vi.fn();
    mockHistory({ isError: true, error: new Error("history unavailable"), refetch });
    rerender(<ApplyHistoryPanel scenario="demo" domain="graph" />);
    expect(screen.getByTestId(selectors.features.apply.history.error)).toBeInTheDocument();
    expect(screen.getByText("history unavailable")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: /retry/i }));
    expect(refetch).toHaveBeenCalledTimes(1);

    mockHistory({ data: { runs: [] } as never });
    rerender(<ApplyHistoryPanel scenario="demo" domain="graph" />);
    expect(screen.getByTestId(selectors.features.apply.history.empty)).toBeInTheDocument();
  });

  it("renders status labels for all apply run states", () => {
    mockHistory({
      data: {
        runs: [
          ApplyStatus.UNSPECIFIED,
          ApplyStatus.PLANNED,
          ApplyStatus.RUNNING,
          ApplyStatus.BUILD_GREEN,
          ApplyStatus.BUILD_RED,
          ApplyStatus.REVERTED,
          ApplyStatus.COMMITTED,
          99 as ApplyStatus,
        ].map((status, index) => ({
          id: index === 7 ? "" : `run-${index}`,
          scenario: "demo",
          domain: index === 7 ? "fallback-domain" : "graph",
          status,
        })),
      } as never,
    });

    renderWithProviders(<ApplyHistoryPanel scenario="demo" domain="graph" />);

    expect(screen.getByTestId(selectors.features.apply.history.root)).toBeInTheDocument();
    expect(screen.getAllByRole("row")).toHaveLength(9);
    expect(screen.getByText("run-1")).toBeInTheDocument();
    expect(screen.getByText("fallback-domain")).toBeInTheDocument();
    expect(screen.getAllByText("apply.status.baseline_captured").length).toBeGreaterThanOrEqual(2);
    expect(screen.getByText("apply.status.plan_generated")).toBeInTheDocument();
    expect(screen.getAllByText("apply.status.applied").length).toBeGreaterThanOrEqual(2);
    expect(screen.getAllByText("apply.status.refused_build_break").length).toBeGreaterThanOrEqual(2);
    expect(screen.getByText("apply.status.committed")).toBeInTheDocument();
  });
});
