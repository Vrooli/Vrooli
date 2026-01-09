/**
 * ViewportProvider Tests
 *
 * Tests for the centralized viewport context provider.
 * Verifies state management, actions, and mismatch detection.
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, act, waitFor, renderHook } from '@testing-library/react';
import { ViewportProvider, useViewport, useViewportOptional } from './ViewportProvider';
import type { ReactNode } from 'react';

// Mock getConfig
vi.mock('@/config', () => ({
  getConfig: vi.fn().mockResolvedValue({
    API_URL: 'http://test-api',
  }),
}));

// Mock fetch
const mockFetch = vi.fn();
global.fetch = mockFetch;

describe('ViewportProvider', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mockFetch.mockReset();
    mockFetch.mockResolvedValue({ ok: true });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  // Helper to create a wrapper with ViewportProvider
  const createWrapper = (sessionId: string | null = 'test-session') => {
    return ({ children }: { children: ReactNode }) => (
      <ViewportProvider sessionId={sessionId}>{children}</ViewportProvider>
    );
  };

  describe('useViewport hook', () => {
    it('should throw error when used outside provider', () => {
      // Suppress console.error for expected error
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

      expect(() => {
        renderHook(() => useViewport());
      }).toThrow('useViewport must be used within a ViewportProvider');

      consoleSpy.mockRestore();
    });

    it('should return context when used within provider', () => {
      const { result } = renderHook(() => useViewport(), {
        wrapper: createWrapper(),
      });

      expect(result.current).toBeDefined();
      expect(result.current.browserViewport).toBeNull();
      expect(result.current.actualViewport).toBeNull();
      expect(result.current.sessionId).toBe('test-session');
    });
  });

  describe('useViewportOptional hook', () => {
    it('should return null when used outside provider', () => {
      const { result } = renderHook(() => useViewportOptional());
      expect(result.current).toBeNull();
    });

    it('should return context when used within provider', () => {
      const { result } = renderHook(() => useViewportOptional(), {
        wrapper: createWrapper(),
      });

      expect(result.current).not.toBeNull();
      expect(result.current?.browserViewport).toBeNull();
    });
  });

  describe('initial state', () => {
    it('should initialize with null viewports', () => {
      const { result } = renderHook(() => useViewport(), {
        wrapper: createWrapper(),
      });

      expect(result.current.browserViewport).toBeNull();
      expect(result.current.actualViewport).toBeNull();
      expect(result.current.hasMismatch).toBe(false);
      expect(result.current.mismatchReason).toBeNull();
    });

    it('should accept initial actual viewport', () => {
      const Wrapper = ({ children }: { children: ReactNode }) => (
        <ViewportProvider
          sessionId="test-session"
          actualViewport={{ width: 1920, height: 1080 }}
        >
          {children}
        </ViewportProvider>
      );

      const { result } = renderHook(() => useViewport(), {
        wrapper: Wrapper,
      });

      expect(result.current.actualViewport).toEqual({ width: 1920, height: 1080 });
    });

    it('should expose sessionId', () => {
      const { result } = renderHook(() => useViewport(), {
        wrapper: createWrapper('my-session-id'),
      });

      expect(result.current.sessionId).toBe('my-session-id');
    });
  });

  describe('updateFromBounds', () => {
    it('should update browser viewport', () => {
      const { result } = renderHook(() => useViewport(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateFromBounds({ width: 1280, height: 720 });
      });

      expect(result.current.browserViewport).toEqual({ width: 1280, height: 720 });
    });

    it('should clamp dimensions to valid range', () => {
      const { result } = renderHook(() => useViewport(), {
        wrapper: createWrapper(),
      });

      // Test small dimensions
      act(() => {
        result.current.updateFromBounds({ width: 100, height: 100 });
      });

      expect(result.current.browserViewport).toEqual({ width: 320, height: 320 });

      // Test large dimensions
      act(() => {
        result.current.updateFromBounds({ width: 5000, height: 5000 });
      });

      expect(result.current.browserViewport).toEqual({ width: 3840, height: 3840 });
    });

    it('should round dimensions', () => {
      const { result } = renderHook(() => useViewport(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateFromBounds({ width: 1280.7, height: 720.3 });
      });

      expect(result.current.browserViewport).toEqual({ width: 1281, height: 720 });
    });

    it('should debounce backend sync', async () => {
      const { result } = renderHook(() => useViewport(), {
        wrapper: createWrapper(),
      });

      // Multiple rapid updates
      act(() => {
        result.current.updateFromBounds({ width: 800, height: 600 });
      });
      act(() => {
        vi.advanceTimersByTime(50);
        result.current.updateFromBounds({ width: 900, height: 700 });
      });
      act(() => {
        vi.advanceTimersByTime(50);
        result.current.updateFromBounds({ width: 1000, height: 800 });
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
          body: JSON.stringify({ width: 1000, height: 800 }),
        })
      );
    });

    it('should not sync when sessionId is null', async () => {
      const { result } = renderHook(() => useViewport(), {
        wrapper: createWrapper(null),
      });

      act(() => {
        result.current.updateFromBounds({ width: 800, height: 600 });
      });

      await act(async () => {
        vi.advanceTimersByTime(300);
        await Promise.resolve();
      });

      expect(mockFetch).not.toHaveBeenCalled();
    });
  });

  describe('setActualViewport', () => {
    it('should update actual viewport', () => {
      const { result } = renderHook(() => useViewport(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.setActualViewport({ width: 1920, height: 1080 });
      });

      expect(result.current.actualViewport).toEqual({ width: 1920, height: 1080 });
    });

    it('should allow clearing actual viewport', () => {
      const Wrapper = ({ children }: { children: ReactNode }) => (
        <ViewportProvider
          sessionId="test-session"
          actualViewport={{ width: 1920, height: 1080 }}
        >
          {children}
        </ViewportProvider>
      );

      const { result } = renderHook(() => useViewport(), {
        wrapper: Wrapper,
      });

      expect(result.current.actualViewport).not.toBeNull();

      act(() => {
        result.current.setActualViewport(null);
      });

      expect(result.current.actualViewport).toBeNull();
    });
  });

  describe('viewport source attribution', () => {
    it('should accept actualViewport with source and reason', () => {
      const Wrapper = ({ children }: { children: ReactNode }) => (
        <ViewportProvider
          sessionId="test-session"
          actualViewport={{
            width: 1920,
            height: 1080,
            source: 'fingerprint',
            reason: 'Browser profile specifies 1920x1080',
          }}
        >
          {children}
        </ViewportProvider>
      );

      const { result } = renderHook(() => useViewport(), {
        wrapper: Wrapper,
      });

      expect(result.current.actualViewport).toEqual({
        width: 1920,
        height: 1080,
        source: 'fingerprint',
        reason: 'Browser profile specifies 1920x1080',
      });
    });

    it('should use reason from actualViewport for mismatch explanation', () => {
      const Wrapper = ({ children }: { children: ReactNode }) => (
        <ViewportProvider
          sessionId="test-session"
          actualViewport={{
            width: 1920,
            height: 1080,
            source: 'fingerprint',
            reason: 'Browser profile specifies 1920x1080',
          }}
        >
          {children}
        </ViewportProvider>
      );

      const { result } = renderHook(() => useViewport(), {
        wrapper: Wrapper,
      });

      // Set browser viewport to something different
      act(() => {
        result.current.updateFromBounds({ width: 1280, height: 720 });
      });

      expect(result.current.hasMismatch).toBe(true);
      expect(result.current.mismatchReason).toBe('Browser profile specifies 1920x1080');
    });

    it('should fall back to default mismatch reason when reason not provided', () => {
      const Wrapper = ({ children }: { children: ReactNode }) => (
        <ViewportProvider
          sessionId="test-session"
          actualViewport={{ width: 1920, height: 1080 }}
        >
          {children}
        </ViewportProvider>
      );

      const { result } = renderHook(() => useViewport(), {
        wrapper: Wrapper,
      });

      act(() => {
        result.current.updateFromBounds({ width: 1280, height: 720 });
      });

      expect(result.current.hasMismatch).toBe(true);
      expect(result.current.mismatchReason).toBe('Session profile has viewport override configured');
    });

    it('should handle fingerprint_partial source', () => {
      const Wrapper = ({ children }: { children: ReactNode }) => (
        <ViewportProvider
          sessionId="test-session"
          actualViewport={{
            width: 1920,
            height: 720,
            source: 'fingerprint_partial',
            reason: 'Profile width 1920px override, using requested height 720px',
          }}
        >
          {children}
        </ViewportProvider>
      );

      const { result } = renderHook(() => useViewport(), {
        wrapper: Wrapper,
      });

      expect(result.current.actualViewport?.source).toBe('fingerprint_partial');
      expect(result.current.actualViewport?.reason).toContain('override');
    });

    it('should handle setActualViewport with full attribution', () => {
      const { result } = renderHook(() => useViewport(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.setActualViewport({
          width: 1280,
          height: 720,
          source: 'requested',
          reason: 'Using requested 1280x720',
        });
      });

      expect(result.current.actualViewport).toEqual({
        width: 1280,
        height: 720,
        source: 'requested',
        reason: 'Using requested 1280x720',
      });
    });
  });

  describe('mismatch detection', () => {
    it('should detect no mismatch when viewports are null', () => {
      const { result } = renderHook(() => useViewport(), {
        wrapper: createWrapper(),
      });

      expect(result.current.hasMismatch).toBe(false);
      expect(result.current.mismatchReason).toBeNull();
    });

    it('should detect no mismatch when viewports match', () => {
      const { result } = renderHook(() => useViewport(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateFromBounds({ width: 1280, height: 720 });
        result.current.setActualViewport({ width: 1280, height: 720 });
      });

      expect(result.current.hasMismatch).toBe(false);
      expect(result.current.mismatchReason).toBeNull();
    });

    it('should detect no mismatch within tolerance', () => {
      const { result } = renderHook(() => useViewport(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateFromBounds({ width: 1280, height: 720 });
        result.current.setActualViewport({ width: 1283, height: 723 }); // Within 5px tolerance
      });

      expect(result.current.hasMismatch).toBe(false);
    });

    it('should detect mismatch when viewports differ', () => {
      const { result } = renderHook(() => useViewport(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateFromBounds({ width: 1280, height: 720 });
        result.current.setActualViewport({ width: 1920, height: 1080 });
      });

      expect(result.current.hasMismatch).toBe(true);
      expect(result.current.mismatchReason).toBe('Session profile has viewport override configured');
    });

    it('should detect mismatch when height differs beyond tolerance', () => {
      const { result } = renderHook(() => useViewport(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateFromBounds({ width: 1280, height: 720 });
        result.current.setActualViewport({ width: 1280, height: 900 }); // Height stuck at 900
      });

      expect(result.current.hasMismatch).toBe(true);
    });
  });

  describe('forceSync', () => {
    it('should sync immediately without waiting for debounce', async () => {
      const { result } = renderHook(() => useViewport(), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.updateFromBounds({ width: 800, height: 600 });
      });

      // Force sync immediately
      await act(async () => {
        await result.current.forceSync();
      });

      expect(mockFetch).toHaveBeenCalledTimes(1);
    });
  });

  describe('reset', () => {
    it('should reset viewport state', () => {
      const { result } = renderHook(() => useViewport(), {
        wrapper: createWrapper(),
      });

      // Set some state
      act(() => {
        result.current.updateFromBounds({ width: 800, height: 600 });
        result.current.setActualViewport({ width: 1920, height: 1080 });
      });

      expect(result.current.browserViewport).not.toBeNull();

      // Reset
      act(() => {
        result.current.reset();
      });

      expect(result.current.browserViewport).toBeNull();
      // Note: actualViewport is managed separately and not reset by syncManager.reset()
    });
  });

  describe('getClampedViewport', () => {
    it('should clamp dimensions correctly', () => {
      const { result } = renderHook(() => useViewport(), {
        wrapper: createWrapper(),
      });

      expect(result.current.getClampedViewport({ width: 100, height: 100 })).toEqual({
        width: 320,
        height: 320,
      });

      expect(result.current.getClampedViewport({ width: 5000, height: 5000 })).toEqual({
        width: 3840,
        height: 3840,
      });

      expect(result.current.getClampedViewport({ width: 1280, height: 720 })).toEqual({
        width: 1280,
        height: 720,
      });
    });
  });

  describe('syncState', () => {
    it('should expose sync state', () => {
      const { result } = renderHook(() => useViewport(), {
        wrapper: createWrapper(),
      });

      expect(result.current.syncState).toBeDefined();
      expect(result.current.syncState.viewport).toBeNull();
      expect(result.current.syncState.isResizing).toBe(false);
      expect(result.current.syncState.isSyncing).toBe(false);
      expect(result.current.syncState.syncError).toBeNull();
    });

    it('should detect resizing during rapid updates', () => {
      const { result } = renderHook(() => useViewport(), {
        wrapper: createWrapper(),
      });

      // First update
      act(() => {
        result.current.updateFromBounds({ width: 800, height: 600 });
      });

      // Rapid second update
      act(() => {
        vi.advanceTimersByTime(50);
        result.current.updateFromBounds({ width: 850, height: 600 });
      });

      expect(result.current.syncState.isResizing).toBe(true);

      // Wait for resize to settle
      act(() => {
        vi.advanceTimersByTime(300);
      });

      expect(result.current.syncState.isResizing).toBe(false);
    });
  });

  describe('component rendering', () => {
    it('should render children', () => {
      render(
        <ViewportProvider sessionId="test-session">
          <div data-testid="child">Child content</div>
        </ViewportProvider>
      );

      expect(screen.getByTestId('child')).toBeInTheDocument();
    });

    it('should provide context to nested children', () => {
      const Consumer = () => {
        const viewport = useViewport();
        return <div data-testid="session">{viewport.sessionId}</div>;
      };

      render(
        <ViewportProvider sessionId="nested-test">
          <div>
            <div>
              <Consumer />
            </div>
          </div>
        </ViewportProvider>
      );

      expect(screen.getByTestId('session')).toHaveTextContent('nested-test');
    });
  });
});
