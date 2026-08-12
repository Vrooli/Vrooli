import { useCallback, useEffect, useRef } from "react";
/**
 * Press-and-hold key repeat for buttons. Fires once on pointerdown, then
 * after initialDelayMs begins firing at repeatIntervalMs until pointerup,
 * pointercancel, or pointerleave. Mirrors OS "typematic" arrow-key behavior
 * so mobile users can hold an arrow button instead of tapping repeatedly.
 *
 * The returned handlers are intended to be spread directly onto a <button>.
 * No onClick binding is needed — pointerdown calls preventDefault, and the
 * hook invokes onFire itself on the initial press.
 */
export function useHoldRepeat({ onFire, initialDelayMs = 400, repeatIntervalMs = 40, }) {
    const delayTimerRef = useRef(null);
    const repeatTimerRef = useRef(null);
    // Keep the latest onFire in a ref so the pointer handlers stay stable
    // across re-renders without re-subscribing timers.
    const onFireRef = useRef(onFire);
    useEffect(() => {
        onFireRef.current = onFire;
    }, [onFire]);
    const stop = useCallback(() => {
        if (delayTimerRef.current !== null) {
            clearTimeout(delayTimerRef.current);
            delayTimerRef.current = null;
        }
        if (repeatTimerRef.current !== null) {
            clearInterval(repeatTimerRef.current);
            repeatTimerRef.current = null;
        }
    }, []);
    // Ensure any live timers are cleared if the component unmounts mid-hold.
    useEffect(() => stop, [stop]);
    const onPointerDown = useCallback((e) => {
        if (e.pointerType === "mouse" && e.button !== 0)
            return;
        // Block the browser from shifting focus to the button — the terminal
        // must keep focus so the virtual keyboard stays up and keystrokes
        // route to xterm.
        e.preventDefault();
        stop();
        onFireRef.current();
        delayTimerRef.current = setTimeout(() => {
            delayTimerRef.current = null;
            repeatTimerRef.current = setInterval(() => {
                onFireRef.current();
            }, repeatIntervalMs);
        }, initialDelayMs);
    }, [initialDelayMs, repeatIntervalMs, stop]);
    const onPointerUp = useCallback(() => stop(), [stop]);
    const onPointerCancel = useCallback(() => stop(), [stop]);
    const onPointerLeave = useCallback(() => stop(), [stop]);
    return { onPointerDown, onPointerUp, onPointerCancel, onPointerLeave };
}
