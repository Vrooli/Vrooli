import { ZIndexContext } from "contexts/ZIndexContext";
import { useContext, useEffect, useState } from "react";

const DEFAULT_Z_INDEX = 200;

export const useZIndex = (visible?: boolean) => {
    const context = useContext(ZIndexContext);
    const [zIndex, setZIndex] = useState<number>(DEFAULT_Z_INDEX);

    useEffect(() => {
        // When the component becomes visible or is always visible
        if (visible === undefined || visible) {
            setZIndex(context?.getZIndex() ?? DEFAULT_Z_INDEX);
        } else {
            setZIndex(DEFAULT_Z_INDEX);
            context?.releaseZIndex();
        }

        // Cleanup when component is unmounted
        return () => {
            if (visible === undefined || visible) {
                context?.releaseZIndex();
            }
        };
    }, [visible, context]);

    return zIndex;
};
