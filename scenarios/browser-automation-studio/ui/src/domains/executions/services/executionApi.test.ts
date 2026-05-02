import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fetchJsonResponse, installFetchMock, type FetchMock } from '@/test-utils';

vi.mock('@/config', () => ({
  getConfig: vi.fn(() => Promise.resolve({ API_URL: 'http://localhost:8080' })),
}));

import { fetchExecutionsList } from './executionApi';

describe('executionApi', () => {
  let fetchMock: FetchMock;

  beforeEach(() => {
    vi.clearAllMocks();
    fetchMock = installFetchMock();
  });

  it('returns a validated execution list', async () => {
    fetchMock.mockResolvedValueOnce(
      fetchJsonResponse({
        executions: [
          {
            execution_id: 'exec-1',
            workflow_id: 'workflow-1',
            status: 'running',
            started_at: '2024-01-01T00:00:00.000Z',
          },
        ],
      })
    );

    const executions = await fetchExecutionsList(5);
    expect(executions).toHaveLength(1);
    expect(executions[0]?.execution_id).toBe('exec-1');
  });

  it('throws when execution list validation fails', async () => {
    fetchMock.mockResolvedValueOnce(
      fetchJsonResponse({
        executions: [{ execution_id: 42 }],
      })
    );

    await expect(fetchExecutionsList()).rejects.toThrow('Invalid ExecutionsList response');
  });
});
