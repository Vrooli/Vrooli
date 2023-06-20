import { useCallback, useEffect, useRef } from "react";

/** Hook for debouncing a function */
export const useDebounce = <T>(callback: (value: T) => void, delay: number) => {
    const callbackRef = useRef(callback);
    const timeoutRef = useRef<number>();

    // Update the callback ref if callback changes
    useEffect(() => {
        callbackRef.current = callback;
    }, [callback]);

    const debouncedCallback = useCallback(
        (value: T) => {
            if (timeoutRef.current) {
                clearTimeout(timeoutRef.current);
            }

            timeoutRef.current = window.setTimeout(() => {
                callbackRef.current(value);
            }, delay);
        },
        [delay],
    );

    useEffect(() => {
        return () => {
            if (timeoutRef.current) {
                clearTimeout(timeoutRef.current);
            }
        };
    }, []);

    return debouncedCallback;
};
