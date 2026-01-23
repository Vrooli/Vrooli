import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useUsageDashboard } from './useUsageDashboard';
import * as usageService from '../services/usage.service';
import * as dateFormatters from '../../../shared/lib/dateFormatters';
import type { AdminUsageSummary, UsageRecord } from '../../../shared/api';

// Mock the service module
vi.mock('../services/usage.service', async () => {
  const actual = await vi.importActual('../services/usage.service');
  return {
    ...actual,
    fetchUsageSummary: vi.fn(),
  };
});

// Mock the date formatters module
vi.mock('../../../shared/lib/dateFormatters', async () => {
  const actual = await vi.importActual('../../../shared/lib/dateFormatters');
  return {
    ...actual,
    getCurrentPeriod: vi.fn(),
    isCurrentPeriod: vi.fn(),
  };
});

const mockFetchUsageSummary = vi.mocked(usageService.fetchUsageSummary);
const mockGetCurrentPeriod = vi.mocked(dateFormatters.getCurrentPeriod);
const mockIsCurrentPeriod = vi.mocked(dateFormatters.isCurrentPeriod);

const createMockUsageRecord = (overrides: Partial<UsageRecord> = {}): UsageRecord => ({
  id: '1',
  user_identity: 'user@example.com',
  limit_key: 'ai_chat',
  usage_amount: 100,
  billing_period: '2024-01',
  app_bundle_key: 'app-1',
  last_operation_at: '2024-01-15T10:00:00Z',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-15T10:00:00Z',
  ...overrides,
});

const createMockSummary = (overrides: Partial<AdminUsageSummary> = {}): AdminUsageSummary => ({
  billing_period: '2024-01',
  total_users: 5,
  total_records: 100,
  user_totals: {
    'user1@example.com': 500,
    'user2@example.com': 300,
  },
  app_totals: {
    'app-1': 600,
    'app-2': 200,
  },
  records: [createMockUsageRecord({ id: '1' }), createMockUsageRecord({ id: '2' })],
  ...overrides,
});

