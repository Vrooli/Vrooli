import { useCallback, useEffect, useRef } from "react";
export const useDebounce = (callback, delay) => {
    const timeoutRef = useRef();
    const debouncedCallback = useCallback((value) => {
        if (timeoutRef.current) {
            clearTimeout(timeoutRef.current);
        }
        timeoutRef.current = window.setTimeout(() => {
            callback(value);
        }, delay);
    }, [callback, delay]);
    useEffect(() => {
        return () => {
            if (timeoutRef.current) {
                clearTimeout(timeoutRef.current);
            }
        };
    }, []);
    return debouncedCallback;
};
//# sourceMappingURL=useDebounce.js.map