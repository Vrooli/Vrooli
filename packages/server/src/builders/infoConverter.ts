/**
 * All functions related to managing the conversion between API endpoint response objects and Prisma select objects. 
 * This is needed to properly format data, query additional fields that can't be included in the initial query 
 * (or are in another database perhaps), handle unions, etc.
 */
import { ModelType, OrArray, SessionUser, exists, getDotNotationValue, isObject, omit, setDotNotationValue } from "@local/shared";
import pkg from "lodash";
import { CustomError } from "../events/error";
import { ModelMap } from "../models/base";
import { FormatMap } from "../models/formats";
import { ApiRelMap, JoinMap, ModelLogicType } from "../models/types";
import { RecursivePartial } from "../types";
import { LRUCache } from "../utils/lruCache";
import { groupPrismaData } from "./groupPrismaData";
import { isRelationshipArray, isRelationshipObject } from "./isOfType";
import { PartialApiInfo, PartialPrismaSelect, PrismaSelect } from "./types";

const { merge } = pkg;

type DataShape = { [key: string]: any[] };

class Unions {
    private static instance: Unions;

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }
    public static get(): Unions {
        if (!Unions.instance) {
            Unions.instance = new Unions();
        }
        return Unions.instance;
    }

    /**
     * Constructs an API response object's relationship fields from database fields
     * @param data - Data object
     * @param partialInfo - Partial info object
     * @param apiRelMap - API response relationship map. Typically used for API-related 
     * operations, but unions in this object are defined with Prisma fields
     * @returns partialInfo object with union fields added
     */
    static construct<
        Typename extends `${ModelType}`,
        ApiModel extends ModelLogicType["ApiModel"],
        DbModel extends ModelLogicType["DbModel"],
    >(
        data: { [x: string]: any },
        partialInfo: { [x: string]: any },
        apiRelMap: ApiRelMap<Typename, ApiModel, DbModel>,
    ): { data: { [x: string]: any }, partialInfo: { [x: string]: any } } {
        // Create result objects
        const resultData: { [x: string]: any } = data;
        const resultPartialInfo: { [x: string]: any } = { ...partialInfo };
        // Any value in the apiRelMap which is an object is a union.
        // All other values can be ignored.
        const unionFields: [string, { [x: string]: ModelType }][] = Object.entries(apiRelMap).filter(([, value]) => typeof value === "object") as any[];
        // For each union field
        for (const [apiField, unionData] of unionFields) {
            // For each entry in the union
            for (const [dbField, type] of Object.entries(unionData)) {
                // If the current field is in the partial info, use it as the union data
                const isInPartialInfo = exists(resultData[dbField]);
                if (isInPartialInfo) {
                    // Set the union field to the type
                    resultData[apiField] = { ...resultData[dbField], __typename: type };
                    // If the union hasn't been converted in partialInfo, convert it
                    if (isObject(resultPartialInfo[apiField]) && isObject(resultPartialInfo[apiField][type])) {
                        resultPartialInfo[apiField] = resultPartialInfo[apiField][type];
                    }
                }
                // Delete the dbField from resultData
                delete resultData[dbField];
            }
            // If no union data was found, set the union field to null
            if (!exists(resultData[apiField])) {
                resultData[apiField] = null;
            }
        }
        return { data: resultData, partialInfo: resultPartialInfo };
    }

    /**
     * Deconstructs an API response object's relationship fields into database fields
     * @param data API response object
     * @param apiRelMap Mapping of relationship names to their transform shapes
     * @returns DB-shaped object
     */
    static deconstruct<
        Typename extends `${ModelType}`,
        ApiModel extends ModelLogicType["ApiModel"],
        DbModel extends ModelLogicType["DbModel"],
    >(
        data: { [x: string]: any },
        apiRelMap: ApiRelMap<Typename, ApiModel, DbModel>
    ): { [x: string]: any } {
        // Create result object
        const result: { [x: string]: any } = data;
        // Any value in the apiRelMap which is an object is a union. 
        // All other values can be ignored.
        const unionFields: [string, { [x: string]: ModelType }][] = Object.entries(apiRelMap).filter(([_, value]) => isRelationshipObject(value)) as any[];
        // For each union field
        for (const [key, value] of unionFields) {
            // If it's not in data, continue
            if (!data[key]) continue;
            // Store data from the union field
            const unionData = data[key];
            // Remove the union field from the result
            delete result[key];
            // If not an object, skip
            if (!isRelationshipObject(unionData)) continue;
            // Each value in "value" 
            // Iterate over the possible types
            for (const [prismaField, type] of Object.entries(value)) {
                // If the type is in the union data, add the db field to the result. 
                if (unionData[type]) {
                    result[prismaField] = unionData[type];
                }
            }
        }
        return result;
    }
}

