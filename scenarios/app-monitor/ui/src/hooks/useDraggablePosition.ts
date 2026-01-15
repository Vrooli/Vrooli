import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import type {
  CSSProperties,
  MouseEvent as ReactMouseEvent,
  PointerEvent as ReactPointerEvent,
  RefObject,
} from 'react';
import { useFloatingPosition } from './useFloatingPosition';
import { logger } from '@/services/logger';

const DEFAULT_DRAG_THRESHOLD = 6;
const DEFAULT_FLOATING_MARGIN = 12;

type DragState = {
  pointerId: number;
  startX: number;
  startY: number;
  offsetX: number;
  offsetY: number;
  width: number;
  height: number;
  pointerCaptured: boolean;
  dragging: boolean;
};

interface StoredPosition {
  x: number;
  y: number;
  savedAt: number;
}

export interface UseDraggablePositionOptions {
  /** Whether dragging is currently active (e.g., fullscreen mode) */
  isActive: boolean;
  /** localStorage key for persisting position. null = no persistence */
  storageKey?: string | null;
  /** Default position when not persisted or on first load */
  defaultPosition: { x: number; y: number } | (() => { x: number; y: number } | null);
  /** Margin from viewport edges in pixels */
  floatingMargin?: number;
  /** Minimum pointer movement before drag activates */
  dragThreshold?: number;
  /** Called when drag starts (e.g., to close menus) */
  onDragStart?: () => void;
  /** Called when drag ends */
  onDragEnd?: () => void;
}

export interface UseDraggablePositionReturn {
  elementRef: RefObject<HTMLElement | null>;
  position: { x: number; y: number };
  isDragging: boolean;
  floatingStyle: CSSProperties | undefined;
  pointerHandlers: {
    onPointerDown: (e: ReactPointerEvent) => void;
    onPointerMove: (e: ReactPointerEvent) => void;
    onPointerUp: (e: ReactPointerEvent) => void;
    onPointerCancel: (e: ReactPointerEvent) => void;
  };
  handleClickCapture: (e: ReactMouseEvent) => void;
  resetPosition: () => void;
}

const getPointerDelta = (state: DragState, event: PointerEvent | ReactPointerEvent) => ({
  deltaX: event.clientX - state.startX,
  deltaY: event.clientY - state.startY,
});

const loadStoredPosition = (storageKey: string): StoredPosition | null => {
  if (typeof window === 'undefined') {
    return null;
  }
  try {
    const stored = localStorage.getItem(storageKey);
    if (!stored) {
      return null;
    }
    const parsed = JSON.parse(stored) as unknown;
    if (
      typeof parsed === 'object' &&
      parsed !== null &&
      'x' in parsed &&
      'y' in parsed &&
      typeof (parsed as StoredPosition).x === 'number' &&
      typeof (parsed as StoredPosition).y === 'number'
    ) {
      return parsed as StoredPosition;
    }
  } catch (error) {
    logger.warn('Failed to load stored position', { storageKey, error });
  }
  return null;
};

const saveStoredPosition = (storageKey: string, position: { x: number; y: number }) => {
  if (typeof window === 'undefined') {
    return;
  }
  try {
    const data: StoredPosition = {
      x: position.x,
      y: position.y,
      savedAt: Date.now(),
    };
    localStorage.setItem(storageKey, JSON.stringify(data));
  } catch (error) {
    logger.warn('Failed to save stored position', { storageKey, error });
  }
};

/**
 * Hook for managing draggable floating element positioning with optional localStorage persistence.
 * Extracted from AppPreviewToolbar's useFloatingToolbar for reuse.
 */
