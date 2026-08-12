import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useFloatingPosition } from "./useFloatingPosition";
const DEFAULT_DRAG_THRESHOLD = 6;
const DEFAULT_FLOATING_MARGIN = 12;
const getPointerDelta = (state, event) => ({
    deltaX: event.clientX - state.startX,
    deltaY: event.clientY - state.startY,
});
const loadStoredPosition = (storageKey) => {
    if (typeof window === "undefined")
        return null;
    try {
        const stored = localStorage.getItem(storageKey);
        if (!stored)
            return null;
        const parsed = JSON.parse(stored);
        if (typeof parsed === "object" &&
            parsed !== null &&
            "x" in parsed &&
            "y" in parsed &&
            typeof parsed.x === "number" &&
            typeof parsed.y === "number") {
            return parsed;
        }
    }
    catch {
        console.warn("Failed to load stored position for", storageKey);
    }
    return null;
};
const saveStoredPosition = (storageKey, position) => {
    if (typeof window === "undefined")
        return;
    try {
        const data = {
            x: position.x,
            y: position.y,
            savedAt: Date.now(),
        };
        localStorage.setItem(storageKey, JSON.stringify(data));
    }
    catch {
        console.warn("Failed to save stored position for", storageKey);
    }
};
export const useDraggablePosition = (options) => {
    const { isActive, storageKey = null, defaultPosition, floatingMargin = DEFAULT_FLOATING_MARGIN, dragThreshold = DEFAULT_DRAG_THRESHOLD, onDragStart, onDragEnd, } = options;
    const { clampPosition } = useFloatingPosition({ floatingMargin });
    const elementRef = useRef(null);
    const dragStateRef = useRef(null);
    const suppressClickRef = useRef(false);
    const getInitialPosition = useCallback(() => {
        if (storageKey) {
            const stored = loadStoredPosition(storageKey);
            if (stored)
                return { x: stored.x, y: stored.y };
        }
        if (typeof defaultPosition === "function") {
            return defaultPosition() ?? { x: floatingMargin, y: floatingMargin };
        }
        return defaultPosition;
    }, [defaultPosition, floatingMargin, storageKey]);
    const [position, setPosition] = useState(getInitialPosition);
    const [isDragging, setIsDragging] = useState(false);
    const [isTrackingPointer, setIsTrackingPointer] = useState(false);
    const releasePointerCapture = useCallback((pointerId) => {
        const element = elementRef.current;
        if (!element)
            return;
        if (typeof element.hasPointerCapture === "function" &&
            !element.hasPointerCapture(pointerId))
            return;
        try {
            element.releasePointerCapture(pointerId);
        }
        catch {
            // pointer may already be released
        }
    }, []);
    // Track whether we've already initialized position for the current active session.
    // Use refs for getInitialPosition and clampPosition to avoid re-triggering the effect
    // when inline callbacks create new references every render.
    const prevActiveRef = useRef(false);
    const getInitialPositionRef = useRef(getInitialPosition);
    getInitialPositionRef.current = getInitialPosition;
    const clampPositionRef = useRef(clampPosition);
    clampPositionRef.current = clampPosition;
    useEffect(() => {
        if (!isActive) {
            if (dragStateRef.current?.pointerCaptured) {
                releasePointerCapture(dragStateRef.current.pointerId);
            }
            setIsDragging(false);
            setIsTrackingPointer(false);
            dragStateRef.current = null;
            prevActiveRef.current = false;
            return;
        }
        if (prevActiveRef.current)
            return; // already initialized
        prevActiveRef.current = true;
        const initialPos = getInitialPositionRef.current();
        const element = elementRef.current;
        if (element) {
            const rect = element.getBoundingClientRect();
            setPosition(clampPositionRef.current(initialPos.x, initialPos.y, {
                width: rect.width,
                height: rect.height,
            }));
        }
        else {
            setPosition(initialPos);
        }
    }, [isActive, releasePointerCapture]);
    useEffect(() => {
        if (typeof window === "undefined" || !isActive)
            return;
        const handleResize = () => {
            const element = elementRef.current;
            if (!element)
                return;
            const rect = element.getBoundingClientRect();
            setPosition((prev) => {
                const next = clampPosition(prev.x, prev.y, {
                    width: rect.width,
                    height: rect.height,
                });
                return next.x === prev.x && next.y === prev.y ? prev : next;
            });
        };
        handleResize();
        window.addEventListener("resize", handleResize);
        return () => window.removeEventListener("resize", handleResize);
    }, [clampPosition, isActive]);
    useEffect(() => {
        if (!storageKey || !isActive)
            return;
        saveStoredPosition(storageKey, position);
    }, [isActive, position, storageKey]);
    const handlePointerDown = useCallback((event) => {
        if (!isActive)
            return;
        if (event.pointerType === "mouse" && event.button !== 0)
            return;
        const element = elementRef.current;
        if (!element)
            return;
        const rect = element.getBoundingClientRect();
        dragStateRef.current = {
            pointerId: event.pointerId,
            startX: event.clientX,
            startY: event.clientY,
            offsetX: event.clientX - rect.left,
            offsetY: event.clientY - rect.top,
            width: rect.width,
            height: rect.height,
            pointerCaptured: false,
            dragging: false,
            lastPosition: null,
            velocitySamples: [],
        };
        setIsDragging(false);
        setIsTrackingPointer(true);
    }, [isActive]);
    const processPointerMove = useCallback((event) => {
        const state = dragStateRef.current;
        if (!state || state.pointerId !== event.pointerId)
            return;
        const element = elementRef.current;
        if (!element)
            return;
        const { deltaX, deltaY } = getPointerDelta(state, event);
        if (!state.dragging) {
            if (Math.abs(deltaX) + Math.abs(deltaY) < dragThreshold)
                return;
            state.dragging = true;
            setIsDragging(true);
            onDragStart?.();
            if (!state.pointerCaptured) {
                try {
                    element.setPointerCapture(event.pointerId);
                    state.pointerCaptured = true;
                }
                catch {
                    state.pointerCaptured = false;
                }
            }
        }
        if (!state.dragging)
            return;
        event.preventDefault?.();
        const next = clampPosition(event.clientX - state.offsetX, event.clientY - state.offsetY, { width: state.width, height: state.height });
        // Write transform directly to the DOM for immediate visual feedback,
        // bypassing React's async render cycle that causes 1+ frame lag (jitter).
        element.style.transform = `translate3d(${Math.round(next.x)}px, ${Math.round(next.y)}px, 0)`;
        state.lastPosition = next;
        // Track velocity samples (keep last 5 for smoothing)
        const now = performance.now();
        state.velocitySamples.push({ x: next.x, y: next.y, t: now });
        if (state.velocitySamples.length > 5)
            state.velocitySamples.shift();
    }, [clampPosition, dragThreshold, onDragStart]);
    const handlePointerMove = useCallback((event) => processPointerMove(event), [processPointerMove]);
    const processPointerEnd = useCallback((event) => {
        const state = dragStateRef.current;
        if (!state || state.pointerId !== event.pointerId)
            return;
        if (state.pointerCaptured)
            releasePointerCapture(event.pointerId);
        if (state.dragging) {
            event.preventDefault?.();
            suppressClickRef.current = true;
            // Compute velocity from recent samples
            let vx = 0;
            let vy = 0;
            const samples = state.velocitySamples;
            if (samples.length >= 2) {
                const first = samples[0];
                const last = samples[samples.length - 1];
                if (first && last) {
                    const dt = (last.t - first.t) / 1000; // seconds
                    if (dt > 0.001) {
                        vx = (last.x - first.x) / dt;
                        vy = (last.y - first.y) / dt;
                    }
                }
            }
            const finalPos = state.lastPosition ?? { x: 0, y: 0 };
            // Sync final drag position to React state (for floatingStyle + localStorage persistence)
            if (state.lastPosition) {
                setPosition(state.lastPosition);
            }
            onDragEnd?.({
                position: finalPos,
                velocity: { vx, vy },
                elementSize: { width: state.width, height: state.height },
            });
            window.setTimeout(() => {
                suppressClickRef.current = false;
            }, 0);
        }
        dragStateRef.current = null;
        setIsDragging(false);
        setIsTrackingPointer(false);
    }, [onDragEnd, releasePointerCapture]);
    const handlePointerEnd = useCallback((event) => processPointerEnd(event), [processPointerEnd]);
    useEffect(() => {
        if (typeof window === "undefined" || !isTrackingPointer)
            return;
        const onMove = (e) => processPointerMove(e);
        const onUp = (e) => processPointerEnd(e);
        window.addEventListener("pointermove", onMove, { passive: false });
        window.addEventListener("pointerup", onUp, { passive: false });
        window.addEventListener("pointercancel", onUp, { passive: false });
        return () => {
            window.removeEventListener("pointermove", onMove);
            window.removeEventListener("pointerup", onUp);
            window.removeEventListener("pointercancel", onUp);
        };
    }, [isTrackingPointer, processPointerEnd, processPointerMove]);
    const handleClickCapture = useCallback((event) => {
        if (suppressClickRef.current) {
            event.preventDefault();
            event.stopPropagation();
            suppressClickRef.current = false;
        }
    }, []);
    const floatingStyle = useMemo(() => {
        if (!isActive)
            return undefined;
        return {
            transform: `translate3d(${Math.round(position.x)}px, ${Math.round(position.y)}px, 0)`,
        };
    }, [isActive, position]);
    const resetPosition = useCallback(() => {
        const initial = getInitialPosition();
        setPosition(initial);
        if (storageKey) {
            try {
                localStorage.removeItem(storageKey);
            }
            catch {
                // noop
            }
        }
    }, [getInitialPosition, storageKey]);
    const moveTo = useCallback((pos) => {
        setPosition(pos);
    }, []);
    return useMemo(() => ({
        elementRef,
        position,
        isDragging,
        floatingStyle,
        pointerHandlers: {
            onPointerDown: handlePointerDown,
            onPointerMove: handlePointerMove,
            onPointerUp: handlePointerEnd,
            onPointerCancel: handlePointerEnd,
        },
        handleClickCapture,
        resetPosition,
        moveTo,
    }), [
        position,
        isDragging,
        floatingStyle,
        handlePointerDown,
        handlePointerMove,
        handlePointerEnd,
        handleClickCapture,
        resetPosition,
        moveTo,
    ]);
};
