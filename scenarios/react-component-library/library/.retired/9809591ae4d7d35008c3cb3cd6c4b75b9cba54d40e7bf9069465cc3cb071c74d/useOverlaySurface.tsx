/**
 * @libraryId react-component-library:useOverlaySurface
 * @displayName useOverlaySurface
 * @description Composes the shared lifecycle, focus, dismissal, portal, and motion contract for overlays.
 * @version 1.3.9
 * @tags ["runtime","overlay","accessibility"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
import {
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
  type PointerEvent as ReactPointerEvent,
  type RefObject,
} from "react";
import { layerManager } from "@vrooli/react-component-library/LayerManager/2.0.0";
import { useControllableState } from "@vrooli/react-component-library/useControllableState/1.0.0";
import { useEscapeKey } from "@vrooli/react-component-library/useEscapeKey/1.0.0";
import { useFocusReturn } from "@vrooli/react-component-library/useFocusReturn/1.1.0";
import { useFocusTrap } from "@vrooli/react-component-library/useFocusTrap/1.1.0";
import { useReducedMotion } from "@vrooli/react-component-library/useReducedMotion/1.0.0";
import { useScrollLock } from "@vrooli/react-component-library/useScrollLock/2.0.0";
import {
  useSwipe,
  type SwipeDirection,
} from "@vrooli/react-component-library/useSwipe/2.0.4";
import { useViewportEnvironmentStyle } from "@vrooli/react-component-library/useViewportEnvironment/1.0.3";
import { baseStyles } from "@vrooli/react-component-library/BaseStyles/1.2.0";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";

/** The presentation an overlay takes, which selects its layer band and role. */
export type OverlayKind =
  | "dialog"
  | "alertdialog"
  | "menu"
  | "popover"
  | "sheet"
  | "drawer";

/**
 * Which gestures dismiss this overlay.
 *
 * `swipe` names the direction a drag on the grabber must travel to dismiss.
 * It is off unless a presentation asks for it, because a surface without a
 * visible grabber must not be draggable.
 */
export interface OverlayDismissPolicy {
  escape?: boolean;
  backdrop?: boolean;
  swipe?: SwipeDirection | false;
}

/** Inputs to {@link useOverlaySurface}. */
export interface UseOverlaySurfaceOptions {
  open?: boolean;
  defaultOpen?: boolean;
  onOpenChange?: (open: boolean) => void;
  modal?: boolean;
  kind: OverlayKind;
  dismiss?: OverlayDismissPolicy;
  initialFocusRef?: RefObject<HTMLElement | null>;
  returnFocusRef?: RefObject<HTMLElement | null>;
  scrollLock?: boolean;
  exitDurationMs?: number;
  /** Distance in CSS pixels a swipe must travel before it commits. */
  swipeThreshold?: number;
}

const NOOP_POINTER = () => {};

/**
 * The drag offset is written as a `transform` on the surface's own inline
 * style, in pixels, so the surface moves exactly as far as the finger.
 *
 * Two earlier shapes were worse. Translating by a fraction of the 96px commit
 * threshold sent a 700px panel roughly seven times faster than the pointer.
 * Translating by a custom property fixed the distance but not the cost: custom
 * properties inherit, so rewriting one on the panel invalidated style
 * resolution for every descendant on every pointer move — for an overlay
 * holding a long document, that is the whole document, sixty times a second,
 * to move one box. `transform` does not inherit, so the invalidation stops at
 * the surface.
 */
function translationFor(direction: SwipeDirection, pixels: number) {
  if (direction === "left") return `translateX(${String(-pixels)}px)`;
  if (direction === "right") return `translateX(${String(pixels)}px)`;
  if (direction === "up") return `translateY(${String(-pixels)}px)`;
  return `translateY(${String(pixels)}px)`;
}

/**
 * Marks the surface and its grabber for the duration of a gesture. A
 * presentation suspends its transform transition on the surface flag so the
 * drag tracks the finger exactly, and styles the grabber's cursor on the other.
 */
const DRAGGING_ATTRIBUTE = "data-dragging";
const GRABBER_DRAGGING_ATTRIBUTE = "data-rcl-overlay-dragging";

/** Distance travelled toward the dismissing edge, in CSS pixels, never negative. */
function travelled(
  direction: SwipeDirection,
  start: { x: number; y: number },
  point: { clientX: number; clientY: number },
) {
  const dx = point.clientX - start.x;
  const dy = point.clientY - start.y;
  if (direction === "left") return Math.max(0, -dx);
  if (direction === "right") return Math.max(0, dx);
  if (direction === "up") return Math.max(0, -dy);
  return Math.max(0, dy);
}

/**
 * The shared substrate every interactive overlay composes.
 *
 * It owns portal layering, top-layer Escape dismissal, modal background
 * inertness, nested scroll locking, focus containment and return, presence
 * across the exit transition, and — from 1.3.0 — swipe dismissal. A
 * presentation component owns geometry and anatomy; it must not re-implement
 * any of the above.
 *
 * Swipe support is deliberately part of the substrate rather than each sheet:
 * dragging a surface toward its dismissing edge is the same decision as
 * pressing Escape, and it has the same relationship to the layer stack.
 */
