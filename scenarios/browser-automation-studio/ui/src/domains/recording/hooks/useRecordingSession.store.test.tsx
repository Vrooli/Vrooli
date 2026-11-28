import { act, renderHook, waitFor } from '@/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useRecordingSession } from './useRecordingSession';
import { useSessionStore } from '../stores';

const create = vi.hoisted(() => vi.fn());
vi.mock('../api', () => ({ recordingApi: { createSession: create } }));
vi.mock('@/config', () => ({ getApiBase: () => 'http://api.test' }));
function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>(done => { resolve = done; });
  return { promise, resolve };
}
const response = (ok: boolean) => ({ ok, status: ok ? 200 : 404 } as Response);

describe('real recording store and hook share retirement and URL redirect ownership', () => {
  beforeEach(() => { useSessionStore.getState().clearSession(); create.mockReset(); });
  afterEach(() => { vi.unstubAllGlobals(); useSessionStore.getState().clearSession(); });
  it.each([true, false])('keeps local and store B/profile after retired A valid=%s completes', async valid => {
    const a = deferred<Response>();
    const fetch = vi.fn().mockReturnValueOnce(a.promise).mockResolvedValue(response(true)); vi.stubGlobal('fetch', fetch);
    const { result, rerender } = renderHook(({ id }) => useRecordingSession({ initialSessionId: id, initialSessionProfileId: `profile-${id}` }), { initialProps: { id: 'a' } });
    rerender({ id: 'b' });
    await waitFor(() => expect(useSessionStore.getState().isValidated).toBe(true));
    await act(async () => a.resolve(response(valid)));
    expect(result.current.sessionId).toBe('b'); expect(result.current.sessionProfileId).toBe('profile-b');
    expect(useSessionStore.getState().sessionId).toBe('b'); expect(useSessionStore.getState().profileId).toBe('profile-b');
    expect(fetch.mock.calls[0][1].signal.aborted).toBe(true);
  });
  it('retires store validation when its owning hook unmounts', async () => {
    const pending = deferred<Response>(); vi.stubGlobal('fetch', vi.fn().mockReturnValue(pending.promise));
    const { unmount } = renderHook(() => useRecordingSession({ initialSessionId: 'a' }));
    unmount(); await act(async () => pending.resolve(response(true)));
    expect(useSessionStore.getState().sessionId).toBeNull();
    expect(useSessionStore.getState().isValidated).toBe(false); expect(useSessionStore.getState().isValidating).toBe(false);
  });
  it('shares creation then preserves profile/viewport through the normal validated URL redirect without revalidation', async () => {
    const pending = deferred<unknown>(); create.mockReturnValue(pending.promise);
    const fetch = vi.fn(); vi.stubGlobal('fetch', fetch); const ready = vi.fn();
    const { result, rerender } = renderHook(({ id }: { id: string | null }) => useRecordingSession({ initialSessionId: id, onSessionReady: ready }), { initialProps: { id: null } });
    let first!: Promise<string | null>; let second!: Promise<string | null>;
    act(() => { first = result.current.ensureSession(); second = result.current.ensureSession(); });
    await act(async () => pending.resolve({ success: true, data: { session_id: 'created', session_profile_id: 'profile', actual_viewport: { width: 640, height: 480, source: 'requested' } } }));
    expect(await first).toBe('created'); expect(await second).toBe('created'); expect(create).toHaveBeenCalledTimes(1);
    rerender({ id: 'created' });
    expect(result.current.sessionId).toBe('created'); expect(result.current.sessionProfileId).toBe('profile');
    expect(result.current.actualViewport?.width).toBe(640); expect(useSessionStore.getState().actualViewport?.width).toBe(640);
    expect(ready).toHaveBeenCalledTimes(1); expect(fetch).not.toHaveBeenCalled();
  });
  it('retires owned creation loading on unmount while ignoring its late allocation result', async () => {
    const pending = deferred<unknown>(); create.mockReturnValue(pending.promise); const ready = vi.fn();
    const { result, unmount } = renderHook(() => useRecordingSession({ initialSessionId: null, onSessionReady: ready }));
    let creation!: Promise<string | null>; act(() => { creation = result.current.ensureSession(); });
    expect(useSessionStore.getState().isCreating).toBe(true); unmount();
    expect(useSessionStore.getState().isCreating).toBe(false);
    await act(async () => { pending.resolve({ success: true, data: { session_id: 'retired' } }); await creation; });
    expect(await creation).toBeNull(); expect(useSessionStore.getState().sessionId).toBeNull(); expect(ready).not.toHaveBeenCalled();
  });
  it('does not revive the first A validation when URL ownership goes A to B to A', async () => {
    const oldA = deferred<Response>();
    vi.stubGlobal('fetch', vi.fn().mockReturnValueOnce(oldA.promise).mockResolvedValue(response(true)));
    const { result, rerender } = renderHook(({ id }) => useRecordingSession({ initialSessionId: id }), { initialProps: { id: 'a' } });
    rerender({ id: 'b' }); await waitFor(() => expect(useSessionStore.getState().sessionId).toBe('b'));
    rerender({ id: 'a' }); await waitFor(() => expect(useSessionStore.getState().sessionId).toBe('a'));
    await act(async () => oldA.resolve(response(false)));
    expect(result.current.sessionId).toBe('a'); expect(useSessionStore.getState().sessionId).toBe('a');
    expect(useSessionStore.getState().isValidated).toBe(true); expect(useSessionStore.getState().isValidating).toBe(false);
  });
});