class JoinTables {
    private static instance: JoinTables;

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }
    public static get(): JoinTables {
        if (!JoinTables.instance) {
            JoinTables.instance = new JoinTables();
        }
        return JoinTables.instance;
    }

    /**
     * Idempotent helper function for adding join tables between 
     * many-to-many relationship parents and children
     * @param partialInfo - API-shaped object
     * @param map - Mapping of many-to-many relationship names to join table names
     */
    static addToData(partialInfo: PartialApiInfo, map: JoinMap | undefined): any {
        if (!map) return partialInfo;
        // Create result object
        const result: any = {};
        // Iterate over join map
        for (const [key, value] of Object.entries(map)) {
            // If the key is in the object, 
            if (partialInfo[key]) {
                // Skip if already padded with join table name
                if (isRelationshipArray(partialInfo[key])) {
                    if ((partialInfo[key] as any).every((o: any) => isRelationshipObject(o) && Object.keys(o).length === 1 && Object.keys(o)[0] !== "id")) {
                        result[key] = partialInfo[key];
                        continue;
                    }
                } else if (isRelationshipObject(partialInfo[key])) {
                    if (Object.keys(partialInfo[key] as any).length === 1 && Object.keys(partialInfo[key] as any)[0] !== "id") {
                        result[key] = partialInfo[key];
                        continue;
                    }
                }
                // Otherwise, pad with the join table name
                result[key] = { [value]: partialInfo[key] };
            }
        }
        return {
            ...partialInfo,
            ...result,
        };
    }

    /**
     * Idempotent helper function for removing join tables between
     * many-to-many relationship parents and children
     * @param obj - DB-shaped object
     * @param map - Mapping of many-to-many relationship names to join table names
     */
    static removeFromData(obj: any, map: JoinMap | undefined): any {
        if (!obj || !map) return obj;
        // Create result object
        const result: any = {};
        // Iterate over join map
        for (const [key, value] of Object.entries(map)) {
            // If the key is in the object
            if (obj[key]) {
                // If the value is an array
                if (Array.isArray(obj[key])) {
                    // Check if the join should be applied (i.e. elements are objects with one non-ID key)
                    if (obj[key].every((o: any) => isRelationshipObject(o) && Object.keys(o).length === 1 && Object.keys(o)[0] !== "id")) {
                        // Remove the join table from each item in the array
                        result[key] = obj[key].map((item: any) => item[value]);
                    }
                } else {
                    // Check if the join should be applied (i.e. element is an object with one non-ID key)
                    if (isRelationshipObject(obj[key]) && Object.keys(obj[key]).length === 1 && Object.keys(obj[key])[0] !== "id") {
                        // Otherwise, remove the join table from the object
                        result[key] = obj[key][value];
                    }
                }
            }
        }
        return {
            ...obj,
            ...result,
        };
    }
}

export class CountFields {
    private static instance: CountFields;

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }
    public static get(): CountFields {
        if (!CountFields.instance) {
            CountFields.instance = new CountFields();
        }
        return CountFields.instance;
    }

    /**
     * Helper function for converting API count fields to Prisma relationship counts
     * @param obj - API-shaped object
     * @param countFields - List of API field names (e.g. ['commentsCount', reportsCount']) 
     * that correspond to Prisma relationship counts (e.g. { _count: { comments: true, reports: true } })
     */
    static addToData(obj: any, countFields: { [x: string]: true } | undefined): any {
        if (!countFields) return obj;
        // Create result object
        const result: any = {};
        // Iterate over count map
        for (const key of Object.keys(countFields)) {
            if (obj[key]) {
                if (!obj._count) obj._count = {};
                // Remove "Count" suffix from key
                const value = key.slice(0, -"Count".length);
                // Add count field to result object
                obj._count[value] = true;
                delete obj[key];
            }
        }
        return {
            ...obj,
            ...result,
        };
    }

    /**
     * Helper function for converting Prisma relationship counts to GraphQL count fields
     * @param obj - Prisma-shaped object
     * @param countFields - List of GraphQL field names (e.g. ['commentsCount', reportsCount']) 
     * that correspond to Prisma relationship counts (e.g. { _count: { comments: true, reports: true } })
     */
    static removeFromData(obj: any, countFields: { [x: string]: true } | undefined): any {
        if (!obj || !countFields) return obj;
        // If no counts, no reason to continue
        if (!obj._count) return obj;
        // Iterate over count map
        for (const key of Object.keys(countFields)) {
            // Remove "Count" suffix from key
            const value = key.slice(0, -"Count".length);
            if (exists(obj._count[value])) {
                obj[key] = obj._count[value];
            }
        }
        // Make sure to delete _count field
        delete obj._count;
        return { ...obj };
    }
}

