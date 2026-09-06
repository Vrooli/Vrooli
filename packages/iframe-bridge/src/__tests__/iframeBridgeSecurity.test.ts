import { afterEach, describe, expect, it, vi } from 'vitest';
import { initIframeBridgeChild } from '../iframeBridgeChild.js';

describe('iframe bridge message admission', () => {
  const originalParent = window.parent;

  afterEach(() => {
    Object.defineProperty(window, 'parent', { configurable: true, value: originalParent });
    window.__vrooliBridgeChildInstalled = false;
    document.body.innerHTML = '';
  });

  it('rejects a same-origin message from a window other than the admitted parent', () => {
    const parent = { postMessage: vi.fn() };
    const hostile = { postMessage: vi.fn() };
    Object.defineProperty(window, 'parent', { configurable: true, value: parent });
    const pushState = vi.spyOn(history, 'pushState');
    const controller = initIframeBridgeChild({ parentOrigin: 'https://portal.example', appId: 'fixture' });

    window.dispatchEvent(new MessageEvent('message', {
      origin: 'https://portal.example',
      source: hostile as unknown as Window,
      data: { v: 1, t: 'NAV', cmd: 'GO', to: '/forged' },
    }));

    expect(pushState).not.toHaveBeenCalled();
    controller.dispose();
  });

  it('accepts a valid navigation message from the exact parent', () => {
    const parent = { postMessage: vi.fn() };
    Object.defineProperty(window, 'parent', { configurable: true, value: parent });
    const pushState = vi.spyOn(history, 'pushState');
    const controller = initIframeBridgeChild({ parentOrigin: 'https://portal.example', appId: 'fixture' });

    window.dispatchEvent(new MessageEvent('message', {
      origin: 'https://portal.example',
      source: parent as unknown as Window,
      data: { v: 1, t: 'NAV', cmd: 'GO', to: '/accepted' },
    }));

    expect(pushState).toHaveBeenCalledWith({}, '', '/accepted');
    controller.dispose();
  });
});
