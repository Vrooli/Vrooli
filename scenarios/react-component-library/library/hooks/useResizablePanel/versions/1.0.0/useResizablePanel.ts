/** @vrooliComponentSource hooks.use-resizable-panel */
import {
  useCallback,
  useEffect,
  useId,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type KeyboardEvent,
  type PointerEvent,
  type RefObject,
} from "react";

import { useDrag } from "@vrooli/react-component-library/useDrag/1.1.0";
import { useDirection } from "@vrooli/react-component-library/useDirection/1.0.0";
import { useElementRect } from "@vrooli/react-component-library/useElementRect/2.0.0";

/** Which dimension the panel owns. Logical, so a writing mode change is not a rewrite. */
export type ResizeAxis = "inline" | "block";
/** Which logical edge of the panel the separator sits on. */
export type ResizeEdge = "start" | "end";

export interface ResizeStorage {
  get: (key: string) => string | null;
  set: (key: string, value: string) => void;
}

export interface ResizeValueTextContext {
  size: number;
  panelName: string;
  isSnapped: boolean;
  isCollapsed: boolean;
}

export interface UseResizablePanelOptions {
  containerRef: RefObject<HTMLElement | null>;
  panelRef: RefObject<HTMLElement | null>;
  /** Defaults to the inline axis — a vertical separator between two columns. */
  axis?: ResizeAxis;
  /** Defaults to the inline-end / block-end edge of the panel. */
  edge?: ResizeEdge;
  min: number;
  max: number;
  defaultSize: number;
  /** Space the region on the other side of the separator must keep. */
  adjacentMin?: number;
  /** Pixels per arrow press. */
  step?: number;
  /** Pixels per arrow press with Shift held. */
  coarseStep?: number;
  /** Sizes the pointer settles onto when released within `snapTolerance`. */
  snapPoints?: readonly number[];
  snapTolerance?: number;
  /** Below this size the panel reports itself collapsed. */
  collapseBelow?: number;
  onCollapse?: (collapsed: boolean) => void;
  storage?: ResizeStorage;
  storageKey?: string;
  /** Used to build the default accessible name and value text. */
  panelName?: string;
  label?: string;
  formatValueText?: (context: ResizeValueTextContext) => string;
  onCommit?: (size: number) => void;
  disabled?: boolean;
}

export interface ResizeSeparatorProps {
  role: "separator";
  "aria-orientation": "horizontal" | "vertical";
  "aria-label": string;
  "aria-controls": string;
  "aria-valuemin": number;
  "aria-valuemax": number;
  "aria-valuenow": number;
  "aria-valuetext": string;
  "aria-disabled"?: true;
  tabIndex: number;
  style: CSSProperties;
  "data-axis": ResizeAxis;
  "data-edge": ResizeEdge;
  "data-dragging": "true" | "false";
  "data-snapped": "true" | "false";
  "data-collapsed": "true" | "false";
  onPointerDown: (event: PointerEvent<HTMLElement>) => void;
  onPointerMove: (event: PointerEvent<HTMLElement>) => void;
  onPointerUp: (event: PointerEvent<HTMLElement>) => void;
  onPointerCancel: (event: PointerEvent<HTMLElement>) => void;
  onKeyDown: (event: KeyboardEvent<HTMLElement>) => void;
  onDoubleClick: () => void;
}

export interface ResizePanelProps {
  id: string;
  style: CSSProperties;
  "data-rcl-resizable-panel": "";
  "data-axis": ResizeAxis;
  "data-collapsed": "true" | "false";
}

export interface UseResizablePanelResult {
  size: number;
  isResizing: boolean;
  isSnapped: boolean;
  isCollapsed: boolean;
  reset: () => void;
  separatorProps: ResizeSeparatorProps;
  panelProps: ResizePanelProps;
}

/** The custom property the live drag writes. Consumers may read it in CSS. */
export const RESIZE_SIZE_PROPERTY = "--rcl-panel-size";

const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value));

