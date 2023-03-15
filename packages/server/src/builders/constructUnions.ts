import { GqlModelType } from "@shared/consts";
import { exists } from "@shared/utils";
import { GqlRelMap } from "../models/types";

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
    // Any value in the gqlRelMap which is an object is a union.
    // All other values can be ignored.
    const unionFields: [string, { [x: string]: GqlModelType }][] = Object.entries(gqlRelMap).filter(([, value]) => typeof value === 'object') as any[];
    // For each union field
    for (const [gqlField, unionData] of unionFields) {
        // For each entry in the union
        for (const [dbField, type] of Object.entries(unionData)) {
            console.log('checking union pair', gqlField, dbField, type, result[dbField])
            // If the current field is in the partial info, use it as the union data
            const isInPartialInfo = exists(result[dbField]);
            if (isInPartialInfo) {
                // Set the union field to the type
                result[gqlField] = { ...result[dbField], __typename: type };
            }
            // Delete the dbField from the result
            delete result[dbField];
        }
        // If no union data was found, set the union field to null
        if (!exists(result[gqlField])) {
            result[gqlField] = null;
        }
    }
    return result;
}