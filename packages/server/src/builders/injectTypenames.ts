import { GqlModelType } from "@local/shared";
import { FormatMap } from "../models/formats";
import { GqlRelMap, ModelLogicType } from "../models/types";
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
    Typename extends `${GqlModelType}`,
    GqlModel extends ModelLogicType["GqlModel"],
    PrismaModel extends ModelLogicType["PrismaModel"],
>(select: { [x: string]: any }, parentRelationshipMap: GqlRelMap<Typename, GqlModel, PrismaModel>): PartialGraphQLInfo => {
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
            result[selectKey] = injectTypenames(selectValue, {} as any);
            continue;
        }
        // If value is an object, recurse
        // If not union, add the single type to the result
        if (typeof nestedValue === "string") {
            if (selectValue && FormatMap[nestedValue!]) {
                result[selectKey] = injectTypenames(selectValue, FormatMap[nestedValue!]!.gqlRelMap);
            }
        }
        // If union, add each possible type to the result
        else if (isRelationshipObject(nestedValue)) {
            // Iterate over possible types
            for (const [_, type] of Object.entries(nestedValue)) {
                // If type is in selectValue, add it to the result
                if (selectValue[type!] && FormatMap[type!]) {
                    if (!result[selectKey]) result[selectKey] = {};
                    result[selectKey][type!] = injectTypenames(selectValue[type!], FormatMap[type!]!.gqlRelMap);
                }
            }
        }
    }
    // Add type field, assuming it exists (if won't exist when recursing the '{} as any')
    if (parentRelationshipMap.__typename) result.__typename = parentRelationshipMap.__typename;
    return result;
};
