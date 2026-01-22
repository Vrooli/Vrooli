import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useTierLimitsForm } from './useTierLimitsForm';
import type { TierLimit } from '../../../shared/api';

// Mock the tiers service
const fetchAllTierLimitsMock = vi.fn();
const saveTierLimitMock = vi.fn();

vi.mock('../services/tiers.service', async () => {
  const actual = await vi.importActual<typeof import('../services/tiers.service')>(
    '../services/tiers.service'
  );
  return {
    ...actual,
    fetchAllTierLimits: (...args: unknown[]) => fetchAllTierLimitsMock(...args),
    saveTierLimit: (...args: unknown[]) => saveTierLimitMock(...args),
  };
});

const mockLimits: Record<string, TierLimit[]> = {
  free: [
    {
      id: '1',
      tier_id: 'free',
      limit_key: 'ai_credits',
      limit_type: 'cost_based',
      limit_value: 0,
      display_dollars: 0,
      cost_multiplier: 1000,
      reset_period: 'monthly',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
  ],
  solo: [
    {
      id: '2',
      tier_id: 'solo',
      limit_key: 'ai_credits',
      limit_type: 'cost_based',
      limit_value: 5000,
      display_dollars: 5,
      cost_multiplier: 1000,
      reset_period: 'monthly',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
  ],
  pro: [
    {
      id: '3',
      tier_id: 'pro',
      limit_key: 'ai_credits',
      limit_type: 'cost_based',
      limit_value: 20000,
      display_dollars: 20,
      cost_multiplier: 1000,
      reset_period: 'monthly',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
  ],
  studio: [
    {
      id: '4',
      tier_id: 'studio',
      limit_key: 'ai_credits',
      limit_type: 'cost_based',
      limit_value: 100000,
      display_dollars: 100,
      cost_multiplier: 1000,
      reset_period: 'monthly',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
  ],
  business: [
    {
      id: '5',
      tier_id: 'business',
      limit_key: 'ai_credits',
      limit_type: 'cost_based',
      limit_value: -1, // unlimited
      display_dollars: undefined,
      cost_multiplier: 1000,
      reset_period: 'monthly',
      created_at: '2024-01-01T00:00:00Z',
      updated_at: '2024-01-01T00:00:00Z',
    },
  ],
};

describe('useTierLimitsForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    fetchAllTierLimitsMock.mockResolvedValue(mockLimits);
  });

  describe('initial state', () => {
    it('starts with loading state', () => {
      const { result } = renderHook(() => useTierLimitsForm());

      expect(result.current.loading).toBe(true);
      expect(result.current.limits).toEqual({});
      expect(result.current.saving).toBeNull();
      expect(result.current.editedValues).toEqual({});
    });

    it('loads limits on mount', async () => {
      const { result } = renderHook(() => useTierLimitsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(fetchAllTierLimitsMock).toHaveBeenCalledTimes(1);
      expect(result.current.limits).toEqual(mockLimits);
    });
  });

  describe('load error handling', () => {
    it('handles load error', async () => {
      fetchAllTierLimitsMock.mockRejectedValue(new Error('API failure'));

      const { result } = renderHook(() => useTierLimitsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.toasts).toHaveLength(1);
      expect(result.current.toasts[0]).toEqual({
        type: 'error',
        message: 'API failure',
      });
    });

    it('handles non-Error load rejection', async () => {
      fetchAllTierLimitsMock.mockRejectedValue('Unknown error');

      const { result } = renderHook(() => useTierLimitsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.toasts[0].message).toBe('Failed to load tier limits');
    });
  });

  describe('editing values', () => {
    it('updates edited value', async () => {
      const { result } = renderHook(() => useTierLimitsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.updateEditedValue('solo:ai_credits', '10');
      });

      expect(result.current.editedValues['solo:ai_credits']).toBe('10');
    });

    it('normalizes edited value to lowercase', async () => {
      const { result } = renderHook(() => useTierLimitsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.updateEditedValue('business:ai_credits', 'UNLIMITED');
      });

      expect(result.current.editedValues['business:ai_credits']).toBe('unlimited');
    });

    it('clears edited value', async () => {
      const { result } = renderHook(() => useTierLimitsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.updateEditedValue('solo:ai_credits', '10');
      });

      expect(result.current.editedValues['solo:ai_credits']).toBe('10');

      act(() => {
        result.current.clearEditedValue('solo:ai_credits');
      });

      expect(result.current.editedValues['solo:ai_credits']).toBeUndefined();
    });
  });

  describe('saving limits', () => {
    it('saves limit successfully', async () => {
      saveTierLimitMock.mockResolvedValue(undefined);

      const { result } = renderHook(() => useTierLimitsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      const soloLimit = mockLimits.solo[0];

      act(() => {
        result.current.updateEditedValue('solo:ai_credits', '15');
      });

      await act(async () => {
        await result.current.handleSave('solo', soloLimit);
      });

      expect(saveTierLimitMock).toHaveBeenCalledWith('solo', 'ai_credits', {
        display_dollars: 15,
      });
      expect(result.current.toasts).toContainEqual({
        type: 'success',
        message: 'Limit for solo/ai_credits updated',
      });
      expect(result.current.editedValues['solo:ai_credits']).toBeUndefined();
    });

    it('saves unlimited value', async () => {
      saveTierLimitMock.mockResolvedValue(undefined);

      const { result } = renderHook(() => useTierLimitsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      const studioLimit = mockLimits.studio[0];

      act(() => {
        result.current.updateEditedValue('studio:ai_credits', 'unlimited');
      });

      await act(async () => {
        await result.current.handleSave('studio', studioLimit);
      });

      expect(saveTierLimitMock).toHaveBeenCalledWith('studio', 'ai_credits', {
        is_unlimited: true,
      });
    });

    it('saves -1 as unlimited', async () => {
      saveTierLimitMock.mockResolvedValue(undefined);

      const { result } = renderHook(() => useTierLimitsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      const studioLimit = mockLimits.studio[0];

      act(() => {
        result.current.updateEditedValue('studio:ai_credits', '-1');
      });

      await act(async () => {
        await result.current.handleSave('studio', studioLimit);
      });

      expect(saveTierLimitMock).toHaveBeenCalledWith('studio', 'ai_credits', {
        is_unlimited: true,
      });
    });

    it('handles invalid value on save', async () => {
      const { result } = renderHook(() => useTierLimitsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      const soloLimit = mockLimits.solo[0];

      act(() => {
        result.current.updateEditedValue('solo:ai_credits', 'invalid');
      });

      await act(async () => {
        await result.current.handleSave('solo', soloLimit);
      });

      expect(saveTierLimitMock).not.toHaveBeenCalled();
      expect(result.current.toasts).toContainEqual({
        type: 'error',
        message: 'Please enter a valid dollar amount',
      });
    });

    it('handles negative value on save', async () => {
      const { result } = renderHook(() => useTierLimitsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      const soloLimit = mockLimits.solo[0];

      act(() => {
        result.current.updateEditedValue('solo:ai_credits', '-5');
      });

      await act(async () => {
        await result.current.handleSave('solo', soloLimit);
      });

      expect(saveTierLimitMock).not.toHaveBeenCalled();
      expect(result.current.toasts).toContainEqual({
        type: 'error',
        message: 'Please enter a valid dollar amount',
      });
    });

    it('does nothing when no edited value exists', async () => {
      const { result } = renderHook(() => useTierLimitsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      const soloLimit = mockLimits.solo[0];

      await act(async () => {
        await result.current.handleSave('solo', soloLimit);
      });

      expect(saveTierLimitMock).not.toHaveBeenCalled();
    });

    it('handles save error', async () => {
      saveTierLimitMock.mockRejectedValue(new Error('Save failed'));

      const { result } = renderHook(() => useTierLimitsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      const soloLimit = mockLimits.solo[0];

      act(() => {
        result.current.updateEditedValue('solo:ai_credits', '15');
      });

      await act(async () => {
        await result.current.handleSave('solo', soloLimit);
      });

      expect(result.current.toasts).toContainEqual({
        type: 'error',
        message: 'Save failed',
      });
    });

    it('sets saving state during save', async () => {
      let resolveSave: () => void;
      saveTierLimitMock.mockReturnValue(
        new Promise<void>((resolve) => {
          resolveSave = resolve;
        })
      );

      const { result } = renderHook(() => useTierLimitsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      const soloLimit = mockLimits.solo[0];

      act(() => {
        result.current.updateEditedValue('solo:ai_credits', '15');
      });

      act(() => {
        result.current.handleSave('solo', soloLimit);
      });

      expect(result.current.saving).toBe('solo:ai_credits');

      await act(async () => {
        resolveSave!();
      });

      await waitFor(() => {
        expect(result.current.saving).toBeNull();
      });
    });

    it('refreshes limits after successful save', async () => {
      saveTierLimitMock.mockResolvedValue(undefined);

      const { result } = renderHook(() => useTierLimitsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      fetchAllTierLimitsMock.mockClear();

      const soloLimit = mockLimits.solo[0];

      act(() => {
        result.current.updateEditedValue('solo:ai_credits', '15');
      });

      await act(async () => {
        await result.current.handleSave('solo', soloLimit);
      });

      expect(fetchAllTierLimitsMock).toHaveBeenCalled();
    });
  });

  describe('quick actions', () => {
    it('resets to default values', async () => {
      const { result } = renderHook(() => useTierLimitsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.resetToDefaults();
      });

      expect(result.current.editedValues).toEqual({
        'free:ai_credits': '0',
        'solo:ai_credits': '5',
        'pro:ai_credits': '20',
        'studio:ai_credits': '100',
        'business:ai_credits': 'unlimited',
      });
    });

    it('doubles all limits', async () => {
      const { result } = renderHook(() => useTierLimitsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      act(() => {
        result.current.doubleAllLimits();
      });

      // Business tier is unlimited so should not be doubled
      expect(result.current.editedValues['solo:ai_credits']).toBe('10');
      expect(result.current.editedValues['pro:ai_credits']).toBe('40');
      expect(result.current.editedValues['studio:ai_credits']).toBe('200');
      expect(result.current.editedValues['business:ai_credits']).toBeUndefined();
    });
  });

  describe('toast management', () => {
    it('clears toasts', async () => {
      fetchAllTierLimitsMock.mockRejectedValue(new Error('API failure'));

      const { result } = renderHook(() => useTierLimitsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.toasts).toHaveLength(1);

      act(() => {
        result.current.clearToasts();
      });

      expect(result.current.toasts).toHaveLength(0);
    });

    it('accumulates multiple toasts', async () => {
      saveTierLimitMock.mockRejectedValue(new Error('Save failed'));

      const { result } = renderHook(() => useTierLimitsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      const soloLimit = mockLimits.solo[0];
      const proLimit = mockLimits.pro[0];

      act(() => {
        result.current.updateEditedValue('solo:ai_credits', '15');
        result.current.updateEditedValue('pro:ai_credits', '25');
      });

      await act(async () => {
        await result.current.handleSave('solo', soloLimit);
      });

      await act(async () => {
        await result.current.handleSave('pro', proLimit);
      });

      expect(result.current.toasts.filter((t) => t.type === 'error')).toHaveLength(2);
    });
  });

  describe('utility functions', () => {
    it('exposes getEditKey function', async () => {
      const { result } = renderHook(() => useTierLimitsForm());

      expect(result.current.getEditKey('solo', 'ai_credits')).toBe('solo:ai_credits');
    });

    it('exposes getTierLabel function', async () => {
      const { result } = renderHook(() => useTierLimitsForm());

      expect(result.current.getTierLabel('solo')).toBe('Solo');
    });

    it('exposes getTierColor function', async () => {
      const { result } = renderHook(() => useTierLimitsForm());

      expect(result.current.getTierColor('solo')).toBe('text-blue-400');
      expect(result.current.getTierColor('pro')).toBe('text-purple-400');
    });

    it('exposes isUnlimitedValue function', async () => {
      const { result } = renderHook(() => useTierLimitsForm());

      expect(result.current.isUnlimitedValue(-1)).toBe(true);
      expect(result.current.isUnlimitedValue(0)).toBe(false);
      expect(result.current.isUnlimitedValue(100)).toBe(false);
    });

    it('exposes findAICreditsLimit function', async () => {
      const { result } = renderHook(() => useTierLimitsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      const soloLimit = result.current.findAICreditsLimit(result.current.limits.solo);
      expect(soloLimit?.tier_id).toBe('solo');
      expect(soloLimit?.limit_key).toBe('ai_credits');
    });

    it('exposes getDisplayValue function', async () => {
      const { result } = renderHook(() => useTierLimitsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      const soloLimit = mockLimits.solo[0];
      expect(result.current.getDisplayValue(soloLimit)).toBe('5.00');

      const businessLimit = mockLimits.business[0];
      expect(result.current.getDisplayValue(businessLimit)).toBe('unlimited');
    });

    it('exposes TIER_OPTIONS', async () => {
      const { result } = renderHook(() => useTierLimitsForm());

      expect(result.current.TIER_OPTIONS).toBeDefined();
      expect(Array.isArray(result.current.TIER_OPTIONS)).toBe(true);
      expect(result.current.TIER_OPTIONS.some((t) => t.value === 'solo')).toBe(true);
    });
  });

  describe('fetchLimits', () => {
    it('can be called manually to refresh', async () => {
      const { result } = renderHook(() => useTierLimitsForm());

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      fetchAllTierLimitsMock.mockClear();
      const newLimits = { ...mockLimits };
      newLimits.solo[0] = { ...newLimits.solo[0], display_dollars: 10 };
      fetchAllTierLimitsMock.mockResolvedValue(newLimits);

      await act(async () => {
        await result.current.fetchLimits();
      });

      expect(fetchAllTierLimitsMock).toHaveBeenCalledTimes(1);
      expect(result.current.limits.solo[0].display_dollars).toBe(10);
    });
  });
});
