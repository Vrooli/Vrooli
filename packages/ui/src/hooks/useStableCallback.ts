import { useCallback, useRef } from "react";

/**
 * A custom hook that returns a stable callback function, ensuring that the
 * callback function does not cause re-renders unless the function itself changes.
 */
export function useStableCallback<T extends ((...args: any[]) => unknown) | undefined>(callback: T): T {
    // Create a ref to store the current callback
    const callbackRef = useRef<T>(callback);

    // Update the ref with the current callback each time it changes
    // useCallback is used here to prevent unnecessary re-renders
    useCallback(() => {
        callbackRef.current = callback;
    }, [callback]);

    // Define a stable callback that calls the latest version of the
    // callback stored in the ref when invoked
    const stableCallback = useCallback((...args: Parameters<NonNullable<T>>) => {
        if (callbackRef.current) {
            return callbackRef.current(...args);
        }
    }, []);

    // If the callback is undefined, return undefined directly
    return callback ? (stableCallback as T) : callback;
}
