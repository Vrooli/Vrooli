import type React from 'react';
import { act, renderHook, waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useInputForwarding } from './useInputForwarding';

const socket = vi.hoisted(() => ({isConnected: false, send: vi.fn(), listeners: new Set<(message: Record<string, unknown>) => void>()}));
vi.mock('@/contexts/WebSocketContext', () => ({
  useWebSocket: () => socket,
  useWebSocketMessage: (callback: (message: Record<string, unknown>) => void) => { socket.listeners.add(callback); },
}));
vi.mock('@/config', () => ({getConfig: async () => ({API_URL:'http://fixture.test/api/v1'})}));

// [REQ:BAS-RH-J13] An empty page selection cannot target an implicit browser page.
describe('input page selection', () => {
  const fetchInput=vi.fn();
  beforeEach(() => {socket.isConnected=false;socket.send.mockReset();socket.listeners.clear();fetchInput.mockReset().mockImplementation(async (_url, request) => {
    const input = JSON.parse(request.body);
    return new Response(JSON.stringify({status:'ok', input_id:input.input_id, applied_sequence:1}));
  });vi.stubGlobal('fetch',fetchInput);});
  afterEach(() => vi.unstubAllGlobals());
  it.each([false,true])('honors explicit empty and implicit active selection (WebSocket=%s)',async(connected)=>{
    socket.isConnected=connected;
    const h=renderHook(({pageId}:{pageId?:string|null})=>useInputForwarding({sessionId:'session-a',pageId}),{initialProps:{pageId:null} as {pageId?:string|null}});
    h.result.current.setWsSubscribed(connected);
    const wheel={deltaX:2,deltaY:3,preventDefault:vi.fn(),stopPropagation:vi.fn()} as unknown as React.WheelEvent<HTMLElement>;
    await act(async()=>{h.result.current.handleWheel(wheel,true);});
    expect(socket.send).not.toHaveBeenCalled();expect(fetchInput).not.toHaveBeenCalled();
    h.rerender({pageId:undefined});
    await act(async()=>{h.result.current.handleWheel(wheel,true);});
    const implicit=connected?socket.send.mock.calls[0]?.[0]:JSON.parse(fetchInput.mock.calls[0]?.[1].body);
    expect(implicit).not.toHaveProperty('page_id');
    h.rerender({pageId:'page-b'});
    await act(async()=>{h.result.current.handleWheel(wheel,true);});
    const explicit=connected?socket.send.mock.calls[1]?.[0]:JSON.parse(fetchInput.mock.calls[1]?.[1].body);
    expect(explicit.page_id).toBe('page-b');
  });
});

