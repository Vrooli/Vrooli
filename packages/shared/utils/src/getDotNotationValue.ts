/**
 * Retrieves the value of an object property specified by a dot notation string, supporting arrays.
 *
 * @param obj The input object.
 * @param keyPath The dot notation string specifying the object property.
 * @returns The value of the object property specified by the dot notation string, or undefined if the property does not exist.
 */
export function getDotNotationValue(obj: object, keyPath: string) {
    // Split the key path into an array of keys
    const keys = keyPath.split('.');
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