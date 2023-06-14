import { Dimensions } from "components/graphs/types";
import { useCallback, useEffect, useRef, useState } from "react";

type UseDimensionsReturn = {
    dimensions: Dimensions;
    ref: React.RefObject<HTMLDivElement>;
    refreshDimensions: () => void;
}

/**
 * A React hook that calculates the dimensions of a given HTML element.
 *
 * @returns an object containing the element's dimensions, a reference to the element,
 * and a function to manually refresh the dimensions.
 */
export const useDimensions = (): UseDimensionsReturn => {
    // Set up state to store the element's dimensions
    const [dimensions, setDimensions] = useState<Dimensions>({ width: 0, height: 0 });
    // Set up a ref to the element whose dimensions we want to calculate
    const ref = useRef<HTMLElement>(null);

    // Define a function that calculates the element's dimensions
    const calculateDimensions = useCallback(() => {
        const width = ref.current?.clientWidth ?? 0;
        const height = ref.current?.clientHeight ?? 0;
        setDimensions({ width, height });
    }, [setDimensions]);

    // Calculate the dimensions when the component mounts or the element changes
    useEffect(() => {
        calculateDimensions();
    }, [calculateDimensions]);

    // Define a function to manually refresh the dimensions
    const refreshDimensions = useCallback(() => {
        calculateDimensions();
    }, [calculateDimensions]);

    // Update on screen resize
    useEffect(() => {
        const handleResize = () => {
            if (ref.current) {
                refreshDimensions();
            } else {
                console.warn("No ref found for useDimensions hook.");
            }
        };
        window.addEventListener("resize", handleResize);
        return () => window.removeEventListener("resize", handleResize);
    }, [refreshDimensions]);

    return { dimensions, ref, refreshDimensions };
};
