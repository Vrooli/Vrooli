import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";

const FALLBACK_VIEWPORT_HEIGHT = 900;
const SIZE_QUANTUM = 2;
/**
 * A scroll the user started is over once no scroll event has arrived for this
 * long and no finger is on the list. `scrollend`, where the browser has it,
 * ends it sooner.
 */
const SCROLL_IDLE_MS = 180;

interface UseVirtualListOptions {
  count: number;
  estimateSize: (index: number) => number;
  /**
   * Stable identity for an item.  Supplying this is essential for lists that
   * prepend pages: an index is no longer the same row after a prepend, while
   * the measured height still belongs to the original item.
   */
  getItemKey?: (index: number) => string | number;
  overscan?: number;
  scrollElementRef: React.RefObject<HTMLElement | null>;
  enabled?: boolean;
  anchorOnResize?: boolean;
}

interface VirtualItem {
  index: number;
  start: number;
  size: number;
}

function binarySearch(starts: number[], value: number): number {
  let low = 0;
  let high = starts.length - 1;
  let answer = 0;

  while (low <= high) {
    const mid = Math.floor((low + high) / 2);
    const start = starts[mid] ?? 0;
    if (start <= value) {
      answer = mid;
      low = mid + 1;
    } else {
      high = mid - 1;
    }
  }

  return answer;
}

