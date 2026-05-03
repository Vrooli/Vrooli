/**
 * Tests for SidebarHeader.
 *
 * Verifies home, settings, and collapse handlers fire, and pins that the
 * Operations Center trigger pill (P8) renders in place of the legacy
 * `<AgentsDropdown>` popover.
 */

import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { SidebarHeader } from "./SidebarHeader";
import { selectors } from "../../../../consts/selectors";

// Mock useCommandPostBadgeCount to avoid QueryClientProvider dependency.
vi.mock("../../../../hooks/useCommandPostBadgeCount", () => ({
  useCommandPostBadgeCount: () => 0,
}));

function renderHeader(overrides?: Partial<React.ComponentProps<typeof SidebarHeader>>) {
  return render(
    <MemoryRouter>
      <SidebarHeader
        onSettingsOpen={vi.fn()}
        onCollapse={vi.fn()}
        onGoHome={vi.fn()}
        {...overrides}
      />
    </MemoryRouter>,
  );
}

describe("SidebarHeader", () => {
  it("renders home button and app title", () => {
    renderHeader();

    expect(screen.getByTestId("sidebar-home")).toBeInTheDocument();
    expect(screen.getByText("Swarm Manager")).toBeInTheDocument();
  });

  it("calls onGoHome when home button is clicked", async () => {
    const onGoHome = vi.fn();
    renderHeader({ onGoHome });

    const user = userEvent.setup();
    await user.click(screen.getByTestId("sidebar-home"));

    expect(onGoHome).toHaveBeenCalledOnce();
  });

  it("calls onSettingsOpen when settings button is clicked", async () => {
    const onSettingsOpen = vi.fn();
    renderHeader({ onSettingsOpen });

    const user = userEvent.setup();
    await user.click(screen.getByTestId("sidebar-settings"));

    expect(onSettingsOpen).toHaveBeenCalledOnce();
  });

  it("calls onCollapse when collapse button is clicked", async () => {
    const onCollapse = vi.fn();
    renderHeader({ onCollapse });

    const user = userEvent.setup();
    await user.click(screen.getByTestId("sidebar-toggle-close"));

    expect(onCollapse).toHaveBeenCalledOnce();
  });

  it("renders the Operations Center trigger pill (compact variant)", () => {
    renderHeader();

    const trigger = screen.getByTestId(selectors.layout.opsTriggerButton);
    expect(trigger).toBeInTheDocument();
    expect(trigger.getAttribute("data-variant")).toBe("compact");
    expect(trigger.getAttribute("href")).toBe("/operations");
  });

  it("does not render the legacy agents dropdown", () => {
    renderHeader();

    expect(screen.queryByTestId("agents-badge")).toBeNull();
    expect(screen.queryByTestId("agents-dropdown")).toBeNull();
  });
});
