import { describe, expect, it, vi } from "vitest";
import type { Terminal } from "@xterm/xterm";
import {
  createScrollController,
  createWheelLineAccumulator,
  scrollTransportFor,
  terminalHasLocalScrollback,
  wheelDeltaToLines,
} from "./terminalScroll";

const NORMAL_BUFFER = { id: "normal" };
const ALT_BUFFER = { id: "alternate" };

/**
 * A terminal on the normal buffer: it holds real scrollback, so a
 * `scrollLines` call moves something.
 */
function terminal(mouseTrackingMode: string) {
  return {
    cols: 80,
    rows: 24,
    modes: { mouseTrackingMode },
    scrollLines: vi.fn(),
    buffer: { active: NORMAL_BUFFER, normal: NORMAL_BUFFER, alternate: ALT_BUFFER },
  } as unknown as Terminal & { scrollLines: ReturnType<typeof vi.fn> };
}

/**
 * A terminal parked on the alternate buffer, which is where every tmux-backed
 * pane lives: the tmux client emits `\x1b[?1049h` on attach and never leaves.
 * There is no client-side scrollback here, so `scrollLines` cannot work.
 */
function altBufferTerminal(mouseTrackingMode: string) {
  return {
    cols: 80,
    rows: 24,
    modes: { mouseTrackingMode },
    scrollLines: vi.fn(),
    buffer: { active: ALT_BUFFER, normal: NORMAL_BUFFER, alternate: ALT_BUFFER },
  } as unknown as Terminal & { scrollLines: ReturnType<typeof vi.fn> };
}

function wheel(deltaY: number, deltaMode = 0): WheelEvent {
  return { deltaY, deltaMode } as WheelEvent;
}

