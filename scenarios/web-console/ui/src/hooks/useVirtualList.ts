import { useCallback, useEffect, useMemo, useRef, useState } from "react";

interface UseVirtualListOptions {
  count: number;
  estimateSize: (index: number) => number;
  overscan?: number;
  scrollElementRef: React.RefObject<HTMLElement | null>;
  enabled?: boolean;
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
  overscan = 6,
  scrollElementRef,
  enabled = true,
}: UseVirtualListOptions) {
  const [viewportHeight, setViewportHeight] = useState(0);
  const [scrollTop, setScrollTop] = useState(0);
  const [measuredSizes, setMeasuredSizes] = useState<Record<number, number>>({});
  const itemNodesRef = useRef(new Map<number, HTMLElement>());
  const itemObserversRef = useRef(new Map<number, ResizeObserver>());

  const updateViewport = useCallback(() => {
    const el = scrollElementRef.current;
    if (!el) return;
    setViewportHeight(el.clientHeight);
    setScrollTop(el.scrollTop);
  }, [scrollElementRef]);

  useEffect(() => {
    const el = scrollElementRef.current;
    if (!el) return;

    updateViewport();
    const onScroll = () => {
      setScrollTop(el.scrollTop);
    };

    el.addEventListener("scroll", onScroll, { passive: true });

    let resizeObserver: ResizeObserver | null = null;
    if (typeof ResizeObserver !== "undefined") {
      resizeObserver = new ResizeObserver(() => updateViewport());
      resizeObserver.observe(el);
    } else {
      window.addEventListener("resize", updateViewport);
    }

    return () => {
      el.removeEventListener("scroll", onScroll);
      resizeObserver?.disconnect();
      if (!resizeObserver) {
        window.removeEventListener("resize", updateViewport);
      }
    };
  }, [scrollElementRef, updateViewport]);

  useEffect(() => {
    return () => {
      for (const observer of itemObserversRef.current.values()) observer.disconnect();
      itemObserversRef.current.clear();
      itemNodesRef.current.clear();
    };
  }, []);

  const measurements = useMemo(() => {
    const sizes = new Array<number>(count);
    const starts = new Array<number>(count);
    let totalSize = 0;

    for (let index = 0; index < count; index += 1) {
      starts[index] = totalSize;
      const size = measuredSizes[index] ?? estimateSize(index);
      sizes[index] = size;
      totalSize += size;
    }

    return { sizes, starts, totalSize };
  }, [count, estimateSize, measuredSizes]);

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
      const height = Math.ceil(node.getBoundingClientRect().height) || estimateSize(index);
      setMeasuredSizes((prev) => (prev[index] === height ? prev : { ...prev, [index]: height }));
    };

    measure();

    if (typeof ResizeObserver !== "undefined") {
      const observer = new ResizeObserver(() => measure());
      observer.observe(node);
      itemObserversRef.current.set(index, observer);
    }
  }, [estimateSize]);

  const virtualItems = useMemo(() => {
    if (count === 0) return [] as VirtualItem[];

    if (!enabled || viewportHeight <= 0) {
      return measurements.sizes.map((size, index) => ({
        index,
        size,
        start: measurements.starts[index] ?? 0,
      }));
    }

    const rawStart = binarySearch(measurements.starts, Math.max(0, scrollTop));
    const rawEnd = binarySearch(
      measurements.starts,
      Math.max(0, scrollTop + viewportHeight),
    );

    const startIndex = Math.max(0, rawStart - overscan);
    const endIndex = Math.min(count - 1, rawEnd + overscan);
    const items: VirtualItem[] = [];

    for (let index = startIndex; index <= endIndex; index += 1) {
      items.push({
        index,
        size: measurements.sizes[index] ?? estimateSize(index),
        start: measurements.starts[index] ?? 0,
      });
    }

    return items;
  }, [count, enabled, estimateSize, measurements, overscan, scrollTop, viewportHeight]);

  const scrollToIndex = useCallback((index: number, behavior: ScrollBehavior = "auto", align: "start" | "center" | "end" = "center") => {
    const el = scrollElementRef.current;
    if (!el || index < 0 || index >= count) return;

    const start = measurements.starts[index] ?? 0;
    const size = measurements.sizes[index] ?? estimateSize(index);
    let top = start;

    if (align === "center") {
      top = start - Math.max(0, (el.clientHeight - size) / 2);
    } else if (align === "end") {
      top = start - Math.max(0, el.clientHeight - size);
    }

    el.scrollTo({ top: Math.max(0, top), behavior });
  }, [count, estimateSize, measurements.sizes, measurements.starts, scrollElementRef]);

  return {
    registerItem,
    scrollTop,
    totalSize: measurements.totalSize,
    viewportHeight,
    virtualItems,
    scrollToIndex,
  };
}
