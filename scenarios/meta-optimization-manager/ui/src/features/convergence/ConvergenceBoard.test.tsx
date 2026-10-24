import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";

import {
  FitnessTier,
  ReferenceEligibility,
} from "@vrooli/proto-types/meta-optimization-manager/v1/convergence/convergence_pb";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { ConvergenceBoard } from "./ConvergenceBoard";

vi.mock("../../api/convergence", () => ({
  convergenceClient: { getConvergenceStatus: vi.fn() },
}));

import { convergenceClient } from "../../api/convergence";

const getStatus = vi.mocked(convergenceClient.getConvergenceStatus);

describe("ConvergenceBoard", () => {
  beforeEach(() => {
    getStatus.mockReset();
  });

  it("renders loading while the query is in flight", () => {
    getStatus.mockReturnValue(new Promise(() => {}) as never);
    renderWithProviders(<ConvergenceBoard />);
    expect(screen.getByTestId(selectors.convergence.loading)).toBeInTheDocument();
  });

  it("renders template fitness + reference health on success", async () => {
    getStatus.mockResolvedValue({
      templates: [
        {
          template: "react-vite",
          perReplicaCost: 1200,
          driftSurfaceCount: 2,
          commentOnlyContractCount: 1,
          coordinatedEditCount: 5,
          tier: FitnessTier.FAIR,
        },
      ],
      references: [
        {
          scenario: "reference-react-vite",
          staleFromTemplate: false,
          cleanOnAllTools: true,
          stabilityDays: 61,
          breadth: 3,
          eligibility: ReferenceEligibility.ELIGIBLE,
        },
      ],
    } as never);

    renderWithProviders(<ConvergenceBoard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.convergence.template)).toBeInTheDocument();
    });
    expect(screen.getByTestId(selectors.convergence.reference)).toBeInTheDocument();
    expect(screen.getAllByText(/react-vite/).length).toBeGreaterThan(0);
  });

  it("renders the empty state when nothing comes back", async () => {
    getStatus.mockResolvedValue({ templates: [], references: [] } as never);
    renderWithProviders(<ConvergenceBoard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.convergence.empty)).toBeInTheDocument();
    });
  });

  it("renders the error state when the query rejects", async () => {
    getStatus.mockRejectedValue(new Error("boom"));
    renderWithProviders(<ConvergenceBoard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.convergence.error)).toBeInTheDocument();
    });
  });
});
