import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { act, renderHook } from '@testing-library/react';
import { StrictMode } from 'react';
import { useBrowserNavigation } from './useBrowserNavigation';

const config = vi.hoisted(() => vi.fn(async () => ({ API_URL: 'http://fixture.test' })));
vi.mock('@/config', () => ({ getConfig: config }));

describe('browser history popup [REQ:BAS-RH-J04]', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('retains untitled browser entries without fabricated visit times and filters malformed entries', async () => {
    const back = { url: 'https://fixture.test/one', title: '' };
    const current = { url: 'https://fixture.test/two', title: 'Current' };
    const forward = { url: 'https://fixture.test/three', title: '', timestamp: '2026-09-22T00:00:00Z' };
    const fetch = vi.fn(async () => new Response(JSON.stringify({
      back_stack: [back, null, { url: 1, title: '' }, { url: 'https://invalid.test', title: null }],
      current, forward_stack: [forward],
    }), { status: 200 }));
    vi.stubGlobal('fetch', fetch);
    const { result } = renderHook(() => useBrowserNavigation({ pageId: 'red-page', sessionId: 'owned-session' }));
    await act(async () => {
      expect(await result.current.handleFetchNavigationStack()).toEqual({
        backStack: [{ ...back, timestamp: undefined }], current: { ...current, timestamp: undefined }, forwardStack: [forward],
      });
    });
    expect(fetch).toHaveBeenCalledWith('http://fixture.test/recordings/live/owned-session/navigation-stack?page_id=red-page', expect.objectContaining({ signal: expect.any(AbortSignal) }));
  });

  it('returns no popup for a rejected session read', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response('{}', { status: 404 })));
    const { result } = renderHook(() => useBrowserNavigation({ pageId: 'red-page', sessionId: 'closed-session' }));
    await act(async () => expect(await result.current.handleFetchNavigationStack()).toBeNull());
  });
});

const response = (url: string, status = 200) => new Response(JSON.stringify({
  url, can_go_back: true, can_go_forward: false,
}), { status });
const deferred = <T,>() => {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => { resolve = done; });
  return { promise, resolve };
};
const flush = () => act(async () => { await Promise.resolve(); });

