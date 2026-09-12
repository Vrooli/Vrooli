import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { Dialog } from "./dialog";

describe("Dialog", () => {
  it("does not render while closed", () => {
    renderWithProviders(<Dialog open={false} title="Title" closeLabel="Close" onClose={vi.fn()}>Body</Dialog>);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("renders optional content and closes from every dismiss surface", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    renderWithProviders(
      <Dialog open title="Title" description="Description" footer={<span>Footer</span>} closeLabel="Close" onClose={onClose}>
        Body
      </Dialog>,
    );
    expect(screen.getByText("Description")).toBeInTheDocument();
    expect(screen.getByText("Footer")).toBeInTheDocument();
    await user.click(screen.getAllByRole("button", { name: "Close" })[1]!);
    fireEvent.keyDown(window, { key: "Escape" });
    expect(onClose).toHaveBeenCalledTimes(2);
  });
});
