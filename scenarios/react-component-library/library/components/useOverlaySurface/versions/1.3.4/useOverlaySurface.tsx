/**
 * @libraryId react-component-library:useOverlaySurface
 * @displayName useOverlaySurface
 * @description Composes the shared lifecycle, focus, dismissal, portal, and motion contract for overlays.
 * @version 1.3.4
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
import { useSwipe, type SwipeDirection } from "@vrooli/react-component-library/useSwipe/2.0.2";
import { baseStyles } from "@vrooli/react-component-library/BaseStyles/1.2.0";
import { useLibraryStyleSheet } from "@vrooli/react-component-library/StyleSheet/1.0.0";

/** The presentation an overlay takes, which selects its layer band and role. */
export type OverlayKind = "dialog" | "alertdialog" | "menu" | "popover" | "sheet" | "drawer";

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
 * The custom property a presentation reads to follow a dismissing drag.
 *
 * It is the fraction of the surface's own block size the drag has travelled,
 * so `translateY(calc(var(--rcl-overlay-progress) * 100%))` moves the surface
 * exactly as far as the finger. It is deliberately *not* the fraction of the
 * commit threshold: that number is a constant 96px, and translating a 700px
 * panel by a fraction of 96px sent the surface roughly seven times faster than
 * the pointer that was dragging it.
 *
 * Each presentation stylesheet declares a resting value for it; it is never
 * set by inline style, so an unrelated re-render mid-gesture cannot reset it.
 */
const PROGRESS_PROPERTY = "--rcl-overlay-progress";

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
  const [present, setPresent] = useState(open);
  // The drag offset is written straight to a custom property on the surface.
  // Routing it through React state instead would re-render the whole overlay
  // subtree on every pointer move, which is exactly the frame budget a drag
  // needs. Only the coarse dragging flag is state, and it flips twice per
  // gesture.
  const [dragging, setDragging] = useState(false);
  const draggingRef = useRef(false);
  const surfaceRef = useRef<HTMLElement | null>(null);
  const id = useId();

  const dragOrigin = useRef({ x: 0, y: 0 });

  // `pixels` is the real distance the pointer has moved toward the dismissing
  // edge. It is divided by the surface's measured block size here rather than
  // by the commit threshold, because `100%` in the presentation's transform
  // means the surface's own height.
  const writeProgress = useCallback((pixels: number) => {
    const surface = surfaceRef.current;
    const extent = surface?.getBoundingClientRect().height ?? 0;
    const value = extent > 0 ? Math.max(0, Math.min(1, pixels / extent)) : 0;
    surface?.style.setProperty(PROGRESS_PROPERTY, String(value));
    const next = pixels > 0;
    if (next === draggingRef.current) return;
    draggingRef.current = next;
    setDragging(next);
  }, []);

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
    if (open) writeProgress(0);
  }, [open, writeProgress]);

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
      if (attemptsRemaining > 0) frame = window.requestAnimationFrame(focusInitial);
    };
    frame = window.requestAnimationFrame(focusInitial);
    return () => window.cancelAnimationFrame(frame);
  }, [initialFocusRef, open]);

  const swipeDirection = dismiss.swipe === undefined ? false : dismiss.swipe;
  const swipeEnabled = swipeDirection !== false;
  const resolvedDirection: SwipeDirection = swipeEnabled ? swipeDirection : "down";
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
      if (layerManager.isTop(id)) close();
      else writeProgress(0);
    },
    onCancel: () => writeProgress(0),
  });

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
    /**
     * True while a dismissing drag is in flight. A presentation suspends its
     * transform transition on this so the surface tracks the finger exactly.
     */
    dragging,
    /** True when this surface accepts a dismissing drag. */
    swipeEnabled,
    surfaceProps: {
      ref: surfaceRef,
      "data-state": open ? "open" : "closed",
      "data-dragging": dragging ? "true" : undefined,
    },
    /**
     * Props for the drag affordance. The presentation supplies the accessible
     * name and the geometry; dismissal mechanics stay here.
     */
    grabberProps: swipeEnabled
      ? {
          type: "button" as const,
          "data-rcl-overlay-grabber": "",
          "data-rcl-overlay-dragging": dragging ? "true" : undefined,
          onKeyDown: grabberKeyDown,
          ...swipe,
          onPointerDown: (event: ReactPointerEvent) => {
            dragOrigin.current = { x: event.clientX, y: event.clientY };
            swipe.onPointerDown(event);
          },
          onPointerMove: (event: ReactPointerEvent) => {
            swipe.onPointerMove(event);
            writeProgress(travelled(resolvedDirection, dragOrigin.current, event));
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
      onPointerDown: (event: { target: EventTarget | null; currentTarget: EventTarget | null }) => {
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
