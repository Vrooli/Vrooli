import { ModelMap } from "../models/base";
import { addCountFields } from "./addCountFields";
import { addJoinTables } from "./addJoinTables";
import { isRelationshipObject } from "./isOfType";
import { removeSupplementalFields } from "./removeSupplementalFields";
import { ApiEndpointInfo, PartialPrismaSelect } from "./types";
import { deconstructUnions } from "./unions";

/**
 * Converts shapes 2 and 3 of a GraphQL to Prisma conversion to shape 3. 
 * This function is useful when we want to check the shape of the requested data, 
 * but not actually query the database.
 * @param partial GraphQL info object, partially converted to Prisma select
 * @returns Prisma select object with calculated fields, unions and join tables removed, 
 * and count fields and types added
 */
export function toPartialPrismaSelect(partial: ApiEndpointInfo | PartialPrismaSelect): PartialPrismaSelect {
    // Create result object
    let result: { [x: string]: any } = {};
    // Loop through each key/value pair in partial
    for (const [key, value] of Object.entries(partial)) {
        // If value is an object (and not date), recurse
        if (isRelationshipObject(value)) {
            result[key] = toPartialPrismaSelect(value as ApiEndpointInfo | PartialPrismaSelect);
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
            result = deconstructUnions(result, format.gqlRelMap);
            result = addJoinTables(result, format.joinMap as any);
            result = addCountFields(result, format.countFields);
        }
    }
    return result;
};
