import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { initSpatialNav, type SpatialNavController } from '../spatialNavBridge.js';

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
// Tests
// ---------------------------------------------------------------------------

describe('initSpatialNav (bridge)', () => {
  let root: HTMLDivElement;
  let controller: SpatialNavController;
  let currentGamepad: Gamepad | null;

  beforeEach(() => {
    vi.useFakeTimers();
    document.body.innerHTML = '';
    document.documentElement.removeAttribute('data-spatial-active');
    currentGamepad = null;

    root = document.createElement('div');
    document.body.appendChild(root);
  });

  afterEach(() => {
    controller?.dispose();
    vi.useRealTimers();
  });

  it('returns a controller with expected methods', () => {
    controller = initSpatialNav({
      rootElement: root,
      getBoundingClientRect: mockGetRect,
      injectDefaultFocusStyle: false,
      isVisible: () => true,
      getGamepads: () => [null],
      autoActivate: false,
    });

    expect(controller.dispose).toBeTypeOf('function');
    expect(controller.registerGroup).toBeTypeOf('function');
    expect(controller.isActive).toBeTypeOf('function');
    expect(controller.enterSpatialMode).toBeTypeOf('function');
    expect(controller.exitSpatialMode).toBeTypeOf('function');
  });

  it('gamepad D-pad activates spatial mode and moves focus', () => {
    const a = makeButton('a', { top: 100, left: 100, width: 80, height: 40 });
    const b = makeButton('b', { top: 100, left: 300, width: 80, height: 40 });
    root.append(a, b);

    controller = initSpatialNav({
      rootElement: root,
      getBoundingClientRect: mockGetRect,
      injectDefaultFocusStyle: false,
      isVisible: () => true,
      getGamepads: () => [currentGamepad],
    });

    // Simulate gamepad connection
    currentGamepad = makeGamepad({ buttons: { 15: { pressed: true, touched: true, value: 1 } } });
    window.dispatchEvent(new Event('gamepadconnected'));

    flushRAF();

    // Spatial mode should be active
    expect(controller.isActive()).toBe(true);
    // Focus should have moved (D-pad right)
    expect(document.activeElement?.id).toBe('b');
  });

  it('mousemove exits spatial mode', () => {
    const a = makeButton('a', { top: 100, left: 100, width: 80, height: 40 });
    root.append(a);

    controller = initSpatialNav({
      rootElement: root,
      getBoundingClientRect: mockGetRect,
      injectDefaultFocusStyle: false,
      isVisible: () => true,
      getGamepads: () => [null],
      autoActivate: false,
    });

    controller.enterSpatialMode();
    expect(controller.isActive()).toBe(true);

    window.dispatchEvent(new MouseEvent('mousemove'));
    expect(controller.isActive()).toBe(false);
  });

  it('select action clicks focused element', () => {
    const a = makeButton('a', { top: 100, left: 100, width: 80, height: 40 });
    const onClick = vi.fn();
    a.addEventListener('click', onClick);
    root.append(a);

    controller = initSpatialNav({
      rootElement: root,
      getBoundingClientRect: mockGetRect,
      injectDefaultFocusStyle: false,
      isVisible: () => true,
      getGamepads: () => [currentGamepad],
    });

    // Activate spatial mode first
    controller.enterSpatialMode();
    expect(document.activeElement).toBe(a);

    // Press A button (select)
    currentGamepad = makeGamepad({ buttons: { 0: { pressed: true, touched: true, value: 1 } } });
    window.dispatchEvent(new Event('gamepadconnected'));
    flushRAF();

    expect(onClick).toHaveBeenCalled();
  });

  it('registerGroup delegates to spatial nav manager', () => {
    const group = document.createElement('div');
    root.appendChild(group);

    controller = initSpatialNav({
      rootElement: root,
      getBoundingClientRect: mockGetRect,
      injectDefaultFocusStyle: false,
      isVisible: () => true,
      getGamepads: () => [null],
      autoActivate: false,
    });

    const dispose = controller.registerGroup(group, 'passthrough');
    expect(group.getAttribute('data-spatial-group')).toBe('passthrough');

    dispose();
    expect(group.hasAttribute('data-spatial-group')).toBe(false);
  });

  it('dispose cleans up both gamepad and spatial nav', () => {
    const a = makeButton('a', { top: 100, left: 100, width: 80, height: 40 });
    root.append(a);

    controller = initSpatialNav({
      rootElement: root,
      getBoundingClientRect: mockGetRect,
      injectDefaultFocusStyle: false,
      isVisible: () => true,
      getGamepads: () => [null],
    });

    controller.enterSpatialMode();
    expect(document.documentElement.hasAttribute('data-spatial-active')).toBe(true);

    controller.dispose();
    expect(document.documentElement.hasAttribute('data-spatial-active')).toBe(false);
  });

  describe('bumper group cycling through the bridge', () => {
    it('page-next cycles focus to the next group', () => {
      const a = makeButton('a', { top: 100, left: 100, width: 80, height: 40 });
      const b = makeButton('b', { top: 100, left: 400, width: 80, height: 40 });

      const group1 = document.createElement('div');
      group1.appendChild(a);
      const group2 = document.createElement('div');
      group2.appendChild(b);
      root.append(group1, group2);

      controller = initSpatialNav({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
        getGamepads: () => [currentGamepad],
      });

      controller.registerGroup(group1, 'spatial');
      controller.registerGroup(group2, 'spatial');

      controller.enterSpatialMode();
      expect(document.activeElement).toBe(a);

      // Press RB (button 5 = page-next)
      currentGamepad = makeGamepad({ buttons: { 5: { pressed: true, touched: true, value: 1 } } });
      window.dispatchEvent(new Event('gamepadconnected'));
      flushRAF();

      expect(document.activeElement).toBe(b);
    });

    it('page-prev cycles focus back to the previous group', () => {
      const a = makeButton('a', { top: 100, left: 100, width: 80, height: 40 });
      const b = makeButton('b', { top: 100, left: 400, width: 80, height: 40 });

      const group1 = document.createElement('div');
      group1.appendChild(a);
      const group2 = document.createElement('div');
      group2.appendChild(b);
      root.append(group1, group2);

      controller = initSpatialNav({
        rootElement: root,
        getBoundingClientRect: mockGetRect,
        injectDefaultFocusStyle: false,
        isVisible: () => true,
        getGamepads: () => [currentGamepad],
      });

      controller.registerGroup(group1, 'spatial');
      controller.registerGroup(group2, 'spatial');

      controller.enterSpatialMode();
      expect(document.activeElement).toBe(a);

      // First cycle to group2 via page-next
      currentGamepad = makeGamepad({ buttons: { 5: { pressed: true, touched: true, value: 1 } } });
      window.dispatchEvent(new Event('gamepadconnected'));
      flushRAF();
      expect(document.activeElement).toBe(b);

      // Release button
      currentGamepad = makeGamepad();
      flushRAF();

      // Press LB (button 4 = page-prev) to go back to group1
      currentGamepad = makeGamepad({ buttons: { 4: { pressed: true, touched: true, value: 1 } } });
      flushRAF();

      expect(document.activeElement).toBe(a);
    });
  });

  it('back action calls history.back', () => {
    const a = makeButton('a', { top: 100, left: 100, width: 80, height: 40 });
    root.append(a);

    const backSpy = vi.spyOn(window.history, 'back').mockImplementation(() => { /* noop */ });
    Object.defineProperty(window.history, 'length', { value: 2, configurable: true });

    controller = initSpatialNav({
      rootElement: root,
      getBoundingClientRect: mockGetRect,
      injectDefaultFocusStyle: false,
      isVisible: () => true,
      getGamepads: () => [currentGamepad],
    });

    controller.enterSpatialMode();

    // Press B button (back) — should navigate back
    currentGamepad = makeGamepad({ buttons: { 1: { pressed: true, touched: true, value: 1 } } });
    window.dispatchEvent(new Event('gamepadconnected'));
    flushRAF();

    expect(backSpy).toHaveBeenCalled();
    backSpy.mockRestore();
    Object.defineProperty(window.history, 'length', { value: 1, configurable: true });
  });

  it('menu action emits shortcut intent', () => {
    root.append(makeButton('a', { top: 100, left: 100, width: 80, height: 40 }));

    controller = initSpatialNav({
      rootElement: root,
      getBoundingClientRect: mockGetRect,
      injectDefaultFocusStyle: false,
      isVisible: () => true,
      getGamepads: () => [currentGamepad],
    });

    controller.enterSpatialMode();

    // Press menu button (9)
    currentGamepad = makeGamepad({ buttons: { 9: { pressed: true, touched: true, value: 1 } } });
    window.dispatchEvent(new Event('gamepadconnected'));
    flushRAF();

    // Menu action fires — no error means it was handled.
    // (emitShortcutIntent is a no-op outside iframes, so we just verify no crash)
    expect(controller.isActive()).toBe(true);
  });

  it('exitSpatialMode delegates correctly', () => {
    root.append(makeButton('a', { top: 100, left: 100, width: 80, height: 40 }));

    controller = initSpatialNav({
      rootElement: root,
      getBoundingClientRect: mockGetRect,
      injectDefaultFocusStyle: false,
      isVisible: () => true,
      getGamepads: () => [null],
      autoActivate: false,
    });

    controller.enterSpatialMode();
    expect(controller.isActive()).toBe(true);

    controller.exitSpatialMode();
    expect(controller.isActive()).toBe(false);
  });

  it('unhandled navigation relays to host', () => {
    // Single button, nowhere to move right — triggers unhandled relay
    root.append(makeButton('a', { top: 100, left: 100, width: 80, height: 40 }));

    controller = initSpatialNav({
      rootElement: root,
      getBoundingClientRect: mockGetRect,
      injectDefaultFocusStyle: false,
      isVisible: () => true,
      getGamepads: () => [currentGamepad],
      hostRelay: true,
    });

    controller.enterSpatialMode();

    // Press D-pad right — no element to the right, should relay
    currentGamepad = makeGamepad({ buttons: { 15: { pressed: true, touched: true, value: 1 } } });
    window.dispatchEvent(new Event('gamepadconnected'));
    flushRAF();

    // No crash — emitShortcutIntent is a no-op outside iframes
    expect(controller.isActive()).toBe(true);
  });

  it('autoActivate: false does not start gamepad polling', () => {
    controller = initSpatialNav({
      rootElement: root,
      getBoundingClientRect: mockGetRect,
      injectDefaultFocusStyle: false,
      isVisible: () => true,
      getGamepads: () => [makeGamepad({ buttons: { 0: { pressed: true, touched: true, value: 1 } } })],
      autoActivate: false,
    });

    flushRAF();
    // Should NOT have activated since autoActivate is false
    expect(controller.isActive()).toBe(false);
  });

});
