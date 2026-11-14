import { useEffect } from 'react';
import { act, render, screen } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { ReplayPlayerController, ReplayPlayerProps } from '@/domains/exports/replay/types';
import ReplayExportPage from './ReplayExportPage';

const player = vi.hoisted(() => ({ props: null as ReplayPlayerProps | null }));
vi.mock('@/domains/exports/replay/ReplayPlayer', () => ({ default: (props: ReplayPlayerProps) => {
  player.props = props;
  useEffect(() => {
    const controller: ReplayPlayerController = { seek: vi.fn(), play: vi.fn(), pause: vi.fn(), getViewportElement: () => null, getPresentationElement: () => null, getFrameCount: () => props.frames.length };
    props.exposeController?.(controller);
    return () => props.exposeController?.(null);
  }, [props.exposeController, props.frames]);
  return <div data-testid="player">{props.frames.length}</div>;
} }));
vi.mock('@/utils/logger', () => ({ logger: { warn: vi.fn(), error: vi.fn() } }));
function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: Error) => void;
  const promise = new Promise<T>((yes, no) => { resolve = yes; reject = no; });
  return { promise, resolve, reject };
}
const spec = (id: string, motion?: { speedProfile?: string; pathStyle?: string }) => ({ execution: { executionId: id }, frames: [{ index: 0, durationMs: 1000, status: 'completed' }], ...(motion ? { cursorMotion: motion } : {}) });
const response = (payload: unknown) => ({ ok: true, json: async () => payload });
async function message(data: Record<string, unknown>) {
  await act(async () => window.dispatchEvent(new MessageEvent('message', { data })));
}
function metadataId() { return window.basExport?.getMetadata().specId; }
describe('replay page request and presentation ownership', () => {
  beforeEach(() => {
    vi.useFakeTimers(); player.props = null; delete window.basExport; delete window.__BAS_EXPORT_BOOTSTRAP__; delete window.__BAS_EXPORT_API_BASE__;
    window.history.replaceState({}, '', '/export?mode=capture');
  });
  afterEach(() => { vi.useRealTimers(); vi.unstubAllGlobals(); });
  it.each(['resolve', 'reject'] as const)('keeps an inline successor when the previous fetch later %ss', async outcome => {
    const old = deferred<unknown>();
    const fetch = vi.fn().mockReturnValue(old.promise); vi.stubGlobal('fetch', fetch);
    render(<ReplayExportPage />);
    await message({ type: 'bas:spec:fetch', executionId: 'a' });
    await message({ type: 'bas:spec:set', spec: spec('b') });
    expect(metadataId()).toBe('b');
    expect(fetch.mock.calls[0]?.[1]?.signal?.aborted).toBe(true);
    await act(async () => outcome === 'resolve' ? old.resolve(response({ status: 'ready', package: spec('a') })) : old.reject(Error('old failed')));
    expect(metadataId()).toBe('b'); expect(window.basExport?.ready).toBe(true);
    expect(screen.queryByText('old failed')).toBeNull();
  });
  it('supersedes an in-flight execution fetch, stays unready while pending, and ignores its old finalizer', async () => {
    const a = deferred<unknown>(); const b = deferred<unknown>();
    const fetch = vi.fn().mockReturnValueOnce(a.promise).mockReturnValueOnce(b.promise); vi.stubGlobal('fetch', fetch);
    render(<ReplayExportPage />);
    await message({ type: 'bas:spec:set', spec: spec('initial') });
    await message({ type: 'bas:spec:fetch', executionId: 'a' });
    expect(window.basExport?.ready).toBe(false);
    await message({ type: 'bas:spec:fetch', executionId: 'b' });
    expect(fetch).toHaveBeenCalledTimes(2);
    await act(async () => a.resolve(response({ status: 'ready', package: spec('a') })));
    expect(window.basExport?.ready).toBe(false);
    await act(async () => b.resolve(response({ status: 'ready', package: spec('b') })));
    expect(metadataId()).toBe('b'); expect(window.basExport?.ready).toBe(true);
  });
  it('does not schedule an old pending retry after replacement or after unmount', async () => {
    const pending = deferred<unknown>(); const fetch = vi.fn().mockReturnValue(pending.promise); vi.stubGlobal('fetch', fetch);
    const { unmount } = render(<ReplayExportPage />);
    await message({ type: 'bas:spec:fetch', executionId: 'a' });
    await message({ type: 'bas:spec:set', spec: spec('b') });
    unmount();
    await act(async () => pending.resolve(response({ status: 'pending' })));
    await act(async () => vi.advanceTimersByTimeAsync(4000));
    expect(fetch).toHaveBeenCalledTimes(1);
    expect(window.basExport?.ready).toBe(false);
  });
  it('retains current pending-to-ready retry and visible current request errors', async () => {
    const fetch = vi.fn().mockResolvedValueOnce(response({ status: 'pending', message: 'Capturing frames' })).mockResolvedValueOnce(response({ status: 'ready', package: spec('a') })).mockRejectedValueOnce(Error('current failed'));
    vi.stubGlobal('fetch', fetch); render(<ReplayExportPage />);
    await message({ type: 'bas:spec:fetch', executionId: 'a' });
    expect(window.basExport?.ready).toBe(false);
    await act(async () => vi.advanceTimersByTimeAsync(4000));
    expect(metadataId()).toBe('a'); expect(window.basExport?.ready).toBe(true);
    await message({ type: 'bas:spec:fetch', executionId: 'b' });
    expect(screen.getByText('current failed')).toBeInTheDocument();
    expect(window.basExport?.ready).toBe(false); expect(window.basExport?.error).toBe('current failed');
  });
  it('rejects malformed current HTTP replay packages through the existing strict parser', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response({ status: 'ready', package: { ...spec('a'), unexpectedField: true } })));
    render(<ReplayExportPage />);
    await message({ type: 'bas:spec:fetch', executionId: 'a' });
    expect(screen.getByText('Replay export unavailable')).toBeInTheDocument();
    expect(window.basExport?.ready).toBe(false);
    expect(window.basExport?.error).toMatch(/unexpectedField|unknown/i);
  });
  it('retires a slow response body after a successor is adopted', async () => {
    const body = deferred<unknown>();
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, json: () => body.promise }));
    render(<ReplayExportPage />);
    await message({ type: 'bas:spec:fetch', executionId: 'a' });
    await message({ type: 'bas:spec:set', spec: spec('b') });
    await act(async () => body.resolve({ status: 'ready', package: spec('a') }));
    expect(metadataId()).toBe('b'); expect(window.basExport?.ready).toBe(true);
  });
  it('removes an already scheduled retry when a valid encoded spec replaces it', async () => {
    const fetch = vi.fn().mockResolvedValue(response({ status: 'pending' })); vi.stubGlobal('fetch', fetch);
    render(<ReplayExportPage />);
    await message({ type: 'bas:spec:fetch', executionId: 'a' });
    await message({ type: 'bas:spec:set-encoded', payload: btoa(JSON.stringify(spec('b'))) });
    await act(async () => vi.advanceTimersByTimeAsync(4000));
    expect(fetch).toHaveBeenCalledTimes(1); expect(metadataId()).toBe('b'); expect(window.basExport?.ready).toBe(true);
  });
  it('ignores an already queued stale retry without losing ownership of the current timer', async () => {
    const timer = vi.spyOn(window, 'setTimeout');
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response({ status: 'pending' })));
    const { unmount } = render(<ReplayExportPage />);
    await message({ type: 'bas:spec:fetch', executionId: 'a' });
    const oldRetry = timer.mock.calls.find(call => call[1] === 4000)?.[0];
    if (typeof oldRetry !== 'function') throw Error('missing retry');
    await message({ type: 'bas:spec:fetch', executionId: 'b' });
    act(() => oldRetry());
    unmount(); expect(vi.getTimerCount()).toBe(0);
    timer.mockRestore();
  });
  it('aborts an active request and never retries its late response after unmount', async () => {
    const pending = deferred<unknown>(); const fetch = vi.fn().mockReturnValue(pending.promise); vi.stubGlobal('fetch', fetch);
    const { unmount } = render(<ReplayExportPage />);
    await message({ type: 'bas:spec:fetch', executionId: 'a' }); unmount();
    expect(fetch.mock.calls[0]?.[1]?.signal?.aborted).toBe(true);
    await act(async () => pending.resolve(response({ status: 'pending' })));
    await act(async () => vi.advanceTimersByTimeAsync(4000));
    expect(fetch).toHaveBeenCalledTimes(1); expect(window.basExport?.ready).toBe(false);
  });
  it.each([
    ['instant', 'linear', 'instant', 'linear'], ['linear', 'pseudorandom', 'linear', 'pseudorandom'],
    ['easeIn', 'parabolicUp', 'easeIn', 'parabolicUp'], ['easeOut', 'parabolicDown', 'easeOut', 'parabolicDown'],
    ['easeInOut', 'cubic', 'easeInOut', 'cubic'], [' EASEINOUT ', ' Bezier ', 'easeInOut', 'cubic'],
    ['unknown', 'unknown', undefined, undefined],
  ])('preserves authored cursor motion %s / %s', async (speedProfile, pathStyle, expectedSpeed, expectedPath) => {
    render(<ReplayExportPage />);
    await message({ type: 'bas:spec:set', spec: spec('a', { speedProfile, pathStyle }) });
    expect(player.props?.cursorDefaultSpeedProfile).toBe(expectedSpeed);
    expect(player.props?.cursorDefaultPathStyle).toBe(expectedPath);
  });
});
