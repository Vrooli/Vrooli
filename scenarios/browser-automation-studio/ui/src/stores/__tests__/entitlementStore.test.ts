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
import { fetchEmptyResponse, fetchJsonResponse, installFetchMock, type FetchMock } from '@/test-utils';

// Mock dependencies using proper factory functions
vi.mock('../../config', () => ({
  API_BASE: 'http://localhost:8080/api/v1',
}));

// Import store AFTER mocks
import { useEntitlementStore, isValidEmail, type ApiSource } from '../entitlementStore';

describe('entitlementStore', () => {
  let fetchMock: FetchMock;

  beforeEach(() => {
    vi.clearAllMocks();
    fetchMock = installFetchMock();

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
      fetchMock.mockResolvedValueOnce(
        fetchJsonResponse({
          source: 'local',
          local_port: 16000,
        })
      );

      await act(async () => {
        await useEntitlementStore.getState().getApiSource();
      });

      const { apiSource, localApiPort } = useEntitlementStore.getState();
      expect(apiSource).toBe('local');
      expect(localApiPort).toBe(16000);
    });

    it('handles 404 gracefully with defaults', async () => {
      fetchMock.mockResolvedValueOnce(fetchEmptyResponse({ status: 404 }));

      await act(async () => {
        await useEntitlementStore.getState().getApiSource();
      });

      // Should keep defaults
      const { apiSource, localApiPort } = useEntitlementStore.getState();
      expect(apiSource).toBe('production');
      expect(localApiPort).toBe(15000);
    });

    it('handles network error gracefully', async () => {
      fetchMock.mockRejectedValueOnce(new Error('Network error'));

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
      fetchMock
        .mockResolvedValueOnce(fetchJsonResponse({ source: 'production', local_port: 15000 }))
        // For fetchStatus call after
        .mockResolvedValueOnce(
          fetchJsonResponse({
            user_identity: '',
            status: 'inactive',
            tier: 'free',
            is_active: false,
          })
        );

      await act(async () => {
        await useEntitlementStore.getState().setApiSource('production');
      });

      const { apiSource, isLoading } = useEntitlementStore.getState();
      expect(apiSource).toBe('production');
      expect(isLoading).toBe(false);
    });

    it('sets API source to local with port', async () => {
      fetchMock
        .mockResolvedValueOnce(fetchJsonResponse({ source: 'local', local_port: 17000 }))
        .mockResolvedValueOnce(
          fetchJsonResponse({
            user_identity: '',
            status: 'inactive',
            tier: 'free',
            is_active: false,
          })
        );

      await act(async () => {
        await useEntitlementStore.getState().setApiSource('local', 17000);
      });

      const { apiSource, localApiPort } = useEntitlementStore.getState();
      expect(apiSource).toBe('local');
      expect(localApiPort).toBe(17000);
    });

    it('sets API source to disabled', async () => {
      fetchMock
        .mockResolvedValueOnce(fetchJsonResponse({ source: 'disabled', local_port: 15000 }))
        .mockResolvedValueOnce(
          fetchJsonResponse({
            user_identity: '',
            status: 'inactive',
            tier: 'free',
            is_active: false,
          })
        );

      await act(async () => {
        await useEntitlementStore.getState().setApiSource('disabled');
      });

      const { apiSource } = useEntitlementStore.getState();
      expect(apiSource).toBe('disabled');
    });

    it('sets loading state during request', async () => {
      let resolvePromise: ((value: Response) => void) | null = null;
      const fetchPromise = new Promise<Response>((resolve) => {
        resolvePromise = resolve;
      });

      fetchMock.mockReturnValueOnce(fetchPromise);

      const setPromise = useEntitlementStore.getState().setApiSource('local');

      // Check loading state immediately
      expect(useEntitlementStore.getState().isLoading).toBe(true);

      // Resolve fetch
      if (!resolvePromise) {
        throw new Error('Expected fetch resolver to be defined');
      }
      resolvePromise(fetchJsonResponse({ source: 'local', local_port: 15000 }));

      await act(async () => {
        await setPromise;
      });
    });

    it('handles error response', async () => {
      fetchMock.mockResolvedValueOnce(fetchJsonResponse({ error: 'Invalid source' }, { status: 400 }));

      await act(async () => {
        await useEntitlementStore.getState().setApiSource('invalid' as ApiSource);
      });

      const { error, isLoading } = useEntitlementStore.getState();
      expect(error).toBe('Invalid source');
      expect(isLoading).toBe(false);
    });

    it('calls fetchStatus after setting source', async () => {
      fetchMock
        .mockResolvedValueOnce(fetchJsonResponse({ source: 'local', local_port: 16000 }))
        .mockResolvedValueOnce(
          fetchJsonResponse({
            user_identity: 'test@example.com',
            status: 'active',
            tier: 'pro',
            is_active: true,
          })
        );

      await act(async () => {
        await useEntitlementStore.getState().setApiSource('local', 16000);
      });

      // Should have called fetch twice (setApiSource + fetchStatus)
      expect(global.fetch).toHaveBeenCalledTimes(2);
    });
  });

  describe('setUserEmail', () => {
    it('sets user email successfully', async () => {
      fetchMock
        .mockResolvedValueOnce(fetchJsonResponse({}))
        // For fetchStatus call after
        .mockResolvedValueOnce(
          fetchJsonResponse({
            user_identity: 'user@example.com',
            status: 'active',
            tier: 'pro',
            is_active: true,
          })
        );

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
      fetchMock
        .mockResolvedValueOnce(fetchJsonResponse({}))
        .mockResolvedValueOnce(
          fetchJsonResponse({
            user_identity: 'user@example.com',
            status: 'inactive',
            tier: 'free',
            is_active: false,
          })
        );

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

      fetchMock.mockResolvedValueOnce(fetchJsonResponse({ status: 'cleared' }));

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
      fetchMock
        .mockResolvedValueOnce(fetchJsonResponse({ tier: 'pro' }))
        // For fetchStatus call after
        .mockResolvedValueOnce(
          fetchJsonResponse({
            user_identity: '',
            status: 'active',
            tier: 'pro',
            is_active: true,
            override_tier: 'pro',
          })
        );

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
      fetchMock
        .mockResolvedValueOnce(fetchEmptyResponse())
        .mockResolvedValueOnce(
          fetchJsonResponse({
            user_identity: '',
            status: 'inactive',
            tier: 'free',
            is_active: false,
          })
        );

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

      fetchMock.mockResolvedValueOnce(fetchJsonResponse(mockStatus));

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
      fetchMock.mockResolvedValueOnce(fetchJsonResponse({ error: 'Internal server error' }, { status: 500 }));

      await act(async () => {
        await useEntitlementStore.getState().fetchStatus();
      });

      const { status, error, isLoading } = useEntitlementStore.getState();

      expect(status).toBeNull();
      expect(error).toBe('Internal server error');
      expect(isLoading).toBe(false);
    });

    it('sets loading state during fetch', async () => {
      let resolvePromise: ((value: Response) => void) | null = null;
      const fetchPromise = new Promise<Response>((resolve) => {
        resolvePromise = resolve;
      });

      fetchMock.mockReturnValueOnce(fetchPromise);

      const fetchCall = useEntitlementStore.getState().fetchStatus();

      // Check loading state immediately
      expect(useEntitlementStore.getState().isLoading).toBe(true);

      // Resolve fetch
      if (!resolvePromise) {
        throw new Error('Expected fetch resolver to be defined');
      }
      resolvePromise(
        fetchJsonResponse({
          user_identity: '',
          status: 'inactive',
          tier: 'free',
          is_active: false,
        })
      );

      await act(async () => {
        await fetchCall;
      });

      expect(useEntitlementStore.getState().isLoading).toBe(false);
    });

    it('updates userEmail from status response', async () => {
      fetchMock.mockResolvedValueOnce(
        fetchJsonResponse({
          user_identity: 'fetched@example.com',
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
          ai_reset_date: '2025-02-01',
        })
      );

      await act(async () => {
        await useEntitlementStore.getState().fetchStatus();
      });

      const { userEmail } = useEntitlementStore.getState();
      expect(userEmail).toBe('fetched@example.com');
    });

    it('updates overrideTier from status response', async () => {
      fetchMock.mockResolvedValueOnce(
        fetchJsonResponse({
          user_identity: '',
          status: 'active',
          tier: 'studio',
          is_active: true,
          override_tier: 'studio',
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
          ai_reset_date: '2025-02-01',
        })
      );

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

      fetchMock.mockResolvedValueOnce(fetchJsonResponse(mockStatus));

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
      fetchMock.mockResolvedValueOnce(fetchJsonResponse({ email: 'stored@example.com' }));

      let result = '';
      await act(async () => {
        result = await useEntitlementStore.getState().getUserEmail();
      });

      expect(result).toBe('stored@example.com');
      expect(useEntitlementStore.getState().userEmail).toBe('stored@example.com');
    });

    it('returns empty string on error', async () => {
      fetchMock.mockResolvedValueOnce(fetchEmptyResponse({ status: 404 }));

      let result = '';
      await act(async () => {
        result = await useEntitlementStore.getState().getUserEmail();
      });

      expect(result).toBe('');
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

      fetchMock.mockResolvedValueOnce(fetchJsonResponse({ periods: mockPeriods }));

      await act(async () => {
        await useEntitlementStore.getState().fetchUsageHistory(6, 0);
      });

      const { usageHistory, historyLoading } = useEntitlementStore.getState();
      expect(usageHistory).toEqual(mockPeriods);
      expect(historyLoading).toBe(false);
    });

    it('handles usage history error gracefully', async () => {
      fetchMock.mockRejectedValueOnce(new Error('Network error'));

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

      fetchMock.mockResolvedValueOnce(fetchJsonResponse(mockPage));

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

      fetchMock.mockResolvedValueOnce(fetchJsonResponse(mockPage));

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
