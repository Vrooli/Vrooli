import { afterEach, describe, expect, it, vi } from 'vitest';
import { initIframeBridgeChild } from '../iframeBridgeChild.js';

describe('iframe bridge message admission', () => {
  const originalParent = window.parent;

  afterEach(() => {
    Object.defineProperty(window, 'parent', { configurable: true, value: originalParent });
    window.__vrooliBridgeChildInstalled = false;
    document.body.innerHTML = '';
  });

  it('rejects a same-origin message from a window other than the admitted parent', () => { // EMB-01
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

  it('ignores native shell requests from embedded content', () => { // EMB-02
    const parent = { postMessage: vi.fn() };
    Object.defineProperty(window, 'parent', { configurable: true, value: parent });
    const controller = initIframeBridgeChild({ parentOrigin: 'https://portal.example', appId: 'fixture' });

    window.dispatchEvent(new MessageEvent('message', {
      origin: 'https://portal.example',
      source: parent as unknown as Window,
      data: { v: 1, t: 'NATIVE_SHELL_CALL', method: 'readFile', args: ['/etc/passwd'] },
    }));

    expect(parent.postMessage).not.toHaveBeenCalledWith(expect.objectContaining({ t: 'NATIVE_RESULT' }), expect.anything());
    controller.dispose();
  });

  it('rejects oversized control payloads without applying navigation', () => { // EMB-03
    const parent = { postMessage: vi.fn() };
    Object.defineProperty(window, 'parent', { configurable: true, value: parent });
    const pushState = vi.spyOn(history, 'pushState');
    const controller = initIframeBridgeChild({ parentOrigin: 'https://portal.example', appId: 'fixture' });

    window.dispatchEvent(new MessageEvent('message', {
      origin: 'https://portal.example',
      source: parent as unknown as Window,
      data: { v: 1, t: 'NAV', cmd: 'GO', to: '/accepted', padding: 'x'.repeat(70_000) },
    }));

    expect(pushState).not.toHaveBeenCalled();
    controller.dispose();
  });

  it('rejects a message after the admitted parent reference changes', () => { // EMB-04
    const parent = { postMessage: vi.fn() };
    const replacement = { postMessage: vi.fn() };
    Object.defineProperty(window, 'parent', { configurable: true, value: parent });
    const pushState = vi.spyOn(history, 'pushState');
    const controller = initIframeBridgeChild({ parentOrigin: 'https://portal.example', appId: 'fixture' });

    // A closed/replaced surface must not accept a delayed message from the
    // old session. The exact parent check is the session identity fence.
    Object.defineProperty(window, 'parent', { configurable: true, value: replacement });
    window.dispatchEvent(new MessageEvent('message', {
      origin: 'https://portal.example',
      source: parent as unknown as Window,
      data: { v: 1, t: 'NAV', cmd: 'GO', to: '/stale' },
    }));

    expect(pushState).not.toHaveBeenCalled();
    controller.dispose();
  });
});
