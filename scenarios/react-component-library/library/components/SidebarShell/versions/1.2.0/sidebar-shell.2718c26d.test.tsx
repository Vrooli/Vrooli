/* eslint-disable no-restricted-syntax -- synthetic shell labels are component-contract fixtures. */
import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../../../../ui/src/test-utils";
import { SidebarShell } from "./SidebarShell.tsx";

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
    expect(screen.getByText("Navigation")).toBeInTheDocument();
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