export function useVirtualList({
  count,
  estimateSize,
  getItemKey,
  overscan = 6,
  scrollElementRef,
  enabled = true,
  anchorOnResize = true,
}: UseVirtualListOptions) {
  const [viewportHeight, setViewportHeight] = useState(0);
  const [scrollTop, setScrollTop] = useState(0);
  const [sizeVersion, setSizeVersion] = useState(0);
  const itemNodesRef = useRef(new Map<number, HTMLElement>());
  const itemObserversRef = useRef(new Map<number, ResizeObserver>());
  const measuredSizesRef = useRef(new Map<string | number, number>());
  const dirtyFromRef = useRef<number | null>(null);
  const frameRef = useRef<number | null>(null);
  const pendingAboveDeltaRef = useRef(0);
  const geometryRef = useRef({ sizes: [] as number[], starts: [] as number[], totalSize: 0, count: -1, estimateSize: null as UseVirtualListOptions["estimateSize"] | null });
  // A correction that arrives while the user is scrolling is carried as an
  // offset on every row (`shift`), not written to scrollTop: on iOS a
  // scrollTop write stops a fling dead. When the scroll ends the offset moves
  // into scrollTop in one write, which changes nothing on screen.
  const [shift, setShift] = useState(0);
  const shiftRef = useRef(0);
  const committedShiftRef = useRef(0);
  const userScrollingRef = useRef(false);
  const touchActiveRef = useRef(false);
  const idleTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const updateViewport = useCallback(() => {
    const el = scrollElementRef.current;
    if (!el) return;
    setViewportHeight(el.clientHeight);
    setScrollTop(el.scrollTop);
  }, [scrollElementRef]);

  const commitShift = useCallback(() => {
    if (shiftRef.current === 0) return;
    committedShiftRef.current += shiftRef.current;
    shiftRef.current = 0;
    setShift(0);
  }, []);

  /** Moves the viewport by `delta` along with its content: in scrollTop when idle, in the row offset mid-scroll. */
  const compensate = useCallback((delta: number) => {
    if (delta === 0) return;
    if (userScrollingRef.current) {
      shiftRef.current += delta;
      setShift(shiftRef.current);
      return;
    }
    const el = scrollElementRef.current;
    if (!el) return;
    el.scrollTop += delta;
    setScrollTop(el.scrollTop);
  }, [scrollElementRef]);

  useLayoutEffect(() => {
    const el = scrollElementRef.current;
    if (!el) return;

    updateViewport();
  }, [scrollElementRef, updateViewport]);

  useEffect(() => {
    const el = scrollElementRef.current;
    if (!el) return;

    const clearIdle = () => {
      if (idleTimerRef.current != null) clearTimeout(idleTimerRef.current);
      idleTimerRef.current = null;
    };
    const endScroll = () => {
      clearIdle();
      if (touchActiveRef.current) return;
      userScrollingRef.current = false;
      commitShift();
    };
    const armIdle = () => {
      clearIdle();
      idleTimerRef.current = setTimeout(endScroll, SCROLL_IDLE_MS);
    };
    const onScroll = () => {
      setScrollTop(el.scrollTop);
      if (userScrollingRef.current) armIdle();
    };
    // A finger on the list has already stopped any fling, so a carried offset
    // moves into scrollTop before this gesture starts a fling of its own.
    const onTouchStart = () => {
      touchActiveRef.current = true;
      clearIdle();
      userScrollingRef.current = false;
      commitShift();
      userScrollingRef.current = true;
    };
    const onTouchEnd = () => {
      touchActiveRef.current = false;
      armIdle();
    };
    const onWheel = () => {
      userScrollingRef.current = true;
      armIdle();
    };

    el.addEventListener("scroll", onScroll, { passive: true });
    el.addEventListener("touchstart", onTouchStart, { passive: true });
    el.addEventListener("touchend", onTouchEnd, { passive: true });
    el.addEventListener("touchcancel", onTouchEnd, { passive: true });
    el.addEventListener("wheel", onWheel, { passive: true });
    el.addEventListener("scrollend", endScroll);

    let resizeObserver: ResizeObserver | null = null;
    if (typeof ResizeObserver !== "undefined") {
      resizeObserver = new ResizeObserver(() => updateViewport());
      resizeObserver.observe(el);
    } else {
      window.addEventListener("resize", updateViewport);
    }

    return () => {
      clearIdle();
      el.removeEventListener("scroll", onScroll);
      el.removeEventListener("touchstart", onTouchStart);
      el.removeEventListener("touchend", onTouchEnd);
      el.removeEventListener("touchcancel", onTouchEnd);
      el.removeEventListener("wheel", onWheel);
      el.removeEventListener("scrollend", endScroll);
      resizeObserver?.disconnect();
      if (!resizeObserver) {
        window.removeEventListener("resize", updateViewport);
      }
    };
  }, [commitShift, scrollElementRef, updateViewport]);

  useEffect(() => {
    const observers = itemObserversRef.current;
    const nodes = itemNodesRef.current;
    return () => {
      for (const observer of observers.values()) observer.disconnect();
      observers.clear();
      nodes.clear();
      if (frameRef.current != null) cancelAnimationFrame(frameRef.current);
    };
  }, []);

  const measurements = useMemo(() => {
    const geometry = geometryRef.current;
    const rebuild = geometry.count !== count || geometry.estimateSize !== estimateSize;
    const dirtyFrom = rebuild ? 0 : dirtyFromRef.current;
    if (rebuild) {
      geometry.sizes = new Array<number>(count);
      geometry.starts = new Array<number>(count);
      geometry.count = count;
      geometry.estimateSize = estimateSize;
    }
    if (dirtyFrom != null) {
      let total = dirtyFrom === 0 ? 0 : (geometry.starts[dirtyFrom - 1] ?? 0) + (geometry.sizes[dirtyFrom - 1] ?? estimateSize(dirtyFrom - 1));
      for (let index = dirtyFrom; index < count; index += 1) {
        geometry.starts[index] = total;
        const size = measuredSizesRef.current.get(getItemKey?.(index) ?? index) ?? estimateSize(index);
        geometry.sizes[index] = size;
        total += size;
      }
      geometry.totalSize = total;
      dirtyFromRef.current = null;
    }
    // A fresh snapshot per recompute: consumers memoize on its identity, and
    // the geometry object itself is mutated in place.
    return { sizes: geometry.sizes, starts: geometry.starts, totalSize: geometry.totalSize };
  }, [count, estimateSize, sizeVersion]);

  const registerItem = useCallback((index: number, node: HTMLElement | null) => {
    const currentNode = itemNodesRef.current.get(index);
    if (currentNode === node) return;

    const currentObserver = itemObserversRef.current.get(index);
    currentObserver?.disconnect();
    itemObserversRef.current.delete(index);

    if (!node) {
      itemNodesRef.current.delete(index);
      return;
    }

    itemNodesRef.current.set(index, node);

    const measure = () => {
      const key = getItemKey?.(index) ?? index;
      const rawHeight = node.getBoundingClientRect().height || estimateSize(index);
      const height = Math.ceil(rawHeight / SIZE_QUANTUM) * SIZE_QUANTUM;
      const geometry = geometryRef.current;
      const previous = measuredSizesRef.current.get(key) ?? geometry.sizes[index] ?? estimateSize(index);
      measuredSizesRef.current.set(key, height);
      if (previous === height) return;
      dirtyFromRef.current = dirtyFromRef.current == null ? index : Math.min(dirtyFromRef.current, index);
      const element = scrollElementRef.current;
      // Compensate only rows entirely above the viewport. The row straddling
      // the top edge keeps its top where it is (the reader is looking at it);
      // compensating it would slide the viewport by its whole size correction.
      if (anchorOnResize && element && (geometry.starts[index] ?? 0) + previous <= element.scrollTop + shiftRef.current) {
        pendingAboveDeltaRef.current += height - previous;
      }
      if (frameRef.current != null) return;
      frameRef.current = requestAnimationFrame(() => {
        frameRef.current = null;
        setSizeVersion((version) => version + 1);
      });
    };

    measure();

    if (typeof ResizeObserver !== "undefined") {
      const observer = new ResizeObserver(() => measure());
      observer.observe(node);
      itemObserversRef.current.set(index, observer);
    }
  }, [anchorOnResize, estimateSize, getItemKey, scrollElementRef]);

  // Rows above the viewport that measured taller push the visible rows down.
  // Compensate in the same commit that moves them, before paint: adjusting
  // scrollTop a frame earlier (in the measurement frame) shows the content
  // shifted by the delta for one frame, which reads as flicker.
  useLayoutEffect(() => {
    const aboveDelta = pendingAboveDeltaRef.current;
    if (aboveDelta === 0) return;
    pendingAboveDeltaRef.current = 0;
    compensate(aboveDelta);
  }, [compensate, sizeVersion]);

  // The scroll ended with an offset carried: it moves into scrollTop in the
  // commit that drops it from the rows, so nothing moves on screen.
  useLayoutEffect(() => {
    const committed = committedShiftRef.current;
    if (committed === 0) return;
    committedShiftRef.current = 0;
    const el = scrollElementRef.current;
    if (!el) return;
    el.scrollTop += committed;
    setScrollTop(el.scrollTop);
  }, [scrollElementRef, shift]);

  const virtualItems = useMemo(() => {
    if (count === 0) return [] as VirtualItem[];

    if (!enabled) {
      return measurements.sizes.map((size, index) => ({
        index,
        size,
        start: (measurements.starts[index] ?? 0) - shift,
      }));
    }

    const effectiveViewportHeight = viewportHeight || FALLBACK_VIEWPORT_HEIGHT;
    const listTop = Math.max(0, scrollTop + shift);
    const rawStart = binarySearch(measurements.starts, listTop);
    const rawEnd = binarySearch(measurements.starts, listTop + effectiveViewportHeight);

    const startIndex = Math.max(0, rawStart - overscan);
    const endIndex = Math.min(count - 1, rawEnd + overscan);
    const items: VirtualItem[] = [];

    for (let index = startIndex; index <= endIndex; index += 1) {
      items.push({
        index,
        size: measurements.sizes[index] ?? estimateSize(index),
        start: (measurements.starts[index] ?? 0) - shift,
      });
    }

    return items;
  }, [count, enabled, estimateSize, measurements, overscan, scrollTop, shift, viewportHeight]);

  const scrollToIndex = useCallback((index: number, behavior: ScrollBehavior = "auto", align: "start" | "center" | "end" = "center") => {
    const el = scrollElementRef.current;
    if (!el || index < 0 || index >= count) return;

    const start = (measurements.starts[index] ?? 0) - shift;
    const size = measurements.sizes[index] ?? estimateSize(index);
    let top = start;

    if (align === "center") {
      top = start - Math.max(0, (el.clientHeight - size) / 2);
    } else if (align === "end") {
      top = start - Math.max(0, el.clientHeight - size);
    }

    el.scrollTo({ top: Math.max(0, top), behavior });
  }, [count, estimateSize, measurements.sizes, measurements.starts, scrollElementRef, shift]);

  /**
   * Keeps an item `offset` px below the viewport top across a change above it,
   * such as a prepended page. Call it from a layout effect of the render that
   * made the change.
   */
  const anchorItem = useCallback((index: number, offset: number) => {
    const el = scrollElementRef.current;
    if (!el) return;
    const top = (geometryRef.current.starts[index] ?? 0) - shiftRef.current;
    compensate(top - el.scrollTop - offset);
  }, [compensate, scrollElementRef]);

  /** The viewport top in list coordinates, counting an offset the rows still carry. */
  const listScrollTop = useCallback(() => (scrollElementRef.current?.scrollTop ?? 0) + shiftRef.current, [scrollElementRef]);

  const itemStart = useCallback((index: number) => (measurements.starts[index] ?? 0) - shift, [measurements, shift]);
  /** True once the item's measured height is part of the geometry (not an estimate). */
  const isMeasured = useCallback((index: number) => {
    const measured = measuredSizesRef.current.get(getItemKey?.(index) ?? index);
    return measured != null && measured === measurements.sizes[index];
  }, [getItemKey, measurements]);

  return {
    registerItem,
    itemStart,
    isMeasured,
    scrollTop,
    totalSize: measurements.totalSize - shift,
    viewportHeight,
    virtualItems,
    scrollToIndex,
    anchorItem,
    listScrollTop,
  };
}
