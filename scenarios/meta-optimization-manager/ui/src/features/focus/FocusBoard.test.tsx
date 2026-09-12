import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor } from "@testing-library/react";

import { Projection, CellStatus, GapAxis } from "@vrooli/proto-types/meta-optimization-manager/v1/shared/model_pb";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { FocusBoard } from "./FocusBoard";

vi.mock("../../api/focus", () => ({
  focusClient: { getFocus: vi.fn(), listGaps: vi.fn(), listCondition: vi.fn() },
}));

import { focusClient } from "../../api/focus";

const getFocus = vi.mocked(focusClient.getFocus);
const listGaps = vi.mocked(focusClient.listGaps);
const listCondition = vi.mocked(focusClient.listCondition);

describe("FocusBoard", () => {
  beforeEach(() => {
    getFocus.mockReset();
    listGaps.mockReset();
    listCondition.mockReset();
  });

  it("renders loading while queries are in flight", () => {
    getFocus.mockReturnValue(new Promise(() => {}) as never);
    listGaps.mockReturnValue(new Promise(() => {}) as never);
    listCondition.mockReturnValue(new Promise(() => {}) as never);
    renderWithProviders(<FocusBoard />);
    expect(screen.getByTestId(selectors.focus.loading)).toBeInTheDocument();
  });

  it("renders ranked focus items + the gaps registry on success", async () => {
    getFocus.mockResolvedValue({
      items: [
        {
          gap: { id: "answer/1", projection: Projection.ANSWER, axis: GapAxis.COVERAGE, title: "explain domain map", status: CellStatus.MISSING },
          impact: 1,
          importance: 1,
          priorityScore: 1,
          rationale: "missing × answer",
        },
      ],
    } as never);
    listGaps.mockResolvedValue({
      gaps: [
        {
          id: "answer/1",
          projection: Projection.ANSWER,
          axis: GapAxis.COVERAGE,
          title: "explain domain map",
          status: CellStatus.MISSING,
          global: false,
          approaches: ["cartographer provider"],
        },
        {
          id: "global-x",
          projection: Projection.UNSPECIFIED,
          axis: GapAxis.EMPIRICAL,
          title: "typed contracts everywhere",
          status: CellStatus.MISSING,
          global: true,
          approaches: [],
        },
      ],
    } as never);
    listCondition.mockResolvedValue({
      gaps: [{ id: "condition/provider-a", title: "provider-a is degraded", notes: ["degradation_rate=1.0"] }],
    } as never);

    renderWithProviders(<FocusBoard />);
    await waitFor(() => {
      expect(screen.getAllByTestId(selectors.focus.item)).toHaveLength(1);
    });
    expect(screen.getAllByTestId(selectors.focus.gap)).toHaveLength(2);
    expect(screen.getByText(/cartographer provider/)).toBeInTheDocument();
    expect(screen.getByTestId("condition-finding")).toHaveTextContent("provider-a is degraded");
    expect(screen.getAllByTestId("focus-axis").length).toBeGreaterThan(0);
  });

  it("renders the empty state when nothing comes back", async () => {
    getFocus.mockResolvedValue({ items: [] } as never);
    listGaps.mockResolvedValue({ gaps: [] } as never);
    listCondition.mockResolvedValue({ gaps: [] } as never);
    renderWithProviders(<FocusBoard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.focus.empty)).toBeInTheDocument();
    });
  });

  it("renders the error state when a query rejects", async () => {
    getFocus.mockRejectedValue(new Error("boom"));
    listGaps.mockResolvedValue({ gaps: [] } as never);
    listCondition.mockResolvedValue({ gaps: [] } as never);
    renderWithProviders(<FocusBoard />);
    await waitFor(() => {
      expect(screen.getByTestId(selectors.focus.error)).toBeInTheDocument();
    });
  });
});
