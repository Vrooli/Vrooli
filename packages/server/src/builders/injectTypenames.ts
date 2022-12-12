import { ObjectMap } from "../models";
import { GraphQLModelType, RelationshipMap } from "../models/types";
import { SingleOrArray } from "../types";
import { PartialGraphQLInfo } from "./types";

/**
 * Recursively injects __typename fields into a select object
 * @param select - GraphQL select object, partially converted without typenames
 * and keys that map to typemappers for each possible relationship
 * @param parentRelationshipMap - Relationship of last known parent
 * @return select with __typename fields
 */
export const injectTypenames = <
    GQLObject extends { [x: string]: any },
    PrismaObject extends { [x: string]: any }
>(select: { [x: string]: any }, parentRelationshipMap: RelationshipMap<GQLObject, PrismaObject>): PartialGraphQLInfo => {
    // Create result object
    let result: any = {};
    // Iterate over select object
    for (const [selectKey, selectValue] of Object.entries(select)) {
        // Skip __typename
        if (selectKey === '__typename') continue;
        // If value is not an object, just add to result
        if (typeof selectValue !== 'object') {
            result[selectKey] = selectValue;
            continue;
        }
        // If value is an object, recurse
        // Find the corresponding relationship map. An array represents a union
        const nestedValue: SingleOrArray<GraphQLModelType> | undefined = parentRelationshipMap[selectKey];
        // If union, add each possible type to the result
        if (nestedValue !== undefined && Array.isArray(nestedValue)) {
            // Iterate over possible types
            for (const type of nestedValue) {
                // If type is in selectValue, add it to the result
                if (selectValue[type] && ObjectMap[type]) {
                    result[type] = injectTypenames(selectValue[type], ObjectMap[type]!.format.relationshipMap);
                }
            }
        }
        // If not union, add the single type to the result
        else {
            if (selectValue && ObjectMap[nestedValue!]) {
                result[selectKey] = injectTypenames(selectValue, ObjectMap[nestedValue!]!.format.relationshipMap);
            }
        }
    }
    // Add __typename field
    result.__typename = parentRelationshipMap.__typename;
    return result;
}
