import { renderHook } from '@/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { useUnifiedTimeline } from './useUnifiedTimeline';
import type { TimelineItem } from '../types/timeline-unified';

vi.mock('@/contexts/WebSocketContext', () => ({
  useWebSocket: () => ({ isConnected: false, send: vi.fn() }),
  useWebSocketMessage: vi.fn(),
}));

describe('useUnifiedTimeline recording projection', () => {
  it('keeps already projected journal items distinct and in order', () => {
    const entries: TimelineItem[] = [
      {
        id: 'first', sequenceNum: 1, timestamp: new Date('2026-09-30T12:00:01Z'), actionType: 'input',
        payload: { text: 'one' }, mode: 'recording', pageId: 'page', entryType: 'action',
      },
      {
        id: 'second', sequenceNum: 2, timestamp: new Date('2026-09-30T12:00:02Z'), actionType: 'input',
        payload: { text: 'two' }, mode: 'recording', pageId: 'page', entryType: 'action',
      },
    ];
    const { result } = renderHook(() =>
      useUnifiedTimeline({ mode: 'recording', initialTimelineItems: entries }),
    );

    expect(result.current.items.map((item) => item.id)).toEqual(['first', 'second']);
  });

  it('does not synthesize timeline items from another action feed', () => {
    const { result } = renderHook(() => useUnifiedTimeline({ mode: 'recording' }));

    expect(result.current.items).toEqual([]);
  });
});
