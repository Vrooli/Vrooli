// Functions for manipulating arrays, especially state arrays
// AI_CHECK: TYPE_SAFETY=shared-array-tools-type-safety-fixes | LAST: 2025-07-01 - Fixed 3 'any[]' type signatures to use proper generics and type constraints

import { valueFromDot } from "./objectTools.js";

export function addToArray<T>(array: T[], value: T) {
    return [...array, value];
}

export function updateArray<T>(array: T[], index: number, value: T) {
    if (JSON.stringify(array[index]) === JSON.stringify(value) || index < 0 || index >= array.length) return array;
    const copy = [...array];
    copy[index] = value;
    return copy;
}

export function deleteArrayIndex<T>(array: T[], index: number) {
    return array.filter((_, i) => i !== index);
}

export function deleteArrayObject<T>(array: T[], obj: T): T[] {
    const index = array.findIndex(item => item === obj);
    if (index !== -1) {
        const copy = [...array];
        copy.splice(index, 1);
        return copy;
    }
    return array;
}

// For an array of objects, return the first index where an object has a key/value 
// pair matching the attr/value pair
export function findWithAttr<T extends Record<string, unknown>>(array: T[], attr: keyof T, value: unknown): number {
    for (let i = 0; i < array.length; i += 1) {
        if (array[i][attr] === value) {
            return i;
        }
    }
    return -1;
}

export function moveArrayIndex<T>(array: T[], from: number, to: number): T[] {
    const copy = [...array];
    copy.splice(to, 0, copy.splice(from, 1)[0]);
    return copy;
}

// Shifts everything to the right, and puts the last element in the beginning
export function rotateArray<T>(array: T[], to_right = true): T[] {
    if (array.length === 0) return array;
    const copy = [...array];
    if (to_right) {
        const last_elem = copy.pop();
        if (last_elem !== undefined) {
            copy.unshift(last_elem);
        }
        return copy;
    } else {
        const first_elem = copy.shift();
        if (first_elem !== undefined) {
            copy.push(first_elem);
        }
        return copy;
    }
}

// If dot notation key exists in object, perform operation and return the results
export function mapIfExists<S extends Record<string, unknown>, T>(object: S, notation: string, operation: (obj: S) => T) {
    const value = valueFromDot(object, notation);
    if (!Array.isArray(value)) return null;
    return value.map(v => operation(v));
}
