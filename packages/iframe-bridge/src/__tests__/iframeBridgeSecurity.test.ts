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

  it('rejects owner log requests from a different window on the admitted origin', () => { // EMB-05
    const parent = { postMessage: vi.fn() };
    const sameOriginPeer = { postMessage: vi.fn() };
    Object.defineProperty(window, 'parent', { configurable: true, value: parent });
    const controller = initIframeBridgeChild({
      parentOrigin: 'https://portal.example',
      appId: 'fixture',
      captureLogs: true,
    });
    parent.postMessage.mockClear();

    window.dispatchEvent(new MessageEvent('message', {
      origin: 'https://portal.example',
      source: sameOriginPeer as unknown as Window,
      data: { v: 1, t: 'LOGS', cmd: 'PULL', requestId: 'forged-request' },
    }));

    expect(parent.postMessage).not.toHaveBeenCalledWith(
      expect.objectContaining({ t: 'LOG_BATCH', requestId: 'forged-request' }),
      expect.anything(),
    );
    controller.dispose();
  });

  it('rejects owner log requests from the admitted parent when its origin is wrong', () => { // EMB-06
    const parent = { postMessage: vi.fn() };
    Object.defineProperty(window, 'parent', { configurable: true, value: parent });
    const controller = initIframeBridgeChild({
      parentOrigin: 'https://portal.example',
      appId: 'fixture',
      captureLogs: true,
    });
    parent.postMessage.mockClear();

    window.dispatchEvent(new MessageEvent('message', {
      origin: 'https://attacker.example',
      source: parent as unknown as Window,
      data: { v: 1, t: 'LOGS', cmd: 'PULL', requestId: 'wrong-origin' },
    }));

    expect(parent.postMessage).not.toHaveBeenCalledWith(
      expect.objectContaining({ t: 'LOG_BATCH', requestId: 'wrong-origin' }),
      expect.anything(),
    );
    controller.dispose();
  });

  it('streams bounded owner log events and returns a request-correlated bounded batch', () => { // EMB-07
    const parent = { postMessage: vi.fn() };
    Object.defineProperty(window, 'parent', { configurable: true, value: parent });
    const originalInfo = console.info;
    const info = vi.spyOn(console, 'info').mockImplementation(() => undefined);
    const controller = initIframeBridgeChild({
      parentOrigin: 'https://portal.example',
      appId: 'fixture',
      captureLogs: { enabled: true, streaming: true, bufferSize: 2 },
    });

    info.mockClear();
    console.info('owner event', { index: 1 });
    console.info('owner event', { index: 2 });
    console.info('owner event', { index: 3 });

    const streamedEvents = parent.postMessage.mock.calls
      .map(([payload]) => payload as Record<string, unknown>)
      .filter(payload => payload.t === 'LOG_EVENT');
    expect(streamedEvents).toHaveLength(3);
    expect(streamedEvents[0]).toMatchObject({
      v: 1,
      t: 'LOG_EVENT',
      event: { level: 'info', source: 'console', args: ['owner event', { index: 1 }] },
    });

    parent.postMessage.mockClear();
    window.dispatchEvent(new MessageEvent('message', {
      origin: 'https://portal.example',
      source: parent as unknown as Window,
      data: {
        v: 1,
        t: 'LOGS',
        cmd: 'PULL',
        requestId: 'portal-log-pull-7',
        options: { limit: 1 },
      },
    }));

    expect(parent.postMessage).toHaveBeenCalledWith(expect.objectContaining({
      v: 1,
      t: 'LOG_BATCH',
      requestId: 'portal-log-pull-7',
      events: [expect.objectContaining({
        seq: 3,
        args: ['owner event', { index: 3 }],
      })],
    }), 'https://portal.example');

    controller.dispose();
    info.mockRestore();
    console.info = originalInfo;
  });

  it('requires the per-viewer nonce when nonce admission is enabled', () => { // EMB-08
    const parent = { postMessage: vi.fn() };
    Object.defineProperty(window, 'parent', { configurable: true, value: parent });
    history.replaceState({}, '', '/?__vrooli_bridge_require_nonce=1&__vrooli_bridge_nonce=viewer-session-42');
    const pushState = vi.spyOn(history, 'pushState');
    const controller = initIframeBridgeChild({ parentOrigin: 'https://portal.example', appId: 'fixture' });

    const sendNavigation = (nonce?: string) => window.dispatchEvent(new MessageEvent('message', {
      origin: 'https://portal.example',
      source: parent as unknown as Window,
      data: {
        v: 1,
        t: 'NAV',
        cmd: 'GO',
        to: '/accepted',
        ...(nonce === undefined ? {} : { nonce }),
      },
    }));

    sendNavigation();
    sendNavigation('stale-viewer-session');
    expect(pushState).not.toHaveBeenCalled();

    sendNavigation('viewer-session-42');
    expect(pushState).toHaveBeenCalledWith({}, '', '/accepted');

    controller.dispose();
  });
});
