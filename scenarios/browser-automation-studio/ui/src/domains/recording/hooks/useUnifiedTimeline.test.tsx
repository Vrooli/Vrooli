import { fromJson } from '@bufbuild/protobuf';
import { TimelineEntrySchema } from '@vrooli/proto-types/browser-automation-studio/v1/timeline/entry_pb';
import { waitFor } from '@testing-library/react';
import { renderHook } from '@/test-utils';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { recordingApi } from '../api';
import { useSessionStore } from '../stores/sessionStore';
import { useWorkspaceTimeline } from './useWorkspaceTimeline';

const websocket = vi.hoisted(() => ({
  send: vi.fn(),
  connected: false,
  handlers: [] as Array<(message: unknown) => void>,
}));

vi.mock('../api', () => ({ recordingApi: { getTimeline: vi.fn() } }));
vi.mock('@/contexts/WebSocketContext', () => ({
  useWebSocket: () => ({ isConnected: websocket.connected, send: websocket.send }),
  useWebSocketMessage: (handler: (message: unknown) => void) => { websocket.handlers.push(handler); },
}));

describe('workspace timeline controller', () => {
  beforeEach(() => {
    websocket.handlers = [];
    websocket.send.mockClear();
    websocket.connected = false;
    useSessionStore.setState({ sessionId: 'session', isValidated: true });
    vi.mocked(recordingApi.getTimeline).mockResolvedValue({
      success: true,
      data: {
        entries: [{
          type: 'action', pageId: 'page',
          entry: fromJson(TimelineEntrySchema, {
            id: 'journal-entry', sequence_num: 1, timestamp: '2026-10-01T00:00:00Z',
            action: { type: 'ACTION_TYPE_CLICK', click: { selector: '#save' } },
          }, { jsonOptions: { useProtoNames: true } }),
        }],
        totalEntries: 1,
        hasMore: false,
      },
    } as never);
  });

  afterEach(() => {
    useSessionStore.setState({ sessionId: null, isValidated: false });
  });

  it('keeps generated recording journal entries while switching between modes', async () => {
    const { result, rerender } = renderHook(
      ({ mode }: { mode: 'recording' | 'execution' }) => useWorkspaceTimeline({
        mode,
        sessionId: 'session',
        pages: [],
        workflowNodes: mode === 'execution' ? [{ id: 'save', action: { type: 'ACTION_TYPE_CLICK' } }] : undefined,
        workflowEdges: mode === 'execution' ? [] : undefined,
      }),
      { initialProps: { mode: 'recording' as const } },
    );

    await waitFor(() => expect(result.current.entries).toHaveLength(1));
    expect(result.current.entries[0]).toMatchObject({ type: 'action', entry: { id: 'journal-entry' } });
    expect(result.current.executionItems).toEqual([]);

    rerender({ mode: 'execution' });
    await waitFor(() => expect(result.current.executionItems.map((item) => item.nodeId)).toEqual(['save']));
    expect(result.current.entries[0]).toMatchObject({ entry: { id: 'journal-entry' } });

    rerender({ mode: 'recording' });
    await waitFor(() => expect(result.current.executionItems).toEqual([]));
    expect(result.current.entries[0]).toMatchObject({ entry: { id: 'journal-entry' } });
  });

  it('reports recording as idle or live from the active capture signal', () => {
    const { result, rerender } = renderHook(
      ({ isRecording }: { isRecording: boolean }) => useWorkspaceTimeline({
        mode: 'recording', isRecording, sessionId: null, pages: [],
      }),
      { initialProps: { isRecording: false } },
    );

    expect(result.current.isLive).toBe(false);
    rerender({ isRecording: true });
    expect(result.current.isLive).toBe(true);
  });

  it('prepopulates workflow order and applies live execution updates to matching items', async () => {
    const { result } = renderHook(() => useWorkspaceTimeline({
      mode: 'execution',
      sessionId: null,
      pages: [],
      workflowNodes: [
        { id: 'navigate', action: { type: 'ACTION_TYPE_NAVIGATE', navigate: { url: 'https://example.test' } } },
        { id: 'click', action: { type: 'ACTION_TYPE_CLICK' } },
      ],
      workflowEdges: [{ source: 'navigate', target: 'click' }],
    }));

    await waitFor(() => expect(result.current.executionItems).toHaveLength(2));
    expect(result.current.executionItems.map((item) => item.nodeId)).toEqual(['navigate', 'click']);
    expect(result.current.executionItems.every((item) => 'executionStatus' in item && item.executionStatus === 'pending')).toBe(true);

    websocket.handlers.at(-1)?.({
      type: 'step',
      entry: { step_index: 0, context: { node_id: 'navigate', success: true, duration_ms: 25 } },
    });
    await waitFor(() => expect(result.current.executionItems[0]).toMatchObject({ nodeId: 'navigate', executionStatus: 'completed' }));
  });

  it('subscribes and cleans up each active mode subscription', async () => {
    websocket.connected = true;
    const { rerender, unmount } = renderHook(
      ({ mode }: { mode: 'recording' | 'execution' }) => useWorkspaceTimeline({
        mode,
        sessionId: 'session',
        pages: [],
        executionId: 'execution',
      }),
      { initialProps: { mode: 'recording' as const } },
    );

    await waitFor(() => expect(websocket.send).toHaveBeenCalledWith({
      type: 'subscribe_recording', session_id: 'session', frames: false,
    }));
    rerender({ mode: 'execution' });
    await waitFor(() => expect(websocket.send).toHaveBeenCalledWith({
      type: 'subscribe_execution', execution_id: 'execution',
    }));
    rerender({ mode: 'recording' });
    await waitFor(() => expect(websocket.send).toHaveBeenCalledWith({
      type: 'unsubscribe_execution', execution_id: 'execution',
    }));
    unmount();
    expect(websocket.send).toHaveBeenCalledWith({ type: 'unsubscribe_recording', session_id: 'session' });
  });
});
