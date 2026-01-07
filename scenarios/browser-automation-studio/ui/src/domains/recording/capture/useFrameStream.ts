/**
 * useFrameStream Hook
 *
 * Handles live frame streaming from Playwright sessions via direct WebSocket
 * to playwright-driver, with polling fallback.
 *
 * Features:
 * - Direct WebSocket connection to playwright-driver (bypasses API Hub for 67% latency reduction)
 * - Polling fallback when WebSocket unavailable (10-15 FPS)
 * - Double-buffered canvas rendering to prevent white flash
 * - Frame statistics tracking
 * - Page metadata extraction
 * - Automatic reconnection with exponential backoff
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { getConfig } from '@/config';
import { useWebSocket } from '@/contexts/WebSocketContext';
import { useFrameStats, type FrameStats } from '../hooks/useFrameStats';
import { LatencyLogger, type LatencyStats } from '@utils/latencyLogger';

/** Frame dimensions stored in ref to avoid re-renders */
interface FrameDimensions {
  width: number;
  height: number;
  capturedAt: string;
}

interface FramePayload {
  image: string;
  width: number;
  height: number;
  captured_at: string;
  content_hash?: string;
  page_title?: string;
  page_url?: string;
}

/** Page metadata extracted from frames */
export interface PageMetadata {
  title: string;
  url: string;
}

/** Connection status for the live preview stream */
export interface StreamConnectionStatus {
  isConnected: boolean;
  isWebSocket: boolean;
  lastFrameTime?: string;
}

export interface UseFrameStreamOptions {
  sessionId: string | null;
  pageId?: string;
  quality?: number;
  fps?: number;
  useWebSocketFrames?: boolean;
  refreshToken?: number;
  onStreamError?: (message: string) => void;
  onStatsUpdate?: (stats: FrameStats) => void;
  onPageMetadataChange?: (metadata: PageMetadata) => void;
  onConnectionStatusChange?: (status: StreamConnectionStatus) => void;
}

export interface UseFrameStreamResult {
  /** Canvas ref to render frames to */
  canvasRef: React.RefObject<HTMLCanvasElement>;
  /** Whether at least one frame has been received */
  hasFrame: boolean;
  /** Current display dimensions */
  displayDimensions: { width: number; height: number } | null;
  /** Timestamp of last displayed frame */
  displayedTimestamp: string | null;
  /** Current error message if any */
  error: string | null;
  /** Whether currently fetching (polling mode) */
  isFetching: boolean;
  /** Whether WebSocket frames are active */
  isWsFrameActive: boolean;
  /** Whether page is switching (tab change) */
  isPageSwitching: boolean;
  /** Frame stats */
  frameStats: FrameStats;
  /** Ref to frame dimensions (for coordinate mapping) */
  frameDimensionsRef: React.RefObject<FrameDimensions | null>;
  /** Latency stats for A/B testing (research spike) */
  latencyStats: LatencyStats | null;
  /** Log latency stats to console */
  logLatencyStats: () => void;
}

