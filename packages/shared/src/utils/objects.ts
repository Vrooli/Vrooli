
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
            // Handle object properties using type assertion after checks
            const currentObj = currentValue as Record<string, unknown>;
            if (!(key in currentObj)) {
                return undefined;
            }
            currentValue = currentObj[key];
        }
    }

    return currentValue;
}

/**
 * Sets data in an object using dot notation (ex: 'parent.child.property' or 'array[1].property')
 */
export function setDotNotationValue<T extends Record<string, any>>(
    object: T,
    notation: string,
    value: any,
): T {
    if (!object || !notation) return object;
    // Split the key path into an array of keys, making sure to handle array indices
    const keys = notation.split(/(\w+|\[\d+\])/g)
        .map(key => key.replace(/^\[|\]$/g, ""))
        .filter(key => !["", "."].includes(key));
    // Pop last key from array, as it will be used to set the value
    const lastKey = keys.pop() as string;
    // Use the other keys to get the target object
    const lastObj = keys.reduce((obj: any, key) => {
        // Check if the key is an array index
        if (/^\d+$/.test(key)) {
            const index = parseInt(key, 10);
            if (!obj[index]) obj[index] = {};
            return obj[index];
        }
        // Ensure the key exists in the object
        if (!(key in obj)) obj[key] = {};
        return obj[key];
    }, object);
    // Set the value
    if (Array.isArray(lastObj) && /^\d+$/.test(lastKey)) {
        const index = parseInt(lastKey, 10);
        lastObj[index] = value;
    } else {
        lastObj[lastKey] = value;
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
export function isOfType<T extends string>(obj: any, ...types: T[]): obj is { __typename: T } {
    if (!obj || !obj.__typename) return false;
    return types.includes(obj.__typename);
}

export function deepClone<T>(obj: T): T {
    if (obj === null) return null as T;
    if (obj instanceof Date) return new Date(obj.getTime()) as unknown as T;

    if (typeof obj !== "object") return obj;

    if (Array.isArray(obj)) {
        const arrCopy: unknown[] = [];
        for (let i = 0; i < obj.length; i++) {
            arrCopy[i] = deepClone((obj as Array<unknown>)[i]);
        }
        return arrCopy as unknown as T;
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
 * @returns A new object with properties from both target and reference.
 */
export function mergeDeep<T>(target: T, reference: T): T {
    if (target === null || typeof target !== "object" || Array.isArray(target)) {
        return target !== undefined && target !== null ? target : reference;
    }

    const result: any = { ...reference };

    for (const key in target) {
        if (typeof target[key] === "object" && target[key] !== null && !Array.isArray(target[key])) {
            result[key] = mergeDeep(target[key] as object, reference[key] || {});
        } else {
            result[key] = target[key];
        }
    }

    return result;
}
