/**
 * Unlazies an object
 */
export const unlazy = async <T extends {}>(obj: T | (() => T) | (() => Promise<T>)): Promise<T> => typeof obj === 'function' ? await (obj as () => T)() : obj;

/**
 * Recursively unlazies an object
 * @param obj The object to unlazify
 * @returns The unlazified object
 */
export const unlazyDeep = async <T extends {}>(obj: T | (() => T) | (() => Promise<T>)): Promise<T> => {
    const unlazyObj = await unlazy(obj);
    for (const key in unlazyObj) {
        const value = unlazyObj[key];
        if (Array.isArray(value)) {
            unlazyObj[key as any] = await Promise.all(value.map(unlazyDeep));
        }
        else if (typeof value === 'function' || typeof value === 'object') {
            unlazyObj[key as any] = await unlazyDeep(value as any);
        }
    }
    return unlazyObj;
}