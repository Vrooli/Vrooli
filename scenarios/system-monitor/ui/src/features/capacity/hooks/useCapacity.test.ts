import { renderHook, act, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useCapacity } from './useCapacity';

const mocks = vi.hoisted(() => ({
  fetchCapacityOverview: vi.fn(), fetchCapacityReconciliation: vi.fn(), fetchCapacityPolicy: vi.fn(), setCapacityPolicy: vi.fn(), usePolling: vi.fn(),
}));
vi.mock('../api', () => ({
  fetchCapacityOverview: mocks.fetchCapacityOverview,
  fetchCapacityReconciliation: mocks.fetchCapacityReconciliation,
  fetchCapacityPolicy: mocks.fetchCapacityPolicy,
  setCapacityPolicy: mocks.setCapacityPolicy,
}));
vi.mock('../../../shared/hooks/usePolling', () => ({ usePolling: mocks.usePolling }));
vi.mock('../../../shared/api/apiFetch', () => ({ extractErrorMessage: (error: unknown, fallback: string) => error instanceof Error ? error.message : fallback }));

describe('useCapacity', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.fetchCapacityOverview.mockResolvedValue({ gpus: [] });
    mocks.fetchCapacityReconciliation.mockResolvedValue({ findings: [] });
    mocks.fetchCapacityPolicy.mockResolvedValue([{ key: 'gpu', value: 'auto' }]);
    mocks.setCapacityPolicy.mockResolvedValue([{ key: 'gpu', value: 'manual' }]);
  });

  it('loads state, tolerates reconciliation failure, and saves policy', async () => {
    mocks.fetchCapacityReconciliation.mockRejectedValueOnce(new Error('reconcile unavailable'));
    const { result } = renderHook(() => useCapacity(1000));
    await waitFor(() => { expect(result.current.overview).toEqual({ gpus: [] }); });
    expect(result.current.reconciliation).toBeNull();
    expect(mocks.usePolling).toHaveBeenCalledWith(expect.any(Function), 1000, true);
    await act(async () => { await result.current.savePolicy('gpu', 'manual'); });
    expect(result.current.policy).toEqual([{ key: 'gpu', value: 'manual' }]);
  });

  it('surfaces load, policy, and abort failures', async () => {
    mocks.fetchCapacityOverview.mockRejectedValueOnce(new Error('capacity failed'));
    const { result } = renderHook(() => useCapacity());
    await waitFor(() => { expect(result.current.error).toBe('capacity failed'); });
    mocks.setCapacityPolicy.mockRejectedValueOnce('policy failed');
    await act(async () => { await result.current.savePolicy('gpu', 'bad'); });
    expect(result.current.policyError).toBe('Failed to update gpu');
    mocks.fetchCapacityOverview.mockRejectedValueOnce(new DOMException('cancelled', 'AbortError'));
    await act(async () => { await result.current.refresh(); });
    expect(result.current.error).toBe('capacity failed');
  });
});
