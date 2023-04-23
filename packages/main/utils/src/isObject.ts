/**
 * Checks if the value is an object.
 * @param value The value to check.
 * @returns True if the value is an object, false otherwise.
 */
export function isObject(value: any): value is object {
    return value !== null && (typeof value === "object" || typeof value === "function");
}
