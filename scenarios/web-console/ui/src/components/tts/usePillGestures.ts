import { useCallback, useRef } from "react";
import type { MouseEvent as ReactMouseEvent, PointerEvent as ReactPointerEvent, RefObject } from "react";
import { PRESS_MOVE_THRESHOLD_PX, movedBeyond } from "../../hooks/usePressGesture";

/** A drag at least this far down dismisses the pill. */
const DISMISS_DRAG_PX = 48;
/** A horizontal drag at least this far changes track. */
const TRACK_SWIPE_PX = 64;
/** Pointer-downs on these keep their own click; they never count as a pill tap. */
const CONTROL_SELECTOR = "button, input, select, textarea, a, [role='slider']";

export interface UsePillGesturesOptions {
  onTap: () => void;
  onDismiss: () => void;
  onNext: () => void;
  onPrevious: () => void;
  /** Called with a 0..1 position while a drag runs along the progress zone. */
  onScrub: (ratio: number) => void;
  progressRef: RefObject<HTMLElement | null>;
  /** Horizontal drags change track; off while the pill is expanded. */
  swipeTracks: boolean;
}

interface Drag {
  pointerId: number;
  start: { x: number; y: number };
  /** Fixed by the first movement past the tap threshold, for the whole drag. */
  axis: "x" | "y" | null;
  scrub: boolean;
  onControl: boolean;
}

function ratioAlong(zone: HTMLElement, clientX: number): number {
  const box = zone.getBoundingClientRect();
  if (box.width <= 0) return 0;
  return Math.min(1, Math.max(0, (clientX - box.left) / box.width));
}

/**
 * Pointer gestures for the playback pill: tap, drag down to dismiss, drag
 * sideways to change track, drag along the progress zone to scrub. The pill
 * sets `touch-action: none` on itself, so these never fight the list's scroll.
 */
export function usePillGestures(options: UsePillGesturesOptions) {
  const optionsRef = useRef(options);
  optionsRef.current = options;
  const dragRef = useRef<Drag | null>(null);
  const suppressClickRef = useRef(false);

  const onPointerDown = useCallback((event: ReactPointerEvent<HTMLElement>) => {
    if (event.button !== 0) return;
    const target = event.target as HTMLElement;
    const zone = optionsRef.current.progressRef.current;
    dragRef.current = {
      pointerId: event.pointerId,
      start: { x: event.clientX, y: event.clientY },
      axis: null,
      scrub: zone?.contains(target) ?? false,
      onControl: target.closest(CONTROL_SELECTOR) !== null,
    };
    suppressClickRef.current = false;
  }, []);

  const onPointerMove = useCallback((event: ReactPointerEvent<HTMLElement>) => {
    const drag = dragRef.current;
    if (!drag || event.pointerId !== drag.pointerId) return;
    const point = { x: event.clientX, y: event.clientY };
    if (drag.axis === null) {
      if (!movedBeyond(drag.start, point, PRESS_MOVE_THRESHOLD_PX)) return;
      drag.axis = Math.abs(point.y - drag.start.y) > Math.abs(point.x - drag.start.x) ? "y" : "x";
    }
    const zone = optionsRef.current.progressRef.current;
    if (drag.scrub && drag.axis === "x" && zone) optionsRef.current.onScrub(ratioAlong(zone, point.x));
  }, []);

  const onPointerUp = useCallback((event: ReactPointerEvent<HTMLElement>) => {
    const drag = dragRef.current;
    if (!drag || event.pointerId !== drag.pointerId) return;
    dragRef.current = null;
    const current = optionsRef.current;
    if (drag.axis === null) {
      if (!drag.onControl) current.onTap();
      return;
    }
    suppressClickRef.current = true;
    const dx = event.clientX - drag.start.x;
    const dy = event.clientY - drag.start.y;
    const zone = current.progressRef.current;
    if (drag.scrub && drag.axis === "x" && zone) {
      current.onScrub(ratioAlong(zone, event.clientX));
    } else if (drag.axis === "y") {
      if (dy >= DISMISS_DRAG_PX) current.onDismiss();
    } else if (current.swipeTracks && Math.abs(dx) >= TRACK_SWIPE_PX) {
      if (dx < 0) current.onNext();
      else current.onPrevious();
    }
  }, []);

  const onPointerCancel = useCallback(() => {
    dragRef.current = null;
  }, []);

  // A drag that ends over a control must not also press it.
  const onClickCapture = useCallback((event: ReactMouseEvent<HTMLElement>) => {
    if (!suppressClickRef.current) return;
    suppressClickRef.current = false;
    event.preventDefault();
    event.stopPropagation();
  }, []);

  return { onPointerDown, onPointerMove, onPointerUp, onPointerCancel, onClickCapture };
}
