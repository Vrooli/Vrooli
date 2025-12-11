import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { getConfig } from "@/config";
import { useWebSocket } from "@/contexts/WebSocketContext";
import { mapClientToViewport } from "../utils/coordinateMapping";
import { useFrameStats, type FrameStats } from "../hooks/useFrameStats";

// Re-export FrameStats type for consumers
export type { FrameStats } from "../hooks/useFrameStats";

type PointerAction = "move" | "down" | "up" | "click";

interface FramePayload {
  image: string;
  width: number;
  height: number;
  captured_at: string;
  /** MD5 hash of the raw frame buffer - used by server for ETag generation */
  content_hash?: string;
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
}: PlaywrightViewProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [frame, setFrame] = useState<FramePayload | null>(null);
  const [isFetching, setIsFetching] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const lastMoveRef = useRef(0);
  const inFlightRef = useRef(false);
  const lastFrameRef = useRef<FramePayload | null>(null);
  const lastSessionRef = useRef<string | null>(null);
  const lastETagRef = useRef<string | null>(null);
  const lastContentHashRef = useRef<string | null>(null);
  const [isTabVisible, setIsTabVisible] = useState(!document.hidden);
  const wsSubscribedRef = useRef(false);
  const [isWsFrameActive, setIsWsFrameActive] = useState(false);

  // WebSocket for real-time frame updates (including binary frames)
  const { isConnected, lastMessage, lastBinaryFrame, send } = useWebSocket();
  const blobUrlRef = useRef<string | null>(null);

  // Frame statistics tracking
  const { stats: frameStats, recordFrame, reset: resetStats } = useFrameStats();
  const onStatsUpdateRef = useRef(onStatsUpdate);
  onStatsUpdateRef.current = onStatsUpdate;

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

      const frameData: FramePayload = {
        image: msg.image,
        width: msg.width,
        height: msg.height,
        captured_at: msg.captured_at,
        content_hash: msg.content_hash,
      };

      // Preload the image before swapping to avoid flicker
      const img = new Image();
      img.onload = () => {
        setFrame(frameData);
        lastFrameRef.current = frameData;
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
  }, [lastMessage, sessionId, useWebSocketFrames, isWsFrameActive]);

  // Handle WebSocket binary frames (raw JPEG data - more efficient)
  // Uses requestAnimationFrame + createImageBitmap for optimal rendering performance:
  // 1. createImageBitmap decodes off-main-thread (non-blocking)
  // 2. requestAnimationFrame batches updates to vsync (no dropped frames)
  // 3. Only one pending frame at a time - newer frames supersede older ones
  const pendingBlobUrlRef = useRef<string | null>(null);
  const pendingBlobRef = useRef<Blob | null>(null);
  const latestFrameIdRef = useRef<number>(0);
  const rafIdRef = useRef<number | null>(null);

  useEffect(() => {
    if (!lastBinaryFrame || !useWebSocketFrames || !isWsFrameActive) return;

    // Create blob from binary data
    const blob = new Blob([lastBinaryFrame], { type: "image/jpeg" });
    const frameId = Date.now(); // Use timestamp as frame ID for ordering

    // Store as pending frame - this supersedes any previous pending frame
    pendingBlobRef.current = blob;
    latestFrameIdRef.current = frameId;

    // If we already have a RAF scheduled, it will pick up the new frame
    if (rafIdRef.current !== null) return;

    // Schedule frame processing on next animation frame
    rafIdRef.current = requestAnimationFrame(async () => {
      rafIdRef.current = null;

      // Get the most recent pending frame
      const pending = pendingBlobRef.current;
      const processingFrameId = latestFrameIdRef.current;
      if (!pending) return;

      // Clear the pending blob (we're about to process it)
      pendingBlobRef.current = null;

      try {
        // Use createImageBitmap for off-main-thread decoding (more efficient than new Image())
        const bitmap = await createImageBitmap(pending);

        // Create blob URL for the img element
        const url = URL.createObjectURL(pending);

        // Check if a newer frame arrived while we were decoding
        if (latestFrameIdRef.current > processingFrameId) {
          // Newer frame exists - discard this one
          URL.revokeObjectURL(url);
          bitmap.close();
          return;
        }

        // Revoke old blob URL before setting new one
        if (blobUrlRef.current && blobUrlRef.current !== url) {
          URL.revokeObjectURL(blobUrlRef.current);
        }
        blobUrlRef.current = url;
        pendingBlobUrlRef.current = url;

        // Create frame data
        const frameData: FramePayload = {
          image: url,
          width: bitmap.width || frame?.width || 1280,
          height: bitmap.height || frame?.height || 720,
          captured_at: new Date().toISOString(),
        };

        // Clean up bitmap (we're using the blob URL for display)
        bitmap.close();

        // Update state - this triggers a single re-render with the ready-to-display frame
        setFrame(frameData);
        lastFrameRef.current = frameData;
        setError(null);

        // Record frame stats (use pending.size for accurate byte count)
        recordFrame(pending.size);
      } catch {
        // Decode failed - only show error if no newer frame is pending
        if (!pendingBlobRef.current) {
          setError("Failed to decode binary frame");
        }
      }
    });
  }, [lastBinaryFrame, useWebSocketFrames, isWsFrameActive, frame?.width, frame?.height, recordFrame]);

  // Cleanup RAF on unmount
  useEffect(() => {
    return () => {
      if (rafIdRef.current !== null) {
        cancelAnimationFrame(rafIdRef.current);
      }
    };
  }, []);

  // Cleanup blob URL on unmount
  useEffect(() => {
    return () => {
      if (blobUrlRef.current) {
        URL.revokeObjectURL(blobUrlRef.current);
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
      // Preload the image before swapping to avoid flicker.
      const img = new Image();
      img.onload = () => {
        setFrame(data);
        lastFrameRef.current = data;
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
  }, [fps, onStreamError, quality, sessionId, pollInterval]);

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
      setFrame(null);
      lastFrameRef.current = null;
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
      const currentFrame = frame || lastFrameRef.current;
      const targetWidth = viewport?.width || currentFrame?.width;
      const targetHeight = viewport?.height || currentFrame?.height;
      if (!rect || !targetWidth || !targetHeight) return { x: 0, y: 0 };

      return mapClientToViewport(clientX, clientY, rect, targetWidth, targetHeight);
    },
    [frame?.height, frame?.width, viewport?.height, viewport?.width]
  );

  const handlePointer = useCallback(
    (action: PointerAction, e: React.PointerEvent<HTMLDivElement>) => {
      // Use ref to avoid stale frame state in callbacks
      const currentFrame = frame || lastFrameRef.current;
      if (!currentFrame) {
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
    [frame, getScaledPoint, sendInput]
  );

  const handleWheel = useCallback(
    (e: React.WheelEvent<HTMLDivElement>) => {
      if (!frame) return;
      void sendInput({
        type: "wheel",
        delta_x: e.deltaX,
        delta_y: e.deltaY,
      });
      e.preventDefault();
      e.stopPropagation();
    },
    [frame, sendInput]
  );

  const handleKey = useCallback(
    (e: React.KeyboardEvent<HTMLDivElement>) => {
      if (!frame) return;
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
    [frame, sendInput]
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
      {lastFrameRef.current ? (
        <img
          src={(frame ?? lastFrameRef.current)?.image}
          alt="Live Playwright preview"
          className="w-full h-full object-contain bg-white pointer-events-none"
        />
      ) : (
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
        {(frame ?? lastFrameRef.current) && (
          <span className="text-gray-500 dark:text-gray-400">
            {new Date((frame ?? lastFrameRef.current)!.captured_at).toLocaleTimeString()}
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
