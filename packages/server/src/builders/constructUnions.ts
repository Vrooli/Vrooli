import { GraphQLModelType, GqlRelMap } from "../models/types";

/**
 * Constructs a GraphQL object's relationship fields from database fields. It's the opposite of deconstructRelationships
 * @param partialInfo - Partial info object
 * @param gqlRelMap - GraphQL relationship map. Typically used for GraphQL-related 
 * operations, but unions in this object are defined with Prisma fields
 * @returns partialInfo object with union fields added
 */
export const constructUnions = <
    GQLObject extends { [x: string]: any },
    PrismaObject extends { [x: string]: any }
>(partialInfo: { [x: string]: any }, gqlRelMap: GqlRelMap<GQLObject, PrismaObject>): { [x: string]: any } => {
    // Create result object
    let result: { [x: string]: any } = partialInfo;
    // Any value in the gqlRelMap which is an array is a union. 
    // All other values can be ignored.
    const unionFields: [string, GraphQLModelType[]][] = Object.entries(gqlRelMap).filter(([_, value]) => Array.isArray(value)) as any[];
    // For each union field
    for (const [key, value] of unionFields) {
        // For each GraphQL type in the union
        for (const type of value) {
            // If the type is in the partialInfo
            const isInPartialInfo = result[type] !== undefined;
            if (isInPartialInfo) {
                // Set the union field to the type
                result[key] = result[type];
                // Delete the type from the result
                delete result[type];
            }
        }
    }
    return result;
}