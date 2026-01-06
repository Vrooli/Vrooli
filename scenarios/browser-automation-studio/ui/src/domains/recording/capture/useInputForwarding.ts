/**
 * useInputForwarding Hook
 *
 * Handles forwarding user input (pointer, keyboard, wheel) to the Playwright session.
 * Extracted from PlaywrightView for better separation of concerns.
 *
 * Features:
 * - Coordinate mapping from screen space to viewport space
 * - WebSocket-based input delivery (low latency)
 * - HTTP fallback when WebSocket unavailable
 * - Move event throttling (100ms)
 * - Multi-tab support via pageId
 */

import React, { useCallback, useRef } from 'react';
import { getConfig } from '@/config';
import { useWebSocket } from '@/contexts/WebSocketContext';
import { mapClientToViewport, type Rect } from '../utils/coordinateMapping';

type PointerAction = 'move' | 'down' | 'up' | 'click';

export interface UseInputForwardingOptions {
  sessionId: string | null;
  pageId?: string;
  viewport?: { width: number; height: number } | null;
  onError?: (message: string) => void;
}

export interface UseInputForwardingResult {
  /**
   * Handle pointer events (move, down, up).
   * @param action - The pointer action type
   * @param e - The React pointer event
   * @param canvasRect - The bounding rect of the canvas element
   * @param hasFrame - Whether a frame has been received
   */
  handlePointer: (
    action: PointerAction,
    e: React.PointerEvent<HTMLElement>,
    canvasRect: Rect | null,
    hasFrame: boolean
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
   */
  const getScaledPoint = useCallback(
    (clientX: number, clientY: number, canvasRect: Rect | null, fallbackDims?: { width: number; height: number }) => {
      const targetWidth = viewport?.width || fallbackDims?.width;
      const targetHeight = viewport?.height || fallbackDims?.height;
      if (!canvasRect || !targetWidth || !targetHeight) return { x: 0, y: 0 };

      return mapClientToViewport(clientX, clientY, canvasRect, targetWidth, targetHeight);
    },
    [viewport?.height, viewport?.width]
  );

  const handlePointer = useCallback(
    (
      action: PointerAction,
      e: React.PointerEvent<HTMLElement>,
      canvasRect: Rect | null,
      hasFrame: boolean
    ) => {
      if (!hasFrame) return;

      if (action === 'move') {
        const now = performance.now();
        if (now - lastMoveRef.current < 100) return; // throttle move (100ms)
        lastMoveRef.current = now;
      }

      const point = getScaledPoint(e.clientX, e.clientY, canvasRect);
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
    [getScaledPoint, sendInput]
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
