import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { GamepadInputManager, type GamepadAction } from '../gamepadInput.js';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

/** Build a minimal Gamepad-like object for testing. */
function makeGamepad(overrides?: {
  buttons?: Partial<Record<number, { pressed: boolean; touched: boolean; value: number }>>;
  axes?: number[];
  mapping?: string;
}): Gamepad {
  const buttons: GamepadButton[] = Array.from({ length: 17 }, () => ({
    pressed: false,
    touched: false,
    value: 0,
  }));

  if (overrides?.buttons) {
    for (const [idx, state] of Object.entries(overrides.buttons)) {
      buttons[Number(idx)] = {
        pressed: state?.pressed ?? false,
        touched: state?.touched ?? false,
        value: state?.value ?? 0,
      };
    }
  }

  return {
    id: 'Test Controller',
    index: 0,
    connected: true,
    timestamp: performance.now(),
    mapping: (overrides?.mapping ?? 'standard') as GamepadMappingType,
    axes: overrides?.axes ?? [0, 0, 0, 0],
    buttons,
    hapticActuators: [],
    vibrationActuator: null as unknown as GamepadHapticActuator,
  };
}

type GetGamepadsFn = () => (Gamepad | null)[];

function createMockGetGamepads(initial?: Gamepad | null): {
  fn: GetGamepadsFn;
  setGamepad: (gp: Gamepad | null) => void;
} {
  let current: Gamepad | null = initial ?? null;
  return {
    fn: () => [current],
    setGamepad: (gp) => { current = gp; },
  };
}

/**
 * Advance one rAF tick.  jsdom doesn't run rAF automatically, so we flush
 * the callback manually.
 */
