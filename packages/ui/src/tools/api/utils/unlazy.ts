/**
 * Unlazies an object
 */
export const unlazy = <T extends {}>(obj: T | (() => T)): T => typeof obj === 'function' ? (obj as () => T)() : obj;

/**
 * Recursively unlazies an object
 * @param obj The object to unlazify
 * @returns The unlazified object
 */
export const unlazyDeep = <T extends {}>(obj: T | (() => T)): T => {
    const unlazyObj = unlazy(obj);
    for (const key in unlazyObj) {
        const value = unlazyObj[key];
        if (Array.isArray(value)) {
            unlazyObj[key as any] = value.map(unlazyDeep);
        }
        else if (typeof value === 'function' || typeof value === 'object') {
            unlazyObj[key as any] = unlazyDeep(value as any);
        }
    }
    return unlazyObj;
}