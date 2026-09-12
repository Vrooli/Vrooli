/**
 * Pointer-and-keyboard reordering for a small list rendered as a grid.
 *
 * Written rather than installed: the scenario carries no drag-and-drop
 * library, dependency changes are governed, and the console already owns two
 * pointer-drag implementations (useDraggablePosition, the tab drag in
 * TerminalHeader). A list reorder over the same pointer events is smaller than
 * the adapter a library would need.
 *
 * Two rules shape the API:
 *
 *   1. Keyboard parity is not optional. A drag-only reorder is unusable with a
 *      keyboard or a screen reader, so every move is also reachable through
 *      moveFocused, and both paths go through the same moveItem.
 *   2. The hook owns the *pending* order only. Committing is the caller's, so
 *      a drag that is still in flight never writes to the server, and a failed
 *      write can restore the previous order without the hook having to know
 *      what a server is.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import { moveItem } from "./agentGrid";

/** How far a pointer must travel before a press becomes a drag, in px. */
const DRAG_THRESHOLD = 6;

export interface ReorderState<T> {
  /** The list in its current (possibly mid-drag) order. */
  items: T[];
  /** Index being dragged, or null. */
  draggingIndex: number | null;
  /** Index the dragged card would land on, or null. */
  targetIndex: number | null;
  /** True once the pointer has passed the movement threshold. */
  active: boolean;
  /** Attach to a card's grip: begins a potential drag. */
  onGripPointerDown: (index: number, event: React.PointerEvent) => void;
  /** Attach to each card element so the hook can hit-test during a drag. */
  registerItem: (index: number, node: HTMLElement | null) => void;
  /** Keyboard move for the focused card. Returns true when it moved. */
  moveFocused: (index: number, delta: number) => boolean;
  /** Discard any pending order and return to the source list. */
  reset: () => void;
}

export interface UseListReorderOptions<T> {
  /** The list to reorder. Changes reset any pending order. */
  source: readonly T[];
  /** Called with the new order once a move completes. */
  onCommit: (next: T[]) => void;
  /** When false the hook is inert, so the grid can share one render path. */
  enabled?: boolean;
}

export function useListReorder<T>({ source, onCommit, enabled = true }: UseListReorderOptions<T>): ReorderState<T> {
  const [pending, setPending] = useState<T[] | null>(null);
  const [draggingIndex, setDraggingIndex] = useState<number | null>(null);
  const [targetIndex, setTargetIndex] = useState<number | null>(null);
  const [active, setActive] = useState(false);
  const nodes = useRef(new Map<number, HTMLElement>());
  const origin = useRef<{ x: number; y: number } | null>(null);

  const items = useMemo(() => pending ?? [...source], [pending, source]);

  // A source change mid-drag means the underlying data moved beneath us — a
  // catalog refresh, say. Dropping the pending order is the honest response:
  // committing a position computed against a list that no longer exists would
  // reorder the wrong cards.
  useEffect(() => {
    setPending(null);
    setDraggingIndex(null);
    setTargetIndex(null);
    setActive(false);
  }, [source]);

  const registerItem = useCallback((index: number, node: HTMLElement | null) => {
    if (node) nodes.current.set(index, node);
    else nodes.current.delete(index);
  }, []);

  const indexAtPoint = useCallback((x: number, y: number): number | null => {
    for (const [index, node] of nodes.current) {
      const rect = node.getBoundingClientRect();
      if (x >= rect.left && x <= rect.right && y >= rect.top && y <= rect.bottom) return index;
    }
    return null;
  }, []);

  const reset = useCallback(() => {
    setPending(null);
    setDraggingIndex(null);
    setTargetIndex(null);
    setActive(false);
    origin.current = null;
  }, []);

  const onGripPointerDown = useCallback((index: number, event: React.PointerEvent) => {
    if (!enabled) return;
    // Only the primary button starts a drag; a right-click or a second finger
    // during a drag must not hijack it.
    if (event.button !== 0) return;
    event.preventDefault();
    origin.current = { x: event.clientX, y: event.clientY };
    setDraggingIndex(index);
    setTargetIndex(index);
    setActive(false);
  }, [enabled]);

  useEffect(() => {
    if (draggingIndex === null) return undefined;

    const onMove = (event: PointerEvent) => {
      const start = origin.current;
      if (!start) return;
      if (!active) {
        const travelled = Math.hypot(event.clientX - start.x, event.clientY - start.y);
        if (travelled < DRAG_THRESHOLD) return;
        setActive(true);
      }
      const over = indexAtPoint(event.clientX, event.clientY);
      if (over !== null) setTargetIndex(over);
    };

    const onUp = () => {
      const from = draggingIndex;
      const to = targetIndex;
      origin.current = null;
      setDraggingIndex(null);
      setActive(false);
      setTargetIndex(null);
      // A press that never moved is a press, not a reorder.
      if (!active || to === null || to === from) return;
      const next = moveItem(source, from, to);
      setPending(next);
      onCommit(next);
    };

    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp);
    window.addEventListener("pointercancel", onUp);
    return () => {
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
      window.removeEventListener("pointercancel", onUp);
    };
  }, [active, draggingIndex, indexAtPoint, onCommit, source, targetIndex]);

  const moveFocused = useCallback((index: number, delta: number): boolean => {
    if (!enabled) return false;
    const to = index + delta;
    if (to < 0 || to >= source.length) return false;
    const next = moveItem(source, index, to);
    setPending(next);
    onCommit(next);
    return true;
  }, [enabled, onCommit, source]);

  return { items, draggingIndex, targetIndex, active, onGripPointerDown, registerItem, moveFocused, reset };
}
