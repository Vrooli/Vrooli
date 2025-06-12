
/**
 * Retrieves the value of an object property specified by a dot notation string, supporting arrays.
 *
 * @param obj The input object.
 * @param keyPath The dot notation string specifying the object property.
 * @returns The value of the object property specified by the dot notation string, or undefined if the property does not exist.
 */
export function getDotNotationValue(obj: object | undefined, keyPath: string): unknown {
    // Handle undefined object
    if (!obj) return undefined;

    // Extract keys, including array indices; return empty array if no matches
    const keys = (keyPath.match(/\w+|\[\d+\]/g) || []).map((key) => key.replace(/^\[|\]$/g, ""));

    // Start with the input object
    let currentValue: unknown = obj;

    // Traverse the key path
    for (const key of keys) {
        // Type guard to ensure currentValue is an object or array
        if (currentValue === null || typeof currentValue !== "object") {
            return undefined;
        }

        // Handle array indices
        if (Array.isArray(currentValue) && /^\d+$/.test(key)) {
            const index = parseInt(key, 10);
            if (index < 0 || index >= currentValue.length) {
                return undefined;
            }
            currentValue = currentValue[index];
        } else {
            // Handle object properties
            // TypeScript already knows currentValue is an object and not null from line 22
            if (!(key in currentValue)) {
                return undefined;
            }
            currentValue = (currentValue as Record<string, unknown>)[key];
        }
    }

    return currentValue;
}

/**
 * Sets data in an object using dot notation (ex: 'parent.child.property' or 'array[1].property')
 */
export function setDotNotationValue<T extends Record<string, unknown>, V = unknown>(
    object: T,
    notation: string,
    value: V,
): T {
    if (!object || !notation) return object;
    // Split the key path into an array of keys, making sure to handle array indices
    const keys = notation.split(/(\w+|\[\d+\])/g)
        .map(key => key.replace(/^\[|\]$/g, ""))
        .filter(key => !["", "."].includes(key));
    // Pop last key from array, as it will be used to set the value
    const lastKey = keys.pop();
    if (!lastKey) {
        throw new Error("Invalid notation: no key found");
    }
    // Use the other keys to get the target object
    const lastObj = keys.reduce((obj: Record<string, unknown> | unknown[], key) => {
        // Check if the key is an array index
        if (/^\d+$/.test(key)) {
            const index = parseInt(key, 10);
            if (Array.isArray(obj)) {
                if (!obj[index]) obj[index] = {};
                return obj[index] as Record<string, unknown>;
            } else {
                if (typeof obj !== "object" || obj === null) {
                    throw new Error("Expected object for numeric key access");
                }
                const objectRef = obj as Record<string, unknown>;
                if (!objectRef[index]) {
                    objectRef[index] = {};
                }
                // Check if the value at this index is a primitive before returning it
                const nextValue = objectRef[index];
                if (nextValue !== null && typeof nextValue !== "object") {
                    throw new Error("Expected object for numeric key access");
                }
                return objectRef[index] as Record<string, unknown>;
            }
        }
        // Ensure the key exists in the object
        if (typeof obj !== "object" || obj === null) {
            throw new Error("Expected object for property access");
        }
        const objectRef = obj as Record<string, unknown>;
        if (!(key in objectRef)) {
            objectRef[key] = {};
        }
        // Check if the value at this key is a primitive before returning it
        const nextValue = objectRef[key];
        if (nextValue !== null && typeof nextValue !== "object") {
            throw new Error("Expected object for property access");
        }
        return objectRef[key] as Record<string, unknown>;
    }, object);
    // Set the value
    if (Array.isArray(lastObj) && /^\d+$/.test(lastKey)) {
        const index = parseInt(lastKey, 10);
        lastObj[index] = value;
    } else {
        if (typeof lastObj !== "object" || lastObj === null) {
            throw new Error("Expected object for property assignment");
        }
        (lastObj as Record<string, unknown>)[lastKey] = value;
    }
    // Return the updated object
    return object;
}