class Typenames {
    private static instance: Typenames;

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }
    public static get(): Typenames {
        if (!Typenames.instance) {
            Typenames.instance = new Typenames();
        }
        return Typenames.instance;
    }

    /**
     * Recursively injects type fields into a select object
     * @param select - GraphQL select object, partially converted without typenames
     * and keys that map to typemappers for each possible relationship
     * @param parentRelationshipMap - Relationship of last known parent
     * @return select with type fields
     */
    static addToData<
        Typename extends `${ModelType}`,
        ApiModel extends ModelLogicType["ApiModel"],
        DbModel extends ModelLogicType["DbModel"],
    >(
        select: { [x: string]: any },
        parentRelationshipMap: ApiRelMap<Typename, ApiModel, DbModel>,
    ): PartialApiInfo {
        // Create result object
        const result: any = {};
        // Iterate over select object
        for (const [selectKey, selectValue] of Object.entries(select)) {
            // Skip type
            if (["type", "__typename"].includes(selectKey)) continue;
            // Find the corresponding relationship map. An array represents a union
            const nestedValue = parentRelationshipMap[selectKey];
            // If value is not an object, just add to result
            if (typeof selectValue !== "object") {
                result[selectKey] = selectValue;
                continue;
            }
            // If value is an object but not in the parent relationship map, add to result 
            // and make sure that no __typenames are present
            if (!nestedValue) {
                result[selectKey] = this.addToData(selectValue, {} as any);
                continue;
            }
            // If value is an object, recurse
            // If not union, add the single type to the result
            if (typeof nestedValue === "string") {
                if (selectValue && FormatMap[nestedValue!]) {
                    result[selectKey] = this.addToData(selectValue, FormatMap[nestedValue!]!.apiRelMap);
                }
            }
            // If union, add each possible type to the result
            else if (isRelationshipObject(nestedValue)) {
                // Iterate over possible types
                for (const [_, type] of Object.entries(nestedValue)) {
                    // If type is in selectValue, add it to the result
                    if (selectValue[type!] && FormatMap[type!]) {
                        if (!result[selectKey]) result[selectKey] = {};
                        result[selectKey][type!] = this.addToData(selectValue[type!], FormatMap[type!]!.apiRelMap);
                    }
                }
            }
        }
        // Add type field, assuming it exists (if won't exist when recursing the '{} as any')
        if (parentRelationshipMap.__typename) result.__typename = parentRelationshipMap.__typename;
        return result;
    }

    /**
     * Removes the "__typename" field recursively from a JSON-serializable object
     * @param obj - JSON-serializable object with possible type fields
     * @return obj without type fields
     */
    static removeFromData(obj: { [x: string]: any }): { [x: string]: any } {
        return JSON.parse(JSON.stringify(obj, (k, v) => (k === "__typename") ? undefined : v));
    }
}

/**
 * Returns a list of dot notation strings that describe every key in the input object, excluding arrays.
 *
 * @param obj The input object.
 * @param parentKey An optional argument used to keep track of the current key path as the function recurses.
 * @returns A list of dot notation strings that describe every key in the input object, excluding arrays.
 */
function getKeyPaths(obj: object, parentKey?: string): string[] {
    // The array to store the key paths
    let keys: string[] = [];
    // Loop through all the properties of the object
    for (const key in obj) {
        // Construct the current key path by concatenating the parent key and the current key, if the parent key is provided
        const currentKey = parentKey ? `${parentKey}.${key}` : key;
        // If the property is an object and not an array, recurse into it
        if (typeof obj[key] === "object" && !Array.isArray(obj[key])) {
            keys = keys.concat(getKeyPaths(obj[key], currentKey));
        } else {
            // If the property is not an object, add the current key path to the list of keys
            keys.push(currentKey);
        }
    }
    // Return the list of keys
    return keys;
}

