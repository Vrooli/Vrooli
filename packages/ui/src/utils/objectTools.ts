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
        console.log('keeeyeyeyeyeye', key, value);
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
                console.log('array loop', i, curr)
                const changedValue = _.isObject(curr) ? formatForCreate(curr) : curr;
                console.log('array loop after changedvalie', changedValue)
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
 * - deletes - existing objects which are being deleted (e.g. deleting a node from a routine)
 * NOTE: By default, missing relationships are assumed to be disconnections. You must specify the relationship in 
 * the "treatLikeDeletes" parameter to treat them as deletes.
 * @param original - The orginal object fetched from the database, in query return format. Needed to exclude fields in the update object that 
 * haven't actually been updated
 * @param updated - The updated object, or partial update changes (which is a superset of the original)
 * @param treatLikeConnects - Array of field names (supports dot notation) that should be treated as connections (i.e. if object has id, ignore extra fields)
 * @param treatLikeDeletes - Array of field names (supports dot notation) that should be treated as deletes if not present in the update object
 */
export const formatForUpdate = <S, T>(
    original: Partial<S> | null | undefined, 
    updated: Partial<T>, 
    treatLikeConnects: string[] = [],
    treatLikeDeletes: string[] = [],
    treatLikeCreates: string[] = []
): Partial<T> => {
    // Create return object
    let changed = {};
    // Create child treatLike arrays
    const childTreatLikeConnects = treatLikeConnects.map(s => s.split('.').slice(1).join('.')).filter(s => s.length > 0);
    const childTreatLikeDeletes = treatLikeDeletes.map(s => s.split('.').slice(1).join('.')).filter(s => s.length > 0);
    // Helper method to add to arrays which might not exist
    const addToChangedArray = (key: string, value) => {
        if (Array.isArray(changed[key])) {
            if (Array.isArray(value)) changed[key] = [...changed[key], ...value];
            else changed[key] = [...changed[key], value];
        }
        else {
            if (Array.isArray(value)) changed[key] = value;
            else changed[key] = [value];
        }
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
                // If the current value is equal to the original value, skip it
                if (originalValue && _.isEqual(originalValue, curr)) continue;
                // Find the changed value at this index
                const changedValue = _.isObject(curr) ? formatForUpdate(originalValue, curr, childTreatLikeConnects, childTreatLikeDeletes) : curr;
                // Check if create (i.e does not contain an id)
                if (!Boolean(curr.id)) {
                    addToChangedArray(`${key}Create`, changedValue);
                }
                // Check if connection (i.e. is an object with only an id, or simply an id, or in treatLikeConnects),
                else if (typeof curr === 'string' || (Object.keys(curr).length === 1 && curr.id) || 
                (treatLikeConnects.includes(key) && Boolean(curr?.id))) {
                    addToChangedArray(`${key}Connect`, curr.id);
                }
                // Check if update. This can only be true if the object has an ID
                else if (Boolean(curr?.id)) {
                    // Loop through original values and find the one with the same ID
                    let originalValue;
                    if (original && Array.isArray(original[key])) {
                        originalValue = original[key].find(o => o.id === curr.id);
                    }
                    // If the original value is not found, treat it as a create
                    if (!originalValue) {
                        console.log('original value not found', key, changedValue, original && original[key])
                        addToChangedArray(`${key}Create`, changedValue);
                    }
                    // Otherwise, treat it as an update
                    else {
                        console.log('treated as update', key, changedValue, original && original[key])
                        addToChangedArray(`${key}Update`, changedValue);
                    }
                }
                // Shouldn't hit this point under normal circumstances. But if you do,
                // add the value to the changed array
                else addToChangedArray(key, changedValue);
            }
            // Loop through original values and determine which are disconnections
            if (original && Array.isArray(original[key])) {
                console.log('in original array check', key, original);
                for (let i = 0; i < original[key].length; i++) {
                    const curr = original[key][i];
                    // If the current value is an object, try to find the changed value
                    let changedValue = value.find(o => o.id === curr.id);
                    console.log('checking original', curr, changedValue)
                    // If the changed value is not found, it must have been deleted/disconnected
                    if (!changedValue) {
                        if (treatLikeDeletes && treatLikeDeletes.includes(key)) addToChangedArray(`${key}Delete`, curr.id);
                        else addToChangedArray(`${key}Disconnect`, curr.id);
                    }
                }
            }
            else console.log('no original array', key, original);
        }
        // If the value is an object
        else if (_.isObject(value)) {
            const curr: any = value;
            // Try to find the original value
            let originalValue = original ? original[key] : undefined;
            // Find the changed value
            const changedValue = formatForUpdate(originalValue, curr, childTreatLikeConnects, childTreatLikeDeletes);
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
                console.log('chedking is updateeeeeeeeee', key, changedValue, originalValue)
                // Only an update if the original value is found
                if (originalValue) changed[`${key}Update`] = changedValue;
                // Only a create if key not in treatLikeConnects
                else if (treatLikeConnects.includes(key) && changedValue.hasOwnProperty('id')) changed[`${key}Connect`] = (changedValue as any).id;
                else changed[`${key}Create`] = changedValue;
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

/**
 * Retrieves a value from an object's translations
 * @param obj The object to retrieve the value from
 * @param field The field to retrieve the value from
 * @param languages The languages the user is requesting
 * @param showAny If true, will default to returning the first language if no value is found
 * @returns The value of the field in the object's translations
 */
export const getTranslation = (obj: any, field: string, languages: readonly string[], showAny: boolean = true): any => {
    if (!obj || !obj.translations) return undefined;
    // Loop through translations
    for (const translation of obj.translations) {
        // If this translation is one of the languages we're looking for, check for the field
        if (languages.includes(translation.language)) {
            if (translation[field]) return translation[field];
        }
    }
    if (showAny && obj.translations.length > 0) return obj.translations[0][field];
    // If we didn't find a translation, return undefined
    return undefined;
}