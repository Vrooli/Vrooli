import { exists } from "@shared/utils";
import { MaybeLazyAsync, NonMaybe } from "types";
import { DeepPartialBooleanWithFragments } from "../types";
import { uniqueFragmentName } from "./uniqueFragmentName";
import { unlazy, unlazyDeep } from "./unlazy";

/**
 * Adds fragments to a fragment object, and returns a new object with the combined fragments.
 * @param fragmentsByShape The fragment object to add to.
 * @param define The __define field of a DeepPartialBooleanWithFragments object.
 * @returns fragmentsByShape with the fragments from define added.
 */
const addFragments = <T extends { __typename: string }>(
    fragmentsByShape: { [key: string]: DeepPartialBooleanWithFragments<NonMaybe<T>> } | undefined,
    define: { [key: string]: DeepPartialBooleanWithFragments<NonMaybe<T>> } | undefined
): { [key: string]: DeepPartialBooleanWithFragments<NonMaybe<T>> } => {
    // Initialize object to store the combined fragments
    let result: { [key: string]: DeepPartialBooleanWithFragments<NonMaybe<T>> } = { ...fragmentsByShape } ?? {};
    // Loop through values in __define. We ignore the keys because they are only important when we 
    // need to convert the object to an actual gql string (i.e. the final step)
    for (const partial of Object.values(define ?? {})) {
        // If __selectionType or __typename are not specified (which should always be for fragments), log error and skip
        if (!exists(partial.__selectionType) || !exists(partial.__typename)) {
            console.error(`Error: __selectionType or __typename is not defined for a fragment ${partial.__typename}`, partial);
            continue;
        }
        const actualKey = uniqueFragmentName(partial.__typename!, partial.__selectionType!);
        // If fragment not in result, add it
        if (!exists(result[actualKey])) {
            const { __define, ...rest } = partial;
            result[actualKey] = rest as DeepPartialBooleanWithFragments<NonMaybe<T>>;
            // Add fragments from __define to result
            result = { ...result, ...__define } as any;
        }
    }
    return result;
}

/**
 * Recursively shapes a gql selection object to remove duplicate fragments and lazy fields.
 * @param selection The selection to shape
 * @param lastDefine The __define field most recently encountered in the recursion, if any. 
 * Used to rename __union and __use keys to avoid conflicts.
 * @returns A properly-shaped selection object.
 */
export const partialShape = async <T extends { __typename: string }>(
    selection: MaybeLazyAsync<DeepPartialBooleanWithFragments<NonMaybe<T>>>,
    lastDefine: { [key: string]: DeepPartialBooleanWithFragments<NonMaybe<T>> } = {}
): Promise<DeepPartialBooleanWithFragments<NonMaybe<T>>> => {
    // Initialize result
    const result: DeepPartialBooleanWithFragments<NonMaybe<T>> = {};
    // Unlazy the selection
    const data = await unlazyDeep(selection);
    // Initialize objects to store renamed fragments. We use this to avoid duplicates.
    // Fragments are keyed by their object type (e.g. Project, Routine) and selection type (e.g. list, full)
    let uniqueFragments: { [key: string]: DeepPartialBooleanWithFragments<NonMaybe<T>> } = {};
    // Add top-level fragments to uniqueFragments
    uniqueFragments = addFragments(uniqueFragments, data.__define as any);
    // Create currDefine object to hold current fragments (which haven't been renamed yet). 
    // Prefer the __define field from first and second object (i.e. current) over lastDefine (i.e. parent, grandparent, etc.).
    let currDefine: { [x: string]: DeepPartialBooleanWithFragments<NonMaybe<T>> } = { ...(data.__define ?? {}) } as any;
    if (Object.keys(currDefine).length === 0) currDefine = { ...lastDefine };
    // Iterate over the keys
    for (const key of Object.keys(data)) {
        // Skip __typename, __define, and __selectionType
        if (['__typename', '__define', '__selectionType'].includes(key)) continue;
        // If the value is an object with key __union, rename each field in the union to ensure uniqueness
        // This will ensure that it is unique across all objects.
        else if (exists(data[key]?.__union)) {
            if (!exists(data[key]?.__union)) continue;
            // Initialize __union field if it doesn't exist
            if (!exists(result[key])) {
                result[key] = { __union: {} };
            }
            for (const [unionKey, value] of Object.entries(data[key].__union)) {
                // If value is a string or number, it must be a key for a fragment in the __define field. 
                if (typeof value === 'string' || typeof value === 'number') {
                    // Rename the field to ensure uniqueness
                    if (!exists(currDefine[value])) continue;
                    const defineData = currDefine[value];
                    result[key].__union![unionKey] = uniqueFragmentName(defineData.__typename!, defineData.__selectionType!);
                }
                // Otherwise (i.e. its a [possibly lazy] object), add without fragments to the __union field
                else {
                    // Split __define (i.e. fragments) from the object so we can move them to shared fragments
                    const { __define, ...rest } = await unlazy(value as any);
                    uniqueFragments = addFragments(uniqueFragments, __define);
                    // Add the object to the __union field
                    result[key].__union![unionKey] = { ...rest };
                }
            }
        }
        // If the value is an object with key __use (i.e. references a fragment), replace value to ensure uniqueness 
        else if (exists(data[key]?.__use)) {
            // This is a single value instead of an object, so logic is much simpler than __union
            const useKey = data[key].__use;
            if (!exists(currDefine[useKey])) continue;
            const defineData = currDefine[useKey];
            result[key] = { __typename: key, __use: uniqueFragmentName(defineData.__typename!, defineData.__selectionType!) };
        }
        // Otherwise, combine the values of the key
        else {
            if (!exists(data[key])) continue;
            // If the key is a boolean, set it to true
            if (typeof data[key] === 'boolean') {
                result[key] = true;
            }
            // Otherwise, assume it's an object and recursively combine
            else {
                // Split __define (i.e. fragments) from the object so we can move them to shared fragments
                const { __define, ...rest } = await partialShape(data[key] ?? {}, currDefine);
                uniqueFragments = addFragments(uniqueFragments, __define as any);
                result[key] = rest;
            }
        }
    }
    // Set the __define field of the combined object
    if(Object.keys(uniqueFragments).length > 0) result.__define = uniqueFragments;
    // Return the combined object
    return result;
}