import { useCallback, useEffect, useState } from "react";

const MOBILE_BREAKPOINT = 768;

/**
 * Hook for detecting if the current viewport is mobile-sized (< 768px width).
 * This hook is optimized to avoid infinite re-renders by using a stable condition.
 */
export function useIsMobile(): boolean {
    const [isMobile, setIsMobile] = useState<boolean>(() => window.innerWidth < MOBILE_BREAKPOINT);

    const checkIfMobile = useCallback(() => {
        setIsMobile(window.innerWidth < MOBILE_BREAKPOINT);
    }, []);

    useEffect(() => {
        checkIfMobile();
        window.addEventListener("resize", checkIfMobile);
        return () => window.removeEventListener("resize", checkIfMobile);
    }, [checkIfMobile]);

    return isMobile;
}

/**
 * Hook for getting current window dimensions that updates on resize.
 * Returns an object with width and height.
 */
export function useWindowDimensions(): { width: number; height: number } {
    const [dimensions, setDimensions] = useState(() => ({
        width: window.innerWidth,
        height: window.innerHeight,
    }));

    const updateDimensions = useCallback(() => {
        setDimensions({
            width: window.innerWidth,
            height: window.innerHeight,
        });
    }, []);

    useEffect(() => {
        updateDimensions();
        window.addEventListener("resize", updateDimensions);
        return () => window.removeEventListener("resize", updateDimensions);
    }, [updateDimensions]);

    return dimensions;
}
