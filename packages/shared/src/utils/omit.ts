/**
 * Omit an array of keys from an object. Supports dot notation.
 * @param obj The object to omit keys from
 * @param keysToOmit The keys to omit
 * @returns The object with the omitted keys
 */
export const omit = <T extends Record<string, any>>(obj: T, keysToOmit: string[]): Partial<T> => {
    // Make a shallow copy of the original object
    const result = { ...obj };

    // Helper function to delete nested keys
    const deleteKey = (obj: any, path: string[]) => {
        let current = obj;
        for (let i = 0; i < path.length - 1; i++) {
            const keyPart = path[i];
            if (keyPart === undefined || typeof current[keyPart] !== "object") return;
            current = current[keyPart];
        }
        const lastKey = path[path.length - 1];
        if (lastKey) {
            delete current[lastKey];
        }
    };

    // Loop through each key in the keysToOmit array
    for (const key of keysToOmit) {
        // Split the key using dot notation to extract nested keys
        const path = key.split(".");
        deleteKey(result, path);

        // Check if parent objects should be removed as well
        while (path.length > 1) {
            path.pop();
            const parentObject = path.reduce((acc, curr) => acc && acc[curr], result);
            if (typeof parentObject === "object" && Object.keys(parentObject).length === 0) {
                deleteKey(result, path);
            } else {
                break;
            }
        }
    }

    // Return the modified object
    return result;
};
