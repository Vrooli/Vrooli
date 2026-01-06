import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import React from 'react';
import { useUnifiedStats } from './useUnifiedStats';

// Mock the WebSocket context
const mockWebSocketContext = {
  isConnected: true,
  lastMessage: null,
  send: vi.fn(),
  subscribeToBinaryFrames: vi.fn(() => () => {}),
};

vi.mock('@/contexts/WebSocketContext', () => ({
  useWebSocket: () => mockWebSocketContext,
}));

describe('useUnifiedStats', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    mockWebSocketContext.lastMessage = null;
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.clearAllMocks();
  });

  describe('initial state', () => {
    it('starts with empty unified stats', () => {
      const { result } = renderHook(() =>
        useUnifiedStats({
          sessionId: 'test-session',
          serverStatsEnabled: false,
        })
      );

      expect(result.current.stats.client.totalFrames).toBe(0);
      expect(result.current.stats.server).toBeNull();
      expect(result.current.stats.serverStatsEnabled).toBe(false);
      expect(result.current.stats.isReceivingServerStats).toBe(false);
      expect(result.current.hasBottleneck).toBe(false);
      expect(result.current.bottleneckSeverity).toBe('none');
    });
  });

  describe('client stats integration', () => {
    it('updates client stats when recording frames', () => {
      const { result } = renderHook(() =>
        useUnifiedStats({
          sessionId: 'test-session',
          serverStatsEnabled: false,
        })
      );

      act(() => {
        result.current.recordFrame(1000);
        result.current.recordFrame(2000);
        vi.advanceTimersByTime(300);
      });

      expect(result.current.stats.client.totalFrames).toBe(2);
      expect(result.current.stats.client.totalBytes).toBe(3000);
    });
  });

  describe('reset', () => {
    it('resets all statistics', () => {
      const { result } = renderHook(() =>
        useUnifiedStats({
          sessionId: 'test-session',
          serverStatsEnabled: false,
        })
      );

      // Record some frames
      act(() => {
        result.current.recordFrame(1000);
        vi.advanceTimersByTime(300);
      });

      expect(result.current.stats.client.totalFrames).toBe(1);

      // Reset
      act(() => {
        result.current.reset();
      });

      expect(result.current.stats.client.totalFrames).toBe(0);
    });
  });

  describe('bottleneck detection', () => {
    it('returns "none" severity when no server stats', () => {
      const { result } = renderHook(() =>
        useUnifiedStats({
          sessionId: 'test-session',
          serverStatsEnabled: true,
        })
      );

      expect(result.current.bottleneckSeverity).toBe('none');
      expect(result.current.hasBottleneck).toBe(false);
    });
  });

  describe('serverStatsEnabled flag', () => {
    it('reflects serverStatsEnabled in stats', () => {
      const { result, rerender } = renderHook(
        ({ enabled }) =>
          useUnifiedStats({
            sessionId: 'test-session',
            serverStatsEnabled: enabled,
          }),
        { initialProps: { enabled: false } }
      );

      expect(result.current.stats.serverStatsEnabled).toBe(false);

      rerender({ enabled: true });

      expect(result.current.stats.serverStatsEnabled).toBe(true);
    });
  });
});
