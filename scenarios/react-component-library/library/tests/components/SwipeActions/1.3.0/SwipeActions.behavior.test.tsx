import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithProviders } from "../../../../../ui/src/test-utils";
import { SwipeActions } from "../../../../components/SwipeActions/versions/1.3.0/SwipeActions";

describe("SwipeActions gesture contract", () => {
  it("keeps a tap on a revealed action an available accelerator", () => {
    const onSelect = vi.fn();
    renderWithProviders(
      <SwipeActions defaultOpen actions={[{ id: "archive", label: "Archive", onSelect }]}>
        <span>Message</span>
      </SwipeActions>,
    );
    fireEvent.click(screen.getByRole("button", { name: "Archive" }), { detail: 0 });
    expect(onSelect).toHaveBeenCalledOnce();
  });

  it("publishes the nested inline-axis claim while open", () => {
    renderWithProviders(
      <SwipeActions defaultOpen actions={[{ id: "archive", label: "Archive", onSelect: vi.fn() }]}>
        <span>Message</span>
      </SwipeActions>,
    );
    expect(screen.getByTestId("patterns.swipe-actions")).toHaveAttribute("data-rcl-gesture-claim");
  });
});
