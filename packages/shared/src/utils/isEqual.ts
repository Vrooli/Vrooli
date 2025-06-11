import { isObject } from "./objects.js";

/**
 * Performs a deep comparison of two objects and returns true if they are the same.
 * 
 * NOTE: Assumes the objects are the same type.
 * @param obj1 First object to compare.
 * @param obj2 Second object to compare.
 * @returns True if objects are the same, false otherwise.
 */
export function isEqual(obj1: unknown, obj2: unknown): boolean {
    // Handle primitives and same reference
    if (obj1 === obj2) return true;
    
    // Handle null/undefined
    if (obj1 == null || obj2 == null) return false;
    
    // Handle NaN special case (NaN should equal NaN in deep equality)
    if (typeof obj1 === 'number' && typeof obj2 === 'number' && isNaN(obj1) && isNaN(obj2)) {
        return true;
    }
    
    // Handle different constructors
    if (obj1.constructor !== obj2.constructor) return false;
    
    // Handle Date objects
    if (obj1 instanceof Date && obj2 instanceof Date) {
        return obj1.getTime() === obj2.getTime();
    }
    
    // Handle RegExp objects
    if (obj1 instanceof RegExp && obj2 instanceof RegExp) {
        return obj1.toString() === obj2.toString();
    }
    
    // Handle Functions
    if (typeof obj1 === 'function' && typeof obj2 === 'function') {
        // Functions are only equal if they're the same reference
        return false; // We already checked === above
    }
    
    // Handle Set objects
    if (obj1 instanceof Set && obj2 instanceof Set) {
        if (obj1.size !== obj2.size) return false;
        for (const value of obj1) {
            if (!obj2.has(value)) return false;
        }
        return true;
    }
    
    // Handle Map objects (order matters in Maps)
    if (obj1 instanceof Map && obj2 instanceof Map) {
        if (obj1.size !== obj2.size) return false;
        const keys1 = Array.from(obj1.keys());
        const keys2 = Array.from(obj2.keys());
        // Check if keys are in the same order
        for (let i = 0; i < keys1.length; i++) {
            if (keys1[i] !== keys2[i]) return false;
            if (!isEqual(obj1.get(keys1[i]), obj2.get(keys2[i]))) return false;
        }
        return true;
    }
    
    // Handle Boolean objects
    if (obj1 instanceof Boolean && obj2 instanceof Boolean) {
        return obj1.valueOf() === obj2.valueOf();
    }
    
    // Handle Number objects
    if (obj1 instanceof Number && obj2 instanceof Number) {
        return obj1.valueOf() === obj2.valueOf();
    }
    
    // Handle String objects
    if (obj1 instanceof String && obj2 instanceof String) {
        return obj1.valueOf() === obj2.valueOf();
    }
    
    // Handle Arrays
    if (Array.isArray(obj1)) {
        if (obj1.length !== (obj2 as Array<unknown>).length) return false;
        for (let i = 0; i < obj1.length; i++) {
            if (!isEqual(obj1[i], obj2[i])) return false;
        }
        return true;
    }
    
    // Handle plain objects
    if (isObject(obj1) && isObject(obj2)) {
        const keys1 = Object.keys(obj1);
        const keys2 = Object.keys(obj2);
        if (keys1.length !== keys2.length) return false;
        for (let i = 0; i < keys1.length; i++) {
            const key = keys1[i]!;
            if (!isEqual(obj1[key], obj2[key])) return false;
        }
        return true;
    }
    
    return false;
}
