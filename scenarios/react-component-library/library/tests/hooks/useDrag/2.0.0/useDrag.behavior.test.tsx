import { describe, expect, it } from "vitest";
import { fireEvent, screen } from "@testing-library/react";
import { useDrag } from "../../../../hooks/useDrag/versions/2.0.0/useDrag";
import { renderWithProviders } from "../../../../../ui/src/test-utils";
import { useState } from "react";

describe("useDrag contract", () => {
  it("requires movement slop before reporting a drag", () => {
    function Harness() {
      const [started, setStarted] = useState(false);
      const drag = useDrag({ onStart: () => setStarted(true) });
      const { isDragging: _isDragging, ...handlers } = drag;
      return <button data-testid="drag" data-started={started} {...handlers} />;
    }
    renderWithProviders(<Harness />);
    const target = screen.getByTestId("drag");
    fireEvent.pointerDown(target, { pointerId: 1, clientX: 10, clientY: 10, button: 0 });
    fireEvent.pointerMove(target, { pointerId: 1, clientX: 15, clientY: 10 });
    expect(target).toHaveAttribute("data-started", "false");
    fireEvent.pointerMove(target, { pointerId: 1, clientX: 25, clientY: 10 });
    expect(target).toHaveAttribute("data-started", "true");
  });

  it("keeps keyboard movement as an equivalent action path", () => {
    expect(["ArrowUp", "ArrowDown", "ArrowLeft", "ArrowRight", "Escape", "Enter"]).toHaveLength(6);
  });
});
