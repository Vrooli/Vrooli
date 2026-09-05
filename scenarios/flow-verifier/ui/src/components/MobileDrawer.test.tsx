import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../test-utils";
import { MobileDrawer } from "./MobileDrawer";

describe("MobileDrawer", () => {
  afterEach(() => cleanup());

  it("renders nothing when closed", () => {
    renderWithProviders(
      <MobileDrawer open={false} onClose={() => {}}>
        <div data-testid="drawer-child">child</div>
      </MobileDrawer>,
    );
    expect(screen.queryByTestId("mobile-drawer-root")).not.toBeInTheDocument();
  });

  it("renders children and dialog semantics when open", () => {
    renderWithProviders(
      <MobileDrawer open onClose={() => {}}>
        <div data-testid="drawer-child">child</div>
      </MobileDrawer>,
    );
    const root = screen.getByTestId("mobile-drawer-root");
    expect(root).toHaveAttribute("role", "dialog");
    expect(root).toHaveAttribute("aria-modal", "true");
    expect(screen.getByTestId("drawer-child")).toBeInTheDocument();
  });

  it("calls onClose when the backdrop is clicked", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <MobileDrawer open onClose={onClose}>
        <div />
      </MobileDrawer>,
    );
    await user.click(screen.getByTestId("mobile-drawer-backdrop"));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("calls onClose when the close button is clicked", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderWithProviders(
      <MobileDrawer open onClose={onClose}>
        <div />
      </MobileDrawer>,
    );
    await user.click(screen.getByTestId("mobile-drawer-close"));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it("calls onClose when Escape is pressed", () => {
    const onClose = vi.fn();
    renderWithProviders(
      <MobileDrawer open onClose={onClose}>
        <div />
      </MobileDrawer>,
    );
    fireEvent.keyDown(window, { key: "Escape" });
    expect(onClose).toHaveBeenCalledOnce();
  });
});
