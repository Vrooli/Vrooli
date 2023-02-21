import { Dimensions } from "components/graphs/types";
import { useState, useEffect, useRef, useCallback } from "react";

type UseDimensionsReturn = {
    dimensions: Dimensions;
    ref: React.RefObject<HTMLDivElement>;
    refreshDimensions: () => void;
}

export const useDimensions = (): UseDimensionsReturn => {
    const [dimensions, setDimensions] = useState<Dimensions>({ width: 0, height: 0 });
    const ref = useRef<HTMLDivElement>(null);

    const calculateDimensions = useCallback(() => {
        const width = ref.current?.clientWidth ? ref.current.clientWidth - 50 : 0;
        const height = ref.current?.clientHeight ? ref.current.clientHeight - 50 : 0;
        setDimensions({ width, height });
    }, [setDimensions])

    useEffect(() => {
        calculateDimensions();
    }, [calculateDimensions]);

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