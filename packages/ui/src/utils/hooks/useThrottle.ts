import { useCallback, useRef } from "react";

/** Hook for throttling a function */
export const useThrottle = <F extends (...args: any[]) => void>(callback: F, delay: number): F => {
    const lastCallRef = useRef(Date.now());

    return useCallback((...args: Parameters<F>) => {
        if (Date.now() - lastCallRef.current >= delay) {
            callback(...args);
            lastCallRef.current = Date.now();
        }
    }, [callback, delay]) as F;
};
