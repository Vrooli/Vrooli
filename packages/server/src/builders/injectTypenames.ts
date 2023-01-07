import { ObjectMap } from "../models";
import { GqlRelMap } from "../models/types";
import { isRelationshipObject } from "./isRelationshipObject";
import { PartialGraphQLInfo } from "./types";

/**
 * Recursively injects type fields into a select object
 * @param select - GraphQL select object, partially converted without typenames
 * and keys that map to typemappers for each possible relationship
 * @param parentRelationshipMap - Relationship of last known parent
 * @return select with type fields
 */
export const injectTypenames = <
    GQLObject extends { [x: string]: any },
    PrismaObject extends { [x: string]: any }
>(select: { [x: string]: any }, parentRelationshipMap: GqlRelMap<GQLObject, PrismaObject>): PartialGraphQLInfo => {
    // Create result object
    let result: any = {};
    // Iterate over select object
    for (const [selectKey, selectValue] of Object.entries(select)) {
        // Skip type
        if (selectKey === 'type') continue;
        // If value is not an object, just add to result
        if (typeof selectValue !== 'object') {
            result[selectKey] = selectValue;
            continue;
        }
        // If value is an object, recurse
        // Find the corresponding relationship map. An array represents a union
        const nestedValue = parentRelationshipMap[selectKey];
        // If not union, add the single type to the result
        if (typeof nestedValue === 'string') {
            if (selectValue && ObjectMap[nestedValue!]) {
                result[selectKey] = injectTypenames(selectValue, ObjectMap[nestedValue!]!.format.gqlRelMap);
            }
        }
        // If union, add each possible type to the result
        else if (isRelationshipObject(nestedValue)) {
            // Iterate over possible types
            for (const [field, type] of Object.entries(nestedValue)) {
                // If type is in selectValue, add it to the result
                if (selectValue[type!] && ObjectMap[type!]) {
                    result[type!] = injectTypenames(selectValue[type!], ObjectMap[type!]!.format.gqlRelMap);
                }
            }
        }
    }
    // Add type field
    result.type = parentRelationshipMap.type;
    return result;
}
