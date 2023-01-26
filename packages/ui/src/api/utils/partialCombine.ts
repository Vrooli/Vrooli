import { exists } from "@shared/utils";
import { DeepPartialBooleanWithFragments, GqlPartial, NonMaybe } from "types";
import { findSelection } from "./findSelection";

/**
 * Recursively combines two DeepPartialBooleanWithFragments objects.
 * @param a The first object.
 * @param b The second object.
 * @param lastDefine The __define field most recently encountered in the recursion, if any. 
 * Used to rename __union and __use keys to avoid conflicts.
 * @returns A new object with the combined fields of a and b, 
 * and its __define field containing a unique set of all fragments from a and b.
 */
export const partialCombine = <T extends { __typename: string }>(
    a: DeepPartialBooleanWithFragments<NonMaybe<T>>,
    b: DeepPartialBooleanWithFragments<NonMaybe<T>>,
    lastDefine?: { [key: string]: [GqlPartial<any>, 'common' | 'full' | 'list' | 'nav'] }
): DeepPartialBooleanWithFragments<NonMaybe<T>> => {
    // Initialize objects to store fragments. We use this to avoid duplicates.
    // Fragments are keyed by their object type (e.g. Project, Routine) and selection type (e.g. list, full)
    const fragmentsByShape: { [key: string]: GqlPartial<any> } = {};
    // Initialize object to store the combined fields
    const combined: DeepPartialBooleanWithFragments<NonMaybe<T>> = {};
    // Combine top-level keys so we can iterate over them
    const keys = new Set([...Object.keys(a), ...Object.keys(b)]);
    console.log('partialcombine keys set', a.__typename, keys);
    // If the key is __define (i.e. specifies fragments), add each fragment in a and b to the combined object
    // We do this first so that the fragments are available in "combined" for use in __union and __use
    let define = { ...(a.__define ?? {}), ...(b.__define ?? {}) };
    if (Object.keys(define).length > 0) {
        // Loop through values in __define. We ignore the keys because they are only important when we 
        // need to convert the object to an actual gql string (i.e. the final step)
        for (const [shapeObj, selectionType] of Object.values(define)) {
            // Determine actual selection type based on available fields in the object
            // Ideally the specified selection type is available, so this is just in case
            const actualSelectionType = findSelection(shapeObj, selectionType);
            // If the fragment doesn't exist in the combined object, add it
            if (fragmentsByShape[`${shapeObj.__typename}_${actualSelectionType}`] === undefined) {
                fragmentsByShape[`${shapeObj.__typename}_${actualSelectionType}`] = shapeObj;
            }
        }
    }
    // If define is empty and lastDefine is defined, use lastDefine
    else if (Object.keys(define).length === 0 && exists(lastDefine)) {
        define = lastDefine;
    }
    // Iterate over the keys
    for (const key of keys) {
        // Skip __define because we handle it separately
        if (key === '__define') continue;
        // If the value is an object with key __union, rename each field in the union to `${objectType}_${selectionType}`. 
        // This will ensure that it is unique across all objects.
        else if (exists(a[key]?.__union) || exists(b[key]?.__union)) {
            console.log('key is __union!!!', key, a, b)
            // If the __union field doesn't exist in the combined object, add it
            if (combined[key] === undefined) {
                combined[key] = { __union: {} };
            }
            // If the __union field exists in the combined object, add the fields from the __union field in a
            if (exists(a[key]?.__union)) {
                for (const [unionKey, value] of Object.entries(a[key].__union)) {
                    // If value is a string or number, it must be a key for a fragment in the __define field. 
                    if (typeof value === 'string' || typeof value === 'number') {
                        // Rename the field to `${objectType}_${selectionType}`
                        const [shapeObj, selectionType] = define![value];
                        combined[key].__union![unionKey] = `${shapeObj.__typename}_${selectionType}`;
                    }
                    // Otherwise (i.e. its a [possibly lazy] object), if not in b, add as-is to the combined object
                    else if (!exists(b[key]?.__union![unionKey])) {
                        console.log('union adding as-is 1', key, unionKey, value);
                        combined[key].__union![unionKey] = value;
                    }
                }
            }
            // If the __union field exists in the combined object, add the fields from the __union field in b
            if (exists(b[key]?.__union)) {
                for (const [unionKey, value] of Object.entries(b[key].__union)) {
                    // If already in combined, skip
                    if (exists(combined[key].__union![unionKey])) continue;
                    // If value is a string or number, it must be a key for a fragment in the __define field. 
                    if (typeof value === 'string' || typeof value === 'number') {
                        // Rename the field to `${objectType}_${selectionType}`
                        const [shapeObj, selectionType] = define![value];
                        combined[key].__union![unionKey] = `${shapeObj.__typename}_${selectionType}`;
                    }
                    // Otherwise (i.e. its a [possibly lazy] object), if not in a, add as-is to the combined object
                    else if (!exists(a[key]?.__union![unionKey])) {
                        console.log('union adding as-is 2', key, unionKey, value)
                        combined[key].__union![unionKey] = value;
                    }
                    // Otherwise, it must also be in a, so combine the values
                    else {
                        console.log('union partial combineeeeeee');
                        combined[key].__union![unionKey] = partialCombine(a[key].__union![unionKey], b[key].__union![unionKey], define);
                    }
                }
            }
        }
        // If the value is an object with key __use (i.e. references a fragment), replace value with `${objectType}_${selectionType}`. 
        // This will ensure that it is unique across all objects.
        else if (exists(a[key]?.__use) || exists(b[key]?.__use)) { 
            // This is a single value instead of an object, so logic is much simpler than __union
            const [shapeObj, selectionType] = define![(a[key]?.__use ?? b[key]?.__use)!];
            combined[key] = { __typename: key, __use: `${shapeObj.__typename}_${selectionType}` };
        }
        // Otherwise, combine the values of the key
        else {
            // If the key doesn't exist in one of the objects, add it to the combined object
            if (a[key] === undefined) {
                combined[key] = b[key];
            }
            else if (b[key] === undefined) {
                combined[key] = a[key];
            }
            // If the key exists in both objects, check if it's a boolean
            else if (typeof a[key] === 'boolean') {
                combined[key] = true
            }
            // Otherwise, assume both are objects and recursively combine them
            else {
                combined[key] = partialCombine(a[key], b[key]);
            }
        }
    }
    // Set the __define field of the combined object
    combined.__define = Object.fromEntries(Object.entries(fragmentsByShape).map(([key, value]) => {
        const selectionType = key.split('_')[1] as 'common' | 'full' | 'list' | 'nav';
        return [key, [value, selectionType]];
    }));
    // Return the combined object
    return combined;
}