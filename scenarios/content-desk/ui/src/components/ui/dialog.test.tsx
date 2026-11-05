import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { Dialog } from "./dialog";
import { renderWithProviders } from "../../test-utils";

describe("Dialog", () => {
  it("is absent when closed and supports backdrop, close-button, and Escape dismissal", () => {
    const onClose = vi.fn();
    const { rerender } = renderWithProviders(<Dialog open={false} title="Review" closeLabel="Close review" onClose={onClose}>Body</Dialog>);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    rerender(<Dialog open title="Review" description="Check this" closeLabel="Close review" footer={<span>Footer</span>} onClose={onClose}>Body</Dialog>);
    expect(screen.getByRole("dialog")).toHaveAccessibleDescription("Check this");
    fireEvent.keyDown(window, { key: "Escape" });
    const closeButtons = screen.getAllByRole("button", { name: "Close review" });
    expect(closeButtons).toHaveLength(2);
    fireEvent.click(closeButtons[1]!);
    expect(onClose).toHaveBeenCalledTimes(2);
  });
});
