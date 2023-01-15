import { isObject } from "@shared/utils";
import { ObjectMap } from "../models";
import { constructUnions } from "./constructUnions";
import { isRelationshipObject } from "./isRelationshipObject";
import { removeCountFields } from "./removeCountFields";
import { removeHiddenFields } from "./removeHiddenFields";
import { removeJoinTables } from "./removeJoinTables";
import { subsetsMatch } from "./subsetsMatch";
import { PartialGraphQLInfo } from "./types";

/**
 * Converts shapes 4 of the GraphQL to Prisma conversion to shape 1. Used to format the result of a query.
 * @param data Prisma object
 * @param partialInfo PartialGraphQLInfo object
 * @returns Valid GraphQL object
 */
export function modelToGraphQL<
    GraphQLModel extends Record<string, any>
>(
    data: { [x: string]: any },
    partialInfo: PartialGraphQLInfo
): GraphQLModel {
    console.log('modeltographql start');
    // Remove top-level union from partialInfo, if necessary
    // If every key starts with a capital letter, it's a union. 
    // There's a catch-22 here which we must account for. Since "data" has not 
    // been shaped yet, it won't match the shape of "partialInfo". But we can't do 
    // this after shaping "data" because we need to know the type of the union. 
    // To account for this, we call modelToGraphQL on each union, to check which one matches "data"
    if (Object.keys(partialInfo).every(k => k[0] === k[0].toUpperCase())) {
        // Find the union type which matches the shape of value. 
        let matchingType: string | undefined;
        for (const unionType of Object.keys(partialInfo)) {
            const unionPartial = partialInfo[unionType];
            if (!isObject(unionPartial)) continue;
            const convertedData = modelToGraphQL(data, unionPartial as any);
            if (subsetsMatch(convertedData, unionPartial)) matchingType = unionType;
        }
        if (matchingType) {
            partialInfo = partialInfo[matchingType] as PartialGraphQLInfo;
        }
    }
    // Convert data to usable shape
    const type = partialInfo?.__typename;
    const format = typeof type === 'string' ? ObjectMap[type as keyof typeof ObjectMap]?.format : undefined;
    if (format) {
        data = constructUnions(data, format.gqlRelMap);
        data = removeJoinTables(data, format.joinMap as any);
        data = removeCountFields(data, format.countFields);
        data = removeHiddenFields(data, format.hiddenFields);
    }
    // Then loop through each key/value pair in data and call modelToGraphQL on each array item/object
    for (const [key, value] of Object.entries(data)) {
        // If key doesn't exist in partialInfo, check if union
        if (!isObject(partialInfo) || !(key in partialInfo)) continue;
        // If value is an array, call modelToGraphQL on each element
        if (Array.isArray(value)) {
            // Pass each element through modelToGraphQL
            data[key] = data[key].map((v: any) => modelToGraphQL(v, partialInfo[key] as PartialGraphQLInfo));
        }
        // If value is an object (and not date), call modelToGraphQL on it
        else if (isRelationshipObject(value)) {
            data[key] = modelToGraphQL(value, (partialInfo as any)[key]);
        }
    }
    return data as any;
}