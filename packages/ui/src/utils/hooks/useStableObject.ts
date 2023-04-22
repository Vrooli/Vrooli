import { useEffect, useMemo, useRef } from "react";

// A simple deep comparison function
function deepEqual(a: any, b: any): boolean {
    if (a === b) return true;
    if (a === null || b === null || a === undefined || b === undefined) return false;
    if (typeof a !== "object" || typeof b !== "object") return false;

    const keysA = Object.keys(a);
    const keysB = Object.keys(b);

    if (keysA.length !== keysB.length) return false;

    for (const key of keysA) {
        if (!keysB.includes(key)) return false;
        if (!deepEqual(a[key], b[key])) return false;
    }

    return true;
}

/**
 * Only triggers a re-render if the object has actually changed. 
 * A changed ref but equal object will not trigger a re-render.
 */
export const useStableObject = <T extends object | null | undefined>(obj: T): T => {
    const prevObjRef = useRef<T>(obj);

    useEffect(() => {
        if (!deepEqual(prevObjRef.current, obj)) {
            prevObjRef.current = obj;
        }
    }, [obj]);

    return useMemo(() => prevObjRef.current, [prevObjRef]);
};
