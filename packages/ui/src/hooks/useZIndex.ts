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

    // If the component is visible and hasn't already been assigned a zIndex, get a new zIndex
    if ((visible === undefined || visible) && !hasCalledGetZIndex.current) {
        setZIndex(context?.getZIndex() ?? DEFAULT_Z_INDEX);
        hasCalledGetZIndex.current = true;
    }

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
