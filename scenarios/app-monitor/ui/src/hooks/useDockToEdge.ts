import { useState, useCallback, useEffect, useMemo, useRef } from 'react';
import type { CSSProperties, MutableRefObject } from 'react';
import type { DragEndInfo } from './useDraggablePosition';

/** Minimum fling speed (px/s) to trigger a dock */
const FLING_VELOCITY_THRESHOLD = 400;
/** How close to an edge (px) a release must be to dock without fling */
const EDGE_PROXIMITY_THRESHOLD = 40;
/** Width of the visible tab when docked (px) */
export const DOCK_TAB_WIDTH = 24;
/** Duration of dock/undock animation (ms) — keep in sync with CSS transition */
const DOCK_ANIMATION_MS = 300;

const DOCK_STORAGE_KEY = 'am-toolbar-dock';

export type DockedEdge = 'left' | 'right' | null;

function loadDockedEdge(): DockedEdge {
  if (typeof window === 'undefined') {
    return null;
  }
  try {
    const raw = localStorage.getItem(DOCK_STORAGE_KEY);
    if (raw === 'left' || raw === 'right') {
      return raw;
    }
  } catch { /* ignore */ }
  return null;
}

function saveDockedEdge(edge: DockedEdge) {
  if (typeof window === 'undefined') {
    return;
  }
  try {
    if (edge) {
      localStorage.setItem(DOCK_STORAGE_KEY, edge);
    } else {
      localStorage.removeItem(DOCK_STORAGE_KEY);
    }
  } catch { /* ignore */ }
}

export interface UseDockToEdgeOptions {
  /** Element ref for measuring toolbar width */
  elementRef: MutableRefObject<HTMLElement | null>;
  /** Only active in fullscreen/full-view mode */
  isActive: boolean;
}

export interface UseDockToEdgeReturn {
  /** Which edge the toolbar is docked to, or null if floating */
  docked: DockedEdge;
  /** Whether a dock/undock animation is in progress */
  animating: boolean;
  /** Measured toolbar width (for computing dock offset externally) */
  toolbarWidth: number;
  /** Current viewport width (for right-dock positioning) */
  vpWidth: number;
  /** Wire to useDraggablePosition's onDragEnd */
  handleDragEnd: (info: DragEndInfo) => void;
  /** Wire to useDraggablePosition's onDragStart — undocks immediately on drag */
  handleDragStart: () => void;
  /** Click handler to undock with animation */
  undock: () => void;
}

/**
 * Compute the CSS style for a docked or floating toolbar.
 * Call this in the component where both dock state and floating position are available.
 */
export function computeDockStyle(
  docked: DockedEdge,
  floatingStyle: CSSProperties | undefined,
  positionY: number,
  toolbarWidth: number,
  vpWidth: number,
): CSSProperties | undefined {
  if (!docked) {
    return floatingStyle;
  }
  if (docked === 'left') {
    return { transform: `translate3d(${-(toolbarWidth - DOCK_TAB_WIDTH)}px, ${positionY}px, 0)` };
  }
  return { transform: `translate3d(${vpWidth - DOCK_TAB_WIDTH}px, ${positionY}px, 0)` };
}

export const useDockToEdge = ({
  elementRef,
  isActive,
}: UseDockToEdgeOptions): UseDockToEdgeReturn => {
  const [docked, setDocked] = useState<DockedEdge>(() => (isActive ? loadDockedEdge() : null));
  const [animating, setAnimating] = useState(false);
  /** Measured full width of toolbar, for computing dock offset */
  const [toolbarWidth, setToolbarWidth] = useState(160);
  /** Viewport width for right-dock positioning */
  const [vpWidth, setVpWidth] = useState(() =>
    typeof window !== 'undefined' ? window.innerWidth : 1000,
  );

  const dockedRef = useRef(docked);
  dockedRef.current = docked;

  // Track viewport width for right-dock transform
  useEffect(() => {
    if (typeof window === 'undefined') {
      return;
    }
    const onResize = () => setVpWidth(window.innerWidth);
    window.addEventListener('resize', onResize);
    return () => window.removeEventListener('resize', onResize);
  }, []);

  // Measure toolbar width so we know how far to slide off-screen
  useEffect(() => {
    const el = elementRef.current;
    if (!el) {
      return;
    }
    const rect = el.getBoundingClientRect();
    if (rect.width > 0) {
      setToolbarWidth(rect.width);
    }
  }, [docked, elementRef]);

  // When becoming inactive, clear docked state
  useEffect(() => {
    if (!isActive) {
      setDocked(null);
      setAnimating(false);
    } else {
      // Restore persisted dock state on activation
      const persisted = loadDockedEdge();
      if (persisted) {
        setDocked(persisted);
      }
    }
  }, [isActive]);

  const handleDragEnd = useCallback((info: DragEndInfo) => {
    const { position: pos, velocity, elementSize } = info;
    const vw = typeof window !== 'undefined' ? window.innerWidth : 1000;

    const flungLeft = velocity.vx < -FLING_VELOCITY_THRESHOLD;
    const flungRight = velocity.vx > FLING_VELOCITY_THRESHOLD;
    const nearLeft = pos.x < EDGE_PROXIMITY_THRESHOLD;
    const nearRight = pos.x + elementSize.width > vw - EDGE_PROXIMITY_THRESHOLD;

    let edge: DockedEdge = null;
    if (flungLeft || nearLeft) {
      edge = 'left';
    } else if (flungRight || nearRight) {
      edge = 'right';
    }

    if (edge) {
      setAnimating(true);
      setDocked(edge);
      saveDockedEdge(edge);
      setTimeout(() => setAnimating(false), DOCK_ANIMATION_MS);
    }
  }, []);

  const undock = useCallback(() => {
    setAnimating(true);
    setDocked(null);
    saveDockedEdge(null);
    setTimeout(() => setAnimating(false), DOCK_ANIMATION_MS);
  }, []);

  const handleDragStart = useCallback(() => {
    if (dockedRef.current) {
      // Immediately undock without animation — user is dragging
      setDocked(null);
      saveDockedEdge(null);
    }
  }, []);

  return useMemo(() => ({
    docked,
    animating,
    toolbarWidth,
    vpWidth,
    handleDragEnd,
    handleDragStart,
    undock,
  }), [docked, animating, toolbarWidth, vpWidth, handleDragEnd, handleDragStart, undock]);
};
