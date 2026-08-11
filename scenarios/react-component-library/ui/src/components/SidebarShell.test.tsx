import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen } from "@testing-library/react";
import { renderWithProviders } from "../test-utils";
import userEvent from "@testing-library/user-event";

import { SidebarShell } from "./SidebarShell";

describe("SidebarShell", () => {
  afterEach(() => cleanup());

  it("renders a persistent desktop sidebar and applies the supplied width", () => {
    renderWithProviders(
      <SidebarShell
        mobileOpen={false}
        onMobileClose={() => {}}
        mobileLabel="Navigation drawer"
        desktopLabel="Primary navigation"
        closeLabel="Close navigation"
        width={320}
      >
        <nav data-testid="shell-child" />
      </SidebarShell>,
    );

    const shell = screen.getByTestId("sidebar-shell");
    expect(shell).toBeInTheDocument();
    expect(shell).toHaveAttribute("aria-label", "Primary navigation");
    expect(shell).toHaveStyle({ width: "320px" });
    expect(shell).not.toHaveAttribute("role", "dialog");
    expect(screen.getByTestId("shell-child")).toBeInTheDocument();
    expect(screen.queryByTestId("sidebar-shell-backdrop")).not.toBeInTheDocument();
  });

  it("renders an open mobile dialog as a full-width safe-area sheet", () => {
    renderWithProviders(
      <SidebarShell
        mobileOpen
        onMobileClose={() => {}}
        mobileLabel="Navigation drawer"
        closeLabel="Close navigation"
        mobileHeader={<span data-testid="mobile-shell-title" />}
      >
        <nav data-testid="shell-child" />
      </SidebarShell>,
    );

    const shell = screen.getByTestId("sidebar-shell");
    expect(shell).toHaveAttribute("role", "dialog");
    expect(shell).toHaveAttribute("aria-modal", "true");
    expect(shell).toHaveAttribute("aria-label", "Navigation drawer");
    expect(shell).toHaveAttribute("data-mode", "responsive");
    expect(shell).toHaveAttribute("data-open", "true");
    expect(screen.getByTestId("mobile-shell-title")).toBeInTheDocument();
  });

  it("calls onMobileClose from backdrop, close button, and Escape", async () => {
    const onMobileClose = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <SidebarShell
        mobileOpen
        onMobileClose={onMobileClose}
        mobileLabel="Navigation drawer"
        closeLabel="Close navigation"
      >
        <nav />
      </SidebarShell>,
    );

    await user.click(screen.getByTestId("sidebar-shell-backdrop"));
    await user.click(screen.getByTestId("sidebar-shell-close"));
    fireEvent.keyDown(window, { key: "Escape" });

    expect(onMobileClose).toHaveBeenCalledTimes(3);
  });

  it("renders resize handle props on desktop", () => {
    renderWithProviders(
      <SidebarShell
        mobileOpen={false}
        onMobileClose={() => {}}
        mobileLabel="Navigation drawer"
        closeLabel="Close navigation"
        resizeHandleProps={{
          role: "separator",
          "aria-orientation": "vertical",
          tabIndex: 0,
        }}
      >
        <nav />
      </SidebarShell>,
    );

    const handle = screen.getByTestId("sidebar-shell-resize-handle");
    expect(handle).toHaveAttribute("role", "separator");
    expect(handle).toHaveAttribute("aria-orientation", "vertical");
  });

  it("can force overlay mode at any viewport width", () => {
    renderWithProviders(
      <SidebarShell
        mode="overlay"
        mobileOpen
        onMobileClose={() => {}}
        mobileLabel="Navigation drawer"
        desktopLabel="Primary navigation"
        closeLabel="Close navigation"
        mobileHeader={<span data-testid="mobile-shell-title" />}
      >
        <nav />
      </SidebarShell>,
    );

    const shell = screen.getByTestId("sidebar-shell");
    expect(shell).toHaveAttribute("data-mode", "overlay");
    expect(shell).toHaveAttribute("role", "dialog");
    expect(shell).toHaveAttribute("data-open", "true");
    expect(screen.getByTestId("sidebar-shell-backdrop")).toHaveAttribute("data-mode", "overlay");
    expect(screen.getByTestId("mobile-shell-title")).toBeInTheDocument();
  });

  it("can force persistent mode and suppress mobile drawer chrome", () => {
    const onMobileClose = vi.fn();
    renderWithProviders(
      <SidebarShell
        mode="persistent"
        mobileOpen
        onMobileClose={onMobileClose}
        mobileLabel="Navigation drawer"
        desktopLabel="Primary navigation"
        closeLabel="Close navigation"
        mobileHeader={<span data-testid="mobile-shell-title" />}
        width={360}
        resizeHandleProps={{ role: "separator" }}
      >
        <nav />
      </SidebarShell>,
    );

    const shell = screen.getByTestId("sidebar-shell");
    expect(shell).toHaveAttribute("data-mode", "persistent");
    expect(shell).toHaveAttribute("role", "complementary");
    expect(shell).toHaveAttribute("aria-label", "Primary navigation");
    expect(shell).not.toHaveAttribute("aria-modal");
    expect(shell).toHaveStyle({ width: "360px" });
    expect(screen.queryByTestId("sidebar-shell-backdrop")).not.toBeInTheDocument();
    expect(screen.queryByTestId("sidebar-shell-close")).not.toBeInTheDocument();
    expect(screen.getByTestId("sidebar-shell-resize-handle").className).not.toContain("md:block");

    fireEvent.keyDown(window, { key: "Escape" });
    expect(onMobileClose).not.toHaveBeenCalled();
  });

  it("declares the responsive desktop override that keeps a closed drawer visible", () => {
    renderWithProviders(
      <SidebarShell
        mobileOpen={false}
        onMobileClose={() => {}}
        mobileLabel="Navigation drawer"
        desktopLabel="Primary navigation"
        closeLabel="Close navigation"
      >
        <nav />
      </SidebarShell>,
    );

    const styles = document.querySelector("[data-rcl-sidebar-shell-styles]");
    expect(styles?.textContent).toContain(
      '[data-rcl-sidebar-shell][data-mode="responsive"][data-open="false"]',
    );
    expect(styles?.textContent).toContain("visibility: visible");
  });
});
