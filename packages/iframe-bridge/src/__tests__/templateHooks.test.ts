import { createElement as h, StrictMode, act, useRef } from 'react';
import { createRoot, type Root } from 'react-dom/client';
import { beforeEach, afterEach, describe, expect, it, vi } from 'vitest';
import { initSpatialNav, type SpatialNavController, type GamepadActionHandler } from '../spatialNavBridge.js';
import { SpatialNavProvider, SpatialGroup, useGamepad, useSpatialNav, useSpatialScope } from '../react.js';

// Exercise real React effects and real input routing, rather than simulating hooks.
describe('shared React spatial adapters', () => {
  let reactRoot: Root;
  let controller: SpatialNavController;
  let host: HTMLDivElement;
  let pressed = false;
  let polls: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    vi.useFakeTimers();
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true });
    document.body.innerHTML = '';
    host = document.createElement('div');
    document.body.append(host);
    pressed = false;
    polls = vi.fn(() => [{
      index: 0, connected: true, mapping: 'standard', axes: [0, 0],
      buttons: [{ pressed, touched: pressed, value: Number(pressed) }],
    } as Gamepad]);
    controller = initSpatialNav({ getGamepads: polls, isVisible: () => true });
    window.dispatchEvent(new Event('gamepadconnected'));
    reactRoot = createRoot(host);
  });

  afterEach(() => {
    act(() => reactRoot.unmount());
    controller.dispose();
    vi.useRealTimers();
    Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: false });
  });

  function press() {
    act(() => { pressed = false; vi.advanceTimersByTime(16); });
    act(() => { pressed = true; vi.advanceTimersByTime(16); });
  }

  function Control({ handler, enabled = true }: { handler: GamepadActionHandler; enabled?: boolean }) {
    const ref = useRef<HTMLButtonElement>(null);
    useGamepad(ref, handler, enabled);
    expect(useSpatialNav()).toBe(controller);
    return h('button', { ref }, 'Custom control');
  }

  function render(handler: GamepadActionHandler, enabled = true) {
    act(() => reactRoot.render(h(StrictMode, null,
      h(SpatialNavProvider, { controller, children: h(Control, { handler, enabled }) }),
    )));
    host.querySelector('button')!.focus();
  }

  it('shares the application controller across StrictMode mounts and uses the committed callback', () => {
    const first = vi.fn(() => true);
    const second = vi.fn(() => true);
    render(first);
    polls.mockClear();
    press();
    expect(first).toHaveBeenCalledTimes(1);
    expect(polls).toHaveBeenCalledTimes(2);
    render(second);
    press();
    expect(first).toHaveBeenCalledTimes(1);
    expect(second).toHaveBeenCalledTimes(1);
    render(second, false);
    press();
    expect(second).toHaveBeenCalledTimes(1);
  });

  it('unregisters input on unmount without stopping application navigation', () => {
    const handler = vi.fn(() => true);
    render(handler);
    press();
    act(() => reactRoot.render(null));
    const button = document.createElement('button');
    const click = vi.fn();
    button.onclick = click;
    host.append(button);
    controller.enterSpatialMode();
    button.focus();
    press();
    expect(handler).toHaveBeenCalledTimes(1);
    expect(click).toHaveBeenCalledTimes(1);
    expect(initSpatialNav()).toBe(controller);
  });

  it('registers nested groups on mount and removes them on unmount', () => {
    act(() => reactRoot.render(h(SpatialNavProvider, { controller,
      children: h(SpatialGroup, { mode: 'passthrough' }, h('button', null, 'Canvas')),
    })));
    expect(host.querySelector('[data-spatial-group]')).not.toBeNull();
    act(() => reactRoot.render(null));
    expect(initSpatialNav()).toBe(controller);
  });

  it('traps modal navigation and restores trigger focus after cleanup', () => {
    const trigger = document.createElement('button');
    document.body.prepend(trigger);
    controller.enterSpatialMode();
    trigger.focus();
    function Dialog() {
      const ref = useRef<HTMLDivElement>(null);
      useSpatialScope(ref);
      return h('div', { ref }, h('button', null, 'Close'));
    }
    act(() => reactRoot.render(h(SpatialNavProvider, { controller, children: h(Dialog) })));
    expect(document.activeElement?.textContent).toBe('Close');
    act(() => reactRoot.render(null));
    expect(document.activeElement).toBe(trigger);
  });
});
