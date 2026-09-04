import { act, fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { useLongPress, type LongPressOrigin } from "../../../../hooks/useLongPress/versions/1.0.0/useLongPress";

function Harness({ disabled = false, onLongPress = vi.fn(), onClick }: { disabled?: boolean; onLongPress?: (origin: LongPressOrigin) => void; onClick?: () => void }) {
  const { longPressProps } = useLongPress({ onLongPress, disabled });
  return <button type="button" data-testid="target" onClick={onClick} {...longPressProps}>row</button>;
}

describe("useLongPress", () => {
  it("fires once with the pointer origin after the token delay", () => {
    vi.useFakeTimers(); const callback = vi.fn(); render(<Harness onLongPress={callback} />); const target = screen.getByTestId("target");
    fireEvent.pointerDown(target, { pointerId: 1, pointerType: "touch", clientX: 12, clientY: 18, button: 0 });
    act(() => vi.advanceTimersByTime(450)); fireEvent.pointerUp(target, { pointerId: 1, pointerType: "touch" });
    expect(callback).toHaveBeenCalledTimes(1); expect(callback).toHaveBeenCalledWith({ x: 12, y: 18, pointerType: "touch" }); vi.useRealTimers();
  });
  it("cancels on movement, release, pointer cancellation, and disabled state", () => {
    vi.useFakeTimers(); const moved = vi.fn(); const { rerender } = render(<Harness onLongPress={moved} />); const target = screen.getByTestId("target");
    fireEvent.pointerDown(target, { pointerId: 1, pointerType: "touch", clientX: 0, clientY: 0, button: 0 }); fireEvent.pointerMove(target, { pointerId: 1, clientX: 15, clientY: 0 }); act(() => vi.advanceTimersByTime(500));
    fireEvent.pointerDown(target, { pointerId: 2, pointerType: "touch", clientX: 0, clientY: 0, button: 0 }); fireEvent.pointerUp(target, { pointerId: 2 });
    fireEvent.pointerDown(target, { pointerId: 3, pointerType: "touch", clientX: 0, clientY: 0, button: 0 }); fireEvent.pointerCancel(target, { pointerId: 3 });
    rerender(<Harness disabled onLongPress={moved} />); fireEvent.pointerDown(screen.getByTestId("target"), { pointerId: 4, pointerType: "mouse", button: 0 }); act(() => vi.advanceTimersByTime(500));
    expect(moved).not.toHaveBeenCalled(); vi.useRealTimers();
  });
  it("suppresses the click and native context menu following a fired press", () => {
    vi.useFakeTimers(); const click = vi.fn(); const callback = vi.fn(); render(<Harness onLongPress={callback} onClick={click} />); const target = screen.getByTestId("target");
    fireEvent.pointerDown(target, { pointerId: 1, pointerType: "mouse", button: 0 }); act(() => vi.advanceTimersByTime(450)); fireEvent.click(target); fireEvent.contextMenu(target); expect(callback).toHaveBeenCalledOnce(); expect(click).not.toHaveBeenCalled(); vi.useRealTimers();
  });
});
