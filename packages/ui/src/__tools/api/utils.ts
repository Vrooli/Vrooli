import { exists, mergeDeep } from "@local/shared";
import { MaybeLazyAsync, NonMaybe } from "../../types";
import { DeepPartialBooleanWithFragments, GqlPartial, SelectionType } from "./types";

/**
 * Converts a GqlPartial-like structure into a plain nested object for REST/Prisma usage.
 */
export async function toObject<T extends GqlPartial<any>>(
    partial: T,
    selectionType: SelectionType
): Promise<Record<string, any>> {
    // 1. Use `rel` to merge partials for the desired selection type
    let shaped = await rel(partial, selectionType);

    // 2. Remove leftover GraphQL fields like __typename, __selectionType
    shaped = await stripGraphQLKeys(shaped);

    // 3. Convert any `__union` data into a more direct object structure (if you want).
    shaped = transformUnion(shaped);

    return shaped;
}

/** Example of removing the special keys you used for GraphQL */
async function stripGraphQLKeys(obj: any): Promise<any> {
    if (typeof obj !== "object" || !obj) return obj;
    if (Array.isArray(obj)) {
        return Promise.all(obj.map(item => stripGraphQLKeys(item)));
    }
    // Remove __typename, __selectionType, etc.
    const { __typename, __selectionType, __define, __union, ...rest } = obj;
    for (const key of Object.keys(rest)) {
        rest[key] = await stripGraphQLKeys(rest[key]);
    }
    // If there's a __union, handle it separately
    if (__union) {
        rest["__union"] = await stripGraphQLKeys(__union);
    }
    return rest;
}

/** Example of turning __union into something more standard, if you choose */
function transformUnion(obj: any): any {
    if (!obj || typeof obj !== "object") return obj;
    if (Array.isArray(obj)) {
        return obj.map(transformUnion);
    }
    if (obj.__union) {
        // your choice how to represent it, e.g. a record keyed by type name
        // or an array of possible sub-objects, etc.
        for (const [typeName, value] of Object.entries(obj.__union)) {
            obj.__union[typeName] = transformUnion(value);
        }
    }
    for (const key of Object.keys(obj)) {
        if (key !== "__union") {
            obj[key] = transformUnion(obj[key]);
        }
    }
    return obj;
}

/**
 * Finds the best DeepPartialBooleanWithFragments 
 * to use in a GqlPartial object
 * @param obj The GqlPartial object to search
 * @param selection The preferred selection to use
 * @returns The preferred selection if it exists, otherwise the best selection
 */
export function findSelection(
    obj: GqlPartial<any>,
    selection: SelectionType,
): SelectionType {
    const selectionOrder: Record<SelectionType, SelectionType[]> = {
        common: ["common", "list", "full", "nav"],
        list: ["list", "common", "full", "nav"],
        full: ["full", "list", "common", "nav"],
        nav: ["nav", "common", "list", "full"],
    };

    // Find the first existing selection based on the preferred order
    const result = selectionOrder[selection].find(sel => exists(obj[sel]));

    if (!result) {
        throw new Error(`Could not determine actual selection type for '${obj.__typename}' '${selection}'`);
    }

    if (result !== selection) {
        console.warn(`Specified selection type '${selection}' for '${obj.__typename}' does not exist. Try using '${result}' instead.`);
    }

    return result;
}

/**
 * Adds fragments to a fragment object, and returns a new object with the combined fragments.
 * @param fragmentsByShape The fragment object to add to.
 * @param define The __define field of a DeepPartialBooleanWithFragments object.
 * @returns fragmentsByShape with the fragments from define added.
 */
