import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import type { ReactFlowInstance } from 'reactflow';
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
      getViewport: vi.fn().mockReturnValue({ x: 100, y: 100, zoom: 0.5 }),
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
      getViewport: vi.fn().mockReturnValue({ x: 100, y: 50, zoom: 0.8 }),
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
      getViewport: vi.fn().mockReturnValue({ x: 0, y: 0, zoom: 1 }), // Default viewport
    } as unknown as ReactFlowInstance;

    const { result, rerender } = renderHook(
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
    mockInstance.getViewport = vi.fn().mockReturnValue({ x: 150, y: 100, zoom: 0.7 });

    await act(async () => {
      vi.advanceTimersByTime(200);
    });

    expect(result.current).toBe(true);
  });

  it('forces ready state after timeout', async () => {
    const mockInstance = {
      project: vi.fn(),
      getViewport: vi.fn().mockReturnValue({ x: 0, y: 0, zoom: 1 }), // Stays default
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
      getViewport: vi.fn().mockReturnValue({ x: 100, y: 50, zoom: 0.8 }),
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
});
