import { create } from '@bufbuild/protobuf';
import { GetScreenshotsResponseSchema } from '@vrooli/proto-types/browser-automation-studio/v1/execution/execution_pb';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
  getExecution: vi.fn(),
  getScreenshots: vi.fn(),
  getTimeline: vi.fn(),
}));

vi.mock('@/domains/executions/services/executionApi', () => ({
  executionMsgToJson: (value: unknown) => value,
  getExecutionViaApi: mocks.getExecution,
  getExecutionScreenshotsViaApi: mocks.getScreenshots,
  getExecutionTimelineViaApi: mocks.getTimeline,
  listExecutionsViaApi: vi.fn(),
  stopExecutionViaApi: vi.fn(),
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
});
