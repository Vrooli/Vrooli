import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, renderHook } from '@testing-library/react';
import { StrictMode, createElement, type PropsWithChildren } from 'react';
import { usePages } from './usePages';
import { useSessionStore } from '../stores/sessionStore';

vi.mock('@/config', () => ({ getApiBase: () => 'http://fixture.test' }));
const socket = vi.hoisted(() => ({lastMessage: null as unknown}));
vi.mock('@/contexts/WebSocketContext', async () => {
  const {useEffect} = await import('react');
  return {useWebSocketMessage: (callback: (message: unknown) => void) => {
    useEffect(() => {if (socket.lastMessage) callback(socket.lastMessage);}, [socket.lastMessage]);
  }};
});

const page = {id: 'target', sessionId: 'session', url: 'https://fixture.test', title: 'Target',
  createdAt: '2026-09-23T00:00:00Z', isInitial: true, status: 'active'};

describe('browser close receipts [REQ:BAS-RH-J03]', () => {
  beforeEach(() => {socket.lastMessage = null; useSessionStore.setState({sessionId: 'session', isValidated: false, pages: new Map(), activePageId: null});});
  afterEach(() => {vi.unstubAllGlobals(); vi.restoreAllMocks();});
  it.each(['remaining', ''])('applies selected page %j to hook and session store without a callback', async selected => {
    const fetch = vi.fn(async (_url: unknown, init?: RequestInit) => new Response(JSON.stringify(init?.method === 'POST'
      ? {closedPageId: 'target', activePageId: selected}
      : {pages: selected ? [page, {...page, id: selected, isInitial: false}] : [page], activePageId: 'target'}), {status: 200}));
    vi.stubGlobal('fetch', fetch);
    const {result} = renderHook(() => usePages({sessionId: 'session'}));
    await act(async () => {await result.current.refreshPages();});
    expect(result.current.activePageId).toBe('target');
    await act(async () => {await result.current.closePage('target');});
    expect(result.current.activePageId).toBe(selected || null);
    expect(useSessionStore.getState().activePageId).toBe(selected || null);
    expect(result.current.pages.get('target')?.status).toBe('closed');
    expect(useSessionStore.getState().pages.get('target')?.status).toBe('closed');
  });
  it.each(['rejected', 'malformed'])('retains the open tab for %s close receipts', async fault => {
    vi.spyOn(console, 'error').mockImplementation(() => undefined);
    vi.stubGlobal('fetch', vi.fn(async (_url: unknown, init?: RequestInit) => new Response(JSON.stringify(init?.method === 'POST'
      ? {} : {pages: [page], activePageId: 'target'}), {status: init?.method === 'POST' && fault === 'rejected' ? 503 : 200})));
    const {result} = renderHook(() => usePages({sessionId: 'session'}));
    await act(async () => {await result.current.refreshPages();});
    await act(async () => {await result.current.closePage('target');});
    expect(result.current.error).toBeTruthy();
    expect(result.current.activePageId).toBe('target');
    expect(result.current.pages.get('target')?.status).toBe('active');
    expect(useSessionStore.getState().pages.get('target')?.status).toBe('active');
  });
  it('clears selection after a final-tab page-switch notification', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify({pages: [page], activePageId: 'target'}), {status: 200})));
    const {result, rerender} = renderHook(() => usePages({sessionId: 'session'}));
    await act(async () => {await result.current.refreshPages();});
    socket.lastMessage = {type: 'page_switch', session_id: 'session', active_page_id: ''};
    rerender();
    expect(result.current.activePageId).toBeNull();
    expect(useSessionStore.getState().activePageId).toBeNull();
  });
  it('applies an explicit tab switch to the shared selection', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response('{}', {status: 200})));
    const {result} = renderHook(() => usePages({sessionId: 'session'}));
    await act(async () => {await result.current.switchToPage('remaining');});
    expect(result.current.activePageId).toBe('remaining');
    expect(useSessionStore.getState().activePageId).toBe('remaining');
  });

});

