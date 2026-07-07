/**
 * Tests for WorkspaceHeader — the unified top bar shared by the Plan board
 * and the Graph canvas.
 *
 * Pins: the Operations trigger pill (HUD variant → /plan), the sidebar toggle
 * only when collapsed, the lens nav, and the lens-aware help affordance.
 */

import { beforeEach, describe, it, expect, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { WorkspaceHeader } from "./WorkspaceHeader";
import { selectors } from "../../../consts/selectors";
import { useOperationsStore } from "../../../stores/operations-store";
import { createPlanDataInitialState, usePlanDataStore } from "../../plan/stores/plan-data-store";

function renderHeader(overrides?: Partial<React.ComponentProps<typeof WorkspaceHeader>>) {
  return render(
    <MemoryRouter>
      <WorkspaceHeader
        lens="topology"
        sidebarCollapsed
        showNavControls={false}
        onToggleSidebar={vi.fn()}
        onToggleSettings={vi.fn()}
        onToggleHelp={vi.fn()}
        onLensChange={vi.fn()}
        {...overrides}
      />
    </MemoryRouter>,
  );
}

describe("WorkspaceHeader", () => {
  beforeEach(() => {
    useOperationsStore.getState().reset();
    usePlanDataStore.setState({ ...createPlanDataInitialState() });
  });

  it("renders the Operations Center trigger pill (compact variant) linking to /plan", () => {
    renderHeader();

    const trigger = screen.getByTestId(selectors.layout.opsTriggerButton);
    expect(trigger).toBeInTheDocument();
    expect(trigger.getAttribute("data-variant")).toBe("compact");
    expect(trigger.getAttribute("href")).toBe("/plan");
  });

  it("does not render the legacy agents dropdown", () => {
    renderHeader();

    expect(screen.queryByTestId("graph-agents-toggle")).toBeNull();
    expect(screen.queryByTestId("graph-agents-dropdown")).toBeNull();
  });

  it("hides the trigger on desktop when the sidebar is open (mobile-only fallback)", () => {
    renderHeader({ sidebarCollapsed: false });

    const trigger = screen.getByTestId(selectors.layout.opsTriggerButton);
    expect(trigger.className).toContain("md:hidden");
  });

  it("always shows the trigger on every breakpoint when the sidebar is collapsed", () => {
    renderHeader({ sidebarCollapsed: true });

    const trigger = screen.getByTestId(selectors.layout.opsTriggerButton);
    expect(trigger.className).not.toContain("md:hidden");
  });

  it("renders the sidebar-toggle button only when the sidebar is collapsed", () => {
    renderHeader({ sidebarCollapsed: false });
    expect(screen.queryByTestId("sidebar-toggle-open")).toBeNull();

    renderHeader({ sidebarCollapsed: true });
    expect(screen.getByTestId("sidebar-toggle-open")).toBeInTheDocument();
  });

  it("renders the Plan/Graph/Stats lens nav", () => {
    renderHeader();
    expect(screen.getByTestId("lens-nav")).toBeInTheDocument();
    expect(screen.getByTestId("lens-plan")).toBeInTheDocument();
    expect(screen.getByTestId("lens-graph")).toBeInTheDocument();
    expect(screen.getByTestId("lens-stats")).toBeInTheDocument();
    expect(screen.queryByTestId("stats-button")).toBeNull();
  });

  it("badges the Plan lens with the active agent count", () => {
    useOperationsStore.setState({
      view: {
        lanes: [],
        queue: { depth: 0, maxDepth: 50 },
        activities: [
          { runId: "run-1" },
          { runId: "run-2" },
        ],
        recentlyFinished: [],
        generatedAt: "2026-07-07T00:00:00Z",
        windowSeconds: 10800,
      } as never,
    });

    renderHeader({ lens: "stats" });

    expect(screen.getByTestId("lens-plan-badge")).toHaveTextContent("2");
  });

  it("labels the help button for the active surface", () => {
    renderHeader({ lens: "plan" });
    expect(screen.getByRole("button", { name: "Plan guide" })).toBeInTheDocument();

    renderHeader({ lens: "topology" });
    expect(screen.getByRole("button", { name: "Graph guide" })).toBeInTheDocument();

    renderHeader({ lens: "stats" });
    expect(screen.getByRole("button", { name: "Stats guide" })).toBeInTheDocument();
  });

  it("shows graph-only controls only on the graph canvas", () => {
    const { unmount } = renderHeader({ lens: "plan", showNavControls: true });
    expect(screen.queryByTestId(selectors.graphNavControls.container)).toBeNull();
    expect(screen.queryByTestId("settings-gear")).toBeNull();
    unmount();

    const graphRender = renderHeader({ lens: "topology", showNavControls: true });
    expect(screen.getByTestId(selectors.graphNavControls.container)).toBeInTheDocument();
    expect(screen.getByTestId("settings-gear")).toBeInTheDocument();
    graphRender.unmount();

    renderHeader({ lens: "stats", showNavControls: true });
    expect(screen.queryByTestId(selectors.graphNavControls.container)).toBeNull();
    expect(screen.queryByTestId("settings-gear")).toBeNull();
  });

  it("renders plan filters and refresh only on the Plan lens", () => {
    const refreshPlan = vi.fn().mockResolvedValue(undefined);
    usePlanDataStore.setState({ fetchBoard: refreshPlan });

    const { unmount } = renderHeader({ lens: "plan" });
    expect(screen.getByTestId(selectors.plan.boardFilters)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.plan.boardRefresh)).toBeInTheDocument();
    fireEvent.click(screen.getByTestId(selectors.plan.boardFilters));
    expect(usePlanDataStore.getState().filterDrawerOpen).toBe(true);
    fireEvent.click(screen.getByTestId(selectors.plan.boardRefresh));
    expect(refreshPlan).toHaveBeenCalledWith({ force: true });
    unmount();

    renderHeader({ lens: "stats" });
    expect(screen.queryByTestId(selectors.plan.boardFilters)).toBeNull();
    expect(screen.queryByTestId(selectors.plan.boardRefresh)).toBeNull();
  });

  it("marks the plan filter button active when filters are applied", () => {
    useOperationsStore.getState().setFilters({ q: "blocked" });
    renderHeader({ lens: "plan" });

    expect(screen.getByTestId(selectors.plan.boardFilters).className).toContain("text-cyan-400");
  });
});