export function useOverlaySurface({
  open: controlledOpen,
  defaultOpen = false,
  onOpenChange,
  modal = true,
  kind,
  dismiss = { escape: true, backdrop: true },
  initialFocusRef,
  returnFocusRef,
  scrollLock = modal,
  exitDurationMs = 180,
  swipeThreshold = 96,
}: UseOverlaySurfaceOptions) {
  // The sheet key names the stylesheet revision, not the asset version, so
  // every asset carrying this revision shares one head node.
  useLibraryStyleSheet("base-styles-1.2.0", baseStyles);
  const [open, setOpen] = useControllableState({
    value: controlledOpen,
    defaultValue: defaultOpen,
    onChange: onOpenChange,
  });
  const reducedMotion = useReducedMotion();
  const viewportStyle = useViewportEnvironmentStyle();
  const [present, setPresent] = useState(open);
  // A dismissing drag does no React work at all. The offset is a custom
  // property and the dragging flag is an attribute, both written straight to
  // the DOM: an overlay's content can be arbitrarily large — a long document,
  // a syntax-highlighted file — and re-rendering that subtree even twice per
  // gesture is enough to be felt at the start and end of the drag.
  const draggingRef = useRef(false);
  const surfaceRef = useRef<HTMLElement | null>(null);
  const grabberRef = useRef<Element | null>(null);
  // The surface's block size is measured once per gesture. Measuring it per
  // pointer move forced a synchronous layout of the whole panel on every
  // frame, which is the one thing a drag cannot afford; the panel cannot
  // change height mid-drag anyway.
  const dragExtent = useRef(0);
  const id = useId();

  const dragOrigin = useRef({ x: 0, y: 0 });
  const movedDuringGesture = useRef(false);
  const suppressNextClick = useRef(false);

  // `pixels` is the real distance the pointer has moved toward the dismissing
  // edge. It is divided by the surface's measured block size here rather than
  // by the commit threshold, because `100%` in the presentation's transform
  // means the surface's own height.
  const markDragging = useCallback((next: boolean) => {
    if (next === draggingRef.current) return;
    draggingRef.current = next;
    const surface = surfaceRef.current;
    const grabber = grabberRef.current;
    if (next) {
      surface?.setAttribute(DRAGGING_ATTRIBUTE, "true");
      grabber?.setAttribute(GRABBER_DRAGGING_ATTRIBUTE, "true");
      return;
    }
    surface?.removeAttribute(DRAGGING_ATTRIBUTE);
    grabber?.removeAttribute(GRABBER_DRAGGING_ATTRIBUTE);
  }, []);

  const resolvedDirectionRef = useRef<SwipeDirection>("down");

  /** Follow the pointer. Clears the offset, and the flag, at `pixels === 0`. */
  const writeOffset = useCallback(
    (pixels: number) => {
      const surface = surfaceRef.current;
      if (surface)
        surface.style.transform =
          pixels > 0
            ? translationFor(resolvedDirectionRef.current, pixels)
            : "";
      markDragging(pixels > 0);
    },
    [markDragging],
  );

  /**
   * Send the surface the rest of the way out from wherever the finger left it,
   * with the transition re-enabled, rather than snapping it back to rest first
   * and animating the closed state from there.
   */
  const writeDismissed = useCallback(() => {
    markDragging(false);
    const surface = surfaceRef.current;
    if (surface && dragExtent.current > 0)
      surface.style.transform = translationFor(
        resolvedDirectionRef.current,
        dragExtent.current,
      );
  }, [markDragging]);

  const resetGestureDom = useCallback(() => {
    writeOffset(0);
    dragOrigin.current = { x: 0, y: 0 };
    dragExtent.current = 0;
    grabberRef.current = null;
    movedDuringGesture.current = false;
  }, [writeOffset]);

  useEffect(() => {
    if (open) {
      setPresent(true);
      return;
    }
    if (reducedMotion) {
      setPresent(false);
      return;
    }
    const timer = window.setTimeout(() => setPresent(false), exitDurationMs);
    return () => window.clearTimeout(timer);
  }, [exitDurationMs, open, reducedMotion]);

  // A surface that reopens before its exit finishes keeps the same element, so
  // the offset that dismissed it has to be cleared explicitly.
  useEffect(() => {
    resetGestureDom();
  }, [open, resetGestureDom]);

  const close = useCallback(() => setOpen(false), [setOpen]);
  useEffect(() => {
    if (!open) return;
    return layerManager.push({ id, kind, modal, dismiss: close });
  }, [close, id, kind, modal, open]);
  useScrollLock(open && scrollLock);
  useFocusTrap(open && modal, surfaceRef);
  useFocusReturn(open, returnFocusRef);
  useEscapeKey(open && dismiss.escape !== false, () => {
    if (layerManager.isTop(id)) close();
  });
  useEffect(() => {
    if (!open || typeof window === "undefined") return;
    let frame = 0;
    let attemptsRemaining = 8;
    const focusInitial = () => {
      const target =
        initialFocusRef?.current ??
        surfaceRef.current?.querySelector<HTMLElement>(
          "[autofocus], button, input, select, textarea, [tabindex]:not([tabindex='-1'])",
        );
      if (target) {
        target.focus();
        return;
      }
      attemptsRemaining -= 1;
      if (attemptsRemaining > 0)
        frame = window.requestAnimationFrame(focusInitial);
    };
    frame = window.requestAnimationFrame(focusInitial);
    return () => window.cancelAnimationFrame(frame);
  }, [initialFocusRef, open]);

  const swipeDirection = dismiss.swipe === undefined ? false : dismiss.swipe;
  const swipeEnabled = swipeDirection !== false;
  const resolvedDirection: SwipeDirection = swipeEnabled
    ? swipeDirection
    : "down";
  resolvedDirectionRef.current = resolvedDirection;
  // useSwipe owns the commit decision — threshold, velocity, and pointer
  // capture. It reports progress only as a fraction of its threshold, which is
  // not the number a transform needs, so the offset is measured here from the
  // same pointer stream.
  const swipe = useSwipe({
    direction: resolvedDirection,
    threshold: swipeThreshold,
    velocity: 0.5,
    onCommit: () => {
      if (!swipeEnabled) return;
      if (movedDuringGesture.current) suppressNextClick.current = true;
      if (!layerManager.isTop(id)) {
        writeOffset(0);
        return;
      }
      writeDismissed();
      close();
    },
    onCancel: () => {
      if (movedDuringGesture.current) suppressNextClick.current = true;
      resetGestureDom();
    },
  });
  // `cancel` is an imperative control, not a pointer-event prop. Keep it out
  // of the object spread onto the grabber button.
  const { cancel: cancelSwipe, ...swipeHandlers } = swipe;

  const cancelSwipeRef = useRef(cancelSwipe);
  cancelSwipeRef.current = cancelSwipe;

  // Responsive and viewport changes can happen while a pointer is captured.
  // Cancel the gesture and clear every imperative DOM write before the new
  // presentation measures itself; otherwise the old translate survives on an
  // otherwise idle overlay and makes successive resizes look cumulative.
  useEffect(() => {
    cancelSwipeRef.current();
    resetGestureDom();
  }, [kind, resetGestureDom, resolvedDirection, swipeEnabled, viewportStyle]);

  const grabberKeyDown = useCallback(
    (event: ReactKeyboardEvent) => {
      if (event.key !== "Enter" && event.key !== " ") return;
      event.preventDefault();
      close();
    },
    [close],
  );

  return {
    id,
    open,
    present,
    state: open ? ("open" as const) : ("closed" as const),
    setOpen,
    close,
    surfaceRef,
    /** True when this surface accepts a dismissing drag. */
    swipeEnabled,
    surfaceProps: {
      ref: surfaceRef,
      "data-state": open ? "open" : "closed",
      style: viewportStyle,
    },
    /**
     * Props for the drag affordance. The presentation supplies the accessible
     * name and the geometry; dismissal mechanics stay here.
     */
    grabberProps: swipeEnabled
      ? {
          type: "button" as const,
          "data-rcl-overlay-grabber": "",
          onKeyDown: grabberKeyDown,
          onClick: (event: {
            preventDefault(): void;
            stopPropagation(): void;
          }) => {
            if (suppressNextClick.current) {
              suppressNextClick.current = false;
              event.preventDefault();
              event.stopPropagation();
              return;
            }
            close();
          },
          ...swipeHandlers,
          onPointerDown: (event: ReactPointerEvent) => {
            dragOrigin.current = { x: event.clientX, y: event.clientY };
            grabberRef.current = event.currentTarget;
            dragExtent.current =
              surfaceRef.current?.getBoundingClientRect().height ?? 0;
            movedDuringGesture.current = false;
            // Mark the gesture before it moves so the presentation's layer
            // promotion happens now rather than between the first two frames.
            markDragging(true);
            swipe.onPointerDown(event);
          },
          onPointerMove: (event: ReactPointerEvent) => {
            swipe.onPointerMove(event);
            const pixels = travelled(
              resolvedDirection,
              dragOrigin.current,
              event,
            );
            if (pixels > 4) movedDuringGesture.current = true;
            writeOffset(pixels);
          },
        }
      : {
          type: "button" as const,
          "data-rcl-overlay-grabber": "",
          onKeyDown: grabberKeyDown,
          onPointerDown: NOOP_POINTER,
          onPointerMove: NOOP_POINTER,
          onPointerUp: NOOP_POINTER,
          onPointerCancel: NOOP_POINTER,
        },
    backdropProps: {
      onPointerDown: (event: {
        target: EventTarget | null;
        currentTarget: EventTarget | null;
      }) => {
        if (
          dismiss.backdrop !== false &&
          event.target === event.currentTarget &&
          layerManager.isTop(id)
        )
          close();
      },
    },
  };
}