describe("terminal scroll seam", () => {
  it("recognizes absent, incomplete, and active mouse modes", () => {
    const noModes = { cols: 80, rows: 24, scrollLines: vi.fn() } as unknown as Terminal;
    const sendControl = vi.fn(() => true);
    createScrollController(() => null, sendControl).scrollBy(1, "wheel");
    createScrollController(() => noModes, sendControl).scrollBy(1, "wheel");
    expect(noModes.scrollLines).toHaveBeenCalledWith(1);
    expect(sendControl).not.toHaveBeenCalled();
  });

  it("ignores zero movement and holds a scaled scroll below one line", () => {
    const term = terminal("none");
    const controller = createScrollController(() => term, vi.fn(), { getSensitivity: () => 0.01 });
    controller.scrollBy(0, "wheel");
    controller.scrollBy(1, "wheel");
    expect(term.scrollLines).not.toHaveBeenCalled();
  });

  it("accumulates a reduced sensitivity instead of discarding every scroll", () => {
    // At 0.4 each single-line scroll used to round to zero and vanish, so the
    // setting did not slow scrolling down, it switched it off.
    const term = terminal("none");
    const controller = createScrollController(() => term, vi.fn(), { getSensitivity: () => 0.4 });
    for (let i = 0; i < 10; i += 1) controller.scrollBy(1, "wheel");
    const moved = term.scrollLines.mock.calls.reduce((sum, call) => sum + (call[0] as number), 0);
    expect(moved).toBe(4);
  });

  it("keeps a separate remainder per scroll source", () => {
    // A wheel notch and a finger drag are different gestures; one must not
    // consume the fraction the other carried.
    const term = terminal("none");
    const controller = createScrollController(() => term, vi.fn(), { getSensitivity: () => 0.3 });
    controller.scrollBy(1, "wheel");
    controller.scrollBy(1, "touch");
    expect(term.scrollLines).not.toHaveBeenCalled();
    // The wheel's own banked 0.3 carries it over, unspent by the touch scroll.
    controller.scrollBy(1, "wheel");
    expect(term.scrollLines).toHaveBeenCalledTimes(1);
  });

  it.each(["touch", "wheel", "programmatic"])("scrolls locally when mouse mode is off (%s)", (source) => {
    const term = terminal("none");
    const sendControl = vi.fn(() => true);
    createScrollController(() => term, sendControl).scrollBy(-2, source as "touch");
    expect(term.scrollLines).toHaveBeenCalledWith(-2);
    expect(sendControl).not.toHaveBeenCalled();
  });

  it.each(["touch", "wheel", "programmatic"])("uses control frames when mouse mode is on (%s)", (source) => {
    const term = terminal("sgr");
    const sendControl = vi.fn(() => true);
    const controller = createScrollController(() => term, sendControl);
    controller.scrollBy(2, source as "touch");
    controller.flush();
    expect(term.scrollLines).not.toHaveBeenCalled();
    expect(sendControl).toHaveBeenCalledTimes(1);
    expect((sendControl.mock.calls[0] as unknown as [string] | undefined)?.[0]).toContain("\x1b[<65;");
  });

  it("accumulates one control frame per animation flush", () => {
    const term = terminal("sgr");
    const sendControl = vi.fn(() => true);
    const controller = createScrollController(() => term, sendControl);

    controller.scrollBy(2, "wheel");
    controller.scrollBy(3, "wheel");
    controller.flush();

    expect(sendControl).toHaveBeenCalledTimes(1);
    expect((sendControl.mock.calls[0] as unknown as [string] | undefined)?.[0]).toBe(
      "\x1b[<65;41;13M".repeat(5),
    );
  });

  it("applies touch and wheel sensitivity independently", () => {
    const sequence = "\x1b[<65;41;13M";
    const send = (source: "touch" | "wheel", sensitivity: number) => {
      const term = terminal("sgr");
      const sendControl = vi.fn<(data: string) => boolean>(() => true);
      const controller = createScrollController(() => term, sendControl, {
        getSensitivity: (candidate) => candidate === source ? sensitivity : 1,
      });
      controller.scrollBy(2, source);
      controller.flush();
      return sendControl.mock.calls[0]?.[0];
    };

    expect(send("touch", 2)).toBe(sequence.repeat(4));
    expect(send("wheel", 1)).toBe(sequence.repeat(2));
    expect(send("touch", 1)).toBe(sequence.repeat(2));
    expect(send("wheel", 2)).toBe(sequence.repeat(4));
  });

  it("keeps pending mouse scroll until output acknowledges a frame and the rate limit opens", () => {
    const term = terminal("sgr");
    const sendControl = vi.fn(() => true);
    let timestamp = 0;
    const controller = createScrollController(() => term, sendControl, {
      now: () => timestamp,
      maxUnacknowledgedFrames: 1,
      maxFramesPerSecond: 10,
    });

    controller.scrollBy(1, "wheel");
    controller.flush();
    controller.scrollBy(1, "wheel");
    controller.flush();
    expect(sendControl).toHaveBeenCalledTimes(1);

    controller.notifyOutput();
    timestamp = 101;
    controller.flush();
    expect(sendControl).toHaveBeenCalledTimes(2);
  });

  it("recovers when the acknowledgement gate reaches its cap without output", () => {
    vi.useFakeTimers();
    try {
      const term = terminal("sgr");
      const sendControl = vi.fn(() => true);
      let timestamp = 0;
      const controller = createScrollController(() => term, sendControl, {
        maxUnacknowledgedFrames: 1,
        acknowledgementTimeoutMs: 100,
        maxFramesPerSecond: 1_000,
        now: () => timestamp,
      });

      controller.scrollBy(1, "touch");
      controller.flush();
      expect(controller.getUnacknowledgedFrames()).toBe(1);

      controller.scrollBy(1, "touch");
      controller.flush();
      expect(sendControl).toHaveBeenCalledTimes(1);

      timestamp = 1000;
      vi.advanceTimersByTime(100);
      expect(controller.getUnacknowledgedFrames()).toBe(0);
      controller.flush();
      expect(sendControl).toHaveBeenCalledTimes(2);
    } finally {
      vi.useRealTimers();
    }
  });

  it("drops a queued frame when the server leaves mouse tracking and tolerates rejected frames", () => {
    const term = terminal("sgr");
    const sendControl = vi.fn(() => false);
    const controller = createScrollController(() => term, sendControl);
    controller.scrollBy(-2, "touch");
    controller.flush();
    expect(sendControl).toHaveBeenCalledTimes(1);

    (term.modes as unknown as { mouseTrackingMode: string }).mouseTrackingMode = "none";
    controller.scrollBy(2, "touch");
    controller.flush();
    expect(term.scrollLines).toHaveBeenCalledWith(2);
  });
});