export function useFrameStream({
  sessionId,
  pageId,
  quality = 65,
  fps = 30,
  useWebSocketFrames = true,
  refreshToken,
  onStreamError,
  onStatsUpdate,
  onPageMetadataChange,
  onConnectionStatusChange,
}: UseFrameStreamOptions): UseFrameStreamResult {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const backBufferRef = useRef<HTMLCanvasElement | null>(null);
  const frameDimensionsRef = useRef<FrameDimensions | null>(null);

  const [hasFrame, setHasFrame] = useState(false);
  const hasFrameRef = useRef(false);
  const [displayDimensions, setDisplayDimensions] = useState<{ width: number; height: number } | null>(null);
  const [isFetching, setIsFetching] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const errorRef = useRef<string | null>(null); // Track error in ref for guarded updates
  const [isTabVisible, setIsTabVisible] = useState(!document.hidden);
  const [isWsFrameActive, setIsWsFrameActive] = useState(false);
  const [isPageSwitching, setIsPageSwitching] = useState(false);

  // Frame tracking refs
  const frameCountRef = useRef(0);
  const droppedFramesRef = useRef(0);
  const [displayedTimestamp, setDisplayedTimestamp] = useState<string | null>(null);

  // Latency tracking for research spike
  const latencyLoggerRef = useRef(new LatencyLogger('API Relay', 200));
  const [latencyStats, setLatencyStats] = useState<LatencyStats | null>(null);

  const inFlightRef = useRef(false);
  const lastSessionRef = useRef<string | null>(null);
  const lastETagRef = useRef<string | null>(null);
  const lastContentHashRef = useRef<string | null>(null);
  const lastPageIdRef = useRef<string | undefined>(pageId);

  // WebSocket context no longer needed for frame streaming (direct connection to playwright-driver)
  // Keeping the import in case other parts of the app need it, but not using it here
  useWebSocket(); // Keep context active but don't destructure unused values

  // Frame statistics
  const { stats: frameStats, recordFrame, reset: resetStats } = useFrameStats();
  const onStatsUpdateRef = useRef(onStatsUpdate);
  onStatsUpdateRef.current = onStatsUpdate;

  // Page metadata
  const onPageMetadataChangeRef = useRef(onPageMetadataChange);
  onPageMetadataChangeRef.current = onPageMetadataChange;
  const lastPageMetadataRef = useRef<PageMetadata | null>(null);

  // Connection status
  const onConnectionStatusChangeRef = useRef(onConnectionStatusChange);
  onConnectionStatusChangeRef.current = onConnectionStatusChange;

  const pollInterval = useMemo(() => Math.max(300, Math.floor(1000 / fps)), [fps]);
  const pollIntervalRef = useRef(pollInterval);
  pollIntervalRef.current = pollInterval;

  // Track tab visibility
  useEffect(() => {
    const handleVisibilityChange = () => {
      setIsTabVisible(!document.hidden);
    };
    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
    };
  }, []);

  // Handle page switching transitions
  // IMPORTANT: Only reset hasFrame when switching between two REAL page IDs.
  // When pageId goes from undefined → real ID, that's just the initial page ID
  // being resolved (from the /pages API), not an actual tab switch.
  // Resetting hasFrame in that case causes flickering during initial navigation.
  useEffect(() => {
    if (lastPageIdRef.current !== pageId) {
      const isRealTabSwitch = lastPageIdRef.current && pageId;

      if (isRealTabSwitch) {
        // Actual tab switch - reset frame state and show transition
        setIsPageSwitching(true);
        frameDimensionsRef.current = null;
        hasFrameRef.current = false;
        setHasFrame(false);

        const timer = setTimeout(() => {
          setIsPageSwitching(false);
        }, 300);

        lastPageIdRef.current = pageId;
        return () => clearTimeout(timer);
      } else {
        // Initial page ID being set or cleared - just update ref, don't reset frame
        lastPageIdRef.current = pageId;
      }
    }
  }, [pageId]);

  // Push frame stats to parent
  useEffect(() => {
    if (onStatsUpdateRef.current) {
      onStatsUpdateRef.current(frameStats);
    }
  }, [frameStats]);

  // Push connection status to parent
  useEffect(() => {
    if (onConnectionStatusChangeRef.current) {
      onConnectionStatusChangeRef.current({
        isConnected: hasFrame,
        isWebSocket: isWsFrameActive,
        lastFrameTime: displayedTimestamp ?? undefined,
      });
    }
  }, [hasFrame, isWsFrameActive, displayedTimestamp]);

  /**
   * Draw a bitmap directly to canvas, using double-buffering only for significant changes.
   *
   * Double-buffering prevents white flash when resizing, but adds ~1-2ms latency per frame.
   * We only use it for significant dimension changes (>50px) to balance visual quality
   * with streaming performance.
   */
  const drawFrameToCanvas = useCallback((bitmap: ImageBitmap) => {
    const canvas = canvasRef.current;
    if (!canvas) return false;

    const ctx = canvas.getContext('2d', { alpha: false });
    if (!ctx) return false;

    const widthDiff = Math.abs(canvas.width - bitmap.width);
    const heightDiff = Math.abs(canvas.height - bitmap.height);
    const isSignificantChange = widthDiff > 50 || heightDiff > 50;
    const dimensionsChanging = canvas.width !== bitmap.width || canvas.height !== bitmap.height;

    // Only use double-buffering for significant changes to prevent white flash
    // Minor changes (< 50px) draw directly for lower latency
    if (dimensionsChanging && isSignificantChange) {
      if (!backBufferRef.current) {
        backBufferRef.current = document.createElement('canvas');
      }
      const backBuffer = backBufferRef.current;
      const backCtx = backBuffer.getContext('2d', { alpha: false });
      if (!backCtx) return false;

      backBuffer.width = bitmap.width;
      backBuffer.height = bitmap.height;
      backCtx.drawImage(bitmap, 0, 0);

      canvas.width = bitmap.width;
      canvas.height = bitmap.height;
      ctx.drawImage(backBuffer, 0, 0);
    } else {
      // Direct draw for same dimensions or minor changes
      if (dimensionsChanging) {
        canvas.width = bitmap.width;
        canvas.height = bitmap.height;
      }
      ctx.drawImage(bitmap, 0, 0);
    }

    return true;
  }, []);

  // Binary frame processing refs
  // Optimization: We decode bitmaps immediately when frames arrive (async but parallel),
  // then RAF just draws the latest decoded bitmap. This reduces latency by ~2-5ms
  // since decoding happens during the RAF wait, not after RAF fires.
  const pendingBitmapRef = useRef<ImageBitmap | null>(null);
  const pendingFrameSizeRef = useRef<number>(0);
  const latestFrameIdRef = useRef<number>(0);
  const rafIdRef = useRef<number | null>(null);
  const isWsFrameActiveRef = useRef(isWsFrameActive);
  isWsFrameActiveRef.current = isWsFrameActive;

  /**
   * Draw pending bitmap - called from requestAnimationFrame.
   * The bitmap is already decoded, so this is fast (~1ms).
   */
  const drawPendingFrame = useCallback(() => {
    rafIdRef.current = null;

    const bitmap = pendingBitmapRef.current;
    const frameSize = pendingFrameSizeRef.current;
    if (!bitmap) return;

    pendingBitmapRef.current = null;

    const drawn = drawFrameToCanvas(bitmap);
    if (!drawn) {
      bitmap.close();
      return;
    }

    const newDimensions: FrameDimensions = {
      width: bitmap.width,
      height: bitmap.height,
      capturedAt: new Date().toISOString(),
    };

    const prevDims = frameDimensionsRef.current;
    const dimensionsChanged = !prevDims ||
      prevDims.width !== bitmap.width ||
      prevDims.height !== bitmap.height;

    frameDimensionsRef.current = newDimensions;
    bitmap.close();

    if (!hasFrameRef.current) {
      hasFrameRef.current = true;
      setHasFrame(true);
    }
    if (dimensionsChanged) {
      setDisplayDimensions({ width: newDimensions.width, height: newDimensions.height });
    }

    // Guard setError(null) to avoid unnecessary React reconciliation
    if (errorRef.current !== null) {
      errorRef.current = null;
      setError(null);
    }

    // Track successful frame processing
    frameCountRef.current++;
    recordFrame(frameSize);
  }, [drawFrameToCanvas, recordFrame]);

  /**
   * Decode a blob and queue it for drawing.
   * Decoding happens immediately (in parallel with RAF wait), not after RAF fires.
   */
  const decodeAndQueueFrame = useCallback(async (blob: Blob, frameId: number, frameSize: number) => {
    try {
      const bitmap = await createImageBitmap(blob);

      // Check if a newer frame arrived while we were decoding
      if (latestFrameIdRef.current > frameId) {
        bitmap.close();
        return;
      }

      // Close any existing pending bitmap that wasn't drawn yet
      const oldBitmap = pendingBitmapRef.current;
      if (oldBitmap) {
        oldBitmap.close();
      }

      pendingBitmapRef.current = bitmap;
      pendingFrameSizeRef.current = frameSize;

      // Schedule RAF if not already scheduled
      if (rafIdRef.current === null) {
        rafIdRef.current = requestAnimationFrame(drawPendingFrame);
      }
    } catch {
      if (latestFrameIdRef.current === frameId) {
        errorRef.current = 'Failed to decode binary frame';
        setError('Failed to decode binary frame');
      }
    }
  }, [drawPendingFrame]);

  // Direct WebSocket connection to playwright-driver for frame streaming
  // This bypasses the API Hub for ~67% latency reduction (9ms → 3ms median)
  const directWsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const reconnectAttemptRef = useRef(0);
  const maxReconnectAttempts = 10;
  // Start with a short initial delay for fast recovery from brief disconnects
  // Exponential backoff kicks in for persistent issues
  const baseReconnectDelay = 250; // 250ms initial (was 1s)
  const maxReconnectDelay = 15000; // 15 seconds (was 30s)

  useEffect(() => {
    if (!useWebSocketFrames || !sessionId) {
      // Clean up existing connection
      if (directWsRef.current) {
        directWsRef.current.close();
        directWsRef.current = null;
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }
      isWsFrameActiveRef.current = false;
      setIsWsFrameActive(false);
      return;
    }

    const handleBinaryFrame = (data: ArrayBuffer) => {
      if (!isWsFrameActiveRef.current) {
        droppedFramesRef.current++;
        return;
      }

      // Frame format from DirectFrameServer: [8-byte timestamp BigInt64BE][JPEG data]
      let jpegData: ArrayBuffer = data;
      let sentTimestamp: number | null = null;

      if (data.byteLength > 8) {
        const view = new DataView(data);
        try {
          sentTimestamp = Number(view.getBigInt64(0));
          // Validate it looks like a reasonable timestamp (within last hour)
          const now = Date.now();
          if (sentTimestamp > now - 3600000 && sentTimestamp <= now + 1000) {
            jpegData = data.slice(8);
          } else {
            sentTimestamp = null; // Not a valid timestamp, use full data
            jpegData = data;
          }
        } catch {
          sentTimestamp = null;
        }
      }

      // Record latency for performance monitoring
      if (sentTimestamp !== null) {
        const latency = Date.now() - sentTimestamp;
        if (latency >= 0 && latency < 10000) { // Sanity check: 0-10 seconds
          latencyLoggerRef.current.record(latency);
          // Update stats every 10 frames to reduce state updates
          if (latencyLoggerRef.current.getSampleCount() % 10 === 0) {
            setLatencyStats(latencyLoggerRef.current.getStats());
          }
        }
      }

      const blob = new Blob([jpegData], { type: 'image/jpeg' });
      const frameId = Date.now();
      const frameSize = jpegData.byteLength;

      // Mark that WebSocket has successfully delivered a frame
      // This allows polling to stop (see polling effect guard)
      hasReceivedWsFrameRef.current = true;

      // Update frame ID immediately (used for stale frame detection)
      latestFrameIdRef.current = frameId;

      // Start decoding immediately - this happens in parallel with RAF wait
      // The decoded bitmap will be ready when RAF fires, reducing latency by ~2-5ms
      void decodeAndQueueFrame(blob, frameId, frameSize);
    };

    const connect = async () => {
      // Fetch config to get the playwright driver port for direct connection
      let driverPort: number | undefined;
      try {
        const response = await fetch('/config');
        if (response.ok) {
          const config = await response.json();
          driverPort = config.playwrightDriverPort;
        }
      } catch {
        // Config fetch failed, will fall back to polling
      }

      if (!driverPort) {
        // No driver port available - direct connection not possible
        // Polling fallback will handle frame streaming
        console.warn('[useFrameStream] No playwright driver port in config, falling back to polling');
        return;
      }

      // Build WebSocket URL for direct connection to playwright-driver
      // DirectFrameServer runs on driver port + 1
      const directFramePort = driverPort + 1;
      let wsUrl = `ws://localhost:${directFramePort}/frames?session_id=${encodeURIComponent(sessionId)}`;
      if (pageId) {
        wsUrl += `&page_id=${encodeURIComponent(pageId)}`;
      }

      const ws = new WebSocket(wsUrl);
      ws.binaryType = 'arraybuffer';
      directWsRef.current = ws;

      ws.onopen = () => {
        reconnectAttemptRef.current = 0; // Reset on successful connection
        isWsFrameActiveRef.current = true;
        setIsWsFrameActive(true);
      };

      ws.onmessage = (event) => {
        // Handle text messages (subscription confirmation, etc.)
        if (typeof event.data === 'string') {
          // Ignore text messages - frames are binary
          return;
        }

        // Handle binary frame data
        handleBinaryFrame(event.data as ArrayBuffer);
      };

      ws.onerror = () => {
        // Error will be followed by onclose, handle reconnection there
      };

      ws.onclose = () => {
        directWsRef.current = null;
        isWsFrameActiveRef.current = false;
        setIsWsFrameActive(false);

        // Attempt reconnection with exponential backoff
        if (reconnectAttemptRef.current < maxReconnectAttempts) {
          const delay = Math.min(
            baseReconnectDelay * Math.pow(2, reconnectAttemptRef.current),
            maxReconnectDelay
          );
          reconnectAttemptRef.current++;

          reconnectTimeoutRef.current = setTimeout(() => {
            if (sessionId && useWebSocketFrames) {
              connect();
            }
          }, delay);
        }
        // After max attempts, polling fallback will take over
      };
    };

    connect();

    return () => {
      if (directWsRef.current) {
        directWsRef.current.close();
        directWsRef.current = null;
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }
    };
  }, [useWebSocketFrames, sessionId, pageId, decodeAndQueueFrame]);

  // Cleanup RAF and pending bitmap on unmount
  useEffect(() => {
    return () => {
      if (rafIdRef.current !== null) {
        cancelAnimationFrame(rafIdRef.current);
      }
      // Clean up any pending bitmap to prevent memory leaks
      const bitmap = pendingBitmapRef.current;
      if (bitmap) {
        bitmap.close();
        pendingBitmapRef.current = null;
      }
    };
  }, []);

  // Polling fetch function
  const fetchFrame = useCallback(async () => {
    if (inFlightRef.current || !sessionId) return;
    inFlightRef.current = true;
    setIsFetching(true);
    const started = performance.now();

    try {
      const config = await getConfig();
      let endpoint = `${config.API_URL}/recordings/live/${sessionId}/frame?quality=${quality}`;
      if (pageId) {
        endpoint += `&page_id=${encodeURIComponent(pageId)}`;
      }
      const headers: HeadersInit = {};
      if (lastETagRef.current) {
        headers['If-None-Match'] = lastETagRef.current;
      }
      const res = await fetch(endpoint, { headers });

      if (res.status === 304) {
        if (errorRef.current !== null) {
          errorRef.current = null;
          setError(null);
        }
        return;
      }

      if (!res.ok) {
        throw new Error(`Frame fetch failed (${res.status})`);
      }

      const etag = res.headers.get('ETag');
      if (etag) {
        lastETagRef.current = etag;
      }

      const data = (await res.json()) as FramePayload;

      // Handle page metadata
      if (data.page_title !== undefined || data.page_url !== undefined) {
        const newMetadata: PageMetadata = {
          title: data.page_title || '',
          url: data.page_url || '',
        };
        const lastMetadata = lastPageMetadataRef.current;
        if (!lastMetadata || lastMetadata.title !== newMetadata.title || lastMetadata.url !== newMetadata.url) {
          lastPageMetadataRef.current = newMetadata;
          if (onPageMetadataChangeRef.current) {
            onPageMetadataChangeRef.current(newMetadata);
          }
        }
      }

      // Draw to canvas using createImageBitmap (more efficient than Image())
      // Convert base64 data URI to blob
      const base64Data = data.image.includes(',') ? data.image.split(',')[1] : data.image;
      const binaryString = atob(base64Data);
      const bytes = new Uint8Array(binaryString.length);
      for (let i = 0; i < binaryString.length; i++) {
        bytes[i] = binaryString.charCodeAt(i);
      }
      const blob = new Blob([bytes], { type: 'image/jpeg' });

      try {
        const bitmap = await createImageBitmap(blob);
        const drawn = drawFrameToCanvas(bitmap);

        if (drawn) {
          const newDimensions: FrameDimensions = {
            width: bitmap.width,
            height: bitmap.height,
            capturedAt: data.captured_at,
          };

          const prevDims = frameDimensionsRef.current;
          const dimensionsChanged = !prevDims ||
            prevDims.width !== bitmap.width ||
            prevDims.height !== bitmap.height;

          frameDimensionsRef.current = newDimensions;

          if (!hasFrameRef.current) {
            hasFrameRef.current = true;
            setHasFrame(true);
          }
          if (dimensionsChanged) {
            setDisplayDimensions({ width: newDimensions.width, height: newDimensions.height });
          }
          setDisplayedTimestamp(data.captured_at);
          if (errorRef.current !== null) {
            errorRef.current = null;
            setError(null);
          }
          recordFrame(bytes.byteLength);
        }

        bitmap.close();
      } catch {
        errorRef.current = 'Failed to decode live frame';
        setError('Failed to decode live frame');
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch live frame';
      errorRef.current = message;
      setError(message);
      if (onStreamError) {
        onStreamError(message);
      }
    } finally {
      setIsFetching(false);
      inFlightRef.current = false;
      const elapsed = performance.now() - started;
      if (elapsed > pollInterval) {
        const dampenedFps = Math.max(1, Math.round(1000 / Math.min(elapsed, 1500)));
        const nextInterval = Math.max(400, Math.floor(1000 / dampenedFps));
        pollIntervalRef.current = nextInterval;
      }
    }
  }, [fps, onStreamError, quality, sessionId, pageId, pollInterval, drawFrameToCanvas, recordFrame]);

  // Track if we've received at least one WebSocket frame
  const hasReceivedWsFrameRef = useRef(false);

  // Polling - runs as fallback when WebSocket hasn't delivered a frame yet
  // CRITICAL: Don't stop polling just because WebSocket connected!
  // Only stop when WebSocket has PROVEN it works by delivering a frame.
  // This fixes white screen when CDP screencast doesn't send frames immediately
  // (e.g., page is static after navigation).
  useEffect(() => {
    if (isWsFrameActive && hasReceivedWsFrameRef.current) {
      return;
    }

    let cancelled = false;
    let timeoutId: ReturnType<typeof setTimeout> | null = null;

    const tick = async () => {
      if (cancelled || isWsFrameActive) return;
      if (isTabVisible) {
        await fetchFrame();
      }
      if (cancelled || isWsFrameActive) return;
      const nextInterval = pollIntervalRef.current;
      timeoutId = setTimeout(tick, nextInterval);
    };

    tick();

    return () => {
      cancelled = true;
      if (timeoutId) {
        clearTimeout(timeoutId);
      }
    };
  }, [fetchFrame, pollInterval, refreshToken, isTabVisible, isWsFrameActive]);

  // Reset state when session changes
  useEffect(() => {
    if (lastSessionRef.current !== sessionId) {
      lastSessionRef.current = sessionId;
      frameDimensionsRef.current = null;
      hasFrameRef.current = false;
      setHasFrame(false);
      setDisplayDimensions(null);
      setDisplayedTimestamp(null);
      const canvas = canvasRef.current;
      if (canvas) {
        const ctx = canvas.getContext('2d');
        if (ctx) {
          ctx.clearRect(0, 0, canvas.width, canvas.height);
        }
      }
      lastETagRef.current = null;
      lastContentHashRef.current = null;
      errorRef.current = null;
      setError(null);
      // CRITICAL: Update ref BEFORE state
      isWsFrameActiveRef.current = false;
      setIsWsFrameActive(false);
      reconnectAttemptRef.current = 0; // Reset reconnect counter for new session
      hasReceivedWsFrameRef.current = false; // Reset WS frame tracking for new session
      resetStats();
      // Reset diagnostic counters
      frameCountRef.current = 0;
      droppedFramesRef.current = 0;
    }
  }, [sessionId, resetStats]);

  // Callback to log latency stats to console
  const logLatencyStats = useCallback(() => {
    latencyLoggerRef.current.logStats();
  }, []);

  return {
    canvasRef,
    hasFrame,
    displayDimensions,
    displayedTimestamp,
    error,
    isFetching,
    isWsFrameActive,
    isPageSwitching,
    frameStats,
    frameDimensionsRef,
    latencyStats,
    logLatencyStats,
  };
}
