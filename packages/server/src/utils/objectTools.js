import isObject from 'lodash/isObject';
import isArray from 'lodash/isArray';

// Remove all non-primitives from an object
export function onlyPrimitives(object) {
    if (!isObject(object)) return {};
    let result = {};
    for (const [key, value] of Object.entries(object)) {
        if (!isObject(value) && !isArray(value)) result[key] = value;
    }
    return result;
}