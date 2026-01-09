/**
 * useInputForwarding Hook
 *
 * Handles forwarding user input (pointer, keyboard, wheel) to the Playwright session.
 * Extracted from PlaywrightView for better separation of concerns.
 *
 * Features:
 * - Coordinate mapping from screen space to viewport space
 * - HiDPI-aware mapping (frame dimensions vs viewport dimensions)
 * - WebSocket-based input delivery (low latency)
 * - HTTP fallback when WebSocket unavailable
 * - Move event throttling (16ms ~60fps for smooth cursor tracking)
 * - Multi-tab support via pageId
 *
 * IMPORTANT: Both viewport AND frame dimensions should be provided for accurate
 * coordinate mapping on HiDPI displays. The frame dimensions determine how the
 * image is displayed (object-contain), while viewport dimensions are the output
 * coordinate space for Playwright.
 *
 * If viewport is null/undefined, pointer events will NOT be forwarded to prevent
 * incorrect cursor positions.
 */

import React, { useCallback, useRef } from 'react';
import { getConfig } from '@/config';
import { useWebSocket } from '@/contexts/WebSocketContext';
import { mapClientToViewportWithFrame, type Rect } from '../utils/coordinateMapping';

type PointerAction = 'move' | 'down' | 'up' | 'click';

export interface UseInputForwardingOptions {
  sessionId: string | null;
  pageId?: string;
  /** Viewport dimensions (what Playwright uses - output coordinate space) */
  viewport?: { width: number; height: number } | null;
  /** Frame dimensions (bitmap size - for display calculation on HiDPI) */
  frameDimensions?: { width: number; height: number } | null;
  onError?: (message: string) => void;
}

export interface UseInputForwardingResult {
  /**
   * Handle pointer events (move, down, up).
   * @param action - The pointer action type
   * @param e - The React pointer event
   * @param canvasRect - The bounding rect of the canvas element
   * @param hasFrame - Whether a frame has been received
   * @param canvasInternalDimensions - Optional canvas internal dimensions (more reliable than state)
   */
  handlePointer: (
    action: PointerAction,
    e: React.PointerEvent<HTMLElement>,
    canvasRect: Rect | null,
    hasFrame: boolean,
    canvasInternalDimensions?: { width: number; height: number }
  ) => void;

  /**
   * Handle wheel events.
   * @param e - The React wheel event
   * @param hasFrame - Whether a frame has been received
   */
  handleWheel: (e: React.WheelEvent<HTMLElement>, hasFrame: boolean) => void;

  /**
   * Handle keyboard events.
   * @param e - The React keyboard event
   * @param hasFrame - Whether a frame has been received
   */
  handleKey: (e: React.KeyboardEvent<HTMLElement>, hasFrame: boolean) => void;

  /**
   * Whether WebSocket is subscribed for input forwarding.
   */
  isWsSubscribed: boolean;

  /**
   * Set the WebSocket subscription state (for coordination with frame streaming).
   */
  setWsSubscribed: (subscribed: boolean) => void;
}

