import { renderWithProviders as render } from "../../test-utils";
import { act, renderHook } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { useTerminalBackgroundDetector } from "./useTerminalBackgroundDetector";

function terminalStub() {
  let render: (() => void) | undefined;
  const osc = new Map<number, (data: string) => boolean>();
  const cell = {
    isBgDefault: () => true,
    isBgPalette: () => false,
    isBgRGB: () => true,
    getBgColor: () => 0x112233,
  };
  const line = { getCell: () => cell };
  const term = {
    rows: 2,
    cols: 3,
    buffer: { active: { viewportY: 0, getNullCell: () => cell, getLine: () => line } },
    onRender: (cb: () => void) => { render = cb; return { dispose: vi.fn() }; },
    parser: { registerOscHandler: (code: number, cb: (data: string) => boolean) => { osc.set(code, cb); return { dispose: vi.fn() }; } },
  };
  return { term, fireRender: () => render?.(), fireOsc: (code: number, data: string) => osc.get(code)?.(data) };
}

describe("useTerminalBackgroundDetector", () => {
  afterEach(() => vi.useRealTimers());

  it("clears the color while disabled and samples the visible buffer", () => {
    vi.useFakeTimers();
    const onColor = vi.fn();
    const stub = terminalStub();
    const { unmount } = renderHook(
      ({ enabled }) => useTerminalBackgroundDetector(stub.term as never, { enabled, defaultBackground: "#000000", onColor }),
      { initialProps: { enabled: false } },
    );
    expect(onColor).toHaveBeenCalledWith(null);
    unmount();

    const rendered = renderHook(() => useTerminalBackgroundDetector(stub.term as never, { enabled: true, defaultBackground: "#000000", onColor }));
    act(() => { vi.advanceTimersByTime(200); });
    expect(onColor).toHaveBeenLastCalledWith("#000000");
    Object.defineProperty(document, "visibilityState", { configurable: true, value: "visible" });
    act(() => { document.dispatchEvent(new Event("visibilitychange")); });
    act(() => { vi.advanceTimersByTime(200); });
    expect(onColor).toHaveBeenLastCalledWith("#000000");
    stub.fireOsc(11, "rgb:aa/bb/cc");
    act(() => { vi.advanceTimersByTime(200); });
    expect(onColor).toHaveBeenLastCalledWith("#aabbcc");
    stub.fireOsc(111, "");
    stub.fireRender();
    act(() => { vi.advanceTimersByTime(200); });
    expect(onColor).toHaveBeenLastCalledWith("#000000");
    rendered.unmount();
    expect(onColor).toHaveBeenLastCalledWith(null);
  });
});
