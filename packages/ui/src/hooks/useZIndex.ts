import { DEFAULT_Z_INDEX, ZIndexContext } from "contexts/ZIndexContext";
import { useContext, useEffect, useRef, useState } from "react";

type UseZIndexReturn<HasTransition extends boolean> = HasTransition extends true ? [number, (() => unknown)] : number;

export const useZIndex = <
    HasTransition extends boolean = false
>(
    visible?: boolean,
    hasTransition?: HasTransition,
): UseZIndexReturn<HasTransition> => {
    const context = useContext(ZIndexContext);
    const [zIndex, setZIndex] = useState<number>(DEFAULT_Z_INDEX);
    const hasCalledGetZIndex = useRef(false);

    // Update the zIndex depending on the visibility of the component   
    useEffect(() => {
        // Set zIndex if component is visible
        if (visible === undefined || visible) {
            if (hasCalledGetZIndex.current) return;
            hasCalledGetZIndex.current = true;
            setZIndex(context?.getZIndex() ?? DEFAULT_Z_INDEX);
        }
        // Remove zIndex if component is not visible, and we're not 
        // waiting for a transition to finish
        else if (!hasTransition) {
            setZIndex(DEFAULT_Z_INDEX);
            if (hasCalledGetZIndex.current) context?.releaseZIndex();
            hasCalledGetZIndex.current = false;
        }
    }, [hasTransition, visible, context]);

    // Cleanup on component unmount
    useEffect(() => {
        return () => {
            if (hasCalledGetZIndex.current) context?.releaseZIndex();
            hasCalledGetZIndex.current = false;
        };
    }, [context]);

    const handleTransitionExit = () => {
        if (!visible && context) {
            setZIndex(DEFAULT_Z_INDEX);
            if (hasCalledGetZIndex.current) context?.releaseZIndex();
            hasCalledGetZIndex.current = false;
        }
    };

    if (hasTransition) return [zIndex, handleTransitionExit] as UseZIndexReturn<HasTransition>;
    return zIndex as UseZIndexReturn<HasTransition>;
};
