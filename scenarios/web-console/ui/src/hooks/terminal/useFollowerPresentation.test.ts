import { describe, expect, it, vi } from "vitest";
import { renderHook } from "@testing-library/react";
import { useFollowerPresentation } from "./useFollowerPresentation";

function terminalFixture(options: { width?: number; height?: number; cols?: number; rows?: number } = {}) {
  const element = document.createElement("div");
  const screen = document.createElement("div");
  Object.defineProperties(screen, {
    clientWidth: { configurable: true, value: options.width ?? 800 },
    clientHeight: { configurable: true, value: options.height ?? 480 },
  });
  screen.className = "xterm-screen";
  element.appendChild(screen);
  return {
    element,
    cols: options.cols ?? 80,
    rows: options.rows ?? 24,
    options: {} as { fontSize?: number },
    resize: vi.fn(),
  };
}

describe("useFollowerPresentation", () => {
  it("does not create a frame for a leader or an unmeasured pane", () => {
    const { result } = renderHook(() => useFollowerPresentation({
      terminal: null,
      serverSize: { cols: 80, rows: 24 },
      isFollower: false,
      paneSize: { width: 800, height: 600 },
    }));
    expect(result.current).toBeNull();
  });

  it("computes a follower frame without mutating xterm's root element", () => {
    const terminal = terminalFixture({ cols: 40, rows: 12 });
    const { result, rerender } = renderHook(({ paneSize }) => useFollowerPresentation({
      terminal: terminal as never,
      serverSize: { cols: 80, rows: 24 },
      isFollower: true,
      paneSize,
    }), { initialProps: { paneSize: { width: 1000, height: 700 } } });

    expect(result.current).not.toBeNull();
    expect(terminal.element.style.cssText).toBe("");
    expect(terminal.resize).not.toHaveBeenCalled();
    expect(terminal.options.fontSize).toBeUndefined();

    rerender({ paneSize: { width: 0, height: 0 } });
    expect(result.current).toBeNull();
    expect(terminal.element.style.cssText).toBe("");
  });

  it("uses the conservative aspect fallback and compact strip for a tiny pane", () => {
    const terminal = terminalFixture({ cols: 0, rows: 0, width: 0, height: 0 });
    const { result } = renderHook(() => useFollowerPresentation({
      terminal: terminal as never,
      serverSize: { cols: 80, rows: 24 },
      isFollower: true,
      paneSize: { width: 120, height: 120 },
    }));

    expect(result.current?.tier).toBe("strip");
    expect(result.current?.screenRect).toEqual(result.current?.rect);
  });

  it("does not apply DOM changes when the fit addon is unavailable", () => {
    const terminal = terminalFixture();
    const { result } = renderHook(() => useFollowerPresentation({
      terminal: terminal as never,
      serverSize: { cols: 80, rows: 24 },
      isFollower: true,
      paneSize: { width: 800, height: 600 },
    }));

    expect(result.current).not.toBeNull();
    expect(terminal.element.style.position).toBe("");
  });
});
