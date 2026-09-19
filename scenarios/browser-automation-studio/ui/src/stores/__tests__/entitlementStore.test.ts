/**
 * EntitlementStore Test Suite (Connect-RPC)
 *
 * Tests entitlement store functionality after migration to EntitlementService
 * Connect-RPC client. Mocks the generated client; verifies that the store
 * adapts proto responses into its snake_case consumer shape and routes
 * actions to the right RPC.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { act } from '@testing-library/react';
import { ConnectError, Code } from '@connectrpc/connect';

// ----------------------------------------------------------------------------
// Client mock — installed BEFORE importing the store.
// ----------------------------------------------------------------------------

const getStatusMock = vi.fn();
const getIdentityMock = vi.fn();
const setIdentityMock = vi.fn();
const clearIdentityMock = vi.fn();
const refreshStatusMock = vi.fn();
const getUsageMock = vi.fn();
const getUsageHistoryMock = vi.fn();
const getOperationLogMock = vi.fn();

vi.mock('../../api/entitlement', () => ({
  entitlementClient: {
    getStatus: (...a: unknown[]) => getStatusMock(...a),
    getIdentity: (...a: unknown[]) => getIdentityMock(...a),
    setIdentity: (...a: unknown[]) => setIdentityMock(...a),
    clearIdentity: (...a: unknown[]) => clearIdentityMock(...a),
    refreshStatus: (...a: unknown[]) => refreshStatusMock(...a),
    getUsage: (...a: unknown[]) => getUsageMock(...a),
    getUsageHistory: (...a: unknown[]) => getUsageHistoryMock(...a),
    getOperationLog: (...a: unknown[]) => getOperationLogMock(...a),
  },
}));

import { useEntitlementStore, isValidEmail } from '../entitlementStore';

const makeProtoStatus = (overrides: Record<string, unknown> = {}) => ({
  userIdentity: 'alice@example.com',
  status: 'active',
  tier: 'pro',
  isActive: true,
  features: ['ai', 'recording'],
  featureAccess: [
    { id: 'ai', label: 'AI', description: 'AI features', requiredTier: 'pro', hasAccess: true },
  ],
  monthlyLimit: 100,
  monthlyUsed: 5,
  monthlyRemaining: 95,
  requiresWatermark: false,
  canUseAi: true,
  canUseRecording: true,
  entitlementsEnabled: true,
  aiCreditsUsed: 5,
  aiCreditsLimit: 100,
  aiCreditsRemaining: 95,
  aiRequestsCount: 2,
  aiResetDate: '2026-06-01',
  ...overrides,
});

const resetStore = () => {
  useEntitlementStore.setState({
    userEmail: '',
    status: null,
    isLoading: false,
    error: null,
    lastFetched: null,
    isOffline: false,
    usageHistory: [],
    historyLoading: false,
    selectedPeriod: null,
    operationLog: [],
    operationLogLoading: false,
    operationLogTotal: 0,
    operationLogHasMore: false,
  });
};

describe('entitlementStore (Connect-RPC)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    for (const m of [
      getStatusMock, getIdentityMock, setIdentityMock, clearIdentityMock, refreshStatusMock,
      getUsageMock, getUsageHistoryMock, getOperationLogMock,
    ]) {
      m.mockReset();
    }
    resetStore();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('isValidEmail', () => {
    it('accepts well-formed addresses', () => {
      expect(isValidEmail('a@example.com')).toBe(true);
      expect(isValidEmail('  foo@bar.co  ')).toBe(true);
    });
    it('rejects malformed addresses', () => {
      expect(isValidEmail('')).toBe(false);
      expect(isValidEmail('plain')).toBe(false);
      expect(isValidEmail('@example.com')).toBe(false);
      expect(isValidEmail('foo@')).toBe(false);
      expect(isValidEmail('foo@bar')).toBe(false);
      expect(isValidEmail('foo@bar.')).toBe(false);
    });
  });

  describe('fetchStatus', () => {
    it('adapts proto status into snake_case store shape', async () => {
      getStatusMock.mockResolvedValueOnce({ status: makeProtoStatus() });
      await act(async () => {
        await useEntitlementStore.getState().fetchStatus();
      });
      const { status, userEmail, isLoading, error, isOffline } = useEntitlementStore.getState();
      expect(status?.user_identity).toBe('alice@example.com');
      expect(status?.tier).toBe('pro');
      expect(status?.can_use_ai).toBe(true);
      expect(status?.monthly_remaining).toBe(95);
      expect(status?.ai_reset_date).toBe('2026-06-01');
      expect(status?.feature_access?.[0]?.required_tier).toBe('pro');
      expect(userEmail).toBe('alice@example.com');
      expect(isLoading).toBe(false);
      expect(error).toBeNull();
      expect(isOffline).toBe(false);
    });

    it('marks offline on Unavailable Connect error', async () => {
      getStatusMock.mockRejectedValueOnce(new ConnectError('upstream', Code.Unavailable));
      await act(async () => {
        await useEntitlementStore.getState().fetchStatus();
      });
      const { isOffline, error, isLoading } = useEntitlementStore.getState();
      expect(isOffline).toBe(true);
      expect(error).toMatch(/upstream/);
      expect(isLoading).toBe(false);
    });

  });

  describe('setUserEmail', () => {
    it('rejects invalid email without calling client', async () => {
      await act(async () => {
        await useEntitlementStore.getState().setUserEmail('not-an-email');
      });
      expect(setIdentityMock).not.toHaveBeenCalled();
      expect(useEntitlementStore.getState().error).toBeTruthy();
    });

    it('rejects empty email without calling client', async () => {
      await act(async () => {
        await useEntitlementStore.getState().setUserEmail('   ');
      });
      expect(setIdentityMock).not.toHaveBeenCalled();
      expect(useEntitlementStore.getState().error).toBe('Email is required');
    });

    it('persists status when client succeeds', async () => {
      setIdentityMock.mockResolvedValueOnce({ status: makeProtoStatus({ userIdentity: 'bob@example.com' }) });
      await act(async () => {
        await useEntitlementStore.getState().setUserEmail('Bob@Example.com');
      });
      expect(setIdentityMock).toHaveBeenCalledWith({ email: 'bob@example.com' });
      expect(useEntitlementStore.getState().userEmail).toBe('bob@example.com');
    });

    it('surfaces client error', async () => {
      setIdentityMock.mockRejectedValueOnce(new ConnectError('bad email', Code.InvalidArgument));
      await act(async () => {
        await useEntitlementStore.getState().setUserEmail('foo@bar.co');
      });
      expect(useEntitlementStore.getState().error).toMatch(/bad email/);
    });
  });

  describe('clearUserEmail', () => {
    it('clears state on success', async () => {
      useEntitlementStore.setState({ userEmail: 'a@b.com' });
      clearIdentityMock.mockResolvedValueOnce({});
      await act(async () => {
        await useEntitlementStore.getState().clearUserEmail();
      });
      expect(clearIdentityMock).toHaveBeenCalled();
      const s = useEntitlementStore.getState();
      expect(s.userEmail).toBe('');
      expect(s.status).toBeNull();
    });
  });

  describe('refreshEntitlement', () => {
    it('falls back to fetchStatus when no current user is known', async () => {
      getStatusMock.mockResolvedValueOnce({ status: makeProtoStatus({ userIdentity: 'anonymous' }) });
      await act(async () => {
        await useEntitlementStore.getState().refreshEntitlement();
      });
      expect(refreshStatusMock).not.toHaveBeenCalled();
      expect(getStatusMock).toHaveBeenCalled();
    });

    it('invokes refreshStatus with stored user', async () => {
      useEntitlementStore.setState({ userEmail: 'alice@example.com' });
      refreshStatusMock.mockResolvedValueOnce({ status: makeProtoStatus() });
      await act(async () => {
        await useEntitlementStore.getState().refreshEntitlement();
      });
      expect(refreshStatusMock).toHaveBeenCalledWith({ user: 'alice@example.com' });
    });
  });

  describe('getUserEmail', () => {
    it('returns stored email and updates state', async () => {
      getIdentityMock.mockResolvedValueOnce({ email: 'cached@example.com' });
      let result: string = '';
      await act(async () => {
        result = await useEntitlementStore.getState().getUserEmail();
      });
      expect(result).toBe('cached@example.com');
      expect(useEntitlementStore.getState().userEmail).toBe('cached@example.com');
    });

    it('returns empty string on error', async () => {
      getIdentityMock.mockRejectedValueOnce(new Error('nope'));
      let result: string = 'unset';
      await act(async () => {
        result = await useEntitlementStore.getState().getUserEmail();
      });
      expect(result).toBe('');
    });
  });

  describe('fetchUsageHistory', () => {
    it('adapts proto periods to snake_case shape', async () => {
      getUsageHistoryMock.mockResolvedValueOnce({
        periods: [{
          billingMonth: '2026-05',
          totalCreditsUsed: 42,
          totalOperations: 7,
          byOperation: { 'ai.workflow_generate': 15 },
          operationCounts: { 'ai.workflow_generate': 3 },
          creditsLimit: 100,
          creditsRemaining: 58,
          periodStart: { seconds: 1746057600n, nanos: 0 },
          periodEnd: undefined,
          resetDate: undefined,
        }],
        hasMore: true,
      });
      await act(async () => {
        await useEntitlementStore.getState().fetchUsageHistory(3, 1);
      });
      expect(getUsageHistoryMock).toHaveBeenCalledWith({ months: 3, offset: 1 });
      const { usageHistory } = useEntitlementStore.getState();
      expect(usageHistory).toHaveLength(1);
      expect(usageHistory[0].billing_month).toBe('2026-05');
      expect(usageHistory[0].by_operation['ai.workflow_generate']).toBe(15);
      expect(usageHistory[0].period_start).toMatch(/^\d{4}-\d{2}-\d{2}T/);
    });
  });

  describe('fetchOperationLog', () => {
    it('replaces operations when offset is 0', async () => {
      getOperationLogMock.mockResolvedValueOnce({
        userIdentity: 'alice@example.com',
        billingMonth: '2026-05',
        operations: [{
          id: 'op-1',
          operationType: 'ai.workflow_generate',
          creditsCharged: 5,
          success: true,
          createdAt: { seconds: 1746057600n, nanos: 0 },
          metadata: { fields: { model: { kind: { case: 'stringValue', value: 'claude' } } } },
          errorMessage: '',
        }],
        total: 1,
        limit: 20,
        offset: 0,
        hasMore: false,
      });
      await act(async () => {
        await useEntitlementStore.getState().fetchOperationLog('2026-05');
      });
      const { operationLog, operationLogTotal } = useEntitlementStore.getState();
      expect(operationLog).toHaveLength(1);
      expect(operationLog[0].id).toBe('op-1');
      expect(operationLogTotal).toBe(1);
    });

    it('appends operations when offset > 0', async () => {
      useEntitlementStore.setState({
        operationLog: [{
          id: 'existing',
          operation_type: 'execution.run',
          credits_charged: 1,
          success: true,
          created_at: '',
        }],
      });
      getOperationLogMock.mockResolvedValueOnce({
        userIdentity: 'alice@example.com',
        billingMonth: '2026-05',
        operations: [{
          id: 'op-2',
          operationType: 'execution.run',
          creditsCharged: 1,
          success: true,
          createdAt: undefined,
          metadata: undefined,
          errorMessage: '',
        }],
        total: 2,
        limit: 20,
        offset: 1,
        hasMore: false,
      });
      await act(async () => {
        await useEntitlementStore.getState().fetchOperationLog('2026-05', 'execution', 20, 1);
      });
      expect(getOperationLogMock).toHaveBeenCalledWith({
        month: '2026-05',
        category: 'execution',
        limit: 20,
        offset: 1,
      });
      const { operationLog } = useEntitlementStore.getState();
      expect(operationLog.map((o) => o.id)).toEqual(['existing', 'op-2']);
    });
  });

  describe('clearOperationLog / setSelectedPeriod', () => {
    it('clearOperationLog resets fields', () => {
      useEntitlementStore.setState({
        operationLog: [{ id: 'x', operation_type: 'foo', credits_charged: 0, success: true, created_at: '' }],
        operationLogTotal: 5,
        operationLogHasMore: true,
      });
      useEntitlementStore.getState().clearOperationLog();
      const s = useEntitlementStore.getState();
      expect(s.operationLog).toEqual([]);
      expect(s.operationLogTotal).toBe(0);
      expect(s.operationLogHasMore).toBe(false);
    });

    it('setSelectedPeriod stores month', () => {
      useEntitlementStore.getState().setSelectedPeriod('2026-04');
      expect(useEntitlementStore.getState().selectedPeriod).toBe('2026-04');
    });
  });
});
