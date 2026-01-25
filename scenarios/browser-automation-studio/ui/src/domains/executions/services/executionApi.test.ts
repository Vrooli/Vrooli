import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('@/config', () => ({
  getConfig: vi.fn(() => Promise.resolve({ API_URL: 'http://localhost:8080' })),
}));

import { fetchExecutionsList } from './executionApi';

describe('executionApi', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    global.fetch = vi.fn();
  });

  it('returns a validated execution list', async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        executions: [
          {
            execution_id: 'exec-1',
            workflow_id: 'workflow-1',
            status: 'running',
            started_at: '2024-01-01T00:00:00.000Z',
          },
        ],
      }),
    } as Response);

    const executions = await fetchExecutionsList(5);
    expect(executions).toHaveLength(1);
    expect(executions[0]?.execution_id).toBe('exec-1');
  });

  it('throws when execution list validation fails', async () => {
    vi.mocked(global.fetch).mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        executions: [{ execution_id: 42 }],
      }),
    } as Response);

    await expect(fetchExecutionsList()).rejects.toThrow('Invalid ExecutionsList response');
  });
});