/**
 * Adds supplemental fields data to the given objects
 */
export async function addSupplementalFieldsHelper<GraphQLModel extends { [x: string]: any }>({ languages, objects, objectType, partial, userData }: {
    languages: string[],
    objects: ({ id: string } & { [x: string]: any })[],
    objectType: `${ModelType}`,
    partial: PartialApiInfo,
    userData: SessionUser | null,
}): Promise<RecursivePartial<GraphQLModel>[]> {
    if (!objects || objects.length === 0) return [];
    // Get supplemental info for object
    const supplementer = ModelMap.get(objectType, false)?.search?.supplemental;
    if (!supplementer) return objects;
    // Get IDs from objects
    const ids = objects.map(({ id }) => id);
    // Get supplemental data by field
    const supplementalData = await supplementer.getSuppFields({ ids, languages, objects, partial, userData });
    // Convert supplemental data shape into dot notation
    const supplementalDotFields = getKeyPaths(supplementalData);
    // Loop through objects
    for (let i = 0; i < objects.length; i++) {
        // Loop through each dot notation field
        for (const field of supplementalDotFields) {
            // Create a dot notation string to retrieve the value from supplemental data
            const dotField = `${field}.${i}` as const;
            // Get the value from the supplemental data
            const suppValue = getDotNotationValue(supplementalData, dotField);
            // Set the value on the object
            setDotNotationValue(objects[i], field as never, suppValue);
        }
    }
    return objects;
}

/**
 * Recombines objects returned from the supplementalFields function into a shape that matches the requested info
 * @param data Original, unsupplemented data, where each object has an ID
 * @param objectsById Dictionary of objects with supplemental fields added, where each value contains at least an ID
 * @returns data with supplemental fields added
 */
function combineSupplements(data: { [x: string]: any }, objectsById: { [x: string]: any }) {
    const result: { [x: string]: any } = {};
    // Loop through each key/value pair in data
    for (const [key, value] of Object.entries(data)) {
        // If value is an array, add each element to the correct key in objectDict
        if (Array.isArray(value)) {
            // Pass each element through combineSupplements
            result[key] = data[key].map((v: any) => combineSupplements(v, objectsById));
        }
        // If value is an object (and not a date), add it to the correct key in objectDict
        else if (isRelationshipObject(value)) {
            // Pass value through combineSupplements
            result[key] = combineSupplements(value, objectsById);
        }
    }
    // Handle base case (combining non-array and non-object values)
    return merge(result, objectsById[data.id]);
}

/**
 * Adds supplemental fields to the Prisma select object, and all of its relationships (and their relationships, etc.)
 * Groups objects types together, so database is called only once for each type.
 * @param userData Requesting user's data
 * @param data Array of GraphQL-shaped data, where each object contains at least an ID
 * @param partialInfo API endpoint info object
 * @returns data array with supplemental fields added to each object
 */
export async function addSupplementalFields(
    userData: SessionUser | null,
    data: ({ [x: string]: any } | null | undefined)[],
    partialInfo: OrArray<PartialApiInfo>,
): Promise<{ [x: string]: any }[]> {
    if (data.length === 0) return [];
    // Group data into dictionaries, which will make later operations easier
    const { objectTypesIdsDict, selectFieldsDict, objectIdsDataDict } = groupPrismaData(data, partialInfo);
    // Dictionary to store supplemental data
    const supplementsByObjectId: { [x: string]: any } = {};

    // Loop through each type in objectTypesIdsDict
    for (const [type, ids] of Object.entries(objectTypesIdsDict)) {
        // Find the supplemental data for each object id in ids
        const objectData = ids.map((id: string) => objectIdsDataDict[id]);
        const supplemental = ModelMap.get(type as ModelType, false)?.search?.supplemental;
        const valuesWithSupplements = supplemental ?
            await addSupplementalFieldsHelper({ languages: userData?.languages ?? ["en"], objects: objectData, objectType: type as ModelType, partial: selectFieldsDict[type], userData }) :
            objectData;
        // Supplements are calculated for an array of objects, so we must loop through 
        // Add each value to supplementsByObjectId
        for (const v of valuesWithSupplements) {
            supplementsByObjectId[v.id] = v;
            // Also add the type to the object, which can be used 
            // by our union resolver to determine which __typename to use
            supplementsByObjectId[v.id].__typename = type;
        }
    }
    // Convert supplementsByObjectId dictionary back into shape of data
    const result = data.map(d => (d === null || d === undefined) ? d : combineSupplements(d, supplementsByObjectId));
    return result;
}