describe("scroll transport selection", () => {
  it("treats the alternate buffer as having no local scrollback", () => {
    expect(terminalHasLocalScrollback(terminal("none"))).toBe(true);
    expect(terminalHasLocalScrollback(altBufferTerminal("none"))).toBe(false);
    expect(terminalHasLocalScrollback(null)).toBe(false);
  });

  it("assumes local scrollback when the terminal does not expose buffer identity", () => {
    const bare = { cols: 80, rows: 24, scrollLines: vi.fn() } as unknown as Terminal;
    expect(terminalHasLocalScrollback(bare)).toBe(true);
    expect(scrollTransportFor(bare)).toBe("local");
  });

  it("routes each terminal state to the mechanism that can actually scroll it", () => {
    // The program asked for the wheel, so it scrolls itself.
    expect(scrollTransportFor(terminal("sgr"))).toBe("mouse-report");
    expect(scrollTransportFor(altBufferTerminal("sgr"))).toBe("mouse-report");
    // Real client-side scrollback: no network round trip needed.
    expect(scrollTransportFor(terminal("none"))).toBe("local");
    // Neither: only the backend holds history. This is every tmux pane whose
    // program does not request mouse tracking.
    expect(scrollTransportFor(altBufferTerminal("none"))).toBe("server-scroll");
    expect(scrollTransportFor(null)).toBeNull();
  });

  it("asks the server to scroll when neither the client nor the program can", () => {
    const term = altBufferTerminal("none");
    const sendControl = vi.fn(() => true);
    const sendScroll = vi.fn(() => true);
    const controller = createScrollController(() => term, sendControl, { sendScroll });

    controller.scrollBy(-3, "wheel");
    controller.flush();

    // The regression this replaces: scrollLines() against the alternate
    // buffer is a silent no-op, so the operator saw nothing happen at all.
    expect(term.scrollLines).not.toHaveBeenCalled();
    expect(sendControl).not.toHaveBeenCalled();
    expect(sendScroll).toHaveBeenCalledWith(-3);
  });

  it("batches server scrolls onto one frame and respects the acknowledgement gate", () => {
    const term = altBufferTerminal("none");
    const sendScroll = vi.fn(() => true);
    let timestamp = 0;
    const controller = createScrollController(() => term, vi.fn(), {
      sendScroll,
      now: () => timestamp,
      maxUnacknowledgedFrames: 1,
      maxFramesPerSecond: 10,
    });

    controller.scrollBy(-2, "wheel");
    controller.scrollBy(-3, "wheel");
    controller.flush();
    expect(sendScroll).toHaveBeenCalledTimes(1);
    expect(sendScroll).toHaveBeenCalledWith(-5);

    controller.scrollBy(-1, "wheel");
    controller.flush();
    expect(sendScroll).toHaveBeenCalledTimes(1);

    controller.notifyOutput();
    timestamp = 101;
    controller.flush();
    expect(sendScroll).toHaveBeenCalledTimes(2);
  });

  it("drops queued lines when the program leaves the alternate buffer mid-gesture", () => {
    const term = altBufferTerminal("none");
    const sendScroll = vi.fn(() => true);
    const controller = createScrollController(() => term, vi.fn(), { sendScroll });

    controller.scrollBy(-4, "wheel");
    (term.buffer as unknown as { active: unknown }).active = NORMAL_BUFFER;
    controller.flush();

    // Those lines addressed a view that no longer exists; replaying them
    // against the normal buffer would scroll somewhere the operator never saw.
    expect(sendScroll).not.toHaveBeenCalled();
    expect(term.scrollLines).not.toHaveBeenCalled();
  });

  it("does not pretend to scroll when no server seam was provided", () => {
    const term = altBufferTerminal("none");
    const controller = createScrollController(() => term, vi.fn());
    controller.scrollBy(-2, "wheel");
    controller.flush();
    expect(term.scrollLines).not.toHaveBeenCalled();
    expect(controller.getUnacknowledgedFrames()).toBe(0);
  });

  it("applies wheel sensitivity on the server-scroll path", () => {
    const term = altBufferTerminal("none");
    const sendScroll = vi.fn(() => true);
    const controller = createScrollController(() => term, vi.fn(), {
      sendScroll,
      getSensitivity: (source) => (source === "wheel" ? 3 : 1),
    });
    controller.scrollBy(-2, "wheel");
    controller.flush();
    expect(sendScroll).toHaveBeenCalledWith(-6);
  });
});

