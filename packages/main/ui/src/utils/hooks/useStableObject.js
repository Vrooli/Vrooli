import { useEffect, useMemo, useRef } from "react";
function deepEqual(a, b) {
    if (a === b)
        return true;
    if (a === null || b === null || a === undefined || b === undefined)
        return false;
    if (typeof a !== "object" || typeof b !== "object")
        return false;
    const keysA = Object.keys(a);
    const keysB = Object.keys(b);
    if (keysA.length !== keysB.length)
        return false;
    for (const key of keysA) {
        if (!keysB.includes(key))
            return false;
        if (!deepEqual(a[key], b[key]))
            return false;
    }
    return true;
}
export const useStableObject = (obj) => {
    const prevObjRef = useRef(obj);
    useEffect(() => {
        if (!deepEqual(prevObjRef.current, obj)) {
            prevObjRef.current = obj;
        }
    }, [obj]);
    return useMemo(() => prevObjRef.current, [prevObjRef]);
};
//# sourceMappingURL=useStableObject.js.map