import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { ConfirmDialog, Modal } from "./Modal";

const LABELS = { closeLabel: "close-x", backdropCloseLabel: "backdrop-x" };

describe("Modal", () => {
  it("does not render when closed", () => {
    render(<Modal {...LABELS} open={false} onClose={() => {}} title="x" data-testid="m" />);
    expect(screen.queryByTestId("m")).toBeNull();
  });

  it("renders dialog with aria-modal and labelled title/description", () => {
    render(
      <Modal {...LABELS} open onClose={() => {}} title="t-x" description="d-x" data-testid="m">
        <span data-testid="body">body-x</span>
      </Modal>,
    );
    const dialog = screen.getByRole("dialog");
    expect(dialog).toHaveAttribute("aria-modal", "true");
    expect(dialog).toHaveAccessibleName("t-x");
    expect(dialog).toHaveAccessibleDescription("d-x");
    expect(screen.getByTestId("body")).toBeInTheDocument();
  });

  it("calls onClose on ESC", async () => {
    const onClose = vi.fn();
    render(<Modal {...LABELS} open onClose={onClose} title="t-x" />);
    await userEvent.keyboard("{Escape}");
    expect(onClose).toHaveBeenCalled();
  });

  it("calls onClose when close button is clicked", async () => {
    const onClose = vi.fn();
    render(<Modal {...LABELS} open onClose={onClose} title="t-x" />);
    await userEvent.click(screen.getByRole("button", { name: LABELS.closeLabel }));
    expect(onClose).toHaveBeenCalled();
  });
});

describe("ConfirmDialog", () => {
  it("invokes onConfirm/onCancel for the respective buttons", async () => {
    const onConfirm = vi.fn();
    const onCancel = vi.fn();
    const confirmLabel = "yes-x";
    const cancelLabel = "no-x";
    render(
      <ConfirmDialog
        {...LABELS}
        open
        onConfirm={onConfirm}
        onCancel={onCancel}
        title="t-x"
        confirmLabel={confirmLabel}
        cancelLabel={cancelLabel}
        destructive
      />,
    );
    await userEvent.click(screen.getByRole("button", { name: confirmLabel }));
    await userEvent.click(screen.getByRole("button", { name: cancelLabel }));
    expect(onConfirm).toHaveBeenCalled();
    expect(onCancel).toHaveBeenCalled();
  });
});
