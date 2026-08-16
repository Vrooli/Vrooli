import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { SidebarShell } from "./sidebar-shell";

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
    expect(screen.getByText(/Navigation/)).toBeInTheDocument();
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

  it("supports a closed responsive shell with an optional resize handle", () => {
    renderWithProviders(
      <SidebarShell
        mode="responsive"
        mobileOpen={false}
        onMobileClose={vi.fn()}
        mobileLabel="Menu"
        closeLabel="Close"
        width={280}
        resizeHandleProps={{ "aria-label": "Resize navigation" }}
      >
        <span>Navigation</span>
      </SidebarShell>,
    );

    expect(screen.getByTestId("sidebar-shell")).toHaveAttribute("data-mode", "responsive");
    expect(screen.getByTestId("sidebar-shell")).toHaveAttribute("role", "complementary");
    expect(screen.getByTestId("sidebar-shell-resize-handle")).toHaveAttribute("aria-label", "Resize navigation");
    expect(screen.queryByTestId("sidebar-shell-backdrop")).not.toBeInTheDocument();
  });
});
