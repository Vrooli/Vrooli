import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useAppLimitsForm } from './useAppLimitsForm';
import * as appLimitsService from '../services/appLimits.service';
import type { TierLimit } from '../../../shared/api';

// Mock the service module
vi.mock('../services/appLimits.service', async () => {
  const actual = await vi.importActual('../services/appLimits.service');
  return {
    ...actual,
    fetchAppLimits: vi.fn(),
    saveTierLimit: vi.fn(),
    createNewTierLimit: vi.fn(),
    removeTierLimit: vi.fn(),
  };
});

const mockFetchAppLimits = vi.mocked(appLimitsService.fetchAppLimits);
const mockSaveTierLimit = vi.mocked(appLimitsService.saveTierLimit);
const mockCreateNewTierLimit = vi.mocked(appLimitsService.createNewTierLimit);
const mockRemoveTierLimit = vi.mocked(appLimitsService.removeTierLimit);

const createMockLimit = (overrides: Partial<TierLimit> = {}): TierLimit => ({
  id: '1',
  tier_id: 'solo',
  limit_type: 'app_specific',
  limit_key: 'workflow_exports',
  limit_value: 10000000,
  cost_multiplier: 1000000,
  app_bundle_key: 'browser-automation-studio',
  reset_period: 'monthly',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
  display_dollars: 100,
  ...overrides,
});

