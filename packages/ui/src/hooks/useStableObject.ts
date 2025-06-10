import { useEffect, useRef, useState } from "react";

// A simple deep comparison function
function deepEqual(a: unknown, b: unknown): boolean {
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

type NonFunction<T> = T extends (...args: never[]) => never ? never : T;

/**
 * Only triggers a re-render if the object has actually changed. 
 * A changed ref but equal object will not trigger a re-render.
 * 
 * NOTE: Does not work for functions. Use useStableCallback for that.
 */
export function useStableObject<T extends string | number | object | null | undefined>(obj: NonFunction<T>): NonFunction<T> {
    const prevObjRef = useRef<NonFunction<T>>(obj as NonFunction<T>);
    const [stableObj, setStableObj] = useState<NonFunction<T>>(obj as NonFunction<T>);

    useEffect(function stableObjectEffect() {
        if (!deepEqual(prevObjRef.current, obj)) {
            prevObjRef.current = obj as NonFunction<T>;
            setStableObj(obj as NonFunction<T>);
        }
    }, [obj]);

    return stableObj;
}
