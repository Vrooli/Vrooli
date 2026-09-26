import { cleanup, fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Dialog } from "./dialog";
import { renderWithProviders } from "../../test-utils";

const retirementDetails = "Retirement details";

describe("Dialog", () => {
  afterEach(cleanup);

  it("stays absent while closed", () => {
    renderWithProviders(<Dialog open={false} title="Retire token" closeLabel="Close" onClose={vi.fn()}>Body</Dialog>);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("renders its complete accessible surface and closes from both controls", () => {
    const onClose = vi.fn();
    renderWithProviders(
      <Dialog
        open
        title="Retire token"
        description="The declaration remains readable."
        closeLabel="Close retirement dialog"
        footer={<button type="button">Confirm retirement</button>}
        onClose={onClose}
        className="custom-dialog"
      >
        <p>{retirementDetails}</p>
      </Dialog>,
    );

    const dialog = screen.getByRole("dialog");
    expect(dialog).toHaveAccessibleName("Retire token");
    expect(dialog).toHaveAccessibleDescription("The declaration remains readable.");
    expect(dialog).toHaveClass("custom-dialog");
    expect(screen.getByText(retirementDetails)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Confirm retirement" })).toBeInTheDocument();

    fireEvent.keyDown(window, { key: "Enter" });
    expect(onClose).not.toHaveBeenCalled();
    fireEvent.keyDown(window, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(1);

    const closeButtons = screen.getAllByRole("button", { name: "Close retirement dialog" });
    expect(closeButtons).toHaveLength(2);
    fireEvent.click(closeButtons[0]!);
    fireEvent.click(closeButtons[1]!);
    expect(onClose).toHaveBeenCalledTimes(3);
  });

  it("does not declare a description or footer when neither is present", () => {
    renderWithProviders(<Dialog open title="Token type" closeLabel="Close" onClose={vi.fn()}>Body</Dialog>);
    expect(screen.getByRole("dialog")).not.toHaveAttribute("aria-describedby");
    expect(document.querySelector("footer")).not.toBeInTheDocument();
  });
});
