/**
 * Tests for ViewportSyncManager
 *
 * These tests verify:
 * - Viewport computation and clamping
 * - Debouncing behavior
 * - Resize detection
 * - Backend sync coordination
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import {
  useViewportSyncManager,
  viewportsEqual,
  getAspectRatio,
  fitViewportToBounds,
} from './ViewportSyncManager';

// Mock getConfig
vi.mock('@/config', () => ({
  getConfig: vi.fn().mockResolvedValue({
    API_URL: 'http://test-api',
  }),
}));

// Mock fetch
const mockFetch = vi.fn();
global.fetch = mockFetch;

describe('ViewportSyncManager', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mockFetch.mockReset();
    mockFetch.mockResolvedValue({ ok: true });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe('useViewportSyncManager hook', () => {
    it('should initialize with null viewport', () => {
      const { result } = renderHook(() =>
        useViewportSyncManager({ sessionId: 'test-session' })
      );

      expect(result.current.state.viewport).toBeNull();
      expect(result.current.state.isResizing).toBe(false);
      expect(result.current.state.isSyncing).toBe(false);
    });

    it('should clamp viewport to min/max dimensions', () => {
      const { result } = renderHook(() =>
        useViewportSyncManager({
          sessionId: 'test-session',
          minDimension: 320,
          maxDimension: 3840,
        })
      );

      // Test clamping small values
      const small = result.current.getClampedViewport({ width: 100, height: 100 });
      expect(small.width).toBe(320);
      expect(small.height).toBe(320);

      // Test clamping large values
      const large = result.current.getClampedViewport({ width: 5000, height: 5000 });
      expect(large.width).toBe(3840);
      expect(large.height).toBe(3840);

      // Test normal values pass through
      const normal = result.current.getClampedViewport({ width: 1920, height: 1080 });
      expect(normal.width).toBe(1920);
      expect(normal.height).toBe(1080);
    });

    it('should update viewport immediately on bounds change', () => {
      const { result } = renderHook(() =>
        useViewportSyncManager({ sessionId: 'test-session' })
      );

      act(() => {
        result.current.updateFromBounds({ width: 800, height: 600 });
      });

      expect(result.current.state.viewport).toEqual({ width: 800, height: 600 });
    });

    it('should detect rapid resize', async () => {
      const { result } = renderHook(() =>
        useViewportSyncManager({
          sessionId: 'test-session',
          resizeThresholdMs: 100,
        })
      );

      // First update
      act(() => {
        result.current.updateFromBounds({ width: 800, height: 600 });
      });

      // Rapid second update (within threshold)
      act(() => {
        vi.advanceTimersByTime(50);
        result.current.updateFromBounds({ width: 850, height: 600 });
      });

      expect(result.current.state.isResizing).toBe(true);

      // Wait for resize to settle
      act(() => {
        vi.advanceTimersByTime(300);
      });

      expect(result.current.state.isResizing).toBe(false);
    });

    it('should debounce backend sync', async () => {
      const { result } = renderHook(() =>
        useViewportSyncManager({
          sessionId: 'test-session',
          debounceMs: 200,
        })
      );

      // Multiple rapid updates
      act(() => {
        result.current.updateFromBounds({ width: 800, height: 600 });
      });
      act(() => {
        vi.advanceTimersByTime(50);
        result.current.updateFromBounds({ width: 850, height: 600 });
      });
      act(() => {
        vi.advanceTimersByTime(50);
        result.current.updateFromBounds({ width: 900, height: 600 });
      });

      // No sync yet
      expect(mockFetch).not.toHaveBeenCalled();

      // Wait for debounce
      await act(async () => {
        vi.advanceTimersByTime(200);
        await Promise.resolve();
      });

      // Should only sync once with final viewport
      expect(mockFetch).toHaveBeenCalledTimes(1);
      expect(mockFetch).toHaveBeenCalledWith(
        'http://test-api/recordings/live/test-session/viewport',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ width: 900, height: 600 }),
        })
      );
    });

    it('should not sync when sessionId is null', async () => {
      const { result } = renderHook(() =>
        useViewportSyncManager({
          sessionId: null,
          debounceMs: 100,
        })
      );

      act(() => {
        result.current.updateFromBounds({ width: 800, height: 600 });
      });

      await act(async () => {
        vi.advanceTimersByTime(200);
        await Promise.resolve();
      });

      expect(mockFetch).not.toHaveBeenCalled();
    });

    it('should skip sync when viewport unchanged', async () => {
      const { result } = renderHook(() =>
        useViewportSyncManager({
          sessionId: 'test-session',
          debounceMs: 100,
        })
      );

      // First update - should sync
      act(() => {
        result.current.updateFromBounds({ width: 800, height: 600 });
      });
      await act(async () => {
        vi.advanceTimersByTime(150);
        await Promise.resolve();
      });
      expect(mockFetch).toHaveBeenCalledTimes(1);

      // Same viewport - should not sync again
      mockFetch.mockClear();
      act(() => {
        result.current.updateFromBounds({ width: 800, height: 600 });
      });
      await act(async () => {
        vi.advanceTimersByTime(150);
        await Promise.resolve();
      });
      expect(mockFetch).not.toHaveBeenCalled();
    });

    it('should reset state on session change', () => {
      const { result, rerender } = renderHook(
        ({ sessionId }) => useViewportSyncManager({ sessionId }),
        { initialProps: { sessionId: 'session-1' } }
      );

      // Set some state
      act(() => {
        result.current.updateFromBounds({ width: 800, height: 600 });
      });
      expect(result.current.state.viewport).not.toBeNull();

      // Change session
      rerender({ sessionId: 'session-2' });

      // State should be reset
      expect(result.current.state.viewport).toBeNull();
    });

    it('should handle sync errors', async () => {
      mockFetch.mockRejectedValueOnce(new Error('Network error'));

      const { result } = renderHook(() =>
        useViewportSyncManager({
          sessionId: 'test-session',
          debounceMs: 100,
        })
      );

      act(() => {
        result.current.updateFromBounds({ width: 800, height: 600 });
      });

      await act(async () => {
        vi.advanceTimersByTime(150);
        await Promise.resolve();
      });

      expect(result.current.state.syncError).toBe('Network error');
    });

    it('should support force sync', async () => {
      const { result } = renderHook(() =>
        useViewportSyncManager({
          sessionId: 'test-session',
          debounceMs: 1000, // Long debounce
        })
      );

      act(() => {
        result.current.updateFromBounds({ width: 800, height: 600 });
      });

      // Force sync immediately (don't wait for debounce)
      await act(async () => {
        await result.current.forceSync();
      });

      expect(mockFetch).toHaveBeenCalledTimes(1);
    });
  });

  describe('viewportsEqual', () => {
    it('should return true for equal viewports', () => {
      expect(
        viewportsEqual({ width: 800, height: 600 }, { width: 800, height: 600 })
      ).toBe(true);
    });

    it('should return true for viewports within tolerance', () => {
      expect(
        viewportsEqual({ width: 800, height: 600 }, { width: 801, height: 600 }, 1)
      ).toBe(true);
    });

    it('should return false for different viewports', () => {
      expect(
        viewportsEqual({ width: 800, height: 600 }, { width: 900, height: 600 })
      ).toBe(false);
    });

    it('should handle null viewports', () => {
      expect(viewportsEqual(null, null)).toBe(true);
      expect(viewportsEqual({ width: 800, height: 600 }, null)).toBe(false);
      expect(viewportsEqual(null, { width: 800, height: 600 })).toBe(false);
    });
  });

  describe('getAspectRatio', () => {
    it('should calculate aspect ratio correctly', () => {
      expect(getAspectRatio({ width: 1920, height: 1080 })).toBeCloseTo(16 / 9);
      expect(getAspectRatio({ width: 800, height: 600 })).toBeCloseTo(4 / 3);
      expect(getAspectRatio({ width: 1000, height: 1000 })).toBe(1);
    });
  });

  describe('fitViewportToBounds', () => {
    it('should scale down to fit width', () => {
      const result = fitViewportToBounds(
        { width: 1920, height: 1080 },
        { width: 960, height: 1000 }
      );
      expect(result.width).toBe(960);
      expect(result.height).toBe(540);
    });

    it('should scale down to fit height', () => {
      const result = fitViewportToBounds(
        { width: 1920, height: 1080 },
        { width: 2000, height: 540 }
      );
      expect(result.width).toBe(960);
      expect(result.height).toBe(540);
    });

    it('should not scale up beyond original size', () => {
      const result = fitViewportToBounds(
        { width: 800, height: 600 },
        { width: 1920, height: 1080 }
      );
      // Should fit the smaller dimension (height at 600)
      // Scale factor: min(1920/800, 1080/600) = min(2.4, 1.8) = 1.8
      expect(result.width).toBe(1440);
      expect(result.height).toBe(1080);
    });
  });
});
