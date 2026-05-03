import { render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it } from "vitest";
import { ByPhaseView } from "./ByPhaseView";
import { OPERATIONS_LANES } from "../../../types/operations";
import type { ActivityRow } from "../../../types/operations";
import { selectors } from "../../../consts/selectors";

function row(overrides: Partial<ActivityRow>): ActivityRow {
  return {
    activityId: "a-x",
    runId: "run-x",
    ownerType: "backlog",
    ownerName: "fix-foo",
    purpose: "process",
    status: "running",
    requestedAt: "2026-05-02T01:00:00Z",
    ...overrides,
  };
}

function renderView(rows: ActivityRow[]) {
  return render(
    <MemoryRouter>
      <ByPhaseView activities={rows} />
    </MemoryRouter>,
  );
}

describe("ByPhaseView", () => {
  it("renders all four canonical lanes in declared order", () => {
    renderView([]);
    const columns = screen.getAllByTestId(
      selectors.operationsCenter.byPhaseColumn,
    );
    expect(columns).toHaveLength(4);
    expect(columns.map((c) => c.getAttribute("data-lane"))).toEqual([
      ...OPERATIONS_LANES,
    ]);
  });

  it("places each activity in the column matching its lane", () => {
    renderView([
      row({
        activityId: "a-1",
        runId: "run-1",
        lane: "investigate",
        ownerTitle: "Workshop A",
      }),
      row({
        activityId: "a-2",
        runId: "run-2",
        lane: "execute",
        ownerTitle: "Backlog B",
      }),
      row({
        activityId: "a-3",
        runId: "run-3",
        lane: "review",
        ownerTitle: "Review C",
      }),
      row({
        activityId: "a-4",
        runId: "run-4",
        lane: "reconcile",
        ownerTitle: "Reconcile D",
      }),
    ]);

    const columns = screen.getAllByTestId(
      selectors.operationsCenter.byPhaseColumn,
    );
    const byLane = (lane: string) => {
      const found = columns.find((c) => c.getAttribute("data-lane") === lane);
      if (!found) throw new Error(`column for lane ${lane} not rendered`);
      return found;
    };

    expect(within(byLane("investigate")).getByText("Workshop A")).toBeInTheDocument();
    expect(within(byLane("execute")).getByText("Backlog B")).toBeInTheDocument();
    expect(within(byLane("review")).getByText("Review C")).toBeInTheDocument();
    expect(within(byLane("reconcile")).getByText("Reconcile D")).toBeInTheDocument();
  });

  it("renders lane row counts in the column header", () => {
    renderView([
      row({ activityId: "a-1", runId: "run-1", lane: "investigate" }),
      row({ activityId: "a-2", runId: "run-2", lane: "investigate" }),
      row({ activityId: "a-3", runId: "run-3", lane: "execute" }),
    ]);
    const columns = screen.getAllByTestId(
      selectors.operationsCenter.byPhaseColumn,
    );
    const byLane = (lane: string) =>
      columns.find((c) => c.getAttribute("data-lane") === lane)!;
    expect(
      within(byLane("investigate")).getByTestId(
        selectors.operationsCenter.byPhaseColumnHeader,
      ).textContent,
    ).toContain("2");
    expect(
      within(byLane("execute")).getByTestId(
        selectors.operationsCenter.byPhaseColumnHeader,
      ).textContent,
    ).toContain("1");
    expect(
      within(byLane("review")).getByTestId(
        selectors.operationsCenter.byPhaseColumnHeader,
      ).textContent,
    ).toContain("0");
  });

  it("renders an empty placeholder when a lane has no activity", () => {
    renderView([
      row({ activityId: "a-1", runId: "run-1", lane: "execute" }),
    ]);
    const empties = screen.getAllByTestId(
      selectors.operationsCenter.byPhaseColumnEmpty,
    );
    // Three lanes empty (investigate / review / reconcile) when only execute
    // has a row.
    expect(empties).toHaveLength(3);
    expect(empties.every((el) => el.textContent?.includes("No active"))).toBe(true);
  });

  it("drops activities without a lane silently", () => {
    renderView([
      row({ activityId: "a-1", runId: "run-1", ownerTitle: "Mystery row" }),
    ]);
    expect(screen.queryByText("Mystery row")).toBeNull();
    // All four columns are empty.
    expect(
      screen.getAllByTestId(selectors.operationsCenter.byPhaseColumnEmpty),
    ).toHaveLength(4);
  });

  it("drops activities whose lane is not canonical", () => {
    renderView([
      row({
        activityId: "a-1",
        runId: "run-1",
        lane: "wandering" as unknown as string,
        ownerTitle: "Wanderer",
      }),
    ]);
    expect(screen.queryByText("Wanderer")).toBeNull();
    expect(
      screen.getAllByTestId(selectors.operationsCenter.byPhaseColumnEmpty),
    ).toHaveLength(4);
  });

  it("hides the lane chip on rows inside the column (column header conveys the lane)", () => {
    renderView([
      row({
        activityId: "a-1",
        runId: "run-1",
        lane: "execute",
        ownerTitle: "Backlog item",
      }),
    ]);
    // ActivityRow renders the lane chip when showLane=true. The
    // by-phase column passes showLane=false, so the lane label appears
    // exactly once on the page (the column header).
    const matches = screen.getAllByText("Execute");
    expect(matches).toHaveLength(1);
  });

  it("uses listitem semantics so the board is keyboard / a11y traversable", () => {
    renderView([]);
    const board = screen.getByTestId(selectors.operationsCenter.byPhaseBoard);
    expect(board).toHaveAttribute("role", "list");
    const columns = within(board).getAllByRole("listitem");
    expect(columns).toHaveLength(4);
  });
});
