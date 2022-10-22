import { useCallback, useEffect, useState } from "react";

/**
 * Hook for detecting if a window's size meets a specified criteria.
 * @param condition The condition to check against the window's size.
 */
export const useWindowSize = (condition: ({ width, height }: { width: number, height: number }) => boolean): boolean => {
    const [conditionMet, setConditionMet] = useState<boolean>(condition({ width: window.innerWidth, height: window.innerHeight }));

    const checkSizeCondition = useCallback(() => setConditionMet(condition({ width: window.innerWidth, height: window.innerHeight })), [condition]);

    useEffect(() => {
        checkSizeCondition();
        window.addEventListener("resize", checkSizeCondition);
        return () => window.removeEventListener("resize", checkSizeCondition);
    }, [checkSizeCondition]);

    return conditionMet;
};