describe('explicit browser navigation [REQ:BAS-RH-J04]', () => {
  beforeEach(() => config.mockReset().mockResolvedValue({ API_URL: 'http://fixture.test' }));
  afterEach(() => vi.unstubAllGlobals());

  it('observes restored, selected and externally navigated URLs without commanding the browser', async () => {
    const fetch = vi.fn();
    vi.stubGlobal('fetch', fetch);
    const { result } = renderHook(() => useBrowserNavigation({ pageId: 'red-page', sessionId: 'owned' }));
    act(() => result.current.setPreviewUrl('about:blank'));
    act(() => result.current.setPreviewUrl('https://fixture.test/external'));
    act(() => result.current.updateNavigationState({ url: 'https://fixture.test/history', can_go_back: true }));
    await flush();
    expect(result.current.previewUrl).toBe('https://fixture.test/history');
    expect(fetch).not.toHaveBeenCalled();
  });

  it('submits an explicit URL once and observes a redirect without replaying it', async () => {
    const fetch = vi.fn<typeof globalThis.fetch>(async () => response('https://fixture.test/redirected'));
    vi.stubGlobal('fetch', fetch);
    const { result } = renderHook(() => useBrowserNavigation({ pageId: 'red-page', sessionId: 'owned' }));
    act(() => result.current.handleNavigate('https://fixture.test/requested'));
    await flush();
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(JSON.parse(fetch.mock.calls[0][1]!.body as string)).toEqual({ url: 'https://fixture.test/requested', page_id: 'red-page' });
    expect(result.current.previewUrl).toBe('https://fixture.test/redirected');
    expect(result.current.canGoBack).toBe(true);
  });

  it('allows explicit repeat navigation to the same URL', async () => {
    const fetch = vi.fn(async () => response('https://fixture.test/repeat'));
    vi.stubGlobal('fetch', fetch);
    const { result } = renderHook(() => useBrowserNavigation({ pageId: 'red-page', sessionId: 'owned' }));
    for (let i = 0; i < 2; i++) {
      act(() => result.current.handleNavigate('https://fixture.test/repeat'));
      await flush();
    }
    expect(fetch).toHaveBeenCalledTimes(2);
  });

  it('admits one initial command under StrictMode effect replay', async () => {
    const fetch = vi.fn(async () => response('https://fixture.test/launch'));
    vi.stubGlobal('fetch', fetch);
    const { result } = renderHook(() => useBrowserNavigation({
      pageId: 'red-page', sessionId: 'owned', initialUrl: 'https://fixture.test/launch',
    }), { wrapper: StrictMode });
    await flush();
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(result.current.isInitialNavigationComplete).toBe(true);
  });

  it('does not abort an explicit request when a browser location notification arrives', async () => {
    const pending = deferred<Response>();
    const fetch = vi.fn().mockReturnValue(pending.promise);
    vi.stubGlobal('fetch', fetch);
    const { result } = renderHook(() => useBrowserNavigation({ pageId: 'red-page', sessionId: 'owned' }));
    act(() => result.current.handleNavigate('https://fixture.test/start'));
    await flush();
    act(() => result.current.setPreviewUrl('https://fixture.test/redirecting'));
    expect(fetch.mock.calls[0][1].signal.aborted).toBe(false);
    await act(async () => pending.resolve(response('https://fixture.test/final')));
    expect(result.current.previewUrl).toBe('https://fixture.test/final');
    expect(fetch).toHaveBeenCalledTimes(1);
  });

  it('rejects a late parsed response after unmount', async () => {
    const body = deferred<unknown>();
    const fetch = vi.fn().mockResolvedValue({ ok: true, json: () => body.promise });
    vi.stubGlobal('fetch', fetch);
    const { result, unmount } = renderHook(() => useBrowserNavigation({
      pageId: 'red-page', sessionId: 'owned', initialUrl: 'https://fixture.test/launch',
    }));
    await flush();
    unmount();
    await act(async () => body.resolve({ url: 'https://fixture.test/stale' }));
    expect(fetch.mock.calls[0][1].signal.aborted).toBe(true);
    expect(result.current.isInitialNavigationComplete).toBe(false);
  });

  it('waits for session admission before executing the initial launch URL', async () => {
    const fetch = vi.fn(async () => response('https://fixture.test/launched'));
    vi.stubGlobal('fetch', fetch);
    const { result, rerender } = renderHook(({ sessionId }: { sessionId: string | null }) =>
      useBrowserNavigation({ pageId: 'red-page', sessionId, initialUrl: 'https://fixture.test/launch' }),
      { initialProps: { sessionId: null } });
    await flush();
    expect(fetch).not.toHaveBeenCalled();
    expect(result.current.isInitialNavigationComplete).toBe(false);
    rerender({ sessionId: 'admitted' });
    await flush();
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(result.current.isInitialNavigationComplete).toBe(true);
  });

  it('retains submitted intent through session creation and unrelated URL observations', async () => {
    const fetch = vi.fn<typeof globalThis.fetch>(async () => response('https://fixture.test/submitted'));
    vi.stubGlobal('fetch', fetch);
    const { result, rerender } = renderHook(({ sessionId }: { sessionId: string | null }) =>
      useBrowserNavigation({ pageId: 'red-page', sessionId }), { initialProps: { sessionId: null } });
    act(() => result.current.handleNavigate('https://fixture.test/submitted'));
    act(() => result.current.setPreviewUrl('https://fixture.test/restored'));
    rerender({ sessionId: 'admitted' });
    await flush();
    expect(JSON.parse(fetch.mock.calls[0][1]!.body as string)).toEqual({ url: 'https://fixture.test/submitted', page_id: 'red-page' });
    expect(fetch).toHaveBeenCalledTimes(1);
  });

  it('does not start an obsolete command after late configuration arrives', async () => {
    const pending = deferred<{ API_URL: string }>();
    config.mockReturnValueOnce(pending.promise);
    const fetch = vi.fn(async () => response('https://fixture.test/new'));
    vi.stubGlobal('fetch', fetch);
    const { result } = renderHook(() => useBrowserNavigation({ pageId: 'red-page', sessionId: 'owned' }));
    act(() => result.current.handleNavigate('https://fixture.test/old'));
    act(() => result.current.handleNavigate('https://fixture.test/new'));
    await flush();
    await act(async () => pending.resolve({ API_URL: 'http://fixture.test' }));
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(result.current.previewUrl).toBe('https://fixture.test/new');
  });

  it('rejects a superseded response even when its transport ignores abort', async () => {
    const pending = deferred<Response>();
    const fetch = vi.fn().mockReturnValueOnce(pending.promise)
      .mockResolvedValueOnce(response('https://fixture.test/new'));
    vi.stubGlobal('fetch', fetch);
    const { result } = renderHook(() => useBrowserNavigation({ pageId: 'red-page', sessionId: 'owned' }));
    act(() => result.current.handleNavigate('https://fixture.test/old'));
    await flush();
    act(() => result.current.handleNavigate('https://fixture.test/new'));
    await flush();
    await act(async () => pending.resolve(response('https://fixture.test/stale')));
    expect(result.current.previewUrl).toBe('https://fixture.test/new');
    expect(fetch.mock.calls[0][1].signal.aborted).toBe(true);
  });

  it('does not replay an admitted intent into a replacement session', async () => {
    const pending = deferred<Response>();
    const fetch = vi.fn().mockReturnValue(pending.promise);
    vi.stubGlobal('fetch', fetch);
    const { result, rerender } = renderHook(({ sessionId }) =>
      useBrowserNavigation({ pageId: 'red-page', sessionId }), { initialProps: { sessionId: 'old-session' } });
    act(() => result.current.handleNavigate('https://fixture.test/old'));
    await flush();
    rerender({ sessionId: 'new-session' });
    act(() => result.current.setPreviewUrl('https://fixture.test/new'));
    await act(async () => pending.resolve(response('https://fixture.test/stale')));
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(result.current.previewUrl).toBe('https://fixture.test/new');
  });

  it('does not fetch after unmount while configuration is pending', async () => {
    const pending = deferred<{ API_URL: string }>();
    config.mockReturnValueOnce(pending.promise);
    const fetch = vi.fn();
    vi.stubGlobal('fetch', fetch);
    const { result, unmount } = renderHook(() => useBrowserNavigation({ pageId: 'red-page', sessionId: 'owned' }));
    act(() => result.current.handleNavigate('https://fixture.test/closed'));
    unmount();
    await act(async () => pending.resolve({ API_URL: 'http://fixture.test' }));
    expect(fetch).not.toHaveBeenCalled();
  });

  it('keeps failed launch navigation visible and retryable without marking it ready', async () => {
    const fetch = vi.fn().mockResolvedValueOnce(response('', 502))
      .mockResolvedValueOnce(response('https://fixture.test/launch'));
    vi.stubGlobal('fetch', fetch);
    const { result } = renderHook(() => useBrowserNavigation({ pageId: 'red-page', sessionId: 'owned', initialUrl: 'https://fixture.test/launch' }));
    await flush();
    expect(result.current.isInitialNavigationComplete).toBe(false);
    expect(result.current.navigationError).toContain('502');
    act(() => result.current.handleNavigate('https://fixture.test/launch'));
    await flush();
    expect(fetch).toHaveBeenCalledTimes(2);
    expect(result.current.isInitialNavigationComplete).toBe(true);
    expect(result.current.navigationError).toBeNull();
  });
});

