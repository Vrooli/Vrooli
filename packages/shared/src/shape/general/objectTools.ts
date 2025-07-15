// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-06-26
/**
 * Functions for manipulating state objects
 */
import { exists } from "../../utils/exists.js";

/**
 * Grabs data from an object using dot notation (ex: 'parent.child.property')
 */
export function valueFromDot<T = unknown>(object: Record<string, unknown>, notation: string): T | null {
    // Return null if either object or notation is falsy
    if (!object || !notation) return null;
    
    // Split notation and traverse the object
    const keys = notation.split(".");
    let current: unknown = object;
    
    for (const key of keys) {
        if (current === null || current === undefined) {
            return null;
        }
        
        // Handle both objects and arrays
        if (typeof current === "object") {
            current = (current as Record<string | number, unknown>)[key];
        } else {
            return null;
        }
    }
    
    return exists(current) ? (current as T) : null;
}

/**
 * Maps the keys of an object to dot notation
 */
export function convertToDot(
    obj: Record<string, unknown>, 
    parent: string[] = [], 
    keyValue: Record<string, unknown> = {},
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
    // If both are null/undefined, no change
    if ((original === null || original === undefined) && (updated === null || updated === undefined)) {
        return false;
    }
    // If one is null/undefined and the other isn't, there's a change
    if ((original === null || original === undefined) || (updated === null || updated === undefined)) {
        return true;
    }

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
            const originalValue = original[topLevelField];
            const updatedValue = updated[topLevelField];
            
            if (Array.isArray(originalValue) && Array.isArray(updatedValue)) {
                if (originalValue.length !== updatedValue.length) return true;
                for (let i = 0; i < originalValue.length; i++) {
                    if (hasObjectChanged(originalValue[i], updatedValue[i])) return true;
                }
                return false;
            } else if (isObject(originalValue) && isObject(updatedValue)) {
                return hasObjectChanged(originalValue, updatedValue);
            }
            return originalValue !== updatedValue;
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
