/**
 * useUnifiedSidebar Hook Tests
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useUnifiedSidebar } from './useUnifiedSidebar';
import { SIDEBAR_DEFAULT_WIDTH, SIDEBAR_MIN_WIDTH, SIDEBAR_MAX_WIDTH, STORAGE_KEYS } from './types';

describe('useUnifiedSidebar', () => {
  // Mock localStorage
  const localStorageMock = (() => {
    let store: Record<string, string> = {};
    return {
      getItem: vi.fn((key: string) => store[key] ?? null),
      setItem: vi.fn((key: string, value: string) => {
        store[key] = value;
      }),
      removeItem: vi.fn((key: string) => {
        delete store[key];
      }),
      clear: vi.fn(() => {
        store = {};
      }),
    };
  })();

  beforeEach(() => {
    vi.useFakeTimers();
    Object.defineProperty(window, 'localStorage', {
      value: localStorageMock,
      writable: true,
    });
    localStorageMock.clear();
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  describe('initial state', () => {
    it('should initialize with default values', () => {
      const { result } = renderHook(() => useUnifiedSidebar());

      expect(result.current.isOpen).toBe(true);
      expect(result.current.activeTab).toBe('timeline');
      expect(result.current.width).toBe(SIDEBAR_DEFAULT_WIDTH);
      expect(result.current.minWidth).toBe(SIDEBAR_MIN_WIDTH);
      expect(result.current.maxWidth).toBe(SIDEBAR_MAX_WIDTH);
      expect(result.current.isResizing).toBe(false);
      expect(result.current.timelineActivity).toBe(false);
      expect(result.current.autoActivity).toBe(false);
    });

    it('should respect initialTab option', () => {
      const { result } = renderHook(() => useUnifiedSidebar({ initialTab: 'auto' }));

      expect(result.current.activeTab).toBe('auto');
    });

    it('should respect initialWidth option', () => {
      const { result } = renderHook(() => useUnifiedSidebar({ initialWidth: 450 }));

      expect(result.current.width).toBe(450);
    });

    it('should respect initialOpen option', () => {
      const { result } = renderHook(() => useUnifiedSidebar({ initialOpen: false }));

      expect(result.current.isOpen).toBe(false);
    });

    it('should read from localStorage if no initial options', () => {
      localStorageMock.getItem.mockImplementation((key: string) => {
        if (key === STORAGE_KEYS.SIDEBAR_TAB) return 'auto';
        if (key === STORAGE_KEYS.SIDEBAR_WIDTH) return '500';
        if (key === STORAGE_KEYS.SIDEBAR_OPEN) return 'false';
        return null;
      });

      // Call with empty options to trigger localStorage read
      const { result } = renderHook(() => useUnifiedSidebar());

      expect(result.current.activeTab).toBe('auto');
      expect(result.current.width).toBe(500);
      expect(result.current.isOpen).toBe(false);
    });
  });

  describe('tab switching', () => {
    it('should switch tabs', () => {
      const { result } = renderHook(() => useUnifiedSidebar());

      act(() => {
        result.current.setActiveTab('auto');
      });

      expect(result.current.activeTab).toBe('auto');
    });

    it('should persist tab to localStorage', () => {
      const { result } = renderHook(() => useUnifiedSidebar());

      act(() => {
        result.current.setActiveTab('auto');
      });

      expect(localStorageMock.setItem).toHaveBeenCalledWith(STORAGE_KEYS.SIDEBAR_TAB, 'auto');
    });

    it('should call onTabChange callback', () => {
      const onTabChange = vi.fn();
      const { result } = renderHook(() => useUnifiedSidebar({ onTabChange }));

      act(() => {
        result.current.setActiveTab('auto');
      });

      expect(onTabChange).toHaveBeenCalledWith('auto');
    });
  });

  describe('open/close', () => {
    it('should toggle open state', () => {
      const { result } = renderHook(() => useUnifiedSidebar());

      expect(result.current.isOpen).toBe(true);

      act(() => {
        result.current.toggleOpen();
      });

      expect(result.current.isOpen).toBe(false);

      act(() => {
        result.current.toggleOpen();
      });

      expect(result.current.isOpen).toBe(true);
    });

    it('should set open state directly', () => {
      const { result } = renderHook(() => useUnifiedSidebar());

      act(() => {
        result.current.setIsOpen(false);
      });

      expect(result.current.isOpen).toBe(false);
    });

    it('should persist open state to localStorage', () => {
      const { result } = renderHook(() => useUnifiedSidebar());

      act(() => {
        result.current.setIsOpen(false);
      });

      expect(localStorageMock.setItem).toHaveBeenCalledWith(STORAGE_KEYS.SIDEBAR_OPEN, 'false');
    });
  });

  describe('activity indicators', () => {
    it('should set timeline activity', () => {
      const { result } = renderHook(() => useUnifiedSidebar());

      act(() => {
        result.current.setTimelineActivity(true);
      });

      expect(result.current.timelineActivity).toBe(true);
    });

    it('should auto-clear timeline activity after 2 seconds', () => {
      const { result } = renderHook(() => useUnifiedSidebar());

      act(() => {
        result.current.setTimelineActivity(true);
      });

      expect(result.current.timelineActivity).toBe(true);

      act(() => {
        vi.advanceTimersByTime(2000);
      });

      expect(result.current.timelineActivity).toBe(false);
    });

    it('should set auto activity', () => {
      const { result } = renderHook(() => useUnifiedSidebar());

      act(() => {
        result.current.setAutoActivity(true);
      });

      expect(result.current.autoActivity).toBe(true);
    });

    it('should auto-clear auto activity after 2 seconds', () => {
      const { result } = renderHook(() => useUnifiedSidebar());

      act(() => {
        result.current.setAutoActivity(true);
      });

      expect(result.current.autoActivity).toBe(true);

      act(() => {
        vi.advanceTimersByTime(2000);
      });

      expect(result.current.autoActivity).toBe(false);
    });

    it('should reset timer when activity is set again', () => {
      const { result } = renderHook(() => useUnifiedSidebar());

      act(() => {
        result.current.setAutoActivity(true);
      });

      act(() => {
        vi.advanceTimersByTime(1500);
      });

      // Should still be active
      expect(result.current.autoActivity).toBe(true);

      // Set again - should reset timer
      act(() => {
        result.current.setAutoActivity(true);
      });

      act(() => {
        vi.advanceTimersByTime(1500);
      });

      // Should still be active (timer was reset)
      expect(result.current.autoActivity).toBe(true);

      act(() => {
        vi.advanceTimersByTime(500);
      });

      // Now should be cleared (2s total from last set)
      expect(result.current.autoActivity).toBe(false);
    });
  });

  describe('resize handling', () => {
    it('should start resize on mouse down', () => {
      const { result } = renderHook(() => useUnifiedSidebar());

      const mockEvent = {
        preventDefault: vi.fn(),
        clientX: 100,
      } as unknown as React.MouseEvent;

      act(() => {
        result.current.handleResizeStart(mockEvent);
      });

      expect(result.current.isResizing).toBe(true);
      expect(mockEvent.preventDefault).toHaveBeenCalled();
    });
  });

  describe('dimension constraints', () => {
    it('should expose correct min/max widths', () => {
      const { result } = renderHook(() => useUnifiedSidebar());

      expect(result.current.minWidth).toBe(320);
      expect(result.current.maxWidth).toBe(640);
    });
  });
});
