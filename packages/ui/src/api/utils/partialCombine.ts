import { exists } from "@shared/utils";
import { DeepPartialBooleanWithFragments, GqlPartial, MaybeLazy, NonMaybe } from "types";
import { findSelection } from "./findSelection";

/**
 * Calculates unique fragment name
 * @param partial GqlPartial object
 * @param actualSelectionType The actual selection type (i.e. the one that exists in the __define object), 
 * NOT necessarily the original selection specified
 */
const uniqueFragmentName = (partial: GqlPartial<any>, actualSelectionType: 'common' | 'full' | 'list' | 'nav') => `${partial.__typename}_${actualSelectionType}`;

/**
 * Adds fragments to a fragment object, and returns a new object with the combined fragments.
 * @param fragmentsByShape The fragment object to add to.
 * @param define The __define field of a DeepPartialBooleanWithFragments object.
 * @returns fragmentsByShape with the fragments from define added.
 */
const addFragments = <T extends { __typename: string }>(
    fragmentsByShape: { [key: string]: DeepPartialBooleanWithFragments<NonMaybe<T>> } | undefined,
    define: { [key: string]: [GqlPartial<any>, 'common' | 'full' | 'list' | 'nav'] } | undefined
): { [key: string]: DeepPartialBooleanWithFragments<NonMaybe<T>> } => {
    console.log('addfragments start', fragmentsByShape, define)
    // Initialize object to store the combined fragments
    let combinedFragments: { [key: string]: DeepPartialBooleanWithFragments<NonMaybe<T>> } = { ...fragmentsByShape } ?? {};
    // Loop through values in __define. We ignore the keys because they are only important when we 
    // need to convert the object to an actual gql string (i.e. the final step)
    for (const [shapeObj, selectionType] of Object.values(define ?? {})) {
        // Determine actual selection type based on available fields in the object
        // Ideally the specified selection type is available, so this is just in case
        const actualSelectionType = findSelection(shapeObj, selectionType);
        const actualKey = uniqueFragmentName(shapeObj, actualSelectionType);
        // If the fragment doesn't exist in the combined object, add it
        if (!exists(combinedFragments[actualKey])) {
            // Use partialCombine to recurse on GqlPartials, since it will combine nested fragments.
            // If selection is 'full' or 'list', and 'common' is defined, combine the two.
            console.log('addfragments before partialCombine', actualKey, actualSelectionType, shapeObj);
            const { __define, ...rest } = partialCombine(shapeObj[actualSelectionType]!, (['full', 'list'].includes(actualSelectionType) && exists(shapeObj.common)) ? shapeObj.common : {});
            combinedFragments[actualKey] = rest as DeepPartialBooleanWithFragments<NonMaybe<T>>;
            // Recurse on fragments found using partialCombine
            combinedFragments = addFragments(combinedFragments, __define);
        }
        // Also add the original name of the fragment to the combined object,
        // replacing any existing fragment with the same name.
        const originalKey = uniqueFragmentName(shapeObj, selectionType);
        combinedFragments[originalKey] = combinedFragments[actualKey];
    }
    console.log('addfragments end', combinedFragments);
    return combinedFragments;
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
    a: MaybeLazy<DeepPartialBooleanWithFragments<NonMaybe<T>>>,
    b: MaybeLazy<DeepPartialBooleanWithFragments<NonMaybe<T>>>,
    lastDefine: { [key: string]: [GqlPartial<any>, 'common' | 'full' | 'list' | 'nav'] } = {}
): DeepPartialBooleanWithFragments<NonMaybe<T>> => {
    const tempA = { ...(unlazy(a) ?? {}) };
    const tempB = { ...(unlazy(b) ?? {}) };
    console.log('partialcombine start ', tempA, tempB)
    // Unlazy a and b
    const first = unlazy(a);
    const second = unlazy(b);
    // Initialize objects to store fragments. We use this to avoid duplicates.
    // Fragments are keyed by their object type (e.g. Project, Routine) and selection type (e.g. list, full)
    let fragmentsByShape: { [key: string]: DeepPartialBooleanWithFragments<NonMaybe<T>> } = {};
    // Initialize object to store the combined fields
    const combined: DeepPartialBooleanWithFragments<NonMaybe<T>> = {};
    // Combine top-level keys so we can iterate over them
    const keys = new Set([...Object.keys(first), ...Object.keys(second)]);
    // Add fragments from first and second object to fragmentsByShape
    console.log('before addFragments a', first.__typename ?? second.__typename, first.__define, second.__define);
    fragmentsByShape = addFragments(fragmentsByShape, { ...(first.__define ?? {}), ...(second.__define ?? {}) });
    // Add fragments from lastDefine to fragmentsByShape
    console.log('before addFragments b', first.__typename ?? second.__typename, lastDefine)
    fragmentsByShape = addFragments(fragmentsByShape, lastDefine);
    // Create currDefine object to hold current fragments (which haven't been renamed yet). 
    // Prefer the __define field from first and second object (i.e. current) over lastDefine (i.e. parent, grandparent, etc.).
    let currDefine = { ...(first.__define ?? {}), ...(second.__define ?? {}) };
    if (Object.keys(currDefine).length === 0) currDefine = { ...lastDefine };
    console.log('currDefine', { ...currDefine }, { ...first }, { ...second });
    console.log('lastDefine', { ...lastDefine });
    console.log('partialcombine keys set', first.__typename ?? second.__typename, keys);
    // Iterate over the keys
    for (const key of keys) {
        // Skip __typename
        if (key === '__typename') continue;
        // Skip __define because we handle it separately
        if (key === '__define') continue;
        // If the value is an object with key __union, rename each field in the union to ensure uniqueness
        // This will ensure that it is unique across all objects.
        else if (exists(first[key]?.__union) || exists(second[key]?.__union)) {
            console.log('key is __union!!!', key, { ...first }, { ...second }, { ...currDefine })
            // If the __union field doesn't exist in the combined object, add it
            if (!exists(combined[key])) {
                console.log('combined[key] does not exist', key, combined[key], first[key], second[key], { ...currDefine })
                combined[key] = { __union: {} };
            }
            // If the __union field exists in the combined object, add the fields from the __union field in first object
            if (exists(first[key]?.__union)) {
                for (const [unionKey, value] of Object.entries(first[key].__union)) {
                    // Split __define (i.e. fragments) from the object so we can move them to shared fragments
                    const { __define, ...rest } = unlazy(value as any);
                    console.log('before addFragments c', first.__typename ?? second.__typename, __define)
                    fragmentsByShape = addFragments(fragmentsByShape, __define);
                    // If value is a string or number, it must be a key for a fragment in the __define field. 
                    if (typeof value === 'string' || typeof value === 'number') {
                        // Rename the field to ensure uniqueness
                        console.log('union rename symbol 1.a', key, unionKey, value)
                        if (!exists(currDefine[value])) continue;
                        const [shapeObj, selectionType] = currDefine[value];
                        combined[key].__union![unionKey] = uniqueFragmentName(shapeObj, selectionType);
                        console.log('union rename symbol 1.b', key, combined[key].__union![unionKey]);
                    }
                    // Otherwise (i.e. its a [possibly lazy] object), if not in b, add as-is to the combined object
                    else if (!exists(second[key]?.__union) || !exists(second[key]?.__union![unionKey])) {
                        console.log('union adding as-is 1.a', key, unionKey, value);
                        combined[key].__union![unionKey] = { ...rest };
                        console.log('union adding as-is 1.b', key, combined[key].__union![unionKey]);
                    }
                }
            }
            // If the __union field exists in the combined object, add the fields from the __union field in second object
            if (exists(second[key]?.__union)) {
                for (const [unionKey, value] of Object.entries(second[key].__union)) {
                    // If already in combined, skip
                    if (exists(combined[key].__union[unionKey])) continue;
                    // Split __define (i.e. fragments) from the object so we can move them to shared fragments
                    const { __define, ...rest } = unlazy(value as any);
                    console.log('before addFragments d', first.__typename ?? second.__typename, __define)
                    fragmentsByShape = addFragments(fragmentsByShape, __define);
                    // If value is a string or number, it must be a key for a fragment in the __define field. 
                    if (typeof value === 'string' || typeof value === 'number') {
                        console.log('union rename symbol 2.a', key, unionKey, value)
                        // Rename the field to ensure uniqueness
                        if (!exists(currDefine[value])) continue;
                        const [shapeObj, selectionType] = currDefine[value];
                        combined[key].__union![unionKey] = uniqueFragmentName(shapeObj, selectionType);
                        console.log('union rename symbol 2.b', key, combined[key].__union![unionKey], Object.keys(fragmentsByShape));
                    }
                    // Otherwise (i.e. its a [possibly lazy] object), if not in a, add as-is to the combined object
                    else if (!exists(first[key]?.__union) || !exists(second[key]?.__union![unionKey])) {
                        console.log('union adding as-is 2.a', key, unionKey, value)
                        combined[key].__union![unionKey] = { ...rest };
                        console.log('union adding as-is 2.b', key, combined[key].__union![unionKey]);
                    }
                    // Otherwise, it must also be in the first object, so combine the values
                    else {
                        console.log('union partial combineeeeeee a', key, unionKey, value)
                        combined[key].__union![unionKey] = { ...rest };
                        console.log('union partial combineeeeeee b', key, combined[key].__union![unionKey]);
                    }
                }
            }
        }
        // If the value is an object with key __use (i.e. references a fragment), replace value to ensure uniqueness 
        // This will ensure that it is unique across all objects.
        else if (exists(first[key]?.__use) || exists(second[key]?.__use)) {
            console.log('partialcombineyyy before use', key, first[key], second[key], { ...currDefine }, { ...lastDefine });
            // This is a single value instead of an object, so logic is much simpler than __union
            const useKey = first[key]?.__use ?? second[key]?.__use;
            if (!exists(currDefine[useKey])) continue;
            const [shapeObj, selectionType] = currDefine[useKey];
            combined[key] = { __typename: key, __use: uniqueFragmentName(shapeObj, findSelection(shapeObj, selectionType)) };
        }
        // Otherwise, combine the values of the key
        else {
            if (exists(first[key]) || exists(second[key])) {
                // If the key is a boolean, set it to true
                if (typeof first[key] === 'boolean' || typeof second[key] === 'boolean') {
                    combined[key] = true;
                }
                // Otherwise, assume both are (possibly lazy) objects and recursively combine them
                else {
                    console.log('partialcombineeeee before recurse', key, { ...first[key] }, { ...second[key] }, { ...currDefine }, { ...lastDefine });
                    // Split __define (i.e. fragments) from the object so we can move them to shared fragments
                    const { __define, ...rest } = partialCombine(first[key] ?? {}, second[key] ?? {}, currDefine);
                    console.log('before addFragments e', first.__typename ?? second.__typename, { ...__define })
                    fragmentsByShape = addFragments(fragmentsByShape, __define);
                    combined[key] = rest;
                }
            }
        }
    }
    // Set the __define field of the combined object
    combined.__define = Object.fromEntries(Object.entries(fragmentsByShape).map(([key, value]) => {
        const objectType = key.split('_')[0] as any;
        const selectionType = key.split('_')[1] as 'common' | 'full' | 'list' | 'nav';
        return [key, [{
            __typename: objectType,
            [selectionType]: value,
        }, selectionType]];
    }));
    console.log('partialcombine complete', first.__typename ?? second.__typename, combined)
    // Return the combined object
    return combined;
}