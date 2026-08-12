import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from "react";
const FALLBACK_VIEWPORT_HEIGHT = 900;
const SIZE_QUANTUM = 2;
function binarySearch(starts, value) {
    let low = 0;
    let high = starts.length - 1;
    let answer = 0;
    while (low <= high) {
        const mid = Math.floor((low + high) / 2);
        const start = starts[mid] ?? 0;
        if (start <= value) {
            answer = mid;
            low = mid + 1;
        }
        else {
            high = mid - 1;
        }
    }
    return answer;
}
export function useVirtualList({ count, estimateSize, getItemKey, overscan = 6, scrollElementRef, enabled = true, anchorOnResize = true, }) {
    const [viewportHeight, setViewportHeight] = useState(0);
    const [scrollTop, setScrollTop] = useState(0);
    const [sizeVersion, setSizeVersion] = useState(0);
    const itemNodesRef = useRef(new Map());
    const itemObserversRef = useRef(new Map());
    const measuredSizesRef = useRef(new Map());
    const dirtyFromRef = useRef(null);
    const frameRef = useRef(null);
    const pendingAboveDeltaRef = useRef(0);
    const geometryRef = useRef({ sizes: [], starts: [], totalSize: 0, count: -1, estimateSize: null });
    const updateViewport = useCallback(() => {
        const el = scrollElementRef.current;
        if (!el)
            return;
        setViewportHeight(el.clientHeight);
        setScrollTop(el.scrollTop);
    }, [scrollElementRef]);
    useLayoutEffect(() => {
        const el = scrollElementRef.current;
        if (!el)
            return;
        updateViewport();
    }, [scrollElementRef, updateViewport]);
    useEffect(() => {
        const el = scrollElementRef.current;
        if (!el)
            return;
        const onScroll = () => {
            setScrollTop(el.scrollTop);
        };
        el.addEventListener("scroll", onScroll, { passive: true });
        let resizeObserver = null;
        if (typeof ResizeObserver !== "undefined") {
            resizeObserver = new ResizeObserver(() => updateViewport());
            resizeObserver.observe(el);
        }
        else {
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
        const observers = itemObserversRef.current;
        const nodes = itemNodesRef.current;
        return () => {
            for (const observer of observers.values())
                observer.disconnect();
            observers.clear();
            nodes.clear();
            if (frameRef.current != null)
                cancelAnimationFrame(frameRef.current);
        };
    }, []);
    const measurements = useMemo(() => {
        const geometry = geometryRef.current;
        const rebuild = geometry.count !== count || geometry.estimateSize !== estimateSize;
        const dirtyFrom = rebuild ? 0 : dirtyFromRef.current;
        if (rebuild) {
            geometry.sizes = new Array(count);
            geometry.starts = new Array(count);
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
        return geometry;
    }, [count, estimateSize, sizeVersion]);
    const registerItem = useCallback((index, node) => {
        const currentNode = itemNodesRef.current.get(index);
        if (currentNode === node)
            return;
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
            if (previous === height)
                return;
            measuredSizesRef.current.set(key, height);
            dirtyFromRef.current = dirtyFromRef.current == null ? index : Math.min(dirtyFromRef.current, index);
            const element = scrollElementRef.current;
            if (anchorOnResize && element && (geometry.starts[index] ?? 0) < element.scrollTop) {
                pendingAboveDeltaRef.current += height - previous;
            }
            if (frameRef.current != null)
                return;
            frameRef.current = requestAnimationFrame(() => {
                frameRef.current = null;
                const aboveDelta = pendingAboveDeltaRef.current;
                pendingAboveDeltaRef.current = 0;
                if (aboveDelta !== 0 && scrollElementRef.current) {
                    scrollElementRef.current.scrollTop += aboveDelta;
                    setScrollTop(scrollElementRef.current.scrollTop);
                }
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
    const virtualItems = useMemo(() => {
        if (count === 0)
            return [];
        if (!enabled) {
            return measurements.sizes.map((size, index) => ({
                index,
                size,
                start: measurements.starts[index] ?? 0,
            }));
        }
        const effectiveViewportHeight = viewportHeight || FALLBACK_VIEWPORT_HEIGHT;
        const rawStart = binarySearch(measurements.starts, Math.max(0, scrollTop));
        const rawEnd = binarySearch(measurements.starts, Math.max(0, scrollTop + effectiveViewportHeight));
        const startIndex = Math.max(0, rawStart - overscan);
        const endIndex = Math.min(count - 1, rawEnd + overscan);
        const items = [];
        for (let index = startIndex; index <= endIndex; index += 1) {
            items.push({
                index,
                size: measurements.sizes[index] ?? estimateSize(index),
                start: measurements.starts[index] ?? 0,
            });
        }
        return items;
    }, [count, enabled, estimateSize, measurements, overscan, scrollTop, viewportHeight]);
    const scrollToIndex = useCallback((index, behavior = "auto", align = "center") => {
        const el = scrollElementRef.current;
        if (!el || index < 0 || index >= count)
            return;
        const start = measurements.starts[index] ?? 0;
        const size = measurements.sizes[index] ?? estimateSize(index);
        let top = start;
        if (align === "center") {
            top = start - Math.max(0, (el.clientHeight - size) / 2);
        }
        else if (align === "end") {
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