function defaultStorage(): ResizeStorage | undefined {
  if (typeof window === "undefined") return undefined;
  return {
    get: (key) => {
      try {
        return window.localStorage.getItem(key);
      } catch {
        return null;
      }
    },
    set: (key, value) => {
      try {
        window.localStorage.setItem(key, value);
      } catch {
        // A blocked or full store must never break resizing.
      }
    },
  };
}

function readPersisted(
  storage: ResizeStorage | undefined,
  key: string | undefined,
  fallback: number,
  min: number,
  max: number,
): number {
  if (!storage || !key) return fallback;
  const raw = storage.get(key);
  if (raw === null) return fallback;
  const parsed = Number(raw);
  return Number.isFinite(parsed) ? clamp(parsed, min, max) : fallback;
}

function snapTo(value: number, points: readonly number[], tolerance: number): number | null {
  let best: number | null = null;
  let bestDistance = tolerance;
  for (const point of points) {
    const distance = Math.abs(value - point);
    if (distance <= bestDistance) {
      best = point;
      bestDistance = distance;
    }
  }
  return best;
}

/**
 * Pointer, keyboard and persistence behavior for one resizable panel.
 *
 * The live drag writes `--rcl-panel-size` on the panel element and commits to
 * React state once, on release, so a drag re-renders the tree once instead of
 * on every pointer tick. The panel's size is expressed as
 * `var(--rcl-panel-size, <committed>px)`, so the custom property is the only
 * writer of the rendered dimension — React owns the fallback and the semantics.
 */
