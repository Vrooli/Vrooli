/**
 * Gamepad input manager — polls the Gamepad API via requestAnimationFrame,
 * detects button-press edges, maps W3C Standard Gamepad buttons to semantic
 * navigation actions, and supports analog stick dead-zone filtering with
 * key-repeat for held directions.
 *
 * Zero runtime dependencies. Framework-agnostic.
 */

// ---------------------------------------------------------------------------
// Public types
// ---------------------------------------------------------------------------

export type GamepadAction =
  | 'navigate-up'
  | 'navigate-down'
  | 'navigate-left'
  | 'navigate-right'
  | 'select'
  | 'back'
  | 'menu'
  | 'page-next'
  | 'page-prev';

export interface GamepadInputOptions {
  /** Analog stick dead-zone threshold (0–1). Default `0.3`. */
  deadZone?: number;
  /** Milliseconds before the first key-repeat fires. Default `400`. */
  repeatInitialDelayMs?: number;
  /** Milliseconds between subsequent key-repeats. Default `100`. */
  repeatIntervalMs?: number;
  /** Called on every detected action (rising edge or repeat). */
  onAction?: (action: GamepadAction) => void;
  /**
   * Injectable seam — override `navigator.getGamepads()` for testing.
   * Must return the same shape as the native API.
   */
  getGamepads?: () => (Gamepad | null)[];
}

// ---------------------------------------------------------------------------
// W3C Standard Gamepad button → action mapping (by *position*, not label)
// ---------------------------------------------------------------------------

/** Map button index → GamepadAction. Only buttons we care about are listed. */
const BUTTON_ACTION_MAP: Readonly<Record<number, GamepadAction>> = {
  0: 'select',       // bottom face (A / Cross / B-on-Switch)
  1: 'back',         // right  face (B / Circle / A-on-Switch)
  4: 'page-prev',    // left bumper  (LB / L1)
  5: 'page-next',    // right bumper (RB / R1)
  9: 'menu',         // start / options / +
  12: 'navigate-up',
  13: 'navigate-down',
  14: 'navigate-left',
  15: 'navigate-right',
};

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

/** Convert stick axes to a directional action (or null if within dead zone). */
function axesToAction(
  axisX: number,
  axisY: number,
  deadZone: number,
): GamepadAction | null {
  const absX = Math.abs(axisX);
  const absY = Math.abs(axisY);

  // Neither axis exceeds dead zone → no action.
  if (absX < deadZone && absY < deadZone) return null;

  // Pick the dominant axis.
  if (absX >= absY) {
    return axisX < 0 ? 'navigate-left' : 'navigate-right';
  }
  return axisY < 0 ? 'navigate-up' : 'navigate-down';
}

// ---------------------------------------------------------------------------
// GamepadInputManager
// ---------------------------------------------------------------------------

export class GamepadInputManager {
  // Options (normalised)
  private readonly deadZone: number;
  private readonly repeatInitialDelayMs: number;
  private readonly repeatIntervalMs: number;
  private readonly getGamepads: () => (Gamepad | null)[];
  private readonly onAction: (action: GamepadAction) => void;

  // State
  private running = false;
  private rafId: number | null = null;
  private gamepadConnected = false;

  /**
   * Per-source (button index or `'stick'`) tracking for edge detection and
   * key-repeat timing.
   *
   * `active`   — whether the source was active last frame.
   * `action`   — the action that was last emitted for this source.
   * `pressTs`  — timestamp of the initial rising edge.
   * `lastEmit` — timestamp of the last emitted action (initial or repeat).
   */
  private sourceState = new Map<
    number | 'stick',
    { active: boolean; action: GamepadAction; pressTs: number; lastEmit: number }
  >();

  // Bound listeners (so we can remove them)
  private readonly onConnected: () => void;
  private readonly onDisconnected: () => void;

