import { Dimensions } from "components/graphs/types";
import { useState, useEffect, useRef, useCallback } from "react";

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
    const ref = useRef<HTMLDivElement>(null);

    // Define a function that calculates the element's dimensions
    const calculateDimensions = useCallback(() => {
        const width = ref.current?.clientWidth ?? 0;
        const height = ref.current?.clientHeight ?? 0;
        setDimensions({ width, height });
    }, [setDimensions])

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
        window.addEventListener('resize', refreshDimensions);
        return () => window.removeEventListener('resize', refreshDimensions);
    }, [refreshDimensions]);

  return { dimensions, ref, refreshDimensions };
}