/**
 * Splits an array of dot notation strings into top-level fields and their corresponding remainders.
 * 
 * @param input An array of strings in dot notation.
 * @param removeEmpty Whether to remove empty strings from the output.
 * @returns A tuple where the first element is an array of top-level fields and the second element is an array of the remaining string parts.
 * 
 * Example:
 *   const input = ["first.second.third", "another.example.string"];
 *   const [fields, remainders] = splitDotNotation(input);
 *   // fields: ["first", "another"]
 *   // remainders: ["second.third", "example.string"]
 */
export function splitDotNotation(input: string[], removeEmpty = false): [string[], string[]] {
    const topLevelFields: string[] = [];
    const remainders: string[] = [];

    input.forEach(dotNotation => {
        if (typeof dotNotation === "string") {
            const [firstPart, ...rest] = dotNotation.split(".");
            topLevelFields.push(firstPart ?? "");
            remainders.push(rest.join("."));
        } else {
            // Handle null or undefined
            topLevelFields.push("");
            remainders.push("");
        }
    });

    if (removeEmpty) {
        return [
            topLevelFields.filter(field => field !== ""),
            remainders.filter(remainder => remainder !== ""),
        ];
    }

    return [topLevelFields, remainders];
}

/**
 * Checks if the value is an object.
 * @param value The value to check.
 * @returns True if the value is an object, false otherwise.
 */
export function isObject(value: unknown): value is object {
    return value !== null && (typeof value === "object" || typeof value === "function");
}

/**
 * Checks if an obect: 
 * 1. Exists
 * 2. Has the "__typename" property
 * 3. The "__typename" property is one of the provided types
 * @param obj The object to check
 * @param types The types to check against
 * @returns True if the object is of one of the provided types, false otherwise. 
 * The function is type safe, so code called after this function will be aware of the type of the object.
 */
export function isOfType<T extends string>(obj: unknown, ...types: T[]): obj is { __typename: T } {
    if (!obj || typeof obj !== "object" || !("__typename" in obj)) return false;
    const typedObj = obj as { __typename: unknown };
    return typeof typedObj.__typename === "string" && types.includes(typedObj.__typename as T);
}

export function deepClone<T>(obj: T): T {
    if (obj === null) return obj;
    if (obj instanceof Date) return new Date(obj.getTime()) as T;

    if (typeof obj !== "object") return obj;

    if (Array.isArray(obj)) {
        const arrCopy: unknown[] = [];
        for (let i = 0; i < obj.length; i++) {
            arrCopy[i] = deepClone((obj as Array<unknown>)[i]);
        }
        return arrCopy as T;
    } else {
        const objCopy: { [key: string]: unknown } = {};
        for (const key in obj) {
            if (Object.prototype.hasOwnProperty.call(obj, key)) {
                objCopy[key] = deepClone((obj as Record<string, unknown>)[key]);
            }
        }
        return objCopy as T;
    }
}

/**
 * Deeply merges two objects, ensuring all properties in the reference object are present in the target object.
 * 
 * @param target - The target object to be merged.
 * @param reference - The default object providing fallback values.
 * @param seen - WeakSet to track visited objects and prevent circular reference loops.
 * @returns A new object with properties from both target and reference.
 */
export function mergeDeep<T>(target: T, reference: T, seen = new WeakSet()): T {
    if (target === null || typeof target !== "object" || Array.isArray(target)) {
        return target !== undefined && target !== null ? target : reference;
    }

    // Check for circular reference
    if (seen.has(target as object)) {
        return target; // Return the circular reference as-is
    }
    seen.add(target as object);

    const result: Record<string, unknown> = { ...(reference as Record<string, unknown>) };

    for (const key in target) {
        const targetValue = (target as Record<string, unknown>)[key];
        const referenceValue = (reference as Record<string, unknown>)[key];
        
        if (typeof targetValue === "object" && targetValue !== null && !Array.isArray(targetValue)) {
            // Don't deep merge class instances (check constructor)
            if (targetValue.constructor && targetValue.constructor !== Object) {
                result[key] = targetValue;
            } else {
                result[key] = mergeDeep(targetValue, referenceValue || {}, seen);
            }
        } else {
            result[key] = targetValue;
        }
    }

    return result as T;
}