describe('navigation page admission [REQ:BAS-RH-J03]', () => {
  const commands = ['navigate', 'handleGoBack', 'handleGoForward', 'handleRefresh'] as const;
  const invoke = (api: ReturnType<typeof useBrowserNavigation>, command: typeof commands[number]) =>
    command === 'navigate' ? api.handleNavigate('https://fixture.test/destination') : api[command]();
  beforeEach(() => config.mockReset().mockResolvedValue({ API_URL: 'http://fixture.test' }));
  afterEach(() => vi.unstubAllGlobals());

  it('binds the selected page at submission even when React batches a subsequent switch', async () => {
    const fetch = vi.fn();
    vi.stubGlobal('fetch', fetch);
    const { result, rerender } = renderHook(({ pageId }) => useBrowserNavigation({ sessionId: 'owned', pageId }),
      { initialProps: { pageId: 'red-page' } });
    act(() => {
      result.current.handleNavigate('https://fixture.test/red-only');
      rerender({ pageId: 'blue-page' });
    });
    await flush();
    expect(fetch).not.toHaveBeenCalled();
  });

  it.each([false, true])('does not replay cancelled intent after returning to the original tab (batched=%s)', async batched => {
    const fetch = vi.fn(async () => response('https://fixture.test/red-only'));
    vi.stubGlobal('fetch', fetch);
    const { result, rerender } = renderHook(({ pageId }) => useBrowserNavigation({ sessionId: 'owned', pageId }),
      { initialProps: { pageId: 'red-page' } });
    act(() => {
      result.current.handleNavigate('https://fixture.test/red-only');
      if (batched) rerender({ pageId: 'blue-page' });
    });
    await flush();
    if (!batched) rerender({ pageId: 'blue-page' });
    rerender({ pageId: 'red-page' });
    await flush();
    expect(fetch).toHaveBeenCalledTimes(batched ? 0 : 1);
  });

  it.each(commands)('%s carries the selected canonical page', async command => {
    const fetch = vi.fn<typeof globalThis.fetch>(async () => response('https://fixture.test/destination'));
    vi.stubGlobal('fetch', fetch);
    const { result } = renderHook(() => useBrowserNavigation({ sessionId: 'owned', pageId: 'red-page' }));
    await act(async () => { await invoke(result.current, command); });
    await flush();
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(JSON.parse(fetch.mock.calls[0][1]!.body as string)).toMatchObject({ page_id: 'red-page' });
  });

  it.each(commands)('%s rejects completion after switching pages', async command => {
    const pending = deferred<Response>();
    const fetch = vi.fn().mockReturnValue(pending.promise);
    vi.stubGlobal('fetch', fetch);
    const { result, rerender } = renderHook(({ pageId }) => useBrowserNavigation({ sessionId: 'owned', pageId }),
      { initialProps: { pageId: 'red-page' } });
    act(() => { void invoke(result.current, command); });
    await flush();
    rerender({ pageId: 'blue-page' });
    act(() => result.current.setPreviewUrl('https://fixture.test/blue'));
    await act(async () => pending.resolve(response('https://fixture.test/stale')));
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(fetch.mock.calls[0][1].signal?.aborted).toBe(true);
    expect(result.current.previewUrl).toBe('https://fixture.test/blue');
    expect(result.current.refreshToken).toBe(0);
  });

  it('waits for a page before admitting the initial URL', async () => {
    const fetch = vi.fn(async () => response('https://fixture.test/launch'));
    vi.stubGlobal('fetch', fetch);
    const { result, rerender } = renderHook(({ pageId }: { pageId: string | null }) => useBrowserNavigation({
      sessionId: 'owned', pageId, initialUrl: 'https://fixture.test/launch',
    }), { initialProps: { pageId: null } });
    await flush();
    expect(fetch).not.toHaveBeenCalled();
    expect(result.current.isInitialNavigationComplete).toBe(false);
    rerender({ pageId: 'red-page' });
    await flush();
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(result.current.isInitialNavigationComplete).toBe(true);
  });

  it('does not send a late config completion or replay its intent into another page', async () => {
    const pending = deferred<{ API_URL: string }>();
    config.mockReturnValueOnce(pending.promise);
    const fetch = vi.fn();
    vi.stubGlobal('fetch', fetch);
    const { result, rerender } = renderHook(({ pageId }) => useBrowserNavigation({ sessionId: 'owned', pageId }),
      { initialProps: { pageId: 'red-page' } });
    act(() => result.current.handleNavigate('https://fixture.test/old-page'));
    rerender({ pageId: 'blue-page' });
    await act(async () => pending.resolve({ API_URL: 'http://fixture.test' }));
    expect(fetch).not.toHaveBeenCalled();
  });

  it('stops a multi-step history operation when its page changes', async () => {
    const pending = deferred<Response>();
    const fetch = vi.fn().mockReturnValueOnce(pending.promise).mockImplementation(async () => response('https://fixture.test/old'));
    vi.stubGlobal('fetch', fetch);
    const { result, rerender } = renderHook(({ pageId }) => useBrowserNavigation({ sessionId: 'owned', pageId }),
      { initialProps: { pageId: 'red-page' } });
    let operation: Promise<void>;
    act(() => { operation = result.current.handleNavigateToIndex(-3); });
    await flush();
    rerender({ pageId: 'blue-page' });
    act(() => result.current.setPreviewUrl('https://fixture.test/blue'));
    await act(async () => { pending.resolve(response('https://fixture.test/stale')); await operation; });
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(result.current.previewUrl).toBe('https://fixture.test/blue');
  });

  it('discards a history-popup read completed for an old page', async () => {
    const pending = deferred<Response>();
    vi.stubGlobal('fetch', vi.fn().mockReturnValue(pending.promise));
    const { result, rerender } = renderHook(({ pageId }) => useBrowserNavigation({ sessionId: 'owned', pageId }),
      { initialProps: { pageId: 'red-page' } });
    const operation = result.current.handleFetchNavigationStack();
    await flush();
    rerender({ pageId: 'blue-page' });
    let received;
    await act(async () => { pending.resolve(new Response(JSON.stringify({ back_stack: [], current: { url: 'https://fixture.test/old', title: 'Old' }, forward_stack: [] }))); received = await operation; });
    expect(received).toBeNull();
  });
});

