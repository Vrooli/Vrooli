import type React from 'react';
import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useInputForwarding } from './useInputForwarding';

const socket = vi.hoisted(() => ({isConnected: false, send: vi.fn()}));
vi.mock('@/contexts/WebSocketContext', () => ({useWebSocket: () => socket}));
vi.mock('@/config', () => ({getConfig: async () => ({API_URL:'http://fixture.test/api/v1'})}));

// [REQ:BAS-RH-J13] An empty page selection cannot target an implicit browser page.
describe('input page selection', () => {
  const fetchInput=vi.fn();
  beforeEach(() => {socket.send.mockReset();fetchInput.mockReset().mockResolvedValue(new Response('{}'));vi.stubGlobal('fetch',fetchInput);});
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
