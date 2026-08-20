import { describe, expect, it, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";

import { renderWithProviders } from "../../test-utils";
import { Dialog } from "./dialog";

describe("Dialog", () => {
  it("renders its complete surface and closes from Escape and controls", () => {
    const onClose = vi.fn();
    renderWithProviders(
      <Dialog open title="Details" description="More context" closeLabel="Close" onClose={onClose} footer={<span>Footer</span>}>
        Body
      </Dialog>,
    );

    expect(screen.getByRole("dialog", { name: "Details" })).toHaveTextContent("More context");
    expect(screen.getByText("Footer")).toBeInTheDocument();
    fireEvent.keyDown(window, { key: "Escape" });
    fireEvent.click(screen.getAllByRole("button", { name: "Close" })[1]!);
    expect(onClose).toHaveBeenCalledTimes(2);
  });

  it("renders nothing when closed and supports a minimal open surface", () => {
    const { rerender } = renderWithProviders(<Dialog open={false} title="Hidden" closeLabel="Close" onClose={vi.fn()}>Hidden body</Dialog>);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();

    rerender(<Dialog open title="Minimal" closeLabel="Close" onClose={vi.fn()}>Visible body</Dialog>);
    expect(screen.getByRole("dialog", { name: "Minimal" })).toHaveTextContent("Visible body");
    expect(screen.getByRole("dialog")).not.toHaveAttribute("aria-describedby");
  });
});
