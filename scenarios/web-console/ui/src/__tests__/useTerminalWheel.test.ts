import { describe, it, expect, vi } from "vitest";
import { renderHook } from "@testing-library/react";
import type { Terminal } from "@xterm/xterm";
import { useTerminalWheel } from "../hooks/useTerminalWheel";

const NORMAL_BUFFER = { id: "normal" };
const ALT_BUFFER = { id: "alternate" };

type WheelHandler = (event: WheelEvent) => boolean;

/**
 * Minimal terminal double that records the handler the hook attaches, so a
 * test can drive it the way xterm's own wheel listener would.
 */
function makeTerminal(options: { mouseTrackingMode: string; onAltBuffer: boolean }) {
  let handler: WheelHandler | null = null;
  const terminal = {
    cols: 80,
    rows: 24,
    modes: { mouseTrackingMode: options.mouseTrackingMode },
    buffer: {
      active: options.onAltBuffer ? ALT_BUFFER : NORMAL_BUFFER,
      normal: NORMAL_BUFFER,
      alternate: ALT_BUFFER,
    },
    scrollLines: vi.fn(),
    attachCustomWheelEventHandler: (next: WheelHandler) => {
      handler = next;
    },
  };
  return {
    terminal: terminal as unknown as Terminal,
    fire: (deltaY: number, deltaMode = 0) => {
      const preventDefault = vi.fn();
      const event = { deltaY, deltaMode, preventDefault } as unknown as WheelEvent;
      const result = handler?.(event);
      return { result, preventDefault };
    },
  };
}

/** A container with a measurable .xterm-screen, giving a 20px cell height. */
function makeContainer(): React.RefObject<HTMLDivElement | null> {
  const container = document.createElement("div");
  const screen = document.createElement("div");
  screen.classList.add("xterm-screen");
  screen.getBoundingClientRect = () =>
    ({ left: 0, top: 0, right: 800, bottom: 480, width: 800, height: 480, x: 0, y: 0, toJSON: () => ({}) }) as DOMRect;
  container.appendChild(screen);
  return { current: container };
}

function render(options: { mouseTrackingMode: string; onAltBuffer: boolean }) {
  const harness = makeTerminal(options);
  const containerRef = makeContainer();
  const scrollBy = vi.fn();
  const view = renderHook(() => {
    useTerminalWheel({ terminal: harness.terminal, containerRef, scrollBy });
  });
  return { ...harness, scrollBy, view };
}

describe("terminal wheel ownership", () => {
  it("claims the wheel when xterm would otherwise emit arrow keys as stdin", () => {
    // A tmux-backed pane running a program that requests no mouse tracking:
    // the alternate buffer has no scrollback, so xterm's built-in handler
    // converts each notch into an Up/Down cursor key and submits it as real
    // input. In an agent composer bound to Up/Down that rewrites the draft.
    const { fire, scrollBy } = render({ mouseTrackingMode: "none", onAltBuffer: true });

    const { result, preventDefault } = fire(-60);

    expect(result).toBe(false); // suppresses the arrow-key emission
    expect(preventDefault).toHaveBeenCalled();
    expect(scrollBy).toHaveBeenCalledWith(-3, "wheel");
  });

  it("defers to xterm when the program owns the wheel through mouse tracking", () => {
    // Claude Code and friends enable mouse tracking; xterm must keep encoding
    // wheel events as mouse reports so the program scrolls itself.
    const { fire, scrollBy } = render({ mouseTrackingMode: "sgr", onAltBuffer: true });

    const { result, preventDefault } = fire(-60);

    expect(result).toBe(true);
    expect(preventDefault).not.toHaveBeenCalled();
    expect(scrollBy).not.toHaveBeenCalled();
  });

  it("defers to xterm when the terminal has real scrollback of its own", () => {
    // A non-tmux pane on the normal buffer scrolls natively, applying xterm's
    // own scrollSensitivity. Taking it over would double-apply sensitivity.
    const { fire, scrollBy } = render({ mouseTrackingMode: "none", onAltBuffer: false });

    const { result, preventDefault } = fire(-60);

    expect(result).toBe(true);
    expect(preventDefault).not.toHaveBeenCalled();
    expect(scrollBy).not.toHaveBeenCalled();
  });

  it("accumulates sub-line trackpad deltas into whole lines", () => {
    const { fire, scrollBy } = render({ mouseTrackingMode: "none", onAltBuffer: true });

    // The screen measures 20px per row, so each event is 0.4 of a line and
    // would round away to nothing on its own.
    fire(8);
    expect(scrollBy).not.toHaveBeenCalled();

    // Five events are two whole lines of travel, and two lines is what the
    // gesture must deliver however finely it was sliced.
    for (let i = 0; i < 4; i += 1) fire(8);
    const moved = scrollBy.mock.calls.reduce((sum, call) => sum + (call[0] as number), 0);
    expect(moved).toBe(2);
  });

  it("still suppresses the arrow-key path when a gesture rounds to zero lines", () => {
    // The suppression must not depend on producing a line: a partial notch
    // that fell through would reach the program as a cursor key.
    const { fire, scrollBy } = render({ mouseTrackingMode: "none", onAltBuffer: true });

    const { result, preventDefault } = fire(8);

    expect(result).toBe(false);
    expect(preventDefault).toHaveBeenCalled();
    expect(scrollBy).not.toHaveBeenCalled();
  });

  it("hands the wheel back to xterm on unmount", () => {
    const { view, terminal, fire, scrollBy } = render({
      mouseTrackingMode: "none",
      onAltBuffer: true,
    });
    expect(terminal).toBeTruthy();

    view.unmount();

    expect(fire(-60).result).toBe(true);
    expect(scrollBy).not.toHaveBeenCalled();
  });

  it("does not attach a handler without a terminal", () => {
    const containerRef = makeContainer();
    const scrollBy = vi.fn();
    expect(() =>
      renderHook(() => {
        useTerminalWheel({ terminal: null, containerRef, scrollBy });
      }),
    ).not.toThrow();
    expect(scrollBy).not.toHaveBeenCalled();
  });
});
