/**
 * Removes "__typename" and "type" fields from the object, as these are only used 
 * for caching and determining the type of the object. They must not be passed 
 * into queries or mutations.
 */
export function removeTypename<T>(value: T): T extends (infer U)[] ? U[] : T extends object ? Omit<T, "__typename" | "type"> : T {
    if (value === null || value === undefined) return value;
    if (Array.isArray(value)) {
        return value.map(v => removeTypename(v));
    }
    if (typeof value === "object") {
        const newObj = {} as Record<string, unknown>;
        Object.keys(value as Record<string, unknown>).forEach(key => {
            if (key !== "__typename" && key !== "type") {
                newObj[key] = removeTypename((value as Record<string, unknown>)[key]);
            }
        });
        return newObj as T extends object ? Omit<T, "__typename" | "type"> : T;
    }
    return value;
}

// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Replaced 'any' types with generic type that preserves input/output relationship while removing typename fields
