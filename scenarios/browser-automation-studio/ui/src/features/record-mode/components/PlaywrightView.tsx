import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { getConfig } from "@/config";
import { useWebSocket } from "@/contexts/WebSocketContext";
import { mapClientToViewport } from "../utils/coordinateMapping";
import { useFrameStats, type FrameStats } from "../hooks/useFrameStats";

// Re-export FrameStats type for consumers
export type { FrameStats } from "../hooks/useFrameStats";

type PointerAction = "move" | "down" | "up" | "click";

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
  /** MD5 hash of the raw frame buffer - used by server for ETag generation */
  content_hash?: string;
  /** Current page title from document.title */
  page_title?: string;
  /** Current page URL */
  page_url?: string;
}

/** Page metadata extracted from frames */
export interface PageMetadata {
  title: string;
  url: string;
}

/** WebSocket message for frame updates */
interface FrameMessage {
  type: "recording_frame";
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
  type: "recording_subscribed";
  session_id: string;
  timestamp: string;
}

/** Union type for recording-related WebSocket messages */
type RecordingMessage = FrameMessage | RecordingSubscribedMessage;

interface PlaywrightViewProps {
  sessionId: string;
  quality?: number;
  fps?: number;
  onStreamError?: (message: string) => void;
  refreshToken?: number;
  /** Whether to use WebSocket for frame updates (default: true) */
  useWebSocketFrames?: boolean;
  /** Logical viewport dimensions (for coordinate mapping, independent of device pixel ratio) */
  viewport?: { width: number; height: number };
  /** Callback to receive frame statistics updates */
  onStatsUpdate?: (stats: FrameStats) => void;
  /** Callback when page metadata (title, url) changes */
  onPageMetadataChange?: (metadata: PageMetadata) => void;
}

/**
 * Renders a live view of the Playwright session.
 *
 * Frame delivery modes:
 * 1. WebSocket (preferred): Receives frames pushed from server in real-time
 * 2. Polling (fallback): Polls for frames when WebSocket unavailable
 *
 * Also forwards user input (pointer/keyboard/wheel) back to the driver.
 *
 * ## Coordinate Mapping (IMPORTANT)
 *
 * The live preview displays JPEG screenshots captured by Playwright. These screenshots
 * are captured at the device's pixel ratio (e.g., 2x on HiDPI/Retina displays), meaning
 * a 900x700 viewport produces an 1800x1400 image.
 *
 * When the user clicks on the preview, we must map their click coordinates to the
 * Playwright viewport coordinates, NOT the screenshot bitmap coordinates.
 *
 * Example:
 * - Viewport: 900x700 (logical pixels)
 * - Screenshot: 1800x1400 (physical pixels at 2x DPR)
 * - User clicks at (450, 350) on the displayed preview
 * - Correct: Send (450, 350) to Playwright
 * - Wrong: Send (900, 700) to Playwright (2x scaled - clicks land off-screen!)
 *
 * The `viewport` prop provides the logical viewport dimensions for correct mapping.
 * If not provided, falls back to frame dimensions (which may be wrong on HiDPI).
 */
