import { describe, it, expect, vi } from "vitest";
import { render, fireEvent, screen } from "@testing-library/react";
import { ConfirmDialog } from "../components/ConfirmDialog";

function renderConfirm(props: Partial<Parameters<typeof ConfirmDialog>[0]> = {}) {
  const onCancel = vi.fn();
  const onConfirm = vi.fn();
  const result = render(
    <ConfirmDialog
      open
      title="Delete thing?"
      body="This cannot be undone."
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

describe("ConfirmDialog", () => {
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

  it("auto-focuses the cancel button on open", () => {
    renderConfirm();
    expect(document.activeElement).toBe(screen.getByTestId("test-confirm-cancel"));
  });

  it("cancels on Escape", () => {
    const { onCancel, onConfirm } = renderConfirm();
    fireEvent.keyDown(window, { key: "Escape" });
    expect(onCancel).toHaveBeenCalledTimes(1);
    expect(onConfirm).not.toHaveBeenCalled();
  });

  it("cancels on backdrop click but not panel click", () => {
    const { onCancel } = renderConfirm();
    fireEvent.click(screen.getByRole("alertdialog"));
    expect(onCancel).not.toHaveBeenCalled();
    fireEvent.click(screen.getByTestId("test-confirm-dialog"));
    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it("fires each button's handler", () => {
    const { onCancel, onConfirm } = renderConfirm();
    fireEvent.click(screen.getByTestId("test-confirm-cancel"));
    expect(onCancel).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByTestId("test-confirm-confirm"));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it("styles the confirm button red only when destructive", () => {
    renderConfirm({ destructive: true });
    expect(screen.getByTestId("test-confirm-confirm").className).toContain("bg-red-600");
    document.body.innerHTML = "";
    renderConfirm();
    expect(screen.getByTestId("test-confirm-confirm").className).not.toContain("bg-red-600");
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

  it("renders on the confirm z tier (above the drawer tier)", () => {
    renderConfirm();
    expect(screen.getByTestId("test-confirm-dialog").className).toContain("z-wc-confirm");
  });
});