/**
 * Combines addSupplementalFields calls for multiple object types
 * @param data Object of arrays, where each value is a list of the same object type queried from the database
 * @param partial API endpoint info object with the same keys as data, and values equal to the partial info for that object type
 * @param userData Requesting user's data
 * @returns Object in same shape as data, but with each value containing supplemental data
 */
export async function addSupplementalFieldsMultiTypes<
    TData extends DataShape,
    TPartial extends { [K in keyof TData]: PartialApiInfo }
>(
    data: TData,
    partial: TPartial,
    userData: SessionUser | null,
): Promise<{ [K in keyof TData]: any[] }> {
    // Flatten data object into an array and create an array of partials that match the data array
    const combinedData: any[] = [];
    const combinedPartialInfo: PartialApiInfo[] = [];
    for (const key in data) {
        for (const item of data[key]) {
            combinedData.push(item);
            combinedPartialInfo.push(partial[key]);
        }
    }

    // Call addSupplementalFields
    const combinedResult = await addSupplementalFields(userData, combinedData, combinedPartialInfo);

    // Convert combinedResult into object in the same shape as data, but with each value containing supplemental data
    const formatted = {} as { [K in keyof TData]: any[] };
    let start = 0;
    for (const key in data) {
        const end = start + data[key].length;
        formatted[key] = combinedResult.slice(start, end);
        start = end;
    }
    return formatted;
}

// Cache results of each `info` conversion, so we only have to 
// do it once. This improves performance a bit
const cache1 = new LRUCache<string, PartialApiInfo>(1000, 250_000);

/**
 * Converts shapes 2 and 3 of a API response to Prisma conversion to shape 3. 
 * 
 * The result is like a Prisma select object, but with __typename fields and no 
 * "select" padding.
 * 
 * This function is useful when we want to check the shape of the requested data
 * 
 * @param partial GraphQL info object, partially converted to Prisma select
 * @returns Prisma select object with calculated fields, unions and join tables removed, 
 * and count fields and types added
 */
function toPartialPrismaSelect(partial: PartialApiInfo | PartialPrismaSelect): PartialPrismaSelect {
    // Create result object
    let result: { [x: string]: any } = {};
    // Loop through each key/value pair in partial
    for (const [key, value] of Object.entries(partial)) {
        // If value is an object (and not date), recurse
        if (isRelationshipObject(value)) {
            result[key] = toPartialPrismaSelect(value as PartialApiInfo | PartialPrismaSelect);
        }
        // Otherwise, add key/value pair to result
        else {
            result[key] = value;
        }
    }
    // Handle base case}
    const type = partial.__typename;
    const format = ModelMap.get(type, false)?.format;
    if (type) {
        result = removeSupplementalFields(type, result);
        if (format) {
            result = Unions.deconstruct(result, format.apiRelMap);
            result = JoinTables.addToData(result, format.joinMap as any);
            result = CountFields.addToData(result, format.countFields);
        }
    }
    return result;
}

// Cache results of each `partial` conversion, so we only have to 
// do it once. This improves performance a bit
const SELECT_CACHE_LIMIT_COUNT = 1_000;
const SELECT_CACHE_LIMIT_SIZE_BYTES = 250_000;
const cache2 = new LRUCache<string, PrismaSelect>(SELECT_CACHE_LIMIT_COUNT, SELECT_CACHE_LIMIT_SIZE_BYTES);

/**
 * Adds "select" to the correct parts of an object to make it a Prisma select. 
 * This function is recursive and idempotent.
 */
export function selPad(fields: object): PrismaSelect {
    // Only pad if fields is an object
    if (!isRelationshipObject(fields)) {
        // Return empty object for invalid/empty fields
        if (fields === null || fields === undefined || Array.isArray(fields)) return {} as PrismaSelect;
        // Anything else (strings, Dates, numbers, etc.) should be returned as true
        return true as unknown as PrismaSelect;
    }
    // Recursively pad
    const converted: object = {};
    Object.keys(fields).forEach((key) => {
        // Check if already padded
        const isAlreadyPadded = key === "select";
        // Recursively pad, even if already padded. 
        // This is because nested relationships may not be padded
        let padded = selPad(fields[key]);
        // If it was padded before, remove top-level "select"
        if (isAlreadyPadded) padded = (padded as unknown as { select: PrismaSelect })["select"];
        // If padded is empty object, skip
        if (isRelationshipObject(padded) && Object.keys(padded).length === 0) return;
        // Add padded to converted
        converted[key] = padded;
    });
    // If converted is empty, return
    if (Object.keys(converted).length === 0) return converted as PrismaSelect;
    // If already padded, return
    if ("select" in converted) return converted as PrismaSelect;
    // Return object with "select" padded
    return { select: converted } as PrismaSelect;
}

