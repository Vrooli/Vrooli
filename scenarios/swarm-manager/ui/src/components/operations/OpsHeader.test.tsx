import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { OpsHeader } from "./OpsHeader";
import type { OperationsView } from "../../types/operations";

function makeView(overrides: Partial<OperationsView> = {}): OperationsView {
  return {
    lanes: [
      { lane: "investigate", active: 4, capacity: 6, queue: 0 },
      { lane: "execute", active: 1, capacity: 3, queue: 2 },
      { lane: "review", active: 0, capacity: 8, queue: 0 },
      { lane: "reconcile", active: 0, capacity: 2, queue: 0 },
    ],
    queue: { depth: 2, maxDepth: 50 },
    activities: [],
    recentlyFinished: [
      {
        activityId: "a-1",
        ownerType: "backlog",
        ownerName: "fix-foo",
        purpose: "process",
        status: "complete",
        requestedAt: "2026-05-02T00:30:00Z",
      },
      {
        activityId: "a-2",
        ownerType: "backlog",
        ownerName: "fix-bar",
        purpose: "process",
        status: "failed",
        requestedAt: "2026-05-02T00:31:00Z",
      },
    ],
    generatedAt: "2026-05-02T01:00:00Z",
    windowSeconds: 10800,
    ...overrides,
  };
}

describe("OpsHeader", () => {
  it("renders four lane bars in canonical order", () => {
    render(
      <OpsHeader
        view={makeView()}
        isRefreshing={false}
        onRefresh={() => {}}
        windowSeconds={10800}
      />,
    );
    const bars = screen.getAllByRole("progressbar");
    expect(bars).toHaveLength(4);
    expect(bars[0]).toHaveAccessibleName(/Investigate/);
    expect(bars[1]).toHaveAccessibleName(/Execute/);
    expect(bars[2]).toHaveAccessibleName(/Review/);
    expect(bars[3]).toHaveAccessibleName(/Reconcile/);
  });

  it("renders queue chip with depth", () => {
    render(
      <OpsHeader
        view={makeView()}
        isRefreshing={false}
        onRefresh={() => {}}
        windowSeconds={10800}
      />,
    );
    expect(screen.getByLabelText("2 queued")).toBeInTheDocument();
  });

  it("counts complete and failed totals from recently finished", () => {
    render(
      <OpsHeader
        view={makeView()}
        isRefreshing={false}
        onRefresh={() => {}}
        windowSeconds={10800}
      />,
    );
    expect(screen.getByText("1 ✓")).toBeInTheDocument();
    expect(screen.getByText("1 ✗")).toBeInTheDocument();
  });

  it("calls onRefresh when the refresh button is clicked", async () => {
    const handler = vi.fn();
    render(
      <OpsHeader
        view={makeView()}
        isRefreshing={false}
        onRefresh={handler}
        windowSeconds={10800}
      />,
    );
    await userEvent.click(screen.getByRole("button", { name: /refresh/i }));
    expect(handler).toHaveBeenCalled();
  });

  it("disables the refresh button while a refresh is in flight", () => {
    render(
      <OpsHeader
        view={makeView()}
        isRefreshing={true}
        onRefresh={() => {}}
        windowSeconds={10800}
      />,
    );
    expect(screen.getByRole("button", { name: /refresh/i })).toBeDisabled();
  });

  it("renders empty bars when view is null", () => {
    render(
      <OpsHeader
        view={null}
        isRefreshing={false}
        onRefresh={() => {}}
        windowSeconds={10800}
      />,
    );
    // No progressbars when no view yet.
    expect(screen.queryAllByRole("progressbar")).toHaveLength(0);
  });

  it("formats the window label as hours when ≥ 1h", () => {
    render(
      <OpsHeader
        view={makeView()}
        isRefreshing={false}
        onRefresh={() => {}}
        windowSeconds={3 * 3600}
      />,
    );
    expect(screen.getByText(/last 3h/i)).toBeInTheDocument();
  });
});
