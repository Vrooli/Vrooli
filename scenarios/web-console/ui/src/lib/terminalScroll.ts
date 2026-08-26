import type { Terminal } from "@xterm/xterm";
import { TOUCH_SCROLL_DECEL } from "../consts/config";
import { mouseWheelSequence } from "./terminalKeys";

export type ScrollSource = "touch" | "wheel" | "programmatic";

export interface TerminalScrollController {
  scrollBy: (lines: number, source: ScrollSource) => void;
  /** Notify the controller that a server output frame made progress. */
  notifyOutput: () => void;
  /** Number of control frames awaiting output progress. */
  getUnacknowledgedFrames: () => number;
  /** Deterministic seam for tests and teardown. */
  flush: () => void;
}

export interface TerminalScrollControllerOptions {
  getSensitivity?: (source: ScrollSource) => number;
  maxUnacknowledgedFrames?: number;
  maxFramesPerSecond?: number;
  /** Injectable monotonic clock for rate-limit tests. */
  now?: () => number;
  /** Watchdog duration for a missing output acknowledgement. */
  acknowledgementTimeoutMs?: number;
}

/** Convert a sampled touch displacement into the legacy 16 ms velocity unit. */
export function touchScrollVelocity(deltaY: number, elapsedMs: number): number {
  return elapsedMs > 0 ? (deltaY / elapsedMs) * 16 : 0;
}

/** Decay momentum by elapsed time so refresh rate does not change fling distance. */
export function decayTouchScrollVelocity(velocity: number, elapsedMs: number): number {
  return velocity * TOUCH_SCROLL_DECEL ** (Math.max(1, elapsedMs) / 16);
}

export function terminalIsInMouseTrackingMode(t: Terminal | null): boolean {
  if (!t) return false;
  const modes = (t as unknown as { modes?: { mouseTrackingMode?: string } }).modes;
  return modes?.mouseTrackingMode !== undefined && modes.mouseTrackingMode !== "none";
}

/** Apply a local viewport correction through the same terminal scroll seam. */
export function scrollTerminalLines(terminal: Terminal, lines: number): void {
  if (lines !== 0) terminal.scrollLines(lines);
}

export function createScrollController(
  getTerminal: () => Terminal | null,
  sendControl: (data: string) => boolean,
  options: TerminalScrollControllerOptions = {},
): TerminalScrollController {
  let pendingLines = 0;
  let frameHandle: number | null = null;
  let unacknowledgedFrames = 0;
  let lastFrameAt = Number.NEGATIVE_INFINITY;
  let acknowledgementWatchdog: ReturnType<typeof setTimeout> | null = null;
  let watchdogWarned = false;
  const maxUnacknowledgedFrames = options.maxUnacknowledgedFrames ?? 8;
  const maxFramesPerSecond = options.maxFramesPerSecond ?? 60;
  const acknowledgementTimeoutMs = options.acknowledgementTimeoutMs ?? 2000;
  const getSensitivity = options.getSensitivity ?? (() => 1);
  const now = options.now ?? (() => typeof performance !== "undefined" ? performance.now() : Date.now());

  const schedule = () => {
    if (frameHandle !== null) return;
    const run = () => {
      frameHandle = null;
      flush();
    };
    if (typeof requestAnimationFrame === "function") {
      frameHandle = requestAnimationFrame(run);
    } else {
      frameHandle = Number(setTimeout(run, 16));
    }
  };

  const armAcknowledgementWatchdog = () => {
    if (acknowledgementWatchdog !== null) return;
    acknowledgementWatchdog = setTimeout(() => {
      acknowledgementWatchdog = null;
      if (unacknowledgedFrames < maxUnacknowledgedFrames) return;
      unacknowledgedFrames = 0;
      if (!watchdogWarned) {
        watchdogWarned = true;
        console.warn("terminal scroll acknowledgements stalled; resetting the control-frame gate");
      }
      if (pendingLines !== 0) schedule();
    }, acknowledgementTimeoutMs);
  };

  function flush() {
    if (pendingLines === 0) return;
    const timestamp = now();
    if (unacknowledgedFrames >= maxUnacknowledgedFrames ||
        timestamp - lastFrameAt < 1000 / maxFramesPerSecond) {
      schedule();
      return;
    }
    const terminal = getTerminal();
    if (!terminal || !terminalIsInMouseTrackingMode(terminal)) {
      pendingLines = 0;
      return;
    }
    const lines = pendingLines;
    pendingLines = 0;
    const col = Math.floor(terminal.cols / 2);
    const row = Math.floor(terminal.rows / 2);
    const sequence = mouseWheelSequence(lines < 0, col, row).repeat(Math.abs(lines));
    if (sendControl(sequence)) {
      unacknowledgedFrames += 1;
      lastFrameAt = timestamp;
      if (unacknowledgedFrames >= maxUnacknowledgedFrames) {
        armAcknowledgementWatchdog();
      }
    }
    if (pendingLines !== 0) schedule();
  }

  return {
    scrollBy(lines, source) {
      const terminal = getTerminal();
      if (!terminal || lines === 0) return;
      const sensitivity = Math.max(0.01, getSensitivity(source));
      const adjusted = Math.round(lines * sensitivity);
      if (adjusted === 0) return;
      if (!terminalIsInMouseTrackingMode(terminal)) {
        scrollTerminalLines(terminal, adjusted);
        return;
      }
      pendingLines += adjusted;
      schedule();
    },
    notifyOutput() {
      unacknowledgedFrames = Math.max(0, unacknowledgedFrames - 1);
      if (unacknowledgedFrames < maxUnacknowledgedFrames) {
        watchdogWarned = false;
        if (acknowledgementWatchdog !== null) {
          clearTimeout(acknowledgementWatchdog);
          acknowledgementWatchdog = null;
        }
      }
    },
    getUnacknowledgedFrames() {
      return unacknowledgedFrames;
    },
    flush,
  };
}
