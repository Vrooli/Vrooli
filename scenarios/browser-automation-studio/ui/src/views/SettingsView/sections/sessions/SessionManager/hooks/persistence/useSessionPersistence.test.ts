import { renderHook, act, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { useSessionPersistence } from './useSessionPersistence';
import type { UseProfileSettingsReturn } from '../settings';
import type { BrowserProfile } from '@/domains/recording/types/types';

describe('useSessionPersistence', () => {
  const mockProfile: BrowserProfile = {
    preset: 'balanced',
    fingerprint: { viewport_width: 1920 },
  };

  const createMockSettings = (overrides: Partial<UseProfileSettingsReturn> = {}): UseProfileSettingsReturn => ({
    preset: 'balanced',
    fingerprint: {},
    behavior: {},
    antiDetection: {},
    proxy: {},
    extraHeaders: {},
    applyPreset: vi.fn(),
    updateFingerprint: vi.fn(),
    updateBehavior: vi.fn(),
    updateAntiDetection: vi.fn(),
    updateProxy: vi.fn(),
    addExtraHeader: vi.fn(),
    updateExtraHeader: vi.fn(),
    removeExtraHeader: vi.fn(),
    resetToInitial: vi.fn(),
    getCurrentProfile: vi.fn(() => mockProfile),
    hasChanges: false,
    ...overrides,
  });

  let mockOnSave: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    mockOnSave = vi.fn().mockResolvedValue(undefined);
  });

  describe('isDirty derivation', () => {
    it('reflects false when settings.hasChanges is false', () => {
      const settings = createMockSettings({ hasChanges: false });
      const { result } = renderHook(() =>
        useSessionPersistence({ settings, onSave: mockOnSave })
      );

      expect(result.current.isDirty).toBe(false);
    });

    it('reflects true when settings.hasChanges is true', () => {
      const settings = createMockSettings({ hasChanges: true });
      const { result } = renderHook(() =>
        useSessionPersistence({ settings, onSave: mockOnSave })
      );

      expect(result.current.isDirty).toBe(true);
    });

    it('updates when settings.hasChanges changes', () => {
      const settingsWithoutChanges = createMockSettings({ hasChanges: false });
      const settingsWithChanges = createMockSettings({ hasChanges: true });

      const { result, rerender } = renderHook(
        ({ settings }) => useSessionPersistence({ settings, onSave: mockOnSave }),
        { initialProps: { settings: settingsWithoutChanges } }
      );

      expect(result.current.isDirty).toBe(false);

      rerender({ settings: settingsWithChanges });

      expect(result.current.isDirty).toBe(true);
    });
  });

  describe('handleSave', () => {
    it('sets saving to true during save operation', async () => {
      let resolveSave: () => void;
      const slowSave = new Promise<void>((resolve) => {
        resolveSave = resolve;
      });
      mockOnSave.mockReturnValue(slowSave);

      const settings = createMockSettings();
      const { result } = renderHook(() =>
        useSessionPersistence({ settings, onSave: mockOnSave })
      );

      expect(result.current.saving).toBe(false);

      let savePromise: Promise<boolean>;
      act(() => {
        savePromise = result.current.handleSave();
      });

      expect(result.current.saving).toBe(true);

      await act(async () => {
        resolveSave!();
        await savePromise;
      });

      expect(result.current.saving).toBe(false);
    });

    it('calls onSave with current profile', async () => {
      const settings = createMockSettings();
      const { result } = renderHook(() =>
        useSessionPersistence({ settings, onSave: mockOnSave })
      );

      await act(async () => {
        await result.current.handleSave();
      });

      expect(settings.getCurrentProfile).toHaveBeenCalled();
      expect(mockOnSave).toHaveBeenCalledWith(mockProfile);
    });

    it('returns true on successful save', async () => {
      const settings = createMockSettings();
      const { result } = renderHook(() =>
        useSessionPersistence({ settings, onSave: mockOnSave })
      );

      let success: boolean;
      await act(async () => {
        success = await result.current.handleSave();
      });

      expect(success!).toBe(true);
    });

    it('clears error on successful save', async () => {
      const settings = createMockSettings();
      mockOnSave
        .mockRejectedValueOnce(new Error('First failure'))
        .mockResolvedValueOnce(undefined);

      const { result } = renderHook(() =>
        useSessionPersistence({ settings, onSave: mockOnSave })
      );

      // First save fails
      await act(async () => {
        await result.current.handleSave();
      });
      expect(result.current.error).toBe('First failure');

      // Second save succeeds
      await act(async () => {
        await result.current.handleSave();
      });
      expect(result.current.error).toBe(null);
    });
  });

  describe('error handling', () => {
    it('sets error on failed save', async () => {
      const settings = createMockSettings();
      mockOnSave.mockRejectedValue(new Error('Save failed'));

      const { result } = renderHook(() =>
        useSessionPersistence({ settings, onSave: mockOnSave })
      );

      await act(async () => {
        await result.current.handleSave();
      });

      expect(result.current.error).toBe('Save failed');
    });

    it('returns false on failed save', async () => {
      const settings = createMockSettings();
      mockOnSave.mockRejectedValue(new Error('Save failed'));

      const { result } = renderHook(() =>
        useSessionPersistence({ settings, onSave: mockOnSave })
      );

      let success: boolean;
      await act(async () => {
        success = await result.current.handleSave();
      });

      expect(success!).toBe(false);
    });

    it('handles non-Error thrown values', async () => {
      const settings = createMockSettings();
      mockOnSave.mockRejectedValue('string error');

      const { result } = renderHook(() =>
        useSessionPersistence({ settings, onSave: mockOnSave })
      );

      await act(async () => {
        await result.current.handleSave();
      });

      expect(result.current.error).toBe('Failed to save profile');
    });

    it('clearError clears the error state', async () => {
      const settings = createMockSettings();
      mockOnSave.mockRejectedValue(new Error('Save failed'));

      const { result } = renderHook(() =>
        useSessionPersistence({ settings, onSave: mockOnSave })
      );

      await act(async () => {
        await result.current.handleSave();
      });
      expect(result.current.error).toBe('Save failed');

      act(() => {
        result.current.clearError();
      });
      expect(result.current.error).toBe(null);
    });
  });

  describe('reset', () => {
    it('calls settings.resetToInitial', () => {
      const settings = createMockSettings();
      const { result } = renderHook(() =>
        useSessionPersistence({ settings, onSave: mockOnSave })
      );

      act(() => {
        result.current.reset();
      });

      expect(settings.resetToInitial).toHaveBeenCalled();
    });

    it('clears error on reset', async () => {
      const settings = createMockSettings();
      mockOnSave.mockRejectedValue(new Error('Save failed'));

      const { result } = renderHook(() =>
        useSessionPersistence({ settings, onSave: mockOnSave })
      );

      await act(async () => {
        await result.current.handleSave();
      });
      expect(result.current.error).toBe('Save failed');

      act(() => {
        result.current.reset();
      });
      expect(result.current.error).toBe(null);
    });
  });

  describe('initial state', () => {
    it('starts with saving false', () => {
      const settings = createMockSettings();
      const { result } = renderHook(() =>
        useSessionPersistence({ settings, onSave: mockOnSave })
      );

      expect(result.current.saving).toBe(false);
    });

    it('starts with no error', () => {
      const settings = createMockSettings();
      const { result } = renderHook(() =>
        useSessionPersistence({ settings, onSave: mockOnSave })
      );

      expect(result.current.error).toBe(null);
    });
  });

  describe('callback stability', () => {
    it('maintains stable callback references', () => {
      const settings = createMockSettings();
      const { result, rerender } = renderHook(() =>
        useSessionPersistence({ settings, onSave: mockOnSave })
      );

      const refs = {
        handleSave: result.current.handleSave,
        reset: result.current.reset,
        clearError: result.current.clearError,
      };

      rerender();

      // Note: handleSave and reset depend on settings/onSave, may change if those change
      expect(result.current.clearError).toBe(refs.clearError);
    });
  });
});
