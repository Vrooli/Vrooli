import { useCallback, useRef } from "react";

/** 
 * Hook for throttling a function.
 * Throttling means that the function will only be called once every `delay` milliseconds. 
 * This is different than debouncing, which means that the function will only be called once
 * after the last call within `delay` milliseconds.
 */
export function useThrottle<T extends unknown[]>(callback: (...args: T) => unknown, delay: number) {
    const lastCallRef = useRef<number>(Date.now());

    const throttledFunction = useCallback((...args: T) => {
        if (Date.now() - lastCallRef.current >= delay) {
            callback(...args);
            lastCallRef.current = Date.now();
        }
    }, [callback, delay]);

    return throttledFunction as ((...args: T) => unknown);
}
