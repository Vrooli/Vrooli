import { exists } from "@shared/utils";
import { DeepPartialBooleanWithFragments, GqlPartial, NonMaybe } from "types";
import { findSelection } from "./findSelection";

/**
 * Calculates unique fragment name
 */
const uniqueFragmentName = (partial: GqlPartial<any>, selectionType: 'common' | 'full' | 'list' | 'nav') => `${partial.__typename}_${selectionType}`;

/**
 * Adds fragments to a fragment object, and returns a new object with the combined fragments.
 * @param fragmentsByShape The fragment object to add to.
 * @param define The __define field of a DeepPartialBooleanWithFragments object.
 * @returns fragmentsByShape with the fragments from define added.
 */
const addFragments = (
    fragmentsByShape: { [key: string]: GqlPartial<any> } | undefined,
    define: { [key: string]: [GqlPartial<any>, 'common' | 'full' | 'list' | 'nav'] } | undefined
): { [key: string]: GqlPartial<any> } => {
    // Initialize object to store the combined fragments
    const combinedFragments: { [key: string]: GqlPartial<any> } = {};
    // Loop through values in __define. We ignore the keys because they are only important when we 
    // need to convert the object to an actual gql string (i.e. the final step)
    for (const [shapeObj, selectionType] of Object.values(define ?? {})) {
        // Determine actual selection type based on available fields in the object
        // Ideally the specified selection type is available, so this is just in case
        const actualSelectionType = findSelection(shapeObj, selectionType);
        // If the fragment doesn't exist in the combined object, add it
        if (combinedFragments[uniqueFragmentName(shapeObj, actualSelectionType)] === undefined) {
            combinedFragments[uniqueFragmentName(shapeObj, actualSelectionType)] = shapeObj;
        }
    }
    // If fragmentsByShape is undefined, return combinedFragments
    if (fragmentsByShape === undefined) {
        return combinedFragments;
    }
    // Otherwise, combine the two objects
    else {
        return { ...fragmentsByShape, ...combinedFragments };
    }
}

/**
 * Unlazies an object
 */
