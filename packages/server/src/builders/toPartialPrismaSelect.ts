import { ObjectMap } from "../models";
import { Formatter, GraphQLModelType } from "../models/types";
import { addCountFieldsHelper } from "./addCountFieldsHelper";
import { addJoinTablesHelper } from "./addJoinTablesHelper";
import { deconstructRelationshipsHelper } from "./deconstructRelationshipsHelper";
import { isRelationshipObject } from "./isRelationshipObject";
import { removeSupplementalFieldsHelper } from "./removeSupplementalFieldsHelper";
import { PartialGraphQLInfo, PartialPrismaSelect } from "./types";

/**
 * Converts shapes 2 and 3 of a GraphQL to Prisma conversion to shape 3. 
 * This function is useful when we want to check the shape of the requested data, 
 * but not actually query the database.
 * @param partial GraphQL info object, partially converted to Prisma select
 * @returns Prisma select object with calculated fields, unions and join tables removed, 
 * and count fields and __typenames added
 */
export const toPartialPrismaSelect = (partial: PartialGraphQLInfo | PartialPrismaSelect): PartialPrismaSelect => {
    // Create result object
    let result: { [x: string]: any } = {};
    // Loop through each key/value pair in partial
    for (const [key, value] of Object.entries(partial)) {
        // If value is an object (and not date), recursively call selectToDB
        if (isRelationshipObject(value)) {
            result[key] = toPartialPrismaSelect(value as PartialGraphQLInfo | PartialPrismaSelect);
        }
        // Otherwise, add key/value pair to result
        else {
            result[key] = value;
        }
    }
    // Handle base case
    const type: GraphQLModelType | undefined = partial?.__typename;
    const formatter: Formatter<any, any> | undefined = typeof type === 'string' ? ObjectMap[type as keyof typeof ObjectMap]?.format : undefined;
    if (formatter) {
        result = removeSupplementalFieldsHelper(type as GraphQLModelType, result);
        result = deconstructRelationshipsHelper(result, formatter.relationshipMap);
        result = addJoinTablesHelper(result, formatter.joinMap);
        result = addCountFieldsHelper(result, formatter.countMap);
    }
    return result;
}
