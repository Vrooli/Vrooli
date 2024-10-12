import { exists, mergeDeep } from "@local/shared";
import { MaybeLazyAsync, NonMaybe } from "../../types";
import { DeepPartialBooleanWithFragments, GqlPartial, SelectionType } from "./types";

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
 * Determines which fragments are needed for a given partial. 
 * Useful for filtering out fragments from omitted fields, since 
 * we have no better way to do this yet.
 * @param fragments list of [name, tag] for all fragments
 * @param partialTag the partial to check
 * @returns fragments list with omitted fragments removed
 */
export function fragmentsNeeded(
    fragments: [string, string][],
    partialTag: string,
) {
    function getDependentFragments(fragmentName: string, fragments: [string, string][]) {
        const fragmentTag = fragments.find(([name]) => name === fragmentName)?.[1];
        if (!fragmentTag) return [];

        const nestedFragments = fragmentTag.match(/\.\.\.(\w+)/g);
        if (!exists(nestedFragments)) return [];

        let dependentFragments: string[] = [];
        nestedFragments.forEach(fragment => {
            const name = fragment.replace("...", "");
            dependentFragments.push(name);
            dependentFragments = [...dependentFragments, ...getDependentFragments(name, fragments)];
        });

        return Array.from(new Set(dependentFragments));
    }

    const inPartial = fragments.filter(([fragmentName]) => partialTag.includes(fragmentName));

    const allFragmentsUsed = new Set(inPartial.map(([fragmentName]) => fragmentName));

    inPartial.forEach(([fragmentName]) => {
        const dependentFragments = getDependentFragments(fragmentName, fragments);
        dependentFragments.forEach(fragment => {
            allFragmentsUsed.add(fragment);
        });
    });

    const temp2 = fragments.filter(([fragmentName]) => allFragmentsUsed.has(fragmentName)).map(([fragmentName]) => fragmentName);
    return fragments.filter(([fragmentName]) => allFragmentsUsed.has(fragmentName));
}

/**
 * Converts fragment data (from DeepPartialBooleanWithFragments.__define) into graphql-tag strings.
 * @param fragments The fragment data.
 * @returns a graphql-tag string for each fragment, along with its name. (Array<[name, tag]>)
 */
