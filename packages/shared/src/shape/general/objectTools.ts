/**
 * Functions for manipulating state objects
 */
import { exists } from "../../utils/exists.js";

/**
 * Grabs data from an object using dot notation (ex: 'parent.child.property')
 */
export function valueFromDot<T = unknown>(object: Record<string, unknown>, notation: string): T | null {
    // Utility function to index into object using string key
    function index(obj: Record<string, unknown> | null, i: string): Record<string, unknown> | null {
        // Return null if obj is falsy
        return exists(obj) ? (obj[i] as Record<string, unknown>) : null;
    }
    // Return null if either object or notation is falsy
    if (!object || !notation) return null;
    // Use reduce method to traverse the object using dot notation
    // If the final result is falsy, return null
    const value = notation.split(".").reduce(index, object);
    return exists(value) ? (value as T) : null;
}

/**
 * Maps the keys of an object to dot notation
 */
export function convertToDot(
    obj: Record<string, unknown>, 
    parent: string[] = [], 
    keyValue: Record<string, unknown> = {}
): Record<string, unknown> {
    for (const key in obj) {
        const keyPath: string[] = [...parent, key];
        if (obj[key] !== null && typeof obj[key] === "object") {
            Object.assign(keyValue, convertToDot(obj[key] as Record<string, unknown>, keyPath, keyValue));
        } else {
            keyValue[keyPath.join(".")] = obj[key];
        }
    }
    return keyValue;
}

/**
 * Checks if any of the specified fields (supports dot notation) in an object have been changed. 
 * If no fields are specified, checks if any fields have been changed.
 * @param original The original object
 * @param updated The updated object
 * @param fields The fields to check for changes, or empty array to check all fields
 */
export function hasObjectChanged(original: unknown, updated: unknown, fields: string[] = []): boolean {
    if (updated === null || updated === undefined) return false;
    if (original === null || original === undefined) return true;

    function isObject(obj: unknown) {
        return obj && typeof obj === "object" && !Array.isArray(obj);
    }

    // Direct comparison for non-objects
    if (!isObject(original) || !isObject(updated)) {
        return original !== updated;
    }

    function checkField(original: Record<string, unknown>, updated: Record<string, unknown>, field: string): boolean {
        const [topLevelField, ...rest] = field.split(".");
        if (typeof topLevelField !== "string") return false;

        if (rest.length > 0) {
            if (isObject(original[topLevelField]) && isObject(updated[topLevelField])) {
                return hasObjectChanged(original[topLevelField], updated[topLevelField], [rest.join(".")]);
            }
            return false;
        } else {
            if (Array.isArray(original[topLevelField]) && Array.isArray(updated[topLevelField])) {
                if (original[topLevelField].length !== updated[topLevelField].length) return true;
                for (let i = 0; i < (original[topLevelField] as unknown[]).length; i++) {
                    if (hasObjectChanged((original[topLevelField] as unknown[])[i], (updated[topLevelField] as unknown[])[i])) return true;
                }
                return false;
            } else if (isObject(original[topLevelField]) && isObject(updated[topLevelField])) {
                return hasObjectChanged(original[topLevelField], updated[topLevelField]);
            }
            return original[topLevelField] !== updated[topLevelField];
        }
    }

    // Combine keys from both objects
    const allKeys = new Set([...Object.keys(original as Record<string, unknown>), ...Object.keys(updated as Record<string, unknown>)]);

    // Check specified fields, or all fields if none specified
    const fieldsToCheck = fields.length > 0 ? fields : Array.from(allKeys);

    for (const field of fieldsToCheck) {
        if (checkField(original as Record<string, unknown>, updated as Record<string, unknown>, field)) {
            return true;
        }
    }

    return false;
}
