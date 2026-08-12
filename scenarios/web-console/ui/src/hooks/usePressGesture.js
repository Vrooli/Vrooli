import { useCallback, useEffect, useRef } from "react";
const DEFAULT_LONG_PRESS_MS = 500;
const DEFAULT_MOVE_THRESHOLD_PX = 8;
const CLICK_SUPPRESSION_MS = 750;
/**
 * Classifies touch/pen item gestures as tap, long-press, or moved/cancelled.
 *
 * Native scroll containers still handle scrolling; this hook only prevents our
 * own item activation from running after the user's finger has moved far enough
 * that the gesture should no longer count as a tap.
 */
export function usePressGesture({ longPressMs = DEFAULT_LONG_PRESS_MS, moveThresholdPx = DEFAULT_MOVE_THRESHOLD_PX, onTap, onLongPress, onMoveThreshold, }) {
    const activeRef = useRef(null);
    const timerRef = useRef(null);
    const callbacksRef = useRef({ onTap, onLongPress, onMoveThreshold });
    const suppressedClickRef = useRef(null);
    const windowListenersRef = useRef({ move: null, up: null, cancel: null });
    useEffect(() => {
        callbacksRef.current = { onTap, onLongPress, onMoveThreshold };
    }, [onTap, onLongPress, onMoveThreshold]);
    const clearTimer = useCallback(() => {
        if (timerRef.current) {
            clearTimeout(timerRef.current);
            timerRef.current = null;
        }
    }, []);
    const suppressClick = useCallback((id) => {
        suppressedClickRef.current = { id, until: Date.now() + CLICK_SUPPRESSION_MS };
    }, []);
    const shouldSuppressClick = useCallback((id) => {
        const suppressed = suppressedClickRef.current;
        if (!suppressed)
            return false;
        if (Date.now() > suppressed.until) {
            suppressedClickRef.current = null;
            return false;
        }
        if (suppressed.id !== id)
            return false;
        suppressedClickRef.current = null;
        return true;
    }, []);
    const removeWindowListeners = useCallback(() => {
        const listeners = windowListenersRef.current;
        if (listeners.move)
            window.removeEventListener("pointermove", listeners.move);
        if (listeners.up)
            window.removeEventListener("pointerup", listeners.up);
        if (listeners.cancel)
            window.removeEventListener("pointercancel", listeners.cancel);
        windowListenersRef.current = { move: null, up: null, cancel: null };
    }, []);
    const reset = useCallback(() => {
        clearTimer();
        activeRef.current = null;
        removeWindowListeners();
    }, [clearTimer, removeWindowListeners]);
    useEffect(() => reset, [reset]);
    const getGestureHandlers = useCallback((id) => ({
        onPointerDown: (event) => {
            if (event.pointerType === "mouse" || event.button !== 0)
                return;
            reset();
            activeRef.current = {
                id,
                pointerId: event.pointerId,
                pointerType: event.pointerType,
                target: event.currentTarget,
                start: { x: event.clientX, y: event.clientY },
                current: { x: event.clientX, y: event.clientY },
                moved: false,
                longPressReady: false,
            };
            timerRef.current = setTimeout(() => {
                const active = activeRef.current;
                if (!active || active.id !== id || active.moved)
                    return;
                active.longPressReady = true;
                timerRef.current = null;
            }, longPressMs);
            const handleWindowPointerMove = (moveEvent) => {
                const active = activeRef.current;
                if (!active || moveEvent.pointerId !== active.pointerId)
                    return;
                active.current = { x: moveEvent.clientX, y: moveEvent.clientY };
                if (active.moved)
                    return;
                const dx = moveEvent.clientX - active.start.x;
                const dy = moveEvent.clientY - active.start.y;
                if (Math.sqrt(dx * dx + dy * dy) <= moveThresholdPx)
                    return;
                active.moved = true;
                active.longPressReady = false;
                suppressClick(active.id);
                clearTimer();
                callbacksRef.current.onMoveThreshold?.({
                    id: active.id,
                    pointerId: active.pointerId,
                    pointerType: active.pointerType,
                    target: active.target,
                    start: active.start,
                    current: active.current,
                });
            };
            const handleWindowPointerUp = (upEvent) => {
                const active = activeRef.current;
                if (!active || upEvent.pointerId !== active.pointerId)
                    return;
                const point = { x: upEvent.clientX, y: upEvent.clientY };
                const { id: activeId, moved, longPressReady } = active;
                reset();
                if (moved)
                    return;
                suppressClick(activeId);
                if (longPressReady) {
                    callbacksRef.current.onLongPress(activeId, point);
                    return;
                }
                callbacksRef.current.onTap(activeId, point);
            };
            const handleWindowPointerCancel = (cancelEvent) => {
                const active = activeRef.current;
                if (!active || cancelEvent.pointerId !== active.pointerId)
                    return;
                reset();
            };
            windowListenersRef.current = {
                move: handleWindowPointerMove,
                up: handleWindowPointerUp,
                cancel: handleWindowPointerCancel,
            };
            window.addEventListener("pointermove", handleWindowPointerMove);
            window.addEventListener("pointerup", handleWindowPointerUp);
            window.addEventListener("pointercancel", handleWindowPointerCancel);
        },
        onPointerCancel: reset,
        onContextMenu: (event) => {
            event.preventDefault();
            reset();
            suppressClick(id);
            callbacksRef.current.onLongPress(id, { x: event.clientX, y: event.clientY });
        },
    }), [clearTimer, longPressMs, moveThresholdPx, reset, suppressClick]);
    return { getGestureHandlers, reset, shouldSuppressClick };
}