describe('browser input semantic preservation [REQ:BAS-RH-J03]', () => {
  const fetchInput = vi.fn();
  beforeEach(() => {
    socket.isConnected = true;
    socket.send.mockReset();
    socket.listeners.clear();
    fetchInput.mockReset().mockImplementation(async (_url, request) => {
      const input = JSON.parse(request.body);
      return new Response(JSON.stringify({status:'ok', input_id:input.input_id, applied_sequence:1}));
    });
    vi.stubGlobal('fetch', fetchInput);
  });
  afterEach(() => vi.unstubAllGlobals());

  function inputHook() {
    const hook = renderHook(() => useInputForwarding({
      sessionId: 'session-input', pageId: 'page-input',
      viewport: { width: 800, height: 600 }, frameDimensions: { width: 800, height: 600 },
    }));
    hook.result.current.setWsSubscribed(true);
    return hook.result.current;
  }

  function keyboardEvent(overrides: Record<string, unknown> = {}) {
    return {
      key: 'a', altKey: false, ctrlKey: false, metaKey: false, shiftKey: false,
      nativeEvent: { isComposing: false }, preventDefault: vi.fn(), stopPropagation: vi.fn(),
      ...overrides,
    } as unknown as React.KeyboardEvent<HTMLElement>;
  }

  it.each([
    ['Ctrl+A', { key: 'a', ctrlKey: true }, 'Control'],
    ['Command+C', { key: 'c', metaKey: true }, 'Meta'],
    ['Alt+F', { key: 'f', altKey: true }, 'Alt'],
  ])('forwards %s as a key chord instead of text', (_, event, modifier) => {
    const hook = inputHook();
    hook.handleKey(keyboardEvent(event), true);
    const input = socket.send.mock.calls[0]?.[0].input;
    expect(input).toMatchObject({ type: 'keyboard', key: event.key, modifiers: [modifier] });
    expect(input).not.toHaveProperty('text');
  });

  it('keeps ordinary printable input as text', () => {
    inputHook().handleKey(keyboardEvent({ key: 'x' }), true);
    expect(socket.send.mock.calls[0]?.[0].input).toMatchObject({ type: 'keyboard', text: 'x' });
    expect(socket.send.mock.calls[0]?.[0].input.input_id).toEqual(expect.any(String));
  });

  it('forwards pointer modifier state', () => {
    const pointer = {
      clientX: 100, clientY: 200, button: 0,
      altKey: false, ctrlKey: false, metaKey: false, shiftKey: true,
      preventDefault: vi.fn(), stopPropagation: vi.fn(),
    } as unknown as React.PointerEvent<HTMLElement>;
    inputHook().handlePointer('down', pointer, { left: 0, top: 0, width: 800, height: 600 }, true);
    expect(socket.send.mock.calls[0]?.[0].input).toMatchObject({
      type: 'pointer', action: 'down', button: 'left', modifiers: ['Shift'],
    });
  });

  it('releases a held pointer on viewer blur with its last position and modifiers', () => {
    const hook = inputHook();
    const pointer = {
      clientX: 100, clientY: 200, button: 0, pointerId: 7,
      altKey: false, ctrlKey: false, metaKey: false, shiftKey: true,
      preventDefault: vi.fn(), stopPropagation: vi.fn(),
    } as unknown as React.PointerEvent<HTMLElement>;
    hook.handlePointer('down', pointer, { left: 0, top: 0, width: 800, height: 600 }, true);

    act(() => window.dispatchEvent(new Event('blur')));

    expect(socket.send).toHaveBeenCalledTimes(2);
    expect(socket.send.mock.calls[1]?.[0].input).toMatchObject({
      type: 'pointer', action: 'up', button: 'left', x: 100, y: 200, modifiers: ['Shift'],
    });
  });

  it('releases on pointer-up outside the viewer and pointer cancellation', () => {
    const hook = inputHook();
    const sendDown = (button: number, pointerId: number) => hook.handlePointer('down', {
      clientX: 100, clientY: 200, button, pointerId,
      altKey: false, ctrlKey: false, metaKey: false, shiftKey: false,
      preventDefault: vi.fn(), stopPropagation: vi.fn(),
    } as unknown as React.PointerEvent<HTMLElement>, { left: 0, top: 0, width: 800, height: 600 }, true);
    sendDown(0, 17);
    act(() => window.dispatchEvent(Object.assign(new Event('pointerup'), { button: 0, pointerId: 17 })));
    expect(socket.send.mock.calls[1]?.[0].input).toMatchObject({ type: 'pointer', action: 'up', button: 'left' });

    sendDown(2, 18);
    act(() => window.dispatchEvent(Object.assign(new Event('pointercancel'), { button: -1, pointerId: 18 })));
    expect(socket.send.mock.calls[3]?.[0].input).toMatchObject({ type: 'pointer', action: 'up', button: 'right' });
  });

  it('releases held buttons through HTTP when the WebSocket connection drops', async () => {
    socket.isConnected = true;
    const mounted = renderHook(() => useInputForwarding({
      sessionId: 'session-input', pageId: 'page-input',
      viewport: { width: 800, height: 600 }, frameDimensions: { width: 800, height: 600 },
    }));
    mounted.result.current.setWsSubscribed(true);
    const pointer = {
      clientX: 45, clientY: 67, button: 2, pointerId: 9,
      altKey: false, ctrlKey: false, metaKey: false, shiftKey: false,
      preventDefault: vi.fn(), stopPropagation: vi.fn(),
    } as unknown as React.PointerEvent<HTMLElement>;
    mounted.result.current.handlePointer('down', pointer, { left: 0, top: 0, width: 800, height: 600 }, true);

    socket.isConnected = false;
    mounted.rerender();

    await waitFor(() => expect(fetchInput).toHaveBeenCalledTimes(2));
    const replay = JSON.parse(fetchInput.mock.calls[0]?.[1].body);
    const release = JSON.parse(fetchInput.mock.calls[1]?.[1].body);
    expect(replay).toMatchObject({ type: 'pointer', action: 'down', button: 'right', x: 45, y: 67 });
    expect(replay.input_id).toBe(socket.send.mock.calls[0]?.[0].input.input_id);
    expect(release).toMatchObject({
      type: 'pointer', action: 'up', button: 'right', x: 45, y: 67,
    });
  });

  it('does not replay an input after receiving its matching WebSocket receipt', async () => {
    socket.isConnected = true;
    const mounted = renderHook(() => useInputForwarding({ sessionId: 'session-input', pageId: 'page-input' }));
    mounted.result.current.setWsSubscribed(true);
    const wheel = { deltaX: 1, deltaY: 2, preventDefault: vi.fn(), stopPropagation: vi.fn() } as unknown as React.WheelEvent<HTMLElement>;
    mounted.result.current.handleWheel(wheel, true);
    const inputId = socket.send.mock.calls[0]?.[0].input.input_id;
    for (const listener of socket.listeners) {
      listener({ type: 'recording_input_applied', session_id: 'session-input', input_id: inputId });
    }

    socket.isConnected = false;
    await act(async () => { mounted.rerender(); await Promise.resolve(); });

    expect(fetchInput).not.toHaveBeenCalled();
  });

  it('does not forward or consume IME composition Process keydown', () => {
    const event = keyboardEvent({ key: 'Process', nativeEvent: { isComposing: true } });
    inputHook().handleKey(event, true);
    expect(socket.send).not.toHaveBeenCalled();
    expect(event.preventDefault).not.toHaveBeenCalled();
  });
});
