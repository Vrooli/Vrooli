import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../../test-utils";
import { Dialog } from "./dialog";
import { EmptyState } from "./empty-state";

describe("EmptyState", () => {
  it("renders optional description, icon, action, and custom class", () => {
    renderWithProviders(
      <EmptyState
        title="Nothing here"
        description="Try another filter"
        icon={<span data-testid="empty-icon">!</span>}
        action={<button type="button">Retry</button>}
        className="custom-empty"
      />,
    );
    expect(screen.getByText(/Nothing here/)).toBeInTheDocument();
    expect(screen.getByText(/Try another filter/)).toBeInTheDocument();
    expect(screen.getByTestId("empty-icon")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /Retry/ })).toBeInTheDocument();
    expect(screen.getByText(/Nothing here/).parentElement?.parentElement).toHaveClass("custom-empty");
  });
});

describe("Dialog", () => {
  it("does not render when closed", () => {
    renderWithProviders(<Dialog open={false} title="Hidden" onClose={vi.fn()} closeLabel="Close">Body</Dialog>);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("closes from Escape, backdrop, and close button while rendering optional regions", () => {
    const onClose = vi.fn();
    renderWithProviders(
      <Dialog
        open
        title="Confirm"
        description="A description"
        footer={<span>Footer</span>}
        onClose={onClose}
        closeLabel="Close dialog"
        className="custom-dialog"
      >
        Body
      </Dialog>,
    );
    expect(screen.getByRole("dialog")).toHaveClass("custom-dialog");
    expect(screen.getByText(/A description/)).toBeInTheDocument();
    expect(screen.getByText(/Footer/)).toBeInTheDocument();
    fireEvent.keyDown(window, { key: "Enter" });
    expect(onClose).not.toHaveBeenCalled();
    fireEvent.keyDown(window, { key: "Escape" });
    const closeButtons = screen.getAllByRole("button", { name: /Close dialog/ });
    expect(closeButtons).toHaveLength(2);
    const [closeButton, closeIconButton] = closeButtons;
    if (!closeButton || !closeIconButton) {
      throw new Error("Expected both dialog close buttons");
    }
    fireEvent.click(closeButton);
    fireEvent.click(closeIconButton);
    expect(onClose).toHaveBeenCalledTimes(3);
  });
});