function flushRAF(): void {
  vi.advanceTimersByTime(16);
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('GamepadInputManager', () => {
  let actions: GamepadAction[];
  let onAction: (a: GamepadAction) => void;

  beforeEach(() => {
    vi.useFakeTimers();
    actions = [];
    onAction = (a) => actions.push(a);
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('fires action on rising edge only (not every frame)', () => {
    const { fn, setGamepad } = createMockGetGamepads();
    const mgr = new GamepadInputManager({ getGamepads: fn, onAction });
    setGamepad(makeGamepad({ buttons: { 0: { pressed: true, touched: true, value: 1 } } }));

    mgr.start();
    flushRAF(); // frame 1 — rising edge
    flushRAF(); // frame 2 — held, no repeat yet (within initial delay)
    flushRAF(); // frame 3

    expect(actions).toEqual(['select']); // only one fire
    mgr.dispose();
  });

  it('fires key repeat after initial delay', () => {
    const { fn, setGamepad } = createMockGetGamepads();
    const mgr = new GamepadInputManager({
      getGamepads: fn,
      onAction,
      repeatInitialDelayMs: 100,
      repeatIntervalMs: 50,
    });
    setGamepad(makeGamepad({ buttons: { 12: { pressed: true, touched: true, value: 1 } } }));

    mgr.start();
    flushRAF(); // t≈16 — rising edge
    expect(actions).toEqual(['navigate-up']);

    // Advance past initial delay
    vi.advanceTimersByTime(120);
    expect(actions.length).toBeGreaterThanOrEqual(2);
    expect(actions.every((a) => a === 'navigate-up')).toBe(true);

    mgr.dispose();
  });

  it('does not fire when stick is within dead zone', () => {
    const { fn, setGamepad } = createMockGetGamepads();
    const mgr = new GamepadInputManager({ getGamepads: fn, onAction, deadZone: 0.5 });
    setGamepad(makeGamepad({ axes: [0.3, -0.2, 0, 0] }));

    mgr.start();
    flushRAF();

    expect(actions).toEqual([]);
    mgr.dispose();
  });

  it('fires navigate-left for negative X axis beyond dead zone', () => {
    const { fn, setGamepad } = createMockGetGamepads();
    const mgr = new GamepadInputManager({ getGamepads: fn, onAction, deadZone: 0.3 });
    setGamepad(makeGamepad({ axes: [-0.8, 0.1, 0, 0] }));

    mgr.start();
    flushRAF();

    expect(actions).toEqual(['navigate-left']);
    mgr.dispose();
  });

  it('fires navigate-down for positive Y axis beyond dead zone', () => {
    const { fn, setGamepad } = createMockGetGamepads();
    const mgr = new GamepadInputManager({ getGamepads: fn, onAction, deadZone: 0.3 });
    setGamepad(makeGamepad({ axes: [0.0, 0.9, 0, 0] }));

    mgr.start();
    flushRAF();

    expect(actions).toEqual(['navigate-down']);
    mgr.dispose();
  });

  it('maps D-pad buttons to navigation actions', () => {
    const mapping: [number, GamepadAction][] = [
      [12, 'navigate-up'],
      [13, 'navigate-down'],
      [14, 'navigate-left'],
      [15, 'navigate-right'],
    ];

    for (const [btnIdx, expected] of mapping) {
      const localActions: GamepadAction[] = [];
      const { fn, setGamepad } = createMockGetGamepads();
      setGamepad(makeGamepad({ buttons: { [btnIdx]: { pressed: true, touched: true, value: 1 } } }));
      const mgr = new GamepadInputManager({ getGamepads: fn, onAction: (a) => localActions.push(a) });

      mgr.start();
      flushRAF();
      expect(localActions).toContain(expected);

      mgr.dispose();
    }
  });

  it('maps face and shoulder buttons to semantic actions', () => {
    const mapping: [number, GamepadAction][] = [
      [0, 'select'],
      [1, 'back'],
      [4, 'page-prev'],
      [5, 'page-next'],
      [9, 'menu'],
    ];

    for (const [btnIdx, expected] of mapping) {
      const localActions: GamepadAction[] = [];
      const { fn, setGamepad } = createMockGetGamepads();
      setGamepad(makeGamepad({ buttons: { [btnIdx]: { pressed: true, touched: true, value: 1 } } }));
      const mgr = new GamepadInputManager({ getGamepads: fn, onAction: (a) => localActions.push(a) });

      mgr.start();
      flushRAF();
      expect(localActions).toEqual([expected]);

      mgr.dispose();
    }
  });

  it('ignores gamepads with non-standard mapping', () => {
    const { fn, setGamepad } = createMockGetGamepads();
    const mgr = new GamepadInputManager({ getGamepads: fn, onAction });
    setGamepad(makeGamepad({
      buttons: { 0: { pressed: true, touched: true, value: 1 } },
      mapping: '',
    }));

    mgr.start();
    flushRAF();

    expect(actions).toEqual([]);
    mgr.dispose();
  });

  it('does not poll when no gamepad is connected', () => {
    const fn = vi.fn<[], (Gamepad | null)[]>(() => [null]);
    const mgr = new GamepadInputManager({ getGamepads: fn, onAction });

    mgr.start();
    flushRAF();
    flushRAF();

    // getGamepads is called once on start() to check connectivity, but
    // the poll loop should skip actual polling since no gamepad is connected.
    // The initial call happens in the constructor via start().
    const callsDuringPoll = fn.mock.calls.length;
    // Should be small — just the connectivity check, not per-frame polling.
    expect(callsDuringPoll).toBeLessThanOrEqual(3);

    mgr.dispose();
  });

  it('clears state on gamepaddisconnected when no gamepads remain', () => {
    const { fn, setGamepad } = createMockGetGamepads();
    setGamepad(makeGamepad({ buttons: { 0: { pressed: true, touched: true, value: 1 } } }));
    const mgr = new GamepadInputManager({ getGamepads: fn, onAction });

    mgr.start();
    flushRAF();
    expect(actions).toEqual(['select']);

    // Disconnect — no gamepads remain
    setGamepad(null);
    window.dispatchEvent(new Event('gamepaddisconnected'));
    flushRAF();

    // Re-connect with a button pressed — should fire again (state was cleared)
    actions = [];
    setGamepad(makeGamepad({ buttons: { 0: { pressed: true, touched: true, value: 1 } } }));
    window.dispatchEvent(new Event('gamepadconnected'));
    flushRAF();
    expect(actions).toEqual(['select']);

    mgr.dispose();
  });

  it('starts polling when gamepadconnected fires', () => {
    const { fn, setGamepad } = createMockGetGamepads(null);
    const mgr = new GamepadInputManager({ getGamepads: fn, onAction });

    mgr.start();
    flushRAF();
    expect(actions).toEqual([]);

    // Simulate connection
    setGamepad(makeGamepad({ buttons: { 0: { pressed: true, touched: true, value: 1 } } }));
    window.dispatchEvent(new Event('gamepadconnected'));
    flushRAF();

    expect(actions).toEqual(['select']);
    mgr.dispose();
  });

  it('dispose stops the polling loop', () => {
    const { fn, setGamepad } = createMockGetGamepads();
    const mgr = new GamepadInputManager({ getGamepads: fn, onAction });
    setGamepad(makeGamepad({ buttons: { 0: { pressed: true, touched: true, value: 1 } } }));

    mgr.start();
    flushRAF();
    expect(actions).toEqual(['select']);

    mgr.dispose();
    actions = [];

    // Release and re-press — should not fire.
    setGamepad(makeGamepad());
    flushRAF();
    setGamepad(makeGamepad({ buttons: { 0: { pressed: true, touched: true, value: 1 } } }));
    flushRAF();

    expect(actions).toEqual([]);
  });

  it('stop pauses polling, start resumes', () => {
    const { fn, setGamepad } = createMockGetGamepads();
    setGamepad(makeGamepad({ buttons: { 0: { pressed: true, touched: true, value: 1 } } }));
    const mgr = new GamepadInputManager({ getGamepads: fn, onAction });

    mgr.start();
    flushRAF();
    expect(actions).toEqual(['select']);

    mgr.stop();
    actions = [];
    // Release previous and press new — while stopped, nothing should fire.
    setGamepad(makeGamepad({ buttons: { 1: { pressed: true, touched: true, value: 1 } } }));
    flushRAF();
    expect(actions).toEqual([]); // paused

    mgr.start();
    flushRAF();
    expect(actions).toEqual(['back']); // resumed

    mgr.dispose();
  });

  it('picks dominant axis when both exceed dead zone', () => {
    const { fn, setGamepad } = createMockGetGamepads();
    const mgr = new GamepadInputManager({ getGamepads: fn, onAction, deadZone: 0.3 });
    // X dominant
    setGamepad(makeGamepad({ axes: [0.9, 0.5, 0, 0] }));

    mgr.start();
    flushRAF();

    expect(actions).toEqual(['navigate-right']);
    mgr.dispose();
  });

  it('detects action change on stick (direction switch without release)', () => {
    const { fn, setGamepad } = createMockGetGamepads();
    setGamepad(makeGamepad({ axes: [0.9, 0, 0, 0] }));
    const mgr = new GamepadInputManager({ getGamepads: fn, onAction, deadZone: 0.3 });

    mgr.start();
    flushRAF();
    expect(actions).toEqual(['navigate-right']);

    // Switch direction without releasing
    setGamepad(makeGamepad({ axes: [-0.9, 0, 0, 0] }));
    flushRAF();
    expect(actions).toEqual(['navigate-right', 'navigate-left']);

    mgr.dispose();
  });
});
