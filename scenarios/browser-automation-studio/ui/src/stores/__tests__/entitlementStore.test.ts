/**
 * EntitlementStore Test Suite
 *
 * Tests entitlement store functionality including:
 * - API source switching (production/local/disabled)
 * - Email identity management
 * - Entitlement status fetching
 * - Override tier functionality
 *
 * Requirements validated:
 * - API source state management
 * - Email validation and persistence
 * - Status fetch and cache
 * - Error handling
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { act } from '@testing-library/react';

// Mock dependencies using proper factory functions
vi.mock('../../config', () => ({
  API_BASE: 'http://localhost:8080/api/v1',
}));

// Import store AFTER mocks
import { useEntitlementStore, isValidEmail, type ApiSource } from '../entitlementStore';

describe('entitlementStore', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    global.fetch = vi.fn();

    // Reset store state
    useEntitlementStore.setState({
      userEmail: '',
      status: null,
      overrideTier: null,
      isLoading: false,
      error: null,
      lastFetched: null,
      isOffline: false,
      apiSource: 'production' as ApiSource,
      localApiPort: 15000,
      usageHistory: [],
      historyLoading: false,
      selectedPeriod: null,
      operationLog: [],
      operationLogLoading: false,
      operationLogTotal: 0,
      operationLogHasMore: false,
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('Initial State', () => {
    it('has empty user email', () => {
      const { userEmail } = useEntitlementStore.getState();
      expect(userEmail).toBe('');
    });

    it('has null status', () => {
      const { status } = useEntitlementStore.getState();
      expect(status).toBeNull();
    });

    it('has production as default API source', () => {
      const { apiSource } = useEntitlementStore.getState();
      expect(apiSource).toBe('production');
    });

    it('has default local API port', () => {
      const { localApiPort } = useEntitlementStore.getState();
      expect(localApiPort).toBe(15000);
    });

    it('is not loading', () => {
      const { isLoading } = useEntitlementStore.getState();
      expect(isLoading).toBe(false);
    });

    it('has no error', () => {
      const { error } = useEntitlementStore.getState();
      expect(error).toBeNull();
    });
  });

  describe('isValidEmail', () => {
    it('validates correct email addresses', () => {
      expect(isValidEmail('user@example.com')).toBe(true);
      expect(isValidEmail('test.user@domain.org')).toBe(true);
      expect(isValidEmail('user+tag@example.co.uk')).toBe(true);
    });

    it('rejects invalid email addresses', () => {
      expect(isValidEmail('')).toBe(false);
      expect(isValidEmail('notanemail')).toBe(false);
      expect(isValidEmail('@example.com')).toBe(false);
      expect(isValidEmail('user@')).toBe(false);
      expect(isValidEmail('user@.')).toBe(false);
      expect(isValidEmail('user@domain')).toBe(false);
      expect(isValidEmail('user@domain.')).toBe(false);
    });

    it('handles whitespace', () => {
      expect(isValidEmail('  user@example.com  ')).toBe(true);
      expect(isValidEmail('  ')).toBe(false);
    });
  });

  describe('getApiSource', () => {
    it('fetches API source from backend', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          source: 'local',
          local_port: 16000,
        }),
      } as Response);

      await act(async () => {
        await useEntitlementStore.getState().getApiSource();
      });

      const { apiSource, localApiPort } = useEntitlementStore.getState();
      expect(apiSource).toBe('local');
      expect(localApiPort).toBe(16000);
    });

    it('handles 404 gracefully with defaults', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: false,
        status: 404,
      } as Response);

      await act(async () => {
        await useEntitlementStore.getState().getApiSource();
      });

      // Should keep defaults
      const { apiSource, localApiPort } = useEntitlementStore.getState();
      expect(apiSource).toBe('production');
      expect(localApiPort).toBe(15000);
    });

    it('handles network error gracefully', async () => {
      vi.mocked(global.fetch).mockRejectedValueOnce(new Error('Network error'));

      await act(async () => {
        await useEntitlementStore.getState().getApiSource();
      });

      // Should keep defaults
      const { apiSource, localApiPort, error } = useEntitlementStore.getState();
      expect(apiSource).toBe('production');
      expect(localApiPort).toBe(15000);
      // Should not set error - silently fail
      expect(error).toBeNull();
    });
  });

  describe('setApiSource', () => {
    it('sets API source to production', async () => {
      vi.mocked(global.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ source: 'production', local_port: 15000 }),
        } as Response)
        // For fetchStatus call after
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({
            user_identity: '',
            status: 'inactive',
            tier: 'free',
            is_active: false,
          }),
        } as Response);

      await act(async () => {
        await useEntitlementStore.getState().setApiSource('production');
      });

      const { apiSource, isLoading } = useEntitlementStore.getState();
      expect(apiSource).toBe('production');
      expect(isLoading).toBe(false);
    });

    it('sets API source to local with port', async () => {
      vi.mocked(global.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ source: 'local', local_port: 17000 }),
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({
            user_identity: '',
            status: 'inactive',
            tier: 'free',
            is_active: false,
          }),
        } as Response);

      await act(async () => {
        await useEntitlementStore.getState().setApiSource('local', 17000);
      });

      const { apiSource, localApiPort } = useEntitlementStore.getState();
      expect(apiSource).toBe('local');
      expect(localApiPort).toBe(17000);
    });

    it('sets API source to disabled', async () => {
      vi.mocked(global.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ source: 'disabled', local_port: 15000 }),
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({
            user_identity: '',
            status: 'inactive',
            tier: 'free',
            is_active: false,
          }),
        } as Response);

      await act(async () => {
        await useEntitlementStore.getState().setApiSource('disabled');
      });

      const { apiSource } = useEntitlementStore.getState();
      expect(apiSource).toBe('disabled');
    });

    it('sets loading state during request', async () => {
      let resolvePromise: (value: unknown) => void;
      const fetchPromise = new Promise((resolve) => {
        resolvePromise = resolve;
      });

      vi.mocked(global.fetch).mockReturnValueOnce(fetchPromise as Promise<Response>);

      const setPromise = useEntitlementStore.getState().setApiSource('local');

      // Check loading state immediately
      expect(useEntitlementStore.getState().isLoading).toBe(true);

      // Resolve fetch
      resolvePromise!({
        ok: true,
        json: async () => ({ source: 'local', local_port: 15000 }),
      });

      await act(async () => {
        await setPromise;
      });
    });

    it('handles error response', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: false,
        status: 400,
        json: async () => ({ error: 'Invalid source' }),
      } as Response);

      await act(async () => {
        await useEntitlementStore.getState().setApiSource('invalid' as ApiSource);
      });

      const { error, isLoading } = useEntitlementStore.getState();
      expect(error).toBe('Invalid source');
      expect(isLoading).toBe(false);
    });

    it('calls fetchStatus after setting source', async () => {
      vi.mocked(global.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ source: 'local', local_port: 16000 }),
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({
            user_identity: 'test@example.com',
            status: 'active',
            tier: 'pro',
            is_active: true,
          }),
        } as Response);

      await act(async () => {
        await useEntitlementStore.getState().setApiSource('local', 16000);
      });

      // Should have called fetch twice (setApiSource + fetchStatus)
      expect(global.fetch).toHaveBeenCalledTimes(2);
    });
  });

  describe('setUserEmail', () => {
    it('sets user email successfully', async () => {
      vi.mocked(global.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({}),
        } as Response)
        // For fetchStatus call after
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({
            user_identity: 'user@example.com',
            status: 'active',
            tier: 'pro',
            is_active: true,
          }),
        } as Response);

      await act(async () => {
        await useEntitlementStore.getState().setUserEmail('user@example.com');
      });

      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('entitlement/identity'),
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ email: 'user@example.com' }),
        })
      );
    });

    it('rejects empty email', async () => {
      await act(async () => {
        await useEntitlementStore.getState().setUserEmail('');
      });

      const { error } = useEntitlementStore.getState();
      expect(error).toBe('Email is required');
      expect(global.fetch).not.toHaveBeenCalled();
    });

    it('rejects invalid email format', async () => {
      await act(async () => {
        await useEntitlementStore.getState().setUserEmail('notanemail');
      });

      const { error } = useEntitlementStore.getState();
      expect(error).toBe('Please enter a valid email address');
      expect(global.fetch).not.toHaveBeenCalled();
    });

    it('normalizes email to lowercase', async () => {
      vi.mocked(global.fetch)
        .mockResolvedValueOnce({ ok: true, json: async () => ({}) } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({
            user_identity: 'user@example.com',
            status: 'inactive',
            tier: 'free',
            is_active: false,
          }),
        } as Response);

      await act(async () => {
        await useEntitlementStore.getState().setUserEmail('USER@EXAMPLE.COM');
      });

      expect(global.fetch).toHaveBeenCalledWith(
        expect.anything(),
        expect.objectContaining({
          body: JSON.stringify({ email: 'user@example.com' }),
        })
      );
    });
  });

  describe('clearUserEmail', () => {
    it('clears user email and resets state', async () => {
      // Set initial state
      useEntitlementStore.setState({
        userEmail: 'test@example.com',
        status: {
          user_identity: 'test@example.com',
          status: 'active',
          tier: 'pro',
          is_active: true,
          features: [],
          monthly_limit: -1,
          monthly_used: 0,
          monthly_remaining: -1,
          requires_watermark: false,
          can_use_ai: true,
          can_use_recording: true,
          entitlements_enabled: true,
          ai_credits_used: 0,
          ai_credits_limit: -1,
          ai_credits_remaining: -1,
          ai_requests_count: 0,
          ai_reset_date: '',
        },
      });

      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ status: 'cleared' }),
      } as Response);

      await act(async () => {
        await useEntitlementStore.getState().clearUserEmail();
      });

      const { userEmail, status, isLoading } = useEntitlementStore.getState();
      expect(userEmail).toBe('');
      expect(status).toBeNull();
      expect(isLoading).toBe(false);
    });
  });

  describe('setOverrideTier', () => {
    it('sets override tier successfully', async () => {
      vi.mocked(global.fetch)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ tier: 'pro' }),
        } as Response)
        // For fetchStatus call after
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({
            user_identity: '',
            status: 'active',
            tier: 'pro',
            is_active: true,
            override_tier: 'pro',
          }),
        } as Response);

      await act(async () => {
        await useEntitlementStore.getState().setOverrideTier('pro');
      });

      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('entitlement/override'),
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ tier: 'pro' }),
        })
      );
    });

    it('clears override tier with null', async () => {
      vi.mocked(global.fetch)
        .mockResolvedValueOnce({
          ok: true,
          status: 204,
          json: async () => ({}),
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({
            user_identity: '',
            status: 'inactive',
            tier: 'free',
            is_active: false,
          }),
        } as Response);

      await act(async () => {
        await useEntitlementStore.getState().setOverrideTier(null);
      });

      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('entitlement/override'),
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ tier: '' }),
        })
      );
    });
  });

  describe('fetchStatus', () => {
    it('fetches entitlement status successfully', async () => {
      const mockStatus = {
        user_identity: 'test@example.com',
        status: 'active',
        tier: 'pro',
        is_active: true,
        features: ['ai', 'recording'],
        monthly_limit: -1,
        monthly_used: 5,
        monthly_remaining: -1,
        requires_watermark: false,
        can_use_ai: true,
        can_use_recording: true,
        entitlements_enabled: true,
        ai_credits_used: 100,
        ai_credits_limit: -1,
        ai_credits_remaining: -1,
        ai_requests_count: 10,
        ai_reset_date: '2025-02-01',
      };

      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => mockStatus,
      } as Response);

      await act(async () => {
        await useEntitlementStore.getState().fetchStatus();
      });

      const { status, isLoading, error, lastFetched } = useEntitlementStore.getState();

      expect(status).toEqual(mockStatus);
      expect(isLoading).toBe(false);
      expect(error).toBeNull();
      expect(lastFetched).toBeInstanceOf(Date);
    });

    it('handles HTTP error responses', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: false,
        status: 500,
        json: async () => ({ error: 'Internal server error' }),
      } as Response);

      await act(async () => {
        await useEntitlementStore.getState().fetchStatus();
      });

      const { status, error, isLoading } = useEntitlementStore.getState();

      expect(status).toBeNull();
      expect(error).toBe('Internal server error');
      expect(isLoading).toBe(false);
    });

    it('sets loading state during fetch', async () => {
      let resolvePromise: (value: unknown) => void;
      const fetchPromise = new Promise((resolve) => {
        resolvePromise = resolve;
      });

      vi.mocked(global.fetch).mockReturnValueOnce(fetchPromise as Promise<Response>);

      const fetchCall = useEntitlementStore.getState().fetchStatus();

      // Check loading state immediately
      expect(useEntitlementStore.getState().isLoading).toBe(true);

      // Resolve fetch
      resolvePromise!({
        ok: true,
        json: async () => ({
          user_identity: '',
          status: 'inactive',
          tier: 'free',
          is_active: false,
        }),
      });

      await act(async () => {
        await fetchCall;
      });

      expect(useEntitlementStore.getState().isLoading).toBe(false);
    });

    it('updates userEmail from status response', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          user_identity: 'fetched@example.com',
          status: 'active',
          tier: 'pro',
          is_active: true,
        }),
      } as Response);

      await act(async () => {
        await useEntitlementStore.getState().fetchStatus();
      });

      const { userEmail } = useEntitlementStore.getState();
      expect(userEmail).toBe('fetched@example.com');
    });

    it('updates overrideTier from status response', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({
          user_identity: '',
          status: 'active',
          tier: 'studio',
          is_active: true,
          override_tier: 'studio',
        }),
      } as Response);

      await act(async () => {
        await useEntitlementStore.getState().fetchStatus();
      });

      const { overrideTier } = useEntitlementStore.getState();
      expect(overrideTier).toBe('studio');
    });
  });

  describe('refreshEntitlement', () => {
    it('refreshes entitlement status', async () => {
      const mockStatus = {
        user_identity: 'refresh@example.com',
        status: 'active',
        tier: 'pro',
        is_active: true,
      };

      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => mockStatus,
      } as Response);

      await act(async () => {
        await useEntitlementStore.getState().refreshEntitlement();
      });

      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('entitlement/refresh'),
        expect.objectContaining({ method: 'POST' })
      );
    });
  });

  describe('getUserEmail', () => {
    it('fetches stored user email', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ email: 'stored@example.com' }),
      } as Response);

      let result: string;
      await act(async () => {
        result = await useEntitlementStore.getState().getUserEmail();
      });

      expect(result!).toBe('stored@example.com');
      expect(useEntitlementStore.getState().userEmail).toBe('stored@example.com');
    });

    it('returns empty string on error', async () => {
      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: false,
        status: 404,
      } as Response);

      let result: string;
      await act(async () => {
        result = await useEntitlementStore.getState().getUserEmail();
      });

      expect(result!).toBe('');
    });
  });

  describe('Usage History', () => {
    it('fetches usage history', async () => {
      const mockPeriods = [
        {
          billing_month: '2025-01',
          total_credits_used: 100,
          total_operations: 10,
          by_operation: {},
          operation_counts: {},
          credits_limit: 500,
          credits_remaining: 400,
          period_start: '2025-01-01',
          period_end: '2025-01-31',
          reset_date: '2025-02-01',
        },
      ];

      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ periods: mockPeriods }),
      } as Response);

      await act(async () => {
        await useEntitlementStore.getState().fetchUsageHistory(6, 0);
      });

      const { usageHistory, historyLoading } = useEntitlementStore.getState();
      expect(usageHistory).toEqual(mockPeriods);
      expect(historyLoading).toBe(false);
    });

    it('handles usage history error gracefully', async () => {
      vi.mocked(global.fetch).mockRejectedValueOnce(new Error('Network error'));

      await act(async () => {
        await useEntitlementStore.getState().fetchUsageHistory();
      });

      const { usageHistory, historyLoading } = useEntitlementStore.getState();
      expect(usageHistory).toEqual([]);
      expect(historyLoading).toBe(false);
    });
  });

  describe('Operation Log', () => {
    it('fetches operation log', async () => {
      const mockPage = {
        user_identity: 'test@example.com',
        billing_month: '2025-01',
        operations: [
          {
            id: '1',
            operation_type: 'ai_generation',
            credits_charged: 10,
            success: true,
            created_at: '2025-01-15T10:00:00Z',
          },
        ],
        total: 1,
        limit: 20,
        offset: 0,
        has_more: false,
      };

      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => mockPage,
      } as Response);

      await act(async () => {
        await useEntitlementStore.getState().fetchOperationLog('2025-01');
      });

      const { operationLog, operationLogTotal, operationLogHasMore } = useEntitlementStore.getState();
      expect(operationLog).toHaveLength(1);
      expect(operationLogTotal).toBe(1);
      expect(operationLogHasMore).toBe(false);
    });

    it('appends to operation log when offset > 0', async () => {
      // Set initial state with existing operations
      useEntitlementStore.setState({
        operationLog: [
          {
            id: '1',
            operation_type: 'ai_generation',
            credits_charged: 10,
            success: true,
            created_at: '2025-01-15T10:00:00Z',
          },
        ],
      });

      const mockPage = {
        user_identity: 'test@example.com',
        billing_month: '2025-01',
        operations: [
          {
            id: '2',
            operation_type: 'ai_generation',
            credits_charged: 5,
            success: true,
            created_at: '2025-01-14T10:00:00Z',
          },
        ],
        total: 2,
        limit: 20,
        offset: 1,
        has_more: false,
      };

      vi.mocked(global.fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => mockPage,
      } as Response);

      await act(async () => {
        await useEntitlementStore.getState().fetchOperationLog('2025-01', undefined, 20, 1);
      });

      const { operationLog } = useEntitlementStore.getState();
      expect(operationLog).toHaveLength(2);
    });

    it('clears operation log', () => {
      useEntitlementStore.setState({
        operationLog: [{ id: '1', operation_type: 'test', credits_charged: 1, success: true, created_at: '' }],
        operationLogTotal: 1,
        operationLogHasMore: true,
      });

      act(() => {
        useEntitlementStore.getState().clearOperationLog();
      });

      const { operationLog, operationLogTotal, operationLogHasMore } = useEntitlementStore.getState();
      expect(operationLog).toEqual([]);
      expect(operationLogTotal).toBe(0);
      expect(operationLogHasMore).toBe(false);
    });
  });

  describe('setSelectedPeriod', () => {
    it('sets selected period', () => {
      act(() => {
        useEntitlementStore.getState().setSelectedPeriod('2025-01');
      });

      const { selectedPeriod } = useEntitlementStore.getState();
      expect(selectedPeriod).toBe('2025-01');
    });

    it('clears selected period with null', () => {
      useEntitlementStore.setState({ selectedPeriod: '2025-01' });

      act(() => {
        useEntitlementStore.getState().setSelectedPeriod(null);
      });

      const { selectedPeriod } = useEntitlementStore.getState();
      expect(selectedPeriod).toBeNull();
    });
  });
});
