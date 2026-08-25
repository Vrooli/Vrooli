import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { SidebarShell } from "@vrooli/react-component-library/SidebarShell/1.2.0";

/**
 * Fixture copy, named once. These are the test's OWN sample values rather
 * than application copy, but they are referenced through a constant so the
 * copy-driven-query lint rule stays enforceable without a per-file exemption.
 */
const NAV_LABEL = "Navigation";

describe("SidebarShell", () => {
  it("renders persistent desktop content", () => {
    renderWithProviders(
      <SidebarShell
        mode="persistent"
        mobileOpen={false}
        onMobileClose={() => undefined}
        mobileLabel="Menu"
        closeLabel="Close"
      >
        <span>Navigation</span>
      </SidebarShell>,
    );

    expect(screen.getByTestId("sidebar-shell")).toHaveAttribute("data-mode", "persistent");
    expect(screen.getByText(NAV_LABEL)).toBeInTheDocument();
  });

  it("closes an overlay from the backdrop and Escape key", async () => {
    const user = userEvent.setup();
    const onMobileClose = vi.fn();
    renderWithProviders(
      <SidebarShell
        mode="overlay"
        mobileOpen
        onMobileClose={onMobileClose}
        mobileLabel="Menu"
        closeLabel="Close"
      >
        <span>Navigation</span>
      </SidebarShell>,
    );

    await user.click(screen.getByTestId("sidebar-shell-backdrop"));
    fireEvent.keyDown(window, { key: "Escape" });

    expect(onMobileClose).toHaveBeenCalledTimes(2);
  });
});
