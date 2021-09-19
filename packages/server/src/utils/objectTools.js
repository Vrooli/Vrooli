import _ from "lodash";

// Remove all non-primitives from an object
export function onlyPrimitives(object) {
    if (!_.isObject(object)) return {};
    let result = {};
    for (const [key, value] of Object.entries(object)) {
        if (!_.isObject(value) && !_.isArray(value)) result[key] = value;
    }
    return result;
}