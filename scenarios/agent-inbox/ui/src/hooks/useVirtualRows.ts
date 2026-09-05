import { useCallback, useEffect, useMemo, useRef, useState, type RefObject } from "react";

interface UseVirtualRowsOptions {
  count: number;
  estimateSize: number;
  overscan?: number;
  enabled?: boolean;
  containerRef?: RefObject<HTMLDivElement | null>;
}

interface VirtualRow {
  index: number;
  offsetTop: number;
  size: number;
}

export function useVirtualRows({
  count,
  estimateSize,
  overscan = 6,
  enabled = true,
  containerRef: externalContainerRef,
}: UseVirtualRowsOptions) {
  const internalContainerRef = useRef<HTMLDivElement | null>(null);
  const containerRef = externalContainerRef ?? internalContainerRef;
  const sizeMapRef = useRef<Map<number, number>>(new Map());
  const [sizeVersion, setSizeVersion] = useState(0);
  const [viewport, setViewport] = useState({ scrollTop: 0, height: 0 });

  const refreshViewport = useCallback(() => {
    const element = containerRef.current;
    if (!element) return;
    setViewport({
      scrollTop: element.scrollTop,
      height: element.clientHeight,
    });
  }, [containerRef]);

  useEffect(() => {
    const element = containerRef.current;
    if (!element || !enabled) return;

    refreshViewport();
    element.addEventListener("scroll", refreshViewport, { passive: true });
    const observer = typeof ResizeObserver !== "undefined"
      ? new ResizeObserver(refreshViewport)
      : null;
    observer?.observe(element);

    return () => {
      element.removeEventListener("scroll", refreshViewport);
      observer?.disconnect();
    };
  }, [containerRef, enabled, refreshViewport]);

  const offsets = useMemo(() => {
    void sizeVersion;
    const next = Array.from<number>({ length: count + 1 });
    next[0] = 0;
    for (let index = 0; index < count; index++) {
      next[index + 1] = (next[index] ?? 0) + (sizeMapRef.current.get(index) ?? estimateSize);
    }
    return next;
    // version tracks measured-size changes stored in sizeMapRef.
  }, [count, estimateSize, sizeVersion]);

  const totalHeight = offsets[count] ?? 0;

  const { startIndex, endIndex } = useMemo(() => {
    if (!enabled) return { startIndex: 0, endIndex: count };

    const scrollTop = viewport.scrollTop;
    const viewportBottom = scrollTop + (viewport.height || estimateSize);
    let start = 0;
    while (start < count && (offsets[start + 1] ?? 0) < scrollTop) start++;
    let end = start;
    while (end < count && (offsets[end] ?? 0) <= viewportBottom) end++;

    return {
      startIndex: Math.max(0, start - overscan),
      endIndex: Math.min(count, end + overscan),
    };
  }, [count, enabled, estimateSize, offsets, overscan, viewport.height, viewport.scrollTop]);

  const virtualRows = useMemo<VirtualRow[]>(() => {
    const rows: VirtualRow[] = [];
    for (let index = startIndex; index < endIndex; index++) {
      rows.push({
        index,
        offsetTop: offsets[index] ?? index * estimateSize,
        size: sizeMapRef.current.get(index) ?? estimateSize,
      });
    }
    return rows;
  }, [endIndex, estimateSize, offsets, startIndex]);

  const measureElement = useCallback((index: number, element: HTMLElement | null) => {
    if (!element || !enabled) return;
    const next = Math.ceil(element.getBoundingClientRect().height);
    if (!Number.isFinite(next) || next <= 0) return;
    const prev = sizeMapRef.current.get(index);
    if (prev !== undefined && Math.abs(prev - next) <= 1) return;
    sizeMapRef.current.set(index, next);
    setSizeVersion((value) => value + 1);
  }, [enabled]);

  const scrollToIndex = useCallback((index: number, align: ScrollLogicalPosition = "start") => {
    const element = containerRef.current;
    if (!element || index < 0 || index >= count) return;
    const offsetTop = offsets[index] ?? index * estimateSize;
    if (align === "center") {
      element.scrollTop = Math.max(0, offsetTop - element.clientHeight / 2);
      return;
    }
    if (align === "end") {
      const size = sizeMapRef.current.get(index) ?? estimateSize;
      element.scrollTop = Math.max(0, offsetTop - element.clientHeight + size);
      return;
    }
    element.scrollTop = offsetTop;
  }, [containerRef, count, estimateSize, offsets]);

  const setContainerElement = useCallback((element: HTMLDivElement | null) => {
    internalContainerRef.current = element;
  }, []);

  return {
    containerRef,
    setContainerElement,
    measureElement,
    scrollToIndex,
    totalHeight,
    virtualRows,
  };
}
