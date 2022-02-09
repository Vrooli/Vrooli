// Functions for manipulating state objects

import _ from "lodash";

// Grabs data from an object using dot notation (ex: 'parent.child.property')
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

// Maps the keys of an object to dot notation
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
 * Formats an object to prepare for a create mutation. Each relationship field is converted into
 * either a connection or create, depending on the existence of an id.
 * @param obj - The object being created
 */
export const formatForCreate = <T>(obj: Partial<T>): Partial<T> => {
    let changed = {};
    console.log('formatForCreate start', obj);
    // Helper method to add to arrays which might not exist
    const addToChangedArray = (key, value) => {
        console.log('add to changed array', key, value);
        if (Array.isArray(changed[key])) {
            console.log('is array');
            if (Array.isArray(value)) changed[key] = [...changed[key], ...value];
            else changed[key] = [...changed[key], value];
        }
        else changed[key] = Array.isArray(value) ? value : [value];
    }
    // Iterate through each field in the object
    for (const [key, value] of Object.entries(obj)) {
        // If the key is already one of the relationship mutation types, don't parse it
        if (['Connect', 'Create'].some(s => key.endsWith(s))) {
            addToChangedArray(key, value);
        }
        // If the value is an array
        else if (Array.isArray(value)) {
            console.log('formatforCreate is array', value)
            // Loop through changed values and determine which are connections and adds
            for (let i = 0; i < value.length; i++) {
                const curr = value[i];
                // Find the changed value at this index
                const changedValue = _.isObject(curr) ? formatForCreate(curr) : curr;
                // Check if create (i.e does not contain an id)
                if (changedValue && curr.id === undefined) {
                    addToChangedArray(`${key}Create`, changedValue);
                }
                // Check if connection type 1 (i.e. is an object with an id)
                else if (curr.id) {
                    addToChangedArray(`${key}Connect`, curr.id);
                }
                // Check if connection type 2 (i.e. simply an id)
                else if (typeof curr === 'string') {
                    addToChangedArray(`${key}Connect`, curr);
                }
                // Shouldn't hit this point under normal circumstances. But if you do,
                // add the value to the changed array
                else addToChangedArray(key, changedValue);
            }
        }
        // If the value is an object
        else if (_.isObject(value)) {
            console.log('formatforCreate in isobject', value)
            const curr: any = value;
            // Find the changed value
            const changedValue = formatForCreate(curr);
            console.log('changed value', changedValue)
            // Check if create (i.e does not contain an id)
            if (changedValue && curr.id === undefined) {
                console.log('create')
                changed[`${key}Create`] = changedValue;
            }
            // Check if connection (i.e. is an object with only an id)
            // NOTE: Does not support passing an id directly, like in the case of an array
            else if (curr.id) {
                console.log('connect')
                changed[`${key}Connect`] = curr.id;
            }
            // Shouldn't hit this point under normal circumstances. But if you do,
            // add the value to the changed array
            else changed[key] = changedValue;
        }
        // If the value is a primitive, add it to the changed object
        else changed[key] = value;
    }
    console.log('formatForCreate complete', changed);
    return changed;
}

/**
 * Compares an object against its updates. Returns fields that have changed.
 * For non-primitive values, splits field into 4 parts:
 * - connections - existing objects which are being newly connected to the object (e.g. associating an existing standard to a routine input)
 * - disconnections - existing objects which are being disconnected from the object (e.g. disassociating a standard from a routine input)
 * - creates - new objects which are being created (e.g. creating a new standard to a routine input)
 * - updates - existing objects which are being updated (e.g. changing a standard's name)
 * @param original - The orginal object fetched from the database, in query return format. Needed to exclude fields in the update object that 
 * haven't actually been updated
 * @param updated - The updated object, or partial update changes (which is a superset of the original)
 */
export const formatForUpdate = <S, T>(original: Partial<S> | null | undefined, updated: Partial<T>): Partial<T> => {
    let changed = {};
    console.log('formatForUpdate start', original, updated);
    // Helper method to add to arrays which might not exist
    const addToChangedArray = (key, value) => {
        if (Array.isArray(changed[key])) {
            if (Array.isArray(value)) changed[key] = [...changed[key], ...value];
            else changed[key] = [...changed[key], value];
        }
        else changed[key] = value;
    }
    // Iterate through each field in the updated object
    for (const [key, value] of Object.entries(updated)) {
        // If the key is already one of the 5 relationship mutation types (connect, disconnect, delete, create, update), don't parse it
        if (['Connect', 'Disconnect', 'Delete', 'Create', 'Update'].some(s => key.endsWith(s))) {
            addToChangedArray(key, value);
        }
        // If the value is identical to the original value (and not an id), skip it
        else if (key !== 'id' && original && _.isEqual(value, original[key])) continue;
        // If the value is an array
        else if (Array.isArray(value)) {
            // Loop through changed values and determine which are connections, creates, and updates
            for (let i = 0; i < value.length; i++) {
                const curr = value[i];
                // If the current value is an object, try to find the original value
                let originalValue;
                if (original && Array.isArray(original[key])) {
                    originalValue = original[key].find(o => o.id === curr.id);
                }
                // Find the changed value at this index
                const changedValue = _.isObject(curr) ? formatForUpdate(originalValue, curr) : curr;
                // Check if create (i.e does not contain an id)
                if (curr.id === undefined) {
                    addToChangedArray(`${key}Create`, changedValue);
                }
                // Check if connection (i.e. is an object with only an id, or simply an id)
                else if (typeof curr === 'string' || (Object.keys(curr).length === 1 && curr.id)) {
                    addToChangedArray(`${key}Connect`, curr.id);
                }
                // Check if update
                else if (curr?.id !== undefined) {
                    addToChangedArray(`${key}Update`, changedValue);
                }
                // Shouldn't hit this point under normal circumstances. But if you do,
                // add the value to the changed array
                else addToChangedArray(key, changedValue);
            }
            // Loop through original values and determine which are disconnections
            if (original && Array.isArray(original[key])) {
                for (let i = 0; i < original[key]; i++) {
                    const curr = original[key][i];
                    // If the current value is an object, try to find the changed value
                    let changedValue = value.find(o => o.id === curr.id);
                    // If the changed value is not found, it must have been deleted
                    if (!changedValue) {
                        addToChangedArray(`${key}Disconnect`, curr.id);
                    }
                }
            }
        }
        // If the value is an object
        else if (_.isObject(value)) {
            const curr: any = value;
            // Try to find the original value
            let originalValue = original ? original[key] : undefined;
            // Find the changed value
            const changedValue = formatForUpdate(originalValue, curr);
            // Check if disconnect
            if (!changedValue && originalValue) {
                changed[`${key}Disconnect`] = curr.id;
            }
            // Check if create (i.e does not contain an id)
            else if (changedValue && curr.id === undefined) {
                changed[`${key}Create`] = changedValue;
            }
            // Check if connection (i.e. is an object with only an id, or simply an id)
            else if (typeof curr === 'string' || (Object.keys(curr).length === 1 && curr.id)) {
                changed[`${key}Connect`] = curr.id;
            }
            // Check if update
            else if (curr.id !== undefined) {
                changed[`${key}Update`] = changedValue;
            }
            // Shouldn't hit this point under normal circumstances. But if you do,
            // add the value to the changed array
            else changed[key] = changedValue;
        }
        // If the value is a primitive, add it to the changed object
        else changed[key] = value;
    }
    console.log('formatForUpdate complete', changed);
    return changed;
}