// [REQ:BAS-RH-J03] Admission and events share one UI page owner across handoffs.
describe('tab admission and session ownership [REQ:BAS-RH-J03]', () => {
  beforeEach(() => {socket.lastMessage = null; useSessionStore.setState({sessionId: 'session', isValidated: false, pages: new Map(), activePageId: null});});
  afterEach(() => {vi.unstubAllGlobals(); vi.restoreAllMocks();});
  const second = {...page, id: 'second', title: 'Second', isInitial: false};
  const response = (payload: unknown, status = 200) => new Response(JSON.stringify(payload), {status});
  function deferred<T>() {
    let resolve!: (value: T) => void;
    const promise = new Promise<T>(done => {resolve = done;});
    return {promise, resolve};
  }
  it('shows and selects a created tab before recording callbacks exist', async () => {
    vi.stubGlobal('fetch', vi.fn(async (_url: unknown, init?: RequestInit) => init?.method === 'POST'
      ? response({driverPageId: 'driver-second', url: second.url, page: second, activePageId: second.id}, 201)
      : response({pages: [page], activePageId: page.id})));
    const onPageCreated = vi.fn();
    const {result} = renderHook(() => usePages({sessionId: 'session', onPageCreated}));
    await act(async () => {await result.current.refreshPages();});
    await act(async () => {await result.current.createPage(second.url);});
    expect(result.current.openPages.map(p => p.id)).toEqual(['target', 'second']);
    expect(result.current.activePageId).toBe('second');
    expect(useSessionStore.getState().pages.get('second')).toEqual(second);
    expect(useSessionStore.getState().activePageId).toBe('second');
    expect(onPageCreated).toHaveBeenCalledTimes(1);
  });
  it.each([false, true])('observes each created page once, StrictMode=%s', async strict => {
    const onPageCreated = vi.fn();
    const wrapper = strict ? ({children}: PropsWithChildren) => createElement(StrictMode, null, children) : undefined;
    const {result, rerender} = renderHook(() => usePages({sessionId: 'session', onPageCreated}), {wrapper});
    const created = {type: 'page_event', session_id: 'session', event: {id: 'event', type: 'page_created', pageId: second.id, url: second.url, title: second.title, timestamp: second.createdAt}};
    socket.lastMessage = created;
    rerender();
    expect(result.current.pages.has('second')).toBe(true);
    expect(onPageCreated).toHaveBeenCalledTimes(1);
    socket.lastMessage = {...created};
    rerender();
    expect(onPageCreated).toHaveBeenCalledTimes(1);
  });
  it('discards an old page-list completion after switching sessions', async () => {
    const old = deferred<Response>();
    const current = {...page, id: 'current', sessionId: 'new-session'};
    vi.stubGlobal('fetch', vi.fn(async (url: unknown) => String(url).includes('/new-session/')
      ? response({pages: [current], activePageId: current.id}) : old.promise));
    const {result, rerender} = renderHook(({sessionId}) => usePages({sessionId}), {initialProps: {sessionId: 'session'}});
    let pending!: Promise<void>;
    act(() => {pending = result.current.refreshPages();});
    rerender({sessionId: 'new-session'});
    await act(async () => {await result.current.refreshPages();});
    await act(async () => {old.resolve(response({pages: [page], activePageId: page.id})); await pending;});
    expect(result.current.pageList).toEqual([current]);
    expect(useSessionStore.getState().pages.get('current')).toEqual(current);
    expect(result.current.activePageId).toBe('current');
  });
  it.each(['close', 'switch', 'create'] as const)('ignores a late %s completion from the previous session', async operation => {
    const old = deferred<Response>();
    const current = {...page, id: 'current', sessionId: 'new-session'};
    vi.stubGlobal('fetch', vi.fn(async (url: unknown, init?: RequestInit) => init?.method === 'POST' ? old.promise
      : response(String(url).includes('/new-session/') ? {pages: [current], activePageId: current.id} : {pages: [page], activePageId: page.id})));
    const onPageCreated = vi.fn();
    const {result, rerender} = renderHook(({sessionId}) => usePages({sessionId, onPageCreated}), {initialProps: {sessionId: 'session'}});
    await act(async () => {await result.current.refreshPages();});
    let pending!: Promise<void>;
    act(() => {pending = operation === 'close' ? result.current.closePage(page.id) : operation === 'switch' ? result.current.switchToPage(page.id) : result.current.createPage(second.url);});
    rerender({sessionId: 'new-session'});
    await act(async () => {await result.current.refreshPages();});
    await act(async () => {old.resolve(response(operation === 'create' ? {page: second, activePageId: second.id} : {closedPageId: page.id, activePageId: ''})); await pending;});
    expect(result.current.pageList).toEqual([current]);
    expect(result.current.activePageId).toBe('current');
    expect(useSessionStore.getState().activePageId).toBe('current');
    expect(onPageCreated).not.toHaveBeenCalled();
  });
  it.each([false, true])('joins the creation receipt and event, callback first=%s', async callbackFirst => {
    const admission = deferred<Response>();
    vi.stubGlobal('fetch', vi.fn(async () => admission.promise));
    const onPageCreated = vi.fn();
    const {result, rerender} = renderHook(() => usePages({sessionId: 'session', onPageCreated}));
    let pending!: Promise<void>;
    act(() => {pending = result.current.createPage(second.url);});
    const callback = () => {
      socket.lastMessage = {type: 'page_event', session_id: 'session', event: {id: 'event', type: 'page_created', pageId: second.id, url: second.url, title: 'Early title', timestamp: second.createdAt}};
      rerender();
    };
    if (callbackFirst) callback();
    await act(async () => {admission.resolve(response({page: second, activePageId: second.id}, 201)); await pending;});
    if (!callbackFirst) callback();
    expect(onPageCreated).toHaveBeenCalledTimes(1);
    expect(result.current.pages.get(second.id)?.title).toBe('Second');
    expect(result.current.activePageId).toBe(second.id);
    expect(result.current.pages.size).toBe(1);
  });
  it('does not publish a page receipt after the hook unmounts', async () => {
    const admission = deferred<Response>();
    vi.stubGlobal('fetch', vi.fn(async () => admission.promise));
    const onPageCreated = vi.fn();
    const {result, unmount} = renderHook(() => usePages({sessionId: 'session', onPageCreated}));
    let pending!: Promise<void>;
    act(() => {pending = result.current.createPage(second.url);});
    unmount();
    await act(async () => {admission.resolve(response({page: second, activePageId: second.id}, 201)); await pending;});
    expect(useSessionStore.getState().pages.size).toBe(0);
    expect(onPageCreated).not.toHaveBeenCalled();
  });
  it.each(['rejected', 'foreign', 'malformed'])('preserves pages for a %s creation response', async fault => {
    vi.spyOn(console, 'error').mockImplementation(() => undefined);
    vi.stubGlobal('fetch', vi.fn(async (_url: unknown, init?: RequestInit) => init?.method === 'POST'
      ? response(fault === 'foreign' ? {page: {...second, sessionId: 'foreign'}, activePageId: second.id} : {}, fault === 'rejected' ? 503 : 201)
      : response({pages: [page], activePageId: page.id})));
    const onPageCreated = vi.fn();
    const {result} = renderHook(() => usePages({sessionId: 'session', onPageCreated}));
    await act(async () => {await result.current.refreshPages();});
    await act(async () => {await result.current.createPage(second.url);});
    expect(result.current.error).toBeTruthy();
    expect(result.current.pageList.map(p => p.id)).toEqual([page.id]);
    expect(result.current.activePageId).toBe(page.id);
    expect(onPageCreated).not.toHaveBeenCalled();
  });

});


