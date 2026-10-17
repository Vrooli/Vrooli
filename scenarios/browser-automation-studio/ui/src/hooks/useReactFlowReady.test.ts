import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@/test-utils';
import type { ReactFlowInstance } from 'reactflow';
import { createReactFlowViewport } from '@/test-utils';
import { useReactFlowReady } from './useReactFlowReady';

// Mock the logger to avoid console noise
vi.mock('@utils/logger', () => ({
  logger: {
    debug: vi.fn(),
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
  },
}));

describe('useReactFlowReady', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('returns true immediately for empty workflows', () => {
    const { result } = renderHook(() =>
      useReactFlowReady({
        reactFlowInstance: null,
        viewMode: 'visual',
        nodeCount: 0,
      })
    );

    expect(result.current).toBe(true);
  });

  it('returns false when not in visual mode', () => {
    const mockInstance = {
      project: vi.fn(),
      getViewport: vi.fn().mockReturnValue(createReactFlowViewport({ x: 100, y: 100, zoom: 0.5 })),
    } as unknown as ReactFlowInstance;

    const { result } = renderHook(() =>
      useReactFlowReady({
        reactFlowInstance: mockInstance,
        viewMode: 'code',
        nodeCount: 5,
      })
    );

    expect(result.current).toBe(false);
  });

  it('returns false when instance is null and has nodes', () => {
    const { result } = renderHook(() =>
      useReactFlowReady({
        reactFlowInstance: null,
        viewMode: 'visual',
        nodeCount: 5,
      })
    );

    expect(result.current).toBe(false);
  });

  it('returns true when instance is ready with non-default viewport', async () => {
    const mockInstance = {
      project: vi.fn(),
      getViewport: vi.fn().mockReturnValue(createReactFlowViewport()),
    } as unknown as ReactFlowInstance;

    const { result } = renderHook(() =>
      useReactFlowReady({
        reactFlowInstance: mockInstance,
        viewMode: 'visual',
        nodeCount: 5,
      })
    );

    // Should become ready after the first check
    await act(async () => {
      vi.advanceTimersByTime(100);
    });

    expect(result.current).toBe(true);
  });

  it('waits for viewport to change from default for non-empty workflows', async () => {
    const mockInstance = {
      project: vi.fn(),
      getViewport: vi.fn().mockReturnValue(createReactFlowViewport({ x: 0, y: 0, zoom: 1 })),
    } as unknown as ReactFlowInstance;

    const { result } = renderHook(
      ({ instance }) =>
        useReactFlowReady({
          reactFlowInstance: instance,
          viewMode: 'visual',
          nodeCount: 5,
        }),
      { initialProps: { instance: mockInstance } }
    );

    // Should not be ready with default viewport
    await act(async () => {
      vi.advanceTimersByTime(100);
    });
    expect(result.current).toBe(false);

    // Simulate fitView completing
    mockInstance.getViewport = vi.fn().mockReturnValue(createReactFlowViewport({ x: 150, y: 100, zoom: 0.7 }));

    await act(async () => {
      vi.advanceTimersByTime(200);
    });

    expect(result.current).toBe(true);
  });

  it('forces ready state after timeout', async () => {
    const mockInstance = {
      project: vi.fn(),
      getViewport: vi.fn().mockReturnValue(createReactFlowViewport({ x: 0, y: 0, zoom: 1 })),
    } as unknown as ReactFlowInstance;

    const { result } = renderHook(() =>
      useReactFlowReady({
        reactFlowInstance: mockInstance,
        viewMode: 'visual',
        nodeCount: 5,
      })
    );

    // Not ready initially
    expect(result.current).toBe(false);

    // Wait for emergency fallback (4s)
    await act(async () => {
      vi.advanceTimersByTime(4000);
    });

    expect(result.current).toBe(true);
  });

  it('resets ready state when workflow changes', async () => {
    const mockInstance = {
      project: vi.fn(),
      getViewport: vi.fn().mockReturnValue(createReactFlowViewport()),
    } as unknown as ReactFlowInstance;

    const { result, rerender } = renderHook(
      ({ workflowId }) =>
        useReactFlowReady({
          reactFlowInstance: mockInstance,
          viewMode: 'visual',
          nodeCount: 5,
          workflowId,
        }),
      { initialProps: { workflowId: 'workflow-1' } }
    );

    await act(async () => {
      vi.advanceTimersByTime(100);
    });
    expect(result.current).toBe(true);

    // Change workflow ID
    rerender({ workflowId: 'workflow-2' });
    expect(result.current).toBe(false);

    // Should become ready again
    await act(async () => {
      vi.advanceTimersByTime(100);
    });
    expect(result.current).toBe(true);
  });

  it('cancels pending checks when a workflow is replaced and on unmount', () => {
    const oldViewport = vi.fn().mockReturnValue(createReactFlowViewport({ x: 0, y: 0, zoom: 1 }));
    const instance = { project: vi.fn(), getViewport: oldViewport } as unknown as ReactFlowInstance;
    const { result, rerender, unmount } = renderHook(
      ({ workflowId, currentInstance }) => useReactFlowReady({ reactFlowInstance: currentInstance, viewMode: 'visual', nodeCount: 2, workflowId }),
      { initialProps: { workflowId: 'old', currentInstance: instance as ReactFlowInstance | null } },
    );
    expect(vi.getTimerCount()).toBeGreaterThan(0);
    rerender({ workflowId: 'new', currentInstance: null });
    oldViewport.mockReturnValue(createReactFlowViewport({ x: 200, y: 200, zoom: 0.5 }));
    act(() => vi.advanceTimersByTime(600));
    expect(result.current).toBe(false);
    unmount();
    expect(vi.getTimerCount()).toBe(0);
  });

  it('cleans pending checks when leaving visual mode and rechecks on return', () => {
    const instance = { project: vi.fn(), getViewport: vi.fn().mockReturnValue(createReactFlowViewport({ x: 0, y: 0, zoom: 1 })) } as unknown as ReactFlowInstance;
    const { result, rerender } = renderHook(
      ({ viewMode }) => useReactFlowReady({ reactFlowInstance: instance, viewMode, nodeCount: 2 }),
      { initialProps: { viewMode: 'visual' as 'visual' | 'code' } },
    );
    rerender({ viewMode: 'code' });
    expect(result.current).toBe(false);
    expect(vi.getTimerCount()).toBe(0);
    instance.getViewport = vi.fn().mockReturnValue(createReactFlowViewport({ x: 100, y: 50, zoom: 0.5 }));
    rerender({ viewMode: 'visual' });
    expect(result.current).toBe(true);
    expect(vi.getTimerCount()).toBe(0);
  });


  it('remains immediately ready when switching between empty workflows', () => {
    const { result, rerender } = renderHook(
      ({ workflowId }) => useReactFlowReady({ reactFlowInstance: null, viewMode: 'visual', nodeCount: 0, workflowId }),
      { initialProps: { workflowId: 'first-empty' } },
    );
    expect(result.current).toBe(true);
    rerender({ workflowId: 'second-empty' });
    expect(result.current).toBe(true);
    expect(vi.getTimerCount()).toBe(0);
  });

});
