import { fireEvent, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Dialog } from "./dialog";
import { renderWithProviders } from "../../test-utils";

describe("Dialog", () => {
  afterEach(() => {
    document.body.innerHTML = "";
  });

  it("does not render closed dialogs", () => {
    renderWithProviders(<Dialog open={false} title="Title" closeLabel="Close" onClose={vi.fn()}>Body</Dialog>);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("closes from escape and both close controls while rendering optional content", () => {
    const onClose = vi.fn();
    renderWithProviders(
      <Dialog
        open
        title="Title"
        description="Description"
        closeLabel="Close"
        onClose={onClose}
        footer={<span>Footer</span>}
      >
        Body
      </Dialog>,
    );
    expect(screen.getByRole("dialog")).toHaveTextContent("Body");
    fireEvent.keyDown(window, { key: "Escape" });
    const closeButtons = screen.getAllByRole("button", { name: "Close" });
    for (const button of closeButtons) {
      fireEvent.click(button);
    }
    expect(onClose).toHaveBeenCalledTimes(3);
  });
});