describe('useAppLimitsForm', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockFetchAppLimits.mockResolvedValue({});
  });

  describe('initial state', () => {
    it('starts with loading state', async () => {
      const { result } = renderHook(() => useAppLimitsForm());
      expect(result.current.loading).toBe(true);
      await waitFor(() => { expect(result.current.loading).toBe(false); });
    });

    it('has default selected app', async () => {
      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });
      expect(result.current.selectedApp).toBe('browser-automation-studio');
    });

    it('has empty limits initially', async () => {
      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });
      expect(result.current.limits).toEqual({});
    });

    it('has default new limit form', async () => {
      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });
      expect(result.current.newLimit).toEqual({
        tier_id: 'solo',
        limit_key: '',
        display_dollars: '',
      });
    });
  });

  describe('loading limits', () => {
    it('fetches limits on mount', async () => {
      mockFetchAppLimits.mockResolvedValue({
        solo: [createMockLimit()],
      });

      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(mockFetchAppLimits).toHaveBeenCalledWith('browser-automation-studio');
      expect(result.current.limits.solo).toHaveLength(1);
    });

    it('refetches when selected app changes', async () => {
      mockFetchAppLimits.mockResolvedValue({});

      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(mockFetchAppLimits).toHaveBeenCalledTimes(1);

      // Note: In actual use, there would be multiple apps - here we test that
      // changing the app triggers a refetch
      act(() => {
        result.current.setSelectedApp('another-app');
      });

      await waitFor(() => {
        expect(mockFetchAppLimits).toHaveBeenCalledTimes(2);
      });
    });
  });

  describe('editing values', () => {
    it('sets edited value correctly', async () => {
      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      act(() => {
        result.current.setEditedValue('solo:workflow_exports', '50');
      });

      expect(result.current.editedValues['solo:workflow_exports']).toBe('50');
    });

    it('converts values to lowercase', async () => {
      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      act(() => {
        result.current.setEditedValue('solo:test', 'UNLIMITED');
      });

      expect(result.current.editedValues['solo:test']).toBe('unlimited');
    });

    it('clears edited value correctly', async () => {
      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      act(() => {
        result.current.setEditedValue('solo:test', '100');
      });
      expect(result.current.editedValues['solo:test']).toBe('100');

      act(() => {
        result.current.clearEditedValue('solo:test');
      });
      expect(result.current.editedValues['solo:test']).toBeUndefined();
    });

    it('isEdited returns true for edited values', async () => {
      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.isEdited('solo:test')).toBe(false);

      act(() => {
        result.current.setEditedValue('solo:test', '100');
      });

      expect(result.current.isEdited('solo:test')).toBe(true);
    });
  });

  describe('getEditedOrDisplayValue', () => {
    it('returns edited value when present', async () => {
      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      const limit = createMockLimit({ display_dollars: 100 });

      act(() => {
        result.current.setEditedValue('solo:workflow_exports', '50');
      });

      expect(result.current.getEditedOrDisplayValue('solo:workflow_exports', limit)).toBe('50');
    });

    it('returns display_dollars when not edited', async () => {
      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      const limit = createMockLimit({ display_dollars: 100 });

      expect(result.current.getEditedOrDisplayValue('solo:workflow_exports', limit)).toBe('100.00');
    });

    it('returns "unlimited" for negative limit values', async () => {
      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      const limit = createMockLimit({ limit_value: -1 });

      expect(result.current.getEditedOrDisplayValue('solo:workflow_exports', limit)).toBe(
        'unlimited'
      );
    });
  });

  describe('handleSave', () => {
    it('returns error when no changes to save', async () => {
      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      const limit = createMockLimit();
      const saveResult = await result.current.handleSave('solo', limit);

      expect(saveResult.success).toBe(false);
      expect(saveResult.message).toBe('No changes to save');
    });

    it('returns error for invalid value', async () => {
      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      act(() => {
        result.current.setEditedValue('solo:workflow_exports', 'invalid');
      });

      const limit = createMockLimit();
      const saveResult = await result.current.handleSave('solo', limit);

      expect(saveResult.success).toBe(false);
      expect(saveResult.message).toBe('Please enter a valid dollar amount');
    });

    it('saves valid dollar value', async () => {
      mockSaveTierLimit.mockResolvedValue(createMockLimit());
      mockFetchAppLimits.mockResolvedValue({});

      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      act(() => {
        result.current.setEditedValue('solo:workflow_exports', '50');
      });

      const limit = createMockLimit();
      let saveResult: { success: boolean; message: string };
      await act(async () => {
        saveResult = await result.current.handleSave('solo', limit);
      });

      expect(saveResult!.success).toBe(true);
      expect(mockSaveTierLimit).toHaveBeenCalledWith(
        'solo',
        'workflow_exports',
        { display_dollars: 50 },
        'browser-automation-studio'
      );
    });

    it('saves unlimited value', async () => {
      mockSaveTierLimit.mockResolvedValue(createMockLimit());
      mockFetchAppLimits.mockResolvedValue({});

      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      act(() => {
        result.current.setEditedValue('solo:workflow_exports', 'unlimited');
      });

      const limit = createMockLimit();
      let saveResult: { success: boolean; message: string };
      await act(async () => {
        saveResult = await result.current.handleSave('solo', limit);
      });

      expect(saveResult!.success).toBe(true);
      expect(mockSaveTierLimit).toHaveBeenCalledWith(
        'solo',
        'workflow_exports',
        { is_unlimited: true },
        'browser-automation-studio'
      );
    });

    it('handles API error', async () => {
      mockSaveTierLimit.mockRejectedValue(new Error('API Error'));

      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      act(() => {
        result.current.setEditedValue('solo:workflow_exports', '50');
      });

      const limit = createMockLimit();
      let saveResult: { success: boolean; message: string };
      await act(async () => {
        saveResult = await result.current.handleSave('solo', limit);
      });

      expect(saveResult!.success).toBe(false);
      expect(saveResult!.message).toBe('API Error');
    });

    it('sets saving state during save', async () => {
      let resolvePromise: () => void;
      mockSaveTierLimit.mockReturnValue(
        new Promise((resolve) => {
          resolvePromise = () => { resolve(createMockLimit()); };
        })
      );
      mockFetchAppLimits.mockResolvedValue({});

      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      act(() => {
        result.current.setEditedValue('solo:workflow_exports', '50');
      });

      const limit = createMockLimit();
      let savePromise: Promise<{ success: boolean; message: string }>;
      act(() => {
        savePromise = result.current.handleSave('solo', limit);
      });

      expect(result.current.saving).toBe('solo:workflow_exports');

      await act(async () => {
        resolvePromise!();
        await savePromise;
      });

      expect(result.current.saving).toBeNull();
    });
  });

  describe('handleAddLimit', () => {
    it('returns error for empty limit_key', async () => {
      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      let addResult: { success: boolean; message: string };
      await act(async () => {
        addResult = await result.current.handleAddLimit();
      });

      expect(addResult!.success).toBe(false);
      expect(addResult!.message).toBe('Please enter a limit key');
    });

    it('creates new limit successfully', async () => {
      mockCreateNewTierLimit.mockResolvedValue(createMockLimit());
      mockFetchAppLimits.mockResolvedValue({});

      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      act(() => {
        result.current.setNewLimit({ limit_key: 'test_limit', display_dollars: '100' });
        result.current.setShowAddLimit(true);
      });

      let addResult: { success: boolean; message: string };
      await act(async () => {
        addResult = await result.current.handleAddLimit();
      });

      expect(addResult!.success).toBe(true);
      expect(addResult!.message).toBe('New limit created');
      expect(result.current.showAddLimit).toBe(false);
      expect(result.current.newLimit.limit_key).toBe('');
    });

    it('handles API error', async () => {
      mockCreateNewTierLimit.mockRejectedValue(new Error('Create failed'));

      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      act(() => {
        result.current.setNewLimit({ limit_key: 'test_limit' });
      });

      let addResult: { success: boolean; message: string };
      await act(async () => {
        addResult = await result.current.handleAddLimit();
      });

      expect(addResult!.success).toBe(false);
      expect(addResult!.message).toBe('Create failed');
    });
  });

  describe('handleDeleteLimit', () => {
    it('deletes limit successfully', async () => {
      mockRemoveTierLimit.mockResolvedValue(undefined);
      mockFetchAppLimits.mockResolvedValue({});

      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      let deleteResult: { success: boolean; message: string };
      await act(async () => {
        deleteResult = await result.current.handleDeleteLimit('solo', 'workflow_exports');
      });

      expect(deleteResult!.success).toBe(true);
      expect(deleteResult!.message).toBe('Limit deleted');
      expect(mockRemoveTierLimit).toHaveBeenCalledWith(
        'solo',
        'workflow_exports',
        'browser-automation-studio'
      );
    });

    it('handles API error', async () => {
      mockRemoveTierLimit.mockRejectedValue(new Error('Delete failed'));

      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      let deleteResult: { success: boolean; message: string };
      await act(async () => {
        deleteResult = await result.current.handleDeleteLimit('solo', 'test');
      });

      expect(deleteResult!.success).toBe(false);
      expect(deleteResult!.message).toBe('Delete failed');
    });

    it('sets saving state during delete', async () => {
      let resolvePromise: () => void;
      mockRemoveTierLimit.mockReturnValue(
        new Promise((resolve) => {
          resolvePromise = () => { resolve(undefined); };
        })
      );
      mockFetchAppLimits.mockResolvedValue({});

      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      let deletePromise: Promise<{ success: boolean; message: string }>;
      act(() => {
        deletePromise = result.current.handleDeleteLimit('solo', 'test');
      });

      expect(result.current.saving).toBe('delete:solo:test');

      await act(async () => {
        resolvePromise!();
        await deletePromise;
      });

      expect(result.current.saving).toBeNull();
    });
  });

  describe('new limit form', () => {
    it('updates new limit form fields', async () => {
      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      act(() => {
        result.current.setNewLimit({ tier_id: 'pro', limit_key: 'test' });
      });

      expect(result.current.newLimit.tier_id).toBe('pro');
      expect(result.current.newLimit.limit_key).toBe('test');
    });

    it('resets new limit form', async () => {
      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      act(() => {
        result.current.setNewLimit({ tier_id: 'pro', limit_key: 'test', display_dollars: '100' });
      });

      act(() => {
        result.current.resetNewLimitForm();
      });

      expect(result.current.newLimit).toEqual({
        tier_id: 'solo',
        limit_key: '',
        display_dollars: '',
      });
    });

    it('toggles showAddLimit', async () => {
      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.showAddLimit).toBe(false);

      act(() => {
        result.current.setShowAddLimit(true);
      });

      expect(result.current.showAddLimit).toBe(true);
    });
  });

  describe('limitKeys computed value', () => {
    it('computes unique limit keys', async () => {
      mockFetchAppLimits.mockResolvedValue({
        solo: [
          createMockLimit({ limit_key: 'key1' }),
          createMockLimit({ limit_key: 'key2' }),
        ],
        pro: [createMockLimit({ limit_key: 'key1' })],
      });

      const { result } = renderHook(() => useAppLimitsForm());
      await waitFor(() => { expect(result.current.loading).toBe(false); });

      expect(result.current.limitKeys.size).toBe(2);
      expect(result.current.limitKeys.has('key1')).toBe(true);
      expect(result.current.limitKeys.has('key2')).toBe(true);
    });
  });
});
