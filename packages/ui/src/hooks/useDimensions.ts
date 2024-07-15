import { Breakpoint, useTheme } from "@mui/material";
import { useCallback, useEffect, useRef, useState } from "react";

type Dimensions = { width: number, height: number };
type UseDimensionsReturn<T extends HTMLElement> = {
    dimensions: Dimensions;
    /** 
     * Uses Material UI spacing syntax to style based on the dimensions of the ref object, 
     * rather than the page's dimensions 
     */
    fromDims: (spacingObj: { [key in Breakpoint]?: unknown }) => any;
    ref: React.RefObject<T>;
    refreshDimensions: () => unknown;
}

/**
 * A React hook that calculates the dimensions of a given HTML element.
 *
 * @returns an object containing the element's dimensions, a reference to the element,
 * and a function to manually refresh the dimensions.
 */
export function useDimensions<T extends HTMLElement>(): UseDimensionsReturn<T> {
    const [dimensions, setDimensions] = useState<Dimensions>({ width: 0, height: 0 });
    const ref = useRef<T>(null);
    const { breakpoints } = useTheme();

    const fromDims = useCallback((spacingObj: { [key in Breakpoint]?: unknown }): any => {
        let appliedSpacing = spacingObj.xs;  // Default to xs
        for (const [breakpoint, value] of Object.entries(breakpoints.values)) {
            if (dimensions.width >= value && spacingObj[breakpoint] !== undefined) {
                appliedSpacing = spacingObj[breakpoint];
            }
        }
        return appliedSpacing;
    }, [dimensions.width, breakpoints]);

    const calculateDimensions = useCallback(() => {
        const width = ref.current?.clientWidth ?? 0;
        const height = ref.current?.clientHeight ?? 0;
        setDimensions({ width, height });
    }, [setDimensions]);

    useEffect(() => {
        calculateDimensions();
    }, [calculateDimensions]);

    const refreshDimensions = useCallback(() => {
        calculateDimensions();
    }, [calculateDimensions]);

    useEffect(function resizeListenerEffect() {
        let cleanup: () => void;

        function handleResize() {
            refreshDimensions();
        }

        if (typeof ResizeObserver === "function") {
            const observer = new ResizeObserver(() => {
                refreshDimensions();
            });

            if (ref.current) {
                observer.observe(ref.current);
            }

            cleanup = () => {
                if (ref.current) {
                    observer.unobserve(ref.current);
                }
            };
        } else {
            console.warn("Browser doesn't support ResizeObserver. Falling back to window resize listener.");


            window.addEventListener("resize", handleResize);
            cleanup = () => {
                window.removeEventListener("resize", handleResize);
            };
        }

        return cleanup;
    }, [refreshDimensions]);

    return { dimensions, fromDims, ref, refreshDimensions };
}
