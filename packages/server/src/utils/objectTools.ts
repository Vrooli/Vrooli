import isObject from 'lodash/isObject';
import isArray from 'lodash/isArray';

// Remove all non-primitives from an object
/**
 * Remove all non-primitives from an object
 * @param object An object with possible non-primitive values (arrays, objects, etc.)
 * @returns An object with only primitive values
 */
export function onlyPrimitives(object: any) {
    if (!isObject(object)) return {};
    let result: {[key: string]: boolean | null | number | string} = {};
    for (const [key, value] of Object.entries(object)) {
        if (!isObject(value) && !isArray(value)) result[key] = value;
    }
    return result;
}