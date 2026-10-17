import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { Dialog } from "./dialog";

describe("Dialog", () => {
  it("renders nothing while closed", () => {
    renderWithProviders(
      <Dialog open={false} title="Details" closeLabel="Close" onClose={vi.fn()}>
        <p>Hidden content</p>
      </Dialog>,
      { withoutRouter: true },
    );

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("closes from Escape, backdrop, and the close control", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderWithProviders(
      <Dialog
        open
        title="Details"
        description="A description"
        closeLabel="Close details"
        footer={<button type="button">Save</button>}
        onClose={onClose}
      >
        <p>Visible content</p>
      </Dialog>,
      { withoutRouter: true },
    );

    expect(screen.getByRole("dialog", { name: "Details" })).toHaveTextContent("Visible content");
    const closeButtons = screen.getAllByRole("button", { name: "Close details" });
    await user.click(closeButtons[0]!);
    fireEvent.keyDown(window, { key: "Escape" });
    fireEvent.click(closeButtons[1]!);
    expect(onClose).toHaveBeenCalledTimes(3);
  });
});
