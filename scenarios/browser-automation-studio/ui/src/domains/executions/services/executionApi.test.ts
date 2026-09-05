import { beforeEach, describe, expect, it, vi } from 'vitest';
import { create } from '@bufbuild/protobuf';
import { ExecutionSchema } from '@vrooli/proto-types/browser-automation-studio/v1/execution/execution_pb';
import { ExecutionStatus } from '@vrooli/proto-types/browser-automation-studio/v1/base/shared_pb';
import { ListExecutionsResponseSchema } from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';

vi.mock('@/config', () => ({
  getConfig: vi.fn(() => Promise.resolve({ API_URL: 'http://localhost:8080' })),
}));

const listExecutionsMock = vi.fn();

vi.mock('@/api/executions', () => ({
  executionsClient: {
    listExecutions: (...args: unknown[]) => listExecutionsMock(...args),
  },
}));

import { fetchExecutionsList } from './executionApi';

describe('executionApi', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('returns a validated execution list', async () => {
    listExecutionsMock.mockResolvedValueOnce(create(ListExecutionsResponseSchema, {
      executions: [create(ExecutionSchema, {
        executionId: 'exec-1',
        workflowId: 'workflow-1',
        status: ExecutionStatus.RUNNING,
      })],
    }));

    const executions = await fetchExecutionsList(5);
    expect(executions).toHaveLength(1);
    expect(executions[0]?.execution_id).toBe('exec-1');
    expect(listExecutionsMock).toHaveBeenCalledWith(expect.objectContaining({ limit: 5 }));
  });

  it('propagates typed transport failures', async () => {
    listExecutionsMock.mockRejectedValueOnce(new Error('transport unavailable'));

    await expect(fetchExecutionsList()).rejects.toThrow('transport unavailable');
  });
});
