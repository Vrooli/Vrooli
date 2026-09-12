import { renderWithProviders as render } from "../test-utils";
import { describe, it, expect, vi } from "vitest";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { AlertDialog } from "@vrooli/react-component-library/AlertDialog/2";

function renderConfirm(props: Partial<Parameters<typeof AlertDialog>[0]> = {}) {
  const onCancel = vi.fn();
  const onConfirm = vi.fn();
  const result = render(
    <AlertDialog
      open
      title="Delete thing?"
      description="This cannot be undone."
      cancelLabel="Keep it"
      confirmLabel="Delete"
      onCancel={onCancel}
      onConfirm={onConfirm}
      testIdPrefix="test-confirm"
      {...props}
    />,
  );
  return { onCancel, onConfirm, ...result };
}

describe("AlertDialog consumer contract", () => {
  it("returns null when closed", () => {
    renderConfirm({ open: false });
    expect(screen.queryByTestId("test-confirm-dialog")).toBeNull();
  });

  it("renders alertdialog semantics with labelled title and described body", () => {
    renderConfirm();
    const panel = screen.getByRole("alertdialog");
    expect(panel.getAttribute("aria-modal")).toBe("true");
    const labelledBy = panel.getAttribute("aria-labelledby");
    const describedBy = panel.getAttribute("aria-describedby");
    expect(document.getElementById(labelledBy as string)?.textContent).toBe("Delete thing?");
    expect(document.getElementById(describedBy as string)?.textContent).toBe("This cannot be undone.");
  });

  it("auto-focuses the cancel button on open", async () => {
    renderConfirm();
    // The overlay substrate directs initial focus on the next animation frame
    // so the surface is laid out before anything is focused into it.
    await waitFor(() => {
      expect(document.activeElement).toBe(screen.getByTestId("test-confirm-cancel"));
    });
  });

  it("cancels on Escape", () => {
    const { onCancel, onConfirm } = renderConfirm();
    fireEvent.keyDown(window, { key: "Escape" });
    expect(onCancel).toHaveBeenCalledTimes(1);
    expect(onConfirm).not.toHaveBeenCalled();
  });

  it("does not confirm or cancel from panel interaction", () => {
    const { onCancel } = renderConfirm();
    fireEvent.click(screen.getByRole("alertdialog"));
    expect(onCancel).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("test-confirm-dialog"));
    expect(onCancel).not.toHaveBeenCalled();
  });

  it("fires each button's handler", () => {
    const { onCancel, onConfirm } = renderConfirm();
    fireEvent.click(screen.getByTestId("test-confirm-cancel"));
    expect(onCancel).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByTestId("test-confirm-confirm"));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it("marks the confirm action destructive only when requested", () => {
    renderConfirm({ destructive: true });
    expect(screen.getByTestId("test-confirm-confirm").getAttribute("data-destructive")).toBe("true");
    document.body.innerHTML = "";
    renderConfirm();
    expect(screen.getByTestId("test-confirm-confirm").getAttribute("data-destructive")).toBe("false");
  });

  it("traps focus inside the card", () => {
    renderConfirm();
    const panel = screen.getByRole("alertdialog");
    const confirm = screen.getByTestId("test-confirm-confirm");
    confirm.focus();
    fireEvent.keyDown(panel, { key: "Tab" });
    expect(panel.contains(document.activeElement)).toBe(true);
    const cancel = screen.getByTestId("test-confirm-cancel");
    cancel.focus();
    fireEvent.keyDown(panel, { key: "Tab", shiftKey: true });
    expect(panel.contains(document.activeElement)).toBe(true);
  });

  it("renders on the alert z tier, above the drawer and menu tiers", () => {
    renderConfirm();
    const sheet = Array.from(document.head.querySelectorAll("style[data-rcl-sheet]"))
      .map((node) => node.textContent ?? "")
      .find((css) => css.includes("[data-rcl-alert-dialog-layer]"));
    expect(sheet).toBeTruthy();
    // A confirmation is the topmost interactive surface: it is raised from
    // inside drawers and menus, so sharing --layer-modal with them left it
    // winning only by DOM order, and losing outright to --layer-menu.
    expect(sheet).toContain("z-index: var(--layer-alert, 700)");
    expect(sheet).not.toContain("z-index: var(--layer-modal");
  });
});
