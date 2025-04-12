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
    if (obj1 === obj2) return true;
    if (obj1 == null || obj2 == null) return false;
    if (obj1.constructor !== obj2.constructor) return false;
    if (Array.isArray(obj1)) {
        if (obj1.length !== (obj2 as Array<unknown>).length) return false;
        for (let i = 0; i < obj1.length; i++) {
            if (!isEqual(obj1[i], obj2[i])) return false;
        }
        return true;
    }
    if (isObject(obj1)) {
        const keys1 = Object.keys(obj1);
        const keys2 = Object.keys(obj2);
        if (keys1.length !== keys2.length) return false;
        for (let i = 0; i < keys1.length; i++) {
            if (!isEqual((obj1 as Array<unknown>)[keys1[i]!], obj2[keys1[i]!])) return false;
        }
        return true;
    }
    return false;
}
