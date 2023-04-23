import { useCallback, useEffect, useRef, useState } from "react";
export const useDimensions = () => {
    const [dimensions, setDimensions] = useState({ width: 0, height: 0 });
    const ref = useRef(null);
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
    useEffect(() => {
        const handleResize = () => {
            if (ref.current) {
                refreshDimensions();
            }
            else {
                console.warn("No ref found for useDimensions hook.");
            }
        };
        window.addEventListener("resize", handleResize);
        return () => window.removeEventListener("resize", handleResize);
    }, [refreshDimensions]);
    return { dimensions, ref, refreshDimensions };
};
//# sourceMappingURL=useDimensions.js.map