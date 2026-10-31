import { act, renderHook } from '@/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useRecordMode } from './useRecordMode';
import type { RecordedAction } from '../types/types';
const mocks = vi.hoisted(() => ({ start: vi.fn(), stop: vi.fn(), generate: vi.fn(), replay: vi.fn(), toast: vi.fn() }));
vi.mock('../api', () => ({ recordingApi: { startRecording: mocks.start, stopRecording: mocks.stop, generateWorkflow: mocks.generate, replayPreview: mocks.replay } }));
vi.mock('react-hot-toast', () => ({ default: { success: mocks.toast } }));
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>(done => { resolve = done; });
  return { promise, resolve };
}
describe('recording lifecycle results belong to the current session', () => {
  beforeEach(() => { vi.clearAllMocks(); mocks.start.mockReset(); mocks.stop.mockReset(); mocks.generate.mockReset(); mocks.replay.mockReset(); });
  it.each([true, false])('keeps current B loading and state when old A start success=%s completes', async success => {
    const a = deferred<unknown>(); const b = deferred<unknown>(); mocks.start.mockReturnValueOnce(a.promise).mockReturnValueOnce(b.promise);
    const { result, rerender } = renderHook(({ id }) => useRecordMode({ sessionId: id }), { initialProps: { id: 'a' } });
    let old!: Promise<void>; let current!: Promise<void>;
    act(() => { old = result.current.startRecording(); });
    rerender({ id: 'b' }); act(() => { current = result.current.startRecording(); });
    await act(async () => { a.resolve(success ? { success: true, data: { recording_id: 'old-recording' } } : { success: false, error: 'old failure' }); await old; });
    expect(result.current.isLoading).toBe(true); expect(result.current.recordingId).toBeNull();
    expect(result.current.error).toBeNull(); expect(mocks.toast).not.toHaveBeenCalled();
    await act(async () => { b.resolve({ success: true, data: { recording_id: 'current-recording' } }); await current; });
    expect(result.current.recordingId).toBe('current-recording'); expect(result.current.isRecording).toBe(true);
  });
  it('does not stop current B when old A stop finishes', async () => {
    const stop = deferred<unknown>(); mocks.start.mockResolvedValue({ success: true, data: { recording_id: 'recording' } }); mocks.stop.mockReturnValue(stop.promise);
    const { result, rerender } = renderHook(({ id }) => useRecordMode({ sessionId: id }), { initialProps: { id: 'a' } });
    await act(async () => result.current.startRecording());
    let oldStop!: Promise<void>; act(() => { oldStop = result.current.stopRecording(); });
    rerender({ id: 'b' }); await act(async () => result.current.startRecording());
    await act(async () => { stop.resolve({ success: true, data: {} }); await oldStop; });
    expect(result.current.isRecording).toBe(true);
  });
  it('resets loading when a pending session is retired even if transport has not settled', () => {
    const pending = deferred<unknown>(); mocks.start.mockReturnValue(pending.promise);
    const { result, rerender } = renderHook(({ id }) => useRecordMode({ sessionId: id }), { initialProps: { id: 'a' } });
    act(() => { void result.current.startRecording(); });
    expect(result.current.isLoading).toBe(true); rerender({ id: 'b' });
    expect(result.current.isLoading).toBe(false);
  });
  it.each(['generate', 'replay'] as const)('retired %s cannot clear a successor loading flag or error', async operation => {
    const a = deferred<unknown>(); const b = deferred<unknown>();
    const api = operation === 'generate' ? mocks.generate : mocks.replay;
    api.mockReturnValueOnce(a.promise).mockReturnValueOnce(b.promise);
    const actions: RecordedAction[] = [{ id: 'action', sessionId: 'a', sequenceNum: 1, timestamp: '2026-10-16T00:00:00Z', actionType: 'click', confidence: 1, url: 'http://fixture.test' }];
    const { result, rerender } = renderHook(({ id }) => useRecordMode({ sessionId: id }), { initialProps: { id: 'a' } });
    const run = () => operation === 'generate' ? result.current.generateWorkflow('flow', undefined, actions) : result.current.replayPreview({}, actions);
    let old!: Promise<unknown>; let current!: Promise<unknown>;
    act(() => { old = run().catch(error => error); });
    rerender({ id: 'b' }); act(() => { current = run(); });
    await act(async () => { a.resolve({ success: false, error: 'old failure' }); await old; });
    expect(operation === 'generate' ? result.current.isLoading : result.current.isReplaying).toBe(true);
    expect(result.current.error).toBeNull();
    await act(async () => { b.resolve({ success: true, data: {} }); await current; });
    expect(operation === 'generate' ? result.current.isLoading : result.current.isReplaying).toBe(false);
  });
  it('does not accept an A result after A to B to A navigation or after unmount', async () => {
    const a = deferred<unknown>(); mocks.start.mockReturnValue(a.promise);
    const { result, rerender, unmount } = renderHook(({ id }) => useRecordMode({ sessionId: id }), { initialProps: { id: 'a' } });
    let pending!: Promise<void>; act(() => { pending = result.current.startRecording(); });
    const signal = mocks.start.mock.calls[0][1].signal as AbortSignal;
    rerender({ id: 'b' }); rerender({ id: 'a' });
    await act(async () => { a.resolve({ success: true, data: { recording_id: 'old-a' } }); await pending; });
    expect(result.current.recordingId).toBeNull(); expect(mocks.toast).not.toHaveBeenCalled(); expect(signal.aborted).toBe(true);
    unmount();
  });
  it('preserves a legitimate explicit initial session handover and current failure visibility', async () => {
    const pending = deferred<unknown>(); mocks.start.mockReturnValueOnce(pending.promise);
    const { result, rerender } = renderHook(({ id }: { id: string | null }) => useRecordMode({ sessionId: id }), { initialProps: { id: null } });
    let start!: Promise<void>; act(() => { start = result.current.startRecording('created'); });
    const signal = mocks.start.mock.calls[0][1].signal as AbortSignal;
    rerender({ id: 'created' }); expect(signal.aborted).toBe(false); expect(result.current.isLoading).toBe(true);
    await act(async () => { pending.resolve({ success: true, data: { recording_id: 'current' } }); await start; });
    expect(result.current.isRecording).toBe(true); expect(result.current.recordingId).toBe('current');
    mocks.stop.mockResolvedValueOnce({ success: false, error: 'current stop failure' });
    await act(async () => result.current.stopRecording());
    expect(result.current.error).toBe('current stop failure'); expect(result.current.isLoading).toBe(false);
  });
  it('preserves a completed initial recording when its created session props arrive afterwards', async () => {
    mocks.start.mockResolvedValueOnce({ success: true, data: { recording_id: 'created-recording' } });
    const { result, rerender } = renderHook(({ id }: { id: string | null }) => useRecordMode({ sessionId: id }), { initialProps: { id: null } });
    await act(async () => result.current.startRecording('created'));
    expect(result.current.recordingId).toBe('created-recording'); expect(result.current.isRecording).toBe(true);
    const signal = mocks.start.mock.calls[0][1].signal as AbortSignal;
    rerender({ id: 'created' });
    expect(signal.aborted).toBe(false); expect(result.current.isLoading).toBe(false);
    expect(result.current.recordingId).toBe('created-recording'); expect(result.current.isRecording).toBe(true);
    expect(mocks.toast).toHaveBeenCalledTimes(1);
    rerender({ id: 'different-session' });
    expect(signal.aborted).toBe(true); expect(result.current.recordingId).toBeNull(); expect(result.current.isRecording).toBe(false);
  });
  it('ignores a late start after unmount', async () => {
    const pending = deferred<unknown>(); mocks.start.mockReturnValue(pending.promise);
    const { result, unmount } = renderHook(() => useRecordMode({ sessionId: 'a' }));
    let start!: Promise<void>; act(() => { start = result.current.startRecording(); });
    const signal = mocks.start.mock.calls[0][1].signal as AbortSignal; unmount();
    await act(async () => { pending.resolve({ success: true, data: { recording_id: 'retired' } }); await start; });
    expect(signal.aborted).toBe(true); expect(mocks.toast).not.toHaveBeenCalled();
  });
  it.each(['generate', 'replay'] as const)('does not dispatch retained %s commands after unmount', async operation => {
    const api = operation === 'generate' ? mocks.generate : mocks.replay;
    api.mockResolvedValue({ success: true, data: {} });
    const { result, unmount } = renderHook(() => useRecordMode({ sessionId: 'a' }));
    const actions: RecordedAction[] = [{ id: 'action', sessionId: 'a', sequenceNum: 1, timestamp: '2026-10-16T00:00:00Z', actionType: 'click', confidence: 1, url: 'http://fixture.test' }];
    const retained = operation === 'generate' ? () => result.current.generateWorkflow('flow', undefined, actions) : () => result.current.replayPreview({}, actions);
    unmount(); await expect(retained()).rejects.toMatchObject({ name: 'AbortError' });
    expect(api).not.toHaveBeenCalled();
  });
});
