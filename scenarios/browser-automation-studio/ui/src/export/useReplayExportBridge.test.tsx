import { act, renderHook } from '@testing-library/react';
import { create } from '@bufbuild/protobuf';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ReplaySpecSchema } from '@vrooli/generated-proto/browser-automation-studio/v1/exports/exports_pb';
import type { ReplayPlayerController } from '@/domains/exports/replay/ReplayPlayer';
import { useReplayExportBridge } from './useReplayExportBridge';
import { logger } from '@/utils/logger';

vi.mock('@/utils/logger', () => ({ logger: { warn: vi.fn(), error: vi.fn() } }));
type Options = Parameters<typeof useReplayExportBridge>[0];
const spec = (id: string) => create(ReplaySpecSchema, { execution: { executionId: id } });
function options(overrides: Partial<Options> = {}): Options {
  return { mode: 'embedded', statusPayload: null, loadError: null, movieSpec: spec('a'),
    replayFrames: [{ id: '0', stepIndex: 0, success: true }], effectiveCanvasWidth: 320, effectiveCanvasHeight: 180,
    currentFrameIndex: 0, currentProgress: 0, isAwaitingSpec: false, assetCount: 0, totalDurationMs: 1000,
    timeline: [{ index: 0, startMs: 0, durationMs: 1000 }], executionSourceRef: { current: 'a' },
    fetchMovieSpec: vi.fn(), cancelSpecRequest: vi.fn(), reportStatus: vi.fn(), setMovieSpec: vi.fn(), setLoadError: vi.fn(),
    setStatusPayload: vi.fn(), setIsAwaitingSpec: vi.fn(), ...overrides };
}
function controller(): ReplayPlayerController {
  return { seek: vi.fn(), play: vi.fn(), pause: vi.fn(), getViewportElement: () => null, getPresentationElement: () => null, getFrameCount: () => 1 };
}
function expose(result: { current: ReturnType<typeof useReplayExportBridge> }, next = controller()) {
  act(() => result.current.handleExposeController(next));
  return next;
}
function seek(ms = 500) {
  if (!window.basExport) throw Error('missing API');
  return window.basExport.seekTo(ms).then(() => 'resolved', (error: Error) => error.message);
}
describe('public replay bridge ownership', () => {
  beforeEach(() => { delete window.basExport; vi.useFakeTimers(); vi.clearAllMocks(); });
  afterEach(() => { vi.useRealTimers(); delete window.basExport; vi.unstubAllGlobals(); });
  it('becomes unready while replacing a ready spec and retires metadata on unmount', () => {
    const initial = options();
    const { result, rerender, unmount } = renderHook(props => useReplayExportBridge(props), { initialProps: initial });
    expose(result);
    expect(window.basExport?.ready).toBe(true);
    rerender({ ...initial, isAwaitingSpec: true });
    expect(window.basExport?.ready).toBe(false);
    rerender({ ...initial, movieSpec: spec('b') });
    expect(window.basExport?.ready).toBe(true);
    expect(window.basExport?.getMetadata().specId).toBe('b');
    unmount();
    expect(window.basExport?.ready).toBe(false);
    expect(window.basExport?.getMetadata().specId).toBeNull();
  });
  it('acknowledges an optional request token only after that exact spec is adopted, including same-ID replacements', async () => {
    const initial = options();
    const { result, rerender } = renderHook(props => useReplayExportBridge(props), { initialProps: initial });
    expose(result);
    const adoptMessage = async (requestToken?: string) => {
      await act(async () => window.dispatchEvent(new MessageEvent('message', { data: { type: 'bas:spec:set', spec: { execution: { executionId: 'a' } }, requestToken } })));
      const adopted = vi.mocked(initial.setMovieSpec).mock.calls.at(-1)?.[0];
      if (!adopted || typeof adopted === 'function') throw Error('spec not adopted');
      return adopted;
    };
    const next = await adoptMessage('new-request');
    expect(window.basExport?.getMetadata().requestToken).toBeUndefined();
    rerender({ ...initial, movieSpec: next, isAwaitingSpec: true });
    expect(window.basExport?.getMetadata().requestToken).toBe('new-request');
    expect(window.basExport?.ready).toBe(false);
    rerender({ ...initial, movieSpec: next });
    expect(window.basExport?.ready).toBe(true);
    expect(window.basExport?.getMetadata().requestToken).toBe('new-request');
    const legacy = await adoptMessage();
    rerender({ ...initial, movieSpec: legacy });
    expect(window.basExport?.ready).toBe(true);
    expect(window.basExport?.getMetadata().requestToken).toBeUndefined();
  });
  it('announces a new spec even when its ready frame count is unchanged', () => {
    const postMessage = vi.fn();
    vi.stubGlobal('parent', { postMessage });
    const initial = options();
    const { result, rerender } = renderHook(props => useReplayExportBridge(props), { initialProps: initial });
    expose(result);
    rerender({ ...initial, movieSpec: spec('b') });
    const ready = postMessage.mock.calls.map(call => call[0]).filter(message => message.type === 'bas:ready');
    expect(ready.map(message => message.specId)).toEqual(['a', 'b']);
  });
  it('associates a current parse error with its attempted token without acknowledging adoption or relabeling a prior error', async () => {
    const initial = options({ movieSpec: null, loadError: 'previous failure' });
    const { result, rerender } = renderHook(props => useReplayExportBridge(props), { initialProps: initial });
    expose(result);
    expect(window.basExport?.getMetadata().requestError).toBeUndefined();
    await act(async () => window.dispatchEvent(new MessageEvent('message', { data: {
      type: 'bas:spec:set', requestToken: 'failed-attempt', spec: { invalidProtoField: true },
    } })));
    expect(window.basExport?.ready).toBe(false);
    expect(window.basExport?.getMetadata().requestToken).toBeUndefined();
    expect(window.basExport?.getMetadata().requestError).toEqual({ requestToken: 'failed-attempt', message: 'Invalid replay spec' });
    expect(initial.reportStatus).toHaveBeenCalledWith('error', 'Invalid replay spec');
    await act(async () => window.dispatchEvent(new MessageEvent('message', { data: {
      type: 'bas:spec:set', requestToken: 'valid-successor', spec: { execution: { executionId: 'a' } },
    } })));
    // The preceding API.error can remain visible until the parent state update;
    // receiving a token must not label that old error as the successor's error.
    expect(window.basExport?.error).toBe('previous failure');
    expect(window.basExport?.getMetadata().requestError).toBeUndefined();
    expect(window.basExport?.getMetadata().requestToken).toBeUndefined();
    const adopted = vi.mocked(initial.setMovieSpec).mock.calls.at(-1)?.[0];
    if (!adopted || typeof adopted === 'function') throw Error('missing parsed spec');
    rerender({ ...initial, movieSpec: adopted, loadError: null });
    expect(window.basExport?.ready).toBe(true);
    expect(window.basExport?.getMetadata().requestToken).toBe('valid-successor');
    expect(window.basExport?.getMetadata().requestError).toBeUndefined();
  });
  it.each(['untokened', 'encoded', 'fetch'])('retires attempted errors on %s replacement', async replacement => {
    const initial = options();
    const { result } = renderHook(props => useReplayExportBridge(props), { initialProps: initial });
    expose(result);
    await act(async () => window.dispatchEvent(new MessageEvent('message', { data: {
      type: 'bas:spec:set', requestToken: 'failed-attempt', spec: { invalidProtoField: true },
    } })));
    expect(window.basExport?.getMetadata().requestError?.requestToken).toBe('failed-attempt');
    const data = replacement === 'fetch' ? { type: 'bas:spec:fetch', executionId: 'a' }
      : replacement === 'encoded' ? { type: 'bas:spec:set-encoded', payload: btoa(JSON.stringify({ execution: { executionId: 'a' } })) }
      : { type: 'bas:spec:set', spec: { execution: { executionId: 'a' } } };
    await act(async () => window.dispatchEvent(new MessageEvent('message', { data })));
    expect(window.basExport?.getMetadata().requestError).toBeUndefined();
  });
  it('rejects a pending seek when its spec is replaced, before a successor acknowledges that frame', async () => {
    const initial = options();
    const { result, rerender } = renderHook(props => useReplayExportBridge(props), { initialProps: initial });
    expose(result);
    const pending = seek();
    rerender({ ...initial, movieSpec: spec('b'), currentProgress: .5 });
    await act(async () => {});
    expect(await pending).toMatch(/replaced|retired|cancel/i);
    expect(vi.getTimerCount()).toBe(0);
  });
  it('rejects outstanding seeks immediately on unmount and removes their timeout', async () => {
    const { result, unmount } = renderHook(props => useReplayExportBridge(props), { initialProps: options() });
    expose(result);
    const pending = seek();
    unmount();
    expect(vi.getTimerCount()).toBe(0);
    expect(await pending).toMatch(/unmount|retired|cancel/i);
  });
  it('does not retire a newer mount when an older mount unmounts', () => {
    const first = renderHook(props => useReplayExportBridge(props), { initialProps: options() });
    expose(first.result);
    const second = renderHook(props => useReplayExportBridge(props), { initialProps: options({ movieSpec: spec('b') }) });
    expose(second.result);
    const owner = window.basExport;
    first.unmount();
    expect(window.basExport).toBe(owner);
    expect(window.basExport?.getMetadata().specId).toBe('b');
  });
  it('retains successful bounded seek acknowledgement and reports timeout to direct callers', async () => {
    const initial = options();
    const { result, rerender } = renderHook(props => useReplayExportBridge(props), { initialProps: initial });
    const player = expose(result);
    const acknowledged = seek();
    expect(player.seek).toHaveBeenCalledWith({ frameIndex: 0, progress: .5 });
    rerender({ ...initial, currentProgress: .5 });
    expect(await acknowledged).toBe('resolved');
    const timedOut = seek(750);
    await act(async () => vi.advanceTimersByTimeAsync(6000));
    expect(await timedOut).toMatch(/Timed out/);
  });
  it('preserves a seek across normal controller re-exposure during frame changes', async () => {
    const initial = options({ replayFrames: [{ id: '0', stepIndex: 0, success: true }, { id: '1', stepIndex: 1, success: true }],
      totalDurationMs: 2000, timeline: [{ index: 0, startMs: 0, durationMs: 1000 }, { index: 1, startMs: 1000, durationMs: 1000 }] });
    const { result, rerender } = renderHook(props => useReplayExportBridge(props), { initialProps: initial });
    expose(result);
    const pending = seek(1250);
    // ReplayPlayer refreshes its controller for layout/current-frame changes.
    act(() => { result.current.handleExposeController(null); result.current.handleExposeController(controller()); });
    rerender({ ...initial, currentFrameIndex: 1, currentProgress: .25 });
    expect(await pending).toBe('resolved'); expect(vi.getTimerCount()).toBe(0);
  });
  it('retires a seek when its controller actually becomes unavailable', async () => {
    const { result } = renderHook(props => useReplayExportBridge(props), { initialProps: options() });
    expose(result);
    const pending = seek();
    act(() => result.current.handleExposeController(null));
    expect(await pending).toMatch(/retired/i); expect(window.basExport?.ready).toBe(false);
    expect(vi.getTimerCount()).toBe(0);
  });
  it('rejects a synchronous controller failure without retaining an orphan waiter', async () => {
    const { result } = renderHook(props => useReplayExportBridge(props), { initialProps: options() });
    const player = expose(result);
    vi.mocked(player.seek).mockImplementation(() => { throw Error('seek failed'); });
    expect(await seek()).toBe('seek failed');
    expect(vi.getTimerCount()).toBe(0);
  });
  it.each(['bas:control:seek', 'bas:control:frame'])('reports a throwing controller once for %s without orphan timers', async type => {
    const { result } = renderHook(props => useReplayExportBridge(props), { initialProps: options() });
    const player = expose(result);
    vi.mocked(player.seek).mockImplementation(() => { throw Error('controller failed'); });
    await act(async () => window.dispatchEvent(new MessageEvent('message', { data: { type, timeMs: 500, frameIndex: 0, progress: .5 } })));
    expect(logger.warn).toHaveBeenCalledTimes(1);
    expect(logger.warn).toHaveBeenCalledWith(expect.any(String), expect.any(Object), expect.objectContaining({ message: 'controller failed' }));
    expect(vi.getTimerCount()).toBe(0);
  });
  it('catches seek command failures through the existing diagnostics channel', async () => {
    renderHook(props => useReplayExportBridge(props), { initialProps: options() });
    await act(async () => window.dispatchEvent(new MessageEvent('message', { data: { type: 'bas:control:seek', timeMs: 100 } })));
    expect(logger.warn).toHaveBeenCalledWith(expect.any(String), expect.any(Object), expect.objectContaining({ message: 'Replay controller not ready' }));
  });
});
