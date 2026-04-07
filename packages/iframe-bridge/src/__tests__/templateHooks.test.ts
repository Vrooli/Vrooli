import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { SpatialNavManager, type FocusGroupMode } from '../spatialNav.js';
import { GamepadInputManager, type GamepadAction } from '../gamepadInput.js';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function makeButton(
  id: string,
  rect: { top: number; left: number; width: number; height: number },
): HTMLButtonElement {
  const btn = document.createElement('button');
  btn.id = id;
  btn.textContent = id;
  btn.dataset.mockTop = String(rect.top);
  btn.dataset.mockLeft = String(rect.left);
  btn.dataset.mockWidth = String(rect.width);
  btn.dataset.mockHeight = String(rect.height);
  return btn;
}

function mockGetRect(el: HTMLElement): DOMRect {
  return {
    top: Number(el.dataset.mockTop ?? 0),
    left: Number(el.dataset.mockLeft ?? 0),
    width: Number(el.dataset.mockWidth ?? 0),
    height: Number(el.dataset.mockHeight ?? 0),
    right: Number(el.dataset.mockLeft ?? 0) + Number(el.dataset.mockWidth ?? 0),
    bottom: Number(el.dataset.mockTop ?? 0) + Number(el.dataset.mockHeight ?? 0),
    x: Number(el.dataset.mockLeft ?? 0),
    y: Number(el.dataset.mockTop ?? 0),
    toJSON: () => ({}),
  };
}

function makeGamepad(overrides?: {
  buttons?: Partial<Record<number, { pressed: boolean; touched: boolean; value: number }>>;
  axes?: number[];
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
    mapping: 'standard' as GamepadMappingType,
    axes: overrides?.axes ?? [0, 0, 0, 0],
    buttons,
    hapticActuators: [],
    vibrationActuator: null as unknown as GamepadHapticActuator,
  };
}

function flushRAF(): void {
  vi.advanceTimersByTime(16);
}

// ---------------------------------------------------------------------------
// Tests — Simulating React hook lifecycle patterns
// ---------------------------------------------------------------------------

describe('Template hook lifecycle patterns', () => {
  afterEach(() => {
    document.body.innerHTML = '';
    document.documentElement.removeAttribute('data-spatial-active');
  });

  describe('SpatialNavController lifecycle (init → registerGroup → dispose)', () => {
    it('init → registerGroup → dispose cleans up groups and mode', () => {
      const root = document.createElement('div');
      const group = document.createElement('div');
      const btn = makeButton('a', { top: 0, left: 0, width: 80, height: 40 });
      group.appendChild(btn);
      root.appendChild(group);
      document.body.appendChild(root);

      // Init (simulates useEffect mount)
      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      // Register group (simulates useFocusGroup hook)
      const disposeGroup = mgr.registerGroup(group, 'spatial');
      expect(group.getAttribute('data-spatial-group')).toBe('spatial');

      mgr.enterSpatialMode();
      expect(mgr.isActive()).toBe(true);
      expect(document.documentElement.hasAttribute('data-spatial-active')).toBe(true);

      // Dispose (simulates useEffect cleanup)
      disposeGroup();
      expect(group.hasAttribute('data-spatial-group')).toBe(false);

      mgr.dispose();
      expect(mgr.isActive()).toBe(false);
      expect(document.documentElement.hasAttribute('data-spatial-active')).toBe(false);
    });

    it('multiple registerGroup calls, then dispose all', () => {
      const root = document.createElement('div');
      const groups: HTMLDivElement[] = [];
      const disposers: (() => void)[] = [];

      for (let i = 0; i < 3; i++) {
        const group = document.createElement('div');
        const btn = makeButton(`btn-${i}`, {
          top: i * 100,
          left: 0,
          width: 80,
          height: 40,
        });
        group.appendChild(btn);
        root.appendChild(group);
        groups.push(group);
      }
      document.body.appendChild(root);

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      // Register all groups
      for (const group of groups) {
        disposers.push(mgr.registerGroup(group, 'spatial'));
      }

      // Verify all registered
      for (const group of groups) {
        expect(group.getAttribute('data-spatial-group')).toBe('spatial');
      }

      mgr.enterSpatialMode();
      expect(mgr.isActive()).toBe(true);

      // Dispose groups individually (simulates component unmounts)
      for (const dispose of disposers) {
        dispose();
      }

      // Verify all cleaned up
      for (const group of groups) {
        expect(group.hasAttribute('data-spatial-group')).toBe(false);
      }

      mgr.dispose();
    });

    it('re-entering spatial mode after exit restores focus', () => {
      const root = document.createElement('div');
      const a = makeButton('a', { top: 0, left: 0, width: 80, height: 40 });
      const b = makeButton('b', { top: 0, left: 200, width: 80, height: 40 });
      root.append(a, b);
      document.body.appendChild(root);

      const mgr = new SpatialNavManager({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
      });

      // First activation
      mgr.enterSpatialMode();
      expect(mgr.isActive()).toBe(true);
      expect(document.activeElement).toBe(a);

      // Navigate to b
      mgr.moveFocus('right');
      expect(document.activeElement).toBe(b);

      // Exit (e.g., mouse movement)
      mgr.exitSpatialMode();
      expect(mgr.isActive()).toBe(false);
      expect(document.documentElement.hasAttribute('data-spatial-active')).toBe(false);

      // Re-enter — should restore focus to previously focused element (b)
      mgr.enterSpatialMode();
      expect(mgr.isActive()).toBe(true);
      expect(document.activeElement).toBe(b);
      expect(b.getAttribute('data-spatial-focus')).toBe('true');

      mgr.dispose();
    });
  });

  describe('GamepadInputManager lifecycle (constructor → start → stop → dispose)', () => {
    beforeEach(() => {
      vi.useFakeTimers();
    });

    afterEach(() => {
      vi.useRealTimers();
    });

    it('constructor → start → stop → dispose lifecycle', () => {
      const actions: GamepadAction[] = [];
      let currentGamepad: Gamepad | null = null;

      const mgr = new GamepadInputManager({
        getGamepads: () => [currentGamepad],
        onAction: (a) => actions.push(a),
      });

      // Start polling
      currentGamepad = makeGamepad({
        buttons: { 0: { pressed: true, touched: true, value: 1 } },
      });
      mgr.start();
      window.dispatchEvent(new Event('gamepadconnected'));
      flushRAF();
      expect(actions).toEqual(['select']);

      // Stop polling — no new actions should fire
      mgr.stop();
      actions.length = 0;
      currentGamepad = makeGamepad({
        buttons: { 1: { pressed: true, touched: true, value: 1 } },
      });
      flushRAF();
      expect(actions).toEqual([]);

      // Resume — should pick up the new button
      mgr.start();
      flushRAF();
      expect(actions).toEqual(['back']);

      // Final dispose
      mgr.dispose();
      actions.length = 0;
      currentGamepad = makeGamepad({
        buttons: { 12: { pressed: true, touched: true, value: 1 } },
      });
      flushRAF();
      expect(actions).toEqual([]); // fully disposed, nothing fires
    });
  });
});
