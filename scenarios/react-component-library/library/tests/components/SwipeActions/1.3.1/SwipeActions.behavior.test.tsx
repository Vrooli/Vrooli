import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { SwipeActions } from "../../../../components/SwipeActions/versions/1.3.1/SwipeActions";

describe("SwipeActions current release contract", () => {
  it("keeps a revealed action available and exposes the gesture claim", async () => {
    const onSelect = vi.fn();
    render(<SwipeActions defaultOpen actions={[{ id: "archive", label: "Archive", onSelect }]}><span>Message</span></SwipeActions>);
    expect(screen.getByTestId("patterns.swipe-actions")).toHaveAttribute("data-rcl-gesture-claim");
    fireEvent.click(screen.getByRole("button", { name: "Archive" }), { detail: 0 });
    expect(onSelect).toHaveBeenCalledOnce();
  });
});
