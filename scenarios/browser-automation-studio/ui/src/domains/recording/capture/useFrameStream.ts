/**
 * useFrameStream Hook
 *
 * Handles live frame streaming from Playwright sessions via WebSocket or polling fallback.
 * Extracted from PlaywrightView for better separation of concerns.
 *
 * Features:
 * - WebSocket-based binary frame streaming (preferred, 30-60 FPS)
 * - Polling fallback when WebSocket unavailable (10-15 FPS)
 * - Double-buffered canvas rendering to prevent white flash
 * - Frame statistics tracking
 * - Page metadata extraction
 * - Automatic reconnection on session change
 */

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { getConfig } from '@/config';
import { useWebSocket } from '@/contexts/WebSocketContext';
import { useFrameStats, type FrameStats } from '../hooks/useFrameStats';

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

/** WebSocket message for frame updates */
interface FrameMessage {
  type: 'recording_frame';
  session_id: string;
  image: string;
  mime: string;
  width: number;
  height: number;
  captured_at: string;
  content_hash: string;
  timestamp: string;
}

/** WebSocket message for subscription confirmation */
interface RecordingSubscribedMessage {
  type: 'recording_subscribed';
  session_id: string;
  timestamp: string;
}

/** Union type for recording-related WebSocket messages */
type RecordingMessage = FrameMessage | RecordingSubscribedMessage;

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

  const inFlightRef = useRef(false);
  const lastSessionRef = useRef<string | null>(null);
  const lastETagRef = useRef<string | null>(null);
  const lastContentHashRef = useRef<string | null>(null);
  const wsSubscribedRef = useRef(false);
  const lastPageIdRef = useRef<string | undefined>(pageId);

  // WebSocket
  const { isConnected, lastMessage, send, subscribeToBinaryFrames } = useWebSocket();

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
  useEffect(() => {
    if (lastPageIdRef.current !== pageId) {
      setIsPageSwitching(true);
      lastPageIdRef.current = pageId;
      frameDimensionsRef.current = null;
      hasFrameRef.current = false;
      setHasFrame(false);

      const timer = setTimeout(() => {
        setIsPageSwitching(false);
      }, 300);

      return () => clearTimeout(timer);
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
   * Draw a bitmap directly to canvas using double-buffering.
   */
  const drawFrameToCanvas = useCallback((bitmap: ImageBitmap) => {
    const canvas = canvasRef.current;
    if (!canvas) return false;

    const ctx = canvas.getContext('2d', { alpha: false });
    if (!ctx) return false;

    const dimensionsChanging = canvas.width !== bitmap.width || canvas.height !== bitmap.height;

    if (dimensionsChanging) {
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
      ctx.drawImage(bitmap, 0, 0);
    }

    return true;
  }, []);

  // Binary frame processing refs
  const pendingBlobRef = useRef<Blob | null>(null);
  const pendingFrameSizeRef = useRef<number>(0);
  const latestFrameIdRef = useRef<number>(0);
  const rafIdRef = useRef<number | null>(null);
  const isWsFrameActiveRef = useRef(isWsFrameActive);
  isWsFrameActiveRef.current = isWsFrameActive;

  /**
   * Process a pending frame - called from requestAnimationFrame.
   */
  const processFrame = useCallback(async () => {
    rafIdRef.current = null;

    const pending = pendingBlobRef.current;
    const frameSize = pendingFrameSizeRef.current;
    const processingFrameId = latestFrameIdRef.current;
    if (!pending) return;

    pendingBlobRef.current = null;

    try {
      const bitmap = await createImageBitmap(pending);

      if (latestFrameIdRef.current > processingFrameId) {
        bitmap.close();
        return;
      }

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
    } catch {
      if (!pendingBlobRef.current) {
        errorRef.current = 'Failed to decode binary frame';
        setError('Failed to decode binary frame');
      }
    }
  }, [drawFrameToCanvas, recordFrame]);

  // Subscribe to recording frames via WebSocket
  useEffect(() => {
    if (!useWebSocketFrames || !isConnected || !sessionId) {
      return;
    }

    if (wsSubscribedRef.current) {
      send({ type: 'unsubscribe_recording' });
      wsSubscribedRef.current = false;
      // CRITICAL: Update ref BEFORE state to prevent race condition with binary frame handler
      isWsFrameActiveRef.current = false;
      setIsWsFrameActive(false);
    }

    const subscriptionMessage: Record<string, unknown> = {
      type: 'subscribe_recording',
      session_id: sessionId,
    };
    if (pageId) {
      subscriptionMessage.page_id = pageId;
    }
    send(subscriptionMessage);
    wsSubscribedRef.current = true;
    // CRITICAL: Update ref BEFORE state to prevent race condition
    // Binary frames may arrive before React re-renders and updates the ref via the render-time sync
    isWsFrameActiveRef.current = true;
    setIsWsFrameActive(true);

    return () => {
      if (wsSubscribedRef.current) {
        send({ type: 'unsubscribe_recording' });
        wsSubscribedRef.current = false;
        // CRITICAL: Update ref BEFORE state
        isWsFrameActiveRef.current = false;
        setIsWsFrameActive(false);
      }
    };
  }, [useWebSocketFrames, isConnected, sessionId, pageId, send]);

  // Handle WebSocket text messages (JSON) - subscription confirmation only
  // Note: Frame data is handled via binary frames (subscribeToBinaryFrames), not JSON messages.
  // The legacy recording_frame JSON handler was removed as binary frames are more efficient.
  useEffect(() => {
    if (!lastMessage || !useWebSocketFrames) return;

    const msg = lastMessage as unknown as RecordingMessage;

    if (msg.type === 'recording_subscribed' && msg.session_id === sessionId) {
      // CRITICAL: Update ref BEFORE state
      isWsFrameActiveRef.current = true;
      setIsWsFrameActive(true);
    }
  }, [lastMessage, sessionId, useWebSocketFrames]);

  // Subscribe to binary frames
  useEffect(() => {
    if (!useWebSocketFrames) return;

    const handleBinaryFrame = (data: ArrayBuffer) => {
      if (!isWsFrameActiveRef.current) {
        droppedFramesRef.current++;
        return;
      }

      const blob = new Blob([data], { type: 'image/jpeg' });
      const frameId = Date.now();

      pendingBlobRef.current = blob;
      pendingFrameSizeRef.current = data.byteLength;
      latestFrameIdRef.current = frameId;

      if (rafIdRef.current !== null) {
        return;
      }

      rafIdRef.current = requestAnimationFrame(() => {
        processFrame();
      });
    };

    const unsubscribe = subscribeToBinaryFrames(handleBinaryFrame);

    return () => {
      unsubscribe();
    };
  }, [useWebSocketFrames, subscribeToBinaryFrames, processFrame]);

  // Cleanup RAF on unmount
  useEffect(() => {
    return () => {
      if (rafIdRef.current !== null) {
        cancelAnimationFrame(rafIdRef.current);
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

  // Polling - runs as fallback when WebSocket not active
  useEffect(() => {
    if (isWsFrameActive) {
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
      wsSubscribedRef.current = false;
      resetStats();
      // Reset diagnostic counters
      frameCountRef.current = 0;
      droppedFramesRef.current = 0;
    }
  }, [sessionId, resetStats]);

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
  };
}