  constructor(options?: GamepadInputOptions) {
    this.deadZone = options?.deadZone ?? 0.3;
    this.repeatInitialDelayMs = options?.repeatInitialDelayMs ?? 400;
    this.repeatIntervalMs = options?.repeatIntervalMs ?? 100;
    this.onAction = options?.onAction ?? (() => { /* noop */ });
    this.getGamepads =
      options?.getGamepads ??
      (() => {
        if (typeof navigator !== 'undefined' && typeof navigator.getGamepads === 'function') {
          return navigator.getGamepads();
        }
        return [];
      });

    this.onConnected = () => {
      this.gamepadConnected = true;
    };
    this.onDisconnected = () => {
      // Check if any gamepads remain.
      const pads = this.getGamepads();
      this.gamepadConnected = pads.some((p) => p !== null);
      if (!this.gamepadConnected) {
        this.sourceState.clear();
      }
    };
  }

  // -----------------------------------------------------------------------
  // Lifecycle
  // -----------------------------------------------------------------------

  /** Start listening for gamepad input. Idempotent. */
  start(): void {
    if (this.running) return;
    this.running = true;

    if (typeof window !== 'undefined') {
      window.addEventListener('gamepadconnected', this.onConnected);
      window.addEventListener('gamepaddisconnected', this.onDisconnected);

      // A gamepad may already be connected (page loaded with controller).
      const pads = this.getGamepads();
      this.gamepadConnected = pads.some((p) => p !== null);
    }

    this.tick();
  }

  /** Pause polling. Call `start()` to resume. */
  stop(): void {
    this.running = false;
    if (this.rafId !== null) {
      cancelAnimationFrame(this.rafId);
      this.rafId = null;
    }
  }

  /** Stop polling and remove all event listeners. */
  dispose(): void {
    this.stop();
    if (typeof window !== 'undefined') {
      window.removeEventListener('gamepadconnected', this.onConnected);
      window.removeEventListener('gamepaddisconnected', this.onDisconnected);
    }
    this.sourceState.clear();
  }

  // -----------------------------------------------------------------------
  // Poll loop
  // -----------------------------------------------------------------------

  private tick = (): void => {
    if (!this.running) return;

    if (this.gamepadConnected) {
      this.poll();
    }

    this.rafId = requestAnimationFrame(this.tick);
  };

  private poll(): void {
    const gamepads = this.getGamepads();
    const now = performance.now();

    // Track which sources were seen this frame so we can release missing ones.
    const seenSources = new Set<number | 'stick'>();

    for (const pad of gamepads) {
      if (!pad) continue;
      // Only use the standard mapping; non-standard layouts are unreliable.
      if (pad.mapping !== 'standard') continue;

      // --- Buttons ---
      for (const [idxStr, action] of Object.entries(BUTTON_ACTION_MAP)) {
        const idx = Number(idxStr);
        const btn = pad.buttons[idx];
        if (!btn) continue;

        if (btn.pressed) {
          seenSources.add(idx);
          this.handleSourceActive(idx, action, now);
        }
      }

      // --- Left stick (axes 0 & 1) ---
      const stickAction = axesToAction(pad.axes[0] ?? 0, pad.axes[1] ?? 0, this.deadZone);
      if (stickAction) {
        seenSources.add('stick');
        this.handleSourceActive('stick', stickAction, now);
      }
    }

    // Release any source that was active last frame but not this frame.
    for (const [source, state] of this.sourceState) {
      if (state.active && !seenSources.has(source)) {
        state.active = false;
      }
    }
  }

  /**
   * Handle a source (button index or `'stick'`) being active this frame.
   * Manages rising-edge detection and key-repeat timing.
   */
  private handleSourceActive(
    source: number | 'stick',
    action: GamepadAction,
    now: number,
  ): void {
    const existing = this.sourceState.get(source);

    if (!existing || !existing.active || existing.action !== action) {
      // Rising edge (or action changed for stick).
      this.sourceState.set(source, {
        active: true,
        action,
        pressTs: now,
        lastEmit: now,
      });
      this.onAction(action);
      return;
    }

    // Source was already active with the same action — check key-repeat.
    const elapsed = now - existing.pressTs;
    if (elapsed < this.repeatInitialDelayMs) return;

    const sinceLast = now - existing.lastEmit;
    if (sinceLast >= this.repeatIntervalMs) {
      existing.lastEmit = now;
      this.onAction(action);
    }
  }
}
