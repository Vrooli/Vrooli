import { useEffect, useRef, useState } from "react";

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

type NonFunction<T> = T extends (...args: any[]) => any ? never : T;

/**
 * Only triggers a re-render if the object has actually changed. 
 * A changed ref but equal object will not trigger a re-render.
 * 
 * NOTE: Does not work for functions. Use useStableCallback for that.
 */
export const useStableObject = <T extends object | null | undefined>(obj: NonFunction<T>): NonFunction<T> => {
    const prevObjRef = useRef<NonFunction<T>>(obj as NonFunction<T>);
    const [, forceUpdate] = useState({});

    useEffect(() => {
        if (!deepEqual(prevObjRef.current, obj)) {
            prevObjRef.current = obj;
            forceUpdate({}); // Force a re-render
        }
    }, [obj]);

    return prevObjRef.current;
};