/**
 * Converts shapes 2 and 3 of the GraphQL to Prisma conversion to shape 4.
 * @returns Object which can be passed into Prisma select directly
 */
export function selectHelper(partial: PartialApiInfo | PartialPrismaSelect): PrismaSelect | undefined {
    // Check if cached
    const cacheKey = JSON.stringify(partial);
    const cachedResult = cache2.get(cacheKey);
    if (cachedResult !== undefined) {
        return cachedResult;
    }
    // Convert partial's special cases (virtual/calculated fields, unions, etc.)
    let modified = toPartialPrismaSelect(partial);
    if (!isObject(modified)) return undefined;
    // Delete type fields
    modified = Typenames.removeFromData(modified);
    // Pad every relationship with "select"
    modified = selPad(modified);
    // Cache result
    cache2.set(cacheKey, modified as PrismaSelect);
    return modified as PrismaSelect;
}

/**
 * Removes a list of fields from a GraphQL object. Useful when additional fields 
 * are added to the Prisma select to calculate supplemental fields, but should never 
 * be returned to the client.
 * 
 * NOTE: Supports dot notation through recursion
 */
function removeHiddenFields<T extends { [x: string]: any }>(
    obj: T,
    fields: string[] | undefined,
): T {
    if (!fields) return obj;
    // Initialize result
    const result: any = {};
    // Iterate over object
    for (const [key, value] of Object.entries(obj)) {
        // If key is in fields, skip
        if (fields.includes(key)) continue;
        // If value is an object
        if (typeof value === "object") {
            // Find nested fields
            const nestedFields = fields.filter((field) => field.startsWith(`${key}.`)).map((field) => field.replace(`${key}.`, ""));
            // If there are nested fields, recurse
            if (nestedFields.length > 0) {
                // Recurse
                result[key] = removeHiddenFields(value, nestedFields);
            }
            // Otherwise, just add the value
            else {
                result[key] = value;
            }
        }
        // Otherwise, add to result
        else result[key] = value;
    }
    return result;
}

/**
 * Removes supplemental fields (i.e. fields that cannot be calculated in the main query), and also may 
 * add additional fields to calculate the supplemental fields
 * @param objectType Type of object to get supplemental fields for
 * @param partial Select fields object
 * @returns partial with supplemental fields removed, and maybe additional fields added
 */
function removeSupplementalFields(objectType: `${ModelType}`, partial: PartialApiInfo | PartialPrismaSelect) {
    // Get supplemental info for object
    const supplementer = ModelMap.get(objectType, false)?.search?.supplemental;
    if (!supplementer) return partial;
    // Remove API supplemental fields
    const withoutGqlSupp = omit(partial, supplementer.suppFields);
    // Add db supplemental fields used to calculate API supplemental fields
    if (supplementer.dbFields) {
        // For each db supplemental field, add it to the select object with value true
        const dbSupp = supplementer.dbFields.reduce((acc, curr) => {
            acc[curr] = true;
            return acc;
        }, {} as PartialPrismaSelect);
        // Merge db supplemental fields with select object
        return merge(withoutGqlSupp, dbSupp);
    }
    return withoutGqlSupp;
}

/**
 * Manages conversions between the API response object and Prisma select object.
 * 
 * There are 4 shapes the info object can take in its lifecycle from API response object (as in the shape we 
 * expect from the API, not the actual response data) to Prisma select object:
 * - Shape 1: API response object
 * - Shape 2: PartialApiInfo - Adds typenames and removes endpoint-specific shapes like pageInfo and edges
 * - Shape 3: PartialPrismaSelect - Removes calculated fields, join tables, and other data transformations
 * - Shape 4: PrismaSelect - Adds "select" padding to the object, making it a valid Prisma select object
 * 
 * Then when we receive the data from the database, we have a single function that goes from the last shape to the first.
 */