function addFragments<T extends { __typename: string }>(
    fragmentsByShape: { [key: string]: DeepPartialBooleanWithFragments<NonMaybe<T>> } | undefined,
    define: { [key: string]: DeepPartialBooleanWithFragments<NonMaybe<T>> } | undefined,
): { [key: string]: DeepPartialBooleanWithFragments<NonMaybe<T>> } {
    // Initialize object to store the combined fragments
    let result: { [key: string]: DeepPartialBooleanWithFragments<NonMaybe<T>> } = { ...fragmentsByShape } ?? {};
    // Loop through values in __define. We ignore the keys because they are only important when we 
    // need to convert the object to an actual gql string (i.e. the final step)
    for (const partial of Object.values(define ?? {})) {
        // If __selectionType or __typename are not specified (which should always be for fragments), log error and skip
        const typename = partial.__typename;
        const selectionType = partial.__selectionType;
        if (!typename || !selectionType) {
            console.error(`Error: __selectionType or __typename is not defined for a fragment ${partial.__typename}`, partial);
            continue;
        }
        const actualKey = uniqueFragmentName(typename, selectionType);
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
export async function partialShape<T extends { __typename: string }>(
    selection: MaybeLazyAsync<DeepPartialBooleanWithFragments<NonMaybe<T>>>,
    lastDefine: { [key: string]: DeepPartialBooleanWithFragments<NonMaybe<T>> } = {},
): Promise<DeepPartialBooleanWithFragments<NonMaybe<T>>> {
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
        if (["__typename", "__define", "__selectionType"].includes(key)) continue;
        // If the key is __union, rename each field in the union to ensure uniqueness
        // NOTE: This case is only reached if the __union field is at the top level of the selection object, rather than 
        // nested within another property. Most of the time, the next case will be reached instead.
        if (key === "__union") {
            const resultUnion = result.__union ?? {};
            for (const [unionKey, value] of Object.entries(data.__union!)) {
                // If value is a string or number, it must be a key for a fragment in the __define field. 
                if (typeof value === "string" || typeof value === "number") {
                    // Rename the field to ensure uniqueness
                    const defineData = currDefine[value];
                    if (!defineData) continue;
                    resultUnion[unionKey] = uniqueFragmentName(defineData.__typename!, defineData.__selectionType!);
                }
                // Otherwise (i.e. its a [possibly lazy] object), add without fragments to the __union field
                else {
                    // Split __define (i.e. fragments) from the object so we can move them to shared fragments
                    const { __define, ...rest } = await unlazy(value as any);
                    uniqueFragments = addFragments(uniqueFragments, __define);
                    // Add the object to the __union field
                    resultUnion[unionKey] = { ...rest };
                }
            }
            result.__union = resultUnion;
        }
        // If the value is an object with key __union, rename each field in the union to ensure uniqueness
        // This will ensure that it is unique across all objects.
        else if (exists(data[key]?.__union)) {
            const resultUnion = result[key]?.__union ?? {};
            for (const [unionKey, value] of Object.entries(data[key].__union)) {
                // If value is a string or number, it must be a key for a fragment in the __define field. 
                if (typeof value === "string" || typeof value === "number") {
                    // Rename the field to ensure uniqueness
                    const defineData = currDefine[value];
                    if (!defineData) continue;
                    resultUnion[unionKey] = uniqueFragmentName(defineData.__typename!, defineData.__selectionType!);
                }
                // Otherwise (i.e. its a [possibly lazy] object), add without fragments to the __union field
                else {
                    // Split __define (i.e. fragments) from the object so we can move them to shared fragments
                    const { __define, ...rest } = await unlazy(value as any);
                    uniqueFragments = addFragments(uniqueFragments, __define);
                    // Add the object to the __union field
                    resultUnion[unionKey] = { ...rest };
                }
            }
            result[key] = { ...result[key], __union: resultUnion };
        }
        // If the value is an object with key __use (i.e. references a fragment), replace value to ensure uniqueness 
        else if (exists(data[key]?.__use)) {
            // This is a single value instead of an object, so logic is much simpler than __union
            const useKey = data[key].__use;
            if (exists(currDefine[useKey])) {
                const defineData = currDefine[useKey];
                if (!defineData) continue;
                result[key] = { __typename: key, __use: uniqueFragmentName(defineData.__typename!, defineData.__selectionType!) };
            }
            // If there are other keys in the object besides __use and __typename, add them to the result
            if (Object.keys(data[key]).filter(k => k !== "__use" && k !== "__typename").length > 0) {
                // Split __define (i.e. fragments) from the object so we can move them to shared fragments
                const { __define, __use, ...rest } = await unlazy(data[key] as any);
                uniqueFragments = addFragments(uniqueFragments, __define);
                // Add the object to the result
                result[key] = { ...result[key], ...rest };
            }
        }
        // Otherwise, combine the values of the key
        else {
            if (!exists(data[key])) continue;
            // If the key is a boolean, set it to true
            if (typeof data[key] === "boolean") {
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
    if (Object.keys(uniqueFragments).length > 0) result.__define = uniqueFragments;
    // Return the combined object
    return result;
}

/**
 * Adds a relation to an GraphQL selection set, while optionally omitting one or more fields.
 * @param partial The GqlPartial object containing the selection set
 * @param selectionType Which selection from GqlPartial to use
 * @param exceptions Exceptions object containing fields to omit. Supports dot notation.
 */
export async function rel<
    Partial extends GqlPartial<any>,
    Selection extends SelectionType,
    OmitField extends string | number | symbol,
>(
    partial: Partial,
    selectionType: Selection,
    exceptions?: { omit: OmitField | OmitField[] },
): Promise<DeepPartialBooleanWithFragments<any>> {
    const hasExceptions = exists(exceptions) && exists(exceptions.omit);
    // Find correct selection to use
    const actualSelectionType = findSelection(partial, selectionType);
    // Get selection data for the partial
    let selectionData = { ...partial[actualSelectionType]! };
    // Remove all exceptions. Supports dot notation.
    hasExceptions && removeValuesUsingDot(selectionData, ...(Array.isArray(exceptions.omit) ? exceptions.omit : [exceptions.omit]));
    // Shape selection data
    selectionData = await partialShape(selectionData);
    // If the selectiion type is 'full' or 'list', and the 'common' selection is defined, combine the two.
    if (["full", "list"].includes(actualSelectionType) && exists(partial.common)) {
        let commonData = partial.common;
        // Remove exceptions from common selection
        hasExceptions && removeValuesUsingDot(commonData, ...(Array.isArray(exceptions.omit) ? exceptions.omit : [exceptions.omit]));
        // Shape common selection
        commonData = await partialShape(commonData);
        // Merge common selection into selection data
        selectionData = mergeDeep(selectionData, commonData);
    }
    return { __typename: partial.__typename, __selectionType: actualSelectionType, ...selectionData } as any;
}

/**
 * Removes one or more properties from an object using dot notation (ex: 'parent.child.property'). 
 * NOTE 1: Supports lazy values, but removes the lazy part
 * NOTE 2: Modifies the original object
 */
export async function removeValuesUsingDot(obj: Record<string | number | symbol, any>, ...keys: (string | number | symbol)[]) {
    keys.forEach(async key => {
        const keyArr = typeof key === "string" ? key.split(".") : [key]; // split the key into an array of keys
        // loop through the keys, checking if each level is lazy-loaded
        let currentObject = obj;
        let currentKey;
        for (let i = 0; i < keyArr.length - 1; i++) {
            currentKey = keyArr[i];
            if (typeof currentObject[currentKey] === "function") {
                currentObject[currentKey] = await currentObject[currentKey]();
            }
            currentObject = currentObject[currentKey];
            if (!exists(currentObject)) break;
        }
        currentKey = keyArr[keyArr.length - 1];
        if (!exists(currentObject) || !exists(currentObject[currentKey])) return;
        delete currentObject[currentKey];
    });
}

/**
 * Helper function for generating a GraphQL selection of a 
 * paginated search result using a template literal.
 * 
 * @param partial A GqlPartial object for the object type being searched
 * @partial overrides Custom overrides for edges and pageInfo
 * @returns A new GqlPartial object like the one passed in, but wrapped 
 * in a paginated search result.
 */
export async function toSearch<
    GqlObject extends { __typename: string }
>(
    partial: GqlPartial<GqlObject>,
    overrides?: {
        edges?: Record<string, true>;
        pageInfo?: Record<string, true>;
    },
): Promise<[GqlPartial<any>, "list"]> {
    // Combine and remove fragments, so we can put them in the top level
    const { __define, ...node } = await rel(partial, "list");
    return [{
        __typename: `${partial.__typename}SearchResult`,
        list: {
            __define,
            edges: overrides?.edges ?? {
                cursor: true,
                node,
            },
            pageInfo: overrides?.pageInfo ?? {
                endCursor: true,
                hasNextPage: true,
            },
        },
    }, "list"];
}

/**
 * Format for unique fragment name
 */
export function uniqueFragmentName(typename: string, actualSelectionType: SelectionType) {
    return `${typename}_${actualSelectionType}`;
}

/**
 * Unlazies an object
 */
export async function unlazy<T extends object>(obj: T | (() => T) | (() => Promise<T>)): Promise<T> {
    return typeof obj === "function" ? await (obj as () => T)() : obj;
}

/**
 * Recursively unlazies an object
 * @param obj The object to unlazify
 * @returns The unlazified object
 */
export async function unlazyDeep<T extends object>(obj: T | (() => T) | (() => Promise<T>)): Promise<T> {
    const unlazyObj = await unlazy(obj);
    for (const key in unlazyObj) {
        const value = unlazyObj[key];
        if (Array.isArray(value)) {
            unlazyObj[key as any] = await Promise.all(value.map(unlazyDeep));
        }
        else if (typeof value === "function" || typeof value === "object") {
            unlazyObj[key as any] = await unlazyDeep(value as any);
        }
    }
    return unlazyObj;
}

