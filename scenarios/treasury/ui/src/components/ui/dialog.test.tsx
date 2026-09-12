import { afterEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { Dialog } from "./dialog";

describe("Dialog", () => {
  afterEach(cleanup);

  it("renders nothing while closed", () => {
    renderWithProviders(
      <Dialog open={false} title="Approve charge" closeLabel="Close" onClose={vi.fn()}>
        Charge details
      </Dialog>,
    );

    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("exposes its description and footer and closes from either close control", () => {
    const onClose = vi.fn();
    renderWithProviders(
      <Dialog
        open
        title="Approve $12.00 to Example API"
        description="This action permits one settlement."
        footer={<button type="button">Approve</button>}
        closeLabel="Close approval dialog"
        onClose={onClose}
      >
        Charge details
      </Dialog>,
    );

    const dialog = screen.getByRole("dialog", { name: "Approve $12.00 to Example API" });
    expect(dialog).toHaveAccessibleDescription("This action permits one settlement.");
    expect(screen.getByRole("button", { name: "Approve" })).toBeInTheDocument();

    const closeControls = screen.getAllByRole("button", { name: "Close approval dialog" });
    expect(closeControls).toHaveLength(2);
    for (const control of closeControls) {
      fireEvent.click(control);
    }
    expect(onClose).toHaveBeenCalledTimes(2);
  });

  it("closes on Escape but ignores unrelated keys", () => {
    const onClose = vi.fn();
    renderWithProviders(
      <Dialog open title="Approval" closeLabel="Close" onClose={onClose}>
        Charge details
      </Dialog>,
    );

    fireEvent.keyDown(window, { key: "Enter" });
    expect(onClose).not.toHaveBeenCalled();
    fireEvent.keyDown(window, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
