import { renderHook } from '@/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useExecutionEvents } from './useExecutionEvents';

const mocks = vi.hoisted(() => ({
  connected: true,
  send: vi.fn(),
  store: {
    updateExecutionStatus: vi.fn(),
    updateProgress: vi.fn(),
    addLog: vi.fn(),
    addScreenshot: vi.fn(),
    recordHeartbeat: vi.fn(),
    refreshTimeline: vi.fn(),
    loadExecution: vi.fn(),
  },
}));

vi.mock('@/contexts/WebSocketContext', () => ({
  useWebSocket: () => ({ isConnected: mocks.connected, send: mocks.send }),
  useWebSocketMessage: () => undefined,
}));

vi.mock('../store', () => ({
  useExecutionStore: (select: (state: typeof mocks.store) => unknown) => select(mocks.store),
}));

describe('useExecutionEvents subscription recovery', () => {
  beforeEach(() => {
    mocks.connected = true;
    mocks.send.mockReset();
    mocks.store.refreshTimeline.mockReset();
    mocks.store.loadExecution.mockReset();
  });

  afterEach(() => vi.useRealTimers());

  it('resubscribes to the active execution after the websocket reconnects', () => {
    const execution = { id: 'execution-1', status: 'running' as const };
    const { rerender, unmount } = renderHook(
      ({ connected }) => {
        mocks.connected = connected;
        return useExecutionEvents(execution);
      },
      { initialProps: { connected: true } },
    );

    expect(mocks.send).toHaveBeenCalledWith({ type: 'subscribe', execution_id: 'execution-1' });

    rerender({ connected: false });
    rerender({ connected: true });

    expect(mocks.send.mock.calls.filter(([message]) => message.type === 'subscribe')).toEqual([
      [{ type: 'subscribe', execution_id: 'execution-1' }],
      [{ type: 'subscribe', execution_id: 'execution-1' }],
    ]);
    expect(mocks.store.loadExecution).toHaveBeenCalledTimes(1);
    expect(mocks.store.loadExecution).toHaveBeenCalledWith('execution-1');
    unmount();
  });

  it('reconciles the active timeline at a bounded one second cadence', async () => {
    vi.useFakeTimers();
    const { unmount } = renderHook(() =>
      useExecutionEvents({ id: 'execution-1', status: 'running' }),
    );
    expect(mocks.store.refreshTimeline).not.toHaveBeenCalled();

    await vi.advanceTimersByTimeAsync(999);
    expect(mocks.store.refreshTimeline).not.toHaveBeenCalled();
    await vi.advanceTimersByTimeAsync(1);
    expect(mocks.store.refreshTimeline).toHaveBeenCalledTimes(1);
    expect(mocks.store.refreshTimeline).toHaveBeenCalledWith('execution-1');

    unmount();
  });
});
