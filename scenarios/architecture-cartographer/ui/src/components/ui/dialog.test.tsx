import { describe, expect, it, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";

import { selectors } from "../../consts/selectors";
import { Dialog } from "./dialog";

describe("Dialog", () => {
  it("renders nothing when closed", () => {
    render(
      <Dialog open={false} onClose={() => {}} ariaLabel="x">
        content
      </Dialog>,
    );
    expect(screen.queryByRole("dialog")).toBeNull();
  });

  it("renders content when open with the supplied aria-label", () => {
    render(
      <Dialog open onClose={() => {}} ariaLabel="confirm-action">
        <p>body</p>
      </Dialog>,
    );
    const dlg = screen.getByRole("dialog");
    expect(dlg).toHaveAttribute("aria-label", "confirm-action");
    expect(dlg).toHaveAttribute("aria-modal", "true");
    expect(screen.getByTestId(selectors.ui.dialog.panel)).toHaveTextContent("body");
  });

  it("uses aria-labelledby when supplied (no aria-label)", () => {
    render(
      <Dialog open onClose={() => {}} ariaLabel="ignored" ariaLabelledBy="t">
        <h1 id="t">heading</h1>
      </Dialog>,
    );
    const dlg = screen.getByRole("dialog");
    expect(dlg).toHaveAttribute("aria-labelledby", "t");
    expect(dlg).not.toHaveAttribute("aria-label");
  });

  it("invokes onClose on Escape", () => {
    const onClose = vi.fn();
    render(
      <Dialog open onClose={onClose} ariaLabel="x">
        body
      </Dialog>,
    );
    fireEvent.keyDown(document, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("invokes onClose when the backdrop is clicked", () => {
    const onClose = vi.fn();
    render(
      <Dialog open onClose={onClose} ariaLabel="x">
        <p>body</p>
      </Dialog>,
    );
    fireEvent.click(screen.getByTestId(selectors.ui.dialog.backdrop));
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
