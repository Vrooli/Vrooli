import { useCallback, useEffect, useState } from "react";
export const useWindowSize = (condition) => {
    const [conditionMet, setConditionMet] = useState(condition({ width: window.innerWidth, height: window.innerHeight }));
    const checkSizeCondition = useCallback(() => setConditionMet(condition({ width: window.innerWidth, height: window.innerHeight })), [condition]);
    useEffect(() => {
        checkSizeCondition();
        window.addEventListener("resize", checkSizeCondition);
        return () => window.removeEventListener("resize", checkSizeCondition);
    }, [checkSizeCondition]);
    return conditionMet;
};
//# sourceMappingURL=useWindowSize.js.map