export function PlaywrightView({
  sessionId,
  quality = 65,
  fps = 6,
  onStreamError,
  refreshToken,
  useWebSocketFrames = true,
  viewport,
  onStatsUpdate,
  onPageMetadataChange,
}: PlaywrightViewProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const canvasRef = useRef<HTMLCanvasElement | null>(null);

  // Store frame dimensions in ref to avoid re-renders on every frame
  // Only trigger re-render when dimensions actually change (for layout) or when we get first frame
  const frameDimensionsRef = useRef<FrameDimensions | null>(null);
  const [hasFrame, setHasFrame] = useState(false);
  const [displayDimensions, setDisplayDimensions] = useState<{ width: number; height: number } | null>(null);

  const [isFetching, setIsFetching] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const lastMoveRef = useRef(0);
  const inFlightRef = useRef(false);
  const lastSessionRef = useRef<string | null>(null);
  const lastETagRef = useRef<string | null>(null);
  const lastContentHashRef = useRef<string | null>(null);
  const [isTabVisible, setIsTabVisible] = useState(!document.hidden);
  const wsSubscribedRef = useRef(false);
  const [isWsFrameActive, setIsWsFrameActive] = useState(false);

  // WebSocket for real-time frame updates (including binary frames)
  const { isConnected, lastMessage, send, subscribeToBinaryFrames } = useWebSocket();

  // Frame statistics tracking
  const { stats: frameStats, recordFrame, reset: resetStats } = useFrameStats();
  const onStatsUpdateRef = useRef(onStatsUpdate);
  onStatsUpdateRef.current = onStatsUpdate;

  // Page metadata tracking (title, url)
  const onPageMetadataChangeRef = useRef(onPageMetadataChange);
  onPageMetadataChangeRef.current = onPageMetadataChange;
  const lastPageMetadataRef = useRef<PageMetadata | null>(null);

  // Track displayed timestamp for UI (updated via ref to avoid re-renders)
  const [displayedTimestamp, setDisplayedTimestamp] = useState<string | null>(null);

  // Track tab visibility to pause polling when hidden
  useEffect(() => {
    const handleVisibilityChange = () => {
      setIsTabVisible(!document.hidden);
    };
    document.addEventListener("visibilitychange", handleVisibilityChange);
    return () => {
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, []);

  // Push frame stats to parent when they change
  useEffect(() => {
    if (onStatsUpdateRef.current) {
      onStatsUpdateRef.current(frameStats);
    }
  }, [frameStats]);

  const pollInterval = useMemo(() => Math.max(300, Math.floor(1000 / fps)), [fps]);
  const pollIntervalRef = useRef(pollInterval);
  pollIntervalRef.current = pollInterval;

  // Subscribe to recording frames via WebSocket
  useEffect(() => {
    if (!useWebSocketFrames || !isConnected || !sessionId) {
      return;
    }

    // Subscribe to recording updates for this session (includes frames)
    if (!wsSubscribedRef.current) {
      send({ type: "subscribe_recording", session_id: sessionId });
      wsSubscribedRef.current = true;
      console.log("[PlaywrightView] Subscribed to recording frames via WebSocket");
    }

    return () => {
      if (wsSubscribedRef.current) {
        send({ type: "unsubscribe_recording" });
        wsSubscribedRef.current = false;
        setIsWsFrameActive(false);
        console.log("[PlaywrightView] Unsubscribed from recording frames");
      }
    };
  }, [useWebSocketFrames, isConnected, sessionId, send]);

  // Handle WebSocket text messages (JSON)
  useEffect(() => {
    if (!lastMessage || !useWebSocketFrames) return;

    const msg = lastMessage as unknown as RecordingMessage;

    // Handle recording_subscribed confirmation
    if (msg.type === "recording_subscribed" && msg.session_id === sessionId) {
      setIsWsFrameActive(true);
      console.log("[PlaywrightView] WebSocket frame streaming active (binary mode)");
      return;
    }

    // Handle legacy JSON frame updates (fallback for base64-encoded frames)
    if (msg.type === "recording_frame" && msg.session_id === sessionId) {
      // Skip if same content hash (shouldn't happen but be safe)
      if (lastContentHashRef.current === msg.content_hash) {
        return;
      }
      lastContentHashRef.current = msg.content_hash;

      // Preload the image and draw to canvas
      const img = new Image();
      img.onload = () => {
        // Draw to canvas
        const canvas = canvasRef.current;
        if (canvas) {
          const ctx = canvas.getContext("2d", { alpha: false });
          if (ctx) {
            if (canvas.width !== img.width || canvas.height !== img.height) {
              canvas.width = img.width;
              canvas.height = img.height;
            }
            ctx.drawImage(img, 0, 0);
          }
        }

        // Update ref and minimal state
        frameDimensionsRef.current = {
          width: msg.width,
          height: msg.height,
          capturedAt: msg.captured_at,
        };

        if (!hasFrame) {
          setHasFrame(true);
        }
        if (!displayDimensions || displayDimensions.width !== msg.width || displayDimensions.height !== msg.height) {
          setDisplayDimensions({ width: msg.width, height: msg.height });
        }
        setDisplayedTimestamp(msg.captured_at);
        setError(null);

        // Mark WebSocket frames as active once we receive the first frame
        if (!isWsFrameActive) {
          setIsWsFrameActive(true);
        }
      };
      img.onerror = () => {
        setError("Failed to decode WebSocket frame");
      };
      img.src = msg.image;
    }
  }, [lastMessage, sessionId, useWebSocketFrames, isWsFrameActive, hasFrame, displayDimensions]);

  // Handle WebSocket binary frames (raw JPEG data - more efficient)
  // Performance optimizations:
  // 1. Direct callback subscription (no React state updates on each frame!)
  // 2. createImageBitmap decodes off-main-thread (non-blocking)
  // 3. requestAnimationFrame batches updates to vsync (no dropped frames)
  // 4. Canvas drawImage is faster than img element (no Blob URL overhead)
  // 5. Only re-render React when dimensions change, not every frame
  const pendingBlobRef = useRef<Blob | null>(null);
  const pendingFrameSizeRef = useRef<number>(0);
  const latestFrameIdRef = useRef<number>(0);
  const rafIdRef = useRef<number | null>(null);
  // Track whether WS frames are active in a ref to avoid stale closures
  const isWsFrameActiveRef = useRef(isWsFrameActive);
  isWsFrameActiveRef.current = isWsFrameActive;

  /**
   * Draw a bitmap directly to canvas - bypasses Blob URL creation entirely.
   * This is significantly faster than creating/revoking Blob URLs per frame.
   */
  const drawFrameToCanvas = useCallback((bitmap: ImageBitmap) => {
    const canvas = canvasRef.current;
    if (!canvas) return false;

    const ctx = canvas.getContext("2d", { alpha: false });
    if (!ctx) return false;

    // Resize canvas if needed (only when dimensions change)
    if (canvas.width !== bitmap.width || canvas.height !== bitmap.height) {
      canvas.width = bitmap.width;
      canvas.height = bitmap.height;
    }

    // Draw directly - no intermediate Blob URL needed
    ctx.drawImage(bitmap, 0, 0);
    return true;
  }, []);

  /**
   * Process a pending frame - called from requestAnimationFrame.
   * This is extracted to avoid recreating the async function on each frame.
   */
  const processFrame = useCallback(async () => {
    rafIdRef.current = null;

    // Get the most recent pending frame
    const pending = pendingBlobRef.current;
    const frameSize = pendingFrameSizeRef.current;
    const processingFrameId = latestFrameIdRef.current;
    if (!pending) return;

    // Clear the pending blob (we're about to process it)
    pendingBlobRef.current = null;

    try {
      // Use createImageBitmap for off-main-thread decoding (non-blocking)
      const bitmap = await createImageBitmap(pending);

      // Check if a newer frame arrived while we were decoding
      if (latestFrameIdRef.current > processingFrameId) {
        // Newer frame exists - discard this one
        bitmap.close();
        return;
      }

      // Draw directly to canvas (no Blob URL overhead!)
      const drawn = drawFrameToCanvas(bitmap);
      if (!drawn) {
        bitmap.close();
        return;
      }

      // Update dimensions ref (no re-render triggered)
      const newDimensions: FrameDimensions = {
        width: bitmap.width,
        height: bitmap.height,
        capturedAt: new Date().toISOString(),
      };

      // Only trigger re-render if this is first frame or dimensions changed
      const prevDims = frameDimensionsRef.current;
      const dimensionsChanged = !prevDims ||
        prevDims.width !== bitmap.width ||
        prevDims.height !== bitmap.height;

      frameDimensionsRef.current = newDimensions;

      // Clean up bitmap after drawing
      bitmap.close();

      // Update React state only when necessary
      setHasFrame(prev => prev || true);
      if (dimensionsChanged) {
        setDisplayDimensions({ width: newDimensions.width, height: newDimensions.height });
      }

      // Update timestamp periodically (not every frame - reduces re-renders)
      // Update every 500ms for UI display
      const now = Date.now();
      if (now % 500 < 100) {
        setDisplayedTimestamp(newDimensions.capturedAt);
      }

      setError(null);

      // Record frame stats
      recordFrame(frameSize);
    } catch {
      // Decode failed - only show error if no newer frame is pending
      if (!pendingBlobRef.current) {
        setError("Failed to decode binary frame");
      }
    }
  }, [drawFrameToCanvas, recordFrame]);

  // Subscribe to binary frames directly (avoids React state updates!)
  useEffect(() => {
    if (!useWebSocketFrames) return;

    const handleBinaryFrame = (data: ArrayBuffer) => {
      // Skip if WS frames not active yet (waiting for subscription confirmation)
      if (!isWsFrameActiveRef.current) return;

      // Create blob from binary data (JPEG format)
      // Note: createImageBitmap auto-detects format from magic bytes
      const blob = new Blob([data], { type: "image/jpeg" });
      const frameId = Date.now(); // Use timestamp as frame ID for ordering

      // Store as pending frame - this supersedes any previous pending frame
      pendingBlobRef.current = blob;
      pendingFrameSizeRef.current = data.byteLength;
      latestFrameIdRef.current = frameId;

      // If we already have a RAF scheduled, it will pick up the new frame
      if (rafIdRef.current !== null) return;

      // Schedule frame processing on next animation frame
      rafIdRef.current = requestAnimationFrame(() => {
        processFrame();
      });
    };

    // Subscribe and get unsubscribe function
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

  const fetchFrame = useCallback(async () => {
    if (inFlightRef.current) return;
    inFlightRef.current = true;
    setIsFetching(true);
    const started = performance.now();
    try {
      const config = await getConfig();
      // Remove cache-busting timestamp; use ETag for conditional requests instead
      const endpoint = `${config.API_URL}/recordings/live/${sessionId}/frame?quality=${quality}`;
      const headers: HeadersInit = {};
      // Send If-None-Match header to skip identical frames
      if (lastETagRef.current) {
        headers["If-None-Match"] = lastETagRef.current;
      }
      const res = await fetch(endpoint, { headers });

      // Handle 304 Not Modified - frame hasn't changed, skip update
      if (res.status === 304) {
        setError(null);
        return;
      }

      if (!res.ok) {
        throw new Error(`Frame fetch failed (${res.status})`);
      }

      // Store ETag for next request
      const etag = res.headers.get("ETag");
      if (etag) {
        lastETagRef.current = etag;
      }

      const data = (await res.json()) as FramePayload;

      // Notify about page metadata changes (title/url for history)
      if (data.page_title !== undefined || data.page_url !== undefined) {
        const newMetadata: PageMetadata = {
          title: data.page_title || '',
          url: data.page_url || '',
        };
        const lastMetadata = lastPageMetadataRef.current;
        // Only notify if metadata actually changed
        if (!lastMetadata || lastMetadata.title !== newMetadata.title || lastMetadata.url !== newMetadata.url) {
          lastPageMetadataRef.current = newMetadata;
          if (onPageMetadataChangeRef.current) {
            onPageMetadataChangeRef.current(newMetadata);
          }
        }
      }

      // Preload the image and draw to canvas
      const img = new Image();
      img.onload = () => {
        // Draw to canvas for polling mode too
        const canvas = canvasRef.current;
        if (canvas) {
          const ctx = canvas.getContext("2d", { alpha: false });
          if (ctx) {
            if (canvas.width !== img.width || canvas.height !== img.height) {
              canvas.width = img.width;
              canvas.height = img.height;
            }
            ctx.drawImage(img, 0, 0);
          }
        }

        // Update ref and state
        frameDimensionsRef.current = {
          width: img.width,
          height: img.height,
          capturedAt: data.captured_at,
        };

        if (!hasFrame) {
          setHasFrame(true);
        }
        if (!displayDimensions || displayDimensions.width !== img.width || displayDimensions.height !== img.height) {
          setDisplayDimensions({ width: img.width, height: img.height });
        }
        setDisplayedTimestamp(data.captured_at);
        setError(null);
      };
      img.onerror = () => {
        setError("Failed to decode live frame");
      };
      img.src = data.image;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Failed to fetch live frame";
      setError(message);
      if (onStreamError) {
        onStreamError(message);
      }
    } finally {
      setIsFetching(false);
      inFlightRef.current = false;
      const elapsed = performance.now() - started;
      // Slow down polling if frames are taking longer than the target interval.
      if (elapsed > pollInterval) {
        const dampenedFps = Math.max(1, Math.round(1000 / Math.min(elapsed, 1500)));
        const nextInterval = Math.max(400, Math.floor(1000 / dampenedFps));
        pollIntervalRef.current = nextInterval;
      }
    }
  }, [fps, onStreamError, quality, sessionId, pollInterval, hasFrame, displayDimensions]);

  // Polling - runs as primary method, stops when WebSocket frames become active
  // This ensures frames are always displayed, with WebSocket as an optimization
  useEffect(() => {
    // Stop polling if WebSocket frames are active
    if (isWsFrameActive) {
      console.log("[PlaywrightView] Stopping polling - WebSocket frames active");
      return;
    }

    let cancelled = false;
    let timeoutId: ReturnType<typeof setTimeout> | null = null;

    const tick = async () => {
      if (cancelled || isWsFrameActive) return;
      // Skip frame fetch when tab is hidden to save bandwidth
      if (isTabVisible) {
        await fetchFrame();
      }
      if (cancelled || isWsFrameActive) return;
      const nextInterval = pollIntervalRef.current;
      timeoutId = setTimeout(tick, nextInterval);
    };

    // Start polling immediately - WebSocket will take over once active
    console.log("[PlaywrightView] Starting frame polling");
    tick();

    return () => {
      cancelled = true;
      if (timeoutId) {
        clearTimeout(timeoutId);
      }
    };
  }, [fetchFrame, pollInterval, refreshToken, isTabVisible, isWsFrameActive]);

  useEffect(() => {
    if (lastSessionRef.current !== sessionId) {
      lastSessionRef.current = sessionId;
      // Reset frame state
      frameDimensionsRef.current = null;
      setHasFrame(false);
      setDisplayDimensions(null);
      setDisplayedTimestamp(null);
      // Clear canvas
      const canvas = canvasRef.current;
      if (canvas) {
        const ctx = canvas.getContext("2d");
        if (ctx) {
          ctx.clearRect(0, 0, canvas.width, canvas.height);
        }
      }
      lastETagRef.current = null; // Reset ETag for new session
      lastContentHashRef.current = null; // Reset content hash for new session
      setError(null);
      setIsWsFrameActive(false); // Reset WebSocket frame state for new session
      wsSubscribedRef.current = false;
      resetStats(); // Reset frame stats for new session
    }
  }, [sessionId, resetStats]);

  // Send input via WebSocket for low latency (falls back to HTTP if WS unavailable)
  const sendInput = useCallback(
    async (payload: unknown) => {
      // Prefer WebSocket for lower latency
      if (isConnected && wsSubscribedRef.current) {
        send({
          type: "recording_input",
          session_id: sessionId,
          input: payload,
        });
        return;
      }

      // Fallback to HTTP POST if WebSocket not available
      try {
        const config = await getConfig();
        const res = await fetch(`${config.API_URL}/recordings/live/${sessionId}/input`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        });
        if (!res.ok) {
          const text = await res.text();
          throw new Error(text || `Input dispatch failed (${res.status})`);
        }
      } catch (err) {
        const message = err instanceof Error ? err.message : "Failed to forward input";
        setError(message);
        if (onStreamError) {
          onStreamError(message);
        }
      }
    },
    [isConnected, onStreamError, send, sessionId]
  );

  const getScaledPoint = useCallback(
    (clientX: number, clientY: number) => {
      const rect = containerRef.current?.getBoundingClientRect();
      // Use logical viewport dimensions for coordinate mapping (not bitmap dimensions which include device pixel ratio)
      // Fall back to frame dimensions if viewport not provided (may be incorrect on HiDPI - see coordinateMapping.ts)
      const currentDims = frameDimensionsRef.current;
      const targetWidth = viewport?.width || currentDims?.width;
      const targetHeight = viewport?.height || currentDims?.height;
      if (!rect || !targetWidth || !targetHeight) return { x: 0, y: 0 };

      return mapClientToViewport(clientX, clientY, rect, targetWidth, targetHeight);
    },
    [viewport?.height, viewport?.width]
  );

  const handlePointer = useCallback(
    (action: PointerAction, e: React.PointerEvent<HTMLDivElement>) => {
      // Use ref to avoid stale frame state in callbacks
      if (!frameDimensionsRef.current) {
        return;
      }
      if (action === "move") {
        const now = performance.now();
        if (now - lastMoveRef.current < 100) return; // throttle move (100ms = max 10 moves/sec)
        lastMoveRef.current = now;
      }

      const point = getScaledPoint(e.clientX, e.clientY);
      const button = e.button === 2 ? "right" : e.button === 1 ? "middle" : "left";
      void sendInput({
        type: "pointer",
        action,
        x: point.x,
        y: point.y,
        button,
      });
      e.preventDefault();
      e.stopPropagation();
    },
    [getScaledPoint, sendInput]
  );

  const handleWheel = useCallback(
    (e: React.WheelEvent<HTMLDivElement>) => {
      if (!frameDimensionsRef.current) return;
      void sendInput({
        type: "wheel",
        delta_x: e.deltaX,
        delta_y: e.deltaY,
      });
      e.preventDefault();
      e.stopPropagation();
    },
    [sendInput]
  );

  const handleKey = useCallback(
    (e: React.KeyboardEvent<HTMLDivElement>) => {
      if (!frameDimensionsRef.current) return;
      const modifiers = [
        e.altKey ? "Alt" : null,
        e.ctrlKey ? "Control" : null,
        e.metaKey ? "Meta" : null,
        e.shiftKey ? "Shift" : null,
      ].filter(Boolean) as string[];

      const payload =
        e.key.length === 1
          ? { type: "keyboard" as const, text: e.key }
          : { type: "keyboard" as const, key: e.key, modifiers };

      void sendInput(payload);
      e.preventDefault();
      e.stopPropagation();
    },
    [sendInput]
  );

  return (
    <div
      ref={containerRef}
      tabIndex={0}
      onPointerMove={(e) => handlePointer("move", e)}
      onPointerDown={(e) => {
        containerRef.current?.focus();
        handlePointer("down", e);
      }}
      onPointerUp={(e) => handlePointer("up", e)}
      onWheel={handleWheel}
      onKeyDown={handleKey}
      className="relative h-full w-full overflow-hidden rounded-md border border-gray-200 dark:border-gray-800 focus:outline-none focus:ring-2 focus:ring-blue-500"
    >
      {/* Canvas for frame rendering - much faster than img + Blob URL */}
      <canvas
        ref={canvasRef}
        className={`w-full h-full object-contain bg-white pointer-events-none ${hasFrame ? "" : "hidden"}`}
        // Set initial dimensions, will be updated when frames arrive
        width={displayDimensions?.width || 1280}
        height={displayDimensions?.height || 720}
      />

      {/* Loading state - shown when no frame yet */}
      {!hasFrame && (
        <div className="absolute inset-0 flex items-center justify-center text-sm text-gray-500 dark:text-gray-400">
          {isFetching ? "Connecting to live session…" : "Waiting for first frame…"}
        </div>
      )}

      <div
        className="absolute left-2 top-2 rounded-full bg-white/90 dark:bg-gray-900/80 px-3 py-1 text-[11px] text-gray-700 dark:text-gray-200 border border-gray-200 dark:border-gray-700 shadow-sm flex items-center gap-2 cursor-help group"
        title="Recording session is synced to this URL. Interactions are executed in the Playwright browser session."
      >
        <span className={`w-2 h-2 rounded-full ${isWsFrameActive ? "bg-green-500" : "bg-yellow-500"}`} />
        Live {isWsFrameActive ? "(WebSocket)" : "(polling)"}
        {displayedTimestamp && (
          <span className="text-gray-500 dark:text-gray-400">
            {new Date(displayedTimestamp).toLocaleTimeString()}
          </span>
        )}
      </div>

      {error && (
        <div className="absolute inset-x-4 bottom-4 rounded-md bg-red-50 dark:bg-red-900/30 text-red-700 dark:text-red-200 text-xs px-3 py-2 border border-red-100 dark:border-red-800 shadow-sm">
          {error}. Streaming will retry automatically.
        </div>
      )}
    </div>
  );
}
