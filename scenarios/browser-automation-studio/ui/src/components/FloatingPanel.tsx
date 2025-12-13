/**
 * FloatingPanel Component
 *
 * A reusable draggable, floating panel that can optionally auto-hide when
 * the mouse moves away. Used for the action bar and mini preview in Record mode.
 *
 * Features:
 * - Draggable via pointer events (mouse and touch)
 * - Optional auto-hide behavior with edge anchoring
 * - Position persistence via localStorage
 * - Boundary constraints to keep panel on-screen
 * - Smooth CSS transitions for show/hide
 */

import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type PointerEvent as ReactPointerEvent,
  type ReactNode,
  type CSSProperties,
} from 'react';

/** Position anchor when auto-hidden */
type EdgeAnchor = 'top' | 'bottom' | 'left' | 'right';

interface Position {
  x: number;
  y: number;
}

interface FloatingPanelProps {
  /** Unique ID for position persistence */
  id: string;
  /** Panel content */
  children: ReactNode;
  /** Default position (used if no stored position) */
  defaultPosition?: Position;
  /** Default edge anchor when auto-hidden (default: closest edge) */
  defaultAnchor?: EdgeAnchor;
  /** Enable auto-hide behavior */
  autoHide?: boolean;
  /** Pixels to keep visible when hidden (default: 8) */
  hiddenVisiblePx?: number;
  /** Distance from panel to trigger show (default: 50px) */
  showProximityPx?: number;
  /** Delay before showing when hovering near (default: 200ms) */
  showDelayMs?: number;
  /** Z-index for stacking (default: 50) */
  zIndex?: number;
  /** Additional class names */
  className?: string;
  /** Minimum margin from viewport edges (default: 12) */
  edgeMargin?: number;
  /** Whether the panel is visible (for external control) */
  visible?: boolean;
  /** Callback when panel visibility changes */
  onVisibilityChange?: (visible: boolean) => void;
}

const STORAGE_PREFIX = 'floating-panel:';

/** Load position from localStorage, clamping to viewport bounds */
function loadPosition(id: string, defaultPos: Position, edgeMargin = 12): Position {
  try {
    const stored = localStorage.getItem(`${STORAGE_PREFIX}${id}`);
    if (stored) {
      const parsed = JSON.parse(stored) as Position;
      if (typeof parsed.x === 'number' && typeof parsed.y === 'number') {
        // Clamp to current viewport bounds (in case screen size changed)
        const viewportWidth = typeof window !== 'undefined' ? window.innerWidth : 1200;
        const viewportHeight = typeof window !== 'undefined' ? window.innerHeight : 800;
        // Estimate panel size - use reasonable defaults
        const estimatedWidth = 320;
        const estimatedHeight = 200;
        return {
          x: Math.min(Math.max(parsed.x, edgeMargin), viewportWidth - estimatedWidth - edgeMargin),
          y: Math.min(Math.max(parsed.y, edgeMargin), viewportHeight - estimatedHeight - edgeMargin),
        };
      }
    }
  } catch {
    // localStorage unavailable or invalid
  }
  return defaultPos;
}

/** Save position to localStorage */
function savePosition(id: string, pos: Position): void {
  try {
    localStorage.setItem(`${STORAGE_PREFIX}${id}`, JSON.stringify(pos));
  } catch {
    // localStorage unavailable
  }
}

/** Calculate closest edge from a position */
function getClosestEdge(pos: Position, width: number, height: number): EdgeAnchor {
  const distTop = pos.y;
  const distBottom = window.innerHeight - (pos.y + height);
  const distLeft = pos.x;
  const distRight = window.innerWidth - (pos.x + width);

  const minDist = Math.min(distTop, distBottom, distLeft, distRight);

  if (minDist === distTop) return 'top';
  if (minDist === distBottom) return 'bottom';
  if (minDist === distLeft) return 'left';
  return 'right';
}

