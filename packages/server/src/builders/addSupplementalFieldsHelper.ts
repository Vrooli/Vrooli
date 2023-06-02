import { getDotNotationValue, GqlModelType, setDotNotationValue } from "@local/shared";
import { ObjectMap } from "../models/base";
import { SupplementalConverter } from "../models/types";
import { PrismaType, RecursivePartial, SessionUserToken } from "../types";
import { PartialGraphQLInfo } from "./types";

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
export const addSupplementalFieldsHelper = async <GraphQLModel extends { [x: string]: any }>({ languages, objects, objectType, partial, prisma, userData }: {
    languages: string[],
    objects: ({ id: string } & { [x: string]: any })[],
    objectType: `${GqlModelType}`,
    partial: PartialGraphQLInfo,
    prisma: PrismaType,
    userData: SessionUserToken | null,
}): Promise<RecursivePartial<GraphQLModel>[]> => {
    if (!objects || objects.length === 0) return [];
    // Get supplemental info for object
    const supplementer: SupplementalConverter<any> | undefined = ObjectMap[objectType]?.search?.supplemental;
    if (!supplementer) return objects;
    // Get IDs from objects
    const ids = objects.map(({ id }) => id);
    // Get supplemental data by field
    const supplementalData = await supplementer.toGraphQL({ ids, languages, objects, partial, prisma, userData });
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
};
