import { isObject } from "@shared/utils";
import { ObjectMap } from "../models";
import { constructUnions } from "./constructUnions";
import { isRelationshipObject } from "./isRelationshipObject";
import { removeCountFields } from "./removeCountFields";
import { removeHiddenFields } from "./removeHiddenFields";
import { removeJoinTables } from "./removeJoinTables";
import { PartialGraphQLInfo } from "./types";

/**
 * Converts shapes 4 of the GraphQL to Prisma conversion to shape 1. Used to format the result of a query.
 * @param data Prisma object
 * @param partialInfo PartialGraphQLInfo object
 * @returns Valid GraphQL object
 */
export function modelToGql<
    GraphQLModel extends Record<string, any>
>(
    data: { [x: string]: any },
    partialInfo: PartialGraphQLInfo
): GraphQLModel {
    // Convert data to usable shape
    const type = partialInfo?.__typename;
    const format = typeof type === 'string' ? ObjectMap[type as keyof typeof ObjectMap]?.format : undefined;
    if (format) {
        data = constructUnions(data, format.gqlRelMap);
        data = removeJoinTables(data, format.joinMap as any);
        data = removeCountFields(data, format.countFields);
        data = removeHiddenFields(data, format.hiddenFields);
    }
    // Then loop through each key/value pair in data and call modelToGql on each array item/object
    for (const [key, value] of Object.entries(data)) {
        // If key doesn't exist in partialInfo, check if union
        if (!isObject(partialInfo) || !(key in partialInfo)) continue;
        // If value is an array, call modelToGql on each element
        if (Array.isArray(value)) {
            // Pass each element through modelToGql
            data[key] = data[key].map((v: any) => modelToGql(v, partialInfo[key] as PartialGraphQLInfo));
        }
        // If value is an object (and not date), call modelToGql on it
        else if (isRelationshipObject(value)) {
            data[key] = modelToGql(value, (partialInfo as any)[key]);
        }
    }
    return data as any;
}