export function useInputForwarding({
  sessionId,
  pageId,
  viewport,
  frameDimensions,
  onError,
}: UseInputForwardingOptions): UseInputForwardingResult {
  const lastMoveRef = useRef(0);
  const wsSubscribedRef = useRef(false);

  const { isConnected, send } = useWebSocket();

  /**
   * Send input via WebSocket for low latency (falls back to HTTP if WS unavailable).
   */
  const sendInput = useCallback(
    async (payload: unknown) => {
      if (!sessionId) return;

      // Prefer WebSocket for lower latency
      if (isConnected && wsSubscribedRef.current) {
        const message: Record<string, unknown> = {
          type: 'recording_input',
          session_id: sessionId,
          input: payload,
        };
        if (pageId) {
          message.page_id = pageId;
        }
        send(message);
        return;
      }

      // Fallback to HTTP POST
      try {
        const config = await getConfig();
        const body = pageId
          ? { ...payload as Record<string, unknown>, page_id: pageId }
          : payload;
        const res = await fetch(`${config.API_URL}/recordings/live/${sessionId}/input`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(body),
        });
        if (!res.ok) {
          const text = await res.text();
          throw new Error(text || `Input dispatch failed (${res.status})`);
        }
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Failed to forward input';
        if (onError) {
          onError(message);
        }
      }
    },
    [isConnected, pageId, send, sessionId, onError]
  );

  /**
   * Get scaled point from client coordinates to viewport coordinates.
   *
   * Uses frame dimensions for display calculation (how the image is shown with object-contain)
   * and viewport dimensions for the output coordinate space (what Playwright uses).
   *
   * This correctly handles HiDPI displays where frame dimensions differ from viewport.
   *
   * @param clientX - X coordinate from pointer event
   * @param clientY - Y coordinate from pointer event
   * @param canvasRect - Bounding rect of the canvas element
   * @param overrideFrameDimensions - Optional frame dimensions (when passed directly from canvas element)
   */
  const getScaledPointWithDimensions = useCallback(
    (
      clientX: number,
      clientY: number,
      canvasRect: Rect | null,
      overrideFrameDimensions?: { width: number; height: number } | null
    ) => {
      const viewportWidth = viewport?.width;
      const viewportHeight = viewport?.height;
      // Use override dimensions if provided, otherwise fall back to state, then to viewport
      const frameWidth = overrideFrameDimensions?.width ?? frameDimensions?.width ?? viewportWidth;
      const frameHeight = overrideFrameDimensions?.height ?? frameDimensions?.height ?? viewportHeight;

      if (!canvasRect || !viewportWidth || !viewportHeight || !frameWidth || !frameHeight) {
        return { x: 0, y: 0 };
      }

      return mapClientToViewportWithFrame(
        clientX,
        clientY,
        canvasRect,
        frameWidth,
        frameHeight,
        viewportWidth,
        viewportHeight
      );
    },
    [viewport?.width, viewport?.height, frameDimensions?.width, frameDimensions?.height]
  );

  const handlePointer = useCallback(
    (
      action: PointerAction,
      e: React.PointerEvent<HTMLElement>,
      canvasRect: Rect | null,
      hasFrame: boolean,
      canvasInternalDimensions?: { width: number; height: number }
    ) => {
      if (!hasFrame) return;

      // CRITICAL: Don't forward pointer events if viewport dimensions are unknown.
      // Without proper viewport dimensions, coordinate mapping will be incorrect
      // (would return 0,0), causing cursor position inaccuracy.
      if (!viewport?.width || !viewport?.height) {
        return;
      }

      if (action === 'move') {
        const now = performance.now();
        // Throttle move events to ~60fps (16ms) for smooth cursor tracking
        // while avoiding excessive network traffic
        if (now - lastMoveRef.current < 16) return;
        lastMoveRef.current = now;
      }

      // Use canvas internal dimensions if provided (more reliable than state),
      // otherwise fall back to frameDimensions state
      const effectiveFrameDimensions = canvasInternalDimensions ?? frameDimensions;
      const point = getScaledPointWithDimensions(
        e.clientX,
        e.clientY,
        canvasRect,
        effectiveFrameDimensions
      );
      const button = e.button === 2 ? 'right' : e.button === 1 ? 'middle' : 'left';

      void sendInput({
        type: 'pointer',
        action,
        x: point.x,
        y: point.y,
        button,
      });

      e.preventDefault();
      e.stopPropagation();
    },
    [getScaledPointWithDimensions, sendInput, viewport?.width, viewport?.height, frameDimensions]
  );

  const handleWheel = useCallback(
    (e: React.WheelEvent<HTMLElement>, hasFrame: boolean) => {
      if (!hasFrame) return;

      void sendInput({
        type: 'wheel',
        delta_x: e.deltaX,
        delta_y: e.deltaY,
      });

      e.preventDefault();
      e.stopPropagation();
    },
    [sendInput]
  );

  const handleKey = useCallback(
    (e: React.KeyboardEvent<HTMLElement>, hasFrame: boolean) => {
      if (!hasFrame) return;

      const modifiers = [
        e.altKey ? 'Alt' : null,
        e.ctrlKey ? 'Control' : null,
        e.metaKey ? 'Meta' : null,
        e.shiftKey ? 'Shift' : null,
      ].filter(Boolean) as string[];

      const payload =
        e.key.length === 1
          ? { type: 'keyboard' as const, text: e.key }
          : { type: 'keyboard' as const, key: e.key, modifiers };

      void sendInput(payload);

      e.preventDefault();
      e.stopPropagation();
    },
    [sendInput]
  );

  const setWsSubscribed = useCallback((subscribed: boolean) => {
    wsSubscribedRef.current = subscribed;
  }, []);

  return {
    handlePointer,
    handleWheel,
    handleKey,
    isWsSubscribed: wsSubscribedRef.current,
    setWsSubscribed,
  };
}
