import { create, toJson } from '@bufbuild/protobuf';
import { ExecutionStatus } from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
import { ExecutionSchema, GetScreenshotsResponseSchema } from '@vrooli/proto-types/browser-automation-studio/v1/execution/execution_pb';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  getExecution: vi.fn(),
  getScreenshots: vi.fn(),
  getTimeline: vi.fn(),
  stopExecution: vi.fn(),
}));

vi.mock('@/domains/executions/services/executionApi', () => ({
  executionMsgToJson: (value: unknown) => {
    if (!value || typeof value !== 'object' || !('$typeName' in value)) return value;
    return toJson(ExecutionSchema, value as never, { useProtoFieldName: true });
  },
  getExecutionViaApi: mocks.getExecution,
  getExecutionScreenshotsViaApi: mocks.getScreenshots,
  getExecutionTimelineViaApi: mocks.getTimeline,
  listExecutionsViaApi: vi.fn(),
  stopExecutionViaApi: mocks.stopExecution,
  exportabilityToLegacy: vi.fn(),
}));

vi.mock('@/domains/workflows/services/workflowApi', () => ({
  executeWorkflowViaApi: vi.fn(),
  getWorkflowViaApi: vi.fn(),
}));

import { useExecutionStore } from './store';

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => { resolve = done; });
  return { promise, resolve };
}

describe('execution detail load ownership', () => {
  beforeEach(() => {
    useExecutionStore.getState().closeViewer();
    mocks.getExecution.mockReset();
    mocks.stopExecution.mockReset();
    mocks.stopExecution.mockResolvedValue(undefined);
    mocks.getScreenshots.mockReset();
    mocks.getScreenshots.mockResolvedValue(create(GetScreenshotsResponseSchema, {}));
    mocks.getTimeline.mockReset();
    mocks.getTimeline.mockResolvedValue({ entries: [], logs: [] });
  });

  it('prevents a stale execution response from replacing the latest selection', async () => {
    const first = deferred<unknown>();
    const second = deferred<unknown>();
    mocks.getExecution.mockReturnValueOnce(first.promise).mockReturnValueOnce(second.promise);

    const loadFirst = useExecutionStore.getState().loadExecution('execution-first');
    const loadSecond = useExecutionStore.getState().loadExecution('execution-second');
    second.resolve({ executionId: 'execution-second', workflowId: 'workflow-second' });
    await loadSecond;
    first.resolve({ executionId: 'execution-first', workflowId: 'workflow-first' });
    await loadFirst;

    expect(useExecutionStore.getState().currentExecution?.id).toBe('execution-second');
    expect(useExecutionStore.getState().viewerWorkflowId).toBe('workflow-second');
  });

  it('does not restore a viewer after it closes during detail loading', async () => {
    const pending = deferred<unknown>();
    mocks.getExecution.mockReturnValue(pending.promise);

    const load = useExecutionStore.getState().loadExecution('execution-closed');
    useExecutionStore.getState().closeViewer();
    pending.resolve({ executionId: 'execution-closed', workflowId: 'workflow-closed' });
    await load;

    expect(useExecutionStore.getState().currentExecution).toBeNull();
    expect(useExecutionStore.getState().viewerWorkflowId).toBeNull();
  });

  it('uses the persisted outcome when normal completion wins a cancel race', async () => {
    useExecutionStore.setState({
      currentExecution: {
        id: 'execution-race', workflowId: 'workflow', status: 'running', startedAt: new Date(),
        screenshots: [], timeline: [], logs: [], progress: 0,
      },
    });
    const completed = create(ExecutionSchema, {
      executionId: 'execution-race', workflowId: 'workflow', status: ExecutionStatus.COMPLETED,
      currentStep: 'finalize', progress: 100,
    });
    mocks.getExecution.mockResolvedValue(completed);
    mocks.getTimeline.mockResolvedValue({ entries: [], logs: [], status: 'EXECUTION_STATUS_COMPLETED' });

    await useExecutionStore.getState().stopExecution('execution-race');

    expect(useExecutionStore.getState().currentExecution?.status).toBe('completed');
    expect(useExecutionStore.getState().currentExecution?.currentStep).toBe('finalize');
    expect(mocks.stopExecution).toHaveBeenCalledOnce();
    expect(mocks.getExecution).toHaveBeenCalledWith('execution-race');
  });

  it('uses the persisted cancellation result and refreshes final artifacts', async () => {
    useExecutionStore.setState({
      currentExecution: {
        id: 'execution-cancel', workflowId: 'workflow', status: 'running', startedAt: new Date(),
        screenshots: [], timeline: [], logs: [], progress: 30,
      },
    });
    mocks.getExecution.mockResolvedValue(create(ExecutionSchema, {
      executionId: 'execution-cancel', workflowId: 'workflow', status: ExecutionStatus.CANCELLED,
      currentStep: 'cleanup', error: 'execution cancelled', progress: 30,
    }));
    mocks.getTimeline.mockResolvedValue({ entries: [], logs: [], status: 'EXECUTION_STATUS_CANCELLED' });

    await useExecutionStore.getState().stopExecution('execution-cancel');

    const current = useExecutionStore.getState().currentExecution;
    expect(current?.status).toBe('cancelled');
    expect(current?.error).toBe('execution cancelled');
    expect(mocks.getTimeline).toHaveBeenCalledWith('execution-cancel');

    await useExecutionStore.getState().stopExecution('execution-cancel');
    expect(useExecutionStore.getState().currentExecution?.status).toBe('cancelled');
    expect(mocks.stopExecution).toHaveBeenCalledTimes(2);
  });

  it('keeps the active UI state when the API owner rejects cancellation', async () => {
    useExecutionStore.setState({
      currentExecution: {
        id: 'execution-close-error', workflowId: 'workflow', status: 'running', startedAt: new Date(),
        screenshots: [], timeline: [], logs: [], progress: 20,
      },
    });
    mocks.stopExecution.mockRejectedValue(new Error('driver close failed'));

    await expect(useExecutionStore.getState().stopExecution('execution-close-error')).rejects.toThrow('driver close failed');

    expect(useExecutionStore.getState().currentExecution?.status).toBe('running');
    expect(mocks.getExecution).not.toHaveBeenCalled();
  });

  it('does not let an old stop response replace the newly selected execution', async () => {
    useExecutionStore.setState({
      currentExecution: {
        id: 'execution-current', workflowId: 'workflow-current', status: 'running', startedAt: new Date(),
        screenshots: [], timeline: [], logs: [], progress: 0,
      },
    });
    mocks.getExecution.mockResolvedValue(create(ExecutionSchema, {
      executionId: 'execution-old', workflowId: 'workflow-old', status: ExecutionStatus.CANCELLED,
    }));

    await useExecutionStore.getState().stopExecution('execution-old');

    expect(useExecutionStore.getState().currentExecution?.id).toBe('execution-current');
    expect(useExecutionStore.getState().currentExecution?.status).toBe('running');
  });
});
