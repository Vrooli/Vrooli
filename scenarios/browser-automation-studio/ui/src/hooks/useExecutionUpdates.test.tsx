import { renderHook } from '@/test-utils';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useExecutionUpdates } from './useExecutionUpdates';

type Message = {
  type: string;
  execution_id?: string;
  data?: { kind?: string | number };
};

const mocks = vi.hoisted(() => ({
  onMessage: undefined as ((message: Message) => void) | undefined,
  fetchRunningExecutions: vi.fn(),
  fetchRecentExecutions: vi.fn(),
  refreshTimeline: vi.fn(),
  currentExecution: { id: 'execution-1' },
}));

vi.mock('@/contexts/WebSocketContext', () => ({
  useWebSocketMessage: (callback: (message: Message) => void) => {
    mocks.onMessage = callback;
  },
}));

vi.mock('@stores/dashboardStore', () => ({
  useDashboardStore: () => ({
    fetchRunningExecutions: mocks.fetchRunningExecutions,
    fetchRecentExecutions: mocks.fetchRecentExecutions,
  }),
}));

vi.mock('@/domains/executions/store', () => ({
  useExecutionStore: () => ({
    refreshTimeline: mocks.refreshTimeline,
    currentExecution: mocks.currentExecution,
  }),
}));

describe('useExecutionUpdates timeline fan-out', () => {
  beforeEach(() => {
    mocks.fetchRunningExecutions.mockReset();
    mocks.fetchRecentExecutions.mockReset();
    mocks.refreshTimeline.mockReset();
    mocks.onMessage = undefined;
    mocks.currentExecution = { id: 'execution-1' };
  });

  it('does not refetch full timeline for incremental step or envelope events', () => {
    renderHook(() => useExecutionUpdates());
    const onMessage = mocks.onMessage;
    expect(onMessage).toBeDefined();

    for (const type of ['step_started', 'step_completed', 'step_failed', 'progress']) {
      onMessage?.({ type, execution_id: 'execution-1' });
    }
    for (const kind of ['step.started', 'step.completed', 'step.failed', 'step.progress', 2, 3, 4, 5]) {
      onMessage?.({ type: 'envelope', execution_id: 'execution-1', data: { kind } });
    }

    expect(mocks.refreshTimeline).not.toHaveBeenCalled();
    expect(mocks.fetchRunningExecutions).not.toHaveBeenCalled();
    expect(mocks.fetchRecentExecutions).not.toHaveBeenCalled();
  });

  it('refreshes the selected execution once on a terminal event', () => {
    renderHook(() => useExecutionUpdates());

    mocks.onMessage?.({ type: 'execution_completed', execution_id: 'execution-1' });

    expect(mocks.refreshTimeline).toHaveBeenCalledTimes(1);
    expect(mocks.refreshTimeline).toHaveBeenCalledWith('execution-1');
    expect(mocks.fetchRunningExecutions).toHaveBeenCalledTimes(1);
    expect(mocks.fetchRecentExecutions).toHaveBeenCalledTimes(1);
  });

  it('refreshes the selected execution once on a terminal envelope', () => {
    renderHook(() => useExecutionUpdates());

    mocks.onMessage?.({
      type: 'envelope',
      execution_id: 'execution-1',
      data: { kind: 'execution.completed' },
    });

    expect(mocks.refreshTimeline).toHaveBeenCalledTimes(1);
    expect(mocks.refreshTimeline).toHaveBeenCalledWith('execution-1');
    expect(mocks.fetchRunningExecutions).toHaveBeenCalledTimes(1);
    expect(mocks.fetchRecentExecutions).toHaveBeenCalledTimes(1);
  });

  it('keeps envelope status updates on dashboard reconciliation without timeline fan-out', () => {
    renderHook(() => useExecutionUpdates());

    mocks.onMessage?.({ type: 'envelope', execution_id: 'execution-1', data: { kind: 1 } });

    expect(mocks.refreshTimeline).not.toHaveBeenCalled();
    expect(mocks.fetchRunningExecutions).toHaveBeenCalledTimes(1);
    expect(mocks.fetchRecentExecutions).toHaveBeenCalledTimes(1);
  });
});
