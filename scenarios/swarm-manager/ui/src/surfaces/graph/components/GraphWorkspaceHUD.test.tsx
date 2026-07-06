/**
 * Tests for GraphWorkspaceHUD.
 *
 * Pins the contract that the agents button is the Operations Center
 * trigger pill (HUD variant) and links to /plan.
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { GraphWorkspaceHUD } from "./GraphWorkspaceHUD";
import { selectors } from "../../../consts/selectors";

function renderHUD(overrides?: Partial<React.ComponentProps<typeof GraphWorkspaceHUD>>) {
  return render(
    <MemoryRouter>
      <GraphWorkspaceHUD
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

describe("GraphWorkspaceHUD", () => {
  it("renders the Operations Center trigger pill (hud variant)", () => {
    renderHUD();

    const trigger = screen.getByTestId(selectors.layout.opsTriggerButton);
    expect(trigger).toBeInTheDocument();
    expect(trigger.getAttribute("data-variant")).toBe("hud");
    expect(trigger.getAttribute("href")).toBe("/plan");
  });

  it("does not render the legacy agents dropdown", () => {
    renderHUD();

    expect(screen.queryByTestId("graph-agents-toggle")).toBeNull();
    expect(screen.queryByTestId("graph-agents-dropdown")).toBeNull();
  });

  it("hides the trigger on desktop when the sidebar is open (mobile-only fallback)", () => {
    renderHUD({ sidebarCollapsed: false });

    const trigger = screen.getByTestId(selectors.layout.opsTriggerButton);
    // Sidebar already shows the compact trigger; the HUD variant collapses
    // to mobile-only via the responsive utility class.
    expect(trigger.className).toContain("md:hidden");
  });

  it("always shows the trigger on every breakpoint when the sidebar is collapsed", () => {
    renderHUD({ sidebarCollapsed: true });

    const trigger = screen.getByTestId(selectors.layout.opsTriggerButton);
    expect(trigger.className).not.toContain("md:hidden");
  });

  it("renders the sidebar-toggle button only when the sidebar is collapsed", () => {
    renderHUD({ sidebarCollapsed: true });
    expect(screen.getByTestId("sidebar-toggle-open")).toBeInTheDocument();

    renderHUD({ sidebarCollapsed: false });
    // The earlier instance is still mounted in the same DOM, so just check
    // there's at most one toggle button — the new instance hides it.
    const toggles = screen.queryAllByTestId("sidebar-toggle-open");
    expect(toggles).toHaveLength(1);
  });
});
