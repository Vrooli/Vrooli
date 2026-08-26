import { renderWithProviders as render } from "../../../../../ui/src/test-utils";
import { describe, it, expect, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";
import { DrawerShell } from "./DrawerShell.tsx";

function renderDrawer(props: Partial<Parameters<typeof DrawerShell>[0]> = {}) {
  const onClose = vi.fn();
  const result = render(
    <DrawerShell
      open
      onClose={onClose}
      closeAriaLabel="Close drawer"
      title="Test Drawer"
      panelTestId="test-drawer-panel"
      {...props}
    >
      <div>
        <button data-testid="inner-first">First</button>
        <button data-testid="inner-last">Last</button>
      </div>
    </DrawerShell>,
  );
  return { onClose, ...result };
}

describe("DrawerShell", () => {
  it("returns null when closed", () => {
    renderDrawer({ open: false });
    expect(screen.queryByTestId("test-drawer-panel")).toBeNull();
  });

  it("renders dialog semantics: role=dialog, aria-modal, labelled title", () => {
    renderDrawer();
    const panel = screen.getByTestId("test-drawer-panel");
    expect(panel.getAttribute("role")).toBe("dialog");
    expect(panel.getAttribute("aria-modal")).toBe("true");
    const labelledBy = panel.getAttribute("aria-labelledby");
    expect(labelledBy).toBeTruthy();
    const heading = document.getElementById(labelledBy as string);
    expect(heading).not.toBeNull();
    expect(heading?.textContent).toBe("Test Drawer");
  });

  it("closes on Escape", () => {
    const { onClose } = renderDrawer();
    fireEvent.keyDown(window, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("closes on backdrop click but not panel click", () => {
    const { onClose } = renderDrawer();
    const panel = screen.getByTestId("test-drawer-panel");
    const backdrop = panel.parentElement?.firstElementChild as HTMLElement;
    fireEvent.click(panel);
    expect(onClose).not.toHaveBeenCalled();
    fireEvent.click(backdrop);
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("closes via the close button", () => {
    const { onClose } = renderDrawer();
    fireEvent.click(screen.getByLabelText("Close drawer"));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("traps focus: Tab from the last control wraps to the first", () => {
    renderDrawer();
    const panel = screen.getByTestId("test-drawer-panel");
    const last = screen.getByTestId("inner-last");
    last.focus();
    // jsdom reports offsetParent as null for everything, so the trap's
    // visibility filter keeps only the active element; Tab from the last
    // (only) candidate must wrap rather than leak behind the overlay.
    fireEvent.keyDown(panel, { key: "Tab" });
    expect(panel.contains(document.activeElement)).toBe(true);
  });

  it("keeps Shift+Tab inside the panel", () => {
    renderDrawer();
    const panel = screen.getByTestId("test-drawer-panel");
    const first = screen.getByTestId("inner-first");
    first.focus();
    fireEvent.keyDown(panel, { key: "Tab", shiftKey: true });
    expect(panel.contains(document.activeElement)).toBe(true);
  });

  it("defaults to the full size variant (inset desktop card)", () => {
    renderDrawer();
    const panel = screen.getByTestId("test-drawer-panel");
    expect(panel.className).toContain("md:inset-x-8");
    expect(panel.className).toContain("md:top-8");
    expect(panel.className).not.toContain("md:max-w-md");
  });

  it("compact renders a centered auto-height desktop card, same mobile sheet", () => {
    renderDrawer({ size: "compact" });
    const panel = screen.getByTestId("test-drawer-panel");
    expect(panel.className).toContain("md:max-w-md");
    expect(panel.className).toContain("md:bottom-auto");
    expect(panel.className).not.toContain("md:inset-x-8");
    // Mobile bottom sheet identical to the full variant.
    expect(panel.className).toContain("top-[max(1rem,var(--wc-safe-top,0px))]");
    expect(panel.className).toContain("rounded-t-[20px]");
  });

  it("anchors to the keyboard height only when avoidKeyboard is set", () => {
    renderDrawer({ avoidKeyboard: true });
    let panel = screen.getByTestId("test-drawer-panel");
    expect(panel.className).toContain("bottom-[var(--wc-kb-height,0px)]");

    // Default stays pinned to the viewport bottom.
    document.body.innerHTML = "";
    renderDrawer();
    panel = screen.getByTestId("test-drawer-panel");
    expect(panel.className).toContain("bottom-0");
    expect(panel.className).not.toContain("bottom-[var(--wc-kb-height,0px)]");
  });

  it("renders headerActions and headerExtra in the header", () => {
    renderDrawer({
      headerActions: <button data-testid="hdr-action">A</button>,
      headerExtra: <div data-testid="hdr-extra">extra</div>,
    });
    expect(screen.getByTestId("hdr-action")).toBeTruthy();
    expect(screen.getByTestId("hdr-extra")).toBeTruthy();
  });
});
