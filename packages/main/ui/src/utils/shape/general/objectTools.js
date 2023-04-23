import { exists, isObject } from "@local/utils";
export const valueFromDot = (object, notation) => {
    function index(obj, i) {
        return exists(obj) ? obj[i] : null;
    }
    if (!object || !notation)
        return null;
    const value = notation.split(".").reduce(index, object);
    return exists(value) ? value : null;
};
export const arrayValueFromDot = (object, notation, index) => {
    const value = valueFromDot(object, notation);
    if (!value || !Array.isArray(value) || index <= 0 || value.length >= index)
        return null;
    return value[index];
};
export function convertToDot(obj, parent = [], keyValue = {}) {
    for (const key in obj) {
        const keyPath = [...parent, key];
        if (obj[key] !== null && typeof obj[key] === "object") {
            Object.assign(keyValue, convertToDot(obj[key], keyPath, keyValue));
        }
        else {
            keyValue[keyPath.join(".")] = obj[key];
        }
    }
    return keyValue;
}
export const removeFirstLevel = (notationArray) => notationArray.map(s => s.split(".").slice(1).join(".")).filter(s => s.length > 0);
export function hasObjectChanged(original, updated, fields = []) {
    if (!updated)
        return false;
    if (!original)
        return true;
    const fieldsToCheck = fields.length > 0 ? fields : Object.keys(original);
    for (let i = 0; i < fieldsToCheck.length; i++) {
        const field = fieldsToCheck[i];
        if (Array.isArray(original[field])) {
            if (original[field].length !== updated[field].length)
                return true;
            for (let j = 0; j < original[field].length; j++) {
                if (hasObjectChanged(original[field][j], updated[field][j]))
                    return true;
            }
        }
        else if (isObject(original[field])) {
            if (hasObjectChanged(original[field], updated[field]))
                return true;
        }
        else if (original[field] !== updated[field])
            return true;
    }
    return false;
}
export const stringSpread = (object, fields, prefix) => {
    const spread = {};
    fields.forEach(field => {
        const value = object ? object[field] : "";
        spread[prefix ? `${prefix}${field.charAt(0).toUpperCase()}${field.slice(1)}` : field] = typeof value === "string" ? value : "";
    });
    return spread;
};
export const booleanSpread = (object, fields, prefix) => {
    const spread = {};
    fields.forEach(field => {
        const value = object ? object[field] : false;
        spread[prefix ? `${prefix}${field.charAt(0).toUpperCase()}${field.slice(1)}` : field] = typeof value === "boolean" ? value : false;
    });
    return spread;
};
//# sourceMappingURL=objectTools.js.map