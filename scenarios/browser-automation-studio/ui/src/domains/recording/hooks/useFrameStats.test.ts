import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useFrameStats, formatBytes, formatBandwidth } from './useFrameStats';

describe('useFrameStats', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  describe('initial state', () => {
    it('starts with zero stats', () => {
      const { result } = renderHook(() => useFrameStats());

      expect(result.current.stats).toEqual({
        currentFps: 0,
        avgFps: 0,
        lastFrameSize: 0,
        avgFrameSize: 0,
        totalFrames: 0,
        totalBytes: 0,
        bytesPerSecond: 0,
      });
    });
  });

  describe('recordFrame', () => {
    it('updates totals immediately after recording frames', () => {
      const { result } = renderHook(() => useFrameStats());

      act(() => {
        result.current.recordFrame(1000);
        result.current.recordFrame(2000);
        result.current.recordFrame(3000);
      });

      // Advance timer to trigger state update (updates every 250ms)
      act(() => {
        vi.advanceTimersByTime(300);
      });

      expect(result.current.stats.totalFrames).toBe(3);
      expect(result.current.stats.totalBytes).toBe(6000);
      expect(result.current.stats.lastFrameSize).toBe(3000);
    });

    it('calculates average frame size correctly', () => {
      const { result } = renderHook(() => useFrameStats());

      act(() => {
        result.current.recordFrame(1000);
        result.current.recordFrame(2000);
        result.current.recordFrame(3000);
      });

      // Advance timer to trigger state update
      act(() => {
        vi.advanceTimersByTime(300);
      });

      expect(result.current.stats.avgFrameSize).toBe(2000);
    });

    it('tracks current FPS over 1 second window', () => {
      const { result } = renderHook(() => useFrameStats());

      // Record 10 frames
      act(() => {
        for (let i = 0; i < 10; i++) {
          result.current.recordFrame(1000);
          vi.advanceTimersByTime(50); // 50ms apart = 20 FPS pace
        }
      });

      // Advance to trigger state update
      act(() => {
        vi.advanceTimersByTime(300);
      });

      // All 10 frames should be within the 1-second window
      expect(result.current.stats.currentFps).toBe(10);
    });

    it('removes old frames from history after 5 seconds', () => {
      const { result } = renderHook(() => useFrameStats());

      // Record 5 frames
      act(() => {
        for (let i = 0; i < 5; i++) {
          result.current.recordFrame(1000);
        }
      });

      // Wait 6 seconds (past the 5-second history window)
      act(() => {
        vi.advanceTimersByTime(6000);
      });

      // Record 1 more frame
      act(() => {
        result.current.recordFrame(2000);
      });

      // Trigger state update
      act(() => {
        vi.advanceTimersByTime(300);
      });

      // Only the last frame should be in the average (old ones pruned)
      // Total frames is cumulative, but avg should reflect recent history
      expect(result.current.stats.totalFrames).toBe(6);
      expect(result.current.stats.avgFrameSize).toBe(2000); // Only the new frame in history
    });
  });

  describe('reset', () => {
    it('clears all statistics', () => {
      const { result } = renderHook(() => useFrameStats());

      // Record some frames
      act(() => {
        result.current.recordFrame(1000);
        result.current.recordFrame(2000);
        vi.advanceTimersByTime(300);
      });

      // Reset
      act(() => {
        result.current.reset();
      });

      expect(result.current.stats).toEqual({
        currentFps: 0,
        avgFps: 0,
        lastFrameSize: 0,
        avgFrameSize: 0,
        totalFrames: 0,
        totalBytes: 0,
        bytesPerSecond: 0,
      });
    });
  });

  describe('batched updates', () => {
    it('batches state updates to 250ms intervals', () => {
      const { result } = renderHook(() => useFrameStats());

      // Record frames without advancing time
      act(() => {
        result.current.recordFrame(1000);
      });

      // Stats should still be zero (no state update yet)
      expect(result.current.stats.totalFrames).toBe(0);

      // Advance past update interval
      act(() => {
        vi.advanceTimersByTime(300);
      });

      // Now stats should be updated
      expect(result.current.stats.totalFrames).toBe(1);
    });
  });
});

describe('formatBytes', () => {
  it('formats bytes correctly', () => {
    expect(formatBytes(0)).toBe('0 B');
    expect(formatBytes(500)).toBe('500.0 B');
    expect(formatBytes(1024)).toBe('1.0 KB');
    expect(formatBytes(1536)).toBe('1.5 KB');
    expect(formatBytes(1048576)).toBe('1.0 MB');
    expect(formatBytes(1572864)).toBe('1.5 MB');
  });

  it('respects decimals parameter', () => {
    expect(formatBytes(1536, 0)).toBe('2 KB');
    expect(formatBytes(1536, 2)).toBe('1.50 KB');
  });
});

describe('formatBandwidth', () => {
  it('formats bandwidth correctly', () => {
    expect(formatBandwidth(0)).toBe('0 B/s');
    expect(formatBandwidth(1024)).toBe('1.0 KB/s');
    expect(formatBandwidth(1048576)).toBe('1.0 MB/s');
  });
});
