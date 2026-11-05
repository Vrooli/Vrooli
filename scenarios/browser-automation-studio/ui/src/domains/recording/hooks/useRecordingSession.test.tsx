import { act, renderHook } from '@/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useRecordingSession } from './useRecordingSession';

const mocks = vi.hoisted(() => ({
  create: vi.fn(), ready: vi.fn(),
  store: { sessionId: null as string | null, isValidated: false, setSession: vi.fn(), setIsCreating: vi.fn(), setError: vi.fn(), setRetryState: vi.fn(), clearSession: vi.fn(), validateSession: vi.fn() },
}));
vi.mock('../api', () => ({ recordingApi: { createSession: mocks.create } }));
vi.mock('../stores', () => ({ useSessionStore: (selector: (state: typeof mocks.store) => unknown) => selector(mocks.store) }));
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>(done => { resolve = done; });
  return { promise, resolve };
}

describe('recording session hook retires callbacks from a replaced identity', () => {
  beforeEach(() => { vi.clearAllMocks(); mocks.create.mockReset(); mocks.store.validateSession.mockReset(); mocks.store.sessionId = null; mocks.store.isValidated = false; });
  afterEach(() => vi.useRealTimers());
  it.each([true, false])('does not let late validation A mutate local/store B when valid=%s', async valid => {
    const a = deferred<boolean>(); mocks.store.validateSession.mockReturnValueOnce(a.promise).mockResolvedValue(true);
    const { result, rerender } = renderHook(({ id }) => useRecordingSession({ initialSessionId: id }), { initialProps: { id: 'a' } });
    rerender({ id: 'b' }); await act(async () => {});
    expect(result.current.sessionId).toBe('b');
    mocks.store.setSession.mockClear();
    await act(async () => a.resolve(valid));
    expect(result.current.sessionId).toBe('b');
    expect(mocks.store.setSession).not.toHaveBeenCalledWith(expect.objectContaining({ sessionId: 'a' }));
  });
  it('does not publish a late created A after URL replacement B', async () => {
    const a = deferred<unknown>(); mocks.create.mockReturnValue(a.promise); mocks.store.validateSession.mockResolvedValue(true);
    const { result, rerender } = renderHook(({ id }: { id: string | null }) => useRecordingSession({ initialSessionId: id, onSessionReady: mocks.ready }), { initialProps: { id: null } });
    let creation!: Promise<string | null>;
    act(() => { creation = result.current.ensureSession(); });
    rerender({ id: 'b' }); await act(async () => {});
    mocks.store.setSession.mockClear();
    await act(async () => { a.resolve({ success: true, data: { session_id: 'old-created-a' } }); await creation; });
    expect(result.current.sessionId).toBe('b');
    expect(mocks.ready).not.toHaveBeenCalled();
    expect(mocks.store.setSession).not.toHaveBeenCalledWith(expect.objectContaining({ sessionId: 'old-created-a' }));
  });
  it('keeps one creation for legitimate concurrent calls in the same session lifetime', async () => {
    const pending = deferred<unknown>(); mocks.create.mockReturnValue(pending.promise);
    const { result } = renderHook(() => useRecordingSession({ initialSessionId: null, onSessionReady: mocks.ready }));
    let first!: Promise<string | null>; let second!: Promise<string | null>;
    act(() => { first = result.current.ensureSession(); second = result.current.ensureSession(); });
    expect(mocks.create).toHaveBeenCalledTimes(1);
    await act(async () => pending.resolve({ success: true, data: { session_id: 'same-session' } }));
    expect(await first).toBe('same-session'); expect(await second).toBe('same-session');
    expect(mocks.ready).toHaveBeenCalledTimes(1);
  });
  it('keeps B creation and its shared promise when A later finalizes', async () => {
    const a = deferred<unknown>(); const b = deferred<unknown>(); mocks.create.mockReturnValueOnce(a.promise).mockReturnValueOnce(b.promise);
    const { result, rerender } = renderHook(({ profile }) => useRecordingSession({ initialSessionId: null, initialSessionProfileId: profile, onSessionReady: mocks.ready }), { initialProps: { profile: 'a' } });
    let old!: Promise<string | null>; let current!: Promise<string | null>;
    act(() => { old = result.current.ensureSession(); });
    rerender({ profile: 'b' }); act(() => { current = result.current.ensureSession(); });
    await act(async () => { a.resolve({ success: true, data: { session_id: 'old-a' } }); await old; });
    expect(await old).toBeNull(); expect(result.current.isCreatingSession).toBe(true);
    let shared!: Promise<string | null>; act(() => { shared = result.current.ensureSession(); });
    expect(mocks.create).toHaveBeenCalledTimes(2);
    await act(async () => b.resolve({ success: true, data: { session_id: 'current-b' } }));
    expect(await current).toBe('current-b'); expect(await shared).toBe('current-b'); expect(mocks.ready).toHaveBeenCalledTimes(1);
  });
  it('ignores late creation after unmount and retains current retry behavior', async () => {
    const pending = deferred<unknown>(); mocks.create.mockReturnValue(pending.promise);
    const { result, unmount } = renderHook(() => useRecordingSession({ initialSessionId: null, onSessionReady: mocks.ready }));
    let creation!: Promise<string | null>; act(() => { creation = result.current.ensureSession(); });
    const signal = mocks.create.mock.calls[0][1].signal as AbortSignal; unmount();
    await act(async () => { pending.resolve({ success: true, data: { session_id: 'retired' } }); await creation; });
    expect(await creation).toBeNull(); expect(signal.aborted).toBe(true); expect(mocks.ready).not.toHaveBeenCalled();
    const fresh = renderHook(() => useRecordingSession({ initialSessionId: null }));
    mocks.create.mockResolvedValueOnce({ success: false, error: 'current failure' });
    await act(async () => { await fresh.result.current.ensureSession(); });
    expect(fresh.result.current.sessionError).toBe('current failure'); expect(fresh.result.current.retryState.inCooldown).toBe(true);
    act(() => fresh.result.current.retrySession());
    expect(fresh.result.current.sessionError).toBeNull(); expect(fresh.result.current.retryState.inCooldown).toBe(false);
    mocks.create.mockResolvedValueOnce({ success: true, data: { session_id: 'retry-success' } });
    await act(async () => { expect(await fresh.result.current.ensureSession()).toBe('retry-success'); });
  });
  it('retires retry cooldown immediately even when the replacement URL already matches validated store state', async () => {
    vi.useFakeTimers(); mocks.create.mockResolvedValue({ success: false, error: 'current failure' });
    const { result, rerender } = renderHook(({ id }: { id: string | null }) => useRecordingSession({ initialSessionId: id }), { initialProps: { id: null } });
    await act(async () => { await result.current.ensureSession(); });
    expect(result.current.retryState.inCooldown).toBe(true);
    mocks.store.sessionId = 'b'; mocks.store.isValidated = true;
    rerender({ id: 'b' });
    expect(result.current.retryState.inCooldown).toBe(false); expect(vi.getTimerCount()).toBe(0);
    expect(result.current.sessionError).toBeNull();
  });
  it('does not inherit the old profile or viewport when adopting a different already-validated URL session', async () => {
    mocks.create.mockResolvedValue({ success: true, data: { session_id: 'a', session_profile_id: 'profile-a', actual_viewport: { width: 640, height: 480, source: 'requested' } } });
    const { result, rerender } = renderHook(({ id }: { id: string | null }) => useRecordingSession({ initialSessionId: id }), { initialProps: { id: null } });
    await act(async () => { expect(await result.current.ensureSession()).toBe('a'); });
    expect(result.current.sessionProfileId).toBe('profile-a'); expect(result.current.actualViewport?.width).toBe(640);
    mocks.store.sessionId = 'b'; mocks.store.isValidated = true;
    rerender({ id: 'b' });
    expect(result.current.sessionId).toBe('b'); expect(result.current.sessionProfileId).toBeNull(); expect(result.current.actualViewport).toBeNull();
  });
});