describe("wheel delta conversion", () => {
  it("converts pixel, line, and page deltas into terminal lines", () => {
    expect(wheelDeltaToLines(wheel(60), 20, 24)).toBe(3);
    expect(wheelDeltaToLines(wheel(3, 1), 20, 24)).toBe(3);
    expect(wheelDeltaToLines(wheel(2, 2), 20, 24)).toBe(48);
  });

  it("yields nothing when the delta or the cell height is unusable", () => {
    expect(wheelDeltaToLines(wheel(0), 20, 24)).toBe(0);
    expect(wheelDeltaToLines(wheel(NaN), 20, 24)).toBe(0);
    // An unmeasured screen must not silently become an enormous scroll.
    expect(wheelDeltaToLines(wheel(60), 0, 24)).toBe(0);
  });

  it("accumulates sub-line trackpad deltas instead of rounding them away", () => {
    const accumulator = createWheelLineAccumulator();
    // Each event alone is a third of a line, so on its own it would be lost.
    expect(accumulator.consume(wheel(6), 18, 24)).toBe(0);
    // Across the gesture the thirds still add up to exactly one line.
    let total = 0;
    for (let i = 0; i < 2; i += 1) total += accumulator.consume(wheel(6), 18, 24);
    expect(total).toBe(1);
  });

  it("costs at most half a line when the gesture reverses", () => {
    const accumulator = createWheelLineAccumulator();
    expect(accumulator.consume(wheel(12), 18, 24)).toBe(1);
    expect(accumulator.consume(wheel(-18), 18, 24)).toBe(-1);
  });

  it("maps total travel to total lines regardless of increment size", () => {
    // The defect this guards: rounding each increment on its own and dropping
    // the remainder means a gesture made of small steps scrolls nothing, while
    // the same distance in large steps scrolls fully.
    const coarse = createWheelLineAccumulator();
    const fine = createWheelLineAccumulator();
    let coarseLines = 0;
    let fineLines = 0;
    for (let i = 0; i < 6; i += 1) coarseLines += coarse.consume(wheel(30), 20, 24);
    for (let i = 0; i < 60; i += 1) fineLines += fine.consume(wheel(3), 20, 24);
    expect(coarseLines).toBe(9);
    expect(fineLines).toBe(9);
  });

  it("drops the remainder on reset", () => {
    const accumulator = createWheelLineAccumulator();
    // Banks a third of a line; without the reset the next third would carry it
    // past the halfway point and scroll.
    expect(accumulator.consume(wheel(6), 18, 24)).toBe(0);
    accumulator.reset();
    expect(accumulator.consume(wheel(6), 18, 24)).toBe(0);
  });
});
