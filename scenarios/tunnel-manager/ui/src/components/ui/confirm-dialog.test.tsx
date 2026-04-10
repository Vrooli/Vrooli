import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ConfirmDialog } from "./confirm-dialog";

// [REQ:UI-A11Y] ConfirmDialog accessibility and interaction
describe("ConfirmDialog", () => {
  const defaults = {
    open: true,
    title: "Confirm Action",
    description: "Are you sure?",
    onConfirm: vi.fn(),
    onCancel: vi.fn(),
  };

  it("renders nothing when closed", () => {
    render(<ConfirmDialog {...defaults} open={false} />);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("renders dialog when open", () => {
    render(<ConfirmDialog {...defaults} />);
    expect(screen.getByRole("dialog")).toBeInTheDocument();
  });

  it("shows title and description", () => {
    render(<ConfirmDialog {...defaults} />);
    expect(screen.getByText("Confirm Action")).toBeInTheDocument();
    expect(screen.getByText("Are you sure?")).toBeInTheDocument();
  });

  it("shows default button labels", () => {
    render(<ConfirmDialog {...defaults} />);
    expect(screen.getByText("Confirm")).toBeInTheDocument();
    expect(screen.getByText("Cancel")).toBeInTheDocument();
  });

  it("shows custom button labels", () => {
    render(<ConfirmDialog {...defaults} confirmLabel="Delete" cancelLabel="Keep" />);
    expect(screen.getByText("Delete")).toBeInTheDocument();
    expect(screen.getByText("Keep")).toBeInTheDocument();
  });

  it("calls onConfirm when confirm clicked", () => {
    const onConfirm = vi.fn();
    render(<ConfirmDialog {...defaults} onConfirm={onConfirm} />);
    fireEvent.click(screen.getByTestId("confirm-dialog-confirm"));
    expect(onConfirm).toHaveBeenCalledTimes(1);
  });

  it("calls onCancel when cancel clicked", () => {
    const onCancel = vi.fn();
    render(<ConfirmDialog {...defaults} onCancel={onCancel} />);
    fireEvent.click(screen.getByTestId("confirm-dialog-cancel"));
    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it("calls onCancel on Escape key", () => {
    const onCancel = vi.fn();
    render(<ConfirmDialog {...defaults} onCancel={onCancel} />);
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it("calls onCancel when clicking backdrop", () => {
    const onCancel = vi.fn();
    render(<ConfirmDialog {...defaults} onCancel={onCancel} />);
    fireEvent.click(screen.getByTestId("confirm-dialog"));
    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it("does not call onCancel when clicking dialog content", () => {
    const onCancel = vi.fn();
    render(<ConfirmDialog {...defaults} onCancel={onCancel} />);
    fireEvent.click(screen.getByText("Are you sure?"));
    expect(onCancel).not.toHaveBeenCalled();
  });

  it("shows danger variant with warning icon", () => {
    render(<ConfirmDialog {...defaults} variant="danger" />);
    expect(screen.getByRole("dialog")).toBeInTheDocument();
    // danger variant adds red styling to confirm button
    const confirmBtn = screen.getByTestId("confirm-dialog-confirm");
    expect(confirmBtn.className).toContain("red");
  });

  it("shows pending state text derived from confirmLabel", () => {
    render(<ConfirmDialog {...defaults} isPending variant="danger" />);
    expect(screen.getByText("Confirm...")).toBeInTheDocument();
  });

  it("shows custom pending label when provided", () => {
    render(<ConfirmDialog {...defaults} isPending pendingLabel="Deleting..." variant="danger" />);
    expect(screen.getByText("Deleting...")).toBeInTheDocument();
  });

  it("disables buttons when pending", () => {
    render(<ConfirmDialog {...defaults} isPending />);
    expect(screen.getByTestId("confirm-dialog-confirm")).toBeDisabled();
    expect(screen.getByTestId("confirm-dialog-cancel")).toBeDisabled();
  });

  it("has aria-modal, aria-labelledby, and aria-describedby", () => {
    render(<ConfirmDialog {...defaults} />);
    const dialog = screen.getByRole("dialog");
    expect(dialog).toHaveAttribute("aria-modal", "true");
    expect(dialog).toHaveAttribute("aria-labelledby", "confirm-dialog-title");
    expect(dialog).toHaveAttribute("aria-describedby");
  });

  it("renders data-testid attributes", () => {
    render(<ConfirmDialog {...defaults} />);
    expect(screen.getByTestId("confirm-dialog")).toBeInTheDocument();
    expect(screen.getByTestId("confirm-dialog-confirm")).toBeInTheDocument();
    expect(screen.getByTestId("confirm-dialog-cancel")).toBeInTheDocument();
  });
});
