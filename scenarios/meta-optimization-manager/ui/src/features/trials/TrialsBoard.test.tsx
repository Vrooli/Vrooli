import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";

import { TrialVerdict } from "@vrooli/proto-types/meta-optimization-manager/v1/trials/trials_pb";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { TrialsBoard } from "./TrialsBoard";

vi.mock("../../api/trials", () => ({
  trialsClient: { getGateCoverage: vi.fn(), getTrialHistory: vi.fn() },
}));

import { trialsClient } from "../../api/trials";

const getGateCoverage = vi.mocked(trialsClient.getGateCoverage);
const getTrialHistory = vi.mocked(trialsClient.getTrialHistory);

describe("TrialsBoard", () => {
  beforeEach(() => {
    getGateCoverage.mockReset();
    getTrialHistory.mockReset();
  });

  it("renders loading while queries are in flight", () => {
    getGateCoverage.mockReturnValue(new Promise(() => {}) as never);
    getTrialHistory.mockReturnValue(new Promise(() => {}) as never);
    renderWithProviders(<TrialsBoard />);
    expect(screen.getByTestId(selectors.trials.loading)).toBeInTheDocument();
  });

  it("renders gate coverage, the trend, and recent runs on success", async () => {
    getGateCoverage.mockResolvedValue({
      gateCoverageRatio: 0.25,
      guideTasksWithGate: 1,
      guideTasksTotal: 4,
    } as never);
    getTrialHistory.mockResolvedValue({
      points: [
        { at: { seconds: 1780000000n, nanos: 0 }, successRate: 0.5, medianTokens: 1000n, medianDurationMs: 5000n, runCount: 2 },
      ],
      recentRuns: [
        { id: "r1", suite: "add-feature", verdict: TrialVerdict.PASS, tokens: 1000n, durationMs: 5000n },
      ],
    } as never);

    renderWithProviders(<TrialsBoard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.trials.coverage)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.trials.point)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.trials.run)).toBeInTheDocument();
  });

  it("renders the empty state when there is no history", async () => {
    getGateCoverage.mockResolvedValue({ gateCoverageRatio: 0, guideTasksWithGate: 0, guideTasksTotal: 0 } as never);
    getTrialHistory.mockResolvedValue({ points: [], recentRuns: [] } as never);
    renderWithProviders(<TrialsBoard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.trials.empty)).toBeInTheDocument();
    });
  });

  it("renders the error state when a query rejects", async () => {
    getGateCoverage.mockRejectedValue(new Error("boom"));
    getTrialHistory.mockResolvedValue({ points: [], recentRuns: [] } as never);
    renderWithProviders(<TrialsBoard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.trials.error)).toBeInTheDocument();
    });
  });
});
