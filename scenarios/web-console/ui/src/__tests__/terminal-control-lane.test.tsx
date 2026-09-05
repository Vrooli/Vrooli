import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useTerminalTouch } from "../hooks/useTerminalTouch";

function touchEvent(type: "touchstart" | "touchmove", x: number, y: number): Event {
  const touch = { identifier: 0, clientX: x, clientY: y, target: null } as unknown as Touch;
  const event = new Event(type, { bubbles: true, cancelable: true }) as Event & {
    touches: Touch[];
    changedTouches: Touch[];
    targetTouches: Touch[];
  };
  Object.defineProperty(event, "touches", { value: [touch] });
  Object.defineProperty(event, "changedTouches", { value: [touch] });
  Object.defineProperty(event, "targetTouches", { value: [touch] });
  return event;
}

describe("terminal control lane", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("sends mouse-wheel controls without entering the reliable submit path", () => {
    const container = document.createElement("div");
    const screen = document.createElement("div");
    screen.className = "xterm-screen";
    screen.getBoundingClientRect = () => ({
      left: 0, top: 0, right: 800, bottom: 480, width: 800, height: 480,
      x: 0, y: 0, toJSON: () => ({}),
    });
    container.appendChild(screen);
    const sendControl = vi.fn<(data: string) => boolean>(() => true);
    const terminal = {
      cols: 80,
      rows: 24,
      modes: { mouseTrackingMode: "sgr" },
      scrollLines: vi.fn(),
      clearSelection: vi.fn(),
      getSelection: vi.fn(() => ""),
      onSelectionChange: vi.fn(() => ({ dispose: vi.fn() })),
    };

    renderHook(() => useTerminalTouch({
      terminal: terminal as never,
      containerRef: { current: container },
      sendControl,
    }));

    act(() => container.dispatchEvent(touchEvent("touchstart", 100, 200)));
    act(() => container.dispatchEvent(touchEvent("touchmove", 100, 160)));
    act(() => container.dispatchEvent(touchEvent("touchmove", 100, 140)));
    act(() => {
      vi.runOnlyPendingTimers();
    });

    expect(sendControl).toHaveBeenCalled();
    expect(sendControl.mock.calls[0]?.[0]).toMatch(/^\x1b\[</);
    expect(terminal.scrollLines).not.toHaveBeenCalled();
  });
});
