import { isObject } from "@local/shared";
import { ModelMap } from "../models/base";
import { isRelationshipObject } from "./isOfType";
import { removeCountFields } from "./removeCountFields";
import { removeHiddenFields } from "./removeHiddenFields";
import { removeJoinTables } from "./removeJoinTables";
import { ApiEndpointInfo } from "./types";
import { constructUnions } from "./unions";

/**
 * Converts shapes 3 of the GraphQL to Prisma conversion to shape 1. Used to format the result of a query.
 * @param data Prisma object
 * @param partialInfo API endpoint info object
 * @returns Valid API response object
 */
export function modelToGql<ObjectModel extends Record<string, any>>(
    data: { [x: string]: any },
    partialInfo: ApiEndpointInfo,
): ObjectModel {
    // Convert data to usable shape
    const type = partialInfo?.__typename;
    const format = ModelMap.get(type, false)?.format;
    if (format) {
        const unionData = constructUnions(data, partialInfo, format.gqlRelMap);
        data = unionData.data;
        partialInfo = unionData.partialInfo;
        data = removeJoinTables(data, format.joinMap as any);
        data = removeCountFields(data, format.countFields);
        data = removeHiddenFields(data, format.hiddenFields);
    }
    // Then loop through each key/value pair in data and call modelToGql on each array item/object
    for (const [key, value] of Object.entries(data)) {
        // If key doesn't exist in partialInfo, check if union
        if (!isObject(partialInfo) || !(key in partialInfo)) {
            continue;
        }
        // If value is an array, call modelToGql on each element
        if (Array.isArray(value)) {
            // Pass each element through modelToGql
            data[key] = data[key].map((v: any) => modelToGql(v, partialInfo[key] as ApiEndpointInfo));
        }
        // If value is an object (and not date), call modelToGql on it
        else if (isRelationshipObject(value)) {
            data[key] = modelToGql(value, (partialInfo as any)[key]);
        }
    }
    return data as ObjectModel;
}
