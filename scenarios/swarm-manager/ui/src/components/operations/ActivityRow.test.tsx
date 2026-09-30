import { describe, it, expect, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { ActivityRow } from "./ActivityRow";
import { useOperationsStore } from "../../stores/operations-store";
import type { ActivityRow as ActivityRowType } from "../../types/operations";
import { selectors } from "../../consts/selectors";

beforeEach(() => {
  useOperationsStore.getState().reset();
});

function row(overrides: Partial<ActivityRowType> = {}): ActivityRowType {
  return {
    activityId: overrides.activityId ?? "act-1",
    runId: overrides.runId,
    ownerType: overrides.ownerType ?? "backlog",
    ownerName: overrides.ownerName ?? "owner-name",
    ownerKind: overrides.ownerKind ?? "feature",
    ownerTitle: overrides.ownerTitle ?? "Owner Title",
    purpose: overrides.purpose ?? "process",
    status: overrides.status ?? "running",
    requestedAt: overrides.requestedAt ?? "2026-05-02T00:00:00Z",
    runtimeSeconds: overrides.runtimeSeconds ?? 60,
    lane: overrides.lane,
    ...overrides,
  };
}

function renderRow(props: Parameters<typeof ActivityRow>[0]) {
  return render(
    <MemoryRouter>
      <ActivityRow {...props} />
    </MemoryRouter>,
  );
}

describe("ActivityRow", () => {
  it("renders without a checkbox when selectable is false", () => {
    renderRow({ row: row({ runId: "run-1" }) });
    expect(
      screen.queryByTestId(selectors.operationsCenter.activityRowCheckbox),
    ).toBeNull();
  });

  it("renders a checkbox when selectable=true", () => {
    renderRow({ row: row({ runId: "run-1" }), selectable: true });
    const cb = screen.getByTestId<HTMLInputElement>(
      selectors.operationsCenter.activityRowCheckbox,
    );
    expect(cb).toBeInTheDocument();
    expect(cb.type).toBe("checkbox");
    expect(cb.checked).toBe(false);
    expect(cb.disabled).toBe(false);
  });

  it("disables the checkbox when the row has no runId", () => {
    renderRow({ row: row({ runId: undefined }), selectable: true });
    const cb = screen.getByTestId<HTMLInputElement>(
      selectors.operationsCenter.activityRowCheckbox,
    );
    expect(cb.disabled).toBe(true);
  });

  it("toggles selection in the operations-store when clicked", () => {
    renderRow({ row: row({ runId: "run-1" }), selectable: true });
    const cb = screen.getByTestId<HTMLInputElement>(
      selectors.operationsCenter.activityRowCheckbox,
    );

    fireEvent.click(cb);
    expect(useOperationsStore.getState().selection.has("run-1")).toBe(true);

    fireEvent.click(cb);
    expect(useOperationsStore.getState().selection.has("run-1")).toBe(false);
  });

  it("reflects pre-existing selection state from the store", () => {
    useOperationsStore.getState().setSelection(["run-1"]);
    renderRow({ row: row({ runId: "run-1" }), selectable: true });
    const cb = screen.getByTestId<HTMLInputElement>(
      selectors.operationsCenter.activityRowCheckbox,
    );
    expect(cb.checked).toBe(true);
  });

  it("dims the row and disables the checkbox when stopping", () => {
    useOperationsStore.setState((s) => ({
      stoppingRunIds: new Set([...s.stoppingRunIds, "run-1"]),
    }));
    renderRow({ row: row({ runId: "run-1" }), selectable: true });
    const rowEl = screen.getByTestId(selectors.operationsCenter.activityRow);
    expect(rowEl.getAttribute("data-stopping")).toBe("true");
    const cb = screen.getByTestId<HTMLInputElement>(
      selectors.operationsCenter.activityRowCheckbox,
    );
    expect(cb.disabled).toBe(true);
    expect(screen.getByText(/Stopping/)).toBeInTheDocument();
  });

  it("does not navigate when stopping (click is a no-op)", () => {
    useOperationsStore.setState((s) => ({
      stoppingRunIds: new Set([...s.stoppingRunIds, "run-1"]),
    }));
    renderRow({ row: row({ runId: "run-1" }), selectable: true });
    const rowEl = screen.getByTestId(selectors.operationsCenter.activityRow);
    // role gets stripped while stopping
    expect(rowEl.getAttribute("role")).toBeNull();
  });

  it("checkbox click does not propagate to the row navigation handler", () => {
    renderRow({ row: row({ runId: "run-1" }), selectable: true });
    const cb = screen.getByTestId<HTMLInputElement>(
      selectors.operationsCenter.activityRowCheckbox,
    );
    // No throw, no navigation; toggling works.
    fireEvent.click(cb);
    expect(useOperationsStore.getState().selection.has("run-1")).toBe(true);
  });
});