describe('canonical page location observation [REQ:BAS-RH-J03]', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('shows an existing selected page on attachment without navigating it', async () => {
    const fetch = vi.fn();
    vi.stubGlobal('fetch', fetch);
    const { result } = renderHook(() => useBrowserNavigation({
      sessionId: 'owned', pageId: 'red-page', observedUrl: 'https://fixture.test/existing',
    }));
    await flush();
    expect(result.current.previewUrl).toBe('https://fixture.test/existing');
    expect(fetch).not.toHaveBeenCalled();
  });

  it('observes late hydration, selected-page changes and empty selection', async () => {
    const fetch = vi.fn();
    vi.stubGlobal('fetch', fetch);
    type Observation = { pageId: string | null; observedUrl: string | undefined };
    const { result, rerender } = renderHook((observation: Observation) => useBrowserNavigation({
      sessionId: 'owned', ...observation,
    }), { initialProps: { pageId: null, observedUrl: undefined } as Observation });
    rerender({ pageId: 'red-page', observedUrl: 'https://fixture.test/red' });
    await flush();
    expect(result.current.previewUrl).toBe('https://fixture.test/red');
    rerender({ pageId: 'blue-page', observedUrl: 'https://fixture.test/blue' });
    await flush();
    expect(result.current.previewUrl).toBe('https://fixture.test/blue');
    rerender({ pageId: null, observedUrl: '' });
    await flush();
    expect(result.current.previewUrl).toBe('');
    expect(fetch).not.toHaveBeenCalled();
  });

  it('retains a local value on unchanged observation and observes external navigation', async () => {
    const fetch = vi.fn();
    vi.stubGlobal('fetch', fetch);
    const { result, rerender } = renderHook(({ observedUrl }) => useBrowserNavigation({
      sessionId: 'owned', pageId: 'red-page', observedUrl,
    }), { initialProps: { observedUrl: 'https://fixture.test/current' } });
    act(() => result.current.setPreviewUrl('https://fixture.test/submitted'));
    rerender({ observedUrl: 'https://fixture.test/current' });
    await flush();
    expect(result.current.previewUrl).toBe('https://fixture.test/submitted');
    rerender({ observedUrl: 'https://fixture.test/external' });
    await flush();
    expect(result.current.previewUrl).toBe('https://fixture.test/external');
    expect(fetch).not.toHaveBeenCalled();
  });
});


