import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, cleanup, renderHook, waitFor } from '@testing-library/react';
import { createElement, type PropsWithChildren } from 'react';
import { WebSocketProvider } from '@/contexts/WebSocketProvider';
import { useWebSocket, useWebSocketMessage } from '@/contexts/WebSocketContext';
import { usePages } from './usePages';
import { useSessionStore } from '../stores/sessionStore';

vi.mock('@/config', () => ({getApiBase: () => 'http://fixture.test'}));
const pages = ['red', 'blue'].map(id => ({id, sessionId: 'session', url: `https://fixture.test/${id}`,
  title: id, isInitial: id === 'red', status: 'active', createdAt: '2026-09-23T00:00:00Z'}));
class TestSocket {
  static OPEN = 1;
  static instances: TestSocket[] = [];
  readyState = 1;
  binaryType = '';
  onopen: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onclose: ((event: CloseEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  send = vi.fn();
  close = vi.fn();
  constructor() {TestSocket.instances.push(this);}
  receive(value: unknown) {this.onmessage?.({data: JSON.stringify(value)} as MessageEvent);}
}
const wrapper = ({children}: PropsWithChildren) => createElement(WebSocketProvider, null, children);

describe('wire-to-page lifecycle delivery [REQ:BAS-RH-J03] [REQ:BAS-RH-J05]', () => {
  beforeEach(() => {
    TestSocket.instances = [];
    vi.stubGlobal('WebSocket', TestSocket);
    vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify({pages, activePageId: 'red'}))));
    useSessionStore.setState({sessionId: 'session', isValidated: true, pages: new Map(), activePageId: null});
  });
  afterEach(() => {cleanup(); vi.useRealTimers(); vi.unstubAllGlobals();});
  it.each(['spaced', 'burst'])('retains both closed pages and final empty selection from %s messages', async delivery => {
    const {result} = renderHook(() => usePages({sessionId: 'session'}), {wrapper});
    await waitFor(() => expect(result.current.openPageCount).toBe(2));
    const socket = TestSocket.instances[0];
    const messages = [
      ...['blue', 'red'].map(id => ({type: 'page_event', session_id: 'session', event: {id: `close-${id}`, pageId: id, type: 'page_closed', timestamp: '2026-09-23T00:01:00Z'}})),
      {type: 'page_switch', session_id: 'session', active_page_id: ''},
    ];
    if (delivery === 'burst') act(() => messages.forEach(message => socket.receive(message)));
    else for (const message of messages) await act(async () => socket.receive(message));
    expect(result.current.openPages).toEqual([]);
    expect(result.current.activePageId).toBeNull();
    expect([...useSessionStore.getState().pages.values()].map(page => page.status)).toEqual(['closed', 'closed']);
  });
  it('rejects callbacks from a replaced socket', () => {
    vi.useFakeTimers();
    const {result} = renderHook(() => useWebSocket(), {wrapper});
    const old = TestSocket.instances[0];
    const opened = old.onopen!;
    const closed = old.onclose!;
    act(() => result.current.reconnect());
    expect(TestSocket.instances).toHaveLength(2);
    act(() => {opened(new Event('open')); closed({code: 1006, reason: 'late old socket'} as CloseEvent); vi.advanceTimersByTime(1000);});
    expect(TestSocket.instances).toHaveLength(2);
    expect(result.current.isConnected).toBe(false);
  });
  it('cannot reconnect from a close callback after provider disposal', () => {
    vi.useFakeTimers();
    const {unmount} = renderHook(() => useWebSocket(), {wrapper});
    const closed = TestSocket.instances[0].onclose!;
    unmount();
    act(() => {closed({code: 1000, reason: 'disposed'} as CloseEvent); vi.advanceTimersByTime(1000);});
    expect(TestSocket.instances).toHaveLength(1);
  });

  it('delivers intact domain fields to every subscriber using the latest committed callback', () => {
    const deliveries: unknown[] = [];
    const {rerender, unmount} = renderHook(({scope}) => {
      useWebSocketMessage(() => {throw new Error('controlled subscriber failure');});
      useWebSocketMessage(message => deliveries.push({scope, message}));
    }, {wrapper, initialProps: {scope: 'first'}});
    const socket = TestSocket.instances[0];
    const message = {type: 'domain_event', custom: {sequence: 1}, active_page_id: ''};
    act(() => socket.receive(message));
    rerender({scope: 'second'});
    act(() => socket.receive({...message, custom: {sequence: 2}}));
    expect(deliveries).toEqual([{scope: 'first', message}, {scope: 'second', message: {...message, custom: {sequence: 2}}}]);
    unmount();
    act(() => socket.receive({...message, custom: {sequence: 3}}));
    expect(deliveries).toHaveLength(2);
  });
  it('ignores malformed JSON and stale socket messages without losing subsequent valid events', () => {
    const callback = vi.fn();
    const {result} = renderHook(() => {useWebSocketMessage(callback); return useWebSocket();}, {wrapper});
    const old = TestSocket.instances[0];
    const receive = old.onmessage!;
    act(() => {receive({data: '{broken'} as MessageEvent); old.receive({type: 123});});
    expect(callback).not.toHaveBeenCalled();
    act(() => result.current.reconnect());
    act(() => receive({data: JSON.stringify({type: 'late'})} as MessageEvent));
    expect(callback).not.toHaveBeenCalled();
    act(() => TestSocket.instances[1].receive({type: 'current', payload: 'retained'}));
    expect(callback).toHaveBeenCalledExactlyOnceWith({type: 'current', payload: 'retained'});
  });

});