export class InfoConverter {
    private static instance: InfoConverter;

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }
    public static get(): InfoConverter {
        if (!InfoConverter.instance) {
            InfoConverter.instance = new InfoConverter();
        }
        return InfoConverter.instance;
    }

    /**
     * Converts shapes 1 and 2 of a API response to Prisma conversion to shape 2. 
     * This means adding typenames and removing any endpoint-specific shapes like pageInfo and edges.
     * @param info API endpoint info object, or result of this function
     * @param apiRelMap Map of relationship names to typenames
     * @param throwIfNotPartial Throw error if info is not partial
     * @returns Partial Prisma select. This can be passed into the function again without changing the result.
     */
    static fromApiToPartialApi<
        Typename extends `${ModelType}`,
        ApiModel extends ModelLogicType["ApiModel"],
        DbModel extends ModelLogicType["DbModel"],
        ThrowErrorIfNotPartial extends boolean
    >(
        info: PartialApiInfo,
        apiRelMap: ApiRelMap<Typename, ApiModel, DbModel>,
        throwIfNotPartial: ThrowErrorIfNotPartial = false as ThrowErrorIfNotPartial,
    ): ThrowErrorIfNotPartial extends true ? PartialApiInfo : (PartialApiInfo | undefined) {
        // Return undefined if info not set
        if (!info) {
            if (throwIfNotPartial)
                throw new CustomError("0345", "InternalError");
            return undefined as any;
        }
        // Check if cached
        const cacheKey = JSON.stringify(info);
        const cachedResult = cache1.get(cacheKey);
        if (cachedResult !== undefined) {
            return cachedResult;
        }
        // Find select fields in info object
        let select = info;
        // If fields are in the shape of a paginated search query, convert to a Prisma select object
        if (Object.prototype.hasOwnProperty.call(select, "pageInfo") && Object.prototype.hasOwnProperty.call(select, "edges")) {
            select = (select as { edges: { node: any } }).edges.node;
        }
        // If fields are in the shape of a comment thread search query, convert to a Prisma select object
        else if (Object.prototype.hasOwnProperty.call(select, "endCursor") && Object.prototype.hasOwnProperty.call(select, "totalThreads") && Object.prototype.hasOwnProperty.call(select, "threads")) {
            select = (select as { threads: { comment: any } }).threads.comment;
        }
        // Inject type fields
        select = Typenames.addToData(select, apiRelMap);
        if (!select)
            throw new CustomError("0346", "InternalError");
        // Cache result
        cache1.set(cacheKey, select);
        return select;
    }

    // static fromApiPartialToPrismaPartial();

    // static fromPrismaPartialToPrismaSelect();

    /**
     * Converts data received from the database to the API response object shape (Shape 3 -> Shape 1)
     * @param data Prisma object
     * @param partialInfo API endpoint info object
     * @returns Valid API response object
     */
    static fromDbToApi<ObjectModel extends Record<string, any>>(
        data: { [x: string]: any },
        partialInfo: PartialApiInfo,
    ): ObjectModel {
        // Convert data to usable shape
        const type = partialInfo?.__typename;
        const format = ModelMap.get(type, false)?.format;
        if (format) {
            const unionData = Unions.construct(data, partialInfo, format.apiRelMap);
            data = unionData.data;
            partialInfo = unionData.partialInfo;
            data = JoinTables.removeFromData(data, format.joinMap as any);
            data = CountFields.removeFromData(data, format.countFields);
            data = removeHiddenFields(data, format.hiddenFields);
        }
        // Then loop through each key/value pair in data and call recurse on each array item/object
        for (const [key, value] of Object.entries(data)) {
            // If key doesn't exist in partialInfo, check if union
            if (!isObject(partialInfo) || !(key in partialInfo)) {
                continue;
            }
            // If value is an array, recurse on each element
            if (Array.isArray(value)) {
                data[key] = data[key].map((v: any) => this.fromDbToApi(v, partialInfo[key] as PartialApiInfo));
            }
            // If value is an object (and not date), recurse
            else if (isRelationshipObject(value)) {
                data[key] = this.fromDbToApi(value, (partialInfo as any)[key]);
            }
        }
        return data as ObjectModel;
    }
}

// TODO should be able to remove toPartialPrismaSelect and put its logic in selectHelper. Maybe also do some simplifying here (e.g. removing typename as we recurse instead of after using a parse/stringify)
//TODO reorganize this file and rewrite in a way that is more efficient