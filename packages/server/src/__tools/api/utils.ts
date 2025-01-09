import { exists, mergeDeep } from "@local/shared";
import { ApiPartial, DeepPartialBooleanWithFragments, MaybeLazyAsync, NonMaybe, SelectionType } from "./types";

/**
 * Converts an ApiPartial-like structure into a plain nested object for REST/Prisma usage.
 */
export async function toObject<T extends ApiPartial<any>>(
    partial: T,
    selectionType: SelectionType,
    options?: {
        asSearch?: boolean,
        searchOverrides?: {
            edges?: Record<string, true>;
            pageInfo?: Record<string, true>;
        },
    },
): Promise<Record<string, any>> {
    // Convert using the 'rel' function
    let shaped = await rel(partial, selectionType);
    // If the 'asSearch' option is true, wrap the object in a 'search' object and convert again
    if (options?.asSearch) {
        shaped = {
            edges: options?.searchOverrides?.edges ?? {
                cursor: true,
                node: shaped,
            },
            pageInfo: options?.searchOverrides?.pageInfo ?? {
                endCursor: true,
                hasNextPage: true,
            },
        }
        shaped = await rel({ [selectionType]: shaped }, selectionType);
    }
    return shaped;
}

/**
 * Finds the best DeepPartialBooleanWithFragments 
 * to use in an ApiPartial object
 * @param obj The ApiPartial object to search
 * @param selection The preferred selection to use
 * @returns The preferred selection if it exists, otherwise the best selection
 */
export function findSelection(
    obj: ApiPartial<any>,
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
        throw new Error(`Could not determine actual selection type for '${obj}' '${selection}'`);
    }

    return result;
}

/**
 * Recursively shapes an API selection object to remove duplicate fragments and lazy fields.
 * @param selection The selection to shape
 * @returns A properly-shaped selection object.
 */
export async function partialShape<T extends object>(
    selection: MaybeLazyAsync<DeepPartialBooleanWithFragments<NonMaybe<T>>>,
): Promise<DeepPartialBooleanWithFragments<NonMaybe<T>>> {
    // Initialize result
    const result: DeepPartialBooleanWithFragments<NonMaybe<T>> = {};
    // Unlazy the selection
    const data = await unlazyDeep(selection);
    // Iterate over the keys
    for (const key of Object.keys(data)) {
        // If the key is a boolean, set it to true
        if (typeof data[key] === "boolean") {
            result[key] = true;
        }
        // Otherwise, assume it's an object and recursively combine
        else {
            const inner = await partialShape(data[key] ?? {});
            // If the key is __union, add each value of inner to the result directly (instead of nesting)
            if (key === "__union") {
                for (const unionKey in inner) {
                    result[unionKey] = inner[unionKey];
                }
            }
            // Otherwise, add the inner object to the result under the key
            else {
                result[key] = inner;
            }
        }
    }
    // Return the combined object
    return result;
}

/**
 * Adds a relation to an GraphQL selection set, while optionally omitting one or more fields.
 * @param partial The ApiPartial object containing the selection set
 * @param selectionType Which selection from ApiPartial to use
 * @param exceptions Exceptions object containing fields to omit. Supports dot notation.
 */
export async function rel<
    Partial extends ApiPartial<any>,
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
    return selectionData as any;
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