const unlazy = <T extends {}>(obj: T | (() => T)): T => typeof obj === 'function' ? (obj as () => T)() : obj;

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
    lastDefine: { [key: string]: [GqlPartial<any>, 'common' | 'full' | 'list' | 'nav'] } = {}
): DeepPartialBooleanWithFragments<NonMaybe<T>> => {
    // Initialize objects to store fragments. We use this to avoid duplicates.
    // Fragments are keyed by their object type (e.g. Project, Routine) and selection type (e.g. list, full)
    let fragmentsByShape: { [key: string]: GqlPartial<any> } = {};
    // Initialize object to store the combined fields
    const combined: DeepPartialBooleanWithFragments<NonMaybe<T>> = {};
    // Combine top-level keys so we can iterate over them
    const keys = new Set([...Object.keys(a), ...Object.keys(b)]);
    console.log('partialcombine keys set', a.__typename, keys);
    // Add fragments from a and b to fragmentsByShape
    fragmentsByShape = addFragments(fragmentsByShape, { ...(a.__define ?? {}), ...(b.__define ?? {}) });
    // Add fragments from lastDefine to fragmentsByShape
    fragmentsByShape = addFragments(fragmentsByShape, lastDefine);
    // Create currDefine object to hold current fragments (which haven't been renamed yet). 
    // Prefer the __define field from a and b (i.e. current) over lastDefine (i.e. parent, grandparent, etc.).
    let currDefine = { ...(a.__define ?? {}), ...(b.__define ?? {}) };
    if (Object.keys(currDefine).length === 0) currDefine = lastDefine;
    // Iterate over the keys
    for (const key of keys) {
        // Skip __define because we handle it separately
        if (key === '__define') continue;
        // If the value is an object with key __union, rename each field in the union to ensure uniqueness
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
                        // Rename the field to ensure uniqueness
                        const [shapeObj, selectionType] = currDefine![value];
                        combined[key].__union![unionKey] = uniqueFragmentName(shapeObj, selectionType);
                    }
                    // Otherwise (i.e. its a [possibly lazy] object), if not in b, add as-is to the combined object
                    else if (!exists(b[key]?.__union![unionKey])) {
                        console.log('union adding as-is 1', key, unionKey, value);
                        // Split __define (i.e. fragments) from the object so we can move them to shared fragments
                        const { __define, ...rest } = unlazy(value as any);
                        fragmentsByShape = addFragments(fragmentsByShape, __define);
                        combined[key].__union![unionKey] = rest;
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
                        // Rename the field to ensure uniqueness
                        const [shapeObj, selectionType] = currDefine![value];
                        combined[key].__union![unionKey] = uniqueFragmentName(shapeObj, selectionType);
                    }
                    // Otherwise (i.e. its a [possibly lazy] object), if not in a, add as-is to the combined object
                    else if (!exists(a[key]?.__union![unionKey])) {
                        console.log('union adding as-is 2', key, unionKey, value)
                        // Split __define (i.e. fragments) from the object so we can move them to shared fragments
                        const { __define, ...rest } = unlazy(value as any);
                        fragmentsByShape = addFragments(fragmentsByShape, __define);
                        combined[key].__union![unionKey] = rest;
                    }
                    // Otherwise, it must also be in a, so combine the values
                    else {
                        console.log('union partial combineeeeeee');
                        // Split __define (i.e. fragments) from the object so we can move them to shared fragments
                        const { __define, ...rest } = partialCombine(a[key].__union![unionKey], b[key].__union![unionKey], currDefine);
                        fragmentsByShape = addFragments(fragmentsByShape, __define);
                        combined[key].__union![unionKey] = rest;
                    }
                }
            }
        }
        // If the value is an object with key __use (i.e. references a fragment), replace value to ensure uniqueness 
        // This will ensure that it is unique across all objects.
        else if (exists(a[key]?.__use) || exists(b[key]?.__use)) {
            // This is a single value instead of an object, so logic is much simpler than __union
            const [shapeObj, selectionType] = currDefine![(a[key]?.__use ?? b[key]?.__use)!];
            combined[key] = { __typename: key, __use: uniqueFragmentName(shapeObj, selectionType) };
        }
        // Otherwise, combine the values of the key
        else {
            // TODO for morning: this code below before the commented-out chunk is new, and doesn't work yet. 
            // What it's trying to do is fix the issue where the __define field is not being added to the combined object
            // If the key exists in both objects
            if (exists(a[key]) && exists(b[key])) {
                console.log('pc 1', key)
                // If the key is a boolean, set it to true
                if (typeof a[key] === 'boolean') {
                    console.log('pc 1.1')
                    combined[key] = true;
                }
                // Otherwise, assume both are (possibly lazy) objects and recursively combine them
                else {
                    console.log('pc 1.2')
                    // Split __define (i.e. fragments) from the object so we can move them to shared fragments
                    const { __define, ...rest } = partialCombine(unlazy(a[key]), unlazy(b[key]), currDefine);
                    fragmentsByShape = addFragments(fragmentsByShape, __define);
                    combined[key] = rest;
                }
            }
            // Check if the key exists in one of the objects
            const anyValue = a[key] ?? b[key];
            if (exists(anyValue)) {
                console.log('pc 2', key, anyValue)
                // If the key is a boolean, set it to true
                if (typeof anyValue === 'boolean') {
                    console.log('pc 2.1')
                    combined[key] = true;
                }
                // Otherwise, remove the __define field and add it to the shared fragments
                else {
                    console.log('pc 2.2')
                    const { __define, ...rest } = unlazy(anyValue);
                    fragmentsByShape = addFragments(fragmentsByShape, __define);
                    combined[key] = rest;
                }
            }
            // // If the key doesn't exist in one of the objects, add it to the combined object
            // if (a[key] === undefined) {
            //     combined[key] = b[key];
            // }
            // else if (b[key] === undefined) {
            //     combined[key] = a[key];
            // }
            // // If the key exists in both objects, check if it's a boolean
            // else if (typeof a[key] === 'boolean') {
            //     combined[key] = true
            // }
            // // Otherwise, assume both are objects and recursively combine them
            // else {
            //     // Split __define (i.e. fragments) from the object so we can move them to shared fragments
            //     const { __define, ...rest } = partialCombine(a[key], b[key], currDefine);
            //     fragmentsByShape = addFragments(fragmentsByShape, __define);
            //     combined[key] = rest;
            // }
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