/**
 * Tests for WorkspaceHeader — the unified top bar shared by the Plan board
 * and the Graph canvas.
 *
 * Pins: the Operations trigger pill (HUD variant → /plan), the sidebar toggle
 * only when collapsed, the lens nav, and the lens-aware help affordance.
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { WorkspaceHeader } from "./WorkspaceHeader";
import { selectors } from "../../../consts/selectors";

function renderHeader(overrides?: Partial<React.ComponentProps<typeof WorkspaceHeader>>) {
  return render(
    <MemoryRouter>
      <WorkspaceHeader
        lens="topology"
        sidebarCollapsed
        showNavControls={false}
        onToggleSidebar={vi.fn()}
        onToggleStats={vi.fn()}
        onToggleSettings={vi.fn()}
        onToggleHelp={vi.fn()}
        onLensChange={vi.fn()}
        {...overrides}
      />
    </MemoryRouter>,
  );
}

describe("WorkspaceHeader", () => {
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

  it("renders the Plan/Graph lens nav", () => {
    renderHeader();
    expect(screen.getByTestId("lens-nav")).toBeInTheDocument();
    expect(screen.getByTestId("lens-plan")).toBeInTheDocument();
    expect(screen.getByTestId("lens-graph")).toBeInTheDocument();
  });

  it("labels the help button for the active surface", () => {
    renderHeader({ lens: "plan" });
    expect(screen.getByRole("button", { name: "Plan guide" })).toBeInTheDocument();

    renderHeader({ lens: "topology" });
    expect(screen.getByRole("button", { name: "Graph guide" })).toBeInTheDocument();
  });

  it("shows the pan/zoom nav row only on the graph canvas, never on the plan board", () => {
    const { unmount } = renderHeader({ lens: "plan", showNavControls: true });
    expect(screen.queryByTestId(selectors.graphNavControls.container)).toBeNull();
    unmount();

    renderHeader({ lens: "topology", showNavControls: true });
    expect(screen.getByTestId(selectors.graphNavControls.container)).toBeInTheDocument();
  });
});