export function useResizablePanel(options: UseResizablePanelOptions): UseResizablePanelResult {
  const {
    containerRef,
    panelRef,
    axis = "inline",
    edge = "end",
    min,
    max,
    defaultSize,
    adjacentMin = 0,
    step = 8,
    coarseStep = 48,
    snapPoints,
    snapTolerance = 10,
    collapseBelow,
    onCollapse,
    storage,
    storageKey,
    panelName = "Panel",
    label,
    formatValueText,
    onCommit,
    disabled = false,
  } = options;

  const store = useMemo(() => storage ?? defaultStorage(), [storage]);
  const [size, setSize] = useState(() => readPersisted(store, storageKey, defaultSize, min, max));
  const [isSnapped, setIsSnapped] = useState(false);
  // Collapse is a projection of the committed size, not a second source of
  // truth that can disagree with it.
  const isCollapsed = collapseBelow !== undefined && size < collapseBelow;

  const documentDirection = useDirection();
  // Direction is resolved from the nearest `dir` ancestor rather than the
  // document alone, so a scoped `dir="rtl"` region resizes the way it reads.
  const [scopedDirection, setScopedDirection] = useState<"ltr" | "rtl" | null>(null);
  const direction = scopedDirection ?? documentDirection;
  const containerRect = useElementRect(containerRef, { disabled });
  const generatedId = useId();
  const panelId = `rcl-resizable-panel-${generatedId.replace(/[:]/g, "")}`;

  const liveSizeRef = useRef(size);
  const dragRectRef = useRef<DOMRect | null>(null);
  // Consumer callbacks are read through a ref so an inline arrow function does
  // not rebuild every handler and re-run every effect on each render.
  const callbacksRef = useRef({ onCommit, onCollapse });
  useEffect(() => {
    callbacksRef.current = { onCommit, onCollapse };
  });

  // The separator's own orientation, which is the axis it is drawn along —
  // a vertical bar separates two inline-flow regions. It is not the drag axis.
  const orientation = axis === "inline" ? "vertical" : "horizontal";

  // Resolve the logical edge to a physical one. Only the inline axis is
  // affected by direction; block flow is top-to-bottom in every writing mode
  // this library targets.
  const growsTowardEnd = axis === "inline" ? (edge === "end") !== (direction === "rtl") : edge === "end";

  const effectiveMax = useMemo(() => {
    const available = axis === "inline" ? containerRect?.width : containerRect?.height;
    // A zero measurement means the container has not been laid out yet (or is
    // rendering in a non-layout environment). Falling through to `max` keeps
    // the ceiling honest instead of pinning every panel to its minimum.
    if (!available || available <= 0) return max;
    return Math.max(min, Math.min(max, available - adjacentMin));
  }, [adjacentMin, axis, containerRect, max, min]);

  const writeLiveSize = useCallback(
    (next: number) => {
      liveSizeRef.current = next;
      panelRef.current?.style.setProperty(RESIZE_SIZE_PROPERTY, `${next}px`);
    },
    [panelRef],
  );

  const commit = useCallback(
    (next: number, snapped: boolean) => {
      const rounded = Math.round(next);
      liveSizeRef.current = rounded;
      setSize(rounded);
      setIsSnapped(snapped);
      if (store && storageKey) store.set(storageKey, String(rounded));
      callbacksRef.current.onCommit?.(rounded);
    },
    [store, storageKey],
  );

  const sizeFromPointer = useCallback(
    (event: globalThis.PointerEvent): number => {
      const rect = dragRectRef.current ?? containerRef.current?.getBoundingClientRect();
      if (!rect) return liveSizeRef.current;
      const raw =
        axis === "inline"
          ? growsTowardEnd
            ? event.clientX - rect.left
            : rect.right - event.clientX
          : growsTowardEnd
            ? event.clientY - rect.top
            : rect.bottom - event.clientY;
      return clamp(raw, min, effectiveMax);
    },
    [axis, containerRef, effectiveMax, growsTowardEnd, min],
  );

  const drag = useDrag({
    disabled,
    onStart: () => {
      // One layout read per drag rather than one per pointer tick.
      dragRectRef.current = containerRef.current?.getBoundingClientRect() ?? null;
      setIsSnapped(false);
    },
    onMove: (event) => writeLiveSize(sizeFromPointer(event)),
    onEnd: (event) => {
      const raw = sizeFromPointer(event);
      const snapped = snapPoints?.length ? snapTo(raw, snapPoints, snapTolerance) : null;
      const settled = snapped === null ? raw : clamp(snapped, min, effectiveMax);
      dragRectRef.current = null;
      writeLiveSize(settled);
      commit(settled, snapped !== null);
    },
    onCancel: () => {
      dragRectRef.current = null;
      writeLiveSize(size);
    },
  });

  const nudge = useCallback(
    (delta: number) => {
      const next = clamp(liveSizeRef.current + delta, min, effectiveMax);
      writeLiveSize(next);
      commit(next, false);
    },
    [commit, effectiveMax, min, writeLiveSize],
  );

  const setTo = useCallback(
    (next: number, snapped: boolean) => {
      const bounded = clamp(next, min, effectiveMax);
      writeLiveSize(bounded);
      commit(bounded, snapped);
    },
    [commit, effectiveMax, min, writeLiveSize],
  );

  const reset = useCallback(() => setTo(defaultSize, false), [defaultSize, setTo]);

  const pageStep = useCallback(
    (forward: boolean) => {
      const current = liveSizeRef.current;
      if (snapPoints?.length) {
        const ordered = [...snapPoints].sort((a, b) => a - b);
        const target = forward
          ? ordered.find((point) => point > current + 0.5)
          : [...ordered].reverse().find((point) => point < current - 0.5);
        if (target !== undefined) {
          setTo(target, true);
          return;
        }
        setTo(forward ? effectiveMax : min, false);
        return;
      }
      nudge(forward ? coarseStep : -coarseStep);
    },
    [coarseStep, effectiveMax, min, nudge, setTo, snapPoints],
  );

  const onKeyDown = useCallback(
    (event: KeyboardEvent<HTMLElement>) => {
      if (disabled) return;
      const distance = event.shiftKey ? coarseStep : step;
      const growKey = axis === "inline"
        ? growsTowardEnd ? "ArrowRight" : "ArrowLeft"
        : growsTowardEnd ? "ArrowDown" : "ArrowUp";
      const shrinkKey = axis === "inline"
        ? growsTowardEnd ? "ArrowLeft" : "ArrowRight"
        : growsTowardEnd ? "ArrowUp" : "ArrowDown";

      switch (event.key) {
        case growKey:
          event.preventDefault();
          nudge(distance);
          return;
        case shrinkKey:
          event.preventDefault();
          nudge(-distance);
          return;
        case "Home":
          event.preventDefault();
          setTo(min, false);
          return;
        case "End":
          event.preventDefault();
          setTo(effectiveMax, false);
          return;
        case "PageUp":
          event.preventDefault();
          pageStep(true);
          return;
        case "PageDown":
          event.preventDefault();
          pageStep(false);
          return;
        case "Enter":
          event.preventDefault();
          reset();
          return;
        default:
      }
    },
    [axis, coarseStep, disabled, effectiveMax, growsTowardEnd, min, nudge, pageStep, reset, setTo, step],
  );

  // Keep the rendered size honest when React owns it: after a commit, after a
  // container shrink that lowers the ceiling, and on first paint.
  useLayoutEffect(() => {
    if (disabled || drag.isDragging) return;
    const bounded = clamp(size, min, effectiveMax);
    if (bounded !== size) {
      commit(bounded, false);
      return;
    }
    writeLiveSize(size);
  }, [commit, disabled, drag.isDragging, effectiveMax, min, size, writeLiveSize]);

  useEffect(() => {
    liveSizeRef.current = size;
  }, [size]);

  // While a drag is live the whole document adopts the resize cursor and stops
  // selecting text, so dragging past the panel edge does not sweep a selection
  // across the workspace. Pointer capture keeps the events; this keeps the feel.
  useEffect(() => {
    if (!drag.isDragging || typeof document === "undefined") return;
    const { body } = document;
    const previousCursor = body.style.cursor;
    const previousUserSelect = body.style.userSelect;
    body.style.cursor = axis === "inline" ? "col-resize" : "row-resize";
    body.style.userSelect = "none";
    return () => {
      body.style.cursor = previousCursor;
      body.style.userSelect = previousUserSelect;
    };
  }, [axis, drag.isDragging]);

  useLayoutEffect(() => {
    const scope = panelRef.current?.closest("[dir]")?.getAttribute("dir");
    setScopedDirection(scope === "rtl" ? "rtl" : scope === "ltr" ? "ltr" : null);
  }, [panelRef]);

  const collapsedRef = useRef(isCollapsed);
  useEffect(() => {
    if (collapsedRef.current === isCollapsed) return;
    collapsedRef.current = isCollapsed;
    callbacksRef.current.onCollapse?.(isCollapsed);
  }, [isCollapsed]);

  const valueText = formatValueText
    ? formatValueText({ size, panelName, isSnapped, isCollapsed })
    : `${panelName} ${size} pixels`;

  const separatorProps: ResizeSeparatorProps = {
    role: "separator",
    "aria-orientation": orientation,
    "aria-label": label ?? `Resize ${panelName}`,
    "aria-controls": panelId,
    "aria-valuemin": min,
    "aria-valuemax": effectiveMax,
    "aria-valuenow": size,
    "aria-valuetext": valueText,
    ...(disabled ? { "aria-disabled": true as const } : {}),
    tabIndex: disabled ? -1 : 0,
    // Without this a touch drag pans the page instead of resizing the panel.
    style: { touchAction: "none" },
    "data-axis": axis,
    "data-edge": edge,
    "data-dragging": drag.isDragging ? "true" : "false",
    "data-snapped": isSnapped ? "true" : "false",
    "data-collapsed": isCollapsed ? "true" : "false",
    onPointerDown: drag.onPointerDown,
    onPointerMove: drag.onPointerMove,
    onPointerUp: drag.onPointerUp,
    onPointerCancel: drag.onPointerCancel,
    onKeyDown,
    onDoubleClick: reset,
  };

  const panelProps: ResizePanelProps = {
    id: panelId,
    style: {
      [axis === "inline" ? "inlineSize" : "blockSize"]: `var(${RESIZE_SIZE_PROPERTY}, ${size}px)`,
    },
    "data-rcl-resizable-panel": "",
    "data-axis": axis,
    "data-collapsed": isCollapsed ? "true" : "false",
  };

  return { size, isResizing: drag.isDragging, isSnapped, isCollapsed, reset, separatorProps, panelProps };
}
