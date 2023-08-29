
/**
 * Retrieves the value of an object property specified by a dot notation string, supporting arrays.
 *
 * @param obj The input object.
 * @param keyPath The dot notation string specifying the object property.
 * @returns The value of the object property specified by the dot notation string, or undefined if the property does not exist.
 */
export function getDotNotationValue(obj: object | undefined, keyPath: string) {
    if (!obj) return undefined;
    // Split the key path into an array of keys, making sure to handle array indices
    const keys = keyPath.split(/(\w+|\[\d+\])/g)
        .map((key) => key.replace(/^\[|\]$/g, ""))
        .filter((key) => !["", "."].includes(key));
    // Set the current value to the input object
    let currentValue: any = obj;
    // Loop through all the keys in the key path
    for (const key of keys) {
        // Check if the current value is an array and the key is an array index
        if (Array.isArray(currentValue) && /^\d+$/.test(key)) {
            const index = parseInt(key, 10);
            // Return undefined if the index is out of range
            if (index < 0 || index >= currentValue.length) {
                return undefined;
            }
            currentValue = currentValue[index];
        } else {
            // Return undefined if the key is not a property of the current value
            if (!currentValue || !(key in currentValue)) {
                return undefined;
            }
            currentValue = currentValue[key];
        }
    }
    // Return the value
    return currentValue;
}

/**
 * Sets data in an object using dot notation (ex: 'parent.child.property' or 'array[1].property')
 */
export const setDotNotationValue = <T extends Record<string, any>>(
    object: T,
    notation: string,
    value: any,
): T => {
    if (!object || !notation) return object;
    // Split the key path into an array of keys, making sure to handle array indices
    const keys = notation.split(/(\w+|\[\d+\])/g)
        .map(key => key.replace(/^\[|\]$/g, ""))
        .filter(key => !["", "."].includes(key));
    // Pop last key from array, as it will be used to set the value
    const lastKey = keys.pop() as string;
    // Use the other keys to get the target object
    const lastObj = keys.reduce((obj: any, key) => {
        console.log("translatedrichinput in handlechange setdotnotationvalue loop", key, obj);
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
};

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
export const isOfType = <T extends string>(obj: any, ...types: T[]): obj is { __typename: T } => {
    if (!obj || !obj.__typename) return false;
    return types.includes(obj.__typename);
};

export const deepClone = <T>(obj: T): T => {
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
};

