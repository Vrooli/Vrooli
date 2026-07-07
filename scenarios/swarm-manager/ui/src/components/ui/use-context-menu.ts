/**
 * useContextMenu — open-state and right-click / long-press triggers for a
 * pointer-anchored context menu. Pair with the <ContextMenu> component, which
 * renders the captured origin through the shared Popover surface.
 */

import { useCallback, useRef, useState, type TouchEvent, type MouseEvent } from "react";

/** Delay before a sustained touch is treated as a long-press. */
const LONG_PRESS_MS = 450;
/** Movement (px) that cancels a pending long-press (treated as a scroll). */
const LONG_PRESS_MOVE_TOLERANCE_PX = 10;

export interface ContextMenuOrigin {
  x: number;
  y: number;
}

export interface ContextMenuTriggerProps {
  onContextMenu: (event: MouseEvent) => void;
  onTouchStart: (event: TouchEvent) => void;
  onTouchMove: (event: TouchEvent) => void;
  onTouchEnd: () => void;
  onTouchCancel: () => void;
}

export interface UseContextMenuResult {
  origin: ContextMenuOrigin | null;
  isOpen: boolean;
  openAt: (x: number, y: number) => void;
  close: () => void;
  /** Spread onto the element that should open the menu. */
  triggerProps: ContextMenuTriggerProps;
}

/**
 * Manages context-menu open state and the right-click / long-press triggers.
 */
export function useContextMenu(): UseContextMenuResult {
  const [origin, setOrigin] = useState<ContextMenuOrigin | null>(null);
  const timer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const startPoint = useRef<ContextMenuOrigin | null>(null);

  const openAt = useCallback((x: number, y: number) => setOrigin({ x, y }), []);
  const close = useCallback(() => setOrigin(null), []);

  const clearTimer = useCallback(() => {
    if (timer.current !== undefined) {
      clearTimeout(timer.current);
      timer.current = undefined;
    }
    startPoint.current = null;
  }, []);

  const onContextMenu = useCallback((event: MouseEvent) => {
    event.preventDefault();
    setOrigin({ x: event.clientX, y: event.clientY });
  }, []);

  const onTouchStart = useCallback(
    (event: TouchEvent) => {
      const touch = event.touches[0];
      if (!touch) return;
      const x = touch.clientX;
      const y = touch.clientY;
      startPoint.current = { x, y };
      clearTimer();
      timer.current = setTimeout(() => {
        setOrigin({ x, y });
        timer.current = undefined;
      }, LONG_PRESS_MS);
    },
    [clearTimer],
  );

  const onTouchMove = useCallback((event: TouchEvent) => {
    const touch = event.touches[0];
    const start = startPoint.current;
    if (!touch || !start) return;
    const moved =
      Math.abs(touch.clientX - start.x) > LONG_PRESS_MOVE_TOLERANCE_PX ||
      Math.abs(touch.clientY - start.y) > LONG_PRESS_MOVE_TOLERANCE_PX;
    if (moved && timer.current !== undefined) {
      clearTimeout(timer.current);
      timer.current = undefined;
      startPoint.current = null;
    }
  }, []);

  return {
    origin,
    isOpen: origin !== null,
    openAt,
    close,
    triggerProps: {
      onContextMenu,
      onTouchStart,
      onTouchMove,
      onTouchEnd: clearTimer,
      onTouchCancel: clearTimer,
    },
  };
}
