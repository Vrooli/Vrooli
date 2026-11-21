import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  useLongPress,
  type LongPressOrigin,
} from "@vrooli/react-component-library/useLongPress/1.0.2";

function Harness({
  disabled = false,
  onLongPress = vi.fn(),
  onClick,
  onInnerClick,
}: {
  disabled?: boolean;
  onLongPress?: (origin: LongPressOrigin) => void;
  onClick?: () => void;
  onInnerClick?: () => void;
}) {
  const { longPressProps } = useLongPress({ onLongPress, disabled });
  return (
    <div data-testid="target" onClick={onClick} {...longPressProps}>
      <button type="button" onClick={onInnerClick}>
        inner
      </button>
    </div>
  );
}

describe("useLongPress", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("fires once with the pointer origin after the token delay", () => {
    const callback = vi.fn();
    render(<Harness onLongPress={callback} />);
    const target = screen.getByTestId("target");
    fireEvent.pointerDown(target, {
      pointerId: 1,
      pointerType: "touch",
      clientX: 12,
      clientY: 18,
      button: 0,
    });
    act(() => vi.advanceTimersByTime(450));
    fireEvent.pointerUp(target, { pointerId: 1, pointerType: "touch" });
    expect(callback).toHaveBeenCalledTimes(1);
    expect(callback).toHaveBeenCalledWith({ x: 12, y: 18, pointerType: "touch" });
  });

  it("cancels on movement, release, pointer cancellation, and disabled state", () => {
    const callback = vi.fn();
    const { rerender } = render(<Harness onLongPress={callback} />);
    const target = screen.getByTestId("target");
    fireEvent.pointerDown(target, { pointerId: 1, pointerType: "touch", clientX: 0, clientY: 0 });
    fireEvent.pointerMove(target, { pointerId: 1, clientX: 15, clientY: 0 });
    act(() => vi.advanceTimersByTime(500));
    fireEvent.pointerDown(target, { pointerId: 2, pointerType: "touch", clientX: 0, clientY: 0 });
    fireEvent.pointerUp(target, { pointerId: 2 });
    act(() => vi.advanceTimersByTime(500));
    fireEvent.pointerDown(target, { pointerId: 3, pointerType: "touch", clientX: 0, clientY: 0 });
    fireEvent.pointerCancel(target, { pointerId: 3 });
    act(() => vi.advanceTimersByTime(500));
    rerender(<Harness disabled onLongPress={callback} />);
    fireEvent.pointerDown(screen.getByTestId("target"), { pointerId: 4, pointerType: "mouse" });
    act(() => vi.advanceTimersByTime(500));
    expect(callback).not.toHaveBeenCalled();
  });

  it("cancels when the pointer moves beyond tolerance outside the element", () => {
    const callback = vi.fn();
    render(<Harness onLongPress={callback} />);
    fireEvent.pointerDown(screen.getByTestId("target"), {
      pointerId: 1,
      pointerType: "mouse",
      clientX: 0,
      clientY: 0,
    });
    fireEvent.pointerMove(window, { pointerId: 1, clientX: 40, clientY: 0 });
    act(() => vi.advanceTimersByTime(500));
    expect(callback).not.toHaveBeenCalled();
  });

  it("cancels when the pointer is released outside the element", () => {
    const callback = vi.fn();
    render(<Harness onLongPress={callback} />);
    fireEvent.pointerDown(screen.getByTestId("target"), { pointerId: 1, pointerType: "mouse" });
    fireEvent.pointerUp(document.body, { pointerId: 1 });
    act(() => vi.advanceTimersByTime(500));
    expect(callback).not.toHaveBeenCalled();
  });

  it("never captures the pointer, so ordinary clicks reach the controls it wraps", () => {
    const setPointerCapture = vi.fn();
    const original = Element.prototype.setPointerCapture;
    Element.prototype.setPointerCapture = setPointerCapture;
    try {
      const innerClick = vi.fn();
      render(<Harness onInnerClick={innerClick} />);
      const inner = screen.getByRole("button", { name: "inner" });
      fireEvent.pointerDown(inner, { pointerId: 1, pointerType: "mouse", button: 0 });
      fireEvent.pointerUp(inner, { pointerId: 1, pointerType: "mouse" });
      fireEvent.click(inner);
      expect(setPointerCapture).not.toHaveBeenCalled();
      expect(innerClick).toHaveBeenCalledOnce();
    } finally {
      Element.prototype.setPointerCapture = original;
    }
  });

  it("suppresses the click and native context menu following a fired press", () => {
    const click = vi.fn();
    const callback = vi.fn();
    render(<Harness onLongPress={callback} onClick={click} />);
    const target = screen.getByTestId("target");
    fireEvent.pointerDown(target, { pointerId: 1, pointerType: "mouse", button: 0 });
    act(() => vi.advanceTimersByTime(450));
    fireEvent.click(target);
    fireEvent.contextMenu(target);
    expect(callback).toHaveBeenCalledOnce();
    expect(click).not.toHaveBeenCalled();
  });
});
