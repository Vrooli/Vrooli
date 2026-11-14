import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { renderWithQueryClient } from "../test-utils";
import { ViewModeCycleButton } from "./ViewModeCycleButton";

describe("ViewModeCycleButton", () => {
  it("keeps the control and glyph mounted across every view mode", () => {
    const onCycle = vi.fn();
    const { rerender } = renderWithQueryClient(
      <ViewModeCycleButton mode="flat" onCycle={onCycle} groupingAvailable />,
    );

    const modes = ["flat", "grouped", "tree"] as const;
    for (const mode of modes) {
      rerender(
        <ViewModeCycleButton mode={mode} onCycle={onCycle} groupingAvailable />,
      );
      const button = screen.getByTestId("view-mode-cycle-button");
      expect(button).toBeVisible();
      expect(button.querySelector("svg")).not.toBeNull();
    }

    fireEvent.click(screen.getByTestId("view-mode-cycle-button"));
    expect(onCycle).toHaveBeenCalledTimes(1);
  });
});