describe('useUsageDashboard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockGetCurrentPeriod.mockReturnValue('2024-01');
    mockIsCurrentPeriod.mockImplementation((period: string) => period === '2024-01');
    mockFetchUsageSummary.mockResolvedValue(createMockSummary());
  });

  describe('initial state', () => {
    it('starts with loading state', async () => {
      const { result } = renderHook(() => useUsageDashboard());
      expect(result.current.loading).toBe(true);
      await waitFor(() => expect(result.current.loading).toBe(false));
    });

    it('has null summary initially', async () => {
      mockFetchUsageSummary.mockImplementation(
        () => new Promise((resolve) => setTimeout(() => resolve(createMockSummary()), 10))
      );
      const { result } = renderHook(() => useUsageDashboard());
      expect(result.current.summary).toBeNull();
      await waitFor(() => expect(result.current.loading).toBe(false));
    });

    it('uses current billing period', async () => {
      mockGetCurrentPeriod.mockReturnValue('2024-03');
      const { result } = renderHook(() => useUsageDashboard());
      expect(result.current.billingPeriod).toBe('2024-03');
      await waitFor(() => expect(result.current.loading).toBe(false));
    });
  });

  describe('loading data', () => {
    it('fetches summary on mount', async () => {
      const mockSummary = createMockSummary();
      mockFetchUsageSummary.mockResolvedValue(mockSummary);

      const { result } = renderHook(() => useUsageDashboard());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(result.current.summary).toEqual(mockSummary);
      expect(mockFetchUsageSummary).toHaveBeenCalledTimes(1);
    });

    it('handles fetch error', async () => {
      mockFetchUsageSummary.mockRejectedValue(new Error('Network error'));

      const { result } = renderHook(() => useUsageDashboard());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(result.current.error).toBe('Network error');
      expect(result.current.summary).toBeNull();
    });

    it('can refresh data', async () => {
      mockFetchUsageSummary.mockResolvedValue(createMockSummary());

      const { result } = renderHook(() => useUsageDashboard());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(mockFetchUsageSummary).toHaveBeenCalledTimes(1);

      await act(async () => {
        await result.current.fetchSummary();
      });

      expect(mockFetchUsageSummary).toHaveBeenCalledTimes(2);
    });
  });

  describe('period navigation', () => {
    it('navigates to previous month', async () => {
      mockGetCurrentPeriod.mockReturnValue('2024-03');

      const { result } = renderHook(() => useUsageDashboard());
      await waitFor(() => expect(result.current.loading).toBe(false));

      act(() => {
        result.current.navigateMonth(-1);
      });

      expect(result.current.billingPeriod).toBe('2024-02');
    });

    it('navigates to next month', async () => {
      mockGetCurrentPeriod.mockReturnValue('2024-03');

      const { result } = renderHook(() => useUsageDashboard());
      await waitFor(() => expect(result.current.loading).toBe(false));

      // First go back
      act(() => {
        result.current.navigateMonth(-1);
      });
      expect(result.current.billingPeriod).toBe('2024-02');

      // Then forward
      act(() => {
        result.current.navigateMonth(1);
      });
      expect(result.current.billingPeriod).toBe('2024-03');
    });

    it('fetches new data when period changes', async () => {
      mockGetCurrentPeriod.mockReturnValue('2024-03');
      mockFetchUsageSummary.mockResolvedValue(createMockSummary());

      const { result } = renderHook(() => useUsageDashboard());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(mockFetchUsageSummary).toHaveBeenCalledTimes(1);

      act(() => {
        result.current.navigateMonth(-1);
      });

      await waitFor(() => {
        expect(mockFetchUsageSummary).toHaveBeenCalledTimes(2);
      });
    });

    it('detects current period correctly', async () => {
      mockGetCurrentPeriod.mockReturnValue('2024-03');
      mockIsCurrentPeriod.mockImplementation((period: string) => period === '2024-03');

      const { result } = renderHook(() => useUsageDashboard());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(result.current.isCurrentPeriod).toBe(true);

      act(() => {
        result.current.navigateMonth(-1);
      });

      expect(result.current.isCurrentPeriod).toBe(false);
    });

    it('formats period correctly', async () => {
      mockGetCurrentPeriod.mockReturnValue('2024-01');

      const { result } = renderHook(() => useUsageDashboard());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(result.current.formattedPeriod).toBe('January 2024');
    });
  });

  describe('computed values', () => {
    it('calculates total usage', async () => {
      const mockSummary = createMockSummary({
        user_totals: {
          'user1@example.com': 500,
          'user2@example.com': 300,
          'user3@example.com': 200,
        },
      });
      mockFetchUsageSummary.mockResolvedValue(mockSummary);

      const { result } = renderHook(() => useUsageDashboard());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(result.current.totalUsage).toBe(1000);
    });

    it('sorts app totals by usage', async () => {
      const mockSummary = createMockSummary({
        app_totals: {
          'low-app': 100,
          'high-app': 500,
          'mid-app': 300,
        },
      });
      mockFetchUsageSummary.mockResolvedValue(mockSummary);

      const { result } = renderHook(() => useUsageDashboard());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(result.current.sortedAppTotals[0].app).toBe('high-app');
      expect(result.current.sortedAppTotals[1].app).toBe('mid-app');
      expect(result.current.sortedAppTotals[2].app).toBe('low-app');
    });

    it('calculates app usage percentages', async () => {
      const mockSummary = createMockSummary({
        app_totals: {
          'half-app': 500,
          'quarter-app': 250,
          'quarter-app-2': 250,
        },
      });
      mockFetchUsageSummary.mockResolvedValue(mockSummary);

      const { result } = renderHook(() => useUsageDashboard());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(result.current.sortedAppTotals[0].percentage).toBe(50);
      expect(result.current.sortedAppTotals[1].percentage).toBe(25);
    });

    it('gets top users sorted by usage', async () => {
      const mockSummary = createMockSummary({
        user_totals: {
          'low@example.com': 100,
          'high@example.com': 500,
          'mid@example.com': 300,
        },
      });
      mockFetchUsageSummary.mockResolvedValue(mockSummary);

      const { result } = renderHook(() => useUsageDashboard());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(result.current.topUsers[0].user).toBe('high@example.com');
      expect(result.current.topUsers[0].usage).toBe(500);
    });

    it('limits recent records', async () => {
      const manyRecords = Array.from({ length: 30 }, (_, i) =>
        createMockUsageRecord({ id: String(i + 1) })
      );
      const mockSummary = createMockSummary({ records: manyRecords });
      mockFetchUsageSummary.mockResolvedValue(mockSummary);

      const { result } = renderHook(() => useUsageDashboard());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(result.current.recentRecords).toHaveLength(20);
    });
  });

  describe('error handling', () => {
    it('clears error state', async () => {
      mockFetchUsageSummary.mockRejectedValue(new Error('Test error'));

      const { result } = renderHook(() => useUsageDashboard());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(result.current.error).toBe('Test error');

      act(() => {
        result.current.clearError();
      });

      expect(result.current.error).toBeNull();
    });

    it('clears error on new fetch', async () => {
      mockFetchUsageSummary
        .mockRejectedValueOnce(new Error('First error'))
        .mockResolvedValueOnce(createMockSummary());

      const { result } = renderHook(() => useUsageDashboard());
      await waitFor(() => expect(result.current.loading).toBe(false));

      expect(result.current.error).toBe('First error');

      await act(async () => {
        await result.current.fetchSummary();
      });

      expect(result.current.error).toBeNull();
    });
  });
});