export const useDraggablePosition = (options: UseDraggablePositionOptions): UseDraggablePositionReturn => {
  const {
    isActive,
    storageKey = null,
    defaultPosition,
    floatingMargin = DEFAULT_FLOATING_MARGIN,
    dragThreshold = DEFAULT_DRAG_THRESHOLD,
    onDragStart,
    onDragEnd,
  } = options;

  const { clampPosition } = useFloatingPosition({ floatingMargin });

  const elementRef = useRef<HTMLElement | null>(null);
  const dragStateRef = useRef<DragState | null>(null);
  const suppressClickRef = useRef(false);

  // Compute initial position from storage or default
  const getInitialPosition = useCallback((): { x: number; y: number } => {
    // Try loading from storage first
    if (storageKey) {
      const stored = loadStoredPosition(storageKey);
      if (stored) {
        return { x: stored.x, y: stored.y };
      }
    }
    // Fall back to default
    if (typeof defaultPosition === 'function') {
      return defaultPosition() ?? { x: floatingMargin, y: floatingMargin };
    }
    return defaultPosition;
  }, [defaultPosition, floatingMargin, storageKey]);

  const [position, setPosition] = useState<{ x: number; y: number }>(getInitialPosition);
  const [isDragging, setIsDragging] = useState(false);
  const [isTrackingPointer, setIsTrackingPointer] = useState(false);

  const releasePointerCapture = useCallback((pointerId: number) => {
    const element = elementRef.current;
    if (!element) {
      return;
    }

    if (typeof element.hasPointerCapture === 'function' && !element.hasPointerCapture(pointerId)) {
      return;
    }

    try {
      element.releasePointerCapture(pointerId);
    } catch {
      // noop - releasePointerCapture may throw if pointer already released
    }
  }, []);

  // Reset position and state when becoming inactive
  useEffect(() => {
    if (!isActive) {
      if (dragStateRef.current?.pointerCaptured) {
        releasePointerCapture(dragStateRef.current.pointerId);
      }
      // Don't reset position - keep last known position for when it becomes active again
      setIsDragging(false);
      setIsTrackingPointer(false);
      dragStateRef.current = null;
      return;
    }

    // When becoming active, recalculate position from storage or compute default
    const initialPos = getInitialPosition();
    const element = elementRef.current;
    if (element) {
      const rect = element.getBoundingClientRect();
      const clamped = clampPosition(initialPos.x, initialPos.y, { width: rect.width, height: rect.height });
      setPosition(clamped);
    } else {
      setPosition(initialPos);
    }
  }, [clampPosition, getInitialPosition, isActive, releasePointerCapture]);

  // Handle window resize - clamp position to new bounds
  useEffect(() => {
    if (typeof window === 'undefined' || !isActive) {
      return;
    }

    const handleResize = () => {
      const element = elementRef.current;
      if (!element) {
        return;
      }
      const rect = element.getBoundingClientRect();
      setPosition(prev => {
        const next = clampPosition(prev.x, prev.y, { width: rect.width, height: rect.height });
        if (next.x === prev.x && next.y === prev.y) {
          return prev;
        }
        return next;
      });
    };

    handleResize();
    window.addEventListener('resize', handleResize);
    window.addEventListener('orientationchange', handleResize);

    return () => {
      window.removeEventListener('resize', handleResize);
      window.removeEventListener('orientationchange', handleResize);
    };
  }, [clampPosition, isActive]);

  // Persist position to localStorage when it changes
  useEffect(() => {
    if (!storageKey || !isActive) {
      return;
    }
    saveStoredPosition(storageKey, position);
  }, [isActive, position, storageKey]);

  const handlePointerDown = useCallback((event: ReactPointerEvent) => {
    if (!isActive) {
      return;
    }

    if (event.pointerType === 'mouse' && event.button !== 0) {
      return;
    }

    const element = elementRef.current;
    if (!element) {
      return;
    }

    const rect = element.getBoundingClientRect();
    dragStateRef.current = {
      pointerId: event.pointerId,
      startX: event.clientX,
      startY: event.clientY,
      offsetX: event.clientX - rect.left,
      offsetY: event.clientY - rect.top,
      width: rect.width,
      height: rect.height,
      pointerCaptured: false,
      dragging: false,
    };
    setIsDragging(false);
    setIsTrackingPointer(true);
  }, [isActive]);

  const processPointerMove = useCallback((event: PointerEvent | ReactPointerEvent) => {
    const state = dragStateRef.current;
    if (!state || state.pointerId !== event.pointerId) {
      return;
    }

    const element = elementRef.current;
    if (!element) {
      return;
    }

    const { deltaX, deltaY } = getPointerDelta(state, event);
    if (!state.dragging) {
      if (Math.abs(deltaX) + Math.abs(deltaY) < dragThreshold) {
        return;
      }
      state.dragging = true;
      setIsDragging(true);
      onDragStart?.();
      if (!state.pointerCaptured) {
        try {
          element.setPointerCapture(event.pointerId);
          state.pointerCaptured = true;
        } catch {
          state.pointerCaptured = false;
        }
      }
    }

    if (!state.dragging) {
      return;
    }

    event.preventDefault?.();
    const next = clampPosition(
      event.clientX - state.offsetX,
      event.clientY - state.offsetY,
      { width: state.width, height: state.height },
    );
    setPosition(prev => (prev.x === next.x && prev.y === next.y ? prev : next));
  }, [clampPosition, dragThreshold, onDragStart]);

  const handlePointerMove = useCallback((event: ReactPointerEvent) => {
    processPointerMove(event);
  }, [processPointerMove]);

  const processPointerEnd = useCallback((event: PointerEvent | ReactPointerEvent) => {
    const state = dragStateRef.current;
    if (!state || state.pointerId !== event.pointerId) {
      return;
    }

    if (state.pointerCaptured) {
      releasePointerCapture(event.pointerId);
    }

    if (state.dragging) {
      event.preventDefault?.();
      suppressClickRef.current = true;
      onDragEnd?.();
      if (typeof window !== 'undefined') {
        window.setTimeout(() => {
          suppressClickRef.current = false;
        }, 0);
      } else {
        suppressClickRef.current = false;
      }
    }

    dragStateRef.current = null;
    setIsDragging(false);
    setIsTrackingPointer(false);
  }, [onDragEnd, releasePointerCapture]);

  const handlePointerEnd = useCallback((event: ReactPointerEvent) => {
    processPointerEnd(event);
  }, [processPointerEnd]);

  // Window-level event listeners for tracking pointer outside element
  useEffect(() => {
    if (typeof window === 'undefined' || !isTrackingPointer) {
      return;
    }

    const handleWindowPointerMove = (event: PointerEvent) => processPointerMove(event);
    const handleWindowPointerUp = (event: PointerEvent) => processPointerEnd(event);
    const handleWindowPointerCancel = (event: PointerEvent) => processPointerEnd(event);

    window.addEventListener('pointermove', handleWindowPointerMove, { passive: false });
    window.addEventListener('pointerup', handleWindowPointerUp, { passive: false });
    window.addEventListener('pointercancel', handleWindowPointerCancel, { passive: false });

    return () => {
      window.removeEventListener('pointermove', handleWindowPointerMove);
      window.removeEventListener('pointerup', handleWindowPointerUp);
      window.removeEventListener('pointercancel', handleWindowPointerCancel);
    };
  }, [isTrackingPointer, processPointerEnd, processPointerMove]);

  const handleClickCapture = useCallback((event: ReactMouseEvent) => {
    if (suppressClickRef.current) {
      event.preventDefault();
      event.stopPropagation();
      suppressClickRef.current = false;
    }
  }, []);

  const floatingStyle = useMemo<CSSProperties | undefined>(() => {
    if (!isActive) {
      return undefined;
    }
    return {
      transform: `translate3d(${Math.round(position.x)}px, ${Math.round(position.y)}px, 0)`,
    };
  }, [isActive, position]);

  const resetPosition = useCallback(() => {
    const initial = getInitialPosition();
    setPosition(initial);
    if (storageKey) {
      try {
        localStorage.removeItem(storageKey);
      } catch {
        // noop
      }
    }
  }, [getInitialPosition, storageKey]);

  return useMemo(() => ({
    elementRef,
    position,
    isDragging,
    floatingStyle,
    pointerHandlers: {
      onPointerDown: handlePointerDown,
      onPointerMove: handlePointerMove,
      onPointerUp: handlePointerEnd,
      onPointerCancel: handlePointerEnd,
    },
    handleClickCapture,
    resetPosition,
  }), [
    position,
    isDragging,
    floatingStyle,
    handlePointerDown,
    handlePointerMove,
    handlePointerEnd,
    handleClickCapture,
    resetPosition,
  ]);
};