export async function fragmentsToString(
    fragments: Exclude<DeepPartialBooleanWithFragments<any>["__define"], undefined>,
) {
    // Initialize result
    const result: [string, string][] = [];
    // Loop through fragments
    for (const [name, partial] of Object.entries(fragments)) {
        const objectType = name.split("_")[0];
        let fragmentString = "";
        fragmentString += `fragment ${name} on ${objectType} {\n`;
        fragmentString += await partialToStringHelper(partial as any);
        fragmentString += "}";
        result.push([name, fragmentString]);
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
 * Helper function to convert part of a GqlPartial object or fragment to a graphql-tag string.
 * @param partial The partial object to convert.
 * @param indent The number of spaces to indent the partial by.
 * @returns a graphql-tag string for the partial.
 */
export async function partialToStringHelper(
    partial: DeepPartialBooleanWithFragments<any>,
    indent = 0,
) {
    // If indent is too high, throw an error.
    if (indent > 69) {
        throw new Error("partialToStringHelper indent too high. Possible infinite loop.");
    }
    // Initialize the result string.
    let result = "";
    // Loop through the partial object.
    for (const [key, value] of Object.entries(partial)) {
        Array.isArray(value) && console.error("Array value in partialToStringHelper", key, value);
        // If key is __typename, __selectionType, or __define, skip it.
        // These are either not needed, or handled by other functions.
        if (["__typename", "__selectionType", "__define"].includes(key)) continue;
        // If key is __union, use unionToString to convert the union.
        else if (key === "__union") {
            result += await unionToString(value as Exclude<DeepPartialBooleanWithFragments<any>["__union"], undefined>, indent);
        }
        // If key is __use, use value as fragment name
        else if (key === "__use") {
            result += `${" ".repeat(indent)}...${value}\n`;
        }
        // If value is a boolean, add the key.
        else if (typeof value === "boolean") {
            result += `${" ".repeat(indent)}${key}\n`;
        }
        // Otherwise, value must be an (possibly lazy) object. So we can recurse.
        else {
            result += `${" ".repeat(indent)}${key} {\n`;
            result += await partialToStringHelper(typeof value === "function" ? await value() : value as any, indent + 4);
            result += `${" ".repeat(indent)}}\n`;
        }
    }
    return result;
}

type PartialToStringProps<
    EndpointType extends "query" | "mutation",
    EndpointName extends string,
    Partial extends GqlPartial<any>,
    Selection extends SelectionType,
> = {
    endpointType: EndpointType,
    endpointName: EndpointName,
    /**
     * The number of spaces to indent the string
     */
    indent: number,
    inputType: string | null,
    partial?: Partial,
    selectionType?: Selection | null | undefined
}

/**
 * Converts a DeepPartialBooleanWithFragments object to a 
 * string that can be used in a GraphQL query/mutation.
 * @param obj The object to convert
 * @param indent The number of spaces to indent the string by
 * @returns A properly-indented string that can be used in a GraphQL query/mutation. 
 * The string is structured from top to bottom in the shape:
 * - Fragment definitions, with duplicate fragments omitted
 * - The query/mutation itself (e.g. 'query team($input: FindByIdOrHandleInput!) {\team(input: $input) {\n')
 * - The fields of the query/mutation, with fragments referenced by name and unions formatted correctly
 * - The closing parentheses and brackets
 */
export async function partialToString<
    EndpointType extends "query" | "mutation",
    EndpointName extends string,
    Partial extends GqlPartial<any>,
    Selection extends SelectionType,
>({
    endpointType,
    endpointName,
    indent,
    inputType,
    partial,
    selectionType,
}: PartialToStringProps<EndpointType, EndpointName, Partial, Selection>): Promise<{
    fragments: [string, string][],
    tag: string
}> {
    // Initialize return data
    let fragments: [string, string][] = [];
    let tag = "";
    // Calculate the fragments and selection set by combining partials
    let combined: DeepPartialBooleanWithFragments<any> = {};
    if (exists(partial) && exists(selectionType)) {
        combined = await rel(partial, selectionType);
    }
    // Split fragments from the rest, so we can handle them separately
    const { __define, ...rest } = combined;
    // Add the query/mutation to the tag
    tag += `
${" ".repeat(indent)}${endpointType} ${endpointName}`;
    // If there is an input type, add it
    if (exists(inputType)) {
        tag += `($input: ${inputType}!)`;
    }
    // Add the opening bracket
    tag += ` {
${" ".repeat(indent + 2)}`;
    // Add name of the query/mutation and input
    tag += `${endpointName}${exists(inputType) ? "(input: $input)" : ""}`;
    // If there is a partial, add the fields
    if (exists(rest) && Object.keys(rest).length > 0) {
        // Add another opening bracket
        tag += ` {
`;
        tag += await partialToStringHelper(rest, indent + 4);
        // Add a closing brackets
        tag += `${" ".repeat(indent + 2)}}`;
    }
    // Add the final closing bracket
    tag += `
${" ".repeat(indent)}}`;
    // Before returning, add fragments to the beginning of the tag. 
    // We do this here so we can filter out additional fragments which may have snuck in 
    // (e.g. if a relation has an omitted field which used a fragment, it could make it here). 
    // Ideally we'd fix this problem earlier in the process, but ¯\_(ツ)_/¯
    if (exists(__define) && Object.keys(__define).length > 0) {
        let fragmentsString = "\n";
        fragments = await fragmentsToString(__define);
        // Filter out fragments not found in the tag
        fragments = fragmentsNeeded(fragments, tag); //TODO for morning: commenting this out fixes bookmarklist findMany, but breaks many other things. For example, projectList findOne now adds Api_list fragment (among others), when that's not needed
        // Sort fragments by name, just because it looks nicer
        fragments.sort(([a], [b]) => a.localeCompare(b));
        // For every fragment, add reference to it in the tag
        fragments.forEach(([fragmentName]) => {
            fragmentsString += `${" ".repeat(indent)}\${${fragmentName}}\n`;
        });
        // Add the fragments to the beginning of the tag
        tag = fragmentsString + tag;
    }
    // Finally, return results
    return { fragments, tag };
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
 * Helper function for generating a GraphQL muation for a given endpoint.
 * @param endpointName The name of the operation (endpoint).
 * @param inputType The name of the input type.
 * @param fragments An array of GraphQL fragments to include in the operation.
 * @param selectionSet The selection set for the operation.
 * @returns An object containing:
 * - fragments: An array of names and gql-tag strings for each fragment used in the operation.
 * - tag: A gql-tag string for the operation.
 */
export async function toMutation<
    Endpoint extends string,
    Partial extends GqlPartial<any>,
    Selection extends SelectionType,
>(
    endpointName: Endpoint,
    inputType: string | null,
    partial?: Partial,
    selectionType?: Selection | null | undefined,
) {
    return await partialToString({
        endpointType: "mutation",
        endpointName,
        inputType,
        indent: 0,
        partial,
        selectionType,
    });
}

/**
 * Helper function for generating a GraphQL query for a given endpoint.
 * @param endpointName The name of the operation (endpoint).
 * @param inputType The name of the input type.
 * @param partial GqlPartial object containing selection data
 * @param selectionType Which selection in the GqlPartial to use
 * @returns An object containing:
 * - fragments: An array of names and gql-tag strings for each fragment used in the operation.
 * - tag: A gql-tag string for the operation.
 */
export async function toQuery<
    Endpoint extends string,
    Partial extends GqlPartial<any>,
    Selection extends SelectionType,
>(
    endpointName: Endpoint,
    inputType: string | null,
    partial?: Partial,
    selectionType?: Selection | null | undefined,
) {
    return await partialToString({
        endpointType: "query",
        endpointName,
        inputType,
        indent: 0,
        partial,
        selectionType,
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
 * Converts union data (from DeepPartialBooleanWithFragments.__union) into a graphql-tag string.
 * @param union The union data.
 * @param indent The number of spaces to indent the union by.
 * @returns a graphql-tag string for the union, without outer braces.
 */
export async function unionToString(
    union: Exclude<DeepPartialBooleanWithFragments<any>["__union"], undefined>,
    indent = 0,
) {
    // Initialize the result string.
    let result = "";
    // Loop through the union object.
    for (const [key, value] of Object.entries(union)) {
        // Add indentation.
        result += " ".repeat(indent);
        // Add ellipsis, object type, and open brace.
        result += `... on ${key} {\n`;
        // Value should be either a string, object, or function.
        // If a string, treat as fragment name
        if (typeof value === "string") {
            result += `${" ".repeat(indent + 4)}...${value}\n`;
        }
        // If an object or function, convert 
        else if (typeof value === "object" || typeof value === "function") {
            result += await partialToStringHelper(typeof value === "function" ? await value() : value, indent + 4);
        }
        // Shouldn't be anything else. If so, there was likely an issue with 
        // converting union references (which can be a string, number, or symbol) to unique strings
        else {
            console.error("unionToString got unexpected value", key, value);
        }
        // Close the brace.
        result += `${" ".repeat(indent)}}\n`;
    }
    return result;
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

