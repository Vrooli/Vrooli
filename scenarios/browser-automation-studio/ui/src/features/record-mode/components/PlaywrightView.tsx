import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { getConfig } from "@/config";

type PointerAction = "move" | "down" | "up" | "click";

interface FramePayload {
  image: string;
  width: number;
  height: number;
  captured_at: string;
}

interface PlaywrightViewProps {
  sessionId: string;
  quality?: number;
  fps?: number;
  onStreamError?: (message: string) => void;
  refreshToken?: number;
}

/**
 * Renders a live view of the Playwright session by polling lightweight JPEG frames
 * and forwards user input (pointer/keyboard/wheel) back to the driver.
 */
export function PlaywrightView({
  sessionId,
  quality = 50,
  fps = 3,
  onStreamError,
  refreshToken,
}: PlaywrightViewProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [frame, setFrame] = useState<FramePayload | null>(null);
  const [isFetching, setIsFetching] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const lastMoveRef = useRef(0);
  const inFlightRef = useRef(false);
  const lastFrameRef = useRef<FramePayload | null>(null);
  const lastSessionRef = useRef<string | null>(null);

  const pollInterval = useMemo(() => Math.max(300, Math.floor(1000 / fps)), [fps]);
  const pollIntervalRef = useRef(pollInterval);
  pollIntervalRef.current = pollInterval;

  const fetchFrame = useCallback(async () => {
    if (inFlightRef.current) return;
    inFlightRef.current = true;
    setIsFetching(true);
    const started = performance.now();
    try {
      const config = await getConfig();
      const endpoint = `${config.API_URL}/recordings/live/${sessionId}/frame?quality=${quality}&t=${Date.now()}`;
      const res = await fetch(endpoint);
      if (!res.ok) {
        throw new Error(`Frame fetch failed (${res.status})`);
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

  useEffect(() => {
    let cancelled = false;

    const tick = async () => {
      if (cancelled) return;
      await fetchFrame();
      if (cancelled) return;
      const nextInterval = pollIntervalRef.current;
      setTimeout(tick, nextInterval);
    };

    tick();

    return () => {
      cancelled = true;
    };
  }, [fetchFrame, pollInterval, refreshToken]);

  useEffect(() => {
    if (lastSessionRef.current !== sessionId) {
      lastSessionRef.current = sessionId;
      setFrame(null);
      lastFrameRef.current = null;
      setError(null);
    }
  }, [sessionId]);

  const sendInput = useCallback(
    async (payload: unknown) => {
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
    [onStreamError, sessionId]
  );

  const getScaledPoint = useCallback(
    (clientX: number, clientY: number) => {
      const rect = containerRef.current?.getBoundingClientRect();
      if (!rect || !frame?.width || !frame?.height) return { x: 0, y: 0 };
      const scale = Math.min(rect.width / frame.width, rect.height / frame.height);
      const displayWidth = frame.width * scale;
      const displayHeight = frame.height * scale;
      const offsetX = (rect.width - displayWidth) / 2;
      const offsetY = (rect.height - displayHeight) / 2;
      const relativeX = clientX - rect.left - offsetX;
      const relativeY = clientY - rect.top - offsetY;
      const clampedX = Math.max(0, Math.min(displayWidth, relativeX));
      const clampedY = Math.max(0, Math.min(displayHeight, relativeY));
      const xRatio = frame.width / displayWidth;
      const yRatio = frame.height / displayHeight;
      return {
        x: clampedX * xRatio,
        y: clampedY * yRatio,
      };
    },
    [frame?.height, frame?.width]
  );

  const handlePointer = useCallback(
    (action: PointerAction, e: React.PointerEvent<HTMLDivElement>) => {
      if (!frame) return;
      if (action === "move") {
        const now = performance.now();
        if (now - lastMoveRef.current < 50) return; // throttle move
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
          className="w-full h-full object-contain bg-white"
        />
      ) : (
        <div className="absolute inset-0 flex items-center justify-center text-sm text-gray-500 dark:text-gray-400">
          {isFetching ? "Connecting to live session…" : "Waiting for first frame…"}
        </div>
      )}

      <div className="absolute left-2 top-2 rounded-full bg-white/90 dark:bg-gray-900/80 px-3 py-1 text-[11px] text-gray-700 dark:text-gray-200 border border-gray-200 dark:border-gray-700 shadow-sm">
        Live Playwright session
        {(frame ?? lastFrameRef.current) && (
          <span className="ml-2 text-gray-500 dark:text-gray-400">
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