/** Calculate hidden position based on edge anchor */
function getHiddenPosition(
  pos: Position,
  width: number,
  height: number,
  edge: EdgeAnchor,
  visiblePx: number
): Position {
  switch (edge) {
    case 'top':
      return { x: pos.x, y: -height + visiblePx };
    case 'bottom':
      return { x: pos.x, y: window.innerHeight - visiblePx };
    case 'left':
      return { x: -width + visiblePx, y: pos.y };
    case 'right':
      return { x: window.innerWidth - visiblePx, y: pos.y };
  }
}

export function FloatingPanel({
  id,
  children,
  defaultPosition = { x: 0, y: 0 },
  defaultAnchor,
  autoHide = false,
  hiddenVisiblePx = 8,
  showProximityPx = 50,
  showDelayMs = 200,
  zIndex = 50,
  className = '',
  edgeMargin = 12,
  visible: externalVisible,
  onVisibilityChange,
}: FloatingPanelProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const [position, setPosition] = useState<Position>(() => loadPosition(id, defaultPosition));
  const [isDragging, setIsDragging] = useState(false);
  const [isHidden, setIsHidden] = useState(false);
  const [currentAnchor, setCurrentAnchor] = useState<EdgeAnchor | null>(null);
  const dragOffset = useRef<Position>({ x: 0, y: 0 });
  const showTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const savedPositionRef = useRef<Position>(position);

  // Use external visibility control if provided (currently for future use)
  const _isVisible = externalVisible !== undefined ? externalVisible : !isHidden;
  void _isVisible; // Suppress unused warning - will be used for future visibility features

  // Update position on window resize to keep panel on-screen
  useEffect(() => {
    const updateBounds = () => {
      const container = containerRef.current;
      if (!container || isDragging) return;

      const width = container.offsetWidth || 200;
      const height = container.offsetHeight || 50;
      const maxX = window.innerWidth - width - edgeMargin;
      const maxY = window.innerHeight - height - edgeMargin;

      setPosition((prev) => ({
        x: Math.min(Math.max(prev.x, edgeMargin), maxX),
        y: Math.min(Math.max(prev.y, edgeMargin), maxY),
      }));
    };

    updateBounds();
    window.addEventListener('resize', updateBounds);
    return () => window.removeEventListener('resize', updateBounds);
  }, [edgeMargin, isDragging]);

  // Handle dragging
  useEffect(() => {
    if (!isDragging) return;

    const handlePointerMove = (event: PointerEvent) => {
      const container = containerRef.current;
      if (!container) return;

      const width = container.offsetWidth || 200;
      const height = container.offsetHeight || 50;
      const maxX = window.innerWidth - width - edgeMargin;
      const maxY = window.innerHeight - height - edgeMargin;

      const newPos = {
        x: Math.min(Math.max(event.clientX - dragOffset.current.x, edgeMargin), maxX),
        y: Math.min(Math.max(event.clientY - dragOffset.current.y, edgeMargin), maxY),
      };

      setPosition(newPos);
      savedPositionRef.current = newPos;
    };

    const handlePointerUp = () => {
      setIsDragging(false);
      document.body.style.userSelect = '';
      savePosition(id, savedPositionRef.current);
    };

    window.addEventListener('pointermove', handlePointerMove);
    window.addEventListener('pointerup', handlePointerUp);
    window.addEventListener('pointercancel', handlePointerUp);
    document.body.style.userSelect = 'none';

    return () => {
      window.removeEventListener('pointermove', handlePointerMove);
      window.removeEventListener('pointerup', handlePointerUp);
      window.removeEventListener('pointercancel', handlePointerUp);
      document.body.style.userSelect = '';
    };
  }, [isDragging, id, edgeMargin]);

  // Handle auto-hide mouse tracking
  useEffect(() => {
    if (!autoHide || externalVisible !== undefined) return;

    const handleMouseMove = (event: MouseEvent) => {
      const container = containerRef.current;
      if (!container || isDragging) return;

      const rect = container.getBoundingClientRect();
      const width = rect.width;
      const height = rect.height;

      // Calculate distance to panel
      const distX = Math.max(0, rect.left - event.clientX, event.clientX - rect.right);
      const distY = Math.max(0, rect.top - event.clientY, event.clientY - rect.bottom);
      const distance = Math.sqrt(distX * distX + distY * distY);

      // If mouse is over or near the panel
      if (distance <= showProximityPx) {
        // Clear any pending hide
        if (showTimeoutRef.current) {
          clearTimeout(showTimeoutRef.current);
          showTimeoutRef.current = null;
        }

        // Show if hidden (with delay for proximity, immediate for hover)
        if (isHidden) {
          const isHovering = distance === 0;
          if (isHovering) {
            setIsHidden(false);
            setPosition(savedPositionRef.current);
            onVisibilityChange?.(true);
          } else {
            showTimeoutRef.current = setTimeout(() => {
              setIsHidden(false);
              setPosition(savedPositionRef.current);
              onVisibilityChange?.(true);
            }, showDelayMs);
          }
        }
      } else if (!isHidden) {
        // Mouse moved away - start hide timeout
        if (!showTimeoutRef.current) {
          showTimeoutRef.current = setTimeout(() => {
            const anchor = defaultAnchor || getClosestEdge(savedPositionRef.current, width, height);
            setCurrentAnchor(anchor);
            setIsHidden(true);
            setPosition(getHiddenPosition(savedPositionRef.current, width, height, anchor, hiddenVisiblePx));
            onVisibilityChange?.(false);
            showTimeoutRef.current = null;
          }, 300); // Small delay before hiding
        }
      }
    };

    window.addEventListener('mousemove', handleMouseMove);
    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
      if (showTimeoutRef.current) {
        clearTimeout(showTimeoutRef.current);
      }
    };
  }, [autoHide, isHidden, isDragging, defaultAnchor, hiddenVisiblePx, showProximityPx, showDelayMs, onVisibilityChange, externalVisible]);

  const handlePointerDown = useCallback(
    (event: ReactPointerEvent) => {
      // Only handle left mouse button or touch
      if (event.pointerType !== 'touch' && event.button !== 0) return;

      // Don't start drag if clicking on a button or interactive element
      const target = event.target as HTMLElement;
      if (target.closest('button, input, select, textarea, a, [role="button"]')) return;

      const container = containerRef.current;
      if (!container) return;

      container.setPointerCapture(event.pointerId);

      const rect = container.getBoundingClientRect();
      dragOffset.current = {
        x: event.clientX - rect.left,
        y: event.clientY - rect.top,
      };

      // If panel was hidden, restore position first
      if (isHidden) {
        setIsHidden(false);
        setPosition(savedPositionRef.current);
        onVisibilityChange?.(true);
      }

      setIsDragging(true);
      event.preventDefault();
    },
    [isHidden, onVisibilityChange]
  );

  // Calculate display position (use saved position for transform when hidden)
  const displayPosition = isHidden && autoHide && currentAnchor
    ? getHiddenPosition(savedPositionRef.current, containerRef.current?.offsetWidth || 200, containerRef.current?.offsetHeight || 50, currentAnchor, hiddenVisiblePx)
    : position;

  const style: CSSProperties = {
    position: 'fixed',
    left: displayPosition.x,
    top: displayPosition.y,
    zIndex,
    transition: isDragging ? 'none' : 'left 0.2s ease-out, top 0.2s ease-out, opacity 0.2s ease-out',
    opacity: isHidden ? 0.6 : 1,
  };

  return (
    <div
      ref={containerRef}
      className={`touch-none ${isDragging ? 'cursor-grabbing' : 'cursor-grab'} ${className}`}
      style={style}
      onPointerDown={handlePointerDown}
    >
      {children}
    </div>
  );
}
