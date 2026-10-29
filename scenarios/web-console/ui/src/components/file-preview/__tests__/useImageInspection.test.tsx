import { act, cleanup, fireEvent, screen } from "@testing-library/react";
import { renderWithProviders as render } from "../../../test-utils";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useImageInspection } from "../useImageInspection";

const stage = { width: 400, height: 300 }, image = { width: 1000, height: 800 };
let nextFrame: FrameRequestCallback | null = null;
beforeEach(() => {
  vi.stubGlobal("requestAnimationFrame", vi.fn((callback: FrameRequestCallback) => { nextFrame = callback; return 1; }));
  vi.stubGlobal("cancelAnimationFrame", vi.fn(() => { nextFrame = null; }));
});
afterEach(() => { cleanup(); vi.unstubAllGlobals(); });
function Inspector({ width = 400 }: { width?: number }) {
  const reader = useImageInspection(width === 400 ? stage : { width, height: 300 }, image, 0);
  return <div ref={reader.stageElement} data-testid="stage" {...reader.pointerEvents}>
    <output>{JSON.stringify(reader.view)}</output>
    <button onClick={() => { reader.zoom(1); }}>100%</button>
    <button onClick={reader.reset}>Fit</button>
  </div>;
}
function pointer(element: HTMLElement, type: string, id: number, x: number, y: number) {
  const event = new MouseEvent(type, { bubbles: true, clientX: x, clientY: y, button: 0 });
  Object.defineProperties(event, { pointerId: { value: id }, pointerType: { value: "touch" } });
  fireEvent(element, event);
}
function paint() { act(() => { const callback = nextFrame; nextFrame = null; callback?.(0); }); }
function canvas() {
  const element = screen.getByTestId("stage");
  element.setPointerCapture = vi.fn();
  vi.spyOn(element, "getBoundingClientRect").mockReturnValue(new DOMRect(0, 0, 400, 300));
  return element;
}

describe("image inspection gestures", () => {
  it("pans within image bounds, keeps an ended gesture stationary and restores Fit", () => {
    render(<Inspector />);
    const element = canvas();
    fireEvent.click(screen.getByText("100%"));
    pointer(element, "pointerdown", 1, 200, 150);
    pointer(element, "pointermove", 1, 900, 900); paint();
    expect(screen.getByRole("status")).toHaveTextContent('{"scale":1,"x":300,"y":250}');
    pointer(element, "pointerup", 1, 900, 900);
    pointer(element, "pointermove", 1, 0, 0); paint();
    expect(screen.getByRole("status")).toHaveTextContent('"x":300');
    fireEvent.click(screen.getByText("Fit"));
    expect(screen.getByRole("status")).toHaveTextContent('{"scale":0.375,"x":0,"y":0}');
  });
  it("pinches around the midpoint and supports continuing with one remaining pointer", () => {
    render(<Inspector />); const element = canvas();
    pointer(element, "pointerdown", 1, 160, 150);
    pointer(element, "pointerdown", 2, 240, 150);
    pointer(element, "pointermove", 1, 120, 150);
    pointer(element, "pointermove", 2, 280, 150); paint();
    expect(screen.getByRole("status")).toHaveTextContent('{"scale":0.75,"x":0,"y":0}');
    pointer(element, "pointerup", 2, 280, 150);
    pointer(element, "pointermove", 1, 160, 170); paint();
    expect(screen.getByRole("status")).toHaveTextContent('{"scale":0.75,"x":40,"y":20}');
  });
  it("zooms at the wheel focal point and releases pending frames/listeners on close", () => {
    const { unmount } = render(<Inspector />); const element = canvas();
    fireEvent.click(screen.getByText("100%"));
    const wheel = new WheelEvent("wheel", { deltaY: -Math.log(2) / 0.002, clientX: 250, clientY: 150, cancelable: true });
    act(() => { element.dispatchEvent(wheel); });
    expect(wheel.defaultPrevented).toBe(true);
    expect(screen.getByRole("status")).toHaveTextContent('{"scale":2,"x":-50,"y":0}');
    pointer(element, "pointerdown", 1, 200, 150);
    pointer(element, "pointermove", 1, 250, 150);
    unmount();
    expect(cancelAnimationFrame).toHaveBeenCalledWith(1);
    const after = new WheelEvent("wheel", { deltaY: 100, cancelable: true });
    element.dispatchEvent(after);
    expect(after.defaultPrevented).toBe(false);
  });
  it("refits a resized stage and invalidates active pointer coordinates", () => {
    const { rerender } = render(<Inspector />); const element = canvas();
    pointer(element, "pointerdown", 1, 200, 150);
    rerender(<Inspector width={200} />);
    expect(screen.getByRole("status")).toHaveTextContent('{"scale":0.2,"x":0,"y":0}');
    pointer(element, "pointermove", 1, 300, 250); paint();
    expect(screen.getByRole("status")).toHaveTextContent('"scale":0.2');
  });
});
