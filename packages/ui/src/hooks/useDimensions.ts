import { useCallback, useEffect, useRef, useState } from "react";

type Dimensions = { width: number, height: number };

type UseDimensionsReturn<T extends HTMLElement> = {
    dimensions: Dimensions;
    ref: React.RefObject<T>;
    refreshDimensions: () => unknown;
}

export function useResizeAndMutationListener(callback: () => void, target: HTMLElement | null) {
    useEffect(() => {
        if (!target) return;

        const resizeObserver = new ResizeObserver(callback);
        const mutationObserver = new MutationObserver(callback);

        resizeObserver.observe(target);
        mutationObserver.observe(target, {
            attributes: true,
            childList: true,
            subtree: true,
        });

        function cleanup() {
            resizeObserver.disconnect();
            mutationObserver.disconnect();
        }

        return cleanup;
    }, [callback, target]);
}


export function useElementDimensions({
    id,
}: {
    id: string;
}): Dimensions {
    const [dimensions, setDimensions] = useState<Dimensions>({ width: 0, height: 0 });

    const updateDimensions = useCallback(() => {
        const element = document.getElementById(id);
        if (element) {
            setDimensions({
                width: element.clientWidth,
                height: element.clientHeight,
            });
        }
    }, [id]);

    useEffect(() => {
        updateDimensions();
    }, [updateDimensions]);

    useResizeAndMutationListener(updateDimensions, document.getElementById(id));

    return dimensions;
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

    const calculateDimensions = useCallback(() => {
        const width = ref.current?.clientWidth ?? 0;
        const height = ref.current?.clientHeight ?? 0;
        setDimensions({ width, height });
    }, []);

    useEffect(() => {
        calculateDimensions();
    }, [calculateDimensions]);

    useResizeAndMutationListener(calculateDimensions, ref.current);

    return { dimensions, ref, refreshDimensions: calculateDimensions };
}
