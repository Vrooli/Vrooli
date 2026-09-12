import type { Terminal } from "@xterm/xterm";
import { TOUCH_SCROLL_DECEL } from "../consts/config";
import { mouseWheelSequence } from "./terminalKeys";

export type ScrollSource = "touch" | "wheel" | "programmatic";

/** How a scroll request reaches the thing that actually holds the history. */
export type ScrollTransport = "local" | "mouse-report" | "server-scroll";

export interface LineAccumulator {
  /** Returns whole lines to scroll, carrying the sub-line remainder forward. */
  consume: (lines: number) => number;
  reset: () => void;
}

export interface WheelLineAccumulator {
  /** Returns whole lines to scroll, carrying the sub-line remainder forward. */
  consume: (event: WheelEvent, cellHeight: number, rows: number) => number;
  reset: () => void;
}

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
  /**
   * Ask the server to scroll the backend's own history. Required for the
   * `server-scroll` transport: when the pane is a tmux client there is no
   * client-side scrollback to move and no application listening for mouse
   * reports, so the only real history lives in the tmux pane.
   */
  sendScroll?: (lines: number) => boolean;
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

/**
 * Reports whether `terminal.scrollLines` can actually move anything.
 *
 * The alternate screen buffer never has scrollback, so a `scrollLines` call
 * against it is a silent no-op. This matters far more than it looks: a tmux
 * client emits `\x1b[?1049h` the instant it attaches, so EVERY tmux-backed
 * pane keeps xterm in the alternate buffer for the whole life of the session,
 * regardless of what the program inside the pane does. Treating "no mouse
 * tracking" as "local scrollback is in control" is therefore wrong for every
 * persistent pane we serve.
 *
 * A terminal that does not expose buffer identity (a test double) is treated
 * as having local scrollback, which is the conservative default: it keeps
 * scrolling client-side instead of speculatively talking to the server.
 */
export function terminalHasLocalScrollback(t: Terminal | null): boolean {
  if (!t) return false;
  const buffer = (t as unknown as { buffer?: { active?: unknown; alternate?: unknown } }).buffer;
  if (!buffer || buffer.active === undefined || buffer.alternate === undefined) return true;
  return buffer.active !== buffer.alternate;
}

/**
 * Which mechanism can actually move this terminal's view.
 *
 * - `mouse-report`: the program requested mouse tracking, so it owns the
 *   wheel and scrolls itself (Claude Code, vim, or tmux with `mouse on`).
 * - `local`: xterm holds real scrollback and can scroll without the network.
 * - `server-scroll`: neither is true — the only history lives in the backend
 *   (a tmux pane), so the server has to do the scrolling.
 */
export function scrollTransportFor(t: Terminal | null): ScrollTransport | null {
  if (!t) return null;
  if (terminalIsInMouseTrackingMode(t)) return "mouse-report";
  return terminalHasLocalScrollback(t) ? "local" : "server-scroll";
}

/** Apply a local viewport correction through the same terminal scroll seam. */
export function scrollTerminalLines(terminal: Terminal, lines: number): void {
  if (lines !== 0) terminal.scrollLines(lines);
}

/** Pixel height of one terminal row, or 0 when the screen is not measurable. */
export function terminalCellHeight(terminal: Terminal, container: HTMLElement): number {
  const screenEl = container.querySelector<HTMLElement>(".xterm-screen");
  if (!screenEl || terminal.rows <= 0) return 0;
  return screenEl.getBoundingClientRect().height / terminal.rows;
}

/**
 * Convert one wheel event into terminal lines, honouring `deltaMode`.
 *
 * Browsers report wheel deltas in pixels, lines, or pages depending on the
 * device and platform; a trackpad reports sub-line pixel deltas that would
 * each round to zero if converted independently.
 */
export function wheelDeltaToLines(event: WheelEvent, cellHeight: number, rows: number): number {
  const delta = event.deltaY;
  if (!Number.isFinite(delta) || delta === 0) return 0;
  switch (event.deltaMode) {
    case 1: // DOM_DELTA_LINE
      return delta;
    case 2: // DOM_DELTA_PAGE
      return delta * Math.max(1, rows);
    default: // DOM_DELTA_PIXEL
      return cellHeight > 0 ? delta / cellHeight : 0;
  }
}

/**
 * Carries the sub-line remainder between scroll increments.
 *
 * Every scroll source produces fractional lines — a trackpad notch, a pixel of
 * finger travel, a sensitivity multiplier below 1 — while the terminal can
 * only scroll whole rows. Rounding each increment independently and dropping
 * the remainder does not merely lose precision, it loses the entire gesture:
 * a slow drag whose every sample rounds to zero scrolls nothing at all, however
 * far the finger travels. Carrying the remainder makes total travel map to
 * total lines, so slow and fast gestures cover the same distance.
 */
export function createLineAccumulator(): LineAccumulator {
  let remainder = 0;
  return {
    consume(lines) {
      if (!Number.isFinite(lines) || lines === 0) return 0;
      const total = remainder + lines;
      const whole = Math.round(total);
      // Bounded to [-0.5, 0.5), so a direction change costs at most half a row.
      remainder = total - whole;
      return whole;
    },
    reset() {
      remainder = 0;
    },
  };
}

/** Wheel events, converted to lines and accumulated through the shared seam. */
export function createWheelLineAccumulator(): WheelLineAccumulator {
  const accumulator = createLineAccumulator();
  return {
    consume: (event, cellHeight, rows) =>
      accumulator.consume(wheelDeltaToLines(event, cellHeight, rows)),
    reset: accumulator.reset,
  };
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
  const sendScroll = options.sendScroll;
  // One remainder per source: a wheel notch and a finger drag are different
  // gestures and must not consume each other's carried fraction.
  const sensitivityRemainders = new Map<ScrollSource, LineAccumulator>();
  const sensitivityAccumulators = (source: ScrollSource): LineAccumulator => {
    let accumulator = sensitivityRemainders.get(source);
    if (!accumulator) {
      accumulator = createLineAccumulator();
      sensitivityRemainders.set(source, accumulator);
    }
    return accumulator;
  };
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
    const transport = scrollTransportFor(terminal);
    // `local` can appear here when the program left the alternate buffer
    // between queueing and this frame; those lines belong to a view that no
    // longer exists, so drop them rather than replay them somewhere else.
    if (!terminal || transport === null || transport === "local") {
      pendingLines = 0;
      return;
    }
    const lines = pendingLines;
    pendingLines = 0;
    let delivered: boolean;
    if (transport === "mouse-report") {
      const col = Math.floor(terminal.cols / 2);
      const row = Math.floor(terminal.rows / 2);
      const sequence = mouseWheelSequence(lines < 0, col, row).repeat(Math.abs(lines));
      delivered = sendControl(sequence);
    } else {
      delivered = sendScroll?.(lines) ?? false;
    }
    if (delivered) {
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
      // Accumulate the scaled value rather than rounding it away. At a
      // sensitivity below 0.5 every single-line scroll used to round to zero
      // and vanish, so the setting did not slow scrolling down — it turned it
      // off entirely.
      const adjusted = sensitivityAccumulators(source).consume(lines * sensitivity);
      if (adjusted === 0) return;
      if (scrollTransportFor(terminal) === "local") {
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
