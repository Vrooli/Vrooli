import { describe, expect, it, vi } from "vitest";
import type { Terminal } from "@xterm/xterm";
import { createScrollController } from "./terminalScroll";

function terminal(mouseTrackingMode: string) {
  return {
    cols: 80,
    rows: 24,
    modes: { mouseTrackingMode },
    scrollLines: vi.fn(),
  } as unknown as Terminal & { scrollLines: ReturnType<typeof vi.fn> };
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

  it("ignores zero movement and sensitivity that rounds to zero", () => {
    const term = terminal("none");
    const controller = createScrollController(() => term, vi.fn(), { getSensitivity: () => 0.01 });
    controller.scrollBy(0, "wheel");
    controller.scrollBy(1, "wheel");
    expect(term.scrollLines).not.toHaveBeenCalled();
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
