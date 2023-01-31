/**
 * Functions for manipulating state objects
 */
import { exists, isObject } from "@shared/utils";

/**
 * Grabs data from an object using dot notation (ex: 'parent.child.property')
 */
export const valueFromDot = (object, notation) => {
    function index(object, i) { return object[i] }
    if (!object || !notation) return null;
    return notation.split('.').reduce(index, object);
}

export const arrayValueFromDot = (object, notation, index) => {
    const value = valueFromDot(object, notation);
    if (!value || !Array.isArray(value) || index <= 0 || value.length >= index) return null;
    return value[index];
}

/**
 * Maps the keys of an object to dot notation
 */
export function convertToDot(obj, parent = [], keyValue = {}) {
    for (let key in obj) {
        let keyPath: any = [...parent, key];
        if (obj[key] !== null && typeof obj[key] === 'object') {
            Object.assign(keyValue, convertToDot(obj[key], keyPath, keyValue));
        } else {
            keyValue[keyPath.join('.')] = obj[key];
        }
    }
    return keyValue;
}

/**
 * Removes the first level of all strings in a dot notation array 
 * (e.g. ['parent.child.property', 'parent'] => ['child.property'])
 * @param notationArray Array of dot notation strings 
 * @returns Array of strings with the first level removed
 */
export const removeFirstLevel = (notationArray: string[]) => notationArray.map(s => s.split('.').slice(1).join('.')).filter(s => s.length > 0);

/**
 * Checks if any of the specified fields in an object have been changed. 
 * If no fields are specified, checks if any fields have been changed.
 * @param original The original object
 * @param updated The updated object
 * @param fields The fields to check for changes
 */
export function hasObjectChanged(original: any, updated: any, fields: string[] = []): boolean {
    if (!updated) return false;
    if (!original) return true;
    const fieldsToCheck = fields.length > 0 ? fields : Object.keys(original);
    for (let i = 0; i < fieldsToCheck.length; i++) {
        const field = fieldsToCheck[i];
        // If array, check if any values have changed
        if (Array.isArray(original[field])) {
            // Check lengths first
            if (original[field].length !== updated[field].length) return true;
            // Check if any values have changed
            for (let j = 0; j < original[field].length; j++) {
                if (hasObjectChanged(original[field][j], updated[field][j])) return true;
            }
        }
        // If object, call hasChanged on it
        else if (isObject(original[field])) {
            if (hasObjectChanged(original[field], updated[field])) return true;
        }
        // Otherwise, check if the values have changed
        else if (original[field] !== updated[field]) return true;
    }
    return false;
}

/**
 * Grabs and shapes strings from an object
 * @param object The object to grab strings from
 * @param fields The fields to grab strings from
 * @param prefix A prefix to add to the beginning of each string
 * @returns object with the specified fields. Each value is the original 
 * value if it was already a string, or '' if it was null or undefined. 
 * If a prefix was specified, it is added to the beginning of each field name, 
 * with the letter after the prefix capitalized.
 */
export const stringSpread = <
    Object extends { [key: string]: any },
    Fields extends keyof Object & string,
    Prefix extends string = ''
>(
    object: Object | null | undefined,
    fields: readonly Fields[],
    prefix?: Prefix
): (Prefix extends '' ? { [key in Fields]: string } : { [key in Fields as `${Prefix}${Capitalize<key>}`]: string }) => {
    const spread: any = {};
    fields.forEach(field => {
        const value = object ? object[field] : '';
        spread[prefix ? `${prefix}${field.charAt(0).toUpperCase()}${field.slice(1)}` : field] = typeof value === 'string' ? value : '';
    });
    return spread;
}

/**
 * Grabs and shapes booleans from an object
 * @param object The object to grab booleans from
 * @param fields The fields to grab booleans from
 * @param prefix A prefix to add to the beginning of each string
 * @returns object with the specified fields. Each value is the original 
 * value if it was already a boolean, or false if it was null or undefined.
 * If a prefix was specified, it is added to the beginning of each field name,
 * with the letter after the prefix capitalized.
 */
export const booleanSpread = <
    Object extends { [key: string]: any },
    Fields extends keyof Object & string,
    Prefix extends string = ''
>(
    object: Object | null | undefined,
    fields: readonly Fields[],
    prefix?: Prefix
): (Prefix extends '' ? { [key in Fields]: boolean } : { [key in Fields as `${Prefix}${Capitalize<key>}`]: boolean }) => {
    const spread: any = {};
    fields.forEach(field => {
        const value = object ? object[field] : false;
        spread[prefix ? `${prefix}${field.charAt(0).toUpperCase()}${field.slice(1)}` : field] = typeof value === 'boolean' ? value : false;
    });
    return spread;
}