describe('owned history capability observations [REQ:BAS-RH-J03]', () => {
  beforeEach(() => config.mockReset().mockResolvedValue({API_URL: 'http://fixture.test'}));
  afterEach(() => vi.unstubAllGlobals());
  it('refreshes browser capabilities without replacing a pending URL draft', async () => {
    const fetch = vi.fn<typeof globalThis.fetch>(async () => response('https://fixture.test/observed'));
    vi.stubGlobal('fetch', fetch);
    const {result} = renderHook(() => useBrowserNavigation({sessionId: 'owned', pageId: 'red-page'}));
    act(() => result.current.setPreviewUrl('https://draft.test/pending'));
    await act(async () => result.current.refreshNavigationState());
    expect(result.current.canGoBack).toBe(true);
    expect(result.current.canGoForward).toBe(false);
    expect(result.current.previewUrl).toBe('https://draft.test/pending');
    expect(fetch).toHaveBeenCalledWith('http://fixture.test/recordings/live/owned/navigation-state?page_id=red-page', expect.objectContaining({signal: expect.any(AbortSignal)}));
    expect(fetch.mock.calls[0][1]?.method).toBeUndefined();
  });
  it('clears old capabilities and discards a read after switching pages', async () => {
    const pending = deferred<Response>();
    vi.stubGlobal('fetch', vi.fn(() => pending.promise));
    const {result, rerender} = renderHook(({pageId}) => useBrowserNavigation({sessionId: 'owned', pageId}), {initialProps: {pageId: 'red'}});
    act(() => result.current.updateNavigationState({can_go_back: true, can_go_forward: true}));
    let read!: Promise<void>;
    act(() => {read = result.current.refreshNavigationState();});
    await flush();rerender({pageId: 'blue'});
    expect(result.current.canGoBack).toBe(false);
    expect(result.current.canGoForward).toBe(false);
    await act(async () => {pending.resolve(response('https://red.test'));await read;});
    expect(result.current.canGoBack).toBe(false);
    expect(result.current.canGoForward).toBe(false);
  });
  it.each(['new read', 'navigation command'])('ignores older capabilities after a %s completes', async newer => {
    const pending = deferred<Response>();
    const fetch = vi.fn<typeof globalThis.fetch>().mockImplementationOnce(() => pending.promise)
      .mockImplementation(async () => new Response(JSON.stringify({can_go_back: false, can_go_forward: true})));
    vi.stubGlobal('fetch', fetch);
    const {result} = renderHook(() => useBrowserNavigation({sessionId: 'owned', pageId: 'red'}));
    let old!: Promise<void>;
    act(() => {old = result.current.refreshNavigationState();});
    await flush();
    await act(async () => {if (newer === 'new read') await result.current.refreshNavigationState();else await result.current.handleGoBack();});
    expect(result.current.canGoForward).toBe(true);
    await act(async () => {pending.resolve(response('https://old.test'));await old;});
    expect(result.current.canGoBack).toBe(false);
    expect(result.current.canGoForward).toBe(true);
  });

});
