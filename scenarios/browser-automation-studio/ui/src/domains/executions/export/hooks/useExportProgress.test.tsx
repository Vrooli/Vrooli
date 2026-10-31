import { act, renderHook, waitFor } from '@/test-utils';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useExportProgress } from './useExportProgress';
const mocks = vi.hoisted(() => ({ connected: true, send: vi.fn(), listener: (_message: unknown) => {}, status: vi.fn() }));
vi.mock('@/contexts/WebSocketContext', () => ({
  useWebSocket: () => ({ isConnected: mocks.connected, send: mocks.send }),
  useWebSocketMessage: (callback: typeof mocks.listener) => { mocks.listener = callback; },
}));
vi.mock('../api/executeExport', () => ({ getExportStatus: mocks.status }));
const status = (id: string, state = 'processing') => ({ export_id: id, execution_id: 'execution', status: state, storage_url: '/complete.mp4' });
describe('export persisted status recovery', () => {
  beforeEach(() => { mocks.connected = true; mocks.send.mockReset(); mocks.status.mockReset(); });
  it('recovers a missed terminal event on reconnect and invokes completion once', async () => {
    mocks.status.mockResolvedValue(status('a'));
    const complete = vi.fn();
    const { result, rerender } = renderHook(() => useExportProgress({ exportId: 'a', onComplete: complete }));
    await waitFor(() => expect(result.current.progress?.status).toBe('processing'));
    mocks.connected = false; rerender();
    mocks.status.mockResolvedValue(status('a', 'completed'));
    mocks.connected = true; rerender();
    await waitFor(() => expect(complete).toHaveBeenCalledTimes(1));
    act(() => mocks.listener({ type: 'export_progress', ...status('a', 'completed') }));
    act(() => mocks.listener({ type: 'export_progress', ...status('a') }));
    expect(complete).toHaveBeenCalledTimes(1);
    expect(result.current.progress?.status).toBe('completed');
  });
  it('rejects an old export response after identity changes', async () => {
    let resolve!: (value: unknown) => void;
    mocks.status.mockImplementationOnce(() => new Promise(r => { resolve = r; })).mockResolvedValue(status('b'));
    const complete = vi.fn();
    const { result, rerender } = renderHook(({ id }) => useExportProgress({ exportId: id, onComplete: complete }), { initialProps: { id: 'a' } });
    rerender({ id: 'b' });
    await waitFor(() => expect(result.current.progress?.export_id).toBe('b'));
    await act(async () => resolve(status('a', 'completed')));
    expect(result.current.progress?.export_id).toBe('b'); expect(complete).not.toHaveBeenCalled();
  });
  it('retains newer terminal events over an in-flight status response', async () => {
    let resolve!: (value: unknown) => void;
    mocks.status.mockImplementation(() => new Promise(r => { resolve = r; }));
    const { result } = renderHook(() => useExportProgress({ exportId: 'a' }));
    act(() => mocks.listener({ type: 'export_progress', ...status('a', 'completed') }));
    await act(async () => resolve(status('a')));
    expect(result.current.progress?.status).toBe('completed');
  });
  it('exposes recovery errors without inventing terminal failure', async () => {
    mocks.status.mockRejectedValue(new Error('status unavailable'));
    const error = vi.fn();
    const { result } = renderHook(() => useExportProgress({ exportId: 'a', onError: error }));
    await waitFor(() => expect(result.current.statusError).toBe('status unavailable'));
    expect(result.current.progress).toBeNull(); expect(error).not.toHaveBeenCalled();
  });
  it('does not repeat completion after the viewer clears its active export ID', async () => {
    mocks.status.mockResolvedValue(status('a'));
    const complete = vi.fn();
    const { rerender } = renderHook(({ id }: {id: string | null}) => useExportProgress({ exportId: id, executionId: id ? undefined : 'execution', onComplete: complete }), {initialProps: {id: 'a' as string | null}});
    await waitFor(() => expect(mocks.status).toHaveBeenCalled());
    act(() => mocks.listener({type: 'export_progress', ...status('a', 'completed')}));
    rerender({id: null});
    act(() => mocks.listener({type: 'export_progress', ...status('a', 'completed')}));
    expect(complete).toHaveBeenCalledTimes(1);
  });

  it('rehydrates completed progress after reset while suppressing duplicate completion callbacks', async () => {
    mocks.status.mockResolvedValue(status('a', 'completed'));
    const complete = vi.fn();
    const { result } = renderHook(() => useExportProgress({ exportId: 'a', onComplete: complete }));
    await waitFor(() => expect(result.current.progress?.status).toBe('completed'));
    act(() => result.current.reset()); expect(result.current.progress).toBeNull();
    await act(async () => result.current.checkStatus());
    expect(result.current.progress?.status).toBe('completed');
    expect(complete).toHaveBeenCalledTimes(1);
  });
  it('rehydrates a revisited completed export without replaying completion side effects', async () => {
    mocks.status.mockImplementation(async (id: string) => status(id, 'completed'));
    const complete = vi.fn();
    const { result, rerender } = renderHook(({ id }) => useExportProgress({ exportId: id, onComplete: complete }), { initialProps: { id: 'a' } });
    await waitFor(() => expect(result.current.progress?.export_id).toBe('a'));
    rerender({ id: 'b' }); await waitFor(() => expect(result.current.progress?.export_id).toBe('b'));
    rerender({ id: 'a' }); await act(async () => {});
    expect(result.current.progress?.export_id).toBe('a');
    expect(result.current.progress?.status).toBe('completed');
    expect(complete).toHaveBeenCalledTimes(2);
  });
});
