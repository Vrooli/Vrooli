function isUnsafeKey(key: string) {
    return key === "__proto__" || key === "constructor" || key === "prototype";
}

/**
 * Omit an array of keys from an object. Supports dot notation.
 * @param obj The object to omit keys from
 * @param keysToOmit The keys to omit
 * @returns The object with the omitted keys
 */
export function omit<T extends Record<string, unknown>>(obj: T, keysToOmit: string[]): Partial<T> {
    // Make a shallow copy of the original object
    const result = { ...obj };

    // Helper function to delete nested keys
    function deleteKey(obj: unknown, path: string[]) {
        let current = obj;
        for (let i = 0; i < path.length - 1; i++) {
            const keyPart = path[i];
            if (keyPart === undefined || typeof current !== "object" || current === null || isUnsafeKey(keyPart)) return;
            current = (current as Record<string, unknown>)[keyPart];
        }
        const lastKey = path[path.length - 1];
        if (lastKey && typeof current === "object" && current !== null && !isUnsafeKey(lastKey)) {
            delete (current as Record<string, unknown>)[lastKey];
        }
    }

    // Loop through each key in the keysToOmit array
    for (const key of keysToOmit) {
        // Split the key using dot notation to extract nested keys
        const path = key.split(".");
        deleteKey(result, path);

        // Check if parent objects should be removed as well
        while (path.length > 1) {
            path.pop();
            const parentObject = path.reduce<unknown>((acc, curr) => {
                if (acc && typeof acc === "object" && curr in acc) {
                    return (acc as Record<string, unknown>)[curr];
                }
                return undefined;
            }, result);
            if (typeof parentObject === "object" && parentObject !== null && Object.keys(parentObject).length === 0) {
                deleteKey(result, path);
            } else {
                break;
            }
        }
    }

    // Return the modified object
    return result;
}
