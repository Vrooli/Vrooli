import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { renderWithProviders } from "../../test-utils";
import { Dialog } from "./dialog";

describe("Dialog", () => {
  it("is absent while closed", () => {
    renderWithProviders(<Dialog open={false} title="Details" closeLabel="Close" onClose={vi.fn()}>Body</Dialog>);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("renders optional content and dismisses through every supported control", () => {
    const onClose = vi.fn();
    renderWithProviders(
      <Dialog open title="Details" description="A description" footer={<span>Footer</span>} closeLabel="Close" onClose={onClose} className="custom-dialog">
        Body
      </Dialog>,
    );

    expect(screen.getByRole("dialog")).toHaveClass("custom-dialog");
    expect(screen.getByText("A description")).toBeInTheDocument();
    expect(screen.getByText("Footer")).toBeInTheDocument();
    fireEvent.keyDown(window, { key: "Escape" });
    fireEvent.click(screen.getAllByRole("button", { name: "Close" })[0]!);
    fireEvent.click(screen.getAllByRole("button", { name: "Close" })[1]!);
    expect(onClose).toHaveBeenCalledTimes(3);
  });
});
