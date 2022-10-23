import { useCallback, useEffect, useRef } from "react";

/**
 * Hook for debouncing a function
 */
export const useDebounce = <T>(callback: (value: T) => void, delay: number) => {
    const timeoutRef = useRef<number>();

    const debouncedCallback = useCallback(
        (value: T) => {
            if (timeoutRef.current) {
                clearTimeout(timeoutRef.current);
            }

            timeoutRef.current = window.setTimeout(() => {
                callback(value);
            }, delay);
        },
        [callback, delay]
    );

    useEffect(() => {
        return () => {
            if (timeoutRef.current) {
                clearTimeout(timeoutRef.current);
            }
        };
    }, []);

    return debouncedCallback;
}