import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";

import { Projection, DenominatorConfidence } from "@vrooli/proto-types/meta-optimization-manager/v1/shared/model_pb";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { ReadinessBoard } from "./ReadinessBoard";

vi.mock("../../api/coverage", () => ({
  coverageClient: { getStatus: vi.fn() },
}));

import { coverageClient } from "../../api/coverage";

const getStatus = vi.mocked(coverageClient.getStatus);

describe("ReadinessBoard", () => {
  beforeEach(() => {
    getStatus.mockReset();
  });

  it("renders the loading state while the coverage query is in flight", () => {
    getStatus.mockReturnValue(new Promise(() => {}) as never);
    renderWithProviders(<ReadinessBoard />);
    expect(screen.getByTestId(selectors.readiness.loading)).toBeInTheDocument();
  });

  it("renders per-projection coverage + the trial trend on success", async () => {
    getStatus.mockResolvedValue({
      projections: [
        {
          projection: Projection.ANSWER,
          coverageRatio: 0.5,
          nowCount: 18,
          totalCells: 36,
          inReachCount: 10,
          missingCount: 8,
          denominatorConfidence: DenominatorConfidence.PARTIAL,
          available: true,
          unavailableReason: "",
        },
        {
          projection: Projection.GUIDE,
          coverageRatio: 0,
          nowCount: 0,
          totalCells: 33,
          inReachCount: 0,
          missingCount: 33,
          denominatorConfidence: DenominatorConfidence.SKETCH,
          available: false,
          unavailableReason: "prompt-manager unreachable",
        },
      ],
      latestTrialTrend: { successRate: 0.75, medianTokens: 1200n },
    } as never);

    renderWithProviders(<ReadinessBoard />);

    await waitFor(() => {
      expect(screen.getAllByTestId(selectors.readiness.projection)).toHaveLength(2);
    });
    expect(screen.getByTestId(selectors.readiness.trend)).toBeInTheDocument();
  });

  it("renders the empty state when no projections come back", async () => {
    getStatus.mockResolvedValue({ projections: [], latestTrialTrend: undefined } as never);
    renderWithProviders(<ReadinessBoard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.readiness.empty)).toBeInTheDocument();
    });
  });

  it("renders the error state when the query rejects", async () => {
    getStatus.mockRejectedValue(new Error("boom"));
    renderWithProviders(<ReadinessBoard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.readiness.error)).toBeInTheDocument();
    });
  });
});
