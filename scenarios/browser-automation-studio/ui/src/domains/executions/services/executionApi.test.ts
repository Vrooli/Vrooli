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

import { fetchExecutionsList, listExecutionsViaApi } from './executionApi';

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
  it('preserves large caller limits with bounded pages', async () => {
    const page = (start: number, count: number) => create(ListExecutionsResponseSchema, {
      executions: Array.from({ length: count }, (_, i) => create(ExecutionSchema, { executionId: `exec-${start+i}` })),
      total: 250, hasMore: true,
    });
    listExecutionsMock.mockResolvedValueOnce(page(0,100)).mockResolvedValueOnce(page(100,100));
    const executions = await fetchExecutionsList(200);
    expect(executions).toHaveLength(200);
    expect(executions[199]?.execution_id).toBe('exec-199');
    expect(listExecutionsMock.mock.calls.map(([req]) => [req.limit, req.offset])).toEqual([[100,0],[100,100]]);
  });

  it('retains complete workflow history and exportability across pages', async () => {
    listExecutionsMock.mockResolvedValueOnce(create(ListExecutionsResponseSchema, {
      executions: [create(ExecutionSchema, { executionId: 'a' })], total: 2, hasMore: true,
      exportability: { a: { hasTimeline: true } },
    })).mockResolvedValueOnce(create(ListExecutionsResponseSchema, {
      executions: [create(ExecutionSchema, { executionId: 'b' })], total: 2,
      exportability: { b: { hasRecordedVideo: true } },
    }));
    const response = await listExecutionsViaApi({ workflowId: 'workflow', includeExportability: true });
    expect(response.executions.map(e => e.executionId)).toEqual(['a','b']);
    expect(response.exportability.a.hasTimeline).toBe(true);
    expect(response.exportability.b.hasRecordedVideo).toBe(true);
    expect(listExecutionsMock.mock.calls.map(([req]) => [req.workflowId,req.includeExportability,req.offset])).toEqual([['workflow',true,0],['workflow',true,1]]);
  });

  it.each(['empty','duplicate'])('rejects a %s continuation instead of presenting partial history', async (fault) => {
    listExecutionsMock.mockResolvedValueOnce(create(ListExecutionsResponseSchema, {
      executions: [create(ExecutionSchema, { executionId: 'a' })], total: 3, hasMore: true,
    })).mockResolvedValueOnce(create(ListExecutionsResponseSchema, {
      executions: fault === 'empty' ? [] : [create(ExecutionSchema, { executionId: 'a' })], total: 3, hasMore: true,
    }));
    await expect(listExecutionsViaApi()).rejects.toThrow(/history changed|pagination/i);
  });

});
