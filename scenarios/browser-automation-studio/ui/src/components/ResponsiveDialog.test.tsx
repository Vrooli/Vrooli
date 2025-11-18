import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import ResponsiveDialog from "@components/ResponsiveDialog";
import { selectors } from "../consts/selectors";

describe("ResponsiveDialog [REQ:BAS-PROJECT-DIALOG-OPEN] [REQ:BAS-PROJECT-DIALOG-CLOSE]", () => {
  it("renders when isOpen is true", () => {
    render(
      <ResponsiveDialog isOpen ariaLabel="Test Dialog">
        <div>Dialog Content</div>
      </ResponsiveDialog>,
    );

    expect(screen.getByRole("dialog")).toBeInTheDocument();
    expect(screen.getByText("Dialog Content")).toBeInTheDocument();
  });

  it("does not render when isOpen is false", () => {
    render(
      <ResponsiveDialog isOpen={false} ariaLabel="Test Dialog">
        <div>Dialog Content</div>
      </ResponsiveDialog>,
    );

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("calls onDismiss when overlay is clicked [REQ:BAS-PROJECT-DIALOG-CLOSE]", async () => {
    const user = userEvent.setup();
    const handleDismiss = vi.fn();

    render(
      <ResponsiveDialog
        isOpen
        onDismiss={handleDismiss}
        ariaLabel="Test Dialog"
      >
        <div>Dialog Content</div>
      </ResponsiveDialog>,
    );

    const overlay = screen.getByRole("presentation");
    await user.pointer({ target: overlay, keys: "[MouseLeft>]" });
    await user.pointer({ target: overlay, keys: "[/MouseLeft]" });

    expect(handleDismiss).toHaveBeenCalledTimes(1);
  });

  it("does not call onDismiss when clicking inside dialog content", async () => {
    const user = userEvent.setup();
    const handleDismiss = vi.fn();

    render(
      <ResponsiveDialog
        isOpen
        onDismiss={handleDismiss}
        ariaLabel="Test Dialog"
      >
        <div data-testid={selectors.dialogs.responsive.content}>Dialog Content</div>
      </ResponsiveDialog>,
    );

    const content = screen.getByTestId(selectors.dialogs.responsive.content);
    await user.click(content);

    expect(handleDismiss).not.toHaveBeenCalled();
  });

  it("renders with correct ARIA attributes", () => {
    const testId = "dialog-title";
    render(
      <ResponsiveDialog isOpen ariaLabelledBy={testId}>
        <h1 id={testId}>Dialog Title</h1>
      </ResponsiveDialog>,
    );

    const dialog = screen.getByRole("dialog");
    expect(dialog).toHaveAttribute("aria-modal", "true");
    expect(dialog).toHaveAttribute("aria-labelledby", testId);
  });

  it("applies size className correctly", () => {
    const { rerender } = render(
      <ResponsiveDialog isOpen ariaLabel="Test" size="wide">
        <div>Content</div>
      </ResponsiveDialog>,
    );

    let dialog = screen.getByRole("dialog");
    expect(dialog.className).toContain("responsive-dialog__content--wide");

    rerender(
      <ResponsiveDialog isOpen ariaLabel="Test" size="xl">
        <div>Content</div>
      </ResponsiveDialog>,
    );

    dialog = screen.getByRole("dialog");
    expect(dialog.className).toContain("responsive-dialog__content--xl");
  });

  it("renders as alertdialog when role prop is set", () => {
    render(
      <ResponsiveDialog isOpen ariaLabel="Alert" role="alertdialog">
        <div>Alert Content</div>
      </ResponsiveDialog>,
    );

    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });

  it("supports custom className on dialog content", () => {
    render(
      <ResponsiveDialog isOpen ariaLabel="Test" className="custom-dialog-class">
        <div>Content</div>
      </ResponsiveDialog>,
    );

    const dialog = screen.getByRole("dialog");
    expect(dialog.className).toContain("custom-dialog-class");
  });
});
