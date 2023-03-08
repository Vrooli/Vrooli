import { ObjectMap } from "../models";
import { addCountFields } from "./addCountFields";
import { addJoinTables } from "./addJoinTables";
import { deconstructUnions } from "./deconstructUnions";
import { isRelationshipObject } from "./isRelationshipObject";
import { removeSupplementalFields } from "./removeSupplementalFields";
import { PartialGraphQLInfo, PartialPrismaSelect } from "./types";

/**
 * Converts shapes 2 and 3 of a GraphQL to Prisma conversion to shape 3. 
 * This function is useful when we want to check the shape of the requested data, 
 * but not actually query the database.
 * @param partial GraphQL info object, partially converted to Prisma select
 * @returns Prisma select object with calculated fields, unions and join tables removed, 
 * and count fields and types added
 */
export const toPartialPrismaSelect = (partial: PartialGraphQLInfo | PartialPrismaSelect): PartialPrismaSelect => {
    // Create result object
    let result: { [x: string]: any } = {};
    // Loop through each key/value pair in partial
    for (const [key, value] of Object.entries(partial)) {
        // If value is an object (and not date), recurse
        if (isRelationshipObject(value)) {
            result[key] = toPartialPrismaSelect(value as PartialGraphQLInfo | PartialPrismaSelect);
        }
        // Otherwise, add key/value pair to result
        else {
            result[key] = value;
        }
    }
    // Handle base case}
    const type = partial.__typename;
    const format = typeof type === 'string' ? ObjectMap[type as keyof typeof ObjectMap]?.format : undefined;
    if (type && format) {
        console.log('going to remove supplemental fields for: ', type);
        result = removeSupplementalFields(type, result);
        result = deconstructUnions(result, format.gqlRelMap);
        result = addJoinTables(result, format.joinMap as any);
        console.log('COUNT FIELDS', partial.__typename, format.countFields)
        result = addCountFields(result, format.countFields);
        console.log('result after count fields', JSON.stringify(result), '\n\n');
    }
    return result;
}
