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
import { renderHook, act } from '@testing-library/react';
import { getConfig } from '@/config';
import { installFetchMock, type FetchMock } from '@/test-utils';
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

describe('ViewportSyncManager', () => {
  let fetchMock: FetchMock;

  beforeEach(() => {
    vi.useFakeTimers();
    fetchMock = installFetchMock();
    fetchMock.mockImplementation(async (_input, init) => new Response(JSON.stringify(JSON.parse(init!.body as string))));
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
  });

  describe('useViewportSyncManager hook', () => {
    it('should initialize with null viewport', () => {
      const { result } = renderHook(() =>
        useViewportSyncManager({ pageId: 'test-page', sessionId: 'test-session' })
      );

      expect(result.current.state.viewport).toBeNull();
      expect(result.current.state.isResizing).toBe(false);
      expect(result.current.state.isSyncing).toBe(false);
    });

    it('should clamp viewport to min/max dimensions', () => {
      const { result } = renderHook(() =>
        useViewportSyncManager({ pageId: 'test-page',
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
        useViewportSyncManager({ pageId: 'test-page', sessionId: 'test-session' })
      );

      act(() => {
        result.current.updateFromBounds({ width: 800, height: 600 });
      });

      expect(result.current.state.viewport).toEqual({ width: 800, height: 600 });
    });

    it('should detect rapid resize', async () => {
      const { result } = renderHook(() =>
        useViewportSyncManager({ pageId: 'test-page',
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
        useViewportSyncManager({ pageId: 'test-page',
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
      expect(fetchMock).not.toHaveBeenCalled();

      // Wait for debounce
      await act(async () => {
        vi.advanceTimersByTime(200);
        await Promise.resolve();
      });

      // Should only sync once with final viewport
      expect(fetchMock).toHaveBeenCalledTimes(1);
      expect(fetchMock).toHaveBeenCalledWith(
        'http://test-api/recordings/live/test-session/viewport',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({ width: 900, height: 600, page_id: 'test-page' }),
        })
      );
    });

    it('should not sync when sessionId is null', async () => {
      const { result } = renderHook(() =>
        useViewportSyncManager({ pageId: 'test-page',
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

      expect(fetchMock).not.toHaveBeenCalled();
    });

    it('should skip sync when viewport unchanged', async () => {
      const { result } = renderHook(() =>
        useViewportSyncManager({ pageId: 'test-page',
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
      expect(fetchMock).toHaveBeenCalledTimes(1);

      // Same viewport - should not sync again
      fetchMock.mockClear();
      act(() => {
        result.current.updateFromBounds({ width: 800, height: 600 });
      });
      await act(async () => {
        vi.advanceTimersByTime(150);
        await Promise.resolve();
      });
      expect(fetchMock).not.toHaveBeenCalled();
    });

    it('should reset state on session change', () => {
      const { result, rerender } = renderHook(
        ({ sessionId }) => useViewportSyncManager({ pageId: 'test-page', sessionId }),
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
      fetchMock.mockRejectedValueOnce(new Error('Network error'));

      const { result } = renderHook(() =>
        useViewportSyncManager({ pageId: 'test-page',
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
        useViewportSyncManager({ pageId: 'test-page',
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

      expect(fetchMock).toHaveBeenCalledTimes(1);
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


describe('viewport request lifetime [REQ:BAS-RH-J03]', () => {
  const deferred = <T,>() => {let resolve!: (value: T) => void;const promise = new Promise<T>(done => {resolve = done;});return {promise, resolve};};
  beforeEach(() => vi.mocked(getConfig).mockResolvedValue({API_URL: 'http://test-api'} as Awaited<ReturnType<typeof getConfig>>));
  afterEach(() => vi.unstubAllGlobals());
  it('submits the selected canonical page with desired dimensions', async () => {
    const fetch = vi.fn<typeof globalThis.fetch>(async () => new Response(JSON.stringify({width: 800, height: 600})));
    vi.stubGlobal('fetch',fetch);
    const {result} = renderHook(() => useViewportSyncManager({sessionId: 'owned', pageId: 'red'}));
    act(() => result.current.updateFromBounds({width: 800,height: 600}));
    await act(async () => result.current.forceSync());
    expect(JSON.parse(fetch.mock.calls[0][1]!.body as string)).toEqual({width: 800,height: 600,page_id: 'red'});
  });
  it('does not send an old request after configuration resolves in a new session', async () => {
    const config = deferred<Awaited<ReturnType<typeof getConfig>>>();
    vi.mocked(getConfig).mockReturnValueOnce(config.promise);
    const fetch = vi.fn<typeof globalThis.fetch>();vi.stubGlobal('fetch',fetch);
    const {result,rerender} = renderHook(({sessionId}) => useViewportSyncManager({sessionId, pageId: 'red'}),{initialProps:{sessionId:'old'}});
    act(() => result.current.updateFromBounds({width:800,height:600}));
    let pending!:Promise<void>;act(() => {pending=result.current.forceSync();});
    rerender({sessionId:'new'});
    await act(async()=>{config.resolve({API_URL:'http://test-api'} as Awaited<ReturnType<typeof getConfig>>);await pending;});
    expect(fetch).not.toHaveBeenCalled();
  });
  it('does not mark a replacement session synchronized after an old response', async () => {
    const response = deferred<Response>();vi.stubGlobal('fetch',vi.fn(()=>response.promise));
    const {result,rerender} = renderHook(({sessionId}) => useViewportSyncManager({sessionId, pageId: 'red'}),{initialProps:{sessionId:'old'}});
    act(() => result.current.updateFromBounds({width:800,height:600}));
    let pending!:Promise<void>;act(() => {pending=result.current.forceSync();});
    await act(async()=>{await Promise.resolve();});rerender({sessionId:'new'});
    await act(async()=>{response.resolve(new Response(JSON.stringify({width:800,height:600})));await pending;});
    expect(result.current.state.lastSyncTime).toBeNull();
  });
  it('rejects a zero-dimension success receipt', async () => {
    vi.stubGlobal('fetch',vi.fn(async()=>new Response(JSON.stringify({width:0,height:0}))));
    const {result} = renderHook(() => useViewportSyncManager({sessionId:'owned',pageId:'red'}));
    act(() => result.current.updateFromBounds({width:800,height:600}));
    await act(async()=>result.current.forceSync());
    expect(result.current.state.lastSyncTime).toBeNull();expect(result.current.state.syncError).toBeTruthy();
  });
  it('synchronizes retained bounds when the selected page changes without a resize', async () => {
    const fetch=vi.fn<typeof globalThis.fetch>(async()=>new Response(JSON.stringify({width:800,height:600})));
    vi.stubGlobal('fetch',fetch);
    const {result,rerender}=renderHook(({pageId})=>useViewportSyncManager({sessionId:'owned',pageId}),{initialProps:{pageId:'red'}});
    act(()=>result.current.updateFromBounds({width:800,height:600}));
    await act(async()=>result.current.forceSync());
    await act(async()=>{rerender({pageId:'blue'});});
    expect(fetch.mock.calls.map(([,init])=>JSON.parse(init!.body as string).page_id)).toEqual(['red','blue']);
    expect(result.current.state.viewport).toEqual({width:800,height:600});
    expect(result.current.state.lastSyncTime).not.toBeNull();
  });
  it('retains bounds without issuing a resize until a page is selected', async () => {
    const fetch=vi.fn<typeof globalThis.fetch>(async()=>new Response(JSON.stringify({width:800,height:600})));
    vi.stubGlobal('fetch',fetch);
    const {result,rerender}=renderHook(({pageId})=>useViewportSyncManager({sessionId:'owned',pageId}),{initialProps:{pageId:null as string|null}});
    act(()=>result.current.updateFromBounds({width:800,height:600}));
    await act(async()=>result.current.forceSync());expect(fetch).not.toHaveBeenCalled();
    await act(async()=>{rerender({pageId:'red'});});
    expect(JSON.parse(fetch.mock.calls[0][1]!.body as string).page_id).toBe('red');
  });

  it('reapplies prior dimensions after an in-flight resize may already have changed the browser', async () => {
    const delayed=deferred<Response>();let browserWidth=0;
    const fetch=vi.fn<typeof globalThis.fetch>(async(_url,init)=>{
      const requested=JSON.parse(init!.body as string);browserWidth=requested.width;
      return requested.width===900?delayed.promise:new Response(JSON.stringify(requested));
    });vi.stubGlobal('fetch',fetch);
    const {result}=renderHook(()=>useViewportSyncManager({sessionId:'owned',pageId:'red'}));
    act(()=>result.current.updateFromBounds({width:800,height:600}));await act(async()=>result.current.forceSync());
    act(()=>result.current.updateFromBounds({width:900,height:700}));
    let pending!:Promise<void>;act(()=>{pending=result.current.forceSync();});await act(async()=>{await Promise.resolve();});
    expect(browserWidth).toBe(900);
    act(()=>result.current.updateFromBounds({width:800,height:600}));await act(async()=>result.current.forceSync());
    await act(async()=>{delayed.resolve(new Response(JSON.stringify({width:900,height:700})));await pending;});
    expect(browserWidth).toBe(800);
    expect(result.current.state.viewport).toEqual({width:800,height:600});
    expect(result.current.state.syncError).toBeNull();
  });

});