describe('browser favicon observations [REQ:BAS-RH-J04]', () => {
  beforeEach(() => {socket.lastMessage = null; useSessionStore.setState({sessionId: 'session', isValidated: false, pages: new Map(), activePageId: null});});
  afterEach(() => {vi.unstubAllGlobals(); vi.restoreAllMocks();});
  it('reads metadata and distinguishes absent, changed-document and explicit empty icons', async () => {
    const icon = 'https://fixture.test/custom.svg';
    vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify({pages: [{...page, faviconUrl: icon}], activePageId: page.id}))));
    const {result, rerender} = renderHook(() => usePages({sessionId: 'session'}));
    await act(async () => {await result.current.refreshPages();});
    expect(result.current.activePage?.faviconUrl).toBe(icon);
    const observe = (metadata: object) => {
      socket.lastMessage = {type: 'page_event', session_id: 'session', event: {id: 'event', type: 'page_navigated', pageId: page.id, timestamp: page.createdAt, ...metadata}};
      rerender();
    };
    observe({url: page.url, title: 'New title'});
    expect(result.current.activePage?.faviconUrl).toBe(icon);
    observe({url: 'https://fixture.test/next'});
    expect(result.current.activePage?.faviconUrl).toBe('');
    observe({faviconUrl: icon});
    expect(result.current.activePage?.faviconUrl).toBe(icon);
    observe({faviconUrl: ''});
    expect(result.current.activePage?.faviconUrl).toBe('');
  });
  it('keeps a newly created tab favicon in the canonical page store', () => {
    const {result, rerender} = renderHook(() => usePages({sessionId: 'session'}));
    socket.lastMessage = {type: 'page_event', session_id: 'session', event: {type: 'page_created', pageId: 'new', url: page.url, title: 'New', faviconUrl: 'data:image/png;base64,fixture'}};
    rerender();
    expect(result.current.pages.get('new')?.faviconUrl).toBe('data:image/png;base64,fixture');
  });
});
